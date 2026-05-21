package manager

import (
	"math/rand"
	"strconv"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/model"
)

func (r *RuntimeManager) ProcessCombatTick() (model.CombatTickResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return model.CombatTickResult{}, err
	}

	result := model.CombatTickResult{}
	updatedRoom := map[string]bool{}
	processed := map[string]bool{}
	r.processDungeonAggroNoLock(players, &result, updatedRoom)

	for _, attacker := range players {
		if attacker.State != "COMBAT" || attacker.TargetID == "" || processed[attacker.ID] {
			continue
		}
		if dungeonTarget, _, ok := r.dungeonNPCInRoomNoLock(attacker.RoomID, attacker.TargetID); ok {
			processed[attacker.ID] = true
			r.resolveDungeonCombatRoundNoLock(players, attacker, dungeonTarget, &result, updatedRoom)
			continue
		}
		if _, ok := r.dungeonNPCByIDNoLock(attacker.TargetID); ok {
			roomID := attacker.RoomID
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: attacker.CharacterName + "茫然地四处张望，找不到对手了。"})
			r.clearCombatNoLock(attacker)
			updatedRoom[roomID] = true
			continue
		}
		target, ok := players[attacker.TargetID]
		if !ok || target.State != "COMBAT" || target.TargetID != attacker.ID || target.RoomID != attacker.RoomID {
			roomID := attacker.RoomID
			lostTargetID := attacker.TargetID
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: attacker.CharacterName + "茫然地四处张望，找不到对手了。"})
			r.resolveCombatTargetLossNoLock(players, attacker, lostTargetID, roomID, &result)
			if ok {
				r.resolveCombatTargetLossNoLock(players, target, attacker.ID, target.RoomID, &result)
			}
			r.resolveOtherCombatantsTargetingNoLock(players, roomID, lostTargetID, attacker.ID, &result)
			updatedRoom[roomID] = true
			if ok && target.RoomID != roomID {
				updatedRoom[target.RoomID] = true
			}
			continue
		}

		processed[attacker.ID] = true
		processed[target.ID] = true
		roomID := attacker.RoomID
		combatType := attacker.CombatType
		if combatType == "" {
			combatType = target.CombatType
		}
		if combatType == "" {
			combatType = "SPAR"
		}

		strikeAToB := r.resolveCombatStrike(attacker, target, combatType)
		strikeBToA := r.resolveCombatStrike(target, attacker, combatType)

		target.Attributes.CurrentHP -= strikeAToB.Damage
		if target.Attributes.CurrentHP < 0 {
			target.Attributes.CurrentHP = 0
		}
		attacker.Attributes.CurrentHP -= strikeBToA.Damage
		if attacker.Attributes.CurrentHP < 0 {
			attacker.Attributes.CurrentHP = 0
		}

		if strikeAToB.Message != "" {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: strikeAToB.Message})
		}
		if strikeBToA.Message != "" {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: strikeBToA.Message})
		}
		updatedRoom[roomID] = true

		if combatType == "KILL" {
			if attacker.Attributes.CurrentHP == 0 || target.Attributes.CurrentHP == 0 {
				r.resolveKillEndNoLock(attacker, target, roomID, &result)
			}
			continue
		}

		attackerLimit := attacker.Attributes.MaxHP / 5
		targetLimit := target.Attributes.MaxHP / 5
		attackerLow := attacker.Attributes.CurrentHP <= attackerLimit
		targetLow := target.Attributes.CurrentHP <= targetLimit
		if attackerLow || targetLow {
			r.clearCombatNoLock(attacker)
			r.clearCombatNoLock(target)
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: r.sparEndMessage(attacker, target, attackerLow, targetLow)})
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

