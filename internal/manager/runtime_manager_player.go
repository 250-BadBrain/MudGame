package manager

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
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
	for id, level := range p.Skills {
		def, ok := fileSkills[id]
		if !ok {
			continue
		}
		out = append(out, map[string]interface{}{
			"id": id, "name": def.Name, "description": def.Description, "rarity": def.Rarity, "level": level,
			"type": def.Type, "category": def.Category, "school": def.School,
			"bonuses": def.Bonuses, "bonusText": skillBonusText(def.Bonuses, level), "requirements": def.Requirements,
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
	p.RoomID = ex.TargetRoomID
	r.applyEnterRoomDungeonObjectiveNoLock(p.RoomID)
	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
	return roomPayload, err
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
		npcName = npc.Name
		npcTitle = npc.Title
		for itemID, limit := range npc.ShopInventory {
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
		shopInventory = npc.ShopInventory
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
	if dnpc, _, ok := r.dungeonNPCInRoomNoLock(room.ID, npcID); ok {
		if _, ok := dnpc.Skills[skillID]; !ok {
			return errors.New("skill_not_taught")
		}
	} else {
		npc, ok := fileNpcs[npcID]
		if !ok {
			return errors.New("npc_not_found")
		}
		if _, ok := npc.Skills[skillID]; !ok {
			return errors.New("skill_not_taught")
		}
	}
	if p.Skills[skillID] >= 300 {
		return errors.New("skill_level_capped")
	}
	if p.Skills == nil {
		p.Skills = map[string]int{}
	}
	p.Attributes.Potential -= 1
	p.Skills[skillID] += 1
	p.State = "LEARNING"
	return r.savePlayers(players)
}

func (r *RuntimeManager) Mine(playerID string) (int, int, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return 0, 0, "", err
	}
	p, ok := players[playerID]
	if !ok {
		return 0, 0, "", exception.ErrPlayerNotFound
	}
	if p.State == "DEAD" {
		return 0, 0, "", errors.New("dead_state")
	}
	if p.State != "IDLE" {
		return 0, 0, "", errors.New("busy")
	}
	if p.RoomID != "ly_mine" {
		return 0, 0, "", errors.New("wrong_location")
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	if p.Equipment["WEAPON"] != "iron_pickaxe" {
		return 0, 0, "", errors.New("no_pickaxe")
	}
	p.State = "MINING"
	if err := r.savePlayers(players); err != nil {
		return 0, 0, "", err
	}
	return p.Attributes.Experience, p.Attributes.Potential, "", nil
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
	if !allowed {
		return nil, errors.New("action_not_available")
	}

	result := map[string]interface{}{"action": action}
	switch action {
	case "gather":
		expGain := 8
		potGain := 12
		p.Attributes.Experience += expGain
		p.Attributes.Potential += potGain
		result["message"] = "你在附近搜寻一番，找到些可用杂物，也增长了几分江湖经验。"
		result["experience"] = p.Attributes.Experience
		result["potential"] = p.Attributes.Potential
	case "work":
		moneyGain := 30
		potGain := 5
		p.Money += moneyGain
		p.Attributes.Potential += potGain
		result["message"] = "你接了些零散差事，换得几枚铜钱，也磨练了耐性。"
		result["money"] = p.Money
		result["potential"] = p.Attributes.Potential
	case "listen_rumor":
		result["message"] = roomRumor(room.ID)
	case "search":
		message, err := r.searchRoomNoLock(p, room)
		if err != nil {
			return nil, err
		}
		result["message"] = message
	default:
		return nil, errors.New("action_not_supported")
	}
	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	return result, nil
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
	if room.WorldID == "dungeon_xunqin_ch01_bronze_box" && strings.HasSuffix(room.ID, "_bronze_01") {
		if p.Flags["xunqin_ch01_completed"] {
			return "你又仔细搜了一遍铜匣暗室，只找到些无用铜锈。", nil
		}
		p.Backpack["bronze_fragment"] += 1
		p.Flags["xunqin_ch01_completed"] = true
		return "你在铜匣暗格中摸到一枚古铜残片，残片入手微凉，纹路似在烛光下缓缓流动。", nil
	}
	return "你仔细搜索四周，没有发现值得带走的东西。", nil
}

func roomRumor(roomID string) string {
	switch roomID {
	case "ly_00":
		return "你听见桥边有人低声谈起扬州来的两个少年，说他们身上似乎牵着一场大风波。"
	case "ly_west_market":
		return "西市掮客说，最近有人在暗中收购战国铜器拓片，出价高得反常。"
	case "ly_inn":
		return "说书人醒木一拍，正讲到一只来历不明的战国铜匣，满堂酒客都屏住了呼吸。"
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
	if p.State == "MINING" {
		return "", errors.New("mining_state")
	}
	if p.State == "LEARNING" {
		return "", errors.New("learning_state")
	}
	if p.Attributes.Resources.HP >= p.Attributes.Resources.MaxHP {
		return "", errors.New("hp_full")
	}
	if p.Attributes.Resources.MP < 10 {
		return "", errors.New("mp_insufficient")
	}
	p.Attributes.Resources.MP -= 10
	p.Attributes.Resources.HP += 20
	if p.Attributes.Resources.HP > p.Attributes.Resources.MaxHP {
		p.Attributes.Resources.HP = p.Attributes.Resources.MaxHP
	}
	p.State = "HEALING"
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你开始运功疗伤。", nil
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
	if p.State == "MINING" {
		return "", errors.New("mining_state")
	}
	if p.State == "LEARNING" {
		return "", errors.New("learning_state")
	}
	if p.Attributes.Resources.TrainedMaxMP >= p.Attributes.Resources.MaxMP && p.Attributes.Resources.MP >= p.Attributes.Resources.TrainedMaxMP {
		return "", errors.New("full_state")
	}
	p.Attributes.Resources.MP += 15
	if p.Attributes.Resources.MP > p.Attributes.Resources.TrainedMaxMP {
		p.Attributes.Resources.MP = p.Attributes.Resources.TrainedMaxMP
	}
	p.State = "MEDITATING"
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
			p.TargetID = ""
			p.CombatType = ""
			_ = r.savePlayers(players)
			switch oldState {
			case "MINING":
				return "你收起铁镐，停止了挖矿。"
			case "HEALING":
				return "你停止了运功疗伤。"
			case "MEDITATING":
				return "你停止了打坐。"
			case "LEARNING":
				return "你停止了学习。"
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
		if dnpc.RoomID != p.RoomID {
			return "", errors.New("target_not_in_room")
		}
		p.State = "COMBAT"
		p.TargetID = dnpc.ID
		p.CombatType = "KILL"
		if err := r.savePlayers(players); err != nil {
			return "", err
		}
		return p.CharacterName + "对" + dnpc.Name + "喝道：纳命来！", nil
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
	p.Attributes.Resources.HP = p.Attributes.Resources.MaxHP
	p.Attributes.Resources.MP = p.Attributes.Resources.TrainedMaxMP
	p.RoomID = "ly_00"
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你苏醒了过来。", nil
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
		switch p.State {
		case "HEALING":
			if p.Attributes.Resources.HP >= p.Attributes.Resources.MaxHP || p.Attributes.Resources.MP < 8 {
				p.State = "IDLE"
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: p.CharacterName + "停止了运功疗伤。"})
				updatedRoom[roomID] = true
				continue
			}
			beforeHP := p.Attributes.Resources.HP
			p.Attributes.Resources.MP -= 8
			p.Attributes.Resources.HP += 20
			if p.Attributes.Resources.HP > p.Attributes.Resources.MaxHP {
				p.Attributes.Resources.HP = p.Attributes.Resources.MaxHP
			}
			recovered := p.Attributes.Resources.HP - beforeHP
			if recovered > 0 {
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你恢复了" + strconv.Itoa(recovered) + "点气血。"})
			}
			updatedRoom[roomID] = true
		case "MEDITATING":
			if p.Attributes.Resources.MP < p.Attributes.Resources.TrainedMaxMP {
				beforeMP := p.Attributes.Resources.MP
				p.Attributes.Resources.MP += 20
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
			if p.Attributes.Resources.TrainedMaxMP >= p.Attributes.Resources.MaxMP {
				p.State = "IDLE"
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: p.CharacterName + "结束了打坐调息。"})
				updatedRoom[roomID] = true
				continue
			}
			beforeMaxMP := p.Attributes.Resources.TrainedMaxMP
			p.Attributes.Resources.TrainedMaxMP += 1
			if p.Attributes.Resources.TrainedMaxMP > p.Attributes.Resources.MaxMP {
				p.Attributes.Resources.TrainedMaxMP = p.Attributes.Resources.MaxMP
			}
			increased := p.Attributes.Resources.TrainedMaxMP - beforeMaxMP
			p.Attributes.Resources.MP = p.Attributes.Resources.TrainedMaxMP
			if increased > 0 {
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你增加了" + strconv.Itoa(increased) + "点最大内力。"})
			}
			updatedRoom[roomID] = true
		case "MINING":
			if p.Equipment["WEAPON"] != "iron_pickaxe" {
				p.State = "IDLE"
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: p.CharacterName + "收起铁镐，停止了挖矿。"})
				updatedRoom[roomID] = true
				continue
			}
			expGain := 20
			potGain := 40
			p.Attributes.Experience += expGain
			p.Attributes.Potential += potGain
			gemID := "cracked_red_gem"
			if _, ok := fileItems[gemID]; !ok {
				gemID = "gem_red"
			}
			if _, ok := fileItems[gemID]; ok {
				if p.Backpack == nil {
					p.Backpack = map[string]int{}
				}
				p.Backpack[gemID] += 1
			}
			result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你获得了" + strconv.Itoa(expGain) + "点经验，" + strconv.Itoa(potGain) + "点潜能。"})
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
