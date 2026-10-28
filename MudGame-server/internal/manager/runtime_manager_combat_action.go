package manager

import (
	"math/rand"
	"strconv"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/model"
)

// 统一动作结算（实现规范§6）：普攻、主动招式、反击、持续伤害共用一套
// 命中/波动/暴击/招架/防护/结束规则。三条战斗路径全部经由本文件结算。

type combatActionKind int

const (
	actionAuto combatActionKind = iota
	actionMove
	actionCounter
	actionDot
)

func (k combatActionKind) String() string {
	switch k {
	case actionMove:
		return "move"
	case actionCounter:
		return "counter"
	case actionDot:
		return "dot"
	}
	return "auto"
}

// combatEntity 三类战斗参与者的适配视图。hp 指针允许副本defeat模式用临时战斗气血。
type combatEntity struct {
	key        ActorKey
	name       string
	attrs      model.EffectiveAttributes
	hp         *int
	maxHP      int
	mp         *int
	mode       string // KILL / SPAR
	isPlayer   bool
	defeatMode bool // 副本defeat首领：降至25%认负
	yieldLine  int  // SPAR=MaxHP/5；defeat=MaxHP/4；生死战=0
}

func (e *combatEntity) keyString() string { return e.key.String() }

func (e *combatEntity) floor() int {
	switch {
	case e.mode == "SPAR":
		return 1
	case e.defeatMode:
		return e.yieldLine
	default:
		return 0
	}
}

// canAct 报告仍可出手：未死亡且不低于认负线。
func (e *combatEntity) canAct() bool {
	if e.hp == nil {
		return false
	}
	return *e.hp > e.floor()
}

func playerEntity(p *model.Player, mode string) *combatEntity {
	attrs := effectiveAttributesForPlayerNoLock(p)
	key := ActorKey{Kind: "player", EntityID: p.ID}
	return &combatEntity{
		key: key, name: p.CharacterName, attrs: attrs,
		hp: &p.Attributes.Resources.HP, maxHP: attrs.MaxHP, mp: &p.Attributes.Resources.MP,
		mode: mode, isPlayer: true,
	}
}

func npcEntity(key ActorKey, n *engruntime.NPCState, mode string) *combatEntity {
	attrs := model.BuildEffectiveAttributes(n.Attributes)
	e := &combatEntity{
		key: key, name: n.Name, attrs: attrs,
		hp: &n.Attributes.Resources.HP, maxHP: attrs.MaxHP, mp: &n.Attributes.Resources.MP,
		mode: mode,
	}
	if n.CombatMode == "defeat" && mode == "KILL" {
		e.defeatMode = true
		e.yieldLine = max(1, n.Attributes.Resources.MaxHP/4)
	}
	return e
}

type combatSegment struct {
	Hit     bool
	Parried bool
	Crit    bool
	Damage  int
}

type combatActionOutcome struct {
	Kind        combatActionKind
	MoveID      string
	Segments    []combatSegment
	TotalDamage int
	AllMissed   bool
	Messages    []model.RoomMessage
	Stopped     bool // 触及认负线/死亡/防室断气，取消后续出手
}

type combatAction struct {
	Kind        combatActionKind
	MoveID      string
	MoveName    string
	MoveDef     model.CombatMoveDef // 命中后效果来源；普攻/反击为零值
	MoveLevel   int
	BattleID    string
	Actor       *combatEntity
	Target      *combatEntity
	HitCount    int
	Power       float64
	HitBonus    float64
	ArmorIgnore float64
	ParryMult   float64
	GuardPierce float64
	ProcDepth   int
	Now         int64
	RoomID      string
}

