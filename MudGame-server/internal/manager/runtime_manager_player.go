package manager

import (
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"time"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

const (
	stateSettlementSeconds  = 10
	weaknessDurationSeconds = 300
	gatheringSeconds        = 30
	workingSeconds          = 60
)

func (r *RuntimeManager) GetAttributes(playerID string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	a := model.NormalizeAttributes(p.Attributes)
	eff := effectiveAttributesForPlayerNoLock(p)
	return map[string]interface{}{
		"level": a.Level, "realm": a.Realm,
		"experience": a.Experience, "potential": a.Potential,
		"currentHp": eff.HP, "maxHp": eff.MaxHP,
		"currentMp": eff.MP, "trainedMaxMp": eff.TrainedMaxMP, "maxMp": eff.MaxMP,
		"spirit": eff.Spirit, "maxSpirit": eff.MaxSpirit,
		"strength": eff.Strength, "constitution": eff.Constitution, "agility": eff.Agility,
		"perception": eff.Perception, "willpower": eff.Willpower, "charm": eff.Charm, "luck": eff.Luck,
		"attack": eff.Attack, "defense": eff.Defense,
		"hitRate": eff.HitRate, "dodge": eff.Dodge, "parry": eff.Parry, "critRate": eff.CritRate,
		"attackSpeed": eff.AttackSpeed, "initiative": eff.Initiative,
		"guardBreak": eff.GuardBreak, "damageReduction": eff.DamageReduction,
		"postStrength": a.Growth.PostStrength, "postConstitution": a.Growth.PostConstitution,
		"postAgility": a.Growth.PostAgility, "postPerception": a.Growth.PostPerception,
		"postWillpower":     a.Growth.PostWillpower,
		"totalPostStrength": eff.Strength - a.Base.Strength, "totalPostConstitution": eff.Constitution - a.Base.Constitution,
		"totalPostAgility": eff.Agility - a.Base.Agility, "totalPostPerception": eff.Perception - a.Base.Perception,
		"totalPostWillpower": eff.Willpower - a.Base.Willpower,
		"fame":               eff.Fame, "notoriety": eff.Notoriety, "authority": eff.Authority, "credit": eff.Credit,
		"base": a.Base, "growth": a.Growth, "resources": a.Resources,
		"derived": a.Derived, "social": a.Social, "resistances": a.Resistances,
		// 虚弱剩余秒数（0 = 无虚弱），前端据此展示虚弱状态
		"weakRemainingSeconds": p.WeakRemainingSeconds(time.Now().Unix()),
	}, nil
}

func (r *RuntimeManager) GetBackpack(playerID string) ([]map[string]interface{}, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, 0, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, 0, exception.ErrPlayerNotFound
	}
	items := make([]map[string]interface{}, 0, len(p.Backpack))
	for id, count := range p.Backpack {
		def, ok := fileItems[id]
		if !ok || count <= 0 {
			continue
		}
		items = append(items, map[string]interface{}{
			"id": def.ID, "uniqueId": def.ID, "name": def.Name, "description": def.Description,
			"rarity": def.Rarity, "value": def.Value, "stackable": def.Stackable,
			"unit": def.Unit, "count": count, "equipmentSlot": def.EquipmentSlot, "sellable": def.Sellable,
			"type": def.Type, "subtype": def.Subtype, "bind": def.Bind, "questItem": def.QuestItem,
			"usable": def.Usable, "effects": def.Effects, "source": def.Source,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i]["name"].(string) < items[j]["name"].(string) })
	return items, p.Money, nil
}

func (r *RuntimeManager) GetSkills(playerID string) ([]map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	out := make([]map[string]interface{}, 0, len(p.Skills))
	levelCap := model.SkillLevelCapForExperience(p.Attributes.Experience)
	for id, level := range p.Skills {
		def, ok := fileSkills[id]
		if !ok {
			continue
		}
		targetLevel := level + 1
		nextLevelPotential := model.SkillPotentialForTargetLevel(targetLevel)
		learningProgress := p.SkillLearningProgress[id]
		if learningProgress > nextLevelPotential {
			learningProgress = nextLevelPotential
		}
		out = append(out, map[string]interface{}{
			"id": id, "name": def.Name, "description": def.Description, "rarity": def.Rarity, "level": level,
			"type": def.Type, "category": def.Category, "school": def.School,
			"bonuses": def.Bonuses, "bonusText": skillBonusText(def.Bonuses, level), "requirements": def.Requirements,
			"levelCap": levelCap, "nextLevelPotential": nextLevelPotential, "learningProgress": learningProgress,
			"isLearning": p.State == "LEARNING" && p.LearningSkillID == id,
			"equipped":   def.Type == "ADVANCED" && skillPrerequisiteID(def) != "" && p.EquippedSkills[skillPrerequisiteID(def)] == id,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
	return out, nil
}

func (r *RuntimeManager) GetEquipment(playerID string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	out := map[string]interface{}{}
	for slot, itemID := range p.Equipment {
		if itemID == "" {
			continue
		}
		def, ok := fileItems[itemID]
		if !ok {
			continue
		}
		out[slot] = map[string]interface{}{"id": def.ID, "name": def.Name, "description": def.Description, "rarity": def.Rarity, "effects": def.Effects}
	}
	return out, nil
}

func (r *RuntimeManager) GetWorlds() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(fileWorldOrder))
	for _, id := range fileWorldOrder {
		w := fileWorlds[id]
		if !w.Visible {
			continue
		}
		out = append(out, map[string]interface{}{
			"id": w.ID, "name": w.Name, "description": w.Description,
			"order": w.Order, "category": w.Category,
		})
	}
	return out
}

func (r *RuntimeManager) EnterWorld(playerID, worldID string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	if _, inDungeon := r.dungeonMgr.PlayerInstanceIDNoLock(playerID); inDungeon {
		return nil, errors.New("in_dungeon_state")
	}
	if p.State != "IDLE" {
		if p.State == "DEAD" {
			return nil, errors.New("dead_state")
		}
		return nil, errors.New("busy_state")
	}
	w, ok := fileWorlds[worldID]
	if !ok {
		return nil, errors.New("world_not_found")
	}
	if p.RoomID == w.SpawnRoomID {
		roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
		return roomPayload, err
	}
	p.RoomID = w.SpawnRoomID
	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
	return roomPayload, err
}

func (r *RuntimeManager) Move(playerID, direction string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	staticRooms, err := r.loadRooms()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		if p.State == "DEAD" {
			return nil, errors.New("dead_state")
		}
		switch p.State {
		case "HEALING":
			return nil, errors.New("healing_state")
		case "MEDITATING":
			return nil, errors.New("meditating_state")
		case "MINING":
			return nil, errors.New("mining_state")
		case "LEARNING":
			return nil, errors.New("learning_state")
		case "COMBAT":
			return nil, errors.New("combat_state")
		default:
			return nil, errors.New("busy_state")
		}
	}
	room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
	if !ok {
		return nil, exception.ErrRoomNotFound
	}
	ex, ok := room.Exits[direction]
	if !ok {
		return nil, errors.New("no_exit_in_that_direction")
	}
	if err := r.checkExitRequirementNoLock(p, room, direction); err != nil {
		return nil, err
	}
	p.RoomID = ex.TargetRoomID
	r.applyEnterRoomDungeonObjectiveNoLock(p.RoomID)
	if r.dungeonMgr != nil {
		r.dungeonMgr.UpdateCheckpointNoLock(p.RoomID)
	}
	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
	return roomPayload, err
}

