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
		r.staticNPCsMu.Lock()
		staticTarget, staticTargetOk := r.staticNPCs[attacker.TargetID]
		r.staticNPCsMu.Unlock()
		if staticTargetOk {
			processed[attacker.ID] = true
			if staticTarget == nil || !staticTarget.Alive || staticTarget.RoomID != attacker.RoomID {
				roomID := attacker.RoomID
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: attacker.CharacterName + "茫然地四处张望，找不到对手了。"})
				r.clearCombatNoLock(attacker)
				updatedRoom[roomID] = true
				continue
			}
			r.resolveStaticNPCCombatRoundNoLock(attacker, staticTarget, &result, updatedRoom)
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

		attackerAttrs := effectiveAttributesForPlayerNoLock(attacker)
		targetAttrs := effectiveAttributesForPlayerNoLock(target)
		strikeAToB := r.resolveCombatStrikeEntity(attacker.CharacterName, attackerAttrs, target.CharacterName, targetAttrs, combatType)
		strikeBToA := r.resolveCombatStrikeEntity(target.CharacterName, targetAttrs, attacker.CharacterName, attackerAttrs, combatType)
		attackerFirst := attackerAttrs.Initiative >= targetAttrs.Initiative

		applyPairedStrikesNoLock(&attacker.Attributes.Resources.HP, &target.Attributes.Resources.HP, strikeAToB, strikeBToA, attackerFirst, combatType == "KILL", roomID, &result.Messages)
		updatedRoom[roomID] = true

		if combatType == "KILL" {
			if attacker.Attributes.Resources.HP == 0 || target.Attributes.Resources.HP == 0 {
				r.resolveKillEndNoLock(attacker, target, roomID, &result)
			}
			continue
		}

		attackerLimit := attackerAttrs.MaxHP / 5
		targetLimit := targetAttrs.MaxHP / 5
		attackerLow := attacker.Attributes.Resources.HP <= attackerLimit
		targetLow := target.Attributes.Resources.HP <= targetLimit
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

func (r *RuntimeManager) resolveStaticNPCCombatRoundNoLock(attacker *model.Player, target *engruntime.NPCState, result *model.CombatTickResult, updatedRoom map[string]bool) {
	if attacker == nil || target == nil {
		return
	}
	roomID := attacker.RoomID
	combatType := attacker.CombatType
	if combatType == "" {
		combatType = "SPAR"
	}

	strikeAToB := r.resolveCombatStrikeEntity(attacker.CharacterName, effectiveAttributesForPlayerNoLock(attacker), target.Name, model.BuildEffectiveAttributes(target.Attributes), combatType)
	counterType := combatType
	if combatType == "KILL" || target.Aggressive {
		counterType = "KILL"
	}
	playerAttrs := effectiveAttributesForPlayerNoLock(attacker)
	npcAttrs := model.BuildEffectiveAttributes(target.Attributes)
	strikeBToA := r.resolveCombatStrikeEntity(target.Name, npcAttrs, attacker.CharacterName, playerAttrs, counterType)

	applyPairedStrikesNoLock(&attacker.Attributes.Resources.HP, &target.Attributes.Resources.HP, strikeAToB, strikeBToA, playerAttrs.Initiative >= npcAttrs.Initiative, combatType == "KILL", roomID, &result.Messages)
	updatedRoom[roomID] = true

	if combatType == "KILL" {
		if target.Attributes.Resources.HP == 0 {
			target.Alive = false
			r.clearCombatNoLock(attacker)
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: target.Name + "大叫一声，倒在地上，死了！"})
		}
		if attacker.Attributes.Resources.HP == 0 {
			attacker.State = "DEAD"
			attacker.TargetID = ""
			attacker.CombatType = ""
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: attacker.CharacterName + "大叫一声，倒在地上，死了！"})
		}
		return
	}

	attackerLimit := playerAttrs.MaxHP / 5
	targetLimit := target.Attributes.Resources.MaxHP / 5
	attackerLow := attacker.Attributes.Resources.HP <= attackerLimit
	targetLow := target.Attributes.Resources.HP <= targetLimit
	if attackerLow || targetLow {
		r.clearCombatNoLock(attacker)
		if targetLow {
			target.Attributes.Resources.HP = target.Attributes.Resources.MaxHP
		}
		if attackerLow && targetLow {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: "双方点到为止，抱拳收招。"})
		} else if attackerLow {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: target.Name + "抱拳道：承让。"})
		} else {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: target.Name + "后退三尺，抱拳道：佩服、佩服！"})
		}
	}
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

	strikeAToB := r.resolveCombatStrikeEntity(attacker.CharacterName, effectiveAttributesForPlayerNoLock(attacker), dungeonTarget.Name, model.BuildEffectiveAttributes(dungeonTarget.Attributes), combatType)
	playerAttrs := effectiveAttributesForPlayerNoLock(attacker)
	npcAttrs := model.BuildEffectiveAttributes(dungeonTarget.Attributes)
	strikeBToA := r.resolveCombatStrikeEntity(dungeonTarget.Name, npcAttrs, attacker.CharacterName, playerAttrs, "KILL")

	yieldAt := 0
	if dungeonTarget.CombatMode == "defeat" && combatType == "KILL" {
		yieldAt = max(1, dungeonTarget.Attributes.Resources.MaxHP/4)
	}
	combatHP := max(0, dungeonTarget.Attributes.Resources.HP-yieldAt)
	applyPairedStrikesNoLock(&attacker.Attributes.Resources.HP, &combatHP, strikeAToB, strikeBToA, playerAttrs.Initiative >= npcAttrs.Initiative, combatType == "KILL", roomID, &result.Messages)
	dungeonTarget.Attributes.Resources.HP = combatHP + yieldAt
	updatedRoom[roomID] = true

	if combatHP == 0 {
		defeatedName := dungeonTarget.Name
		defeatOnly := dungeonTarget.CombatMode == "defeat"
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
		if defeatOnly {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: defeatedName + "踉跄后退，已经无力再战。"})
		} else {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: defeatedName + "大叫一声，倒在地上，死了！"})
		}
		r.clearCombatNoLock(attacker)
	}

	if attacker.Attributes.Resources.HP == 0 {
		attacker.State = "DEAD"
		attacker.TargetID = ""
		attacker.CombatType = ""
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: attacker.CharacterName + "大叫一声，倒在地上，死了！"})
		updatedRoom[roomID] = true
	}
}

