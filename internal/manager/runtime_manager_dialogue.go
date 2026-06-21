package manager

import (
	"errors"
	"sort"
	"time"

	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

func (r *RuntimeManager) Talk(playerID, npcID string) (map[string]interface{}, error) {
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
	npc, err := r.npcInPlayerRoomNoLock(p, npcID)
	if err != nil {
		return nil, err
	}
	if npc.DialogueID == "" {
		return nil, errors.New("dialogue_not_found")
	}
	dialogue, ok := fileDialogues[npc.DialogueID]
	if !ok {
		return nil, errors.New("dialogue_not_found")
	}
	return r.dialoguePayloadNoLock(p, npc, dialogue), nil
}

func (r *RuntimeManager) ChooseDialogue(playerID, npcID, optionID string) (map[string]interface{}, string, []string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, "", nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, "", nil, exception.ErrPlayerNotFound
	}
	if p.QuestStates == nil {
		p.QuestStates = map[string]model.QuestState{}
	}
	if p.Flags == nil {
		p.Flags = map[string]bool{}
	}
	if p.Reputations == nil {
		p.Reputations = map[string]int{}
	}
	npc, err := r.npcInPlayerRoomNoLock(p, npcID)
	if err != nil {
		return nil, "", nil, err
	}
	dialogue, ok := fileDialogues[npc.DialogueID]
	if !ok {
		return nil, "", nil, errors.New("dialogue_not_found")
	}
	var selected *model.DialogueOption
	for i := range dialogue.Options {
		opt := dialogue.Options[i]
		if opt.ID == optionID && r.dialogueRequirementMetNoLock(p, opt.Requires) {
			selected = &opt
			break
		}
	}
	if selected == nil {
		return nil, "", nil, errors.New("dialogue_option_not_found")
	}

	logs := []string{}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	for itemID, amount := range selected.TakeItems {
		if amount <= 0 {
			continue
		}
		p.Backpack[itemID] -= amount
		if p.Backpack[itemID] <= 0 {
			delete(p.Backpack, itemID)
		}
	}
	for itemID, amount := range selected.GiveItems {
		if amount <= 0 {
			continue
		}
		p.Backpack[itemID] += amount
		r.applyCollectDungeonObjectiveNoLock(p.ID, itemID)
	}
	for flag, value := range selected.SetFlags {
		p.Flags[flag] = value
	}
	if selected.StartQuestID != "" {
		if log := r.startQuestNoLock(p, selected.StartQuestID); log != "" {
			logs = append(logs, log)
		}
	}
	if selected.CompleteQuestID != "" {
		questLogs, err := r.completeQuestNoLock(p, selected.CompleteQuestID)
		if err != nil {
			return nil, "", nil, err
		}
		logs = append(logs, questLogs...)
	}

	if err := r.savePlayers(players); err != nil {
		return nil, "", nil, err
	}
	payload := r.dialoguePayloadNoLock(p, npc, dialogue)
	return payload, selected.Response, logs, nil
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
	out := make([]map[string]interface{}, 0, len(p.QuestStates))
	for questID, state := range p.QuestStates {
		quest, ok := fileQuests[questID]
		if !ok {
			continue
		}
		out = append(out, questPayload(quest, state))
	}
	return out, nil
}

func (r *RuntimeManager) npcInPlayerRoomNoLock(p *model.Player, npcID string) (model.NPC, error) {
	if p.State == "DEAD" {
		return model.NPC{}, errors.New("dead_state")
	}
	staticRooms, err := r.loadRooms()
	if err != nil {
		return model.NPC{}, err
	}
	room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
	if !ok {
		return model.NPC{}, exception.ErrRoomNotFound
	}
	for _, id := range room.NPCIDs {
		if id != npcID {
			continue
		}
		if npc, ok := fileNpcs[id]; ok {
			state, _ := r.staticNPCStateNoLock(id, room.ID)
			if state != nil && !state.Alive {
				return model.NPC{}, errors.New("target_is_dead")
			}
			return npc, nil
		}
		if dnpc, ok := r.dungeonNPCByIDNoLock(id); ok {
			if !dnpc.Alive {
				return model.NPC{}, errors.New("target_is_dead")
			}
			return model.NPC{
				ID:          dnpc.ID,
				Name:        dnpc.Name,
				Title:       dnpc.Title,
				Description: dnpc.Description,
				DialogueID:  dnpc.DialogueID,
				IsShop:      dnpc.IsShop,
				Skills:      dnpc.Skills,
				Attributes:  dnpc.Attributes,
			}, nil
		}
	}
	return model.NPC{}, errors.New("npc_not_in_room")
}

