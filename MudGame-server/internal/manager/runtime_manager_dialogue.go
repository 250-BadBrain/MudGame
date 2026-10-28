package manager

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

func (r *RuntimeManager) Talk(playerID, npcID string) (map[string]interface{}, error) {
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
		return nil, err
	}
	if npc.DialogueID == "" {
		return nil, errors.New("dialogue_not_found")
	}
	dialogue, ok := fileDialogues[npc.DialogueID]
	if !ok {
		return nil, errors.New("dialogue_not_found")
	}
	result := map[string]interface{}{"dialogue": r.dialoguePayloadNoLock(p, npc, dialogue)}
	if dnpc, exists := r.dungeonNPCByIDNoLock(npc.ID); exists && r.dungeonMgr != nil {
		staticRooms, loadErr := r.loadRooms()
		if loadErr != nil {
			return nil, loadErr
		}
		room, roomOK := r.roomByIDNoLock(p.RoomID, staticRooms)
		if !roomOK {
			return nil, exception.ErrRoomNotFound
		}
		inst, inDungeon := r.dungeonMgr.PlayerInstanceNoLock(playerID)
		bound := r.dungeonMgr.RoomDialogueActionsNoLock(playerID, room.ID, dnpc.TemplateID)
		if inDungeon && len(bound) > 0 {
			message, actionResult, stepErr := r.advanceDungeonDialogueNoLock(p, room, npc, inst)
			if stepErr != nil {
				return nil, stepErr
			}
			for key, value := range actionResult {
				result[key] = value
			}
			result["message"] = message
			if err := r.savePlayers(players); err != nil {
				return nil, err
			}
			return result, nil
		}
		if inDungeon {
			if inst.DialogueHistory == nil {
				inst.DialogueHistory = map[string]bool{}
			}
			message, logs, stepErr := r.advanceStaticDialogueNoLock(p, npc, dialogue, inst.DialogueHistory)
			if stepErr != nil {
				return nil, stepErr
			}
			result["message"] = message
			result["logs"] = logs
			result["dialogue"] = r.dialoguePayloadNoLock(p, npc, dialogue)
			if err := r.savePlayers(players); err != nil {
				return nil, err
			}
			return result, nil
		}
	}
	if p.DialogueHistory == nil {
		p.DialogueHistory = map[string]bool{}
	}
	message, logs, err := r.advanceStaticDialogueNoLock(p, npc, dialogue, p.DialogueHistory)
	if err != nil {
		return nil, err
	}
	result["message"] = message
	result["logs"] = logs
	result["dialogue"] = r.dialoguePayloadNoLock(p, npc, dialogue)
	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RuntimeManager) advanceDungeonDialogueNoLock(p *model.Player, room model.Room, npc model.NPC, inst *engruntime.Instance) (string, map[string]interface{}, error) {
	if inst.DialogueProgress == nil {
		inst.DialogueProgress = map[string]int{}
	}
	if inst.DialogueDoneMessages == nil {
		inst.DialogueDoneMessages = map[string]string{}
	}
	if len(inst.PendingChoices) > 0 {
		return r.dungeonMgr.PendingChoicePromptNoLock(inst), map[string]interface{}{
			"choices": dungeonChoicePayload(r.dungeonMgr.PendingActionChoicesNoLock(inst)),
		}, nil
	}
	for _, action := range r.dungeonMgr.DialogueActionsNoLock(p.ID, room.ID, dungeonNPCTemplateID(inst, npc.ID)) {
		if !r.puzzleActionRequirementsMetNoLock(p, room, inst, action) {
			message := action.FailureMessage
			if message == "" {
				message = "眼下时机未到，再问也得不到更多消息。"
			}
			return message, nil, nil
		}
		lines := action.DialogueLines
		if len(lines) == 0 {
			lines = []string{npcSpeech(npc.Name, action.Message, 0)}
		}
		step := inst.DialogueProgress[action.ID]
		if step >= len(lines) {
			step = len(lines) - 1
		}
		message := lines[step]
		inst.DialogueProgress[action.ID] = step + 1
		if step+1 < len(lines) {
			return message, nil, nil
		}
		actionResult, handled := r.executePuzzleActionNoLock(p, room, action.ID)
		if !handled {
			return "", nil, errors.New("dialogue_action_not_available")
		}
		if len(action.Choices) > 0 {
			inst.PendingChoicePrompt = message
		}
		if action.DialogueDoneMessage != "" {
			inst.DialogueDoneMessages[npc.ID] = action.DialogueDoneMessage
		}
		return message, actionResult, nil
	}
	if message := inst.DialogueDoneMessages[npc.ID]; message != "" {
		return message, nil, nil
	}
	for _, action := range r.dungeonMgr.RoomDialogueActionsNoLock(p.ID, room.ID, dungeonNPCTemplateID(inst, npc.ID)) {
		if action.FailureMessage != "" {
			return action.FailureMessage, nil, nil
		}
	}
	return fmt.Sprintf("%s已无更多话要说。", npc.Name), nil, nil
}