func (r *RuntimeManager) checkExitRequirementNoLock(p *model.Player, room model.Room, direction string) error {
	requirement, ok := room.ExitRequirements[direction]
	if !ok {
		return nil
	}
	blockedTemplates := map[string]bool{}
	for _, templateID := range requirement.DefeatNPCs {
		blockedTemplates[templateID] = true
	}
	for _, npcID := range room.NPCIDs {
		if dungeonNPC, exists := r.dungeonNPCByIDNoLock(npcID); exists && dungeonNPC.Alive && !dungeonNPC.Defeated && blockedTemplates[dungeonNPC.TemplateID] {
			if requirement.Message != "" {
				return errors.New(requirement.Message)
			}
			return errors.New(dungeonNPC.Name + "拦住了你。")
		}
		if staticNPC, exists := fileNpcs[npcID]; exists && blockedTemplates[npcID] {
			state, _ := r.staticNPCStateNoLock(npcID, room.ID)
			if state == nil || state.Alive {
				if requirement.Message != "" {
					return errors.New(requirement.Message)
				}
				return errors.New(staticNPC.Name + "拦住了你。")
			}
		}
	}
	if r.dungeonMgr != nil {
		if inst, exists := r.dungeonMgr.PlayerInstanceNoLock(p.ID); exists {
			for puzzleID, expectedState := range requirement.RequiredPuzzleStates {
				if inst.PuzzleStates[puzzleID] != expectedState {
					if requirement.Message != "" {
						return errors.New(requirement.Message)
					}
					return errors.New("眼前的机关尚未解开。")
				}
			}
		}
	}
	if requirement.RequiredItemID == "" {
		return nil
	}
	requiredAmount := requirement.RequiredAmount
	if requiredAmount <= 0 {
		requiredAmount = 1
	}
	if p.Backpack[requirement.RequiredItemID] < requiredAmount {
		if requirement.Message != "" {
			return errors.New(requirement.Message)
		}
		return errors.New("你还没有找到通行所需的物品。")
	}
	if requirement.ConsumeItem {
		p.Backpack[requirement.RequiredItemID] -= requiredAmount
		if p.Backpack[requirement.RequiredItemID] <= 0 {
			delete(p.Backpack, requirement.RequiredItemID)
		}
	}
	return nil
}

func (r *RuntimeManager) GetShop(playerID, npcID string) ([]map[string]interface{}, string, string, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, "", "", "", err
	}
	staticRooms, err := r.loadRooms()
	if err != nil {
		return nil, "", "", "", err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, "", "", "", exception.ErrPlayerNotFound
	}
	room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
	if !ok {
		return nil, "", "", "", exception.ErrRoomNotFound
	}
	npcAllowed := false
	for _, id := range room.NPCIDs {
		if id == npcID {
			npcAllowed = true
			break
		}
	}
	if !npcAllowed {
		return nil, "", "", "", errors.New("shop_npc_not_in_room")
	}
	npcName := ""
	npcTitle := ""
	shopInventory := map[string]int{}
	if dnpc, _, ok := r.dungeonNPCInRoomNoLock(room.ID, npcID); ok {
		if !dnpc.IsShop {
			return nil, "", "", "", errors.New("npc_not_shop")
		}
		npcName = dnpc.Name
		npcTitle = dnpc.Title
		for itemID, limit := range dnpc.ShopInventory {
			shopInventory[itemID] = limit
		}
	} else {
		npc, ok := fileNpcs[npcID]
		if !ok || !npc.IsShop {
			return nil, "", "", "", errors.New("npc_not_shop")
		}
		state, ok := r.staticNPCStateNoLock(npcID, room.ID)
		if !ok || state == nil {
			return nil, "", "", "", errors.New("npc_not_shop")
		}
		npcName = state.Name
		npcTitle = state.Title
		for itemID, limit := range state.ShopInventory {
			shopInventory[itemID] = limit
		}
	}
	out := make([]map[string]interface{}, 0, len(shopInventory))
	for itemID, limit := range shopInventory {
		def, ok := fileItems[itemID]
		if !ok {
			continue
		}
		entry := map[string]interface{}{
			"id":          def.ID,
			"name":        def.Name,
			"description": def.Description,
			"type":        def.Type,
			"subtype":     def.Subtype,
			"rarity":      def.Rarity,
			"value":       def.Value,
			"stackable":   def.Stackable,
			"unit":        def.Unit,
			"bind":        def.Bind,
			"questItem":   def.QuestItem,
			"usable":      def.Usable,
			"effects":     def.Effects,
			"source":      def.Source,
		}
		if limit != -1 {
			entry["limit"] = limit
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["name"].(string) < out[j]["name"].(string) })
	return out, npcID, npcName, npcTitle, nil
}

func (r *RuntimeManager) Buy(playerID, npcID, itemID string, amount int) (int, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if amount <= 0 {
		amount = 1
	}
	players, err := r.loadPlayers()
	if err != nil {
		return 0, 0, err
	}
	staticRooms, err := r.loadRooms()
	if err != nil {
		return 0, 0, err
	}
	p, ok := players[playerID]
	if !ok {
		return 0, 0, exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return 0, 0, errors.New("busy")
	}
	room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
	if !ok {
		return 0, 0, exception.ErrRoomNotFound
	}
	npcInRoom := false
	for _, id := range room.NPCIDs {
		if id == npcID {
			npcInRoom = true
			break
		}
	}
	if !npcInRoom {
		return 0, 0, errors.New("shop_npc_not_in_room")
	}
	shopInventory := map[string]int{}
	dungeonShop := false
	if dnpc, _, ok := r.dungeonNPCInRoomNoLock(room.ID, npcID); ok {
		if !dnpc.IsShop {
			return 0, 0, errors.New("shop_not_found")
		}
		dungeonShop = true
		shopInventory = dnpc.ShopInventory
	} else {
		npc, ok := fileNpcs[npcID]
		if !ok || !npc.IsShop {
			return 0, 0, errors.New("shop_not_found")
		}
		// 静态商店使用运行时隔离库存，扣减不得污染全局 NPC 模板
		state, ok := r.staticNPCStateNoLock(npcID, room.ID)
		if !ok || state == nil {
			return 0, 0, errors.New("shop_not_found")
		}
		shopInventory = state.ShopInventory
	}
	limit, ok := shopInventory[itemID]
	if !ok {
		return 0, 0, errors.New("sold_out")
	}
	if limit != -1 && limit <= 0 {
		return 0, 0, errors.New("sold_out")
	}
	if limit != -1 && amount > limit {
		amount = limit
	}
	def, ok := fileItems[itemID]
	if !ok {
		return 0, 0, errors.New("item_not_found")
	}
	total := def.Value * amount
	if p.Money < total {
		return 0, 0, errors.New("money_not_enough")
	}
	p.Money -= total
	p.Backpack[itemID] += amount
	if limit != -1 {
		shopInventory[itemID] -= amount
		if dungeonShop {
			if dnpc, _, ok := r.dungeonNPCInRoomNoLock(room.ID, npcID); ok {
				dnpc.ShopInventory[itemID] = shopInventory[itemID]
			}
		}
	}
	if err := r.savePlayers(players); err != nil {
		return 0, 0, err
	}
	return amount, p.Money, nil
}

