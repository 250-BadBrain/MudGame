package manager

import (
	"reflect"
	"testing"

	engruntime "mudgame/server/internal/engine/runtime"
)

func createDungeonTestPlayer(t *testing.T, dungeonID string) (*RuntimeManager, string, *engruntime.Instance) {
	t.Helper()
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	username := "user_" + dungeonID[len(dungeonID)-2:]
	if err := r.Register(username, "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter(username, "testhero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	r.mu.Lock()
	inst, err := r.dungeonMgr.CreateInstanceNoLock(playerID, dungeonID, "ly_00")
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
	return r, playerID, inst
}

func moveDungeonTestPlayer(t *testing.T, r *RuntimeManager, playerID, roomID string) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	players, _ := r.loadPlayers()
	players[playerID].RoomID = roomID
	if err := r.savePlayers(players); err != nil {
		t.Fatalf("save player failed: %v", err)
	}
}

func TestBronzeDiskWrongOrderSpawnsSnakesAndCorrectOrderSolves(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "xunqin_ch01_bronze_tomb")
	roomID := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj_01_04")
	moveDungeonTestPlayer(t, r, playerID, roomID)
	inst.PuzzleStates["disk"] = "keyed"

	for i := 0; i < 3; i++ {
		result, err := r.DoRoomAction(playerID, "turn_zhao_ring")
		if err != nil || result["actionStatus"] != "blocked" {
			t.Fatalf("wrong ring attempt %d should spawn a penalty, result=%+v err=%v", i+1, result, err)
		}
	}
	if inst.PuzzleStates["disk"] != "keyed" || inst.PuzzleAttempts["disk"] != 0 {
		t.Fatalf("third error should reset disk sequence, state=%s attempts=%d", inst.PuzzleStates["disk"], inst.PuzzleAttempts["disk"])
	}
	spawned := 0
	for _, npc := range inst.NPCs {
		if npc.RoomID == roomID && npc.TemplateID == "xqj_bronze_snake" {
			spawned++
		}
	}
	if spawned != 3 {
		t.Fatalf("expected one bronze snake per mistake, got %d", spawned)
	}
	for _, actionID := range []string{"turn_qin_ring", "turn_zhao_ring", "turn_chu_ring", "turn_qi_ring", "turn_hanwei_ring", "turn_yan_ring"} {
		if _, err := r.DoRoomAction(playerID, actionID); err != nil {
			t.Fatalf("correct disk action %s failed: %v", actionID, err)
		}
	}
	if inst.PuzzleStates["disk"] != "solved" {
		t.Fatalf("expected seven-state disk solved, got %s", inst.PuzzleStates["disk"])
	}
}

func TestWildConvoyArrowClueSelectsReducedAmbushRoute(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "xunqin_ch02_wild_convoy")
	trackRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj_02_03")
	arrowRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj_02_04")
	ambushRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj_02_05")
	moveDungeonTestPlayer(t, r, playerID, trackRoom)
	if _, err := r.DoRoomAction(playerID, "inspect_wagon_tracks"); err != nil {
		t.Fatalf("inspect tracks failed: %v", err)
	}
	moveDungeonTestPlayer(t, r, playerID, arrowRoom)
	if _, err := r.DoRoomAction(playerID, "search_broken_arrows"); err != nil {
		t.Fatalf("search arrows failed: %v", err)
	}
	inst.PuzzleStates["rescue"] = "complete"
	moveDungeonTestPlayer(t, r, playerID, ambushRoom)
	snapshot, _, err := r.GetRoomSnapshot(playerID)
	if err != nil {
		t.Fatalf("get ambush room failed: %v", err)
	}
	exits := snapshot["exits"].(map[string]map[string]interface{})
	if _, ok := exits["northwest"]; !ok {
		t.Fatalf("expected reduced-ambush route visible, exits=%+v", exits)
	}
	if _, ok := exits["southwest"]; ok {
		t.Fatalf("expected three-reinforcement route hidden after arrow clue, exits=%+v", exits)
	}
}

