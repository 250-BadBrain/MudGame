package manager

import (
	"testing"
)

// 慈航剑典：残页参悟 → 灰袍老者传授；洛水沉箱打捞给残页
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
	players[playerID].RoomID = "ly_weapon_school"
	players[playerID].Skills["basic_sword"] = 120
	players[playerID].Skills["basic_literacy"] = 80
	players[playerID].Flags["dungeon_cleared_changan_trial_hall"] = true
	players[playerID].Attributes.Experience = 200000 // 慈航剑典前置门槛
	players[playerID].Backpack["manual_cihang_jiandian"] = 1
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	// 未参悟 → 灰袍拒绝
	if err := r.LearnSkill(playerID, "ancient_scroll_keeper", "cihang_jiandian"); err == nil {
		t.Fatal("learning cihang without insight must fail")
	}
	// 参悟残页 → 1 级
	if _, err := r.UseItem(playerID, "manual_cihang_jiandian"); err != nil {
		t.Fatalf("insight cihang page failed: %v", err)
	}
	after, _ := r.loadPlayers()
	if after[playerID].Skills["cihang_jiandian"] != 1 {
		t.Fatalf("expected cihang_jiandian level 1, got %d", after[playerID].Skills["cihang_jiandian"])
	}
	// 灰袍传授（上限 120）
	if err := r.LearnSkill(playerID, "ancient_scroll_keeper", "cihang_jiandian"); err != nil {
		t.Fatalf("learn from keeper failed: %v", err)
	}
	after, _ = r.loadPlayers()
	if after[playerID].State != "LEARNING" || after[playerID].LearningSkillID != "cihang_jiandian" {
		t.Fatalf("expected LEARNING cihang, got state=%s skill=%s", after[playerID].State, after[playerID].LearningSkillID)
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
