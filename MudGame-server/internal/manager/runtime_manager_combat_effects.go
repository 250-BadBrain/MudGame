package manager

import (
	"mudgame/server/internal/model"
)

// 主动招式的效果引擎（实现规范§2/§6）：施放时落效果、命中后触发、tick处理持续效果。
// 所有幅度在施放时按S(L)成长快照，之后不随等级变化。

const defaultEffectWindowMs = 12000 // 次数效果最长12秒

// applyMoveCastEffectsNoLock 施放成功后按定义落效果。返回自损扣血量（魔极蚀心）。
// 即时Op（cleanse）当场结算；持续/计数Op创建效果实例。
func (r *RuntimeManager) applyMoveCastEffectsNoLock(def model.CombatMoveDef, level int, actor, target ActorKey, battleID string, paidMP int) int {
	now := combatNowUnix(r)
	selfDamage := 0
	for _, e := range def.EffectiveEffects(level) {
		switch e.Op {
		case "cleanse":
			r.applyCleanseNoLock(actor, e.Tags)
		case "self_hp_cost":
			selfDamage += int(float64(actorMaxHPNoLock(r, actor)) * e.Value)
		case "next_active", "guard", "stat_mod", "vulnerability", "haste", "immunity", "mp_discount", "paid_mp_refund", "delayed_damage_heal":
			if e.Target != "self" {
				continue // 敌向效果由命中触发或debuff专用路径处理
			}
			ef := &CombatEffect{
				ID: newBattleToken(), BattleID: battleID, MoveID: def.ID,
				Owner: actor, Source: actor, Op: e.Op, Stat: e.Stat, Scope: e.Scope,
				Group: e.Group, Value: e.Value, Cap: e.Cap, Power: e.Power,
				Charges: e.Charges, Ticks: e.Ticks, DurationMs: e.DurationMs,
				CreatedAt: now, ExpiresAt: effectExpiry(now, e.DurationMs), Tags: e.Tags,
				PaidMP: paidMP,
			}
			r.addCombatEffectNoLock(ef)
		}
	}
	_ = target
	return selfDamage
}

// applyMoveHitEffectsNoLock 命中后效果：敌向削弱、流血、吸血、驱散、剥护体。
// 由resolveCombatActionNoLock在总实扣>0时调用一次（多段不重复触发）。
func (r *RuntimeManager) applyMoveHitEffectsNoLock(def model.CombatMoveDef, level int, act combatAction, out combatActionOutcome, battleID string) {
	now := combatNowUnix(r)
	for _, e := range def.EffectiveEffects(level) {
		switch e.Op {
		case "stat_mod":
			if e.Trigger != "hit" || e.Target != "opponent" {
				continue
			}
			ef := &CombatEffect{
				ID: newBattleToken(), BattleID: battleID, MoveID: def.ID,
				Owner: act.Target.key, Source: act.Actor.key, Op: e.Op, Stat: e.Stat, Scope: e.Scope,
				Group: e.Group, Value: e.Value, Charges: max(1, e.Charges),
				CreatedAt: now, ExpiresAt: effectExpiry(now, e.DurationMs), Tags: e.Tags,
			}
			r.addCombatEffectNoLock(ef)
		case "bleed":
			if e.Trigger != "hit" {
				continue
			}
			ticks := max(1, e.Ticks)
			budget := int(float64(bleedBaseNoLock(act)) * e.Power)
			if budget < 1 {
				budget = 1
			}
			ef := &CombatEffect{
				ID: newBattleToken(), BattleID: battleID, MoveID: def.ID,
				Owner: act.Target.key, Source: act.Actor.key, Op: "bleed", Scope: "dot",
				Group: "bleed", Power: e.Power, Charges: ticks, Ticks: ticks,
				TickBudget: splitBudget(budget, ticks),
				CreatedAt:  now, ExpiresAt: effectExpiry(now, 0),
			}
			r.addCombatEffectNoLock(ef)
		case "lifesteal":
			heal := 0
			gap := act.Actor.maxHP - *act.Actor.hp
			if gap > 0 && act.Actor.canAct() {
				heal = min(int(float64(out.TotalDamage)*e.Value), int(float64(act.Actor.maxHP)*e.Cap))
				heal = min(heal, gap)
				if heal > 0 {
					*act.Actor.hp += heal
					out.Messages = append(out.Messages, model.RoomMessage{RoomID: act.RoomID, Message: act.Actor.name + "以血还血，恢复" + itoa(heal) + "点气血。"})
				}
			}
		case "dispel_bonus":
			r.applyDispelBonusNoLock(act.Target.key, e.Tags)
		case "strip_guard_charge":
			r.applyStripGuardChargeNoLock(act.Target.key)
		}
	}
}