func dungeonNPCTemplateID(inst *engruntime.Instance, npcID string) string {
	if inst != nil {
		if npc := inst.NPCs[npcID]; npc != nil {
			return npc.TemplateID
		}
	}
	return ""
}

func (r *RuntimeManager) advanceStaticDialogueNoLock(p *model.Player, npc model.NPC, dialogue model.DialogueDef, history map[string]bool) (string, []string, error) {
	if history == nil {
		return "", nil, errors.New("dialogue_history_not_initialized")
	}
	openingKey := npc.ID + "|opening"
	if !history[openingKey] {
		history[openingKey] = true
		return npcSpeech(npc.Name, dialogue.Opening, 0), nil, nil
	}
	for i := range dialogue.Options {
		option := dialogue.Options[i]
		key := npc.ID + "|" + option.ID
		if option.Explicit || option.Repeatable || !r.dialogueOptionAvailableNoLock(p, npc.ID, option, history) {
			continue
		}
		logs, err := r.applyDialogueOptionEffectsNoLock(p, &option, npc.ID)
		if err != nil {
			return "", nil, err
		}
		history[key] = true
		return npcSpeech(npc.Name, option.Response, i+1), logs, nil
	}
	endings := []string{
		"该说的都已说了，余下的路要你自己走。",
		"言尽于此，莫要误了眼前的事。",
		"你已经知道该怎么做了，快去吧。",
		"再问也是旧话，先把眼前之事办妥。",
	}
	return npcSpeech(npc.Name, endings[dialogueVariant(npc.ID, len(endings))], len(dialogue.Options)+1), nil, nil
}

func npcSpeech(name, text string, step int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return name + "没有开口。"
	}
	if strings.Contains(text, name) || strings.Contains(text, "说道") || strings.Contains(text, "道：") {
		return text
	}
	manners := []string{"压低声音", "略一沉吟", "缓缓", "正色", "淡淡", "侧目看了看四周，低声"}
	return fmt.Sprintf("%s%s说道：%s", name, manners[step%len(manners)], text)
}

func dialogueVariant(value string, count int) int {
	if count <= 1 {
		return 0
	}
	total := 0
	for _, r := range value {
		total += int(r)
	}
	return total % count
}

func (r *RuntimeManager) applyDialogueOptionEffectsNoLock(p *model.Player, selected *model.DialogueOption, npcID string) ([]string, error) {
	if err := r.validateDialogueEffectsNoLock(p, selected, npcID); err != nil {
		return nil, err
	}
	logs := []string{}
	// Finish the only fallible effect before costs, flags or dungeon collection mutate.
	if selected.CompleteQuestID != "" {
		questLogs, err := r.completeQuestNoLock(p, selected.CompleteQuestID, npcID)
		if err != nil {
			return nil, err
		}
		logs = append(logs, questLogs...)
	}
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
	if selected.TakeMoney > 0 {
		p.Money -= selected.TakeMoney
	}
	if selected.GiveMoney > 0 {
		p.Money += selected.GiveMoney
	}
	for flag, value := range selected.SetFlags {
		p.Flags[flag] = value
	}
	if selected.StartQuestID != "" {
		if log := r.startQuestNoLock(p, selected.StartQuestID); log != "" {
			logs = append(logs, log)
		}
	}
	return logs, nil
}