func (r *RuntimeManager) resolveDungeonCombatRoundNoLock(players map[string]*model.Player, attacker *model.Player, dungeonTarget *engruntime.NPCState, result *model.CombatTickResult, updatedRoom map[string]bool) {
	if attacker == nil || dungeonTarget == nil {
		return
	}
	roomID := attacker.RoomID
	combatType := attacker.CombatType
	if combatType == "" {
		combatType = "KILL"
	}

	strikeAToB := r.resolveCombatStrikeEntity(attacker.CharacterName, attacker.Attributes, dungeonTarget.Name, dungeonTarget.Attributes, combatType)
	strikeBToA := r.resolveCombatStrikeEntity(dungeonTarget.Name, dungeonTarget.Attributes, attacker.CharacterName, attacker.Attributes, "KILL")

	dungeonTarget.Attributes.CurrentHP -= strikeAToB.Damage
	if dungeonTarget.Attributes.CurrentHP < 0 {
		dungeonTarget.Attributes.CurrentHP = 0
	}
	attacker.Attributes.CurrentHP -= strikeBToA.Damage
	if attacker.Attributes.CurrentHP < 0 {
		attacker.Attributes.CurrentHP = 0
	}

	if strikeAToB.Message != "" {
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: strikeAToB.Message})
	}
	if strikeBToA.Message != "" {
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: strikeBToA.Message})
	}
	updatedRoom[roomID] = true

	if dungeonTarget.Attributes.CurrentHP == 0 {
		defeatedName := dungeonTarget.Name
		progress := 0
		if inst, defeated, progressGain, ok := r.defeatDungeonNPCNoLock(dungeonTarget.ID); ok {
			if defeated != nil && defeated.Name != "" {
				defeatedName = defeated.Name
			}
			progress = inst.Progress
			if progressGain > 0 {
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: attacker.ID, Message: "你击败了" + defeatedName + "。当前完成度" + intToString(progress) + "%"})
			}
		}
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: defeatedName + "大叫一声，倒在地上，死了！"})
		r.clearCombatNoLock(attacker)
	}

	if attacker.Attributes.CurrentHP == 0 {
		attacker.State = "DEAD"
		attacker.TargetID = ""
		attacker.CombatType = ""
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: attacker.CharacterName + "大叫一声，倒在地上，死了！"})
		oldRoomID := roomID
		settlement := r.forceLeaveDungeonOnDeathNoLock(players, attacker.ID)
		updatedRoom[oldRoomID] = true
		if attacker.RoomID != "" {
			updatedRoom[attacker.RoomID] = true
		}
		if settlement != nil {
			result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: attacker.ID, Message: formatDungeonSettlementMessage(*settlement)})
		}
	}
}

func (r *RuntimeManager) resolveKillEndNoLock(a, b *model.Player, roomID string, result *model.CombatTickResult) {
	if a.Attributes.CurrentHP == 0 && b.Attributes.CurrentHP == 0 {
		a.State = "DEAD"
		a.TargetID = ""
		a.CombatType = ""
		b.State = "DEAD"
		b.TargetID = ""
		b.CombatType = ""
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: a.CharacterName + "和" + b.CharacterName + "同归于尽。"})
		return
	}
	if b.Attributes.CurrentHP == 0 {
		b.State = "DEAD"
		b.TargetID = ""
		b.CombatType = ""
		r.clearCombatNoLock(a)
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: b.CharacterName + "大叫一声，倒在地上，死了！"})
		return
	}
	if a.Attributes.CurrentHP == 0 {
		a.State = "DEAD"
		a.TargetID = ""
		a.CombatType = ""
		r.clearCombatNoLock(b)
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: a.CharacterName + "大叫一声，倒在地上，死了！"})
	}
}

func (r *RuntimeManager) clearCombatNoLock(p *model.Player) {
	if p.State != "DEAD" {
		p.State = "IDLE"
	}
	p.TargetID = ""
	p.CombatType = ""
}

func (r *RuntimeManager) resolveCombatTargetLossNoLock(players map[string]*model.Player, combatant *model.Player, lostTargetID, roomID string, result *model.CombatTickResult) {
	if combatant == nil || combatant.State != "COMBAT" || combatant.TargetID != lostTargetID {
		return
	}
	replacement := r.findReplacementTargetNoLock(players, combatant, lostTargetID)
	if replacement == nil {
		r.clearCombatNoLock(combatant)
		return
	}
	combatant.TargetID = replacement.ID
	result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: combatant.CharacterName + "转身盯上了" + replacement.CharacterName + "。"})
}