// applyDebuffCastNoLock 纯敌向减益（缚龙引）命中后落效果：按正常命中概率单独判定，
// 无伤害、无暴击/招架，不消耗next_active增伤，也不触发来袭防护/反击。
func (r *RuntimeManager) applyDebuffCastNoLock(def model.CombatMoveDef, level int, source, targetOwner ActorKey, battleID string) {
	now := combatNowUnix(r)
	for _, e := range def.EffectiveEffects(level) {
		if e.Op != "stat_mod" || e.Target != "opponent" {
			continue
		}
		ef := &CombatEffect{
			ID: newBattleToken(), BattleID: battleID, MoveID: def.ID,
			Owner: targetOwner, Source: source, Op: e.Op, Stat: e.Stat, Scope: e.Scope,
			Group: e.Group, Value: e.Value, Charges: max(1, e.Charges),
			CreatedAt: now, ExpiresAt: effectExpiry(now, e.DurationMs), Tags: e.Tags,
		}
		r.addCombatEffectNoLock(ef)
	}
}

// bleedBaseNoLock 流血预算的D快照基数：以动作结算时的基础伤害公式重算（不含倍率）。
func bleedBaseNoLock(act combatAction) int {
	aAttack := act.Actor.attrs.Attack
	tDefense := act.Target.attrs.Defense
	modeBonus := 14
	if act.Actor.mode == "SPAR" {
		modeBonus = 8
	}
	base := aAttack - tDefense/3 + modeBonus
	return max(2, base)
}

func splitBudget(total, parts int) []int {
	out := make([]int, parts)
	for i := 0; i < total; i++ {
		out[i%parts]++
	}
	return out
}

func effectExpiry(now int64, durationMs int) int64 {
	if durationMs > 0 {
		return now + int64(durationMs)
	}
	return now + defaultEffectWindowMs
}

// applyCleanseNoLock 移除自身1项负面：优先流血，再按削弱幅度（同幅按攻击/命中/躲闪/先手/易伤序）。
func (r *RuntimeManager) applyCleanseNoLock(owner ActorKey, tags []string) {
	wantBleed := containsTag(tags, "bleed")
	wantStatDown := containsTag(tags, "stat_down")
	wantVuln := containsTag(tags, "vulnerability")
	var candidates []*CombatEffect
	for _, ef := range r.combatEffects[owner.String()] {
		switch {
		case wantBleed && ef.Op == "bleed":
			candidates = append(candidates, ef)
		case wantVuln && ef.Op == "vulnerability":
			candidates = append(candidates, ef)
		case wantStatDown && ef.Op == "stat_mod" && ef.Value < 0:
			candidates = append(candidates, ef)
		}
	}
	if len(candidates) == 0 {
		return
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if cleansePriority(c) > cleansePriority(best) || (cleansePriority(c) == cleansePriority(best) && magnitude(c) > magnitude(best)) {
			best = c
		}
	}
	r.removeCombatEffectNoLock(best)
}

// cleansePriority 阴阳相济/澄心明鉴的移除优先级：流血 > 攻击降 > 命中降 > 躲闪降 > 先手降 > 易伤。
func cleansePriority(ef *CombatEffect) int {
	switch {
	case ef.Op == "bleed":
		return 6
	case ef.Op == "stat_mod" && ef.Stat == "attack":
		return 5
	case ef.Op == "stat_mod" && ef.Stat == "hit":
		return 4
	case ef.Op == "stat_mod" && ef.Stat == "dodge":
		return 3
	case ef.Op == "stat_mod" && ef.Stat == "initiative":
		return 2
	case ef.Op == "vulnerability":
		return 1
	}
	return 0
}

// applyDispelBonusNoLock 移除目标1项先手或攻速增益（镇步伏形），不倒改其已确定的出招CD。
func (r *RuntimeManager) applyDispelBonusNoLock(owner ActorKey, tags []string) {
	var candidates []*CombatEffect
	for _, ef := range r.combatEffects[owner.String()] {
		if ef.Op == "haste" && containsTag(tags, "haste") {
			candidates = append(candidates, ef)
		}
		if ef.Op == "stat_mod" && ef.Stat == "initiative" && ef.Value > 0 && containsTag(tags, "initiative") {
			candidates = append(candidates, ef)
		}
	}
	if len(candidates) == 0 {
		return
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if magnitude(c) > magnitude(best) {
			best = c
		}
	}
	r.removeCombatEffectNoLock(best)
}

