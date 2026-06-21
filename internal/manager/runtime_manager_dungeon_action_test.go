package manager

import (
	"strings"
	"testing"

	engruntime "mudgame/server/internal/engine/runtime"
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

func TestBronzeFallDialogueGatesNPCOutcomeHiddenExitAndOneWayVision(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("bronze_user", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("bronze_user", "bronze_hero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	r.mu.Lock()
	inst, err := r.dungeonMgr.CreateInstanceNoLock(playerID, "xunqin_ch01_bronze_fall", "ly_00")
	if err != nil {
		r.mu.Unlock()
		t.Fatalf("create dungeon failed: %v", err)
	}
	players, _ := r.loadPlayers()
	players[playerID].RoomID = inst.SpawnRoomID
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save player failed: %v", err)
	}
	r.mu.Unlock()

	snapshot, _, err := r.GetRoomSnapshot(playerID)
	if err != nil {
		t.Fatalf("get initial room failed: %v", err)
	}
	if _, visible := snapshot["exits"].(map[string]map[string]interface{})["east"]; visible {
		t.Fatalf("expected east exit hidden before entry story, got %+v", snapshot["exits"])
	}
	if _, err := r.Move(playerID, "east"); err == nil {
		t.Fatal("expected keyboard movement through hidden east exit to be blocked")
	}
	if actions := snapshot["actions"].([]string); testStringSliceContains(actions, "question_raider_choice") {
		t.Fatalf("expected wounded-raider decision to live in dialogue, got actions %+v", actions)
	}
	if _, err := r.DoRoomAction(playerID, "inspect_ruins"); err != nil {
		t.Fatalf("inspect ruins failed: %v", err)
	}
	woundedID := dungeonNPCIDByTemplateInRoom(t, inst, inst.SpawnRoomID, "xqj_wounded_raider")
	dialogue, err := r.Talk(playerID, woundedID)
	if err != nil || !dialogueHasOption(dialogue, "dungeon_action:question_raider_choice") {
		t.Fatalf("expected disposal option after observation, dialogue=%+v err=%v", dialogue, err)
	}
	_, firstMessage, _, firstEffect, err := r.ChooseDialogue(playerID, woundedID, "dungeon_action:question_raider_choice")
	if err != nil || len(firstEffect["choices"].([]map[string]string)) != 3 {
		t.Fatalf("expected first disposal prompt, message=%s effect=%+v err=%v", firstMessage, firstEffect, err)
	}
	_, secondMessage, _, secondEffect, err := r.ChooseDialogue(playerID, woundedID, "dungeon_action:question_raider_choice")
	if err != nil || secondMessage != "受伤盗墓贼连声求饶。" || len(secondEffect["choices"].([]map[string]string)) != 3 {
		t.Fatalf("expected repeated prompt rather than completed text, message=%s effect=%+v err=%v", secondMessage, secondEffect, err)
	}
	if _, err := r.DoRoomAction(playerID, "spare_raider"); err != nil {
		t.Fatalf("spare raider failed: %v", err)
	}
	snapshot, _, _ = r.GetRoomSnapshot(playerID)
	if _, visible := snapshot["exits"].(map[string]map[string]interface{})["east"]; !visible {
		t.Fatalf("expected east exit visible after resolving wounded raider, got %+v", snapshot["exits"])
	}
	for _, npc := range snapshot["npcs"].([]map[string]interface{}) {
		if npc["name"] == "受伤盗墓贼" {
			t.Fatalf("expected spared raider removed from room, got %+v", npc)
		}
	}

	r.mu.Lock()
	visionRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj01_08")
	players, _ = r.loadPlayers()
	players[playerID].RoomID = visionRoom
	inst.PuzzleStates["vision"] = "vision_seen"
	inst.PendingChoices = map[string]bool{"choose_wu_horse": true, "choose_qin_flag": true, "choose_zhao_city": true}
	_ = r.savePlayers(players)
	r.mu.Unlock()
	if _, err := r.DoRoomAction(playerID, "choose_wu_horse"); err != nil {
		t.Fatalf("choose vision destination failed: %v", err)
	}
	status, _ := r.GetPlayerStatus(playerID)
	if !strings.HasSuffix(status["roomId"].(string), "_xqj01_09") {
		t.Fatalf("expected forced move to Zhao slope, got %+v", status)
	}
	snapshot, _, _ = r.GetRoomSnapshot(playerID)
	if _, visible := snapshot["exits"].(map[string]map[string]interface{})["west"]; visible {
		t.Fatalf("expected bronze vision disconnected behind player, got %+v", snapshot["exits"])
	}

	r.mu.Lock()
	lastRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj01_12")
	players, _ = r.loadPlayers()
	players[playerID].RoomID = lastRoom
	_ = r.savePlayers(players)
	r.mu.Unlock()
	taoID := dungeonNPCIDByTemplateInRoom(t, inst, lastRoom, "xqj_tao_fang")
	dialogue, _ = r.Talk(playerID, taoID)
	if dialogueHasOption(dialogue, "dungeon_action:invite_wu_fort") {
		t.Fatalf("expected Tao Fang invitation hidden while enemies live, got %+v", dialogue)
	}
	r.mu.Lock()
	for _, npc := range inst.NPCs {
		if npc.RoomID == lastRoom && (npc.TemplateID == "xqj_bandit_boss" || npc.TemplateID == "xqj_bandit") {
			npc.Alive = false
			npc.Attributes.Resources.HP = 0
		}
	}
	r.mu.Unlock()
	dialogue, _ = r.Talk(playerID, taoID)
	if !dialogueHasOption(dialogue, "dungeon_action:invite_wu_fort") {
		t.Fatalf("expected Tao Fang invitation after both enemies fall, got %+v", dialogue)
	}
}

func dungeonNPCIDByTemplateInRoom(t *testing.T, inst *engruntime.Instance, roomID, templateID string) string {
	t.Helper()
	for id, candidate := range inst.NPCs {
		if candidate.RoomID == roomID && candidate.TemplateID == templateID {
			return id
		}
	}
	t.Fatalf("missing dungeon npc template %s", templateID)
	return ""
}

func dialogueHasOption(dialogue map[string]interface{}, optionID string) bool {
	options, _ := dialogue["options"].([]map[string]interface{})
	for _, option := range options {
		if option["id"] == optionID {
			return true
		}
	}
	return false
}

func testStringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