// resolveCombatActionNoLock 结算一次直接攻击动作（普攻/招式/反击）。
// 先算整个动作的整数预算，再按段序拆分；各段独立命中/波动/暴击/招架；
// 防护按整个动作取最强并消耗一次；随后按触发过滤器处理至多一次反击。
func (r *RuntimeManager) resolveCombatActionNoLock(act combatAction) combatActionOutcome {
	out := combatActionOutcome{Kind: act.Kind, MoveID: act.MoveID}
	now := act.Now
	if act.Now == 0 {
		now = combatNowUnix(r)
	}
	if act.ParryMult == 0 {
		act.ParryMult = 1
	}
	if act.Power == 0 {
		act.Power = 1 // 普攻/反击无招式倍率
	}
	if act.HitCount <= 0 {
		act.HitCount = 1
	}
	floatRoll, intRoll := r.rollFuncs()

	// ---- 出手侧快照：自身出招修正（stat_mod normal_or_active）----
	aAttack := scaledStatNoLock(act.Actor.attrs.Attack, r.collectedStatModsNoLock(act.Actor.key, "attack", false))
	aHitRate := scaledStatNoLock(act.Actor.attrs.HitRate, r.collectedStatModsNoLock(act.Actor.key, "hit", false))
	aCrit := act.Actor.attrs.CritRate

	// 主动招式：消耗并应用 next_active 蓄势（普攻/反击不享受也不消耗）
	var dmgBuff, moveHitPP, moveCritPP, moveIgnore, movePierce float64
	if act.Kind == actionMove {
		dmgBuff, moveHitPP, moveCritPP, moveIgnore, movePierce = r.consumeNextActiveNoLock(act.Actor.key)
	}
	aCrit = min(25, max(0, aCrit+moveCritPP))
	totalIgnore := min(0.5, act.ArmorIgnore+moveIgnore)
	_ = totalIgnore
	moveHitTotal := act.HitBonus + moveHitPP

	// ---- 防守侧快照：来袭修正（direct_incoming 与 normal_or_active 的防御侧）----
	tDodge := scaledStatNoLock(act.Target.attrs.Dodge, r.incomingStatModsNoLock(act.Target.key, "dodge", true))
	tParry := scaledStatNoLock(act.Target.attrs.Parry, r.incomingStatModsNoLock(act.Target.key, "parry", true))
	tParry = max(0, int(float64(tParry)*r.parryMultProductNoLock(act.Target.key)))
	tDefense := scaledStatNoLock(act.Target.attrs.Defense, r.incomingStatModsNoLock(act.Target.key, "defense", true))
	tDefense = max(0, int(float64(tDefense)*(1-totalIgnore)))

	hitChance := boundedChance(aHitRate, tDodge) + moveHitTotal
	hitChance = min(0.95, max(0.15, hitChance))
	parryChance := boundedParryChance(tParry, aHitRate) * act.ParryMult
	parryChance = min(0.60, max(0, parryChance))

	modeBonus := 14
	if act.Actor.mode == "SPAR" {
		modeBonus = 8
	}
	base := int(float64(aAttack)-float64(tDefense)/3) + modeBonus
	if base < 2 {
		base = 2
	}
	budget := int(float64(base) * act.Power * (1 + dmgBuff))
	if budget < 1 {
		budget = 1
	}

	// ---- 整动作防护与减伤（多段整体一次）----
	guard, guardEf := r.strongestGuardNoLock(act.Target.key, act)
	pierce := min(0.30, act.GuardPierce+movePierce)
	guardEffective := guard * (1 - pierce)
	outgoingReduce := r.outgoingReduceProductNoLock(act.Actor.key)
	vuln := r.vulnerabilitySumNoLock(act.Target.key)
	totalReduction := 1 - (1-guardEffective)*(1-outgoingReduce)
	totalReduction = min(0.65, max(0, totalReduction))
	finalMul := (1 - totalReduction) * (1 + vuln)

	// ---- 预算拆段：余数按段序分配，保持总和 ----
	segments := make([]int, act.HitCount)
	for i := 0; i < budget; i++ {
		segments[i%act.HitCount]++
	}

	targetFloor := act.Target.floor()
	capTotal := *act.Target.hp - targetFloor
	sparYield := 0
	if act.Actor.mode == "SPAR" {
		sparYield = act.Target.maxHP / 5
	}

	out.Segments = make([]combatSegment, 0, act.HitCount)
	stopped := false
	for _, seg := range segments {
		segRes := combatSegment{}
		if stopped || capTotal <= 0 {
			out.Segments = append(out.Segments, segRes)
			continue
		}
		segRes.Hit = floatRoll() < hitChance
		if !segRes.Hit {
			out.Segments = append(out.Segments, segRes)
			continue
		}
		damage := seg * (90 + intRoll(21)) / 100
		segRes.Crit = floatRoll() < critChanceValue(aCrit)
		if segRes.Crit {
			damage *= 2
		}
		segRes.Parried = floatRoll() < parryChance
		if segRes.Parried {
			damage /= 2
		}
		damage = int(float64(damage) * finalMul)
		if damage < 1 {
			damage = 1
		}
		damage = min(damage, capTotal)
		if sparYield > 0 {
			damage = min(damage, sparYield)
		}
		segRes.Damage = damage
		capTotal -= damage
		out.TotalDamage += damage
		projected := *act.Target.hp - out.TotalDamage
		if act.Actor.mode == "SPAR" && projected <= sparYield {
			stopped = true // 认输线：取消之后的普攻、剩余段与反击
		}
		if projected <= targetFloor {
			stopped = true // 死亡或defeat认负
		}
		out.Segments = append(out.Segments, segRes)
	}
	out.AllMissed = out.TotalDamage == 0
	out.Stopped = stopped

	// ---- 扣血与日志 ----
	msg := r.actionMessage(act, out)
	if msg != "" {
		out.Messages = append(out.Messages, model.RoomMessage{RoomID: act.RoomID, Message: msg})
	}
	*act.Target.hp = max(targetFloor, *act.Target.hp-out.TotalDamage)

	// ---- 来袭防护/闪避按动作消耗一次；落空仍消耗 ----
	if guardEf != nil {
		r.consumeEffectChargeNoLock(guardEf)
	}
	r.consumeIncomingChargesNoLock(act.Target.key)

	// ---- 触发域处理：仅总实扣>0时执行命中后效果 ----
	if out.TotalDamage > 0 && act.ProcDepth == 0 {
		r.applyMoveHitEffectsNoLock(act.MoveDef, act.MoveLevel, act, out, act.BattleID)
	}

	// ---- 最近事件（供踏浪穿隙/踏雪折影判定；多段合为一笔）----
	if act.ProcDepth == 0 {
		if binding := r.currentBindingNoLock(act.Target.key); binding != nil {
			r.recordCombatEventNoLock(binding, RecentCombatEvent{
				BattleID: binding.ID, Source: act.Actor.key, ActionKind: act.Kind.String(),
				Dodged: out.AllMissed, ActualDamage: out.TotalDamage, At: now,
			})
		}
	}

	// ---- 反击：每来袭至多一次，取预算最大者；反击不连锁 ----
	if act.ProcDepth == 0 {
		r.triggerCounterNoLock(act, out, now)
	}
	return out
}

