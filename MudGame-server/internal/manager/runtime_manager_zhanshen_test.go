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
