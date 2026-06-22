package manager

import (
	"reflect"
	"strings"
	"testing"

	engruntime "mudgame/server/internal/engine/runtime"
)

func TestRepeatedPuzzlePromptKeepsOriginalChoiceLabels(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("cipher_user", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("cipher_user", "cipher_hero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	r.mu.Lock()
	inst, err := r.dungeonMgr.CreateInstanceNoLock(playerID, "xunqin_ch03_handan_opening", "ly_00")
	if err != nil {
		r.mu.Unlock()
		t.Fatalf("create dungeon failed: %v", err)
	}
	players, _ := r.loadPlayers()
	player := players[playerID]
	player.RoomID = r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj03_11")
	player.Backpack = map[string]int{
		"xqj_code_sheet": 1,
		"xqj_fish_token": 1,
		"xqj_deer_token": 1,
		"xqj_spy_letter": 1,
	}
	inst.PuzzleStates["cipher"] = "tokens"
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save player failed: %v", err)
	}
	r.mu.Unlock()

	first, err := r.DoRoomAction(playerID, "decode_letter")
	if err != nil {
		t.Fatalf("first decode prompt failed: %v", err)
	}
	second, err := r.DoRoomAction(playerID, "decode_letter")
	if err != nil {
		t.Fatalf("repeated decode prompt failed: %v", err)
	}
	if !reflect.DeepEqual(first["choices"], second["choices"]) {
		t.Fatalf("repeated prompt changed visible labels: first=%+v second=%+v", first["choices"], second["choices"])
	}
}

func TestDungeonNPCDialogueForcesSequentialStoryActions(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("guard_dialogue_user", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("guard_dialogue_user", "guardhero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	r.mu.Lock()
	inst, err := r.dungeonMgr.CreateInstanceNoLock(playerID, "xunqin_ch04_hostage_camp_night", "ly_00")
	if err != nil {
		r.mu.Unlock()
		t.Fatalf("create dungeon failed: %v", err)
	}
	roomID := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj04_05")
	players, _ := r.loadPlayers()
	players[playerID].RoomID = roomID
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save player failed: %v", err)
	}
	r.mu.Unlock()
	guardID := dungeonNPCIDByTemplateInRoom(t, inst, roomID, "xqj_prison_guard")

	blocked, err := r.Talk(playerID, guardID)
	if err != nil || inst.DialogueProgress["show_badge"] != 0 || inst.PuzzleStates["outer_prison"] != "locked" {
		t.Fatalf("expected missing badge to block dialogue progression, result=%+v state=%+v err=%v", blocked, inst.PuzzleStates, err)
	}
	r.mu.Lock()
	players, _ = r.loadPlayers()
	players[playerID].Backpack["xqj_guard_badge"] = 1
	_ = r.savePlayers(players)
	r.mu.Unlock()
	if _, err := r.Talk(playerID, guardID); err != nil {
		t.Fatalf("first badge dialogue line failed: %v", err)
	}
	if inst.PuzzleStates["outer_prison"] != "locked" {
		t.Fatalf("expected first line not to execute badge action, got %s", inst.PuzzleStates["outer_prison"])
	}
	if _, err := r.Talk(playerID, guardID); err != nil {
		t.Fatalf("final badge dialogue line failed: %v", err)
	}
	if inst.PuzzleStates["outer_prison"] != "open" {
		t.Fatalf("expected badge action before key dialogue, got %s", inst.PuzzleStates["outer_prison"])
	}
	if _, err := r.Talk(playerID, guardID); err != nil {
		t.Fatalf("first key dialogue line failed: %v", err)
	}
	if inst.PuzzleStates["outer_prison"] != "open" {
		t.Fatalf("expected first key line not to grant key, got %s", inst.PuzzleStates["outer_prison"])
	}
	if _, err := r.Talk(playerID, guardID); err != nil {
		t.Fatalf("final key dialogue line failed: %v", err)
	}
	if inst.PuzzleStates["outer_prison"] != "key_taken" {
		t.Fatalf("expected second story action after its final line, got %s", inst.PuzzleStates["outer_prison"])
	}
	players, _ = r.loadPlayers()
	if players[playerID].Backpack["xqj_prison_key"] != 1 {
		t.Fatalf("expected key granted after ordered dialogue, backpack=%+v", players[playerID].Backpack)
	}
}

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
	firstLine, err := r.Talk(playerID, woundedID)
	if err != nil || firstLine["message"] == "" || firstLine["choices"] != nil {
		t.Fatalf("expected first dialogue line without choices, result=%+v err=%v", firstLine, err)
	}
	choiceLine, err := r.Talk(playerID, woundedID)
	choices, _ := choiceLine["choices"].([]map[string]string)
	if err != nil || len(choices) != 3 {
		t.Fatalf("expected choices after final dialogue line, result=%+v err=%v", choiceLine, err)
	}
	repeatedChoiceLine, err := r.Talk(playerID, woundedID)
	repeatedChoices, _ := repeatedChoiceLine["choices"].([]map[string]string)
	if err != nil || repeatedChoiceLine["message"] != choiceLine["message"] || !reflect.DeepEqual(repeatedChoices, choices) {
		t.Fatalf("expected repeated final line and choices, first=%+v repeated=%+v err=%v", choiceLine, repeatedChoiceLine, err)
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
	blockedTalk, _ := r.Talk(playerID, taoID)
	if blockedTalk["choices"] != nil {
		t.Fatalf("expected Tao Fang invitation blocked while enemies live, got %+v", blockedTalk)
	}
	r.mu.Lock()
	for _, npc := range inst.NPCs {
		if npc.RoomID == lastRoom && (npc.TemplateID == "xqj_bandit_boss" || npc.TemplateID == "xqj_bandit") {
			npc.Alive = false
			npc.Attributes.Resources.HP = 0
		}
	}
	r.mu.Unlock()
	firstTaoLine, _ := r.Talk(playerID, taoID)
	finalTaoLine, _ := r.Talk(playerID, taoID)
	finalChoices, _ := finalTaoLine["choices"].([]map[string]string)
	if firstTaoLine["choices"] != nil || len(finalChoices) != 3 {
		t.Fatalf("expected Tao Fang invitation after two dialogue lines, first=%+v final=%+v", firstTaoLine, finalTaoLine)
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

func testStringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