// applyStripGuardChargeNoLock 额外消去目标最强次数型护体1次（凝虚破障）。
func (r *RuntimeManager) applyStripGuardChargeNoLock(owner ActorKey) {
	var best *CombatEffect
	for _, ef := range r.combatEffects[owner.String()] {
		if ef.Op != "guard" || ef.Scope != "direct_incoming" {
			continue
		}
		if best == nil || ef.Value > best.Value || (ef.Value == best.Value && ef.Charges > best.Charges) {
			best = ef
		}
	}
	if best != nil {
		r.consumeEffectChargeNoLock(best)
	}
}

// triggerCounterNoLock 防守方姿态反击：每来袭至多一次，取预算最大者；反击不连锁。
func (r *RuntimeManager) triggerCounterNoLock(act combatAction, out combatActionOutcome, now int64) {
	if !act.Target.canAct() {
		return
	}
	tag := "damaged"
	if out.AllMissed {
		tag = "missed"
	}
	var best *CombatEffect
	for _, ef := range r.activeEffectsNoLock(act.Target.key) {
		if ef.Op != "counter" || !containsTag(ef.Tags, tag) {
			continue
		}
		if ef.Scope != "normal_or_active" && ef.Scope != "" {
			continue
		}
		if best == nil || ef.Power > best.Power {
			best = ef
		}
	}
	if best == nil {
		return
	}
	// 反击消耗对应来袭计数（护体已在主结算消耗；这里只消费反击姿态本身）
	r.consumeEffectChargeNoLock(best)
	counter := combatAction{
		Kind: actionCounter, Actor: act.Target, Target: act.Actor,
		Power: best.Power, HitCount: 1, ProcDepth: 1, Now: now, RoomID: act.RoomID,
	}
	// 反击使用触发当时有效攻防与原姿态等级（Power已按施放时S成长快照）
	res := r.resolveCombatActionNoLock(counter)
	out.Messages = append(out.Messages, res.Messages...)
}

// processCombatEffectsTickNoLock 战斗tick开头处理：到期清理→持续伤害/治疗→镜湖待回疗。
// dot不消耗来袭护体、不触发反应，仍遵守切磋/认负线；本tick新建的持续效果下tick才执行。
func (r *RuntimeManager) processCombatEffectsTickNoLock(players map[string]*model.Player, result *model.CombatTickResult) {
	now := combatNowUnix(r)
	r.expireCombatEffectsNoLock(now)
	for ownerKey, list := range r.combatEffects {
		if len(list) == 0 {
			continue
		}
		entity := r.entityByKeyNoLock(parseActorKey(ownerKey), players)
		if entity == nil {
			continue
		}
		kept := list[:0]
		for _, ef := range list {
			switch ef.Op {
			case "bleed":
				if len(ef.TickBudget) == 0 {
					continue
				}
				dmg := ef.TickBudget[0]
				ef.TickBudget = ef.TickBudget[1:]
				ef.Charges = len(ef.TickBudget)
				r.applyEffectDamageNoLock(entity, dmg, ef, result)
				if ef.Charges <= 0 {
					continue
				}
			case "hot":
				if len(ef.TickBudget) == 0 {
					continue
				}
				heal := ef.TickBudget[0]
				ef.TickBudget = ef.TickBudget[1:]
				ef.Charges = len(ef.TickBudget)
				r.applyEffectHealNoLock(entity, heal, result)
				if ef.Charges <= 0 {
					continue
				}
			}
			kept = append(kept, ef)
		}
		if len(kept) == 0 {
			delete(r.combatEffects, ownerKey)
		} else {
			r.combatEffects[ownerKey] = kept
		}
	}
	// 镜湖倒影待回疗：先核存活/同场/未认负，再恢复；战斗结束取消
	for _, b := range r.combatBindings {
		if b.PendingHeal == nil {
			continue
		}
		ph := b.PendingHeal
		b.PendingHeal = nil
		entity := r.entityByKeyNoLock(ph.Owner, players)
		if entity == nil || !entity.canAct() {
			continue
		}
		r.applyEffectHealNoLock(entity, ph.Amount, result)
	}
}