func (r *RuntimeManager) DiscardItem(playerID, itemID string, amount int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if amount <= 0 {
		amount = 1
	}
	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok {
		return exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return errors.New("busy")
	}
	have := p.Backpack[itemID]
	if have <= 0 {
		return errors.New("item_not_found")
	}
	if amount > have {
		amount = have
	}
	p.Backpack[itemID] -= amount
	if p.Backpack[itemID] <= 0 {
		delete(p.Backpack, itemID)
	}
	return r.savePlayers(players)
}

func (r *RuntimeManager) SellItem(playerID, itemID string, amount int) (int, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if amount <= 0 {
		amount = 1
	}
	players, err := r.loadPlayers()
	if err != nil {
		return 0, 0, err
	}
	staticRooms, err := r.loadRooms()
	if err != nil {
		return 0, 0, err
	}
	p, ok := players[playerID]
	if !ok {
		return 0, 0, exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return 0, 0, errors.New("busy")
	}
	room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
	if !ok {
		return 0, 0, exception.ErrRoomNotFound
	}
	if len(room.NPCIDs) == 0 {
		return 0, 0, errors.New("no_npc_buyer")
	}
	have := p.Backpack[itemID]
	if have <= 0 {
		return 0, 0, errors.New("item_not_found")
	}
	if amount > have {
		amount = have
	}
	def, ok := fileItems[itemID]
	if !ok {
		return 0, 0, errors.New("item_not_found")
	}
	if !def.Sellable {
		return 0, 0, errors.New("item_not_sellable")
	}
	p.Backpack[itemID] -= amount
	if p.Backpack[itemID] <= 0 {
		delete(p.Backpack, itemID)
	}
	gain := (def.Value / 2) * amount
	p.Money += gain
	if err := r.savePlayers(players); err != nil {
		return 0, 0, err
	}
	return gain, p.Money, nil
}

