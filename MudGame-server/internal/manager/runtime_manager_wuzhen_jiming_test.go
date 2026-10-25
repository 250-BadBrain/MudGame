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

// 边荒集夺旗全流程
func TestDuoqiFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "bianhuang_duoqi")
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
	if _, err := r.DoRoomAction(playerID, "listen_caravan"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defeatAll(room("bd_01"), "bd_flag_guard", 2)
	yuanmen := room("bd_02")
	moveDungeonTestPlayer(t, r, playerID, yuanmen)
	defeatAll(yuanmen, "bd_flag_vanguard", 1)
	banner := room("bd_03")
	moveDungeonTestPlayer(t, r, playerID, banner)
	if _, err := r.DoRoomAction(playerID, "seize_flag"); err != nil {
		t.Fatalf("seize flag failed: %v", err)
	}
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, banner, "bd_tuoba_gu")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage tuoba failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected flag seizure to weaken tuoba gu")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat tuoba failed")
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
		t.Fatalf("expected 100%%, got %d", inst.Progress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	if after[playerID].Backpack["bianhuang_xue"] != 1 || !after[playerID].Flags["dungeon_cleared_bianhuang_duoqi"] {
		t.Fatal("expected rewards")
	}
}

// 枯井之下全流程
func TestKujingFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "luoshui_kujing")
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
	if _, err := r.DoRoomAction(playerID, "listen_rope"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defeatAll(room("kj_01"), "kj_yinkui_guard", 2)
	depth := room("kj_02")
	moveDungeonTestPlayer(t, r, playerID, depth)
	defeatAll(depth, "kj_yinkui_adept", 1)
	if _, err := r.DoRoomAction(playerID, "descend_well"); err != nil {
		t.Fatalf("descend failed: %v", err)
	}
	cellar := room("kj_03")
	moveDungeonTestPlayer(t, r, playerID, cellar)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, cellar, "kj_kujing_weng")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage kujing failed: %v", err)
	}
	if _, err := r.DoRoomAction(playerID, "descend_well"); err == nil {
		t.Fatal("descend should not repeat")
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected well descent to weaken kujing weng")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat kujing failed")
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
		t.Fatalf("expected 100%%, got %d", inst.Progress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	if after[playerID].Backpack["zhenshui_huan"] != 1 || !after[playerID].Flags["dungeon_cleared_luoshui_kujing"] {
		t.Fatal("expected rewards")
	}
}

// 蜀道护镖全流程
func TestShudaoFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_shudao")
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
	if _, err := r.DoRoomAction(playerID, "listen_escort"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defeatAll(room("sd_01"), "sd_ape_scout", 2)
	net := room("sd_02")
	moveDungeonTestPlayer(t, r, playerID, net)
	defeatAll(net, "sd_ape_rouge", 1)
	if _, err := r.DoRoomAction(playerID, "cut_ropenet"); err != nil {
		t.Fatalf("cut ropenet failed: %v", err)
	}
	end := room("sd_03")
	moveDungeonTestPlayer(t, r, playerID, end)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, end, "sd_yuan_sou")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage yuan sou failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected ropenet cut to weaken yuan sou")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat yuan sou failed")
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
		t.Fatalf("expected 100%%, got %d", inst.Progress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	if after[playerID].Backpack["shujin_pifeng"] != 1 || !after[playerID].Flags["dungeon_cleared_datang_shudao"] {
		t.Fatal("expected rewards")
	}
}

// 蜀道栈桥全流程：余党 → 断藤破招 → 擒小猿王 → 铁蒺藜+任务旗标结算
func TestZhandaoFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_zhandao")
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
	if _, err := r.DoRoomAction(playerID, "listen_tang"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	cliff := room("zd2_01")
	moveDungeonTestPlayer(t, r, playerID, cliff)
	defeatAll(cliff, "zd2_ape_rouge", 2)
	ladder := room("zd2_02")
	moveDungeonTestPlayer(t, r, playerID, ladder)
	if _, err := r.DoRoomAction(playerID, "cut_vinerope"); err != nil {
		t.Fatalf("cut vinerope failed: %v", err)
	}
	nest := room("zd2_03")
	moveDungeonTestPlayer(t, r, playerID, nest)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, nest, "zd2_ape_king")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage ape king failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected cut vines to weaken ape king")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat ape king failed")
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
		t.Fatalf("expected 100%%, got %d", inst.Progress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["tangmen_tili"] != 1 || !p.Flags["dungeon_cleared_datang_zhandao"] || !p.Flags["zhandao_case_done"] {
		t.Fatalf("expected rewards, got tili=%d flags=%v", p.Backpack["tangmen_tili"], p.Flags)
	}
}

