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
	if p.Backpack["dahuandan"] != 1 {
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
	if p.Backpack["zhanma_dao"] != 1 || !p.Flags["dungeon_cleared_datang_liangdao"] {
		t.Fatalf("expected rewards, got dao=%d flag=%v", p.Backpack["zhanma_dao"], p.Flags["dungeon_cleared_datang_liangdao"])
	}
}
