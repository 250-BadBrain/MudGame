package storage

import (
	"os"
	"path/filepath"
	"testing"

	"mudgame/server/internal/model"
)

func TestFileStoreUsersRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	users := map[string]*model.User{
		"u1": {
			Username:           "u1",
			PasswordHash:       "h1",
			CharacterIDs:       []string{"p1"},
			ReconnectTokenHash: "hashed-token-1",
		},
	}
	if err := store.SaveUsers(users); err != nil {
		t.Fatalf("save users failed: %v", err)
	}

	loaded, err := store.LoadUsers()
	if err != nil {
		t.Fatalf("load users failed: %v", err)
	}
	if len(loaded) != 1 || loaded["u1"] == nil {
		t.Fatalf("unexpected users payload: %+v", loaded)
	}
	if loaded["u1"].ReconnectTokenHash != "hashed-token-1" {
		t.Fatalf("unexpected reconnect token hash: %s", loaded["u1"].ReconnectTokenHash)
	}
}

func TestFileStorePlayersAndRoomsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	players := map[string]*model.Player{
		"p1": {
			ID:                    "p1",
			UserID:                "u1",
			CharacterName:         "hero",
			RoomID:                "ly_00",
			State:                 "LEARNING",
			StateTickCountdown:    7,
			LearningSkillID:       "basic_unarmed",
			LearningTeacherID:     "weapon_school_master",
			LearningTeacherLevel:  300,
			SkillLearningProgress: map[string]int{"basic_unarmed": 2},
			Money:                 123,
			Attributes: model.Attributes{
				Resources: model.ResourceAttributes{HP: 88, MaxHP: 100},
			},
			Skills:          map[string]int{"basic_unarmed": 2},
			Backpack:        map[string]int{"iron_pickaxe": 1},
			Equipment:       map[string]string{},
			DialogueHistory: map[string]bool{"storyteller|opening": true},
		},
	}
	if err := store.SavePlayers(players); err != nil {
		t.Fatalf("save players failed: %v", err)
	}

	loadedPlayers, err := store.LoadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	if len(loadedPlayers) != 1 || loadedPlayers["p1"] == nil {
		t.Fatalf("unexpected players payload: %+v", loadedPlayers)
	}
	if loadedPlayers["p1"].Backpack["iron_pickaxe"] != 1 {
		t.Fatalf("unexpected backpack payload: %+v", loadedPlayers["p1"].Backpack)
	}
	loadedPlayer := loadedPlayers["p1"]
	if loadedPlayer.State != "LEARNING" || loadedPlayer.StateTickCountdown != 7 || loadedPlayer.LearningSkillID != "basic_unarmed" || loadedPlayer.LearningTeacherID != "weapon_school_master" || loadedPlayer.LearningTeacherLevel != 300 || loadedPlayer.SkillLearningProgress["basic_unarmed"] != 2 {
		t.Fatalf("unexpected learning state payload: %+v", loadedPlayer)
	}
	if !loadedPlayer.DialogueHistory["storyteller|opening"] {
		t.Fatalf("expected dialogue progress to survive reload, got %+v", loadedPlayer.DialogueHistory)
	}

	rooms := map[string]RoomDynamic{
		"ly_00": {NPCs: []string{"npc_a"}, Players: []string{"p1"}},
	}
	if err := store.SaveDynamicRooms(rooms); err != nil {
		t.Fatalf("save rooms failed: %v", err)
	}
	loadedRooms, err := store.LoadDynamicRooms()
	if err != nil {
		t.Fatalf("load rooms failed: %v", err)
	}
	if len(loadedRooms) != 1 || len(loadedRooms["ly_00"].Players) != 1 {
		t.Fatalf("unexpected rooms payload: %+v", loadedRooms)
	}

	if err := os.WriteFile(filepath.Join(dir, "rooms", "dg_abandoned_temple_rain_deadbeef_temple_00.json"), []byte(`{"npcs":["wolf"],"players":["p2"]}`), 0o644); err != nil {
		t.Fatalf("seed dungeon room file failed: %v", err)
	}
	loadedRooms, err = store.LoadDynamicRooms()
	if err != nil {
		t.Fatalf("reload rooms failed: %v", err)
	}
	if _, ok := loadedRooms["dg_abandoned_temple_rain_deadbeef_temple_00"]; ok {
		t.Fatalf("dungeon room file should be ignored: %+v", loadedRooms)
	}
	if err := store.SaveDynamicRooms(loadedRooms); err != nil {
		t.Fatalf("save cleaned rooms failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "rooms", "dg_abandoned_temple_rain_deadbeef_temple_00.json")); !os.IsNotExist(err) {
		t.Fatalf("expected dungeon room file to be removed, got err=%v", err)
	}
}