func critChanceValue(crit float64) float64 { return min(25, max(0, crit)) / 100 }

func (r *RuntimeManager) rollFuncs() (func() float64, func(int) int) {
	if r.combatRandom != nil {
		return r.combatRandom.Float64, r.combatRandom.Intn
	}
	return rand.Float64, rand.Intn
}

// actionMessage 生成动作日志：命中/招架/闪避→伤害→段数。
func (r *RuntimeManager) actionMessage(act combatAction, out combatActionOutcome) string {
	name := act.Actor.name
	target := act.Target.name
	verb := "一招击中"
	if act.Kind == actionMove {
		verb = "使出" + act.MoveName + "击中"
	} else if act.Kind == actionCounter {
		verb = "反击" + target + "，造成"
	}
	if out.AllMissed {
		if act.Kind == actionMove {
			return name + "使出" + act.MoveName + "，但被" + target + "躲开了！"
		}
		return name + "发起攻击，但被" + target + "躲开了！"
	}
	if len(out.Segments) > 1 {
		hits := 0
		for _, seg := range out.Segments {
			if seg.Hit {
				hits++
			}
		}
		return name + "使出" + act.MoveName + "，" + strconv.Itoa(hits) + "段命中，共造成" + strconv.Itoa(out.TotalDamage) + "点伤害。"
	}
	seg := out.Segments[0]
	if seg.Damage <= 0 {
		return name + verb + target + "，但没有造成任何伤害。"
	}
	damageText := strconv.Itoa(out.TotalDamage)
	if act.Kind == actionCounter {
		return name + "反击" + target + "，造成" + damageText + "点伤害。"
	}
	if seg.Crit {
		return name + "发动致命一击，重创" + target + "，造成" + damageText + "点伤害。"
	}
	return name + verb + target + "，造成" + damageText + "点伤害。"
}

// ---- 效果查询与数值合并 ----

// 出手侧只消费攻击属性（attack/hit/crit_pp）的 normal_or_active 修正。
func (r *RuntimeManager) collectedStatModsNoLock(owner ActorKey, stat string, defensive bool) float64 {
	total := 0.0
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op != "stat_mod" || ef.Stat != stat {
			continue
		}
		if defensive {
			if ef.Scope != "direct_incoming" && ef.Scope != "normal_or_active" {
				continue
			}
		} else {
			if ef.Scope != "normal_or_active" {
				continue
			}
			r.consumeEffectChargeNoLock(ef) // 出手侧修正每动作消费一次
		}
		total += ef.Value
	}
	return clampStatMod(stat, total)
}

