package gamedata

import (
	"fmt"
	"math"

	"mudgame/server/internal/model"
)

// ValidateCombatMoves 校验全部武功的 moves 配置（实现规范§2）：
// 无运行时eval，未知字段与未知枚举一律拒绝；容器沿用现有技能表，不建第二份。
// 无moves的旧资源整包可启动；带moves的发布包必须覆盖全部现有武功，不接受半份目录。
func ValidateCombatMoves(skills map[string]model.SkillDef) error {
	movesWithConfig := 0
	movesTotal := 0
	for sid, skill := range skills {
		if len(skill.Moves) == 0 {
			continue
		}
		movesWithConfig++
		if len(skill.Moves) > 3 {
			return fmt.Errorf("skill %s has %d moves, max 3", sid, len(skill.Moves))
		}
		seen := map[string]bool{}
		for i, m := range skill.Moves {
			wantID := fmt.Sprintf("%s/m%02d", sid, i+1)
			if m.ID != wantID {
				return fmt.Errorf("move %d of %s has id %q, want %q", i+1, sid, m.ID, wantID)
			}
			if seen[m.ID] {
				return fmt.Errorf("duplicate move id %s", m.ID)
			}
			seen[m.ID] = true
			movesTotal++
			if err := validateCombatMoveDef(m); err != nil {
				return fmt.Errorf("move %s: %w", m.ID, err)
			}
		}
	}
	if movesWithConfig == 0 {
		return nil
	}
	// 发布目录必须整包覆盖：每门现有武功都携带moves。
	for sid := range skills {
		if len(skills[sid].Moves) == 0 {
			return fmt.Errorf("release package must cover all skills with moves, missing %s", sid)
		}
	}
	if movesTotal != 84 {
		return fmt.Errorf("release package expects 84 moves, got %d", movesTotal)
	}
	return nil
}

var moveKinds = map[string]bool{"attack": true, "support": true, "debuff": true}

var moveEffectOps = map[string]bool{
	"heal": true, "hot": true, "next_active": true, "guard": true, "stat_mod": true,
	"outgoing_reduce": true, "vulnerability": true, "counter": true, "bleed": true,
	"lifesteal": true, "cleanse": true, "immunity": true, "mp_discount": true, "haste": true,
	"conditional_power": true, "conditional_ignore": true, "dispel_bonus": true,
	"strip_guard_charge": true, "paid_mp_refund": true, "delayed_damage_heal": true,
	"recent_damage_heal": true,
}

var moveTargets = map[string]bool{"self": true, "opponent": true}

var moveTriggers = map[string]bool{
	"cast": true, "hit": true, "incoming": true, "damaged": true, "missed": true, "next_tick": true,
}

var moveScopes = map[string]bool{
	"active_attack": true, "normal_or_active": true, "direct_incoming": true,
	"multi_active": true, "dot": true, "self": true,
}

var moveConditionKinds = map[string]bool{
	"self_hp_below": true, "target_hp_at_most": true, "recent_dodge": true,
	"recent_active_damage": true, "self_hp_cost": true,
}

var moveStats = map[string]bool{
	"damage": true, "hit_pp": true, "crit_pp": true, "armor_ignore": true, "guard_pierce": true,
	"attack": true, "defense": true, "hit": true, "dodge": true, "parry": true,
	"initiative": true, "vulnerability": true, "bleed": true, "haste": true,
}

func validateCombatMoveDef(m model.CombatMoveDef) error {
	if m.Name == "" {
		return fmt.Errorf("missing name")
	}
	if !moveKinds[m.Kind] {
		return fmt.Errorf("unknown kind %q", m.Kind)
	}
	if m.CooldownSec < 10 || m.CooldownSec > 60 {
		return fmt.Errorf("cooldown %d outside 10-60s", m.CooldownSec)
	}
	if m.ActionMs != 4000 {
		return fmt.Errorf("actionMs %d, this release fixes 4000", m.ActionMs)
	}
	if m.MPCost <= 0 {
		return fmt.Errorf("mpCost %d must be positive", m.MPCost)
	}
	for _, v := range []float64{m.Power, m.ArmorIgnore, m.ParryReduction} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return fmt.Errorf("invalid scalar")
		}
	}
	// 命中加成允许负值（乱披风等高伤不稳定招的自损），界内即可
	if math.IsNaN(m.HitBonusPP) || math.IsInf(m.HitBonusPP, 0) || m.HitBonusPP < -0.25 || m.HitBonusPP > 0.25 {
		return fmt.Errorf("hit bonus %v outside [-25pp, +25pp]", m.HitBonusPP)
	}
	switch m.Kind {
	case "attack":
		if m.HitCount < 1 || m.HitCount > 3 {
			return fmt.Errorf("attack hitCount %d outside 1-3", m.HitCount)
		}
		if m.Power <= 0 {
			return fmt.Errorf("attack power missing")
		}
	default:
		if m.HitCount != 0 || m.Power != 0 {
			return fmt.Errorf("support/debuff must have zero hitCount/power")
		}
	}
	if m.ArmorIgnore > 0.5 {
		return fmt.Errorf("armor ignore %v exceeds 50%%", m.ArmorIgnore)
	}
	if m.ParryReduction > 0.7 {
		return fmt.Errorf("parry reduction %v exceeds 70%%", m.ParryReduction)
	}
	for _, c := range m.Conditions {
		if !moveConditionKinds[c.Kind] {
			return fmt.Errorf("unknown condition %q", c.Kind)
		}
		if math.IsNaN(c.Value) || math.IsInf(c.Value, 0) {
			return fmt.Errorf("invalid condition value")
		}
	}
	for _, e := range m.Effects {
		if err := validateMoveEffect(m, e); err != nil {
			return err
		}
	}
	return nil
}