func (r *RuntimeManager) EquipItem(playerID, itemID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok {
		return exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return errors.New("busy")
	}
	if itemID == "" {
		return errors.New("missing_item_id")
	}
	if p.Backpack[itemID] <= 0 {
		return errors.New("item_not_in_inventory")
	}
	def, ok := fileItems[itemID]
	if !ok || def.EquipmentSlot == "" {
		return errors.New("not_equippable")
	}
	if p.Equipment == nil {
		p.Equipment = map[string]string{}
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	slot := def.EquipmentSlot
	if old := p.Equipment[slot]; old != "" {
		p.Backpack[old] += 1
	}
	p.Backpack[itemID] -= 1
	if p.Backpack[itemID] <= 0 {
		delete(p.Backpack, itemID)
	}
	p.Equipment[slot] = itemID
	return r.savePlayers(players)
}

func (r *RuntimeManager) UnequipItem(playerID, slot string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok {
		return exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return errors.New("busy")
	}
	if slot == "" {
		return errors.New("missing_slot")
	}
	if p.Equipment == nil {
		p.Equipment = map[string]string{}
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	itemID := p.Equipment[slot]
	if itemID == "" {
		return errors.New("slot_empty")
	}
	p.Backpack[itemID] += 1
	delete(p.Equipment, slot)
	return r.savePlayers(players)
}

func (r *RuntimeManager) LearnSkill(playerID, npcID, skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	staticRooms, err := r.loadRooms()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok {
		return exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return errors.New("busy_state")
	}
	if npcID == "" || skillID == "" {
		return errors.New("missing_params")
	}
	if p.Attributes.Potential <= 0 {
		return errors.New("no_potential")
	}
	room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
	if !ok {
		return exception.ErrRoomNotFound
	}
	npcInRoom := false
	for _, id := range room.NPCIDs {
		if id == npcID {
			npcInRoom = true
			break
		}
	}
	if !npcInRoom {
		return errors.New("npc_not_found")
	}
	teacherSkillLevel := 0
	if dnpc, _, ok := r.dungeonNPCInRoomNoLock(room.ID, npcID); ok {
		var taught bool
		teacherSkillLevel, taught = dnpc.Skills[skillID]
		if !taught || teacherSkillLevel <= 0 {
			return errors.New("skill_not_taught")
		}
	} else {
		npc, ok := fileNpcs[npcID]
		if !ok {
			return errors.New("npc_not_found")
		}
		var taught bool
		teacherSkillLevel, taught = npc.Skills[skillID]
		if !taught || teacherSkillLevel <= 0 {
			return errors.New("skill_not_taught")
		}
	}
	skillDef, ok := fileSkills[skillID]
	if !ok {
		return errors.New("skill_not_taught")
	}
	// 进阶武学（隐秘传承）必须先通过残页参悟入门，老师只负责传授深造
	if skillDef.Type == "ADVANCED" && p.Skills[skillID] < 1 {
		return errors.New("你尚未参悟" + skillDef.Name + "的传承残页，无从学起。")
	}
	// 秘籍学习的前置门槛：对应基本武功必须达到指定等级（见 docs/design/martial_arts_curriculum.md 段位模型）
	if reqSkill := skillPrerequisiteID(skillDef); reqSkill != "" {
		if reqLevel := skillPrerequisiteLevel(skillDef); p.Skills[reqSkill] < reqLevel {
			if base, ok := fileSkills[reqSkill]; ok {
				return errors.New("你的" + base.Name + "尚未练到" + strconv.Itoa(reqLevel) + "级，无从修习" + skillDef.Name + "。")
			}
			return errors.New("skill_prerequisite_not_met")
		}
	}
	if p.Skills[skillID] >= teacherSkillLevel {
		return errors.New("你的" + skillDef.Name + "已经不输你师傅了")
	}
	if p.Skills[skillID] >= model.SkillLevelCapForExperience(p.Attributes.Experience) {
		return errors.New("skill_level_capped")
	}
	if p.Skills == nil {
		p.Skills = map[string]int{}
	}
	if p.SkillLearningProgress == nil {
		p.SkillLearningProgress = map[string]int{}
	}
	// 新学技能立即注册 0 级条目，学习期间面板可见（首次 10 秒结算后升 1 级）
	if _, exists := p.Skills[skillID]; !exists {
		p.Skills[skillID] = 0
	}
	p.State = "LEARNING"
	p.StateTickCountdown = stateSettlementSeconds
	p.LearningSkillID = skillID
	p.LearningTeacherID = npcID
	p.LearningTeacherLevel = teacherSkillLevel
	return r.savePlayers(players)
}

// EquipSkill 将已参悟的秘籍装备到其前置基本武功槽位：同槽择一，仅装备中的秘籍参与属性加成
// （装备规则见 docs/design/formulas.md 3.3 与 docs/design/martial_arts_curriculum.md 段位模型）。
func (r *RuntimeManager) EquipSkill(playerID, skillID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	def, ok := fileSkills[skillID]
	if !ok {
		return "", errors.New("skill_not_found")
	}
	if def.Type != "ADVANCED" {
		return "", errors.New(def.Name + "是基本武功，无需装备。")
	}
	if p.Skills[skillID] < 1 {
		return "", errors.New("你尚未学会" + def.Name + "，无从装备。")
	}
	slot := skillPrerequisiteID(def)
	if slot == "" {
		return "", errors.New(def.Name + "没有对应的基本武功槽位。")
	}
	base := fileSkills[slot]
	if p.EquippedSkills == nil {
		p.EquippedSkills = map[string]string{}
	}
	p.EquippedSkills[slot] = skillID
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你将" + def.Name + "融汇于" + base.Name + "之中。", nil
}

func (r *RuntimeManager) DoRoomAction(playerID, action string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	staticRooms, err := r.loadRooms()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	if p.State == "DEAD" {
		return nil, errors.New("dead_state")
	}
	if p.State != "IDLE" {
		return nil, errors.New("busy_state")
	}
	room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
	if !ok {
		return nil, exception.ErrRoomNotFound
	}
	allowed := false
	for _, a := range room.Actions {
		if a == action {
			allowed = true
			break
		}
	}
	if !allowed && r.dungeonMgr != nil {
		if inst, exists := r.dungeonMgr.PlayerInstanceNoLock(playerID); exists && inst.PendingChoices[action] {
			allowed = true
		}
	}
	if !allowed {
		return nil, errors.New("action_not_available")
	}

	result := map[string]interface{}{"action": action}
	switch action {
	case "gather":
		// 采集/帮工是世界的专门资源点：进入忙碌状态，倒计时结束由状态 tick 一次结算收益
		p.State = "GATHERING"
		p.StateTickCountdown = gatheringSeconds
		result["message"] = "你开始在附近采集可用之物……"
	case "work":
		p.State = "WORKING"
		p.StateTickCountdown = workingSeconds
		result["message"] = "你接了份零散差事，埋头干了起来……"
	case "listen_rumor":
		result["message"] = roomRumor(room.ID)
	case "search":
		message, err := r.searchRoomNoLock(p, room)
		if err != nil {
			return nil, err
		}
		result["message"] = message
	default:
		puzzleResult, handled := r.executePuzzleActionNoLock(p, room, action)
		if !handled {
			return nil, errors.New("action_not_supported")
		}
		for key, value := range puzzleResult {
			result[key] = value
		}
	}
	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RuntimeManager) executePuzzleActionNoLock(p *model.Player, room model.Room, actionID string) (map[string]interface{}, bool) {
	if r.dungeonMgr == nil || p == nil {
		return nil, false
	}
	inst, action, ok := r.dungeonMgr.PuzzleActionNoLock(p.ID, room.ID, actionID)
	if !ok {
		return nil, false
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	if p.Flags == nil {
		p.Flags = map[string]bool{}
	}
	currentState := inst.PuzzleStates[action.PuzzleID]
	stateStatus := r.dungeonMgr.PuzzleActionStateStatusNoLock(inst, action)
	if stateStatus == engruntime.PuzzleActionRepeat {
		message := action.RepeatMessage
		if message == "" {
			message = "你已经完成“" + action.Label + "”，无需重复。"
		}
		result := map[string]interface{}{
			"message":      message,
			"puzzleId":     action.PuzzleID,
			"puzzleState":  currentState,
			"actionStatus": "repeat",
		}
		if len(inst.PendingChoices) > 0 {
			if prompt := r.dungeonMgr.PendingChoicePromptNoLock(inst); prompt != "" {
				result["message"] = prompt
			}
			result["choices"] = dungeonChoicePayload(r.dungeonMgr.PendingActionChoicesNoLock(inst))
		}
		if currentState == action.NextState && len(action.Choices) > 0 && len(inst.PendingChoices) == 0 {
			result["message"] = action.Message
			result["choices"] = activateDungeonChoices(inst, action.Choices, action.Message)
		}
		return result, true
	}
	allowedState := stateStatus == engruntime.PuzzleActionReady
	itemsReady := true
	for itemID, amount := range action.RequiredItems {
		if amount > 0 && p.Backpack[itemID] < amount {
			itemsReady = false
			break
		}
	}
	puzzlesReady := true
	for puzzleID, requiredState := range action.RequiredPuzzleStates {
		if inst.PuzzleStates[puzzleID] != requiredState {
			puzzlesReady = false
			break
		}
	}
	npcsDefeated := true
	for _, templateID := range action.RequiredDefeatNPCs {
		found := false
		for _, npc := range inst.NPCs {
			if npc != nil && npc.TemplateID == templateID && npc.RoomID == room.ID {
				found = true
				if npc.Alive && !npc.Defeated {
					npcsDefeated = false
				}
			}
		}
		if !found {
			npcsDefeated = false
		}
	}
	if !allowedState || !itemsReady || !puzzlesReady || !npcsDefeated {
		inst.PuzzleAttempts[action.PuzzleID]++
		if len(action.FailureSpawnNPCs) > 0 || action.FailureResetAfter > 0 {
			r.dungeonMgr.ApplyActionNPCEffectsNoLock(inst, room.ID, nil, nil, action.FailureSpawnNPCs)
			if action.FailureResetAfter > 0 && inst.PuzzleAttempts[action.PuzzleID] >= action.FailureResetAfter {
				inst.PuzzleStates[action.PuzzleID] = action.FailureResetState
				inst.PuzzleAttempts[action.PuzzleID] = 0
			}
		}
		message := action.FailureMessage
		if message == "" {
			message = "“" + action.Label + "”所需条件尚未满足。"
		}
		result := map[string]interface{}{
			"message":        message,
			"puzzleId":       action.PuzzleID,
			"puzzleState":    currentState,
			"puzzleAttempts": inst.PuzzleAttempts[action.PuzzleID],
			"actionStatus":   "blocked",
		}
		if inst.PendingChoices[actionID] {
			result["choices"] = dungeonChoicePayload(r.dungeonMgr.PendingActionChoicesNoLock(inst))
		}
		return result, true
	}
	if inst.PendingChoices[actionID] {
		inst.PendingChoices = map[string]bool{}
		inst.PendingChoicePrompt = ""
		inst.PendingChoiceOptions = nil
	}
	for itemID, amount := range action.TakeItems {
		if amount <= 0 {
			continue
		}
		p.Backpack[itemID] -= amount
		if p.Backpack[itemID] <= 0 {
			delete(p.Backpack, itemID)
		}
	}
	for itemID, amount := range action.GiveItems {
		if amount <= 0 {
			continue
		}
		p.Backpack[itemID] += amount
		r.applyCollectDungeonObjectiveNoLock(p.ID, itemID)
	}
	for flag, value := range action.SetFlags {
		p.Flags[flag] = value
	}
	if action.NextState != "" {
		inst.PuzzleStates[action.PuzzleID] = action.NextState
		currentState = action.NextState
		r.dungeonMgr.ApplyPuzzleObjectiveNoLock(inst, action.PuzzleID, currentState)
	}
	r.dungeonMgr.CompleteObjectivesNoLock(inst, action.CompleteObjectives)
	r.dungeonMgr.ApplyActionNPCEffectsNoLock(inst, room.ID, action.RemoveNPCs, action.KillNPCs, action.SpawnNPCs)
	moved := false
	if action.MoveToRoomID != "" {
		if targetRoomID := r.dungeonMgr.InstanceRoomIDNoLock(inst, action.MoveToRoomID); targetRoomID != "" {
			p.RoomID = targetRoomID
			moved = true
			r.applyEnterRoomDungeonObjectiveNoLock(targetRoomID)
			r.dungeonMgr.UpdateCheckpointNoLock(targetRoomID)
		}
	}
	message := action.Message
	if message == "" {
		message = "你完成了眼前的操作。"
	}
	result := map[string]interface{}{
		"message":      message,
		"puzzleId":     action.PuzzleID,
		"puzzleState":  currentState,
		"actionStatus": "success",
		"moved":        moved,
	}
	if len(action.Choices) > 0 {
		result["choices"] = activateDungeonChoices(inst, action.Choices, action.Message)
	}
	return result, true
}

func activateDungeonChoices(inst *engruntime.Instance, choices []engruntime.ActionChoice, prompt string) []map[string]string {
	inst.PendingChoices = map[string]bool{}
	inst.PendingChoicePrompt = prompt
	inst.PendingChoiceOptions = append([]engruntime.ActionChoice{}, choices...)
	for _, choice := range choices {
		inst.PendingChoices[choice.ID] = true
	}
	return dungeonChoicePayload(choices)
}

func dungeonChoicePayload(choices []engruntime.ActionChoice) []map[string]string {
	out := make([]map[string]string, 0, len(choices))
	for _, choice := range choices {
		out = append(out, map[string]string{"id": choice.ID, "label": choice.Label})
	}
	return out
}

func (r *RuntimeManager) searchRoomNoLock(p *model.Player, room model.Room) (string, error) {
	if p == nil {
		return "", exception.ErrPlayerNotFound
	}
	if p.Flags == nil {
		p.Flags = map[string]bool{}
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	// 剧情调查点：按房间脚本化（一次性旗标，供任务目标使用）
	if room.ID == "yz_reed_marsh" && !p.Flags["found_caoyin_clue"] {
		p.Flags["found_caoyin_clue"] = true
		return "你在苇荡深处的泥滩上刨出几只空银箱——箱底的官印拓痕清清楚楚，是盐运司的封条。水路通向运河下游，箱子是从船上搬下来的。", nil
	}
	if room.ID == "ly_salt_wharf" && !p.Flags["found_smuggling_evidence"] {
		p.Flags["found_smuggling_evidence"] = true
		return "你伏在盐垛后看了半晌：几条黑影正把官盐搬上无籍小船，船帮上刻着一只歪歪扭扭的鹞子——是水匪的记号。你默默记下了船号。", nil
	}
	if room.ID == "tg_jingou_beacon" && !p.Flags["found_beacon_signal"] {
		p.Flags["found_beacon_signal"] = true
		return "你攀上残破的烽燧：燧台积灰下压着半燃的火薪，柴堆摆得整整齐齐却始终没有点起——不是断讯，是有人不许点。你在灰里捡到一枚扯掉的号牌。", nil
	}
	return "你仔细搜索四周，没有发现值得带走的东西。", nil
}

func roomRumor(roomID string) string {
	switch roomID {
	case "ly_00":
		return "你听见桥边有人低声谈起扬州来的两个少年，说他们身上似乎牵着一场大风波。"
	case "ly_west_market":
		return "西市掮客神秘兮兮地说，惊雁宫重见天日的消息已经传开，有人正重金打探《战神图录》的下落。"
	case "ly_inn":
		return "说书人醒木一拍，正讲到一只来历不明的战国铜匣，满堂酒客都屏住了呼吸。"
	case "ly_nan_market":
		return "南市布庄的伙计压低嗓子：前几日有位白衣女子来买素帕，付账的银子边缘竟微微发乌——账房先生收了银子，转头就点了三炷香。"
	case "ly_dock":
		return "码头脚夫们都在议论江淮战事——听说竟陵城头一架大弩，一箭钉穿了敌将的将旗。"
	case "tg_taolin_station":
		return "驿丞给往来的客商斟茶：长安来的邸报说，唐国公府最近车马进出得勤——改元也好、变天也罢，咱们小驿站只管把火把备足。"
	case "ca2_west_market":
		return "胡商们压低嗓门谈和氏璧——有人说慈航静斋已经带着它出了城，有人说它还在别院的青灯之下。"
	case "yz_guanjie":
		return "茶楼里的闲人压着嗓子：三万两漕银没了影，盐运司的书办却半点不急——有人赌定，这案子查不到他头上。"
	case "ca2_beitian_fang":
		return "施粥棚前的老流民喃喃自语：北边来的那位白衣娘娘，在长安也有人烧香了。"
	case "ly_north_market":
		return "估衣摊的贩子跟人咬耳朵：洛水上下最近不太平，好好的漕船说没就没——南边码头的把头们都愁白了头。"
	case "ly_refugee_camp":
		return "营里传来孩子的童谣：白衣娘娘摸一摸，夜里梦见亲娘哭。大人们听着都把火往旺里拨——没人肯接这个话头。"
	default:
		return "你驻足听了一阵，只听见市井喧嚣里夹着几句真假难辨的江湖传闻。"
	}
}

func (r *RuntimeManager) Heal(playerID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	if p.State == "COMBAT" {
		return "", errors.New("combat_state")
	}
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}
	if p.State == "HEALING" {
		return "", errors.New("healing_state")
	}
	if p.State == "MEDITATING" {
		return "", errors.New("meditating_state")
	}
	if p.State == "LEARNING" {
		return "", errors.New("learning_state")
	}
	effectiveMaxHP := effectiveAttributesForPlayerNoLock(p).MaxHP
	if p.Attributes.Resources.HP >= effectiveMaxHP {
		return "", errors.New("hp_full")
	}
	if p.Attributes.Resources.MP < 10 {
		return "", errors.New("mp_insufficient")
	}
	p.Attributes.Resources.MP -= 10
	p.Attributes.Resources.HP += 20
	if p.Attributes.Resources.HP > effectiveMaxHP {
		p.Attributes.Resources.HP = effectiveMaxHP
	}
	p.State = "HEALING"
	p.StateTickCountdown = stateSettlementSeconds
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你开始运功疗伤。", nil
}

// insightSkillPageNoLock 参悟传承残页：消耗残页，对应进阶武学升至 1 级（入门）
func (r *RuntimeManager) insightSkillPageNoLock(players map[string]*model.Player, p *model.Player, itemID, skillID string) (string, error) {
	skillDef, ok := fileSkills[skillID]
	if !ok {
		return "", errors.New("skill_not_found")
	}
	if p.Skills[skillID] > 0 {
		return "", errors.New("你已参悟过" + skillDef.Name + "，无需重复参悟。")
	}
	p.Backpack[itemID] -= 1
	if p.Backpack[itemID] <= 0 {
		delete(p.Backpack, itemID)
	}
	if p.Skills == nil {
		p.Skills = map[string]int{}
	}
	p.Skills[skillID] = 1
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你屏息凝神，逐字参悟残页上的传承，初步领悟了" + skillDef.Name + "。", nil
}

func (r *RuntimeManager) UseItem(playerID, itemID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if itemID == "" {
		return "", errors.New("missing_item_id")
	}
	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	// 战斗中允许使用药品：金创药等是副本战斗的核心消耗品（见收益与经济设计文档）
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}
	if p.Backpack[itemID] <= 0 {
		return "", errors.New("item_not_in_inventory")
	}
	def, ok := fileItems[itemID]
	if !ok || !def.Usable {
		return "", errors.New("item_not_usable")
	}
	// 传承残页：参悟进阶武学入门（见收益与经济设计文档 1.3.1）
	if learnSkillID, ok := def.Effects["learnSkill"].(string); ok && learnSkillID != "" {
		return r.insightSkillPageNoLock(players, p, itemID, learnSkillID)
	}
	recoverHp := 0
	if value, ok := def.Effects["recoverHp"].(float64); ok && value > 0 {
		recoverHp = int(value)
	}
	if recoverHp <= 0 {
		return "", errors.New("item_not_usable_now")
	}
	effectiveMaxHP := effectiveAttributesForPlayerNoLock(p).MaxHP
	if p.Attributes.Resources.HP >= effectiveMaxHP {
		return "", errors.New("hp_full")
	}
	p.Backpack[itemID] -= 1
	if p.Backpack[itemID] <= 0 {
		delete(p.Backpack, itemID)
	}
	beforeHP := p.Attributes.Resources.HP
	p.Attributes.Resources.HP += recoverHp
	if p.Attributes.Resources.HP > effectiveMaxHP {
		p.Attributes.Resources.HP = effectiveMaxHP
	}
	recovered := p.Attributes.Resources.HP - beforeHP
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你服下" + def.Name + "，恢复了" + strconv.Itoa(recovered) + "点气血。", nil
}

func (r *RuntimeManager) Meditate(playerID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	if p.State == "COMBAT" {
		return "", errors.New("combat_state")
	}
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}
	if p.State == "HEALING" {
		return "", errors.New("healing_state")
	}
	if p.State == "MEDITATING" {
		return "", errors.New("meditating_state")
	}
	if p.State == "LEARNING" {
		return "", errors.New("learning_state")
	}
	eff := effectiveAttributesForPlayerNoLock(p)
	if p.Attributes.Resources.TrainedMaxMP >= eff.MaxMP && p.Attributes.Resources.MP >= p.Attributes.Resources.TrainedMaxMP {
		return "", errors.New("full_state")
	}
	p.Attributes.Resources.MP += 15
	if p.Attributes.Resources.MP > p.Attributes.Resources.TrainedMaxMP {
		p.Attributes.Resources.MP = p.Attributes.Resources.TrainedMaxMP
	}
	p.State = "MEDITATING"
	p.StateTickCountdown = stateSettlementSeconds
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你开始打坐调息。", nil
}

func (r *RuntimeManager) Stop(playerID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err == nil {
		if p, ok := players[playerID]; ok {
			if p.State == "COMBAT" {
				return "你正在战斗，待会再说"
			}
			oldState := p.State
			p.State = "IDLE"
			p.StateTickCountdown = 0
			p.LearningSkillID = ""
			p.LearningTeacherID = ""
			p.LearningTeacherLevel = 0
			p.TargetID = ""
			p.CombatType = ""
			_ = r.savePlayers(players)
			switch oldState {
			case "HEALING":
				return "你停止了运功疗伤。"
			case "MEDITATING":
				return "你停止了打坐。"
			case "LEARNING":
				return "你停止了学习。"
			case "GATHERING":
				return "你放弃了采集，拍拍身上的尘土。"
			case "WORKING":
				return "你放下了手头的活计。"
			}
		}
	}
	return "你停止了动作。"
}

func (r *RuntimeManager) Spar(playerID, targetID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok1 := players[playerID]
	if !ok1 {
		return "", errors.New("player_not_found")
	}
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}
	if p.State != "IDLE" {
		return "", errors.New("busy")
	}
	if p.Attributes.Resources.HP < p.Attributes.Resources.MaxHP {
		return "", errors.New("self_not_full")
	}
	if targetID == "" {
		return "", errors.New("target_missing")
	}
	if p.ID == targetID {
		return "", errors.New("self_target")
	}
	if dnpc, ok := r.dungeonNPCByIDNoLock(targetID); ok {
		if dnpc.CombatMode == "protected" {
			return "", errors.New("target_is_protected")
		}
		if dnpc.Defeated {
			return "", errors.New("target_is_defeated")
		}
		if !dnpc.Alive {
			return "", errors.New("target_is_dead")
		}
		if dnpc.RoomID != p.RoomID {
			return "", errors.New("target_not_in_room")
		}
		return "", errors.New("npc_combat_unsupported")
	}
	if _, isNPC := fileNpcs[targetID]; isNPC {
		room, ok := r.roomByIDNoLock(p.RoomID, fileRooms)
		if !ok {
			return "", errors.New("room_not_found")
		}
		npc, ok := r.staticNPCInRoomNoLock(room, targetID)
		if !ok {
			return "", errors.New("target_not_in_room")
		}
		if npc.Attributes.Resources.HP < npc.Attributes.Resources.MaxHP {
			return "", errors.New("target_not_full")
		}
		p.State = "COMBAT"
		p.TargetID = npc.ID
		p.CombatType = "SPAR"
		if err := r.savePlayers(players); err != nil {
			return "", err
		}
		return p.CharacterName + "对" + npc.Name + "抱拳道：请赐教！", nil
	}
	t, ok2 := players[targetID]
	if !ok2 {
		return "", errors.New("target_not_found")
	}
	if p.RoomID != t.RoomID {
		return "", errors.New("target_not_in_room")
	}
	if t.State == "DEAD" {
		return "", errors.New("target_is_dead")
	}
	if t.State != "IDLE" {
		return "", errors.New("target_busy")
	}
	if t.Attributes.Resources.HP < t.Attributes.Resources.MaxHP {
		return "", errors.New("target_not_full")
	}
	p.State = "COMBAT"
	p.TargetID = t.ID
	p.CombatType = "SPAR"
	t.State = "COMBAT"
	t.TargetID = p.ID
	t.CombatType = "SPAR"
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return p.CharacterName + "对" + t.CharacterName + "抱拳道：请赐教！", nil
}

