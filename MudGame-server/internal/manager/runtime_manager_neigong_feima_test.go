package manager

import (
	"testing"
)

// 混元功：静坐观心得残页 → 参悟 → 装备到基本内功槽 → 打坐恢复实测提升
func TestHunyuanMeditationBonus(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("meditator", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("meditator", "坐忘客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	// 基本内功与混元功在首次基准前就位（未装备时不计入），仅考察装备前后的差值
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].Skills["basic_neigong"] = 100
	grantArt(players[playerID], "hunyuan_neigong", 100)
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	baselineMP := func() int {
		r.mu.Lock()
		players, _ := r.loadPlayers()
		p := players[playerID]
		p.State = "MEDITATING"
		p.StateTickCountdown = 1 // 直接对齐到结算点
		p.Attributes.Resources.TrainedMaxMP = 100
		p.Attributes.Resources.MP = 0
		if err := r.savePlayers(players); err != nil {
			r.mu.Unlock()
			t.Fatalf("save failed: %v", err)
		}
		r.mu.Unlock()
		if _, err := r.ProcessStatusTick(); err != nil {
			t.Fatalf("tick failed: %v", err)
		}
		after, _ := r.loadPlayers()
		return after[playerID].Attributes.Resources.MP
	}

	before := baselineMP()
	players, _ = r.loadPlayers()
	players[playerID].State = "IDLE"
	_ = r.savePlayers(players)

	// 装备混元功（100 级 → 打坐恢复 +10：每 100 级 10）
	if _, err := r.EquipSkill(playerID, "hunyuan_neigong"); err != nil {
		t.Fatalf("equip hunyuan failed: %v", err)
	}

	after := baselineMP()
	if after != before+10 {
		t.Fatalf("expected meditation recovery +10 with hunyuan equipped, got %d -> %d", before, after)
	}
}

// 飞马牧场全流程：刺客/杀手 → 安群哨 → 逼退婠婠 → 玉匣 → 100% 结算
func TestFeimaRanchFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_feima")

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

	stable := room("fm_01")
	moveDungeonTestPlayer(t, r, playerID, stable)
	defeatAll(stable, "fm_yinkui_acolyte", 2)
	field := room("fm_02")
	moveDungeonTestPlayer(t, r, playerID, field)
	defeatAll(field, "fm_yinkui_slasher", 2)

	// 天马厩：问哨 → 吹哨
	paddock := room("fm_03")
	moveDungeonTestPlayer(t, r, playerID, paddock)
	for _, action := range []string{"listen_hand", "calm_horses"} {
		if _, err := r.DoRoomAction(playerID, action); err != nil {
			t.Fatalf("action %s failed: %v", action, err)
		}
	}
	if inst.PuzzleStates["horses"] != "calmed" {
		t.Fatalf("expected horses calmed, got %s", inst.PuzzleStates["horses"])
	}

	// 主宅：逼退婠婠（defeat 模式）+ 拾取玉匣
	courtyard := room("fm_04")
	moveDungeonTestPlayer(t, r, playerID, courtyard)
	wanID := dungeonNPCIDByTemplateInRoom(t, inst, courtyard, "fm_wanwan")
	r.mu.Lock()
	_, defeated, _, ok := r.dungeonMgr.DefeatNPCNoLock(wanID)
	r.mu.Unlock()
	if !ok || !defeated.Alive || !defeated.Defeated {
		t.Fatalf("wanwan should be defeated but alive, got %+v", defeated)
	}
	if inst.Progress != 100 {
		t.Fatalf("expected 100%% without optional relic, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}
	if _, err := r.DoRoomAction(playerID, "retrieve_relic"); err != nil {
		t.Fatalf("retrieve relic failed: %v", err)
	}

	// 结算：奖励与旗标
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave dungeon failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["xiaohuandan"] != 1 {
		t.Fatalf("expected tianmo page, got %d", p.Backpack["xiaohuandan"])
	}
	if !p.Flags["dungeon_cleared_datang_feima"] {
		t.Fatal("expected dungeon_cleared_datang_feima flag")
	}
}
