package runtime

import (
	"reflect"
	"testing"

	"mudgame/server/internal/model"
)

func TestXunqinRewrittenFirstSixTemplatesMatchDesign(t *testing.T) {
	m := NewManager(map[string]model.NPC{}, map[string]Template{}, map[string]*Instance{}, map[string]string{}, map[string]string{}, map[string]*NPCState{})
	if err := m.LoadTemplatesNoLock(); err != nil {
		t.Fatalf("load dungeon templates failed: %v", err)
	}
	expected := map[string]int{
		"xunqin_ch01_bronze_tomb":  6,
		"xunqin_ch02_wild_convoy":  8,
		"xunqin_ch03_wu_stables":   8,
		"xunqin_ch04_zhao_inn":     8,
		"xunqin_ch05_hostage_camp": 9,
		"xunqin_ch06_handan_alley": 9,
	}
	chapters := map[int]string{}
	for _, id := range m.DungeonTemplateIDsNoLock() {
		tpl, _ := m.DungeonTemplateNoLock(id)
		if tpl.ChainID == "xunqin_legend" {
			if _, ok := expected[id]; !ok {
				t.Fatalf("unexpected xunqin dungeon remains loaded: %s", id)
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
	if !reflect.DeepEqual(chapters, map[int]string{1: "xunqin_ch01_bronze_tomb", 2: "xunqin_ch02_wild_convoy", 3: "xunqin_ch03_wu_stables", 4: "xunqin_ch04_zhao_inn", 5: "xunqin_ch05_hostage_camp", 6: "xunqin_ch06_handan_alley"}) {
		t.Fatalf("unexpected rewritten chapter ordering: %+v", chapters)
	}
}

func TestXunqinPuzzleStateGraphsClassifyEveryConfiguredState(t *testing.T) {
	m := NewManager(map[string]model.NPC{}, map[string]Template{}, map[string]*Instance{}, map[string]string{}, map[string]string{}, map[string]*NPCState{})
	if err := m.LoadTemplatesNoLock(); err != nil {
		t.Fatalf("load dungeon templates failed: %v", err)
	}
	for _, id := range m.DungeonTemplateIDsNoLock() {
		tpl, _ := m.DungeonTemplateNoLock(id)
		if tpl.ChainID != "xunqin_legend" {
			continue
		}
		inst := &Instance{DungeonID: id, PuzzleStates: map[string]string{}}
		choiceActions := map[string]bool{}
		for _, parent := range tpl.PuzzleActions {
			for _, choice := range parent.Choices {
				choiceActions[choice.ID] = true
			}
		}
		for roomID, room := range tpl.Rooms {
			for _, actionID := range room.Actions {
				if choiceActions[actionID] {
					t.Fatalf("dungeon %s room %s exposes message choice %s as a room action", id, roomID, actionID)
				}
			}
		}
		for puzzleID, initial := range tpl.PuzzleInitialStates {
			inst.PuzzleStates[puzzleID] = initial
			reachable := map[string]bool{initial: true}
			changed := true
			for changed {
				changed = false
				for _, action := range tpl.PuzzleActions {
					if action.PuzzleID != puzzleID || action.NextState == "" {
						continue
					}
					for _, from := range action.FromStates {
						if from != "never" && reachable[from] && !reachable[action.NextState] {
							reachable[action.NextState] = true
							changed = true
						}
					}
				}
			}
			for actionID, action := range tpl.PuzzleActions {
				if action.PuzzleID != puzzleID {
					continue
				}
				for _, from := range action.FromStates {
					if from != "never" && !reachable[from] {
						t.Fatalf("dungeon %s action %s starts from unreachable %s state %s", id, actionID, puzzleID, from)
					}
				}
			}
		}
		totalWeight := 0
		groupWeights := map[string]int{}
		for _, objective := range tpl.Objectives {
			if objective.Optional {
				continue
			}
			if objective.ExclusiveGroup == "" {
				totalWeight += objective.Weight
			} else if objective.Weight > groupWeights[objective.ExclusiveGroup] {
				groupWeights[objective.ExclusiveGroup] = objective.Weight
			}
		}
		for _, weight := range groupWeights {
			totalWeight += weight
		}
		if totalWeight != 100 {
			t.Fatalf("dungeon %s objective weights total %d, want 100", id, totalWeight)
		}
		for actionID, action := range tpl.PuzzleActions {
			if _, ok := tpl.PuzzleInitialStates[action.PuzzleID]; !ok {
				t.Fatalf("dungeon %s action %s uses unknown puzzle %s", id, actionID, action.PuzzleID)
			}
			for _, from := range action.FromStates {
				inst.PuzzleStates[action.PuzzleID] = from
				if status := m.PuzzleActionStateStatusNoLock(inst, action); status != PuzzleActionReady {
					t.Fatalf("dungeon %s action %s should be ready from %s, got %s", id, actionID, from, status)
				}
			}
			if action.NextState != "" && !testContainsString(action.FromStates, action.NextState) {
				inst.PuzzleStates[action.PuzzleID] = action.NextState
				if status := m.PuzzleActionStateStatusNoLock(inst, action); status != PuzzleActionRepeat {
					t.Fatalf("dungeon %s action %s should repeat at %s, got %s", id, actionID, action.NextState, status)
				}
			}
			for _, choice := range action.Choices {
				child, ok := tpl.PuzzleActions[choice.ID]
				if !ok || !testContainsString(child.FromStates, action.NextState) && !testContainsString(child.FromStates, "never") {
					t.Fatalf("dungeon %s action %s choice %s does not consume offered state %s", id, actionID, choice.ID, action.NextState)
				}
			}
		}
	}
}

func testContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestXunqinPendingChoicesKeepTheirExactPrompt(t *testing.T) {
	m := NewManager(map[string]model.NPC{}, map[string]Template{}, map[string]*Instance{}, map[string]string{}, map[string]string{}, map[string]*NPCState{})
	if err := m.LoadTemplatesNoLock(); err != nil {
		t.Fatalf("load dungeon templates failed: %v", err)
	}
	for _, id := range m.DungeonTemplateIDsNoLock() {
		tpl, _ := m.DungeonTemplateNoLock(id)
		if tpl.ChainID != "xunqin_legend" || tpl.Chapter > 6 {
			continue
		}
		for actionID, action := range tpl.PuzzleActions {
			if len(action.Choices) == 0 {
				continue
			}
			pending := map[string]bool{}
			for _, choice := range action.Choices {
				pending[choice.ID] = true
			}
			inst := &Instance{
				DungeonID:            id,
				PendingChoices:       pending,
				PendingChoicePrompt:  action.Message,
				PendingChoiceOptions: append([]ActionChoice{}, action.Choices...),
			}
			if got := m.PendingChoicePromptNoLock(inst); got != action.Message {
				t.Fatalf("dungeon %s action %s lost its pending prompt: got %q want %q", id, actionID, got, action.Message)
			}
			if got := m.PendingActionChoicesNoLock(inst); !reflect.DeepEqual(got, action.Choices) {
				t.Fatalf("dungeon %s action %s exposed internal action labels: got %+v want %+v", id, actionID, got, action.Choices)
			}
		}
	}
}

func TestXunqinDungeonDialogueScriptsCoverEveryStoryAction(t *testing.T) {
	m := NewManager(map[string]model.NPC{}, map[string]Template{}, map[string]*Instance{}, map[string]string{}, map[string]string{}, map[string]*NPCState{})
	if err := m.LoadTemplatesNoLock(); err != nil {
		t.Fatalf("load dungeon templates failed: %v", err)
	}
	covered := 0
	for _, id := range m.DungeonTemplateIDsNoLock() {
		tpl, _ := m.DungeonTemplateNoLock(id)
		if tpl.ChainID != "xunqin_legend" || tpl.Chapter > 6 {
			continue
		}
		for actionID, action := range tpl.PuzzleActions {
			if action.DialogueNPCID == "" {
				continue
			}
			covered++
			if len(action.DialogueLines) < 2 {
				t.Fatalf("dungeon %s action %s needs at least two ordered dialogue lines", id, actionID)
			}
			if action.DialogueDoneMessage == "" {
				t.Fatalf("dungeon %s action %s needs a contextual completion response", id, actionID)
			}
		}
	}
	if covered != 15 {
		t.Fatalf("expected all 15 rewritten story dialogue actions, got %d", covered)
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

func TestPuzzleActionNPCEffectsRemoveOrLeaveCorpse(t *testing.T) {
	roomID := "room@instance"
	removed := &NPCState{ID: "wounded", TemplateID: "wounded", RoomID: roomID, Alive: true}
	killed := &NPCState{ID: "enemy", TemplateID: "enemy", RoomID: roomID, Alive: true, Aggressive: true, Attributes: model.Attributes{Resources: model.ResourceAttributes{HP: 20, MaxHP: 20}}}
	inst := &Instance{
		ID:    "instance",
		Rooms: map[string]model.Room{roomID: {ID: roomID, NPCIDs: []string{removed.ID, killed.ID}}},
		NPCs:  map[string]*NPCState{removed.ID: removed, killed.ID: killed},
	}
	m := NewManager(nil, nil, map[string]*Instance{inst.ID: inst}, nil, map[string]string{roomID: inst.ID}, map[string]*NPCState{removed.ID: removed, killed.ID: killed})
	m.ApplyActionNPCEffectsNoLock(inst, roomID, []string{"wounded"}, []string{"enemy"}, nil)
	if _, exists := inst.NPCs[removed.ID]; exists {
		t.Fatalf("expected removed npc deleted from instance")
	}
	if killed.Alive || killed.Attributes.Resources.HP != 0 || killed.Aggressive {
		t.Fatalf("expected killed npc to remain as non-hostile corpse, got %+v", killed)
	}
	room := inst.Rooms[roomID]
	if len(room.NPCIDs) != 1 || room.NPCIDs[0] != killed.ID {
		t.Fatalf("expected only corpse to remain in room, got %+v", room.NPCIDs)
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