func (r *RuntimeManager) Kill(playerID, targetID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok1 := players[playerID]
	if !ok1 {
		return "", errors.New("player_not_found")
	}
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}
	if p.State != "IDLE" {
		return "", errors.New("busy_state")
	}
	if targetID == "" {
		return "", errors.New("target_missing")
	}
	if p.ID == targetID {
		return "", errors.New("self_target")
	}
	if dnpc, ok := r.dungeonNPCByIDNoLock(targetID); ok {
		if !dnpc.Alive {
			return "", errors.New("target_is_dead")
		}
		if dnpc.NonCombat {
			return "", errors.New(dnpc.Name + "并无战意，你无从下手。")
		}
		if dnpc.RoomID != p.RoomID {
			return "", errors.New("target_not_in_room")
		}
		counterMessage := ""
		if !dnpc.WeaknessApplied && dnpc.WeaknessFlag != "" && p.Flags[dnpc.WeaknessFlag] {
			applyNPCWeaknessNoLock(dnpc)
			dnpc.WeaknessApplied = true
			counterMessage = "你已看破对方招路，抢先封住了他的杀招。"
		}
		p.State = "COMBAT"
		p.TargetID = dnpc.ID
		p.CombatType = "KILL"
		if err := r.savePlayers(players); err != nil {
			return "", err
		}
		message := p.CharacterName + "对" + dnpc.Name + "喝道：纳命来！"
		if counterMessage != "" {
			message += counterMessage
		}
		return message, nil
	}
	if _, isNPC := fileNpcs[targetID]; isNPC {
		room, ok := r.roomByIDNoLock(p.RoomID, fileRooms)
		if !ok {
			return "", errors.New("room_not_found")
		}
		npc, ok := r.staticNPCInRoomNoLock(room, targetID)
		if !ok {
			return "", errors.New("target_not_in_room")
		}
		if npc.NonCombat {
			return "", errors.New(npc.Name + "并无战意，你无从下手。")
		}
		p.State = "COMBAT"
		p.TargetID = npc.ID
		p.CombatType = "KILL"
		if err := r.savePlayers(players); err != nil {
			return "", err
		}
		return p.CharacterName + "对" + npc.Name + "喝道：纳命来！", nil
	}
	t, ok2 := players[targetID]
	if !ok2 {
		return "", errors.New("target_not_found")
	}
	if p.RoomID != t.RoomID {
		return "", errors.New("target_not_in_room")
	}
	if t.State == "DEAD" {
		return "", errors.New("target_is_dead")
	}
	if t.State == "COMBAT" {
		return "", errors.New("target_in_combat")
	}
	p.State = "COMBAT"
	p.TargetID = t.ID
	p.CombatType = "KILL"
	t.State = "COMBAT"
	t.TargetID = p.ID
	t.CombatType = "KILL"
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return p.CharacterName + "对" + t.CharacterName + "喝道：纳命来！", nil
}

