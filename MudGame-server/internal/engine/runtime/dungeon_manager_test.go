package runtime

import (
	"strings"
	"testing"

	"mudgame/server/internal/model"
)

func TestHuangyiChapterTemplatesMatchDesign(t *testing.T) {
	m := NewManager(map[string]model.NPC{}, map[string]Template{}, map[string]*Instance{}, map[string]string{}, map[string]string{}, map[string]*NPCState{})
	if err := m.LoadTemplatesNoLock(); err != nil {
		t.Fatalf("load dungeon templates failed: %v", err)
	}
	expected := map[string]struct {
		rooms     int
		chainID   string
		chainName string
		chapter   int
	}{
		"abandoned_temple_rain": {3, "luoyang_rain_temple", "洛阳轶事", 1},
		"datang_changshengjue":  {5, "datang_double_dragon", "大唐双龙传", 1},
		"fuyu_nujiao":           {5, "fuyu_fan_yun", "覆雨翻云", 1},
		"posui_jingyan":         {5, "posui_xukong", "破碎虚空", 1},
		"bianhuang_escort":      {5, "bianhuang_legend", "边荒传说", 1},
		"datang_yanggong":       {6, "datang_double_dragon", "大唐双龙传", 2},
		"fuyu_lanjiang":         {3, "fuyu_fan_yun", "覆雨翻云", 2},
		"datang_jingling":       {7, "datang_double_dragon", "大唐双龙传", 3},
		"datang_tiandao":        {5, "datang_double_dragon", "大唐双龙传", 4},
		"liaoshui_lair":         {3, "luoshui_raiders", "洛水风浪", 1},
		"datang_feima":          {5, "datang_double_dragon", "大唐双龙传", 5},
		"datang_changan":        {5, "datang_double_dragon", "大唐双龙传", 6},
		"posui_jiming":          {5, "posui_xukong", "破碎虚空", 2},
		"posui_zhanshen":        {4, "posui_xukong", "破碎虚空", 3},
		"bianhuang_wangting":    {4, "bianhuang_legend", "边荒传说", 2},
		"fuyu_moshigong":        {4, "fuyu_fan_yun", "覆雨翻云", 3},
		"datang_wagang":         {4, "datang_double_dragon", "大唐双龙传", 7},
		"datang_liangdao":       {4, "datang_double_dragon", "大唐双龙传", 8},
		"datang_baling":         {4, "datang_double_dragon", "大唐双龙传", 9},
		"datang_sanhua":         {4, "datang_double_dragon", "大唐双龙传", 10},
		"fuyu_fangfuchou":       {4, "fuyu_fan_yun", "覆雨翻云", 4},
		"fuyu_shuizhan":         {4, "fuyu_fan_yun", "覆雨翻云", 5},
		"posui_houjuan":         {4, "posui_xukong", "破碎虚空", 4},
		"bianhuang_duoqi":       {4, "bianhuang_legend", "边荒传说", 3},
		"luoshui_kujing":        {4, "luoshui_raiders", "洛水风浪", 2},
		"datang_shudao":         {4, "datang_double_dragon", "大唐双龙传", 11},
		"datang_zhandao":        {4, "datang_double_dragon", "大唐双龙传", 12},
		"saibei_langshan":       {4, "bianhuang_legend", "边荒传说", 4},
		"jiangnan_jianchi":      {4, "luoshui_raiders", "洛水风浪", 3},
		"dongdu_tianjin":        {4, "luoshui_raiders", "洛水风浪", 4},
		"dongming_haisi":        {4, "luoshui_raiders", "洛水风浪", 5},
	}
	loaded := map[string]bool{}
	for _, id := range m.DungeonTemplateIDsNoLock() {
		loaded[id] = true
		if strings.HasPrefix(id, "xunqin") {
			t.Fatalf("xunqin dungeon should have been removed, got %s", id)
		}
	}
	for id, meta := range expected {
		tpl, ok := m.DungeonTemplateNoLock(id)
		if !ok {
			t.Fatalf("missing redesigned dungeon %s", id)
		}
		if len(tpl.Rooms) != meta.rooms {
			t.Fatalf("dungeon %s expected %d rooms, got %d", id, meta.rooms, len(tpl.Rooms))
		}
		if tpl.ChainID != meta.chainID || tpl.ChainName != meta.chainName || tpl.Chapter != meta.chapter {
			t.Fatalf("dungeon %s has invalid chain metadata: chain=%s name=%s chapter=%d", id, tpl.ChainID, tpl.ChainName, tpl.Chapter)
		}
	}
	if len(loaded) != len(expected) {
		t.Fatalf("unexpected extra dungeons loaded: %+v", loaded)
	}
}

