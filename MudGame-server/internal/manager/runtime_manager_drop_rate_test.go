package manager

import (
	"testing"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/model"
)

// 掉落率系统核心逻辑：直接测试 applyDungeonSettlementReward
func TestDropRateLogic(t *testing.T) {
	// 构造一个 100% 结算
	s := engruntime.Settlement{
		DungeonID:        "test_dungeon",
		DungeonName:      "测试副本",
		Progress:         100,
		RewardExperience: 100,
		RewardPotential:  50,
		RewardMoney:      200,
		RewardItems:      map[string]int{"sealed_letter": 1, "iron_sword": 1},
		RewardFlags:      map[string]bool{},
	}

	p := &model.Player{
		ID:       "test_player",
		Backpack: map[string]int{},
		Flags:    map[string]bool{},
		Skills:   map[string]int{},
	}

	// 首通
	result := applyDungeonSettlementReward(p, s)
	if !result.IsFirstClear {
		t.Fatal("first call should be first clear")
	}
	if p.Backpack["sealed_letter"] < 1 {
		t.Fatal("first clear should give sealed_letter")
	}
	if !p.Flags["dungeon_cleared_test_dungeon"] {
		t.Fatal("first clear should set dungeon_cleared flag")
	}

	// 复刷（同一个玩家，同一个副本）
	p.Backpack["bronze_fragment"] = 0
	result2 := applyDungeonSettlementReward(p, s)
	if result2.IsFirstClear {
		t.Fatal("second call should not be first clear")
	}

	// FirstClaim 物品不应重复
	if p.Backpack["sealed_letter"] > 1 {
		t.Fatalf("FirstClaim item should not duplicate, got %d", p.Backpack["sealed_letter"])
	}

	// 复刷保底铜片
	if p.Backpack["bronze_fragment"] < 2 {
		t.Fatalf("repeat clear should give bronze_fragment >= 2, got %d", p.Backpack["bronze_fragment"])
	}
}
