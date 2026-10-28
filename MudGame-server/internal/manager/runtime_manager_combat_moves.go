package manager

import (
	"errors"

	"mudgame/server/internal/model"
)

// 主动招式施放入口（实现规范§5）：原子结算、双CD、序号重放保护。
// 失败返回业务code并附权威视图；普通error仅用于玩家不存在等系统级异常。

const combatMovesCapability = "combat_moves_v1"

// playerMoveEntry 当前合法可用的一个招式（五位置装备/回退后）。
type playerMoveEntry struct {
	ArtID string
	Slot  string
	Level int
	Def   model.CombatMoveDef
}

// currentWeaponBasicSkillNoLock 战斗位回退：按主兵器匹配基本功，空手用拳脚。
func currentWeaponBasicSkillNoLock(p *model.Player) string {
	wt := currentWeaponType(p)
	switch wt {
	case "blade":
		return "basic_blade"
	case "sword":
		return "basic_sword"
	case "staff":
		return "basic_staff"
	case "hidden_weapon":
		return "basic_hidden_weapon"
	case "spear":
		return "basic_spear"
	case "whip":
		return "basic_whip"
	case "mace":
		return "basic_mace"
	case "hook":
		return "basic_hook"
	}
	return "basic_unarmed"
}

// movesForPlayerNoLock 五位置招式：装备命名武学优先，空缺回退基本功；
// 战斗位按主武器匹配；未学/传承门不通过的位置不提供招式。
func movesForPlayerNoLock(p *model.Player) []playerMoveEntry {
	out := make([]playerMoveEntry, 0, 15)
	addArt := func(artID, slot string) {
		if artID == "" {
			return
		}
		def, ok := fileSkills[artID]
		if !ok || p.Skills[artID] < 1 {
			return
		}
		if reason := skillActiveReason(p, def); reason != "" {
			return // 兵器不符/传承门不通过：该位置不提供招式
		}
		for _, m := range def.Moves {
			out = append(out, playerMoveEntry{ArtID: artID, Slot: slot, Level: p.Skills[artID], Def: m})
		}
	}
	// 战斗位：先装备的命名武学；兵器不符或未装备时按主兵器回退基本功
	combatArt := p.ActiveArts["combat"]
	if combatArt != "" {
		addArt(combatArt, "combat")
	}
	if len(out) == 0 || !hasSlotMove(out, "combat") {
		addArt(currentWeaponBasicSkillNoLock(p), "combat")
	}
	addArt(p.ActiveArts["internal"], "internal")
	if !hasSlotMove(out, "internal") {
		addArt("basic_neigong", "internal")
	}
	addArt(p.ActiveArts["movement"], "movement")
	if !hasSlotMove(out, "movement") {
		addArt("basic_qinggong", "movement")
	}
	addArt(p.ActiveArts["guard"], "guard")
	if !hasSlotMove(out, "guard") {
		addArt("basic_parry", "guard")
	}
	addArt(p.ActiveArts["study"], "study")
	if !hasSlotMove(out, "study") {
		addArt("basic_literacy", "study")
	}
	return out
}

func hasSlotMove(list []playerMoveEntry, slot string) bool {
	for _, e := range list {
		if e.Slot == slot {
			return true
		}
	}
	return false
}

// effectiveCastSpeedNoLock 有效出招攻速：基础攻速乘haste增益（取最强，≤+50%）。
func (r *RuntimeManager) effectiveCastSpeedNoLock(owner ActorKey, base float64) float64 {
	if base <= 0 {
		base = 4
	}
	haste := 0.0
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op == "haste" && ef.Value > haste {
			haste = ef.Value
		}
	}
	return base * (1 + min(0.5, haste))
}

// currentBattleIDNoLock 本人当前战斗令牌；无绑定返回空串。
func (r *RuntimeManager) currentBattleIDNoLock(self ActorKey) string {
	if b := r.currentBindingNoLock(self); b != nil {
		return b.ID
	}
	return ""
}

