package manager

import (
 "strings"
	"testing"
)

// 进阶武学链路：副本残页 → use_item 参悟入门 → 暗巷传承人传授升级
// （见 docs/design/economy_design.md 1.3.1）
func setupAdvancedSkillPlayer(t *testing.T) (*RuntimeManager, string) {
	t.Helper()
	r := NewRuntimeManager(t.TempDir())
	if err := r.Register("heir", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("heir", "传承者", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	players, err := r.loadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	p := players[playerID]
	p.RoomID = "ly_weapon_school"
	p.Attributes.Potential = 500
	// 秘籍前置门槛：长生诀需基本内功达到 20 级
	p.Skills["basic_neigong"] = 120
	p.Skills["basic_literacy"] = 80
	p.Flags["dungeon_cleared_datang_xinggong"] = true
	p.Attributes.Experience = 200000
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	p.Backpack["manual_csj_changshengjue"] = 1
	if err := r.savePlayers(players); err != nil {
		t.Fatalf("save players failed: %v", err)
	}
	return r, playerID
}

func TestAdvancedSkillRequiresInsightBeforeTeaching(t *testing.T) {
	r, playerID := setupAdvancedSkillPlayer(t)

	// 未参悟：传承人拒绝传授
	err := r.LearnSkill(playerID, "ancient_scroll_keeper", "csj_changshengjue")
	if err == nil {
		t.Fatalf("expected insight requirement error, got %v", err)
	}
}

func TestInsightPageGrantsAdvancedSkill(t *testing.T) {
	r, playerID := setupAdvancedSkillPlayer(t)

	msg, err := r.UseItem(playerID, "manual_csj_changshengjue")
	if err != nil {
		t.Fatalf("insight failed: %v", err)
	}
	if !strings.Contains(msg, "长生诀") {
		t.Fatalf("insight message should mention the skill, got %s", msg)
	}

	players, err := r.loadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	p := players[playerID]
	if p.Skills["csj_changshengjue"] != 1 {
		t.Fatalf("expected skill at level 1 after insight, got %d", p.Skills["csj_changshengjue"])
	}
	if p.Backpack["manual_csj_changshengjue"] != 0 {
		t.Fatalf("page should be consumed, remaining %d", p.Backpack["manual_csj_changshengjue"])
	}

	// 重复参悟被拒
	if _, err := r.UseItem(playerID, "manual_csj_changshengjue"); err == nil {
		t.Fatalf("second insight should fail")
	}
}

func TestAdvancedSkillTeachingAfterInsight(t *testing.T) {
	r, playerID := setupAdvancedSkillPlayer(t)

	if _, err := r.UseItem(playerID, "manual_csj_changshengjue"); err != nil {
		t.Fatalf("insight failed: %v", err)
	}

	// 参悟后可向传承人学习
	if err := r.LearnSkill(playerID, "ancient_scroll_keeper", "csj_changshengjue"); err != nil {
		t.Fatalf("learn after insight failed: %v", err)
	}

	players, err := r.loadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	p := players[playerID]
	if p.State != "LEARNING" || p.LearningSkillID != "csj_changshengjue" {
		t.Fatalf("expected learning state for advanced skill, state=%s skill=%s", p.State, p.LearningSkillID)
	}
	if p.LearningTeacherLevel != 0 {
		t.Fatalf("self practice should not retain a teacher cap, got %d", p.LearningTeacherLevel)
	}
}
