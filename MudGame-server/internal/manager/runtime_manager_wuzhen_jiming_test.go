package manager

import (
	"testing"
)

// 悟真篇：装备后学习结算注入上限提升（10 → 10+X）
func TestWuzhenLearningInvestBonus(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("scholar", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("scholar", "读书人", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	setup := func(potential int) {
		t.Helper()
		r.mu.Lock()
		players, _ := r.loadPlayers()
		p := players[playerID]
		p.RoomID = "ly_weapon_school"
		// 经验给到上限 100 级档，技能顶到 80 级（升 81 级恰需 11 潜能，可检验注入上限）
		p.Attributes.Experience = 20480
		p.Attributes.Potential = potential
		p.Skills["basic_unarmed"] = 80
		p.LearningSkillID = "basic_unarmed"
		p.LearningTeacherID = "weapon_school_master"
		p.LearningTeacherLevel = 300
		p.SkillLearningProgress = map[string]int{}
		p.State = "LEARNING"
		p.StateTickCountdown = 1 // 直接对齐结算点
		if err := r.savePlayers(players); err != nil {
			r.mu.Unlock()
			t.Fatalf("save failed: %v", err)
		}
		r.mu.Unlock()
	}

	// 无秘籍：一次结算注入 10
	setup(50)
	if _, err := r.ProcessStatusTick(); err != nil {
		t.Fatalf("tick failed: %v", err)
	}
	after, _ := r.loadPlayers()
	if spent := 50 - after[playerID].Attributes.Potential; spent != 10 {
		t.Fatalf("expected 10 invested without manual, got %d", spent)
	}

	// 装备悟真篇 10 级：注入上限 11
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].Skills["basic_literacy"] = 20
	players[playerID].Skills["wuzhen_pian"] = 10
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()
	if _, err := r.EquipSkill(playerID, "wuzhen_pian"); err != nil {
		t.Fatalf("equip wuzhen failed: %v", err)
	}

	setup(50)
	if _, err := r.ProcessStatusTick(); err != nil {
		t.Fatalf("tick failed: %v", err)
	}
	after, _ = r.loadPlayers()
	if spent := 50 - after[playerID].Attributes.Potential; spent != 11 {
		t.Fatalf("expected 11 invested with wuzhen equipped, got %d", spent)
	}
}

// 鸡鸣山之约全流程：武士/高手 → 悟破碎之意 → 破招逐走思汉飞 → 供品箱 → 100% 结算
func TestJimingshanFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "posui_jiming")

	room := func(id string) string { return r.dungeonMgr.InstanceRoomIDNoLock(inst, id) }
	defeatAll := func(roomID, template string, count int) {
		t.Helper()
		for i := 0; i < count; i++ {
			npcID := livingDungeonNPCByTemplateInRoom(t, inst, roomID, template)
			r.mu.Lock()
			_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(npcID)
			r.mu.Unlock()
			if !ok {
				t.Fatalf("defeat %s #%d failed", template, i+1)
			}
		}
	}

	if _, err := r.DoRoomAction(playerID, "listen_qi"); err != nil {
		t.Fatalf("listen qi failed: %v", err)
	}
	slope := room("jm_01")
	moveDungeonTestPlayer(t, r, playerID, slope)
	defeatAll(slope, "jm_mongol_warrior", 2)
	stairs := room("jm_02")
	moveDungeonTestPlayer(t, r, playerID, stairs)
	defeatAll(stairs, "jm_mongol_adept", 2)

	// 石壁：静观 → 悟意（旗标写入）
	wall := room("jm_03")
	moveDungeonTestPlayer(t, r, playerID, wall)
	for _, action := range []string{"view_carving", "comprehend_void"} {
		if _, err := r.DoRoomAction(playerID, action); err != nil {
			t.Fatalf("action %s failed: %v", action, err)
		}
	}
	if inst.PuzzleStates["carving"] != "comprehended" {
		t.Fatalf("expected carving comprehended, got %s", inst.PuzzleStates["carving"])
	}

	// 山巅：破招 → defeat 思汉飞 → 归位 → 供品箱
	peak := room("jm_04")
	moveDungeonTestPlayer(t, r, playerID, peak)
	siID := dungeonNPCIDByTemplateInRoom(t, inst, peak, "jm_si_hanfei")
	baseAttack := inst.NPCs[siID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, siID); err != nil {
		t.Fatalf("engage si hanfei failed: %v", err)
	}
	if !inst.NPCs[siID].WeaknessApplied {
		t.Fatal("expected posui insight to weaken si hanfei")
	}
	if got := inst.NPCs[siID].Attributes.Derived.Attack; got != baseAttack-14 {
		t.Fatalf("expected attack weakened by 14, got %d (base %d)", got, baseAttack)
	}
	r.mu.Lock()
	_, defeated, _, ok := r.dungeonMgr.DefeatNPCNoLock(siID)
	r.mu.Unlock()
	if !ok || !defeated.Alive || !defeated.Defeated {
		t.Fatalf("si hanfei should yield and stay, got %+v", defeated)
	}
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].State = "IDLE"
	players[playerID].TargetID = ""
	players[playerID].CombatType = ""
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()
	if inst.Progress != 100 {
		t.Fatalf("expected 100%% without optional supply, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}
	if _, err := r.DoRoomAction(playerID, "take_supply"); err != nil {
		t.Fatalf("take supply failed: %v", err)
	}

	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave dungeon failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["dry_ration"] < 2 || p.Backpack["bronze_fragment"] < 2 {
		t.Fatalf("expected supply items, got ration=%d bronze=%d", p.Backpack["dry_ration"], p.Backpack["bronze_fragment"])
	}
	if !p.Flags["dungeon_cleared_posui_jiming"] {
		t.Fatal("expected dungeon_cleared_posui_jiming flag")
	}
}
