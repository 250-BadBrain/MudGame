package model

// 武功专属主动招式的数据契约（设计来源 docs/design/active_martial_moves_implementation.md §2/§5）。
// 招式是原武功的下属动作：稳定ID为 武功ID/m01～m03，学会武功即获得本门全部招式。
// 运行时禁止任何脚本/eval：全部效果由 Op 白名单数据组合实现。

// CooldownWindow 一次连续受限窗口的起止（Unix 毫秒，服务端权威）。
type CooldownWindow struct {
	StartedAt int64 `json:"startedAt,omitempty"`
	ReadyAt   int64 `json:"readyAt,omitempty"`
}

// Expired 报告窗口在给定时刻是否已结束。
func (w CooldownWindow) Expired(now int64) bool { return w.ReadyAt <= now }

// Active 报告窗口是否正在限制中（未结束且起点不晚于当前时刻）。
func (w CooldownWindow) Active(now int64) bool { return w.ReadyAt > now && w.StartedAt <= now }

// CombatMoveState 玩家的主动招式冷却状态，随存档持久化（Unix 毫秒，离线继续计时）。
type CombatMoveState struct {
	// Sequence 服务端单调递增施放序号：只有施放成功才推进。
	Sequence int64 `json:"sequence,omitempty"`
	// Action 角色级出招CD窗口：期间所有主动招式都不可用，普攻不受影响。
	Action CooldownWindow `json:"action,omitempty"`
	// Skills 逐招技能CD窗口：键为稳定招式ID。
	Skills map[string]CooldownWindow `json:"skills,omitempty"`
	// Blocks 逐按钮合并阻塞窗口（仅用于表现，不是第三种CD）：键为稳定招式ID。
	Blocks map[string]CooldownWindow `json:"blocks,omitempty"`
}

// MoveCondition 招式施放条件。Kind 仅允许 self_hp_below、target_hp_at_most、
// recent_dodge、recent_active_damage、self_hp_cost（见实现规范§2）。
type MoveCondition struct {
	Kind  string  `json:"kind"`
	Value float64 `json:"value,omitempty"`
}

