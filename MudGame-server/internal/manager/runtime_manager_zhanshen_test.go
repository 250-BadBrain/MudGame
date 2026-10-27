package manager

import (
	"testing"
)

// 战神殿全流程：萨满/高手 → 参悟全图（两步破招）→ 斩碎心象 → 玉盒 → 100% 结算
func TestZhanshenFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "posui_zhanshen")

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

	if _, err := r.DoRoomAction(playerID, "listen_ying"); err != nil {
		t.Fatalf("listen ying failed: %v", err)
	}
	defeatAll(room("zd_00"), "zd_shaman_guard", 2)
	steps := room("zd_01")
	moveDungeonTestPlayer(t, r, playerID, steps)
	defeatAll(steps, "zd_tulu_adept", 2)

	gate := room("zd_02")
	moveDungeonTestPlayer(t, r, playerID, gate)
	for _, action := range []string{"view_atlas", "comprehend_atlas"} {
		if _, err := r.DoRoomAction(playerID, action); err != nil {
			t.Fatalf("action %s failed: %v", action, err)
		}
	}
	if inst.PuzzleStates["atlas"] != "comprehended" {
		t.Fatalf("expected atlas comprehended, got %s", inst.PuzzleStates["atlas"])
	}

	hall := room("zd_03")
	moveDungeonTestPlayer(t, r, playerID, hall)
	phantomID := dungeonNPCIDByTemplateInRoom(t, inst, hall, "zd_war_phantom")
	baseAttack := inst.NPCs[phantomID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, phantomID); err != nil {
		t.Fatalf("engage phantom failed: %v", err)
	}
	if !inst.NPCs[phantomID].WeaknessApplied {
		t.Fatal("expected atlas insight to weaken war phantom")
	}
	if got := inst.NPCs[phantomID].Attributes.Derived.Attack; got != baseAttack-14 {
		t.Fatalf("expected attack weakened by 14, got %d (base %d)", got, baseAttack)
	}
	r.mu.Lock()
	_, defeated, _, ok := r.dungeonMgr.DefeatNPCNoLock(phantomID)
	r.mu.Unlock()
	if !ok || !defeated.Alive || !defeated.Defeated {
		t.Fatalf("phantom should shatter but stay, got %+v", defeated)
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
		t.Fatalf("expected 100%% without optional box, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}
	if _, err := r.DoRoomAction(playerID, "open_box"); err != nil {
		t.Fatalf("open box failed: %v", err)
	}

	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave dungeon failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["dahuandan"] < 1 { // FirstClaim/learnSkill guaranteed
		t.Fatalf("expected dahuandan, got %d", p.Backpack["dahuandan"])
	}
	if !p.Flags["dungeon_cleared_posui_zhanshen"] {
		t.Fatal("expected dungeon_cleared_posui_zhanshen flag")
	}
}

// 洛阳粮道全流程：游骑 → 断桥破招 → 逐走王伯当 → 斩马刀结算
func TestLiangdaoFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_liangdao")
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
	if _, err := r.DoRoomAction(playerID, "listen_courier"); err != nil {
		t.Fatalf("listen courier failed: %v", err)
	}
	pass := room("ld_01")
	moveDungeonTestPlayer(t, r, playerID, pass)
	defeatAll(pass, "ld_wagang_rider", 2)
	gorge := room("ld_02")
	moveDungeonTestPlayer(t, r, playerID, gorge)
	if _, err := r.DoRoomAction(playerID, "break_bridge"); err != nil {
		t.Fatalf("break bridge failed: %v", err)
	}
	bank := room("ld_03")
	moveDungeonTestPlayer(t, r, playerID, bank)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, bank, "ld_wang_bodang")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage wang failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected bridge sabotage to weaken wang bodang")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat wang failed")
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
		t.Fatalf("expected 100%%, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave dungeon failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if !p.Flags["dungeon_cleared_datang_liangdao"] {
		t.Fatal("expected cleared flag")
	}
	t.Logf("zhanma_dao count: %d (probabilistic)", p.Backpack["zhanma_dao"])
}

// 巴陵夜泊全流程：哨探 → 烧桥破招 → 挫张绣 → 软甲+沉箱结算
func TestBalingFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_baling")
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
	if _, err := r.DoRoomAction(playerID, "listen_boatwoman"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	reed := room("db_01")
	moveDungeonTestPlayer(t, r, playerID, reed)
	if _, err := r.DoRoomAction(playerID, "salvage_float_box"); err != nil {
		t.Fatalf("salvage failed: %v", err)
	}
	defeatAll(reed, "db_scout", 2)
	bridge := room("db_02")
	moveDungeonTestPlayer(t, r, playerID, bridge)
	defeatAll(bridge, "db_commander", 2)
	if _, err := r.DoRoomAction(playerID, "burn_float_bridge"); err != nil {
		t.Fatalf("burn bridge failed: %v", err)
	}
	ship := room("db_03")
	moveDungeonTestPlayer(t, r, playerID, ship)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, ship, "db_zhang_xiu")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	baseHit := inst.NPCs[bossID].Attributes.Derived.HitRate
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage zhang failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-24 || inst.NPCs[bossID].Attributes.Derived.HitRate != baseHit-26 {
		t.Fatal("expected bridge fire to weaken zhang xiu")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat zhang failed")
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
		t.Fatalf("expected 100%%, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["dongting_ruanjia"] != 1 || p.Backpack["manual_dongting_xinf"] != 1 {
		t.Fatalf("expected rewards, got ruanjia=%d page=%d", p.Backpack["dongting_ruanjia"], p.Backpack["manual_dongting_xinf"])
	}
	if !p.Flags["dungeon_cleared_datang_baling"] {
		t.Fatal("expected cleared flag")
	}
}