func (r *RuntimeManager) dialoguePayloadNoLock(p *model.Player, npc model.NPC, dialogue model.DialogueDef) map[string]interface{} {
	options := make([]map[string]interface{}, 0, len(dialogue.Options))
	for _, opt := range dialogue.Options {
		if !r.dialogueRequirementMetNoLock(p, opt.Requires) {
			continue
		}
		options = append(options, map[string]interface{}{
			"id":   opt.ID,
			"text": opt.Text,
		})
	}
	return map[string]interface{}{
		"id":      dialogue.ID,
		"npcId":   npc.ID,
		"npcName": npc.Name,
		"title":   dialogue.Title,
		"opening": dialogue.Opening,
		"options": options,
	}
}

func (r *RuntimeManager) dialogueRequirementMetNoLock(p *model.Player, req model.DialogueRequire) bool {
	if req.QuestID != "" {
		state, ok := p.QuestStates[req.QuestID]
		if !ok {
			return req.QuestStatus == "not_started"
		}
		if req.QuestStatus != "" && state.Status != req.QuestStatus {
			return false
		}
	}
	if req.Flag != "" {
		actual := p.Flags[req.Flag]
		expected := true
		if req.FlagValue != nil {
			expected = *req.FlagValue
		}
		if actual != expected {
			return false
		}
	}
	if req.ReputationID != "" {
		if p.Reputations[req.ReputationID] < req.ReputationMin {
			return false
		}
	}
	for itemID, amount := range req.Items {
		if amount <= 0 {
			continue
		}
		if p.Backpack[itemID] < amount {
			return false
		}
	}
	return true
}

func (r *RuntimeManager) startQuestNoLock(p *model.Player, questID string) string {
	quest, ok := fileQuests[questID]
	if !ok {
		return ""
	}
	state, exists := p.QuestStates[questID]
	if exists && state.Status != "completed" {
		return ""
	}
	if exists && state.Status == "completed" && !quest.Repeatable {
		return ""
	}
	p.QuestStates[questID] = model.QuestState{ID: questID, Status: "active", Progress: 0}
	return "你接下了任务：" + quest.Name
}

func (r *RuntimeManager) completeQuestNoLock(p *model.Player, questID string) ([]string, error) {
	quest, ok := fileQuests[questID]
	if !ok {
		return nil, errors.New("quest_not_found")
	}
	state, ok := p.QuestStates[questID]
	if !ok || state.Status != "active" {
		return nil, errors.New("quest_not_active")
	}
	if !r.questObjectivesMetNoLock(p, quest) {
		return nil, errors.New("quest_not_finished")
	}
	state.Status = "completed"
	state.Progress = 100
	state.CompletedAt = time.Now().Unix()
	p.QuestStates[questID] = state
	p.Attributes.Experience += quest.Rewards.Experience
	p.Attributes.Potential += quest.Rewards.Potential
	p.Money += quest.Rewards.Money
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	for itemID, amount := range quest.Rewards.Items {
		if amount > 0 {
			p.Backpack[itemID] += amount
		}
	}
	for flag, value := range quest.Rewards.Flags {
		p.Flags[flag] = value
	}
	if p.Reputations == nil {
		p.Reputations = map[string]int{}
	}
	for factionID, value := range quest.Rewards.Reputations {
		p.Reputations[factionID] += value
	}
	logs := []string{"你完成了任务：" + quest.Name}
	if quest.Rewards.Experience > 0 || quest.Rewards.Potential > 0 || quest.Rewards.Money > 0 {
		logs = append(logs, "你获得了任务奖励。")
	}
	return logs, nil
}

func (r *RuntimeManager) GetFactions(playerID string) ([]map[string]interface{}, error) {
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
	out := make([]map[string]interface{}, 0, len(fileFactions))
	for id, faction := range fileFactions {
		if !faction.Visible {
			continue
		}
		out = append(out, map[string]interface{}{
			"id":          id,
			"name":        faction.Name,
			"description": faction.Description,
			"category":    faction.Category,
			"order":       faction.Order,
			"reputation":  p.Reputations[id],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		oi, _ := out[i]["order"].(int)
		oj, _ := out[j]["order"].(int)
		if oi != oj {
			return oi < oj
		}
		return out[i]["id"].(string) < out[j]["id"].(string)
	})
	return out, nil
}

func (r *RuntimeManager) questObjectivesMetNoLock(p *model.Player, quest model.QuestDef) bool {
	for _, obj := range quest.Objectives {
		switch obj.Type {
		case "flag":
			if !p.Flags[obj.TargetID] {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func questPayload(quest model.QuestDef, state model.QuestState) map[string]interface{} {
	objectives := make([]map[string]interface{}, 0, len(quest.Objectives))
	for _, obj := range quest.Objectives {
		objectives = append(objectives, map[string]interface{}{
			"type":        obj.Type,
			"targetId":    obj.TargetID,
			"description": obj.Description,
			"required":    obj.Required,
		})
	}
	return map[string]interface{}{
		"id":          quest.ID,
		"name":        quest.Name,
		"description": quest.Description,
		"status":      state.Status,
		"progress":    state.Progress,
		"objectives":  objectives,
		"rewards":     quest.Rewards,
	}
}