// buildCombatMovesViewNoLock 组装get_combat_moves权威视图。
func (r *RuntimeManager) buildCombatMovesViewNoLock(p *model.Player, players map[string]*model.Player) *model.CombatMovesView {
	r.initCombatStateNoLock()
	now := combatNowUnix(r)
	key := ActorKey{Kind: "player", EntityID: p.ID}
	eff := effectiveAttributesForPlayerNoLock(p)
	speed := r.effectiveCastSpeedNoLock(key, eff.EffectiveActionSpeed)
	actionDur := model.ActionDurationMs(4000, speed)
	view := &model.CombatMovesView{
		Capabilities:         []string{combatMovesCapability},
		ServerEpoch:          r.combatEpoch,
		Revision:             r.combatRevision[p.ID],
		ServerTime:           now,
		BattleID:             r.currentBattleIDNoLock(key),
		Sequence:             p.CombatMoves.Sequence,
		MP:                   p.Attributes.Resources.MP,
		EffectiveActionSpeed: speed,
		ActionStartedAt:      p.CombatMoves.Action.StartedAt,
		ActionReadyAt:        p.CombatMoves.Action.ReadyAt,
		ActionDurationMs:     actionDur,
	}
	binding := r.currentBindingNoLock(key)
	inCombat := p.State == "COMBAT" && binding != nil
	for _, entry := range movesForPlayerNoLock(p) {
		mv := model.CombatMoveView{
			ID: entry.Def.ID, Name: entry.Def.Name, ArtID: entry.ArtID, Slot: entry.Slot,
			Level: entry.Level, Kind: entry.Def.Kind, Description: entry.Def.Description,
			ActionMs: int64(entry.Def.ActionMs), CooldownSec: entry.Def.CooldownSec, MPCost: entry.Def.MPCost,
			HitCount: entry.Def.HitCount,
			Power:    entry.Def.EffectivePower(entry.Level), HitBonusPP: entry.Def.EffectiveHitBonus(entry.Level),
			ArmorIgnore: entry.Def.EffectiveArmorIgnore(entry.Level), ParryReduction: entry.Def.ParryReduction,
			Conditions: entry.Def.Conditions,
			Effects:    entry.Def.EffectiveEffects(entry.Level),
		}
		if skill, ok := p.CombatMoves.Skills[entry.Def.ID]; ok {
			mv.SkillStartedAt, mv.SkillReadyAt = skill.StartedAt, skill.ReadyAt
		}
		if block, ok := p.CombatMoves.Blocks[entry.Def.ID]; ok {
			mv.BlockedSince, mv.BlockedUntil = block.StartedAt, block.ReadyAt
		}
		mv.UnavailableReason = r.moveUnavailableReasonNoLock(p, binding, inCombat, entry, now, players)
		view.Moves = append(view.Moves, mv)
	}
	return view
}

// moveUnavailableReasonNoLock 非冷却类资格的具体原因；进度空不代表其他资格满足。
func (r *RuntimeManager) moveUnavailableReasonNoLock(p *model.Player, binding *combatBinding, inCombat bool, entry playerMoveEntry, now int64, players map[string]*model.Player) string {
	if !inCombat {
		return "未在战斗中"
	}
	// 条件与目标资格
	target, terr := r.targetEntityByBindingNoLock(binding, players)
	if terr != nil || target == nil {
		return "目标已失效"
	}
	if reason := r.moveConditionReasonNoLock(p, binding, entry.Def, target, now); reason != "" {
		return reason
	}
	// 内力（按折扣后实付）
	if cost := r.discountedMoveCostNoLock(p, entry.Def); p.Attributes.Resources.MP < cost {
		return "内力不足"
	}
	// 纯治疗无缺口
	if reason := r.moveBenefitReasonNoLock(p, entry.Def); reason != "" {
		return reason
	}
	return ""
}

