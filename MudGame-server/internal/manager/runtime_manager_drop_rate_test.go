package manager

import (
	"testing"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/model"
)

func TestDropRateLogic(t *testing.T) {
	s := engruntime.Settlement{
		DungeonID:        "test_dungeon",
		DungeonName:      "Test",
		Progress:         100,
		RewardExperience: 100,
		RewardPotential:  50,
		RewardMoney:      200,
		RewardItems:      map[string]int{"sealed_letter": 1, "iron_sword": 1},
		RewardFlags:      map[string]bool{},
	}
	p := &model.Player{
		ID: "test", Backpack: map[string]int{}, Flags: map[string]bool{}, Skills: map[string]int{},
	}
	result := applyDungeonSettlementReward(p, s)
	if !result.IsFirstClear {
		t.Fatal("should be first clear")
	}
	if p.Backpack["sealed_letter"] < 1 {
		t.Fatal("quest item must be guaranteed")
	}
	if !p.Flags["dungeon_cleared_test_dungeon"] {
		t.Fatal("flag missing")
	}
	t.Logf("first clear: iron_sword=%d (40%% chance)", p.Backpack["iron_sword"])

	p.Backpack["bronze_fragment"] = 0
	result2 := applyDungeonSettlementReward(p, s)
	if result2.IsFirstClear {
		t.Fatal("should not be first clear")
	}
	if p.Backpack["sealed_letter"] > 1 {
		t.Fatalf("FirstClaim dup: %d", p.Backpack["sealed_letter"])
	}
	if p.Backpack["bronze_fragment"] < 2 {
		t.Fatalf("bronze bonus missing: %d", p.Backpack["bronze_fragment"])
	}
}
