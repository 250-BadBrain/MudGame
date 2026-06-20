package manager

import (
	"strings"
	"testing"
)

func setupLearningManager(t *testing.T) (*RuntimeManager, string) {
	t.Helper()
	r := NewRuntimeManager(t.TempDir())
	if err := r.Register("learner", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("learner", "学徒", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	return r, playerID
}

func prepareTeacherCapPlayer(t *testing.T, r *RuntimeManager, playerID string, level int) {
	t.Helper()
	players, err := r.loadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	p := players[playerID]
	p.RoomID = "ly_weapon_school"
	p.Attributes.Experience = 5488000
	p.Attributes.Potential = 500
	p.Skills["basic_unarmed"] = level
	if p.SkillLearningProgress == nil {
		p.SkillLearningProgress = map[string]int{}
	}
	if err := r.savePlayers(players); err != nil {
		t.Fatalf("save player failed: %v", err)
	}
}

func TestLearnSkillRejectsTeacherLevelCap(t *testing.T) {
	r, playerID := setupLearningManager(t)
	prepareTeacherCapPlayer(t, r, playerID, 300)

	err := r.LearnSkill(playerID, "weapon_school_master", "basic_unarmed")
	if err == nil || err.Error() != "你的基本拳脚已经不输你师傅了" {
		t.Fatalf("expected teacher cap message, got %v", err)
	}
}

func TestLearningStopsWhenTeacherLevelCapIsReached(t *testing.T) {
	r, playerID := setupLearningManager(t)
	prepareTeacherCapPlayer(t, r, playerID, 299)

	players, err := r.loadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	players[playerID].SkillLearningProgress["basic_unarmed"] = 200
	if err := r.savePlayers(players); err != nil {
		t.Fatalf("save learning progress failed: %v", err)
	}
	if err := r.LearnSkill(playerID, "weapon_school_master", "basic_unarmed"); err != nil {
		t.Fatalf("start learning failed: %v", err)
	}

	var message string
	for i := 0; i < stateSettlementSeconds; i++ {
		result, err := r.ProcessStatusTick()
		if err != nil {
			t.Fatalf("process learning tick failed: %v", err)
		}
		if len(result.PlayerMessages) > 0 {
			message = result.PlayerMessages[0].Message
		}
	}
	players, err = r.loadPlayers()
	if err != nil {
		t.Fatalf("reload players failed: %v", err)
	}
	p := players[playerID]
	if p.Skills["basic_unarmed"] != 300 || p.State != "IDLE" {
		t.Fatalf("expected learning to stop at teacher cap, level=%d state=%s", p.Skills["basic_unarmed"], p.State)
	}
	if !strings.Contains(message, "你的基本拳脚已经不输你师傅了") {
		t.Fatalf("expected teacher cap feedback, got %q", message)
	}
}