func applyNPCWeaknessNoLock(npc *engruntime.NPCState) {
	if npc == nil {
		return
	}
	for attribute, amount := range npc.WeaknessModifiers {
		switch attribute {
		case "attack":
			npc.Attributes.Derived.Attack = max(0, npc.Attributes.Derived.Attack+amount)
		case "defense":
			npc.Attributes.Derived.Defense = max(0, npc.Attributes.Derived.Defense+amount)
		case "hitRate":
			npc.Attributes.Derived.HitRate = max(0, npc.Attributes.Derived.HitRate+amount)
		case "dodge":
			npc.Attributes.Derived.Dodge = max(0, npc.Attributes.Derived.Dodge+amount)
		case "parry":
			npc.Attributes.Derived.Parry = max(0, npc.Attributes.Derived.Parry+amount)
		case "initiative":
			npc.Attributes.Derived.Initiative = max(0, npc.Attributes.Derived.Initiative+amount)
		}
	}
}

func (r *RuntimeManager) Relive(playerID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	if p.State != "DEAD" {
		return "", errors.New("not_dead")
	}
	p.State = "IDLE"
	p.TargetID = ""
	p.CombatType = ""
	eff := effectiveAttributesForPlayerNoLock(p)
	// 复活仅恢复半血半内力，并附带 5 分钟虚弱（战斗属性减半，见收益与经济设计文档）
	p.Attributes.Resources.HP = eff.MaxHP / 2
	p.Attributes.Resources.MP = p.Attributes.Resources.TrainedMaxMP / 2
	p.WeakUntil = time.Now().Unix() + weaknessDurationSeconds
	if r.dungeonMgr != nil {
		if inst, exists := r.dungeonMgr.PlayerInstanceNoLock(playerID); exists && inst.CheckpointRoomID != "" {
			p.RoomID = inst.CheckpointRoomID
		} else {
			p.RoomID = "ly_00"
		}
	} else {
		p.RoomID = "ly_00"
	}
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你苏醒了过来，但一阵虚弱感涌上全身（战斗属性减半，持续5分钟）。", nil
}