// effectOpRules 每个Op允许的 Target/Trigger/Scope 组合白名单（实现规范§2表）。
var effectOpRules = map[string]struct{ targets, triggers, scopes []string }{
	"heal":                {[]string{"self"}, []string{"cast"}, nil},
	"hot":                 {[]string{"self"}, []string{"cast"}, nil},
	"next_active":         {[]string{"self"}, []string{"cast"}, []string{"active_attack"}},
	"guard":               {[]string{"self"}, []string{"cast"}, []string{"direct_incoming", "multi_active"}},
	"stat_mod":            {[]string{"self", "opponent"}, []string{"cast", "hit"}, nil},
	"outgoing_reduce":     {[]string{"opponent"}, []string{"cast", "hit"}, []string{"normal_or_active"}},
	"vulnerability":       {[]string{"self"}, []string{"cast"}, []string{"direct_incoming"}},
	"counter":             {[]string{"self"}, []string{"incoming"}, []string{"normal_or_active"}},
	"bleed":               {[]string{"opponent"}, []string{"hit"}, nil},
	"lifesteal":           {[]string{"self"}, []string{"hit"}, nil},
	"cleanse":             {[]string{"self"}, []string{"cast"}, nil},
	"immunity":            {[]string{"self"}, []string{"cast"}, nil},
	"mp_discount":         {[]string{"self"}, []string{"cast"}, nil},
	"haste":               {[]string{"self"}, []string{"cast"}, nil},
	"conditional_power":   {[]string{"self"}, []string{"cast"}, nil},
	"conditional_ignore":  {[]string{"opponent"}, []string{"cast"}, nil},
	"dispel_bonus":        {[]string{"opponent"}, []string{"hit"}, nil},
	"strip_guard_charge":  {[]string{"opponent"}, []string{"hit"}, nil},
	"paid_mp_refund":      {[]string{"self"}, []string{"damaged"}, []string{"normal_or_active"}},
	"delayed_damage_heal": {[]string{"self"}, []string{"damaged"}, []string{"active_attack"}},
	"recent_damage_heal":  {[]string{"self"}, []string{"cast"}, nil},
}

func validateMoveEffect(m model.CombatMoveDef, e model.MoveEffect) error {
	if !moveEffectOps[e.Op] {
		return fmt.Errorf("unknown op %q", e.Op)
	}
	if e.Target != "" && !moveTargets[e.Target] {
		return fmt.Errorf("unknown target %q", e.Target)
	}
	if e.Trigger != "" && !moveTriggers[e.Trigger] {
		return fmt.Errorf("unknown trigger %q", e.Trigger)
	}
	if e.Scope != "" && !moveScopes[e.Scope] {
		return fmt.Errorf("unknown scope %q", e.Scope)
	}
	if e.Stat != "" && !moveStats[e.Stat] {
		return fmt.Errorf("unknown stat %q", e.Stat)
	}
	rules, ok := effectOpRules[e.Op]
	if ok {
		if len(rules.targets) > 0 && !contains(rules.targets, e.Target) {
			return fmt.Errorf("op %s forbids target %q", e.Op, e.Target)
		}
		if len(rules.triggers) > 0 && !contains(rules.triggers, e.Trigger) {
			return fmt.Errorf("op %s forbids trigger %q", e.Op, e.Trigger)
		}
		if len(rules.scopes) > 0 && !contains(rules.scopes, e.Scope) {
			return fmt.Errorf("op %s forbids scope %q", e.Op, e.Scope)
		}
	}
	for _, v := range []float64{e.Value, e.Cap, e.Power} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return fmt.Errorf("op %s has invalid scalar", e.Op)
		}
	}
	if e.Charges < 0 || e.Ticks < 0 || e.DurationMs < 0 {
		return fmt.Errorf("op %s has negative charges/ticks/duration", e.Op)
	}
	return nil
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
