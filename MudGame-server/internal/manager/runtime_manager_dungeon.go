package manager

import (
	"errors"
	"sort"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

// dungeonTemplatesNoLock 返回按 (order, id) 排序的全部副本模板。
func (r *RuntimeManager) dungeonTemplatesNoLock() []engruntime.Template {
	ids := r.dungeonMgr.DungeonTemplateIDsNoLock()
	templates := make([]engruntime.Template, 0, len(ids))
	for _, id := range ids {
		if tpl, ok := r.dungeonMgr.DungeonTemplateNoLock(id); ok {
			templates = append(templates, tpl)
		}
	}
	sort.Slice(templates, func(i, j int) bool {
		oi, oj := templates[i].Order, templates[j].Order
		if oi <= 0 {
			oi = 1
		}
		if oj <= 0 {
			oj = 1
		}
		if oi != oj {
			return oi < oj
		}
		return templates[i].ID < templates[j].ID
	})
	return templates
}

func (r *RuntimeManager) GetDungeons(playerID string) []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.dungeonMgr == nil {
		return []map[string]interface{}{}
	}
	players, err := r.loadPlayers()
	if err != nil {
		return []map[string]interface{}{}
	}
	player := players[playerID]

	templates := r.dungeonTemplatesNoLock()
	out := make([]map[string]interface{}, 0, len(templates))
	for _, tpl := range templates {
		order := tpl.Order
		if order <= 0 {
			order = 1
		}
		locked, lockHint := r.dungeonLockStateNoLock(player, tpl, templates)
		objectives := make([]map[string]interface{}, 0, len(tpl.Objectives))
		for _, obj := range tpl.Objectives {
			objectives = append(objectives, map[string]interface{}{
				"id":             obj.ID,
				"type":           obj.Type,
				"targetId":       obj.TargetID,
				"required":       obj.Required,
				"description":    obj.Description,
				"optional":       obj.Optional,
				"exclusiveGroup": obj.ExclusiveGroup,
			})
		}
		out = append(out, map[string]interface{}{
			"id":          tpl.ID,
			"name":        tpl.Name,
			"description": tpl.Description,
			"order":       order,
			"chainId":     tpl.ChainID,
			"chainName":   tpl.ChainName,
			"chapter":     tpl.Chapter,
			"objectives":  objectives,
			"rewards":     tpl.Rewards,
			"rewardBase":  order * 100,
			"locked":      locked,
			"lockHint":    lockHint,
		})
	}
	return out
}

// dungeonLockStateNoLock 综合显式 unlockConditions 与章节链规则判定锁定状态。
// 章节链：order N 需任一 order N-1 副本达成一次 100% 通关（dungeon_cleared_<id> 旗标）；
// 若不存在 order N-1 的副本（序号稀疏），不构成锁定。
func (r *RuntimeManager) dungeonLockStateNoLock(player *model.Player, tpl engruntime.Template, templates []engruntime.Template) (bool, string) {
	if !dungeonUnlockedForPlayer(player, tpl.UnlockConditions) {
		return true, "尚未满足进入条件"
	}
	order := tpl.Order
	if order <= 1 {
		return false, ""
	}
	prevOrderExists := false
	prevCleared := false
	firstPendingName := ""
	for _, other := range templates {
		if other.Order != order-1 {
			continue
		}
		prevOrderExists = true
		if player != nil && player.Flags["dungeon_cleared_"+other.ID] {
			prevCleared = true
			break
		}
		if firstPendingName == "" {
			firstPendingName = other.Name
		}
	}
	if prevOrderExists && !prevCleared {
		return true, "需先通关《" + firstPendingName + "》"
	}
	return false, ""
}