func (r *RuntimeManager) resolveOtherCombatantsTargetingNoLock(players map[string]*model.Player, roomID, lostTargetID, excludeCombatantID string, result *model.CombatTickResult) {
	if roomID == "" || lostTargetID == "" {
		return
	}
	for _, p := range players {
		if p.State != "COMBAT" {
			continue
		}
		if excludeCombatantID != "" && p.ID == excludeCombatantID {
			continue
		}
		if p.RoomID != roomID || p.TargetID != lostTargetID {
			continue
		}
		r.resolveCombatTargetLossNoLock(players, p, lostTargetID, roomID, result)
	}
}

func (r *RuntimeManager) findReplacementTargetNoLock(players map[string]*model.Player, combatant *model.Player, lostTargetID string) *model.Player {
	if combatant == nil || combatant.RoomID == "" {
		return nil
	}
	for _, candidate := range players {
		if candidate == nil || candidate.ID == combatant.ID || candidate.ID == lostTargetID {
			continue
		}
		if candidate.State != "COMBAT" || candidate.RoomID != combatant.RoomID || candidate.TargetID != combatant.ID {
			continue
		}
		if combatant.CombatType != "" && candidate.CombatType != "" && combatant.CombatType != candidate.CombatType {
			continue
		}
		return candidate
	}
	return nil
}

func (r *RuntimeManager) sparEndMessage(attacker, target *model.Player, attackerLow, targetLow bool) string {
	if attackerLow && targetLow {
		return "双方点到为止，抱拳收招。"
	}
	winner := attacker
	loser := target
	if attackerLow && !targetLow {
		winner = target
		loser = attacker
	}
	if rand.Intn(2) == 0 {
		return loser.CharacterName + "后退三尺，抱拳道：佩服、佩服！"
	}
	return winner.CharacterName + "抱拳道：承让。"
}

type combatStrikeResult struct {
	Damage  int
	Message string
}

func (r *RuntimeManager) resolveCombatStrike(attacker, defender *model.Player, combatType string) combatStrikeResult {
	return r.resolveCombatStrikeEntity(attacker.CharacterName, attacker.Attributes, defender.CharacterName, defender.Attributes, combatType)
}

func (r *RuntimeManager) resolveCombatStrikeEntity(attackerName string, attackerAttrs model.Attributes, defenderName string, defenderAttrs model.Attributes, combatType string) combatStrikeResult {
	hitChance := boundedChance(attackerAttrs.HitRate, defenderAttrs.Dodge)
	if rand.Float64() > hitChance {
		return combatStrikeResult{Damage: 0, Message: attackerName + "发起攻击，但被" + defenderName + "躲开了！"}
	}

	base := attackerAttrs.Attack - defenderAttrs.Defense/3
	if combatType == "KILL" {
		base += 14
	} else {
		base += 8
	}
	if base < 2 {
		base = 2
	}

	fluctuation := 90 + rand.Intn(21)
	damage := base * fluctuation / 100

	crit := rand.Intn(100) < attackerAttrs.CritRate
	if crit {
		damage *= 2
	}

	parryChance := boundedChance(defenderAttrs.Parry, attackerAttrs.HitRate)
	parried := rand.Float64() < parryChance
	if parried {
		damage /= 2
	}

	msg := attackerName + "一招击中" + defenderName + "，造成" + strconv.Itoa(damage) + "点伤害"
	if crit {
		msg = attackerName + "发动致命一击，重创" + defenderName + "，造成" + strconv.Itoa(damage) + "点伤害"
	}
	if parried {
		msg += "（被招架减伤）"
	}
	if damage <= 0 {
		msg = attackerName + "一招击中" + defenderName + "，但没有造成任何伤害。"
		return combatStrikeResult{Damage: 0, Message: msg}
	}
	msg += "。"
	return combatStrikeResult{Damage: damage, Message: msg}
}

func boundedChance(attackerStat, defenderStat int) float64 {
	den := attackerStat + defenderStat
	if den <= 0 {
		return 0.7
	}
	raw := float64(attackerStat) / float64(den)
	if raw < 0.45 {
		return 0.45
	}
	if raw > 0.95 {
		return 0.95
	}
	return raw
}