// MoveEffect 单条效果：Op 为白名单枚举，组合表达84招，禁止运行时脚本。
type MoveEffect struct {
	Op         string   `json:"op"`
	Target     string   `json:"target,omitempty"` // self / opponent
	Trigger    string   `json:"trigger,omitempty"`
	Scope      string   `json:"scope,omitempty"`
	Group      string   `json:"group,omitempty"` // 同组取强，不叠层
	Stat       string   `json:"stat,omitempty"`
	Value      float64  `json:"value,omitempty"` // 100级幅度，随S成长
	Cap        float64  `json:"cap,omitempty"`   // 该项基础上限（比例类）
	Power      float64  `json:"power,omitempty"` // 反击/流血的D倍率
	Charges    int      `json:"charges,omitempty"`
	Ticks      int      `json:"ticks,omitempty"`
	DurationMs int      `json:"durationMs,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// CombatMoveDef 单个主动招式定义，配置于原武功的 moves 子数组。
type CombatMoveDef struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Kind           string          `json:"kind"` // attack / support / debuff
	Description    string          `json:"description,omitempty"`
	CooldownSec    int             `json:"cooldownSec"` // 固定10～60秒，攻速不改变
	ActionMs       int             `json:"actionMs"`    // 基础出招时间，本目录统一4000
	MPCost         int             `json:"mpCost"`
	HitCount       int             `json:"hitCount,omitempty"` // attack为1～3；support/debuff为0
	Power          float64         `json:"power,omitempty"`
	HitBonusPP     float64         `json:"hitBonusPP,omitempty"`
	ArmorIgnore    float64         `json:"armorIgnore,omitempty"`
	ParryReduction float64         `json:"parryReduction,omitempty"` // 削减幅度："乘0.5"存0.5
	Conditions     []MoveCondition `json:"conditions,omitempty"`
	Effects        []MoveEffect    `json:"effects,omitempty"`
}

// NPCCombatArt NPC 的只读战斗配置：不进入教学 Skills，防止玩家免费请教绝学。
type NPCCombatArt struct {
	ArtID   string   `json:"artId"`
	Level   int      `json:"level"`
	MoveIDs []string `json:"moveIds"`
}

// CombatMoveGrowth 招式效果成长系数 S(L)=0.8+0.4L/(L+100)：
// S(1)≈0.804，S(20)≈0.867，S(100)=1，S(200)≈1.067，渐近上限1.2。
func CombatMoveGrowth(level int) float64 {
	if level < 1 {
		level = 1
	}
	return 0.8 + 0.4*float64(level)/(float64(level)+100)
}

// scaledEffectValue 返回随等级成长后的效果幅度。
func scaledEffectValue(value float64, level int) float64 { return value * CombatMoveGrowth(level) }

// EffectiveEffects 返回成长后的效果副本：Value/Cap/Power乘S；Charges/Ticks/DurationMs固定不成长。
func (m CombatMoveDef) EffectiveEffects(level int) []MoveEffect {
	if len(m.Effects) == 0 {
		return nil
	}
	s := CombatMoveGrowth(level)
	out := make([]MoveEffect, 0, len(m.Effects))
	for _, e := range m.Effects {
		e.Value = e.Value * s
		e.Cap = e.Cap * s
		e.Power = e.Power * s
		out = append(out, e)
	}
	return out
}

// EffectivePower 成长后的全动作伤害倍率（attack 招式）。
func (m CombatMoveDef) EffectivePower(level int) float64 {
	return m.Power * CombatMoveGrowth(level)
}

// EffectiveHitBonus 成长后的本招额外命中概率（小数，0.15=15个百分点）。
func (m CombatMoveDef) EffectiveHitBonus(level int) float64 {
	return m.HitBonusPP * CombatMoveGrowth(level)
}

// EffectiveArmorIgnore 成长后的忽略防御比例。
func (m CombatMoveDef) EffectiveArmorIgnore(level int) float64 {
	return m.ArmorIgnore * CombatMoveGrowth(level)
}

// ParryMultiplier 成长后的招架概率乘数：新倍率=1-削减幅度×S，下限0.3（即最多削减70%）。
// 不直接把"乘0.5"乘S，否则等级越高反而越容易被架住。
func (m CombatMoveDef) ParryMultiplier(level int) float64 {
	if m.ParryReduction <= 0 {
		return 1
	}
	return max(0.3, 1-m.ParryReduction*CombatMoveGrowth(level))
}

// CombatMoveView 单招在玩家视角下的实际数值与冷却窗口（get_combat_moves 逐招返回）。
type CombatMoveView struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	ArtID             string          `json:"artId"`
	Slot              string          `json:"slot"`
	Level             int             `json:"level"`
	Kind              string          `json:"kind"`
	Description       string          `json:"description,omitempty"`
	ActionMs          int64           `json:"actionMs"`
	CooldownSec       int             `json:"cooldownSec"`
	MPCost            int             `json:"mpCost"`
	HitCount          int             `json:"hitCount,omitempty"`
	Power             float64         `json:"power,omitempty"`
	HitBonusPP        float64         `json:"hitBonusPP,omitempty"`
	ArmorIgnore       float64         `json:"armorIgnore,omitempty"`
	ParryReduction    float64         `json:"parryReduction,omitempty"`
	Conditions        []MoveCondition `json:"conditions,omitempty"`
	Effects           []MoveEffect    `json:"effects,omitempty"`
	SkillStartedAt    int64           `json:"skillStartedAt,omitempty"`
	SkillReadyAt      int64           `json:"skillReadyAt,omitempty"`
	BlockedSince      int64           `json:"blockedSince,omitempty"`
	BlockedUntil      int64           `json:"blockedUntil,omitempty"`
	UnavailableReason string          `json:"unavailableReason,omitempty"`
}

// CombatMovesView get_combat_moves 的权威视图：所有时序字段由服务端给出。
type CombatMovesView struct {
	Capabilities         []string         `json:"capabilities"`
	ServerEpoch          string           `json:"serverEpoch"`
	Revision             int64            `json:"revision"`
	ServerTime           int64            `json:"serverTime"`
	BattleID             string           `json:"battleId,omitempty"`
	Sequence             int64            `json:"sequence"`
	MP                   int              `json:"mp"`
	EffectiveActionSpeed float64          `json:"effectiveActionSpeed"`
	ActionStartedAt      int64            `json:"actionStartedAt,omitempty"`
	ActionReadyAt        int64            `json:"actionReadyAt,omitempty"`
	ActionDurationMs     int64            `json:"actionDurationMs"`
	Moves                []CombatMoveView `json:"moves"`
}

// CombatMovesUpdate 战斗tick结果中的私有视图集合：只定向发给本人，不得混入公共房间payload。
type CombatMovesUpdate struct {
	PlayerID string          `json:"playerId"`
	View     CombatMovesView `json:"view"`
}

// CombatMoveResult cast_combat_move 回包：成功与失败都携带权威视图，避免普通error丢失状态。
type CombatMoveResult struct {
	View            *CombatMovesView `json:"view,omitempty"`
	Effects         CombatTickResult `json:"effects,omitempty"`
	Code            string           `json:"code,omitempty"`
	Message         string           `json:"message,omitempty"`
	AppliedSequence int64            `json:"appliedSequence,omitempty"`
}

// NPCMoveIntent NPC 公开预告的招式意图（公共房间可观察信息）：
// 这是AI提前透露攻击决定，不是施放读条；到公开时刻仍按即时技能结算。
type NPCMoveIntent struct {
	NPCID       string `json:"npcId"`
	MoveID      string `json:"moveId"`
	MoveName    string `json:"moveName"`
	TargetID    string `json:"targetId,omitempty"`
	AnnouncedAt int64  `json:"announcedAt"`
	ReleaseAt   int64  `json:"releaseAt"`
}

// scaledEffectValue 供包外引擎按等级取效果幅度。
func ScaledEffectValue(value float64, level int) float64 { return scaledEffectValue(value, level) }