func (r *RuntimeManager) ChooseDialogue(playerID, npcID, optionID string) (map[string]interface{}, string, []string, map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, "", nil, nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, "", nil, nil, exception.ErrPlayerNotFound
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
		return nil, "", nil, nil, err
	}
	dialogue, ok := fileDialogues[npc.DialogueID]
	if !ok {
		return nil, "", nil, nil, errors.New("dialogue_not_found")
	}
	if strings.HasPrefix(optionID, "dungeon_action:") {
		actionID := strings.TrimPrefix(optionID, "dungeon_action:")
		staticRooms, loadErr := r.loadRooms()
		if loadErr != nil {
			return nil, "", nil, nil, loadErr
		}
		room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
		if !ok {
			return nil, "", nil, nil, exception.ErrRoomNotFound
		}
		inst, action, exists := r.dungeonMgr.PuzzleActionNoLock(p.ID, room.ID, actionID)
		dnpc, npcExists := r.dungeonNPCByIDNoLock(npc.ID)
		if !exists || !npcExists || action.DialogueNPCID != dnpc.TemplateID || !r.puzzleActionRequirementsMetNoLock(p, room, inst, action) {
			return nil, "", nil, nil, errors.New("dialogue_option_not_found")
		}
		actionResult, handled := r.executePuzzleActionNoLock(p, room, actionID)
		if !handled {
			return nil, "", nil, nil, errors.New("dialogue_option_not_found")
		}
		if err := r.savePlayers(players); err != nil {
			return nil, "", nil, nil, err
		}
		payload := r.dialoguePayloadNoLock(p, npc, dialogue)
		message, _ := actionResult["message"].(string)
		return payload, message, nil, actionResult, nil
	}
	var selected *model.DialogueOption
	for i := range dialogue.Options {
		opt := dialogue.Options[i]
		if opt.ID == optionID && r.dialogueOptionAvailableNoLock(p, npc.ID, opt, r.dialogueHistoryNoLock(p, npc.ID)) {
			selected = &opt
			break
		}
	}
	if selected == nil {
		return nil, "", nil, nil, errors.New("dialogue_option_not_found")
	}

	logs, err := r.applyDialogueOptionEffectsNoLock(p, selected, npc.ID)
	if err != nil {
		return nil, "", nil, nil, err
	}
	if !selected.Repeatable {
		r.dialogueHistoryNoLock(p, npc.ID)[npc.ID+"|"+selected.ID] = true
	}

	if err := r.savePlayers(players); err != nil {
		return nil, "", nil, nil, err
	}
	payload := r.dialoguePayloadNoLock(p, npc, dialogue)
	return payload, selected.Response, logs, nil, nil
}

func (r *RuntimeManager) npcInPlayerRoomNoLock(p *model.Player, npcID string) (model.NPC, error) {
	if p.State == "DEAD" {
		return model.NPC{}, errors.New("dead_state")
	}
	if p.State != "IDLE" {
		return model.NPC{}, errors.New("busy_state")
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
		available := r.dialogueOptionAvailableNoLock(p, npc.ID, opt, r.dialogueHistoryNoLock(p, npc.ID))
		if !available && !opt.Repeatable {
			continue
		}
		options = append(options, map[string]interface{}{
			"available": available, "repeatable": opt.Repeatable,
			"id":   opt.ID,
			"text": opt.Text,
		})
	}
	if r.dungeonMgr != nil {
		if dnpc, ok := r.dungeonNPCByIDNoLock(npc.ID); ok {
			staticRooms, _ := r.loadRooms()
			room, roomOK := r.roomByIDNoLock(p.RoomID, staticRooms)
			if roomOK {
				for _, action := range r.dungeonMgr.DialogueActionsNoLock(p.ID, room.ID, dnpc.TemplateID) {
					inst, _, _ := r.dungeonMgr.PuzzleActionNoLock(p.ID, room.ID, action.ID)
					if r.puzzleActionRequirementsMetNoLock(p, room, inst, action) {
						options = append(options, map[string]interface{}{"id": "dungeon_action:" + action.ID, "text": action.Label})
					}
				}
			}
		}
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

func (r *RuntimeManager) puzzleActionRequirementsMetNoLock(p *model.Player, room model.Room, inst *engruntime.Instance, action engruntime.PuzzleAction) bool {
	if p == nil || inst == nil {
		return false
	}
	if !contentEvidenceMet(p, action.RequiredFlags, action.PresentItems) {
		return false
	}
	if len(inst.PendingChoices) > 0 && r.dungeonMgr.PuzzleActionStateStatusNoLock(inst, action) == engruntime.PuzzleActionRepeat {
		return true
	}
	for itemID, amount := range action.RequiredItems {
		if p.Backpack[itemID] < amount {
			return false
		}
	}
	for puzzleID, requiredState := range action.RequiredPuzzleStates {
		if inst.PuzzleStates[puzzleID] != requiredState {
			return false
		}
	}
	for _, templateID := range action.RequiredDefeatNPCs {
		found := false
		for _, npc := range inst.NPCs {
			if npc != nil && npc.RoomID == room.ID && npc.TemplateID == templateID {
				found = true
				if npc.Alive && !npc.Defeated {
					return false
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (r *RuntimeManager) dialogueRequirementMetNoLock(p *model.Player, req model.DialogueRequire) bool {
	if !contentEvidenceMet(p, req.Flags, req.PresentItems) {
		return false
	}
	for id, level := range req.Skills {
		if p.Skills[id] < level {
			return false
		}
	}
	if req.QuestID != "" {
		state := questStateNoLock(p, req.QuestID)
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