func TestStableInvestigationRequiresPoisonBeforeOldGroomClue(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "xunqin_ch03_wu_stables")
	roomID := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj_03_05")
	moveDungeonTestPlayer(t, r, playerID, roomID)
	groomID := dungeonNPCIDByTemplateInRoom(t, inst, roomID, "xqj_old_groom")
	blocked, err := r.Talk(playerID, groomID)
	if err != nil || inst.PuzzleStates["investigation"] != "unknown" {
		t.Fatalf("old groom should not reveal traitor before poison, result=%+v err=%v", blocked, err)
	}
	inst.PuzzleStates["investigation"] = "poison"
	if _, err := r.Talk(playerID, groomID); err != nil {
		t.Fatalf("first old groom line failed: %v", err)
	}
	if inst.PuzzleStates["investigation"] != "poison" {
		t.Fatal("first dialogue line must not advance investigation")
	}
	if _, err := r.Talk(playerID, groomID); err != nil {
		t.Fatalf("final old groom line failed: %v", err)
	}
	if inst.PuzzleStates["investigation"] != "known" {
		t.Fatalf("expected traitor clue after full dialogue, got %s", inst.PuzzleStates["investigation"])
	}
}

func TestHostagePasswordKeepsChoicesAndWrongAnswerSpawnsJailers(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "xunqin_ch05_hostage_camp")
	roomID := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj_05_06")
	moveDungeonTestPlayer(t, r, playerID, roomID)
	inst.PuzzleStates["password"] = "known"
	first, err := r.DoRoomAction(playerID, "speak_inner_password")
	if err != nil {
		t.Fatalf("start password prompt failed: %v", err)
	}
	before := first["choices"]
	wrong, err := r.DoRoomAction(playerID, "password_no_moon")
	if err != nil || wrong["actionStatus"] != "blocked" {
		t.Fatalf("wrong password should remain retryable, result=%+v err=%v", wrong, err)
	}
	if !reflect.DeepEqual(before, wrong["choices"]) {
		t.Fatalf("wrong password changed choices: before=%+v after=%+v", before, wrong["choices"])
	}
	spawned := 0
	for _, npc := range inst.NPCs {
		if npc.RoomID == roomID && npc.TemplateID == "xqj_jailer" {
			spawned++
		}
	}
	if spawned != 2 {
		t.Fatalf("expected two jailers after wrong password, got %d", spawned)
	}
	if _, err := r.DoRoomAction(playerID, "password_old_guest"); err != nil {
		t.Fatalf("correct retry failed: %v", err)
	}
	if inst.PuzzleStates["password"] != "passed" {
		t.Fatalf("expected inner prison opened, got %s", inst.PuzzleStates["password"])
	}
	if _, err := r.Move(playerID, "north"); err == nil {
		t.Fatal("spawned jailers should block the opened inner prison door")
	}
	for _, npc := range inst.NPCs {
		if npc.RoomID == roomID && npc.TemplateID == "xqj_jailer" {
			npc.Alive = false
			npc.Defeated = true
		}
	}
	if _, err := r.Move(playerID, "north"); err != nil {
		t.Fatalf("defeating spawned jailers should clear the inner prison door: %v", err)
	}
}

func TestHandanEscapeRouteRequiresReadingNotice(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "xunqin_ch06_handan_alley")
	roomID := r.dungeonMgr.InstanceRoomIDNoLock(inst, "xqj_06_02")
	moveDungeonTestPlayer(t, r, playerID, roomID)

	blocked, err := r.DoRoomAction(playerID, "choose_escape_route")
	if err != nil || blocked["actionStatus"] != "blocked" {
		t.Fatalf("route choice should be blocked before notice, result=%+v err=%v", blocked, err)
	}
	if len(inst.PendingChoices) != 0 {
		t.Fatal("blocked route choice must not create pending choices")
	}
	if _, err := r.DoRoomAction(playerID, "inspect_lockdown_notice"); err != nil {
		t.Fatalf("inspect notice failed: %v", err)
	}
	ready, err := r.DoRoomAction(playerID, "choose_escape_route")
	if err != nil {
		t.Fatalf("route choice after notice failed: %v", err)
	}
	if len(ready["choices"].([]map[string]string)) != 3 {
		t.Fatalf("expected three route choices, got %+v", ready["choices"])
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
