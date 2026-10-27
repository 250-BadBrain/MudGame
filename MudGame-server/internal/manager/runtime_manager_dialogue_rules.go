package manager

import (
	"errors"
	"mudgame/server/internal/model"
)

func (r *RuntimeManager) dialogueHistoryNoLock(p *model.Player, npcID string) map[string]bool {
	if npc, ok := r.dungeonNPCByIDNoLock(npcID); ok {
		if inst, exists := r.dungeonMgr.PlayerInstanceNoLock(p.ID); exists && inst.NPCs[npc.ID] != nil {
			if inst.DialogueHistory == nil {
				inst.DialogueHistory = map[string]bool{}
			}
			return inst.DialogueHistory
		}
	}
	if p.DialogueHistory == nil {
		p.DialogueHistory = map[string]bool{}
	}
	return p.DialogueHistory
}

func (r *RuntimeManager) validateDialogueEffectsNoLock(p *model.Player, opt *model.DialogueOption, npcID string) error {
	if opt.TakeMoney < 0 || opt.TakeMoney > p.Money {
		return errors.New("money_not_enough")
	}
	for item, n := range opt.TakeItems {
		if n < 0 || p.Backpack[item] < n {
			return errors.New("item_not_in_inventory")
		}
	}
	if !r.dialogueRequirementMetNoLock(p, opt.Requires) {
		return errors.New("dialogue_option_not_found")
	}
	if opt.StartQuestID != "" {
		id := canonicalQuestID(opt.StartQuestID)
		q, ok := fileQuests[id]
		state := questStateNoLock(p, id)
		if !ok || q.GiverNPCID != npcID || state.Status == "active" || (state.Status == "completed" && !q.Repeatable) || r.questLockHintNoLock(p, q) != "" {
			return errors.New("quest_not_available")
		}
	}
	if opt.CompleteQuestID != "" {
		return r.questCompletionErrorNoLock(p, opt.CompleteQuestID, npcID)
	}
	return nil
}

func (r *RuntimeManager) dialogueOptionAvailableNoLock(p *model.Player, npcID string, opt model.DialogueOption, history map[string]bool) bool {
	if !opt.Repeatable && history[npcID+"|"+opt.ID] {
		return false
	}
	return r.validateDialogueEffectsNoLock(p, &opt, npcID) == nil
}