func (r *RuntimeManager) resolveKillEndNoLock(a, b *model.Player, roomID string, result *model.CombatTickResult) {
	if a.Attributes.Resources.HP == 0 && b.Attributes.Resources.HP == 0 {
		a.State = "DEAD"
		a.TargetID = ""
		a.CombatType = ""
		b.State = "DEAD"
		b.TargetID = ""
		b.CombatType = ""
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: a.CharacterName + "和" + b.CharacterName + "同归于尽。"})
		return
	}
	if b.Attributes.Resources.HP == 0 {
		b.State = "DEAD"
		b.TargetID = ""
		b.CombatType = ""
		r.clearCombatNoLock(a)
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: b.CharacterName + "大叫一声，倒在地上，死了！"})
		return
	}
	if a.Attributes.Resources.HP == 0 {
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

// applyPairedStrikesNoLock 按先攻值决定双方出手顺序：先攻高者先出手。
// killRound（生死战）中已倒下的一方不再出手；切磋（SPAR）双方各自打完一轮再判定点到为止。
// strikeA 为 A 方打向 B 方的一击，strikeB 为 B 方打向 A 方的一击。
func applyPairedStrikesNoLock(aHP, bHP *int, strikeA, strikeB combatStrikeResult, aFirst bool, killRound bool, roomID string, messages *[]model.RoomMessage) {
	swing := func(strikerHP, victimHP *int, strike combatStrikeResult) {
		if killRound && *strikerHP <= 0 {
			return
		}
		*victimHP -= strike.Damage
		if *victimHP < 0 {
			*victimHP = 0
		}
		if strike.Message != "" {
			*messages = append(*messages, model.RoomMessage{RoomID: roomID, Message: strike.Message})
		}
	}
	if aFirst {
		swing(aHP, bHP, strikeA)
		swing(bHP, aHP, strikeB)
		return
	}
	swing(bHP, aHP, strikeB)
	swing(aHP, bHP, strikeA)
}

func (r *RuntimeManager) resolveCombatStrikeEntity(attackerName string, attackerAttrs model.EffectiveAttributes, defenderName string, defenderAttrs model.EffectiveAttributes, combatType string) combatStrikeResult {
	floatRoll, intRoll := rand.Float64, rand.Intn
	if r.combatRandom != nil {
		floatRoll, intRoll = r.combatRandom.Float64, r.combatRandom.Intn
	}
	hitChance := boundedChance(attackerAttrs.HitRate, defenderAttrs.Dodge)
	if floatRoll() > hitChance {
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

	fluctuation := 90 + intRoll(21)
	damage := base * fluctuation / 100

	crit := floatRoll() < min(25, max(0, attackerAttrs.CritRate))/100
	if crit {
		damage *= 2
	}

	parryChance := boundedParryChance(defenderAttrs.Parry, attackerAttrs.HitRate)
	parried := floatRoll() < parryChance
	if parried {
		damage /= 2
	}

	msg := attackerName + "一招击中" + defenderName + "，造成" + strconv.Itoa(damage) + "点伤害"
	if crit {
		msg = attackerName + "发动致命一击，重创" + defenderName + "，造成" + strconv.Itoa(damage) + "点伤害"
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
	if raw < 0.15 {
		return 0.15
	}
	if raw > 0.95 {
		return 0.95
	}
	return raw
}

func boundedParryChance(parry, hit int) float64 {
	if parry <= 0 {
		return 0
	}
	return min(.60, max(0, float64(parry)/float64(parry+max(0, hit))))
}
