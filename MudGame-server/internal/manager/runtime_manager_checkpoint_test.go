package manager

import "testing"

func TestReliveReturnsDungeonPlayerToCheckpoint(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("checkpoint_user", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("checkpoint_user", "checkpoint_hero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	r.mu.Lock()
	inst, err := r.dungeonMgr.CreateInstanceNoLock(playerID, "abandoned_temple_rain", "ly_00")
	if err != nil {
		r.mu.Unlock()
		t.Fatalf("create dungeon instance failed: %v", err)
	}
	checkpoint := inst.SpawnRoomID
	for roomID := range inst.Rooms {
		if roomID != inst.SpawnRoomID {
			checkpoint = roomID
			break
		}
	}
	inst.CheckpointRoomID = checkpoint
	players, err := r.loadPlayers()
	if err != nil {
		r.mu.Unlock()
		t.Fatalf("load players failed: %v", err)
	}
	players[playerID].State = "DEAD"
	players[playerID].RoomID = inst.SpawnRoomID
	players[playerID].Attributes.Resources.HP = 0
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save dead player failed: %v", err)
	}
	r.mu.Unlock()

	if _, err := r.Relive(playerID); err != nil {
		t.Fatalf("relive failed: %v", err)
	}
	status, err := r.GetPlayerStatus(playerID)
	if err != nil {
		t.Fatalf("get player status failed: %v", err)
	}
	if status["roomId"] != checkpoint {
		t.Fatalf("expected checkpoint %s after relive, got %+v", checkpoint, status)
	}
}
