package manager

import (
	"testing"
)

// 芦苇滩采集走采药表（止血草/干粮），不再产出城郊的古铜残片
func TestReedMarshGatherDropsHerbs(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("herbalist", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("herbalist", "采药人", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	r.mu.Lock()
	players, _ := r.loadPlayers()
	p := players[playerID]
	p.RoomID = "ly_reed_marsh"
	p.State = "GATHERING"
	p.StateTickCountdown = 1
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	if _, err := r.ProcessStatusTick(); err != nil {
		t.Fatalf("status tick failed: %v", err)
	}
	after, _ := r.loadPlayers()
	got := after[playerID]
	if got.State != "IDLE" {
		t.Fatalf("gather should settle to IDLE, got %s", got.State)
	}
	if got.Backpack["herb"]+got.Backpack["dry_ration"] != 1 {
		t.Fatalf("expected exactly one herb or ration, got herb=%d ration=%d", got.Backpack["herb"], got.Backpack["dry_ration"])
	}
	if got.Backpack["bronze_fragment"] > 0 {
		t.Fatal("reed marsh must not drop bronze fragments")
	}
}

// 盐仓渡口搜索：首次拿到私盐凭据旗标（漕运疑云任务目标），重复为空
func TestSaltWharfSearchSetsEvidenceFlag(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("sleuth", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("sleuth", "查案客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].RoomID = "ly_salt_wharf"
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	first, err := r.DoRoomAction(playerID, "search")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if msg, _ := first["message"].(string); len(msg) < 10 {
		t.Fatalf("expected scripted evidence message, got %+v", first)
	}
	after, _ := r.loadPlayers()
	if !after[playerID].Flags["found_smuggling_evidence"] {
		t.Fatal("expected found_smuggling_evidence flag")
	}

	second, err := r.DoRoomAction(playerID, "search")
	if err != nil {
		t.Fatalf("repeat search failed: %v", err)
	}
	if msg, _ := second["message"].(string); msg == first["message"] {
		t.Fatal("repeat search should not replay the evidence event")
	}
}
