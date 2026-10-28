package manager

import (
	"math/rand"

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
	// 持续效果先于普攻：到期清理→流血/回气→镜湖待回疗；本tick新建的持续效果下tick执行
	r.processCombatEffectsTickNoLock(players, &result)
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

		if room, ok := fileRooms[attacker.RoomID]; ok && room.SafeZone {
			r.clearCombatNoLock(attacker)
			r.clearCombatNoLock(target)
			updatedRoom[attacker.RoomID] = true
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: attacker.RoomID, Message: "安全区保护生效，双方停止交战。"})
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

		now := combatNowUnix(r)
		actorA := playerEntity(attacker, combatType)
		actorB := playerEntity(target, combatType)
		first, second := actorA, actorB
		if actorB.attrs.Initiative > actorA.attrs.Initiative {
			first, second = actorB, actorA
		}
		r.runAutoRoundNoLock(first, second, roomID, now, &result)
		updatedRoom[roomID] = true

		if combatType == "KILL" {
			if attacker.Attributes.Resources.HP == 0 || target.Attributes.Resources.HP == 0 {
				r.resolveKillEndNoLock(attacker, target, roomID, &result)
			}
			continue
		}

		attackerLimit := actorA.maxHP / 5
		targetLimit := actorB.maxHP / 5
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

	now := combatNowUnix(r)
	actorA := playerEntity(attacker, combatType)
	actorB := npcEntity(ActorKey{Kind: "static_npc", EntityID: target.ID}, target, combatType)
	first, second := actorA, actorB
	if actorB.attrs.Initiative > actorA.attrs.Initiative {
		first, second = actorB, actorA
	}
	r.runAutoRoundNoLock(first, second, roomID, now, result)
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

	attackerLimit := actorA.maxHP / 5
	targetLimit := actorB.maxHP / 5
	attackerLow := attacker.Attributes.Resources.HP <= attackerLimit
	targetLow := target.Attributes.Resources.HP <= targetLimit
	if attackerLow || targetLow {
		r.clearCombatNoLock(attacker)
		target.Attributes.Resources.HP = target.Attributes.Resources.MaxHP
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

	now := combatNowUnix(r)
	yieldAt := 0
	if dungeonTarget.CombatMode == "defeat" && combatType == "KILL" {
		yieldAt = max(1, dungeonTarget.Attributes.Resources.MaxHP/4)
	}
	combatHP := max(0, dungeonTarget.Attributes.Resources.HP-yieldAt)
	actorA := playerEntity(attacker, combatType)
	actorB := npcEntity(ActorKey{Kind: "dungeon_npc", EntityID: dungeonTarget.ID}, dungeonTarget, "KILL")
	actorB.hp = &combatHP
	actorB.defeatMode = false // combatHP已扣除认负线：floor=0，触0即认负
	first, second := actorA, actorB
	if actorB.attrs.Initiative > actorA.attrs.Initiative {
		first, second = actorB, actorA
	}
	r.runAutoRoundNoLock(first, second, roomID, now, result)
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
	// Aborting a spar (including smoke escape) must not leave the shared teacher
	// permanently injured and unable to accept the next full-health challenge.
	if p.CombatType == "SPAR" {
		r.staticNPCsMu.Lock()
		if npc := r.staticNPCs[p.TargetID]; npc != nil && npc.Alive {
			npc.Attributes.Resources.HP = npc.Attributes.Resources.MaxHP
		}
		r.staticNPCsMu.Unlock()
	}
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

// runAutoRoundNoLock 一轮普攻：先手者先结算，其减益即时影响本轮后手；
// 认输线/死亡取消之后的普攻（实现规范§6，废弃双方预计算）。
func (r *RuntimeManager) runAutoRoundNoLock(first, second *combatEntity, roomID string, now int64, result *model.CombatTickResult) {
	for _, e := range []*combatEntity{first, second} {
		if b := r.currentBindingNoLock(e.key); b != nil {
			b.AutoRounds++
		}
	}
	out := r.resolveCombatActionNoLock(combatAction{Kind: actionAuto, Actor: first, Target: second, RoomID: roomID, Now: now})
	result.Messages = append(result.Messages, out.Messages...)
	r.bumpRevisionForEntityNoLock(first)
	r.bumpRevisionForEntityNoLock(second)
	if out.Stopped || !second.canAct() || !first.canAct() {
		return
	}
	out2 := r.resolveCombatActionNoLock(combatAction{Kind: actionAuto, Actor: second, Target: first, RoomID: roomID, Now: now})
	result.Messages = append(result.Messages, out2.Messages...)
	r.bumpRevisionForEntityNoLock(first)
	r.bumpRevisionForEntityNoLock(second)
}

// bumpRevisionForEntityNoLock 玩家气血/状态变化后递增其招式视图修订号。
func (r *RuntimeManager) bumpRevisionForEntityNoLock(e *combatEntity) {
	if e != nil && e.isPlayer {
		r.bumpCombatRevisionNoLock(e.key.EntityID)
	}
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
