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
			ID:            "p1",
			UserID:        "u1",
			CharacterName: "hero",
			RoomID:        "ly_00",
			State:         "IDLE",
			Money:         123,
			Attributes: model.Attributes{
				Resources: model.ResourceAttributes{HP: 88, MaxHP: 100},
			},
			Skills:    map[string]int{"basic_unarmed": 2},
			Backpack:  map[string]int{"iron_pickaxe": 1},
			Equipment: map[string]string{},
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