// 狼山口全流程：死士/狼卫长 → 拔狼王旗破招 → 挫白狼主 → 巨锤+任务旗标结算
func TestLangshanFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "saibei_langshan")
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
	if _, err := r.DoRoomAction(playerID, "listen_centurion"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	pass := room("sb2_01")
	moveDungeonTestPlayer(t, r, playerID, pass)
	defeatAll(pass, "sb2_wolf_guard", 2)
	tent := room("sb2_02")
	moveDungeonTestPlayer(t, r, playerID, tent)
	defeatAll(tent, "sb2_wolf_champion", 1)
	flagRoom := room("sb2_03")
	moveDungeonTestPlayer(t, r, playerID, flagRoom)
	if _, err := r.DoRoomAction(playerID, "pull_wolfflag"); err != nil {
		t.Fatalf("pull wolfflag failed: %v", err)
	}
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, flagRoom, "sb2_white_wolf")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage white wolf failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected flag pull to weaken white wolf")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat white wolf failed")
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
		t.Fatalf("expected 100%%, got %d", inst.Progress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["langya_chui"] != 1 || !p.Flags["dungeon_cleared_saibei_langshan"] || !p.Flags["langshan_case_done"] {
		t.Fatalf("expected rewards, got chui=%d flags=%v", p.Backpack["langya_chui"], p.Flags)
	}
}

// 剑池全流程：蛙人 → 开泉眼 → 挫青衣剑客 → 淬泉剑结算
func TestJianchiFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "jiangnan_jianchi")
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
	if _, err := r.DoRoomAction(playerID, "listen_zither"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	steps := room("jn2_01")
	moveDungeonTestPlayer(t, r, playerID, steps)
	defeatAll(steps, "jn2_pool_frog", 2)
	gate := room("jn2_02")
	moveDungeonTestPlayer(t, r, playerID, gate)
	if _, err := r.DoRoomAction(playerID, "open_spring"); err != nil {
		t.Fatalf("open spring failed: %v", err)
	}
	cave := room("jn2_03")
	moveDungeonTestPlayer(t, r, playerID, cave)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, cave, "jn2_sword_thief")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage thief failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected spring opening to weaken thief")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat thief failed")
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
		t.Fatalf("expected 100%%, got %d", inst.Progress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["jianchi_jian"] != 1 || !p.Flags["dungeon_cleared_jiangnan_jianchi"] || !p.Flags["jianchi_case_done"] {
		t.Fatalf("expected rewards, got jian=%d flags=%v", p.Backpack["jianchi_jian"], p.Flags)
	}
}

// 天津晓月全流程：水鬼 → 取账册（破招）→ 挫起居郎 → 晓月佩+任务旗标结算
func TestTianjinFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "dongdu_tianjin")
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
	if _, err := r.DoRoomAction(playerID, "listen_poet"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	if _, err := r.DoRoomAction(playerID, "recite_poem"); err != nil {
		t.Fatalf("recite poem failed: %v", err)
	}
	shallows := room("dd3_01")
	moveDungeonTestPlayer(t, r, playerID, shallows)
	defeatAll(shallows, "dd3_water_ghost", 2)
	corridor := room("dd3_02")
	moveDungeonTestPlayer(t, r, playerID, corridor)
	if _, err := r.DoRoomAction(playerID, "take_ledger"); err == nil {
		t.Fatal("ledger must be gated behind the keeper")
	}
	chamber := room("dd3_03")
	moveDungeonTestPlayer(t, r, playerID, chamber)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, chamber, "dd3_ledger_guard")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage keeper failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected water ghosts retreat to weaken keeper")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat keeper failed")
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
	if _, err := r.DoRoomAction(playerID, "take_ledger"); err != nil {
		t.Fatalf("take ledger failed: %v", err)
	}
	if inst.Progress != 100 {
		t.Fatalf("expected 100%%, got %d", inst.Progress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["tianjin_yuepei"] != 1 || !p.Flags["dungeon_cleared_dongdu_tianjin"] || !p.Flags["tianjin_case_done"] {
		t.Fatalf("expected rewards, got pei=%d flags=%v", p.Backpack["tianjin_yuepei"], p.Flags)
	}
}
