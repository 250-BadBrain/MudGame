package manager

import (
	"testing"
)

// 回春堂炼药：材料齐 → 金创药到手、材料扣除；材料不足 → 选项不可用
func TestHerbalBrewDialogue(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("brewer", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("brewer", "炼药师", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	r.mu.Lock()
	players0, _ := r.loadPlayers()
	players0[playerID].RoomID = "ly_herbal_hall"
	if err := r.savePlayers(players0); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	give := func(items map[string]int) {
		t.Helper()
		r.mu.Lock()
		players, _ := r.loadPlayers()
		p := players[playerID]
		for id, n := range items {
			p.Backpack[id] += n
		}
		if err := r.savePlayers(players); err != nil {
			r.mu.Unlock()
			t.Fatalf("save failed: %v", err)
		}
		r.mu.Unlock()
	}

	// 材料不足：两株草 → 主配方选项不可用
	give(map[string]int{"herb": 2})
	if _, _, _, _, err := r.ChooseDialogue(playerID, "herbalist_sun", "brew_with_herbs"); err == nil {
		t.Fatal("brew with insufficient herbs must be rejected")
	}

	// 材料齐：3 草 + 1 粮 → 1 金创药
	give(map[string]int{"herb": 1, "dry_ration": 1})
	if _, _, _, _, err := r.ChooseDialogue(playerID, "herbalist_sun", "brew_with_herbs"); err != nil {
		t.Fatalf("brew failed: %v", err)
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.Backpack["simple_medicine"] != 1 {
		t.Fatalf("expected 1 simple_medicine, got %d", p.Backpack["simple_medicine"])
	}
	if p.Backpack["herb"] != 0 || p.Backpack["dry_ration"] != 0 {
		t.Fatalf("expected materials consumed, got herb=%d ration=%d", p.Backpack["herb"], p.Backpack["dry_ration"])
	}

	// 收草换粮：2 草 → 1 干粮
	give(map[string]int{"herb": 2})
	if _, _, _, _, err := r.ChooseDialogue(playerID, "herbalist_sun", "sell_herbs"); err != nil {
		t.Fatalf("sell herbs failed: %v", err)
	}
	after, _ = r.loadPlayers()
	if after[playerID].Backpack["dry_ration"] != 1 {
		t.Fatalf("expected 1 ration from selling herbs, got %d", after[playerID].Backpack["dry_ration"])
	}
}
