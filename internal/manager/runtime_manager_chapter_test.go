package manager

import (
	"strings"
	"testing"
)

// 战斗中允许使用药品（金创药等是副本战斗核心消耗品）
func TestPotionUsableDuringCombat(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	if err := r.Register("fighter", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("fighter", "用药侠", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	players, err := r.loadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	p := players[playerID]
	p.Backpack["simple_medicine"] = 2
	p.Attributes.Resources.HP = 10 // 需要回血
	p.State = "COMBAT"
	if err := r.savePlayers(players); err != nil {
		t.Fatalf("save players failed: %v", err)
	}

	if _, err := r.UseItem(playerID, "simple_medicine"); err != nil {
		t.Fatalf("potion should be usable in combat, got %v", err)
	}
	reloaded, _ := r.loadPlayers()
	if reloaded[playerID].Attributes.Resources.HP <= 10 {
		t.Fatalf("potion should heal in combat, hp=%d", reloaded[playerID].Attributes.Resources.HP)
	}
}

// 非战斗 NPC 不可被攻击
func TestNonCombatNPCRejectsKill(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	if err := r.Register("pacifist", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("pacifist", "和平客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	// nj_wounded（怒蛟帮伤兵）在副本内：先解锁章节链（order 3 需通关 order 2）
	players, _ := r.loadPlayers()
	players[playerID].Flags["dungeon_cleared_abandoned_temple_rain"] = true
	players[playerID].Flags["dungeon_cleared_datang_changshengjue"] = true
	if err := r.savePlayers(players); err != nil {
		t.Fatalf("save flags failed: %v", err)
	}
	if _, err := r.EnterDungeon(playerID, "fuyu_nujiao"); err != nil {
		t.Fatalf("enter dungeon failed: %v", err)
	}
	players2, _ := r.loadPlayers()
	p := players2[playerID]
	// 伤兵在 dock_03
	var woundedID string
	instanceID, _ := r.dungeonMgr.PlayerInstanceIDNoLock(playerID)
	inst, _ := r.dungeonMgr.DungeonInstanceNoLock(instanceID)
	roomID := r.dungeonMgr.InstanceRoomIDNoLock(inst, "dock_03")
	for _, npcID := range inst.Rooms[roomID].NPCIDs {
		if inst.NPCs[npcID] != nil && inst.NPCs[npcID].TemplateID == "nj_wounded" {
			woundedID = npcID
			break
		}
	}
	if woundedID == "" {
		t.Fatal("wounded npc not found in dock_03")
	}
	p.RoomID = roomID
	if err := r.savePlayers(players2); err != nil {
		t.Fatalf("save players failed: %v", err)
	}

	_, err = r.Kill(playerID, woundedID)
	if err == nil || !strings.Contains(err.Error(), "并无战意") {
		t.Fatalf("expected non-combat rejection, got %v", err)
	}
}

// 章节链：order 2 需通关 order 1；通关后解锁
func TestChapterChainUnlocksNextOrder(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	if err := r.Register("chainer", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("chainer", "链行侠", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	// order 1 可进
	if _, err := r.EnterDungeon(playerID, "abandoned_temple_rain"); err != nil {
		t.Fatalf("order 1 should be open, got %v", err)
	}
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave dungeon failed: %v", err)
	}

	// order 2（扬子江）未通关 order 1 时被拒
	if _, err := r.EnterDungeon(playerID, "datang_changshengjue"); err == nil || !strings.Contains(err.Error(), "需先通关") {
		t.Fatalf("order 2 should be locked before clearing order 1, got %v", err)
	}

	// 列表带锁定信息
	list := r.GetDungeons(playerID)
	var changsheng map[string]interface{}
	for _, d := range list {
		if d["id"] == "datang_changshengjue" {
			changsheng = d
		}
	}
	if changsheng == nil || changsheng["locked"] != true {
		t.Fatalf("order 2 should be marked locked in list, got %+v", changsheng)
	}

	// 通关 order 1 后解锁 order 2
	players, _ := r.loadPlayers()
	players[playerID].Flags["dungeon_cleared_abandoned_temple_rain"] = true
	if err := r.savePlayers(players); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if _, err := r.EnterDungeon(playerID, "datang_changshengjue"); err != nil {
		t.Fatalf("order 2 should unlock after clearing order 1, got %v", err)
	}
}
