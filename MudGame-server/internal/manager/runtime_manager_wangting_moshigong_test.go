package manager

import (
	"testing"
)

// 马贼王庭全流程：弓手/头目 → 焚粮破招 → 逐走赫连霸 → 狼王刀结算
func TestWangtingFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "bianhuang_wangting")
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
		t.Fatalf("listen scout failed: %v", err)
	}
	tower := room("wt_01")
	moveDungeonTestPlayer(t, r, playerID, tower)
	defeatAll(tower, "wt_archer", 2)
	front := room("wt_02")
	moveDungeonTestPlayer(t, r, playerID, front)
	defeatAll(front, "wt_chief", 2)
	if _, err := r.DoRoomAction(playerID, "burn_granary"); err != nil {
		t.Fatalf("burn granary failed: %v", err)
	}
	if inst.PuzzleStates["granary"] != "burned" {
		t.Fatalf("expected granary burned, got %s", inst.PuzzleStates["granary"])
	}

	tent := room("wt_03")
	moveDungeonTestPlayer(t, r, playerID, tent)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, tent, "wt_helian_ba")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage helian failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied {
		t.Fatal("expected granary burning to weaken helian ba")
	}
	if got := inst.NPCs[bossID].Attributes.Derived.Attack; got != baseAttack-14 {
		t.Fatalf("expected attack weakened by 14, got %d (base %d)", got, baseAttack)
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat helian failed")
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
	if p.Backpack["langwang_dao"] != 1 {
		t.Fatalf("expected langwang dao, got %d", p.Backpack["langwang_dao"])
	}
	if !p.Flags["dungeon_cleared_bianhuang_wangting"] {
		t.Fatal("expected cleared flag")
	}
}

// 魔师宫索战全流程：弟子/好手 → 击鼓聚将破招 → 逐走厉工 → 天魔佩结算
func TestMoshigongFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "fuyu_moshigong")
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

	if _, err := r.DoRoomAction(playerID, "listen_steward"); err != nil {
		t.Fatalf("listen steward failed: %v", err)
	}
	bridge := room("mg_01")
	moveDungeonTestPlayer(t, r, playerID, bridge)
	defeatAll(bridge, "mg_disciple", 2)
	ground := room("mg_02")
	moveDungeonTestPlayer(t, r, playerID, ground)
	defeatAll(ground, "mg_adept", 2)
	if _, err := r.DoRoomAction(playerID, "beat_drum"); err != nil {
		t.Fatalf("beat drum failed: %v", err)
	}
	if inst.PuzzleStates["drum"] != "thundering" {
		t.Fatalf("expected drum thundering, got %s", inst.PuzzleStates["drum"])
	}

	hall := room("mg_03")
	moveDungeonTestPlayer(t, r, playerID, hall)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, hall, "mg_li_gong")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage li gong failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied {
		t.Fatal("expected drum morale to weaken li gong")
	}
	if got := inst.NPCs[bossID].Attributes.Derived.Attack; got != baseAttack-14 {
		t.Fatalf("expected attack weakened by 14, got %d (base %d)", got, baseAttack)
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat li gong failed")
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
	if p.Backpack["tianmo_pei"] != 1 {
		t.Fatalf("expected tianmo pei, got %d", p.Backpack["tianmo_pei"])
	}
	if !p.Flags["dungeon_cleared_fuyu_moshigong"] {
		t.Fatal("expected cleared flag")
	}
}

// 瓦岗惊变全流程：锐士/辎重卫 → 开仓放粮破招 → 逐走罗士信 → 铁甲结算
func TestWagangFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_wagang")
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

	if _, err := r.DoRoomAction(playerID, "listen_defender"); err != nil {
		t.Fatalf("listen defender failed: %v", err)
	}
	camp := room("wg_01")
	moveDungeonTestPlayer(t, r, playerID, camp)
	defeatAll(camp, "wg_elite", 2)
	depot := room("wg_02")
	moveDungeonTestPlayer(t, r, playerID, depot)
	defeatAll(depot, "wg_quartermaster", 2)
	if _, err := r.DoRoomAction(playerID, "open_granary"); err != nil {
		t.Fatalf("open granary failed: %v", err)
	}
	if inst.PuzzleStates["granary_wg"] != "opened" {
		t.Fatalf("expected granary opened, got %s", inst.PuzzleStates["granary_wg"])
	}

	banner := room("wg_03")
	moveDungeonTestPlayer(t, r, playerID, banner)
	bossID := dungeonNPCIDByTemplateInRoom(t, inst, banner, "wg_luo_shixin")
	baseAttack := inst.NPCs[bossID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, bossID); err != nil {
		t.Fatalf("engage luo failed: %v", err)
	}
	if !inst.NPCs[bossID].WeaknessApplied {
		t.Fatal("expected minxin to weaken luo shixin")
	}
	if got := inst.NPCs[bossID].Attributes.Derived.Attack; got != baseAttack-14 {
		t.Fatalf("expected attack weakened by 14, got %d (base %d)", got, baseAttack)
	}
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(bossID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat luo failed")
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
	if p.Backpack["wagang_tiejia"] != 1 {
		t.Fatalf("expected wagang tiejia, got %d", p.Backpack["wagang_tiejia"])
	}
	if !p.Flags["dungeon_cleared_datang_wagang"] {
		t.Fatal("expected cleared flag")
	}
}