func dungeonUnlockedForPlayer(p *model.Player, conditions []engruntime.UnlockCondition) bool {
	if len(conditions) == 0 {
		return true
	}
	if p == nil {
		return false
	}
	for _, cond := range conditions {
		switch cond.Type {
		case "flag":
			want := true
			if cond.Value == "false" {
				want = false
			}
			if p.Flags[cond.TargetID] != want {
				return false
			}
		case "reputation":
			if p.Reputations[cond.TargetID] < cond.Min {
				return false
			}
		case "item":
			required := cond.Min
			if required <= 0 {
				required = 1
			}
			if p.Backpack[cond.TargetID] < required {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func (r *RuntimeManager) loadDungeonTemplatesNoLock() error {
	if r.dungeonMgr == nil {
		return nil
	}
	return r.dungeonMgr.LoadTemplatesNoLock()
}

func (r *RuntimeManager) createDungeonInstanceNoLock(playerID, dungeonID, entryRoomID string) (*engruntime.Instance, error) {
	if r.dungeonMgr == nil {
		return nil, errors.New("dungeon_unavailable")
	}
	return r.dungeonMgr.CreateInstanceNoLock(playerID, dungeonID, entryRoomID)
}

func (r *RuntimeManager) destroyDungeonInstanceNoLock(instanceID string) {
	if r.dungeonMgr == nil {
		return
	}
	r.dungeonMgr.DestroyInstanceNoLock(instanceID)
}

func (r *RuntimeManager) roomByIDNoLock(roomID string, staticRooms map[string]model.Room) (model.Room, bool) {
	if r.dungeonMgr != nil {
		if room, ok := r.dungeonMgr.RoomByIDNoLock(roomID, staticRooms); ok {
			return room, true
		}
	}
	room, ok := staticRooms[roomID]
	return room, ok
}

func (r *RuntimeManager) dungeonNPCByIDNoLock(npcID string) (*engruntime.NPCState, bool) {
	if r.dungeonMgr == nil {
		return nil, false
	}
	return r.dungeonMgr.DungeonNPCByIDNoLock(npcID)
}

func (r *RuntimeManager) dungeonNPCInRoomNoLock(roomID, npcID string) (*engruntime.NPCState, *engruntime.Instance, bool) {
	if r.dungeonMgr == nil {
		return nil, nil, false
	}
	npc, ok := r.dungeonMgr.DungeonNPCByIDNoLock(npcID)
	if !ok || npc == nil || !npc.Alive || npc.RoomID != roomID {
		return nil, nil, false
	}
	instanceID, ok := r.dungeonMgr.RoomInstanceIDNoLock(roomID)
	if !ok {
		return nil, nil, false
	}
	inst, ok := r.dungeonMgr.DungeonInstanceNoLock(instanceID)
	if !ok || inst == nil {
		return nil, nil, false
	}
	return npc, inst, true
}

func (r *RuntimeManager) defeatDungeonNPCNoLock(npcID string) (*engruntime.Instance, *engruntime.NPCState, int, bool) {
	if r.dungeonMgr == nil {
		return nil, nil, 0, false
	}
	return r.dungeonMgr.DefeatNPCNoLock(npcID)
}

func (r *RuntimeManager) dungeonPayloadByRoomIDNoLock(roomID string) (map[string]interface{}, bool) {
	if r.dungeonMgr == nil {
		return nil, false
	}
	instanceID, ok := r.dungeonMgr.RoomInstanceIDNoLock(roomID)
	if !ok {
		return nil, false
	}
	inst, ok := r.dungeonMgr.DungeonInstanceNoLock(instanceID)
	if !ok || inst == nil {
		return nil, false
	}
	// 深拷贝机关状态：payload 会在锁外被 JSON 序列化，直接引用实例的活 map
	// 会与持写锁的机关指令产生并发读写
	puzzleStates := make(map[string]string, len(inst.PuzzleStates))
	for puzzleID, state := range inst.PuzzleStates {
		puzzleStates[puzzleID] = state
	}
	return map[string]interface{}{
		"id":           inst.DungeonID,
		"name":         inst.Name,
		"progress":     inst.Progress,
		"sequence":     inst.Sequence,
		"puzzleStates": puzzleStates,
	}, true
}

func (r *RuntimeManager) buildDungeonSettlementNoLock(inst *engruntime.Instance) engruntime.Settlement {
	if r.dungeonMgr == nil {
		return engruntime.Settlement{}
	}
	return r.dungeonMgr.BuildDungeonSettlementNoLock(inst)
}

func (r *RuntimeManager) applyEnterRoomDungeonObjectiveNoLock(roomID string) int {
	if r.dungeonMgr == nil {
		return 0
	}
	return r.dungeonMgr.ApplyEnterRoomObjectiveNoLock(roomID)
}

func (r *RuntimeManager) applyCollectDungeonObjectiveNoLock(playerID, itemID string) int {
	if r.dungeonMgr == nil {
		return 0
	}
	return r.dungeonMgr.ApplyCollectObjectiveNoLock(playerID, itemID)
}

func applyDungeonSettlementReward(p *model.Player, settlement engruntime.Settlement) {
	if p == nil {
		return
	}
	// 章节链通关旗标：100% 结算时写入，驱动 order N+1 解锁（见 economy_design.md 1.3）
	if settlement.Progress >= 100 {
		if p.Flags == nil {
			p.Flags = map[string]bool{}
		}
		p.Flags["dungeon_cleared_"+settlement.DungeonID] = true
	}
	p.Attributes.Experience += settlement.RewardExperience
	p.Attributes.Potential += settlement.RewardPotential
	p.Money += settlement.RewardMoney
	if len(settlement.RewardItems) > 0 {
		if p.Backpack == nil {
			p.Backpack = map[string]int{}
		}
		for itemID, amount := range settlement.RewardItems {
			if amount > 0 {
				p.Backpack[itemID] += amount
			}
		}
	}
	if len(settlement.RewardFlags) > 0 {
		if p.Flags == nil {
			p.Flags = map[string]bool{}
		}
		for flag, value := range settlement.RewardFlags {
			p.Flags[flag] = value
		}
	}
	if len(settlement.RewardReputations) > 0 {
		if p.Reputations == nil {
			p.Reputations = map[string]int{}
		}
		for factionID, value := range settlement.RewardReputations {
			p.Reputations[factionID] += value
		}
	}
}

func (r *RuntimeManager) processDungeonAggroNoLock(players map[string]*model.Player, result *model.CombatTickResult, updatedRoom map[string]bool) {
	if r.dungeonMgr == nil {
		return
	}
	r.dungeonMgr.ProcessDungeonAggroNoLock(players, result, updatedRoom)
}

func intToString(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func (r *RuntimeManager) EnterDungeon(playerID, dungeonID string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.dungeonLoadErr != nil {
		return nil, errors.New("dungeon_unavailable")
	}
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
	if tpl, ok := r.dungeonMgr.DungeonTemplateNoLock(dungeonID); ok {
		if !dungeonUnlockedForPlayer(p, tpl.UnlockConditions) {
			return nil, errors.New("dungeon_locked")
		}
		if locked, hint := r.dungeonLockStateNoLock(p, tpl, r.dungeonTemplatesNoLock()); locked {
			return nil, errors.New(hint)
		}
	}

	inst, err := r.createDungeonInstanceNoLock(playerID, dungeonID, p.RoomID)
	if err != nil {
		return nil, err
	}
	for _, itemID := range inst.TemporaryItems {
		delete(p.Backpack, itemID)
	}
	spawn := inst.SpawnRoomID
	if spawn == "" {
		spawn = p.RoomID
	}
	p.RoomID = spawn
	p.State = "IDLE"
	p.StateTickCountdown = 0
	p.LearningSkillID = ""
	p.LearningTeacherID = ""
	p.LearningTeacherLevel = 0
	p.TargetID = ""
	p.CombatType = ""
	r.applyEnterRoomDungeonObjectiveNoLock(p.RoomID)

	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
	if err != nil {
		return nil, err
	}
	return roomPayload, nil
}

func (r *RuntimeManager) LeaveDungeon(playerID string) (map[string]interface{}, error) {
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
	instanceID, inDungeon := r.dungeonMgr.PlayerInstanceIDNoLock(playerID)
	if !inDungeon {
		return nil, errors.New("not_in_dungeon")
	}
	inst, _ := r.dungeonMgr.DungeonInstanceNoLock(instanceID)
	var settlement *engruntime.Settlement
	if inst != nil {
		s := r.buildDungeonSettlementNoLock(inst)
		settlement = &s
		applyDungeonSettlementReward(p, s)
		for _, itemID := range inst.TemporaryItems {
			delete(p.Backpack, itemID)
		}
	}
	fallback := "ly_00"
	if inst != nil && inst.EntryRoomID != "" {
		fallback = inst.EntryRoomID
	}
	p.RoomID = fallback
	p.State = "IDLE"
	p.StateTickCountdown = 0
	p.LearningSkillID = ""
	p.LearningTeacherID = ""
	p.LearningTeacherLevel = 0
	p.TargetID = ""
	p.CombatType = ""
	r.destroyDungeonInstanceNoLock(instanceID)

	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
	if err != nil {
		return nil, err
	}
	if settlement != nil {
		roomPayload["dungeonSettlement"] = map[string]interface{}{
			"dungeonId":        settlement.DungeonID,
			"dungeonName":      settlement.DungeonName,
			"sequence":         settlement.Sequence,
			"chainId":          settlement.ChainID,
			"chapter":          settlement.Chapter,
			"progress":         settlement.Progress,
			"rewardExperience": settlement.RewardExperience,
			"rewardPotential":  settlement.RewardPotential,
			"rewardMoney":      settlement.RewardMoney,
			"rewardItems":      settlement.RewardItems,
		}
	}
	return roomPayload, nil
}
