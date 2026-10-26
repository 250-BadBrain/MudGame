package manager

import (
	"strings"
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

func TestDatangChangshengjueBlocksAndSalvage(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_changshengjue")

	// 渡口: 听老船夫交代情由（副本对话逐行推进）
	boatmanID := dungeonNPCIDByTemplateInRoom(t, inst, r.dungeonMgr.InstanceRoomIDNoLock(inst, "river_00"), "csj_boatman")
	if _, err := r.Talk(playerID, boatmanID); err != nil {
		t.Fatalf("first boatman line failed: %v", err)
	}
	if inst.DialogueProgress["listen_boatman"] != 1 {
		t.Fatalf("expected dialogue progress after first line, got %+v", inst.DialogueProgress)
	}
	if _, err := r.Talk(playerID, boatmanID); err != nil {
		t.Fatalf("final boatman line failed: %v", err)
	}
	if inst.PuzzleStates["briefing"] != "told" {
		t.Fatalf("expected briefing told, got %s", inst.PuzzleStates["briefing"])
	}

	// 芦苇荡: 武士拦路，杀不完过不去
	reedRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "river_01")
	moveDungeonTestPlayer(t, r, playerID, reedRoom)
	if _, err := r.Move(playerID, "north"); err == nil || !strings.Contains(err.Error(), "泥道") {
		t.Fatalf("expected blocked reed path, got %v", err)
	}
	for _, npc := range inst.NPCs {
		if npc.TemplateID == "csj_yuwen_soldier" {
			npc.Alive = false
		}
	}
	if _, err := r.Move(playerID, "north"); err != nil {
		t.Fatalf("north should clear after soldiers fall: %v", err)
	}

	// 沉船浅滩: 打捞铁匣（可选解谜收益）
	salvageRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "river_02")
	moveDungeonTestPlayer(t, r, playerID, salvageRoom)
	result, err := r.DoRoomAction(playerID, "salvage_iron_box")
	if err != nil || result["actionStatus"] != "success" {
		t.Fatalf("salvage failed: result=%+v err=%v", result, err)
	}
	r.mu.RLock()
	pageCount := r.playersCache[playerID].Backpack["herb"]
	r.mu.RUnlock()
	if pageCount != 4 {
		t.Fatalf("expected changshengjue page in backpack, got %d", pageCount)
	}
	result, err = r.DoRoomAction(playerID, "salvage_iron_box")
	if err != nil || result["actionStatus"] != "repeat" {
		t.Fatalf("repeat salvage should be a no-op repeat, got %+v err=%v", result, err)
	}
}

func TestPosuiStarMapGatesExitAndWeakensPhantom(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "posui_jingyan")
	hallRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "palace_02")
	moveDungeonTestPlayer(t, r, playerID, hallRoom)

	// 星图未解: 北门在房间数据里不可见，移动也被拦
	snapshot, _, err := r.GetRoomSnapshot(playerID)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	exits := snapshot["exits"].(map[string]map[string]interface{})
	if _, ok := exits["north"]; ok {
		t.Fatalf("star gate should be hidden before solving, exits=%+v", exits)
	}
	if _, err := r.Move(playerID, "north"); err == nil {
		t.Fatal("star gate should block north exit")
	}

	// 三步机关：观察 → 对照 → 转盘选择（选错可重试）
	if _, err := r.DoRoomAction(playerID, "observe_star_map"); err != nil {
		t.Fatalf("observe star map failed: %v", err)
	}
	if inst.PuzzleStates["star_map"] != "sketched" {
		t.Fatalf("expected star map sketched, got %s", inst.PuzzleStates["star_map"])
	}
	if _, err := r.DoRoomAction(playerID, "compare_murals"); err != nil {
		t.Fatalf("compare murals failed: %v", err)
	}
	if inst.PuzzleStates["star_map"] != "matched" {
		t.Fatalf("expected star map matched, got %s", inst.PuzzleStates["star_map"])
	}
	if _, err := r.DoRoomAction(playerID, "turn_star_dial"); err != nil {
		t.Fatalf("turn dial failed: %v", err)
	}
	// 先选错方向：回到 sketched，需重新对照后再转
	if _, err := r.DoRoomAction(playerID, "dial_right"); err != nil {
		t.Fatalf("dial right failed: %v", err)
	}
	if inst.PuzzleStates["star_map"] != "sketched" {
		t.Fatalf("wrong dial should reset to sketched, got %s", inst.PuzzleStates["star_map"])
	}
	if _, err := r.DoRoomAction(playerID, "compare_murals"); err != nil {
		t.Fatalf("re-compare murals failed: %v", err)
	}
	if _, err := r.DoRoomAction(playerID, "turn_star_dial"); err != nil {
		t.Fatalf("re-turn dial failed: %v", err)
	}
	if _, err := r.DoRoomAction(playerID, "dial_left"); err != nil {
		t.Fatalf("dial left failed: %v", err)
	}
	if inst.PuzzleStates["star_map"] != "read" {
		t.Fatalf("expected star map read, got %s", inst.PuzzleStates["star_map"])
	}

	// 图录大殿: 玩家看破星图 -> 对战神幻影出手时触发破招
	phantomRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "palace_04")
	moveDungeonTestPlayer(t, r, playerID, phantomRoom)
	phantomID := dungeonNPCIDByTemplateInRoom(t, inst, phantomRoom, "jyg_war_phantom")
	baseAttack := inst.NPCs[phantomID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, phantomID); err != nil {
		t.Fatalf("kill phantom failed: %v", err)
	}
	if !inst.NPCs[phantomID].WeaknessApplied {
		t.Fatal("expected star map flag to weaken the war phantom")
	}
	if got := inst.NPCs[phantomID].Attributes.Derived.Attack; got != baseAttack-8 {
		t.Fatalf("expected attack weakened by 8, got %d (base %d)", got, baseAttack)
	}
}