// settleWorldLaborNoLock 结算采集/帮工的一次性收益（世界专门资源点，数值见收益与经济设计文档）
func (r *RuntimeManager) settleWorldLaborNoLock(p *model.Player, result *model.CombatTickResult, updatedRoom map[string]bool) {
	state := p.State
	p.State = "IDLE"
	p.StateTickCountdown = 0
	roomID := p.RoomID

	if state == "GATHERING" {
		p.Attributes.Experience += 4
		p.Attributes.Potential += 6
		itemID := "dry_ration"
		itemName := "干粮"
		// 采集收益按房间区分（见收益与经济设计 1.1）：默认城郊=古铜残片，芦苇滩=止血草
		switch roomID {
		case "yz_reed_marsh":
			if rand.Intn(100) < 60 {
				itemID = "reed_root"
				itemName = "芦根"
			}
		case "ly_reed_marsh":
			if rand.Intn(100) < 60 {
				itemID = "herb"
				itemName = "止血草"
			}
		default:
			if rand.Intn(100) < 60 {
				itemID = "bronze_fragment"
				itemName = "古铜残片"
			}
		}
		if p.Backpack == nil {
			p.Backpack = map[string]int{}
		}
		p.Backpack[itemID]++
		result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{
			PlayerID: p.ID,
			Message:  "你采集完毕，获得" + itemName + "×1，增长了4点经验、6点潜能。",
		})
	} else {
		p.Money += 20
		p.Attributes.Potential += 2
		result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{
			PlayerID: p.ID,
			Message:  "你干完了活计，赚得20两银子，也磨炼出2点潜能。",
		})
	}
	updatedRoom[roomID] = true
}