func TestFileStoreLoadPlayersSanitizesDungeonRoomIDs(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	if err := os.MkdirAll(filepath.Join(dir, "players"), 0o755); err != nil {
		t.Fatalf("mkdir players failed: %v", err)
	}
	raw := `{"id":"p1","characterName":"hero","userId":"u1","roomId":"dg_abandoned_temple_rain_1234abcd_temple_00","state":"IDLE"}`
	if err := os.WriteFile(filepath.Join(dir, "players", "p1.json"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write player failed: %v", err)
	}

	loaded, err := store.LoadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	if loaded["p1"].RoomID != "ly_00" {
		t.Fatalf("expected dungeon room to sanitize to ly_00, got %q", loaded["p1"].RoomID)
	}
}

func TestFileStoreSavesReplaceStaleFilesAndLeaveNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	users := map[string]*model.User{
		"u1": {Username: "u1", PasswordHash: "h1", CharacterIDs: []string{}},
		"u2": {Username: "u2", PasswordHash: "h2", CharacterIDs: []string{}},
	}
	if err := store.SaveUsers(users); err != nil {
		t.Fatalf("save users failed: %v", err)
	}
	if err := store.SaveUsers(map[string]*model.User{"u1": users["u1"]}); err != nil {
		t.Fatalf("resave users failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "users", "u2.json")); !os.IsNotExist(err) {
		t.Fatalf("expected stale user file to be removed, got err=%v", err)
	}

	players := map[string]*model.Player{
		"p1": {ID: "p1", UserID: "u1", CharacterName: "hero", Money: 100},
		"p2": {ID: "p2", UserID: "u1", CharacterName: "hero2", Money: 200},
	}
	if err := store.SavePlayers(players); err != nil {
		t.Fatalf("save players failed: %v", err)
	}
	if err := store.SavePlayers(map[string]*model.Player{"p1": players["p1"]}); err != nil {
		t.Fatalf("resave players failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "players", "p2.json")); !os.IsNotExist(err) {
		t.Fatalf("expected stale player file to be removed, got err=%v", err)
	}

	// 覆盖写不应破坏既有内容（tmp+rename 语义）
	players["p1"].Money = 777
	if err := store.SavePlayers(players); err != nil {
		t.Fatalf("overwrite player failed: %v", err)
	}
	loaded, err := store.LoadPlayers()
	if err != nil {
		t.Fatalf("load players failed: %v", err)
	}
	if loaded["p1"].Money != 777 {
		t.Fatalf("expected overwritten money 777, got %d", loaded["p1"].Money)
	}

	for _, sub := range []string{"users", "players", "rooms"} {
		entries, err := os.ReadDir(filepath.Join(dir, sub))
		if err != nil {
			t.Fatalf("read dir %s failed: %v", sub, err)
		}
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".tmp" {
				t.Fatalf("unexpected temp file leftover: %s/%s", sub, entry.Name())
			}
		}
	}
}

func TestFileStoreQuarantinesCorruptFiles(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	// 正常保存一名玩家，再注入一个损坏的玩家文件
	if err := store.SavePlayers(map[string]*model.Player{
		"p_ok": {ID: "p_ok", CharacterName: "ok", UserID: "u1"},
	}); err != nil {
		t.Fatalf("save players failed: %v", err)
	}
	corruptPath := filepath.Join(dir, "players", "p_bad.json")
	if err := os.WriteFile(corruptPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write corrupt file failed: %v", err)
	}

	loaded, err := store.LoadPlayers()
	if err != nil {
		t.Fatalf("load players should tolerate corrupt files, got error: %v", err)
	}
	if _, ok := loaded["p_ok"]; !ok {
		t.Fatalf("healthy player should still load: %+v", loaded)
	}
	if _, ok := loaded["p_bad"]; ok {
		t.Fatalf("corrupt player should not load")
	}
	if _, err := os.Stat(corruptPath + ".corrupt"); err != nil {
		t.Fatalf("corrupt file should be quarantined: %v", err)
	}
	if _, err := os.Stat(corruptPath); err == nil {
		t.Fatalf("corrupt file should no longer sit in load path")
	}
}