// TestCombatFocusedDesignConstraints 落实"战斗为主、解谜克制"的设计约束：
// 击杀权重必须过半、解谜动作至多两个、终局副本必须由前序副本完成旗标解锁。
func TestCombatFocusedDesignConstraints(t *testing.T) {
	m := NewManager(map[string]model.NPC{}, map[string]Template{}, map[string]*Instance{}, map[string]string{}, map[string]string{}, map[string]*NPCState{})
	if err := m.LoadTemplatesNoLock(); err != nil {
		t.Fatalf("load dungeon templates failed: %v", err)
	}
	for _, id := range m.DungeonTemplateIDsNoLock() {
		tpl, _ := m.DungeonTemplateNoLock(id)
		killWeight := 0
		puzzleWeight := 0
		for _, objective := range tpl.Objectives {
			if objective.Optional {
				continue
			}
			switch objective.Type {
			case "kill":
				killWeight += objective.Weight
			case "puzzle":
				puzzleWeight += objective.Weight
			}
		}
		if killWeight < 50 {
			t.Fatalf("dungeon %s is not combat focused: kill weight %d", id, killWeight)
		}
		if puzzleWeight > 20 {
			t.Fatalf("dungeon %s carries too much puzzle weight: %d", id, puzzleWeight)
		}
	}

	lanjiang, ok := m.DungeonTemplateNoLock("fuyu_lanjiang")
	if !ok {
		t.Fatalf("missing finale dungeon fuyu_lanjiang")
	}
	if len(lanjiang.UnlockConditions) != 2 {
		t.Fatalf("finale dungeon must require two completion flags, got %+v", lanjiang.UnlockConditions)
	}
	requiredFlags := map[string]bool{}
	for _, cond := range lanjiang.UnlockConditions {
		if cond.Type != "flag" {
			t.Fatalf("finale unlock must use flag conditions, got %+v", lanjiang.UnlockConditions)
		}
		requiredFlags[cond.TargetID] = true
	}
	if !requiredFlags["fuyu_nujiao_completed"] || !requiredFlags["posui_jingyan_completed"] {
		t.Fatalf("finale unlock flags mismatch: %+v", lanjiang.UnlockConditions)
	}
	nujiao, _ := m.DungeonTemplateNoLock("fuyu_nujiao")
	if !nujiao.Rewards.Flags["fuyu_nujiao_completed"] {
		t.Fatalf("fuyu_nujiao must reward its completion flag")
	}
	posui, _ := m.DungeonTemplateNoLock("posui_jingyan")
	if !posui.Rewards.Flags["posui_jingyan_completed"] {
		t.Fatalf("posui_jingyan must reward its completion flag")
	}
}

func TestDungeonPuzzleStateGraphsClassifyEveryConfiguredState(t *testing.T) {
	m := NewManager(map[string]model.NPC{}, map[string]Template{}, map[string]*Instance{}, map[string]string{}, map[string]string{}, map[string]*NPCState{})
	if err := m.LoadTemplatesNoLock(); err != nil {
		t.Fatalf("load dungeon templates failed: %v", err)
	}
	for _, id := range m.DungeonTemplateIDsNoLock() {
		tpl, _ := m.DungeonTemplateNoLock(id)
		inst := &Instance{DungeonID: id, PuzzleStates: map[string]string{}}
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
			if action.DialogueNPCID == "" && len(action.Rooms) == 0 {
				t.Fatalf("dungeon %s action %s is bound to no room", id, actionID)
			}
			for _, from := range action.FromStates {
				if !stateReachableNoLock(tpl, action.PuzzleID, from) {
					t.Fatalf("dungeon %s action %s starts from unreachable %s state %s", id, actionID, action.PuzzleID, from)
				}
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
		}
	}
}

func stateReachableNoLock(tpl Template, puzzleID, target string) bool {
	if initial, ok := tpl.PuzzleInitialStates[puzzleID]; !ok {
		return false
	} else if initial == target {
		return true
	}
	reachable := map[string]bool{tpl.PuzzleInitialStates[puzzleID]: true}
	changed := true
	for changed {
		changed = false
		for _, action := range tpl.PuzzleActions {
			if action.PuzzleID != puzzleID || action.NextState == "" {
				continue
			}
			for _, from := range action.FromStates {
				if reachable[from] && !reachable[action.NextState] {
					reachable[action.NextState] = true
					changed = true
				}
			}
		}
	}
	return reachable[target]
}

func testContainsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestDungeonDialogueScriptsCoverEveryStoryAction(t *testing.T) {
	m := NewManager(map[string]model.NPC{}, map[string]Template{}, map[string]*Instance{}, map[string]string{}, map[string]string{}, map[string]*NPCState{})
	if err := m.LoadTemplatesNoLock(); err != nil {
		t.Fatalf("load dungeon templates failed: %v", err)
	}
	covered := 0
	for _, id := range m.DungeonTemplateIDsNoLock() {
		tpl, _ := m.DungeonTemplateNoLock(id)
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
	if covered != 34 {
		t.Fatalf("expected all 34 story dialogue actions, got %d", covered)
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