func (r *RuntimeManager) ProcessStatusTick() (model.CombatTickResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return model.CombatTickResult{}, err
	}

	result := model.CombatTickResult{}
	updatedRoom := map[string]bool{}

	for _, p := range players {
		roomID := p.RoomID
		// 采集/帮工：逐秒倒数，归零时一次性结算收益（见收益与经济设计文档 1.1）
		if p.State == "GATHERING" || p.State == "WORKING" {
			if p.StateTickCountdown > 0 {
				p.StateTickCountdown--
			}
			if p.StateTickCountdown > 0 {
				continue
			}
			r.settleWorldLaborNoLock(p, &result, updatedRoom)
			continue
		}
		if p.State == "HEALING" || p.State == "MEDITATING" || p.State == "LEARNING" {
			if p.StateTickCountdown <= 0 {
				p.StateTickCountdown = stateSettlementSeconds
			}
			p.StateTickCountdown--
			if p.StateTickCountdown > 0 {
				continue
			}
			p.StateTickCountdown = stateSettlementSeconds
		}
		switch p.State {
		case "HEALING":
			effectiveMaxHP := effectiveAttributesForPlayerNoLock(p).MaxHP
			if p.Attributes.Resources.HP >= effectiveMaxHP || p.Attributes.Resources.MP < 8 {
				p.State = "IDLE"
				p.StateTickCountdown = 0
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: p.CharacterName + "停止了运功疗伤。"})
				updatedRoom[roomID] = true
				continue
			}
			beforeHP := p.Attributes.Resources.HP
			p.Attributes.Resources.MP -= 8
			p.Attributes.Resources.HP += 20
			if p.Attributes.Resources.HP > effectiveMaxHP {
				p.Attributes.Resources.HP = effectiveMaxHP
			}
			recovered := p.Attributes.Resources.HP - beforeHP
			if recovered > 0 {
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你恢复了" + strconv.Itoa(recovered) + "点气血。"})
			}
			updatedRoom[roomID] = true
		case "MEDITATING":
			effTick := effectiveAttributesForPlayerNoLock(p)
			effectiveMaxMP := effTick.MaxMP
			if p.Attributes.Resources.MP < p.Attributes.Resources.TrainedMaxMP {
				beforeMP := p.Attributes.Resources.MP
				// 打坐回复量与基本内功等级挂钩，秘籍（如混元功）装备后追加恢复（见公式文档 1.4 / 3.3）
				recovery := 15 + p.Skills["basic_neigong"]/2 + effTick.MeditateRecoveryBonus
				p.Attributes.Resources.MP += recovery
				if p.Attributes.Resources.MP > p.Attributes.Resources.TrainedMaxMP {
					p.Attributes.Resources.MP = p.Attributes.Resources.TrainedMaxMP
				}
				recovered := p.Attributes.Resources.MP - beforeMP
				if recovered > 0 {
					result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你恢复了" + strconv.Itoa(recovered) + "点内力。"})
				}
				updatedRoom[roomID] = true
				continue
			}
			if p.Attributes.Resources.TrainedMaxMP >= effectiveMaxMP {
				p.State = "IDLE"
				p.StateTickCountdown = 0
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: p.CharacterName + "结束了打坐调息。"})
				updatedRoom[roomID] = true
				continue
			}
			beforeMaxMP := p.Attributes.Resources.TrainedMaxMP
			p.Attributes.Resources.TrainedMaxMP += 1
			if p.Attributes.Resources.TrainedMaxMP > effectiveMaxMP {
				p.Attributes.Resources.TrainedMaxMP = effectiveMaxMP
			}
			increased := p.Attributes.Resources.TrainedMaxMP - beforeMaxMP
			p.Attributes.Resources.MP = p.Attributes.Resources.TrainedMaxMP
			if increased > 0 {
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你增加了" + strconv.Itoa(increased) + "点最大内力。"})
			}
			updatedRoom[roomID] = true
		case "LEARNING":
			skillID := p.LearningSkillID
			skillDef, ok := fileSkills[skillID]
			if !ok || skillID == "" {
				stopLearningNoLock(p)
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你失去了学习的头绪，只得停了下来。"})
				updatedRoom[roomID] = true
				continue
			}
			if p.LearningTeacherLevel <= 0 {
				if teacher, exists := fileNpcs[p.LearningTeacherID]; exists {
					p.LearningTeacherLevel = teacher.Skills[skillID]
				} else if teacher, exists := r.dungeonNPCByIDNoLock(p.LearningTeacherID); exists {
					p.LearningTeacherLevel = teacher.Skills[skillID]
				}
			}
			if p.LearningTeacherLevel <= 0 {
				stopLearningNoLock(p)
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你已无法继续向这位老师学习" + skillDef.Name + "。"})
				updatedRoom[roomID] = true
				continue
			}
			levelCap := model.SkillLevelCapForExperience(p.Attributes.Experience)
			if p.Skills[skillID] >= p.LearningTeacherLevel {
				stopLearningNoLock(p)
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你的" + skillDef.Name + "已经不输你师傅了"})
				updatedRoom[roomID] = true
				continue
			}
			if p.Skills[skillID] >= levelCap {
				stopLearningNoLock(p)
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你的经验尚不足以继续领悟" + skillDef.Name + "，停止了学习。"})
				updatedRoom[roomID] = true
				continue
			}
			if p.Attributes.Potential <= 0 {
				stopLearningNoLock(p)
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你的潜能已经耗尽，无法继续学习" + skillDef.Name + "。"})
				updatedRoom[roomID] = true
				continue
			}
			if p.SkillLearningProgress == nil {
				p.SkillLearningProgress = map[string]int{}
			}
			targetLevel := p.Skills[skillID] + 1
			requiredPotential := model.SkillPotentialForTargetLevel(targetLevel)
			remainingPotential := requiredPotential - p.SkillLearningProgress[skillID]
			if remainingPotential <= 0 {
				remainingPotential = requiredPotential
				p.SkillLearningProgress[skillID] = 0
			}
			// 学习注入上限可被读书秘籍（如悟真篇）按装备规则提升（见公式文档 3.3 学习效率加成）
			investCap := model.LearningPotentialPerTick + effectiveAttributesForPlayerNoLock(p).LearningInvestBonus
			spentPotential := min(investCap, p.Attributes.Potential, remainingPotential)
			p.Attributes.Potential -= spentPotential
			p.SkillLearningProgress[skillID] += spentPotential
			if p.SkillLearningProgress[skillID] >= requiredPotential {
				p.SkillLearningProgress[skillID] = 0
				p.Skills[skillID]++
				message := "你的" + skillDef.Name + "等级提升了，现在是" + strconv.Itoa(p.Skills[skillID]) + "级。"
				if p.Skills[skillID] >= p.LearningTeacherLevel {
					stopLearningNoLock(p)
					message += "你的" + skillDef.Name + "已经不输你师傅了"
				} else if p.Skills[skillID] >= levelCap {
					stopLearningNoLock(p)
					message += "你已达到当前经验所能领悟的上限。"
				} else if p.Attributes.Potential <= 0 {
					stopLearningNoLock(p)
					message += "你的潜能已经耗尽。"
				}
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: message})
			} else {
				message := "你对" + skillDef.Name + "略有所悟。"
				if p.Attributes.Potential <= 0 {
					stopLearningNoLock(p)
					message += "你的潜能已经耗尽，停止了学习。"
				}
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: message})
			}
			updatedRoom[roomID] = true
		}
	}

	for roomID := range updatedRoom {
		result.UpdatedRoomIDs = append(result.UpdatedRoomIDs, roomID)
	}
	if err := r.savePlayers(players); err != nil {
		return model.CombatTickResult{}, err
	}
	return result, nil
}

func stopLearningNoLock(p *model.Player) {
	if p == nil {
		return
	}
	p.State = "IDLE"
	p.StateTickCountdown = 0
	p.LearningSkillID = ""
	p.LearningTeacherID = ""
	p.LearningTeacherLevel = 0
}
