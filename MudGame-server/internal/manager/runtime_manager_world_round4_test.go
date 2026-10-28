package manager

import (
	"testing"
)

// 钟铁衣熔铸：银两+旧武器 → 钢刀；钱不够时选项被过滤
func TestBlacksmithReforgeMoney(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("smith", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("smith", "换刀客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	setup := func(money int, items map[string]int) {
		t.Helper()
		r.mu.Lock()
		players, _ := r.loadPlayers()
		p := players[playerID]
		p.RoomID = "ly_east_market"
		p.Money = money
		for id, n := range items {
			p.Backpack[id] += n
		}
		if err := r.savePlayers(players); err != nil {
			r.mu.Unlock()
			t.Fatalf("save failed: %v", err)
		}
		r.mu.Unlock()
	}

	// 钱不够（初始 50 两）→ 熔铸被拒
	setup(50, map[string]int{"iron_sword": 1})
	if _, _, _, _, err := r.ChooseDialogue(playerID, "heluo_blacksmith", "reforge_sword"); err == nil {
		t.Fatal("reforge must be rejected with insufficient money")
	}

	// 银两齐 → 钢刀到手，旧棍与银两扣除（换用铁棍配方，避免与上一段背包叠加）
	setup(500, map[string]int{"iron_staff": 1})
	if _, _, _, _, err := r.ChooseDialogue(playerID, "heluo_blacksmith", "reforge_staff"); err != nil {
		t.Fatalf("reforge failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["steel_blade"] != 1 || p.Backpack["iron_staff"] != 0 {
		t.Fatalf("expected steel_blade=1 iron_staff=0, got %v", p.Backpack)
	}
	if p.Money != 350 {
		t.Fatalf("expected 500-150 copper, got %d", p.Money)
	}
}

// 洛水风浪全流程：清剿水匪 → 救漕工 → 斩头目 → 100% 结算 + 盐帮声望
func TestLiaoshuiLairFullFlow(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "liaoshui_lair")

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

	// 外围与栅门的水匪
	outer := room("ls_00")
	defeatAll(outer, "ls_pirate", 2)
	gate := room("ls_01")
	moveDungeonTestPlayer(t, r, playerID, gate)
	defeatAll(gate, "ls_pirate", 2)

	// 快船：探问 → 救人
	deck := room("ls_02")
	moveDungeonTestPlayer(t, r, playerID, deck)
	for _, action := range []string{"listen_captive", "free_captives"} {
		if _, err := r.DoRoomAction(playerID, action); err != nil {
			t.Fatalf("action %s failed: %v", action, err)
		}
	}
	if inst.PuzzleStates["captives"] != "freed" {
		t.Fatalf("expected captives freed, got %s", inst.PuzzleStates["captives"])
	}

	// 斩头目
	chiefID := dungeonNPCIDByTemplateInRoom(t, inst, deck, "ls_chief")
	r.mu.Lock()
	_, _, _, ok := r.dungeonMgr.DefeatNPCNoLock(chiefID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("defeat chief failed")
	}
	if inst.Progress != 100 {
		t.Fatalf("expected 100%%, got %d (%+v)", inst.Progress, inst.ObjectiveProgress)
	}

	// 结算：声望与通关旗标
	if _, err := r.LeaveDungeon(playerID); err != nil {
		t.Fatalf("leave dungeon failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Reputations["salt_gang"] < 20 {
		t.Fatalf("expected salt_gang reputation +20, got %v", p.Reputations)
	}
	if !p.Flags["dungeon_cleared_liaoshui_lair"] {
		t.Fatal("expected dungeon_cleared_liaoshui_lair flag")
	}
}

// 回春堂：材料价值高于成品，诊金按现货价格收取
func TestHerbalPillAndConsultFee(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("alchemist", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("alchemist", "丹客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	r.mu.Lock()
	players, _ := r.loadPlayers()
	p := players[playerID]
	p.RoomID = "ly_herbal_hall"
	p.Backpack["herb"] = 15
	p.Backpack["dry_ration"] = 4
	p.Backpack["bronze_fragment"] = 1
	p.Money = 120
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	// 炼丹
	if _, _, _, _, err := r.ChooseDialogue(playerID, "herbalist_sun", "brew_herb_pill"); err != nil {
		t.Fatalf("brew pill failed: %v", err)
	}
	// 诊金
	if _, _, _, _, err := r.ChooseDialogue(playerID, "herbalist_sun", "consult_fee"); err != nil {
		t.Fatalf("consult failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p = after[playerID]
	if p.Backpack["herb_pill"] != 1 || p.Backpack["simple_medicine"] != 1 {
		t.Fatalf("expected herb_pill=1 simple_medicine=1, got %+v", p.Backpack)
	}
	if p.Money != 0 {
		t.Fatalf("expected money drained, got %d", p.Money)
	}
}
