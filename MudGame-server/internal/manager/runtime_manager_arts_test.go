package manager

import (
	"testing"
)

// 慈航剑典：秘本参悟与灰袍传授共用同一道传承门，入门后装备生效
func TestCihangInsightAndTeaching(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("swordseeker", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("swordseeker", "求剑客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].RoomID = "ly_old_alley" // 灰袍老者所在
	players[playerID].Skills["basic_sword"] = 120
	players[playerID].Skills["basic_literacy"] = 80
	players[playerID].Attributes.Experience = 200000 // 慈航剑典前置门槛
	players[playerID].Backpack["manual_cihang_jiandian"] = 1
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	// 未入门 → 参悟被拒且秘本保留
	if _, err := r.UseItem(playerID, "manual_cihang_jiandian"); err == nil {
		t.Fatal("insight without introduction must fail")
	}
	after, _ := r.loadPlayers()
	if after[playerID].Backpack["manual_cihang_jiandian"] != 1 {
		t.Fatal("failed insight consumed book")
	}
	// 灰袍传授入门 → 1 级并留下传承旗标
	if err := r.LearnSkill(playerID, "ancient_scroll_keeper", "cihang_jiandian"); err != nil {
		t.Fatalf("learn from keeper failed: %v", err)
	}
	after, _ = r.loadPlayers()
	if after[playerID].Skills["cihang_jiandian"] != 1 {
		t.Fatalf("expected cihang_jiandian level 1, got %d", after[playerID].Skills["cihang_jiandian"])
	}
	// 已入门后再参悟完整秘本 → 重复参悟被拒，秘本保留
	if _, err := r.UseItem(playerID, "manual_cihang_jiandian"); err == nil {
		t.Fatal("duplicate insight must fail")
	}
	after, _ = r.loadPlayers()
	if after[playerID].Backpack["manual_cihang_jiandian"] != 1 {
		t.Fatal("duplicate insight consumed book")
	}
	// 入门后即可装备
	if _, err := r.EquipSkill(playerID, "cihang_jiandian"); err != nil {
		t.Fatalf("equip cihang failed: %v", err)
	}
	after, _ = r.loadPlayers()
	if after[playerID].ActiveArts["combat"] != "cihang_jiandian" {
		t.Fatalf("expected cihang equipped, got %+v", after[playerID].ActiveArts)
	}
}

// 宋缺宗师亲传：defeat 模式 Boss 败而不死，玩家在其房内可请教天刀八诀（上限 320）
func TestEarlySongQueCannotBypassLateTransmission(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_tiandao")
	arenaRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "td_04")
	moveDungeonTestPlayer(t, r, playerID, arenaRoom)
	songID := dungeonNPCIDByTemplateInRoom(t, inst, arenaRoom, "td_song_que")

	// 击败宋缺（defeat 模式：留 1/4 气血，留在房内不死）
	r.mu.Lock()
	_, defeated, _, ok := r.dungeonMgr.DefeatNPCNoLock(songID)
	r.mu.Unlock()
	if !ok || !defeated.Alive {
		t.Fatal("song que should remain after defeat")
	}

	// Early Song Que remains a valid story encounter but is not the late transmission.
	if err := r.LearnSkill(playerID, songID, "tiandao_blade"); err == nil {
		t.Fatal("early encounter bypassed late transmission")
	}
	after, _ := r.loadPlayers()
	if after[playerID].Skills["tiandao_blade"] != 0 {
		t.Fatal("early encounter granted tiandao")
	}
}
