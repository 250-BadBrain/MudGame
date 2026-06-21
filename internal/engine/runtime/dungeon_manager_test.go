package runtime

import (
	"reflect"
	"testing"

	"mudgame/server/internal/model"
)

func TestXunqinFirstSixTemplatesMatchRedesign(t *testing.T) {
	m := NewManager(map[string]model.NPC{}, map[string]Template{}, map[string]*Instance{}, map[string]string{}, map[string]string{}, map[string]*NPCState{})
	if err := m.LoadTemplatesNoLock(); err != nil {
		t.Fatalf("load dungeon templates failed: %v", err)
	}
	expected := map[string]int{
		"xunqin_ch01_bronze_fall":         12,
		"xunqin_ch02_wu_fort_trial":       10,
		"xunqin_ch03_handan_opening":      12,
		"xunqin_ch04_hostage_camp_night":  11,
		"xunqin_ch05_handan_blood_road":   12,
		"xunqin_ch06_wu_fort_blood_night": 14,
	}
	chapters := map[int]string{}
	for _, id := range m.DungeonTemplateIDsNoLock() {
		tpl, _ := m.DungeonTemplateNoLock(id)
		if tpl.ChainID == "xunqin_legend" && tpl.Chapter >= 1 && tpl.Chapter <= 6 {
			if _, ok := expected[id]; !ok {
				t.Fatalf("obsolete first-six xunqin dungeon is still loaded: %s", id)
			}
		}
	}
	for id, roomCount := range expected {
		tpl, ok := m.DungeonTemplateNoLock(id)
		if !ok {
			t.Fatalf("missing redesigned dungeon %s", id)
		}
		if len(tpl.Rooms) != roomCount {
			t.Fatalf("dungeon %s expected %d rooms, got %d", id, roomCount, len(tpl.Rooms))
		}
		if tpl.ChainID != "xunqin_legend" || tpl.Chapter < 1 || tpl.Chapter > 6 {
			t.Fatalf("dungeon %s has invalid chain metadata: %+v", id, tpl)
		}
		chapters[tpl.Chapter] = id
	}
	if !reflect.DeepEqual(chapters, map[int]string{1: "xunqin_ch01_bronze_fall", 2: "xunqin_ch02_wu_fort_trial", 3: "xunqin_ch03_handan_opening", 4: "xunqin_ch04_hostage_camp_night", 5: "xunqin_ch05_handan_blood_road", 6: "xunqin_ch06_wu_fort_blood_night"}) {
		t.Fatalf("unexpected first-six chapter ordering: %+v", chapters)
	}
}

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

func TestObjectiveProgressSupportsOptionalAndExclusiveBranches(t *testing.T) {
	m := NewManager(nil, nil, nil, nil, nil, nil)
	inst := &Instance{
		Objectives: []Objective{
			{ID: "clue", Required: 1},
			{ID: "stealth_entry", Required: 1, ExclusiveGroup: "entry"},
			{ID: "combat_entry", Required: 1, ExclusiveGroup: "entry"},
			{ID: "bonus", Required: 1, Optional: true},
		},
		ObjectiveProgress: map[string]int{"clue": 1, "stealth_entry": 1},
	}
	m.updateObjectiveProgressNoLock(inst)
	if inst.Progress != 100 {
		t.Fatalf("expected one exclusive branch plus required objective to complete dungeon, got %d", inst.Progress)
	}
}

func TestObjectiveProgressUsesWeightsAndOneExclusiveBranch(t *testing.T) {
	m := NewManager(nil, nil, nil, nil, nil, nil)
	inst := &Instance{
		Objectives: []Objective{
			{ID: "story", Required: 1, Weight: 70},
			{ID: "front_gate", Required: 1, Weight: 30, ExclusiveGroup: "route"},
			{ID: "secret_tunnel", Required: 1, Weight: 30, ExclusiveGroup: "route"},
		},
		ObjectiveProgress: map[string]int{"story": 1, "secret_tunnel": 1},
	}
	m.updateObjectiveProgressNoLock(inst)
	if inst.Progress != 100 {
		t.Fatalf("expected weighted story plus one route to complete dungeon, got %d", inst.Progress)
	}
}

func TestDefeatModeNPCRemainsInRoomWithoutBeingKillableAgain(t *testing.T) {
	npc := &NPCState{
		ID:         "trial_master@instance",
		TemplateID: "trial_master",
		RoomID:     "trial_room@instance",
		Alive:      true,
		Aggressive: true,
		CombatMode: "defeat",
		Attributes: model.Attributes{Resources: model.ResourceAttributes{HP: 40, MaxHP: 40}},
	}
	inst := &Instance{
		ID:                "instance",
		Rooms:             map[string]model.Room{npc.RoomID: {ID: npc.RoomID, NPCIDs: []string{npc.ID}}},
		NPCs:              map[string]*NPCState{npc.ID: npc},
		Objectives:        []Objective{{ID: "win_trial", Type: "kill", TargetID: "trial_master", Required: 1}},
		ObjectiveProgress: map[string]int{},
	}
	m := NewManager(nil, nil, map[string]*Instance{inst.ID: inst}, nil, map[string]string{npc.RoomID: inst.ID}, map[string]*NPCState{npc.ID: npc})
	_, defeated, _, ok := m.DefeatNPCNoLock(npc.ID)
	if !ok || !defeated.Alive || !defeated.Defeated || defeated.Aggressive {
		t.Fatalf("expected defeated npc to remain peacefully in room, got %+v", defeated)
	}
	if defeated.Attributes.Resources.HP != 10 {
		t.Fatalf("expected defeated npc to retain quarter health, got %d", defeated.Attributes.Resources.HP)
	}
}

func TestAdvanceStealthMovesPatrolAndChangesAlertByPlayerTurn(t *testing.T) {
	inst := &Instance{
		ID:           "instance",
		Stealth:      StealthConfig{Enabled: true, AlertOnMove: 1, SafeRoomRecovery: 2, MaxAlert: 6},
		Rooms:        map[string]model.Room{"a": {ID: "a", NPCIDs: []string{"guard"}}, "b": {ID: "b"}, "safe": {ID: "safe", StealthSafe: true}},
		NPCs:         map[string]*NPCState{"guard": {ID: "guard", RoomID: "a", Alive: true}},
		PatrolRoutes: map[string][]string{"guard": {"a", "b"}},
	}
	m := NewManager(nil, nil, map[string]*Instance{"instance": inst}, nil, map[string]string{"a": "instance", "b": "instance", "safe": "instance"}, nil)
	m.AdvanceStealthNoLock("b")
	if inst.NPCs["guard"].RoomID != "b" || inst.AlertLevel != 2 {
		t.Fatalf("expected patrol encounter to move guard and raise alert to 2, got room=%s alert=%d", inst.NPCs["guard"].RoomID, inst.AlertLevel)
	}
	m.AdvanceStealthNoLock("safe")
	if inst.AlertLevel != 0 {
		t.Fatalf("expected safe room to recover alert, got %d", inst.AlertLevel)
	}
}
