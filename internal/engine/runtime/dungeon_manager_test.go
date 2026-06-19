package runtime

import (
	"testing"

	"mudgame/server/internal/model"
)

func TestBuildDungeonSettlementUsesConfiguredRewards(t *testing.T) {
	m := NewManager(nil, nil, nil, nil, nil, nil)
	inst := &Instance{
		DungeonID: "test_dungeon",
		Name:      "测试副本",
		Sequence:  3,
		ChainID:   "test_chain",
		Chapter:   2,
		Progress:  100,
		Rewards: model.Rewards{
			Experience: 180,
			Potential:  120,
			Money:      60,
			Items: map[string]int{
				"sealed_letter": 1,
			},
		},
	}

	settlement := m.BuildDungeonSettlementNoLock(inst)
	if settlement.RewardExperience != 180 {
		t.Fatalf("expected configured experience reward, got %d", settlement.RewardExperience)
	}
	if settlement.RewardPotential != 120 {
		t.Fatalf("expected configured potential reward, got %d", settlement.RewardPotential)
	}
	if settlement.RewardMoney != 60 {
		t.Fatalf("expected configured money reward, got %d", settlement.RewardMoney)
	}
	if settlement.RewardItems["sealed_letter"] != 1 {
		t.Fatalf("expected sealed_letter reward, got %+v", settlement.RewardItems)
	}
	if settlement.ChainID != "test_chain" || settlement.Chapter != 2 {
		t.Fatalf("expected chain metadata, got chain=%s chapter=%d", settlement.ChainID, settlement.Chapter)
	}
}

func TestBuildDungeonSettlementProratesRewardsAndWithholdsItems(t *testing.T) {
	m := NewManager(nil, nil, nil, nil, nil, nil)
	inst := &Instance{
		DungeonID: "test_dungeon",
		Name:      "测试副本",
		Sequence:  1,
		Progress:  50,
		Rewards: model.Rewards{
			Experience: 180,
			Potential:  120,
			Money:      60,
			Items: map[string]int{
				"sealed_letter": 1,
			},
		},
	}

	settlement := m.BuildDungeonSettlementNoLock(inst)
	if settlement.RewardExperience != 90 || settlement.RewardPotential != 60 || settlement.RewardMoney != 30 {
		t.Fatalf("expected prorated rewards, got exp=%d potential=%d money=%d", settlement.RewardExperience, settlement.RewardPotential, settlement.RewardMoney)
	}
	if len(settlement.RewardItems) != 0 {
		t.Fatalf("expected no item reward before full completion, got %+v", settlement.RewardItems)
	}
}