// targetEntityByBindingNoLock 按绑定解析当前目标实体。
func (r *RuntimeManager) targetEntityByBindingNoLock(binding *combatBinding, players map[string]*model.Player) (*combatEntity, error) {
	if binding == nil {
		return nil, errors.New("no binding")
	}
	switch binding.Target.Kind {
	case "player":
		p, ok := players[binding.Target.EntityID]
		if !ok || p == nil {
			return nil, errors.New("target lost")
		}
		return playerEntity(p, binding.Mode), nil
	case "static_npc":
		r.staticNPCsMu.Lock()
		npc := r.staticNPCs[binding.Target.EntityID]
		r.staticNPCsMu.Unlock()
		if npc == nil || !npc.Alive {
			return nil, errors.New("target lost")
		}
		return npcEntity(binding.Target, npc, binding.Mode), nil
	case "dungeon_npc":
		npc, ok := r.dungeonNPCByIDNoLock(binding.Target.EntityID)
		if !ok || npc == nil || !npc.Alive {
			return nil, errors.New("target lost")
		}
		return npcEntity(binding.Target, npc, binding.Mode), nil
	}
	return nil, errors.New("unknown target")
}

// moveConditionReasonNoLock 施放条件检查；不满足返回具体原因。
func (r *RuntimeManager) moveConditionReasonNoLock(p *model.Player, binding *combatBinding, def model.CombatMoveDef, target *combatEntity, now int64) string {
	eff := effectiveAttributesForPlayerNoLock(p)
	for _, c := range def.Conditions {
		switch c.Kind {
		case "self_hp_below":
			if float64(p.Attributes.Resources.HP) >= float64(eff.MaxHP)*c.Value {
				return "气血条件不满足"
			}
		case "target_hp_at_most":
			if float64(*target.hp) > float64(target.maxHP)*c.Value {
				return "目标气血条件不满足"
			}
		case "recent_dodge":
			if binding.recentEventNoLock(now, binding.Target, "", false) == nil {
				return "尚无可乘之隙"
			}
		case "recent_active_damage":
			if binding.recentEventNoLock(now, binding.Target, "move", true) == nil {
				return "尚无recent_active_damage条件"
			}
		case "self_hp_cost":
			cost := int(float64(eff.MaxHP) * c.Value)
			if p.Attributes.Resources.HP-cost <= selfDamageFloor(eff, binding.Mode) {
				return "气血不足以支付代价"
			}
		}
	}
	return ""
}

// selfDamageFloor 自损后必须高于的界：生死战>0，切磋不触认输线。
func selfDamageFloor(eff model.EffectiveAttributes, mode string) int {
	if mode == "SPAR" {
		return eff.MaxHP / 5
	}
	return 0
}

// discountedMoveCostNoLock 聚气归元折扣后的实付内力；每次最低1。
func (r *RuntimeManager) discountedMoveCostNoLock(p *model.Player, def model.CombatMoveDef) int {
	discount := 0.0
	for _, ef := range r.activeEffectsNoLock(ActorKey{Kind: "player", EntityID: p.ID}) {
		if ef.Op == "mp_discount" && ef.Value > discount {
			discount = ef.Value
		}
	}
	cost := def.MPCost - int(float64(def.MPCost)*discount)
	return max(1, cost)
}

// moveBenefitReasonNoLock 纯治疗在无缺口时不可用；组合招式仍可施放。
func (r *RuntimeManager) moveBenefitReasonNoLock(p *model.Player, def model.CombatMoveDef) string {
	onlyHeal := len(def.Effects) > 0
	for _, e := range def.Effects {
		if e.Op != "heal" && e.Op != "hot" {
			onlyHeal = false
			break
		}
	}
	if !onlyHeal {
		return ""
	}
	eff := effectiveAttributesForPlayerNoLock(p)
	if p.Attributes.Resources.HP >= eff.MaxHP {
		return "气血已满"
	}
	return ""
}