// expireCombatEffectsNoLock 清理到期效果。
func (r *RuntimeManager) expireCombatEffectsNoLock(now int64) {
	for owner, list := range r.combatEffects {
		kept := list[:0]
		for _, ef := range list {
			if ef.ExpiresAt > 0 && ef.ExpiresAt <= now && !(ef.Op == "bleed" && len(ef.TickBudget) > 0) {
				continue
			}
			kept = append(kept, ef)
		}
		if len(kept) == 0 {
			delete(r.combatEffects, owner)
		} else {
			r.combatEffects[owner] = kept
		}
	}
}

// entityByKeyNoLock 按ActorKey解析当前实体（玩家/静态NPC/副本NPC），用于持续效果的HP读写。
func (r *RuntimeManager) entityByKeyNoLock(key ActorKey, players map[string]*model.Player) *combatEntity {
	switch key.Kind {
	case "player":
		if p, ok := players[key.EntityID]; ok && p != nil {
			return playerEntity(p, p.CombatType)
		}
	case "static_npc":
		r.staticNPCsMu.Lock()
		npc := r.staticNPCs[key.EntityID]
		r.staticNPCsMu.Unlock()
		if npc != nil && npc.Alive {
			mode := "KILL"
			return npcEntity(key, npc, mode)
		}
	case "dungeon_npc":
		if npc, ok := r.dungeonNPCByIDNoLock(key.EntityID); ok && npc != nil && npc.Alive {
			return npcEntity(key, npc, "KILL")
		}
	}
	return nil
}

// parseActorKey 还原ActorKey（map键以"kind|instance|entity"存储）。
func parseActorKey(s string) ActorKey {
	parts := splitN3(s)
	return ActorKey{Kind: parts[0], InstanceID: parts[1], EntityID: parts[2]}
}

func splitN3(s string) [3]string {
	var out [3]string
	idx := 0
	for _, ch := range s {
		if ch == '|' {
			if idx < 2 {
				idx++
			}
			continue
		}
		out[idx] += string(ch)
	}
	return out
}

// applyEffectDamageNoLock 持续伤害：不消耗护体不触发反应，遵守认负线与死亡。
func (r *RuntimeManager) applyEffectDamageNoLock(entity *combatEntity, dmg int, ef *CombatEffect, result *model.CombatTickResult) {
	if dmg <= 0 || !entity.canAct() {
		return
	}
	floor := entity.floor()
	dmg = min(dmg, *entity.hp-floor)
	if dmg <= 0 {
		return
	}
	*entity.hp -= dmg
	if entity.isPlayer {
		result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: entity.key.EntityID, Message: "你因流血损失" + itoa(dmg) + "点气血。"})
		r.bumpCombatRevisionNoLock(entity.key.EntityID)
	}
}

// applyEffectHealNoLock 持续治疗：不超过真实缺口。
func (r *RuntimeManager) applyEffectHealNoLock(entity *combatEntity, heal int, result *model.CombatTickResult) {
	if heal <= 0 || entity.hp == nil {
		return
	}
	gap := entity.maxHP - *entity.hp
	if gap <= 0 {
		return
	}
	heal = min(heal, gap)
	*entity.hp += heal
	if entity.isPlayer {
		result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: entity.key.EntityID, Message: "气血缓缓回复了" + itoa(heal) + "点。"})
		r.bumpCombatRevisionNoLock(entity.key.EntityID)
	}
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func itoa(v int) string {
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

// actorMaxHPNoLock 查询角色最大气血（魔极蚀心自损基数）。
func actorMaxHPNoLock(r *RuntimeManager, key ActorKey) int {
	// 施放入口在resolve之前已持有实体视图；此处按key兜底解析。
	switch key.Kind {
	case "player":
		if players, err := r.loadPlayers(); err == nil {
			if p, ok := players[key.EntityID]; ok {
				return effectiveAttributesForPlayerNoLock(p).MaxHP
			}
		}
	case "static_npc":
		r.staticNPCsMu.Lock()
		npc := r.staticNPCs[key.EntityID]
		r.staticNPCsMu.Unlock()
		if npc != nil {
			return model.BuildEffectiveAttributes(npc.Attributes).MaxHP
		}
	case "dungeon_npc":
		if npc, ok := r.dungeonNPCByIDNoLock(key.EntityID); ok && npc != nil {
			return model.BuildEffectiveAttributes(npc.Attributes).MaxHP
		}
	}
	return 0
}
