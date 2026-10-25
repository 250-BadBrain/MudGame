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

// 禁沟烽燧搜索：一次性断讯凭据旗标（烽燧疑警任务目标）
func TestBeaconSearchSetsSignalFlag(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("scout", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("scout", "斥候客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].RoomID = "tg_jingou_beacon"
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	if _, err := r.DoRoomAction(playerID, "search"); err != nil {
		t.Fatalf("search failed: %v", err)
	}
	after, _ := r.loadPlayers()
	if !after[playerID].Flags["found_beacon_signal"] {
		t.Fatal("expected found_beacon_signal flag")
	}
}

// 跨城：enter_world 直达长安出生点，再切回洛阳
func TestEnterWorldChangan(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("traveler", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("traveler", "两京客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	if _, err := r.EnterWorld(playerID, "changan"); err != nil {
		t.Fatalf("enter changan failed: %v", err)
	}
	after, _ := r.loadPlayers()
	if after[playerID].RoomID != "ca2_west_market" {
		t.Fatalf("expected changan spawn, got %s", after[playerID].RoomID)
	}

	if _, err := r.EnterWorld(playerID, "luoyang"); err != nil {
		t.Fatalf("enter luoyang failed: %v", err)
	}
	after, _ = r.loadPlayers()
	if after[playerID].RoomID != "ly_00" {
		t.Fatalf("expected luoyang spawn, got %s", after[playerID].RoomID)
	}
}

// 流民疑云：接取 → 哑婆旗标 → 交付（对话数据闭环）
func TestRefugeeRumorQuest(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("rumorhunter", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("rumorhunter", "探谣客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].RoomID = "ly_refugee_camp"
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()

	// 接取
	if _, _, _, _, err := r.ChooseDialogue(playerID, "old_refugee", "start_rumor_case"); err != nil {
		t.Fatalf("start quest failed: %v", err)
	}
	after, _ := r.loadPlayers()
	if after[playerID].QuestStates["refugee_rumor_case"].Status != "active" {
		t.Fatal("expected quest active")
	}
	// 打听旗标
	if _, _, _, _, err := r.ChooseDialogue(playerID, "old_refugee", "gather_rumor"); err != nil {
		t.Fatalf("gather rumor failed: %v", err)
	}
	after, _ = r.loadPlayers()
	if !after[playerID].Flags["heard_refugee_whisper"] {
		t.Fatal("expected whisper flag")
	}
	// 交付
	if _, _, logs, _, err := r.ChooseDialogue(playerID, "old_refugee", "turn_in_rumor"); err != nil {
		t.Fatalf("turn in failed: %v", err)
	} else if len(logs) == 0 {
		t.Fatal("expected quest completion logs")
	}
	after, _ = r.loadPlayers()
	p := after[playerID]
	if p.QuestStates["refugee_rumor_case"].Status != "completed" {
		t.Fatal("expected quest completed")
	}
	if p.Backpack["smoke_powder"] != 1 {
		t.Fatalf("expected smoke powder reward, got %d", p.Backpack["smoke_powder"])
	}
}

// R259 马车费：跨城需 100 两
func TestEnterWorldCarriageFee(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("carriage", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("carriage", "马车客", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	// 银两不足 → 拒绝
	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].Money = 50
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()
	if _, err := r.EnterWorld(playerID, "changan"); err == nil {
		t.Fatal("enter world with insufficient silver must fail")
	}

	// 银两充足 → 成功 + 扣费
	r.mu.Lock()
	players, _ = r.loadPlayers()
	players[playerID].Money = 200
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save failed: %v", err)
	}
	r.mu.Unlock()
	if _, err := r.EnterWorld(playerID, "changan"); err != nil {
		t.Fatalf("enter world with sufficient silver must pass: %v", err)
	}
	after, _ := r.loadPlayers()
	if after[playerID].Money != 100 {
		t.Fatalf("expected money 200-100=100, got %d", after[playerID].Money)
	}
	if after[playerID].RoomID != "ca2_west_market" {
		t.Fatalf("expected changan spawn, got %s", after[playerID].RoomID)
	}
}