// GetCombatMoves 返回本人当前合法招式与冷却视图（非战斗可查，battleId为空且不可施放）。
func (r *RuntimeManager) GetCombatMoves(playerID string) (model.CombatMovesView, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	players, err := r.loadPlayers()
	if err != nil {
		return model.CombatMovesView{}, err
	}
	p, ok := players[playerID]
	if !ok {
		return model.CombatMovesView{}, errors.New("player_not_found")
	}
	return *r.buildCombatMovesViewNoLock(p, players), nil
}

// CastCombatMove 施放一次主动招式：锁内验证→扣费→即时结算→双CD→序号推进，全部原子提交。
func (r *RuntimeManager) CastCombatMove(playerID, moveID, battleID string, expectedSequence int64) (model.CombatMoveResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	players, err := r.loadPlayers()
	if err != nil {
		return model.CombatMoveResult{}, err
	}
	p, ok := players[playerID]
	if !ok {
		return model.CombatMoveResult{}, errors.New("player_not_found")
	}
	res := model.CombatMoveResult{}
	reject := func(code, message string) (model.CombatMoveResult, error) {
		res.Code = code
		res.Message = message
		res.View = r.buildCombatMovesViewNoLock(p, players)
		return res, nil
	}

	key := ActorKey{Kind: "player", EntityID: p.ID}
	// 1. 序号：只有相等才执行；较旧返回stale_sequence与最新视图，不执行不重播
	if expectedSequence != p.CombatMoves.Sequence {
		return reject("stale_sequence", "招式状态已更新，请以最新视图重试。")
	}
	// 2. 招式与资格
	var entry *playerMoveEntry
	moveList := movesForPlayerNoLock(p)
	for i := range moveList {
		if moveList[i].Def.ID == moveID {
			entry = &moveList[i]
			break
		}
	}
	if entry == nil {
		return reject("invalid_move", "该招式当前不可用。")
	}
	// 3. 战斗绑定与目标
	binding := r.currentBindingNoLock(key)
	if binding == nil || p.State != "COMBAT" {
		return reject("not_in_combat", "未处于战斗中。")
	}
	if battleID == "" || battleID != binding.ID {
		return reject("stale_battle", "战斗状态已变化，请刷新后重试。")
	}
	if room, ok := fileRooms[p.RoomID]; ok && room.SafeZone {
		return reject("safe_zone", "安全区内不能出招。")
	}
	target, terr := r.targetEntityByBindingNoLock(binding, players)
	if terr != nil || target == nil {
		return reject("target_lost", "对手已经不在了。")
	}
	now := combatNowUnix(r)
	// 4. 双CD：先验证，失败不扣费不启动任何CD
	if p.CombatMoves.Action.ReadyAt > now {
		return reject("action_cooldown", "出招尚在冷却。")
	}
	if skill, ok := p.CombatMoves.Skills[moveID]; ok && skill.ReadyAt > now {
		return reject("skill_cooldown", "此招尚在冷却。")
	}
	// 5. 条件、内力、收益预检（只读计划）
	if reason := r.moveConditionReasonNoLock(p, binding, entry.Def, target, now); reason != "" {
		return reject("condition_not_met", reason+"。")
	}
	cost := r.discountedMoveCostNoLock(p, entry.Def)
	if p.Attributes.Resources.MP < cost {
		return reject("mp_not_enough", "内力不足。")
	}
	if reason := r.moveBenefitReasonNoLock(p, entry.Def); reason != "" {
		return reject("no_benefit", reason+"。")
	}
	if !target.canAct() {
		return reject("target_invalid", "对手已无法战斗。")
	}
	// ---- 原子提交：扣费→效果→结算→双CD→序号 ----
	p.Attributes.Resources.MP -= cost
	r.consumeMPDiscountNoLock(key)
	// 条件型招式成功使用即消耗对应最近事件，防止重复兑换
	for _, c := range entry.Def.Conditions {
		if c.Kind == "recent_dodge" {
			if ev := binding.recentEventNoLock(now, binding.Target, "", false); ev != nil {
				binding.consumeRecentEventNoLock(ev)
			}
		}
		if c.Kind == "recent_active_damage" {
			if ev := binding.recentEventNoLock(now, binding.Target, "move", true); ev != nil {
				binding.consumeRecentEventNoLock(ev)
			}
		}
	}
	selfDamage := r.applyMoveCastEffectsNoLock(entry.Def, entry.Level, key, binding.Target, binding.ID, cost)
	if selfDamage > 0 {
		p.Attributes.Resources.HP = max(selfDamageFloor(effectiveAttributesForPlayerNoLock(p), binding.Mode)+0, p.Attributes.Resources.HP-selfDamage)
	}

	result := model.CombatTickResult{}
	if entry.Def.Kind == "attack" {
		actorEnt := playerEntity(p, binding.Mode)
		out := r.resolveCombatActionNoLock(combatAction{
			Kind: actionMove, MoveID: entry.Def.ID, MoveName: entry.Def.Name,
			MoveDef: entry.Def, MoveLevel: entry.Level, BattleID: binding.ID,
			Actor: actorEnt, Target: target,
			HitCount: entry.Def.HitCount, Power: entry.Def.EffectivePower(entry.Level),
			HitBonus: entry.Def.EffectiveHitBonus(entry.Level), ArmorIgnore: entry.Def.EffectiveArmorIgnore(entry.Level),
			ParryMult: entry.Def.ParryMultiplier(entry.Level),
			Now:       now, RoomID: p.RoomID,
		})
		result.Messages = append(result.Messages, out.Messages...)
		r.finalizeMoveTargetNoLock(p, players, actorEnt, target, out, binding, &result)
	} else if entry.Def.Kind == "debuff" {
		// 纯敌向减益：按正常命中概率单独判定，无伤害/暴击/招架
		floatRoll, _ := r.rollFuncs()
		eff := effectiveAttributesForPlayerNoLock(p)
		hitChance := boundedChance(eff.HitRate, target.attrs.Dodge)
		if floatRoll() < hitChance {
			r.applyDebuffCastNoLock(entry.Def, entry.Level, key, binding.Target, binding.ID)
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: p.RoomID, Message: p.CharacterName + "使出" + entry.Def.Name + "，干扰了" + target.name + "的气势。"})
		} else {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: p.RoomID, Message: p.CharacterName + "使出" + entry.Def.Name + "，被" + target.name + "看破了。"})
		}
	} else {
		// support：防护/恢复/蓄势已在applyMoveCastEffects落效果
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: p.RoomID, Message: p.CharacterName + "使出了" + entry.Def.Name + "。"})
	}

	// 6. 双CD：出招CD按施放后攻速快照；技能CD固定；更新全部按钮合并窗口
	speed := r.effectiveCastSpeedNoLock(key, effectiveAttributesForPlayerNoLock(p).EffectiveActionSpeed)
	actionDur := model.ActionDurationMs(entry.Def.ActionMs, speed)
	actionWindow := model.CooldownWindow{StartedAt: now, ReadyAt: now + actionDur}
	skillWindow := model.CooldownWindow{StartedAt: now, ReadyAt: now + int64(entry.Def.CooldownSec)*1000}
	p.CombatMoves.Action = actionWindow
	if p.CombatMoves.Skills == nil {
		p.CombatMoves.Skills = map[string]model.CooldownWindow{}
	}
	p.CombatMoves.Skills[moveID] = skillWindow
	if p.CombatMoves.Blocks == nil {
		p.CombatMoves.Blocks = map[string]model.CooldownWindow{}
	}
	for _, sk := range fileSkills {
		for _, m := range sk.Moves {
			own := skillWindow
			if m.ID != moveID {
				own = p.CombatMoves.Skills[m.ID]
			}
			p.CombatMoves.Blocks[m.ID] = model.MergeMoveBlock(p.CombatMoves.Blocks[m.ID], actionWindow, own, now)
		}
	}
	p.CombatMoves.Sequence++
	r.bumpCombatRevisionNoLock(p.ID)
	if err := r.savePlayers(players); err != nil {
		return model.CombatMoveResult{}, err
	}
	res.View = r.buildCombatMovesViewNoLock(p, players)
	res.Effects = result
	res.AppliedSequence = p.CombatMoves.Sequence
	res.Message = "施放成功"
	return res, nil
}

