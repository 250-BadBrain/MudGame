package manager

import (
	"errors"
	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
	"sort"
	"time"
)

func canonicalQuestID(id string) string {
	if q, ok := fileQuests[id]; ok && q.AliasOf != "" {
		return q.AliasOf
	}
	return id
}

func questStateNoLock(p *model.Player, id string) model.QuestState {
	id = canonicalQuestID(id)
	state := p.QuestStates[id]
	state.ID = id
	if state.Status == "" {
		state.Status = "not_started"
	}
	if p.Flags[id+"_completed"] {
		state.Status = "completed"
	}
	for alias, q := range fileQuests {
		if q.AliasOf == id {
			old := p.QuestStates[alias]
			if old.Status == "completed" || p.Flags[alias+"_completed"] {
				state.Status = "completed"
				state.CompletedAt = max(state.CompletedAt, old.CompletedAt)
			}
			if state.Status == "not_started" && old.Status == "active" {
				state.Status = "active"
			}
		}
	}
	if state.Status == "completed" {
		state.Progress = 100
	}
	return state
}

func (r *RuntimeManager) questLockHintNoLock(p *model.Player, q model.QuestDef) string {
	if q.SourceDungeon == "" || p.Flags["dungeon_cleared_"+q.SourceDungeon] {
		return ""
	}
	templates := r.dungeonTemplatesNoLock()
	for _, tpl := range templates {
		if tpl.ID == q.SourceDungeon {
			if locked, hint := r.dungeonLockStateNoLock(p, tpl, templates); locked {
				return hint
			}
			return ""
		}
	}
	return "任务来源尚未开放"
}

func (r *RuntimeManager) startQuestNoLock(p *model.Player, id string) string {
	id = canonicalQuestID(id)
	q, ok := fileQuests[id]
	if !ok {
		return ""
	}
	state := questStateNoLock(p, id)
	if state.Status == "active" || (state.Status == "completed" && !q.Repeatable) || r.questLockHintNoLock(p, q) != "" {
		return ""
	}
	if p.QuestStates == nil {
		p.QuestStates = map[string]model.QuestState{}
	}
	p.QuestStates[id] = model.QuestState{ID: id, Status: "active"}
	return "你接下了任务：" + q.Name
}

func (r *RuntimeManager) questCompletionErrorNoLock(p *model.Player, id, npcID string) error {
	id = canonicalQuestID(id)
	q, ok := fileQuests[id]
	if !ok {
		return errors.New("quest_not_found")
	}
	if q.TurnInNPCID != "" && q.TurnInNPCID != npcID {
		return errors.New("quest_wrong_npc")
	}
	if questStateNoLock(p, id).Status != "active" {
		return errors.New("quest_not_active")
	}
	if !r.questObjectivesMetNoLock(p, q) {
		return errors.New("quest_not_finished")
	}
	return nil
}

func (r *RuntimeManager) completeQuestNoLock(p *model.Player, id, npcID string) ([]string, error) {
	id = canonicalQuestID(id)
	if err := r.questCompletionErrorNoLock(p, id, npcID); err != nil {
		return nil, err
	}
	q := fileQuests[id]
	state := questStateNoLock(p, id)
	state.Status = "completed"
	state.Progress = 100
	state.CompletedAt = time.Now().Unix()
	p.QuestStates[id] = state
	p.Attributes.Experience += q.Rewards.Experience
	p.Attributes.Potential += q.Rewards.Potential
	p.Money += q.Rewards.Money
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	for item, n := range q.Rewards.Items {
		if n > 0 {
			p.Backpack[item] += n
		}
	}
	if p.Flags == nil {
		p.Flags = map[string]bool{}
	}
	for flag, value := range q.Rewards.Flags {
		p.Flags[flag] = value
	}
	if p.Reputations == nil {
		p.Reputations = map[string]int{}
	}
	for faction, value := range q.Rewards.Reputations {
		p.Reputations[faction] += value
	}
	return []string{"你完成了任务：" + q.Name, "你获得了任务奖励。"}, nil
}

func (r *RuntimeManager) questObjectivesMetNoLock(p *model.Player, q model.QuestDef) bool {
	if len(q.Objectives) == 0 {
		return false
	}
	for _, o := range q.Objectives {
		if o.Type != "flag" || !p.Flags[o.TargetID] {
			return false
		}
	}
	return true
}

func questPayload(q model.QuestDef, state model.QuestState) map[string]interface{} {
	objectives := make([]map[string]interface{}, 0, len(q.Objectives))
	for _, o := range q.Objectives {
		objectives = append(objectives, map[string]interface{}{"type": o.Type, "targetId": o.TargetID, "description": o.Description, "required": o.Required})
	}
	return map[string]interface{}{"id": q.ID, "name": q.Name, "description": q.Description, "status": state.Status, "progress": state.Progress, "objectives": objectives, "rewards": q.Rewards}
}

func npcLocation(npcID string) string {
	for _, room := range fileRooms {
		for _, nid := range room.NPCIDs {
			if nid == npcID {
				return room.WorldName + "·" + room.Name + "·" + fileNpcs[npcID].Name
			}
		}
	}
	return fileNpcs[npcID].Name
}

func (r *RuntimeManager) GetQuests(playerID string) ([]map[string]interface{}, error) {
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
	out := make([]map[string]interface{}, 0, len(fileQuests))
	for id, q := range fileQuests {
		if q.AliasOf != "" {
			continue
		}
		state := questStateNoLock(p, id)
		row := questPayload(q, state)
		for index, obj := range q.Objectives {
			row["objectives"].([]map[string]interface{})[index]["completed"] = state.Status == "completed" || p.Flags[obj.TargetID]
		}
		completed := 0
		for _, o := range q.Objectives {
			if p.Flags[o.TargetID] {
				completed++
			}
		}
		if state.Status == "active" {
			row["progress"] = 100 * completed / len(q.Objectives)
		}
		row["canTurnIn"] = state.Status == "active" && r.questObjectivesMetNoLock(p, q)
		hint := ""
		if state.Status == "not_started" {
			hint = r.questLockHintNoLock(p, q)
		}
		row["locked"] = hint != ""
		row["lockHint"] = hint
		row["giverLocation"] = npcLocation(q.GiverNPCID)
		row["turnInLocation"] = npcLocation(q.TurnInNPCID)
		row["giverNpcId"] = q.GiverNPCID
		row["turnInNpcId"] = q.TurnInNPCID
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		rank := func(x map[string]interface{}) int {
			if x["canTurnIn"].(bool) {
				return 0
			}
			switch x["status"] {
			case "active":
				return 1
			case "not_started":
				if !x["locked"].(bool) {
					return 2
				}
				return 3
			default:
				return 4
			}
		}
		if a, b := rank(out[i]), rank(out[j]); a != b {
			return a < b
		}
		return out[i]["id"].(string) < out[j]["id"].(string)
	})
	return out, nil
}
