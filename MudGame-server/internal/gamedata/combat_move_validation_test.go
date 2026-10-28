package gamedata

import (
	"strings"
	"testing"

	"mudgame/server/internal/model"
)

func moveSkill(id string, moves ...model.CombatMoveDef) map[string]model.SkillDef {
	return map[string]model.SkillDef{id: {ID: id, Name: id, Moves: moves}}
}

func validMove(id string) model.CombatMoveDef {
	return model.CombatMoveDef{ID: id, Name: "招式", Kind: "attack", CooldownSec: 20, ActionMs: 4000, MPCost: 8, HitCount: 1, Power: 1.4}
}

// 84招整包目录与逐招边界（实现规范§10：38/84固定，CD10～60，MP正值，无等级解锁字段）。
func TestValidateCombatMovesAcceptsFullCatalog(t *testing.T) {
	skills := map[string]model.SkillDef{}
	for i := 0; i < 38; i++ {
		id := string(rune('a'+i%26)) + string(rune('a'+i/26)) + "_art"
		skills[id] = model.SkillDef{ID: id, Name: id, Moves: []model.CombatMoveDef{
			validMove(id + "/m01"), validMove(id + "/m02"), validMove(id + "/m03"),
		}}
	}
	// 38×3=114 超过84，只取84招的整包：构造84个唯一ID
	skills = map[string]model.SkillDef{}
	for i := 0; i < 28; i++ {
		id := "art" + string(rune('a'+i)) + "x"
		skills[id] = model.SkillDef{ID: id, Name: id, Moves: []model.CombatMoveDef{validMove(id + "/m01"), validMove(id + "/m02"), validMove(id + "/m03")}}
	}
	if err := ValidateCombatMoves(skills); err != nil {
		t.Fatalf("full catalog rejected: %v", err)
	}
}

func TestValidateCombatMovesRejectsBrokenConfigs(t *testing.T) {
	cases := map[string]map[string]model.SkillDef{
		"half catalog": moveSkill("art", validMove("art/m01")),
		"bad id":       moveSkill("art", validMove("other/m01")),
		"duplicate":    moveSkill("art", validMove("art/m01"), validMove("art/m01")),
	}
	bad := validMove("art/m01")
	bad.CooldownSec = 8
	cases["cd too short"] = moveSkill("art", bad)
	bad2 := validMove("art/m01")
	bad2.CooldownSec = 61
	cases["cd too long"] = moveSkill("art", bad2)
	bad3 := validMove("art/m01")
	bad3.MPCost = 0
	cases["free move"] = moveSkill("art", bad3)
	bad4 := validMove("art/m01")
	bad4.HitCount = 4
	cases["too many segments"] = moveSkill("art", bad4)
	bad5 := validMove("art/m01")
	bad5.Kind = "ultimate"
	cases["unknown kind"] = moveSkill("art", bad5)
	bad6 := validMove("art/m01")
	bad6.Effects = []model.MoveEffect{{Op: "self_destruct", Target: "self", Trigger: "cast"}}
	cases["unknown op"] = moveSkill("art", bad6)
	bad7 := validMove("art/m01")
	bad7.Effects = []model.MoveEffect{{Op: "guard", Target: "self", Trigger: "cast", Scope: "dot", Value: .3}}
	cases["bad scope for guard"] = moveSkill("art", bad7)
	bad8 := validMove("art/m01")
	bad8.ArmorIgnore = 0.6
	cases["ignore over cap"] = moveSkill("art", bad8)
	bad9 := validMove("art/m01")
	bad9.ParryReduction = 0.9
	cases["parry over cap"] = moveSkill("art", bad9)
	bad10 := validMove("art/m01")
	bad10.Effects = []model.MoveEffect{{Op: "stat_mod", Target: "self", Trigger: "cast", Stat: "luck", Value: 1}}
	cases["unknown stat"] = moveSkill("art", bad10)
	for name, skills := range cases {
		if err := ValidateCombatMoves(skills); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

// 无moves的旧资源整包可启动（向后兼容）。
func TestValidateCombatMovesAllowsLegacyResources(t *testing.T) {
	if err := ValidateCombatMoves(map[string]model.SkillDef{"art": {ID: "art", Name: "art"}}); err != nil {
		t.Fatalf("legacy resources must load: %v", err)
	}
}

// 惊雷趁隙示例（实现规范§2）：conditional_ignore 的 durationMs 是条件阈值。
func TestValidateCombatMovesAcceptsConditionalIgnoreExample(t *testing.T) {
	skills := map[string]model.SkillDef{}
	// 需整包84招，此处直接走单招校验路径：先通过整包包装
	for i := 0; i < 28; i++ {
		id := "art" + string(rune('a'+i)) + "x"
		skills[id] = model.SkillDef{ID: id, Name: id, Moves: []model.CombatMoveDef{validMove(id + "/m01"), validMove(id + "/m02"), validMove(id + "/m03")}}
	}
	spec := skills["artax"]
	spec.Moves[2] = model.CombatMoveDef{
		ID: "artax/m03", Name: "惊雷趁隙", Kind: "attack", CooldownSec: 35, ActionMs: 4000, MPCost: 18, HitCount: 1, Power: 1.4,
		Effects: []model.MoveEffect{{Op: "conditional_ignore", Target: "opponent", Trigger: "cast", Scope: "active_attack", Group: "armor_ignore", Value: .35, DurationMs: 1000, Tags: []string{"target_action_remaining_gte"}}},
	}
	skills["artax"] = spec
	if err := ValidateCombatMoves(skills); err != nil {
		t.Fatalf("spec example rejected: %v", strings.TrimSpace(err.Error()))
	}
}
