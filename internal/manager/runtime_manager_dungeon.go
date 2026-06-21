package manager

import (
	"errors"
	"sort"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

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
	ids := r.dungeonMgr.DungeonTemplateIDsNoLock()
	sort.Strings(ids)
	out := make([]map[string]interface{}, 0, len(ids))
	for _, id := range ids {
		tpl, ok := r.dungeonMgr.DungeonTemplateNoLock(id)
		if !ok {
			continue
		}
		if !dungeonUnlockedForPlayer(player, tpl.UnlockConditions) {
			continue
		}
		order := tpl.Order
		if order <= 0 {
			order = 1
		}
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
			"chapter":     tpl.Chapter,
			"objectives":  objectives,
			"rewards":     tpl.Rewards,
			"rewardBase":  order * 100,
		})
	}
	return out
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

func (r *RuntimeManager) destroyPlayerDungeonInstanceNoLock(playerID string) *engruntime.Instance {
	if r.dungeonMgr == nil {
		return nil
	}
	return r.dungeonMgr.DestroyPlayerInstanceNoLock(playerID)
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

func (r *RuntimeManager) isDungeonRoomIDNoLock(roomID string) bool {
	if r.dungeonMgr == nil {
		return false
	}
	return r.dungeonMgr.IsDungeonRoomIDNoLock(roomID)
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
	return map[string]interface{}{
		"id":           inst.DungeonID,
		"name":         inst.Name,
		"progress":     inst.Progress,
		"sequence":     inst.Sequence,
		"puzzleStates": inst.PuzzleStates,
		"alertLevel":   inst.AlertLevel,
	}, true
}

func (r *RuntimeManager) removeDungeonNPCFromRoomNoLock(inst *engruntime.Instance, npcID string) {
	if r.dungeonMgr == nil {
		return
	}
	r.dungeonMgr.RemoveDungeonNPCFromRoomNoLock(inst, npcID)
}

func (r *RuntimeManager) firstAggressiveDungeonNPCInRoomNoLock(roomID string) (*engruntime.NPCState, bool) {
	if r.dungeonMgr == nil {
		return nil, false
	}
	return r.dungeonMgr.FirstAggressiveDungeonNPCInRoomNoLock(roomID)
}

func (r *RuntimeManager) dungeonBaseRewardNoLock(inst *engruntime.Instance) int {
	if r.dungeonMgr == nil {
		return 0
	}
	return r.dungeonMgr.DungeonBaseRewardNoLock(inst)
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

func formatDungeonSettlementMessage(settlement engruntime.Settlement) string {
	return engruntime.FormatDungeonSettlementMessage(settlement)
}

func applyDungeonSettlementReward(p *model.Player, settlement engruntime.Settlement) {
	if p == nil {
		return
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

func (r *RuntimeManager) forceLeaveDungeonOnDeathNoLock(players map[string]*model.Player, playerID string) *engruntime.Settlement {
	if r.dungeonMgr == nil {
		return nil
	}
	instanceID, ok := r.dungeonMgr.PlayerInstanceIDNoLock(playerID)
	if !ok {
		return nil
	}
	inst, exists := r.dungeonMgr.DungeonInstanceNoLock(instanceID)
	if !exists || inst == nil {
		r.destroyDungeonInstanceNoLock(instanceID)
		return nil
	}
	settlement := r.buildDungeonSettlementNoLock(inst)
	p, pok := players[playerID]
	if pok && p != nil {
		fallback := inst.EntryRoomID
		if fallback == "" {
			fallback = "ly_00"
		}
		p.RoomID = fallback
		applyDungeonSettlementReward(p, settlement)
	}
	r.destroyDungeonInstanceNoLock(instanceID)
	return &settlement
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
	}

	inst, err := r.createDungeonInstanceNoLock(playerID, dungeonID, p.RoomID)
	if err != nil {
		return nil, err
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
