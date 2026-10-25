package manager

import (
	"testing"
)

// 装备强化：银两+铜片→等级提升→属性叠加
func TestEnhanceItemFlow(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("smith2", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("smith2", "强化师", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	setup := func(money int, bronze int) {
		r.mu.Lock()
		players, _ := r.loadPlayers()
		p := players[playerID]
		p.Money = money
		p.Backpack["iron_sword"] = 1
		p.Backpack["bronze_fragment"] = bronze
		if err := r.savePlayers(players); err != nil {
			r.mu.Unlock()
			t.Fatalf("save failed: %v", err)
		}
		r.mu.Unlock()
	}

	// 银两不足 → 拒绝
	setup(100, 5)
	if _, err := r.EnhanceItem(playerID, "iron_sword"); err == nil {
		t.Fatal("enhance with insufficient silver must fail")
	}

	// 银两+材料充足 → 强化到 1 级
	setup(1000, 5)
	msg, err := r.EnhanceItem(playerID, "iron_sword")
	if err != nil {
		t.Fatalf("enhance failed: %v", err)
	}
	if msg == "" {
		t.Fatal("expected success message")
	}
	after, _ := r.loadPlayers()
	p := after[playerID]
	if p.EnhancedItems["iron_sword"] != 1 {
		t.Fatalf("expected enhancement level 1, got %d", p.EnhancedItems["iron_sword"])
	}
	if p.Money != 800 {
		t.Fatalf("expected money 1000-200=800, got %d", p.Money)
	}
	if p.Backpack["bronze_fragment"] != 4 {
		t.Fatalf("expected bronze 5-1=4, got %d", p.Backpack["bronze_fragment"])
	}

	// 强化到 5 级后 → 再强化被拒
	for i := 0; i < 4; i++ {
		setup(5000, 10)
		if _, err := r.EnhanceItem(playerID, "iron_sword"); err != nil {
			t.Fatalf("enhance #%d failed: %v", i+2, err)
		}
	}
	setup(10000, 10)
	if _, err := r.EnhanceItem(playerID, "iron_sword"); err == nil {
		t.Fatal("enhance at max level must fail")
	}
}