// incomingStatModsNoLock 防御侧修正：direct_incoming 与 normal_or_active 的防御属性。
// 不在这里消耗——整动作结算后由 consumeIncomingChargesNoLock 统一扣一次。
func (r *RuntimeManager) incomingStatModsNoLock(owner ActorKey, stat string, _ bool) float64 {
	total := 0.0
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op != "stat_mod" || ef.Stat != stat {
			continue
		}
		if ef.Scope == "direct_incoming" || ef.Scope == "normal_or_active" {
			total += ef.Value
		}
	}
	return clampStatMod(stat, total)
}

// clampStatMod 同类数值增减上限：躲闪提升最多45%，其余±35%。
func clampStatMod(stat string, total float64) float64 {
	if stat == "dodge" {
		return min(0.45, max(-0.35, total))
	}
	return min(0.35, max(-0.35, total))
}

func scaledStatNoLock(base int, mod float64) int {
	return max(0, int(float64(base)*(1+mod)))
}

// parryMultProductNoLock 对手承受攻击时招架概率乘数（锁浪借势）：乘数=max(0.3,1-Value)。
func (r *RuntimeManager) parryMultProductNoLock(owner ActorKey) float64 {
	mult := 1.0
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op == "stat_mod" && ef.Stat == "parry_mult" {
			mult *= max(0.3, 1-ef.Value)
			r.consumeEffectChargeNoLock(ef)
		}
	}
	return mult
}

// consumeNextActiveNoLock 消耗并返回主动招式的蓄势加成；每条各扣一次。
func (r *RuntimeManager) consumeNextActiveNoLock(actor ActorKey) (dmg, hitPP, critPP, ignore, pierce float64) {
	for _, ef := range r.activeEffectsNoLock(actor) {
		if ef.Op != "next_active" {
			continue
		}
		switch ef.Stat {
		case "damage":
			dmg += ef.Value
		case "hit_pp":
			hitPP += ef.Value
		case "crit_pp":
			critPP += ef.Value
		case "armor_ignore":
			ignore += ef.Value
		case "guard_pierce":
			pierce += ef.Value
		}
		r.consumeEffectChargeNoLock(ef)
	}
	hitPP = min(0.25, hitPP)
	return
}

// strongestGuardNoLock 取owner名下匹配动作范围的最强护体；multi_active只匹配多段主动。
func (r *RuntimeManager) strongestGuardNoLock(owner ActorKey, act combatAction) (float64, *CombatEffect) {
	var best *CombatEffect
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op != "guard" {
			continue
		}
		if ef.Scope == "multi_active" {
			if !(act.Kind == actionMove && act.HitCount > 1) {
				continue
			}
		} else if ef.Scope != "direct_incoming" && ef.Scope != "" {
			continue
		}
		if best == nil || ef.Value > best.Value {
			best = ef
		}
	}
	if best == nil {
		return 0, nil
	}
	return best.Value, best
}

// outgoingReduceProductNoLock 攻击方自身的出招伤害减免（截脉封劲）。
func (r *RuntimeManager) outgoingReduceProductNoLock(owner ActorKey) float64 {
	product := 1.0
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op == "outgoing_reduce" {
			product *= (1 - min(0.35, ef.Value))
			r.consumeEffectChargeNoLock(ef)
		}
	}
	return 1 - product
}

// vulnerabilitySumNoLock 防守方承受的易伤增伤。
func (r *RuntimeManager) vulnerabilitySumNoLock(owner ActorKey) float64 {
	total := 0.0
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op == "vulnerability" {
			total += ef.Value
			r.consumeEffectChargeNoLock(ef)
		}
	}
	return total
}

// consumeIncomingChargesNoLock 一次来袭消耗防御侧计数型效果：
// direct_incoming 全部，以及 normal_or_active 中防御属性的敌向削弱（雷回夺势等）。
func (r *RuntimeManager) consumeIncomingChargesNoLock(owner ActorKey) {
	for _, ef := range r.activeEffectsNoLock(owner) {
		if ef.Op != "stat_mod" {
			continue
		}
		defensiveStat := ef.Stat == "parry" || ef.Stat == "defense" || ef.Stat == "dodge" || ef.Stat == "parry_mult"
		if ef.Scope == "direct_incoming" || (ef.Scope == "normal_or_active" && defensiveStat) {
			r.consumeEffectChargeNoLock(ef)
		}
	}
}