// 散花楼全流程：暗桩 → 护楼使 → 识破师承 → 挫花弄影 → 玉镯+锦盒结算
func TestSanhuaFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_sanhua")
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
	if _, err := r.DoRoomAction(playerID, "listen_informant"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	hall := room("sh_01")
	moveDungeonTestPlayer(t, r, playerID, hall)
	defeatAll(hall, "sh_yinkui_maid", 2)
	attic := room("sh_02")
	moveDungeonTestPlayer(t, r, playerID, attic)
	defeatAll(attic, "sh_guard_captain", 1)
	if _, err := r.DoRoomAction(playerID, "see_lodge_truth"); err != nil {
		t.Fatalf("see truth failed: %v", err)
	}
	top := room("sh_03")
	moveDungeonTestPlayer(t, r, playerID, top)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, top, "sh_hua_wuying")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage hua failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected lodge truth to weaken hua wuying")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat hua failed")
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
		t.Fatalf("expected 100%%, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}
	if _, err := r.DoRoomAction(playerID, "open_box_sh"); err != nil {
		t.Fatalf("open box failed: %v", err)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["sanhua_yuzhuo"] < 0 || p.Backpack["manual_lingxu_bu"] != 1 {
		t.Fatalf("expected rewards, got yuzhuo=%d page=%d", p.Backpack["sanhua_yuzhuo"], p.Backpack["manual_lingxu_bu"])
	}
	if !p.Flags["dungeon_cleared_datang_sanhua"] {
		t.Fatal("expected cleared flag")
	}
}

// 方夜羽复仇全流程：好手 → 设伏破招 → 再挫方夜羽 → 双刀结算
func TestFangfuchouFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "fuyu_fangfuchou")
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
	if _, err := r.DoRoomAction(playerID, "listen_helmsman"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	gate := room("fy2_01")
	moveDungeonTestPlayer(t, r, playerID, gate)
	defeatAll(gate, "fy2_moshi_elite", 2)
	hall := room("fy2_02")
	moveDungeonTestPlayer(t, r, playerID, hall)
	if _, err := r.DoRoomAction(playerID, "set_ambush"); err != nil {
		t.Fatalf("set ambush failed: %v", err)
	}
	inner := room("fy2_03")
	moveDungeonTestPlayer(t, r, playerID, inner)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, inner, "fy2_fang_yeyu")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage fang failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected ambush to weaken fang yeyu")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat fang failed")
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
		t.Fatalf("expected 100%%, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if !p.Flags["dungeon_cleared_fuyu_fangfuchou"] {
		t.Fatal("expected cleared flag")
	}
	t.Logf("yulie_shuangdao count: %d (probabilistic)", p.Backpack["yulie_shuangdao"])
}

// 怒蛟水战全流程：水手/百户 → 沉船锁江破招 → 挫赤老温 → 分水刺结算
func TestShuizhanFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "fuyu_shuizhan")
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
	if _, err := r.DoRoomAction(playerID, "listen_scout"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	waves := room("hy2_01")
	moveDungeonTestPlayer(t, r, playerID, waves)
	defeatAll(waves, "hy2_yuan_sailor", 2)
	mouth := room("hy2_02")
	moveDungeonTestPlayer(t, r, playerID, mouth)
	defeatAll(mouth, "hy2_yuan_captain", 2)
	if _, err := r.DoRoomAction(playerID, "sink_ships"); err != nil {
		t.Fatalf("sink ships failed: %v", err)
	}
	deck := room("hy2_03")
	moveDungeonTestPlayer(t, r, playerID, deck)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, deck, "hy2_chilaowen")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage chi failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected sunken ships to weaken chi laowen")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat chi failed")
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
		t.Fatalf("expected 100%%, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["fenshui_ci"] < 0 || !p.Flags["dungeon_cleared_fuyu_shuizhan"] {
		t.Fatalf("expected rewards, got ci=%d flag=%v", p.Backpack["fenshui_ci"], p.Flags["dungeon_cleared_fuyu_shuizhan"])
	}
}

// 图录·后卷全流程：武士/萨满 → 参悟后卷 → 挫耶律玄 → 残页结算
func TestHoujuanFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "posui_houjuan")
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
	if _, err := r.DoRoomAction(playerID, "listen_monk"); err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	camp := room("jm2_01")
	moveDungeonTestPlayer(t, r, playerID, camp)
	defeatAll(camp, "jm2_mongol_prince_guard", 2)
	convoy := room("jm2_02")
	moveDungeonTestPlayer(t, r, playerID, convoy)
	defeatAll(convoy, "jm2_saman_elder", 1)
	if _, err := r.DoRoomAction(playerID, "read_scroll"); err != nil {
		t.Fatalf("read scroll failed: %v", err)
	}
	cliff := room("jm2_03")
	moveDungeonTestPlayer(t, r, playerID, cliff)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, cliff, "jm2_yelv_xuan")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage yelv failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied || inst.NPCs[bossID].Attributes.Derived.Attack != baseAttack-14 {
		t.Fatal("expected scroll insight to weaken yelv xuan")
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat yelv failed")
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
		t.Fatalf("expected 100%%, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["manual_poxu_zhi"] != 1 || !p.Flags["dungeon_cleared_posui_houjuan"] {
		t.Fatalf("expected rewards, got page=%d flag=%v", p.Backpack["manual_poxu_zhi"], p.Flags["dungeon_cleared_posui_houjuan"])
	}
}