// consumeMPDiscountNoLock 消耗一次聚气归元折扣（仅在成功扣费后调用）。
func (r *RuntimeManager) consumeMPDiscountNoLock(owner ActorKey) {
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op == "mp_discount" {
			r.consumeEffectChargeNoLock(ef)
			return
		}
	}
}

// finalizeMoveTargetNoLock 主动招式击败/致认负后的终止处理（与tick终止规则一致）。
func (r *RuntimeManager) finalizeMoveTargetNoLock(p *model.Player, players map[string]*model.Player, actorEnt, target *combatEntity, out combatActionOutcome, binding *combatBinding, result *model.CombatTickResult) {
	if !out.Stopped && target.canAct() && actorEnt.canAct() {
		return
	}
	roomID := p.RoomID
	switch target.key.Kind {
	case "player":
		tp, ok := players[target.key.EntityID]
		if !ok {
			return
		}
		if binding.Mode == "KILL" && *target.hp <= 0 {
			tp.State = "DEAD"
			tp.TargetID = ""
			tp.CombatType = ""
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: tp.CharacterName + "大叫一声，倒在地上，死了！"})
			r.clearCombatNoLock(p)
		} else if binding.Mode == "SPAR" && *target.hp <= target.maxHP/5 {
			r.clearCombatNoLock(p)
			r.clearCombatNoLock(tp)
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: tp.CharacterName + "后退三尺，抱拳道：佩服、佩服！"})
		}
	case "static_npc":
		r.staticNPCsMu.Lock()
		npc := r.staticNPCs[target.key.EntityID]
		r.staticNPCsMu.Unlock()
		if npc == nil {
			return
		}
		if binding.Mode == "KILL" && *target.hp <= 0 {
			npc.Alive = false
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: npc.Name + "大叫一声，倒在地上，死了！"})
			r.clearCombatNoLock(p)
		} else if binding.Mode == "SPAR" && *target.hp <= target.maxHP/5 {
			r.clearCombatNoLock(p)
			npc.Attributes.Resources.HP = npc.Attributes.Resources.MaxHP
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: npc.Name + "后退三尺，抱拳道：佩服、佩服！"})
		}
	case "dungeon_npc":
		if *target.hp <= target.floor() {
			npc, _ := r.dungeonNPCByIDNoLock(target.key.EntityID)
			name := target.name
			defeatOnly := npc != nil && npc.CombatMode == "defeat"
			if inst, defeated, _, ok := r.defeatDungeonNPCNoLock(target.key.EntityID); ok {
				if defeated != nil && defeated.Name != "" {
					name = defeated.Name
				}
				if inst != nil && defeatOnly {
					result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你击败了" + name + "。当前完成度" + intToString(inst.Progress) + "%"})
				}
			}
			if defeatOnly {
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: name + "踉跄后退，已经无力再战。"})
			} else {
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: name + "大叫一声，倒在地上，死了！"})
			}
			r.clearCombatNoLock(p)
		}
	}
	// 战斗结束：清双方绑定与两场效果（CD保留）
	if p.State != "COMBAT" {
		r.endCombatBindingNoLock(actorEnt.key)
		r.endCombatBindingNoLock(target.key)
	}
}
