package manager

import (
	"strings"
	"testing"
)

func TestDungeonActionsDistinguishSuccessRepeatBlockedAndRetryableChoice(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("action_user", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("action_user", "action_hero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	r.mu.Lock()
	inst, err := r.dungeonMgr.CreateInstanceNoLock(playerID, "xunqin_ch01_bronze_fall", "ly_00")
	if err != nil {
		r.mu.Unlock()
		t.Fatalf("create dungeon failed: %v", err)
	}
	players, err := r.loadPlayers()
	if err != nil {
		r.mu.Unlock()
		t.Fatalf("load players failed: %v", err)
	}
	players[playerID].RoomID = inst.SpawnRoomID
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save player failed: %v", err)
	}
	r.mu.Unlock()

	first, err := r.DoRoomAction(playerID, "inspect_ruins")
	if err != nil || first["actionStatus"] != "success" {
		t.Fatalf("expected first observation to succeed, result=%+v err=%v", first, err)
	}
	repeat, err := r.DoRoomAction(playerID, "inspect_ruins")
	if err != nil || repeat["actionStatus"] != "repeat" || !strings.Contains(repeat["message"].(string), "无需重复") {
		t.Fatalf("expected second observation to be a repeat, result=%+v err=%v", repeat, err)
	}

	r.mu.Lock()
	var puzzleRoomID string
	for roomID := range inst.Rooms {
		if strings.HasSuffix(roomID, "_xqj01_06") {
			puzzleRoomID = roomID
			break
		}
	}
	players, _ = r.loadPlayers()
	players[playerID].RoomID = puzzleRoomID
	inst.PuzzleStates["seven_states"] = "choosing"
	inst.PendingChoices = map[string]bool{"seven_correct": true, "seven_wrong_east": true, "seven_wrong_circle": true}
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save puzzle room failed: %v", err)
	}
	r.mu.Unlock()

	wrong, err := r.DoRoomAction(playerID, "seven_wrong_east")
	if err != nil || wrong["actionStatus"] != "blocked" {
		t.Fatalf("expected wrong answer to be retryable blocked result, result=%+v err=%v", wrong, err)
	}
	choices, ok := wrong["choices"].([]map[string]string)
	if !ok || len(choices) != 3 {
		t.Fatalf("expected all three choices after wrong answer, got %+v", wrong["choices"])
	}

	correct, err := r.DoRoomAction(playerID, "seven_correct")
	if err != nil || correct["actionStatus"] != "success" || correct["puzzleState"] != "solved" {
		t.Fatalf("expected correct retry to solve puzzle, result=%+v err=%v", correct, err)
	}
	if len(inst.PendingChoices) != 0 {
		t.Fatalf("expected successful choice to clear pending choices, got %+v", inst.PendingChoices)
	}
}
