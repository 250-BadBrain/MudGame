package manager

import (
	"testing"
)

// 天刀试炼全流程：木人阵 → 悟刀（双解机关）→ 心魔 → 宋缺三刀之约（defeat 模式）→ 100% 结算 + 残页参悟
func TestTiandaoTrialFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_tiandao")

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

	// 第一关：木人阵
	hall := room("td_01")
	moveDungeonTestPlayer(t, r, playerID, hall)
	defeatAll(hall, "td_wood_puppet", 4)
	if inst.ObjectiveProgress["break_puppets"] != 4 {
		t.Fatalf("expected puppets objective done, got %+v", inst.ObjectiveProgress)
	}

	// 第二关：悟刀（双解——刚/灵皆正解）
	wall := room("td_02")
	moveDungeonTestPlayer(t, r, playerID, wall)
	for _, action := range []string{"trace_carving", "ponder_blade", "choose_intent", "flow_path"} {
		if _, err := r.DoRoomAction(playerID, action); err != nil {
			t.Fatalf("action %s failed: %v", action, err)
		}
	}
	if inst.PuzzleStates["blade"] != "enlightened" {
		t.Fatalf("expected blade enlightened via flow path, got %s", inst.PuzzleStates["blade"])
	}

	// 第三关：心魔
	pavilion := room("td_03")
	moveDungeonTestPlayer(t, r, playerID, pavilion)
	defeatAll(pavilion, "td_inner_shadow", 1)

	// 决战：宋缺三刀之约——悟刀旗标先行破招，defeat 模式力竭收刀不死
	arena := room("td_04")
	moveDungeonTestPlayer(t, r, playerID, arena)
	songID := dungeonNPCIDByTemplateInRoom(t, inst, arena, "td_song_que")
	baseAttack := inst.NPCs[songID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, songID); err != nil {
		t.Fatalf("engage song que failed: %v", err)
	}
	if !inst.NPCs[songID].WeaknessApplied {
		t.Fatal("expected enlightened blade to weaken song que")
	}
	if got := inst.NPCs[songID].Attributes.Derived.Attack; got != baseAttack-12 {
		t.Fatalf("expected attack weakened by 12, got %d (base %d)", got, baseAttack)
	}
	r.mu.Lock()
	_, defeated, _, ok := r.dungeonMgr.DefeatNPCNoLock(songID)
	// Direct objective simulation bypasses ProcessCombatTick's combat cleanup.
	finishedPlayers, _ := r.loadPlayers()
	r.clearCombatNoLock(finishedPlayers[playerID])
	_ = r.savePlayers(finishedPlayers)
	r.mu.Unlock()
	if !ok || !defeated.Defeated || !defeated.Alive {
		t.Fatalf("song que should yield at quarter health and stay alive, got %+v", defeated)
	}
	if inst.Progress != 100 {
		t.Fatalf("expected 100%% after three-trial completion, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}

	// 结算：奖励 + 通关旗标 + 残页
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave dungeon failed: %v", err)
	}
	players, _ := r.loadPlayers()
	p := players[playerID]
	if p.Backpack["xiaohuandan"] != 1 {
		t.Fatalf("expected 1 tiandao page, got %d", p.Backpack["xiaohuandan"])
	}
	if !p.Flags["dungeon_cleared_datang_tiandao"] {
		t.Fatal("expected dungeon_cleared_datang_tiandao flag")
	}

	if p.Skills["tiandao_blade"] != 0 || p.Backpack["manual_tiandao_blade"] != 0 {
		t.Fatal("early trial still grants absolute art")
	}
}