func TestYanggongPuzzlesGateVaultExits(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "datang_yanggong")

	mechRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "vault_01")
	moveDungeonTestPlayer(t, r, playerID, mechRoom)
	if _, err := r.Move(playerID, "north"); err == nil {
		t.Fatal("mechanism gate should block north before turning the dial")
	}
	if _, err := r.DoRoomAction(playerID, "turn_mechanism"); err != nil {
		t.Fatalf("turn mechanism failed: %v", err)
	}
	if inst.PuzzleStates["mechanism"] != "open" {
		t.Fatalf("expected mechanism open, got %s", inst.PuzzleStates["mechanism"])
	}
	if _, err := r.Move(playerID, "north"); err != nil {
		t.Fatalf("north should open after mechanism solved: %v", err)
	}

	chessRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "vault_03")
	moveDungeonTestPlayer(t, r, playerID, chessRoom)
	if _, err := r.Move(playerID, "north"); err == nil || !strings.Contains(err.Error(), "死局") {
		t.Fatalf("expected chess gate to block north, got %v", err)
	}
	if _, err := r.DoRoomAction(playerID, "solve_chess"); err != nil {
		t.Fatalf("solve chess failed: %v", err)
	}
	if inst.PuzzleStates["chess"] != "solved" {
		t.Fatalf("expected chess solved, got %s", inst.PuzzleStates["chess"])
	}
	if _, err := r.Move(playerID, "north"); err != nil {
		t.Fatalf("north should open after chess solved: %v", err)
	}
}

func TestFuyuNujiaoDefeatModeBoss(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "fuyu_nujiao")
	hallRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "dock_04")
	moveDungeonTestPlayer(t, r, playerID, hallRoom)

	fangID := dungeonNPCIDByTemplateInRoom(t, inst, hallRoom, "nj_fang_yeyu")
	maxHP := inst.NPCs[fangID].Attributes.Resources.MaxHP

	r.mu.Lock()
	before := inst.Progress
	_, defeated, _, ok := r.dungeonMgr.DefeatNPCNoLock(fangID)
	r.mu.Unlock()
	if !ok {
		t.Fatal("expected defeat to register")
	}
	if !defeated.Defeated || !defeated.Alive {
		t.Fatal("defeat mode boss should be Defeated but remain alive in the room")
	}
	if defeated.Attributes.Resources.HP != maxHP/4 {
		t.Fatalf("defeat mode boss should retain quarter health, got %d", defeated.Attributes.Resources.HP)
	}
	r.mu.RLock()
	progress := inst.Progress
	r.mu.RUnlock()
	if progress != before+30 {
		t.Fatalf("expected fang yeyu objective to contribute 30 progress, got %d (before %d)", progress, before)
	}
}

func TestLanjiangTrialRequiresCompletionFlags(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("finale_user", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("finale_user", "finale_hero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	if _, err := r.EnterDungeon(playerID, "fuyu_lanjiang"); err == nil {
		t.Fatal("finale should stay locked without completion flags")
	}

	r.mu.Lock()
	players, _ := r.loadPlayers()
	players[playerID].Flags["fuyu_nujiao_completed"] = true
	players[playerID].Flags["posui_jingyan_completed"] = true
	// 章节链规则：order 7 还需 order 6（杨公宝库）通关旗标
	players[playerID].Flags["dungeon_cleared_datang_yanggong"] = true
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save flags failed: %v", err)
	}
	r.mu.Unlock()

	if _, err := r.EnterDungeon(playerID, "fuyu_lanjiang"); err != nil {
		t.Fatalf("finale should open after both flags, got %v", err)
	}
}

func TestLanjiangTideObservationWeakensPangBan(t *testing.T) {
	r, playerID, inst := createDungeonTestPlayer(t, "fuyu_lanjiang")
	boatRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "lj_01")
	moveDungeonTestPlayer(t, r, playerID, boatRoom)

	if _, err := r.DoRoomAction(playerID, "observe_tide"); err != nil {
		t.Fatalf("observe tide failed: %v", err)
	}
	if inst.PuzzleStates["tide"] != "seen" {
		t.Fatalf("expected tide seen, got %s", inst.PuzzleStates["tide"])
	}

	bossRoom := r.dungeonMgr.InstanceRoomIDNoLock(inst, "lj_02")
	moveDungeonTestPlayer(t, r, playerID, bossRoom)
	pangID := dungeonNPCIDByTemplateInRoom(t, inst, bossRoom, "lj_pang_ban")
	baseAttack := inst.NPCs[pangID].Attributes.Derived.Attack
	if _, err := r.Kill(playerID, pangID); err != nil {
		t.Fatalf("engage pang ban failed: %v", err)
	}
	if !inst.NPCs[pangID].WeaknessApplied {
		t.Fatal("expected tide observation to weaken pang ban echo")
	}
	if got := inst.NPCs[pangID].Attributes.Derived.Attack; got != baseAttack-10 {
		t.Fatalf("expected attack weakened by 10, got %d (base %d)", got, baseAttack)
	}
}
