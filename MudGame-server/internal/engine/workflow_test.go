package engine

import (
	"strings"
	"testing"

	"mudgame/server/internal/manager"
	"mudgame/server/internal/model"
)

func extractRoomNPCs(room map[string]interface{}) []map[string]interface{} {
	raw, ok := room["npcs"]
	if !ok || raw == nil {
		return nil
	}
	if typed, ok := raw.([]map[string]interface{}); ok {
		return typed
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if ok {
			out = append(out, m)
		}
	}
	return out
}

func roomNPCIDByName(t *testing.T, repo RuntimeRepository, playerID, npcName string) string {
	t.Helper()
	snapshot, err := GetRoomSnapshotView(repo, playerID)
	if err != nil {
		t.Fatalf("get room snapshot failed: %v", err)
	}
	for _, npc := range extractRoomNPCs(snapshot.Room) {
		if npc["name"] == npcName && npc["state"] != "DEAD" {
			if id, _ := npc["id"].(string); id != "" {
				return id
			}
		}
	}
	t.Fatalf("expected living npc %s in room, got %+v", npcName, extractRoomNPCs(snapshot.Room))
	return ""
}

const (
	workflowShopNPCID    = "east_market_keeper"
	workflowTeacherNPCID = "weapon_school_master"
	workflowShopItemID   = "common_clothes"
)

type testHelper struct {
	repo     RuntimeRepository
	username string
	playerID string
}

func moveToOutskirtsFromSpawn(t *testing.T, repo RuntimeRepository, playerID string) {
	t.Helper()
	path := []string{"south", "south"}
	for _, dir := range path {
		if _, err := MoveWithTransition(repo, playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveBackFromOutskirtsToSpawn(t *testing.T, repo RuntimeRepository, playerID string) {
	t.Helper()
	path := []string{"north", "north"}
	for _, dir := range path {
		if _, err := MoveWithTransition(repo, playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveToShopFromSpawn(t *testing.T, repo RuntimeRepository, playerID string) {
	t.Helper()
	path := []string{"east"}
	for _, dir := range path {
		if _, err := MoveWithTransition(repo, playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveToTeacherFromSpawn(t *testing.T, repo RuntimeRepository, playerID string) {
	t.Helper()
	path := []string{"east", "south"}
	for _, dir := range path {
		if _, err := MoveWithTransition(repo, playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveToAcademyFromSpawn(t *testing.T, repo RuntimeRepository, playerID string) {
	t.Helper()
	path := []string{"north", "east"}
	for _, dir := range path {
		if _, err := MoveWithTransition(repo, playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func runStatusTicks(t *testing.T, repo RuntimeRepository, times int) {
	t.Helper()
	for i := 0; i < times; i++ {
		if _, err := ProcessStatusTickWithEffects(repo); err != nil {
			t.Fatalf("process status tick failed at %d: %v", i, err)
		}
	}
}

func killRoomNPCByName(t *testing.T, repo RuntimeRepository, playerID, npcName string) {
	t.Helper()
	snapshot, err := GetRoomSnapshotView(repo, playerID)
	if err != nil {
		t.Fatalf("get room snapshot failed: %v", err)
	}
	targetID := roomNPCIDByName(t, repo, playerID, npcName)
	if _, err := KillWithEffects(repo, playerID, targetID); err != nil {
		t.Fatalf("kill %s failed: %v", npcName, err)
	}
	for i := 0; i < 80; i++ {
		if _, err := ProcessCombatTickWithEffects(repo); err != nil {
			t.Fatalf("process combat tick for %s failed at %d: %v", npcName, i, err)
		}
		status, err := GetPlayerStatusView(repo, playerID)
		if err != nil {
			t.Fatalf("get status after combat failed: %v", err)
		}
		if status["state"] == "IDLE" {
			recoverPlayerBetweenFights(t, repo, playerID)
			return
		}
	}
	status, _ := GetPlayerStatusView(repo, playerID)
	snapshot, _ = GetRoomSnapshotView(repo, playerID)
	t.Fatalf("combat with %s did not finish, status=%+v roomNPCs=%+v", npcName, status, extractRoomNPCs(snapshot.Room))
}

func recoverPlayerBetweenFights(t *testing.T, repo RuntimeRepository, playerID string) {
	t.Helper()
	for round := 0; round < 4; round++ {
		attrs, err := GetAttributesView(repo, playerID)
		if err != nil {
			t.Fatalf("get recovery attributes failed: %v", err)
		}
		if attrs["currentHp"].(int) >= attrs["maxHp"].(int) && attrs["currentMp"].(int) >= attrs["trainedMaxMp"].(int) {
			return
		}
		if attrs["currentHp"].(int) < attrs["maxHp"].(int) && attrs["currentMp"].(int) >= 10 {
			if _, err := HealWithEffects(repo, playerID); err != nil {
				t.Fatalf("start recovery healing failed: %v", err)
			}
			for i := 0; i < 100; i++ {
				if _, err := ProcessStatusTickWithEffects(repo); err != nil {
					t.Fatalf("process healing recovery tick failed: %v", err)
				}
				current, _ := GetAttributesView(repo, playerID)
				if current["currentHp"].(int) >= current["maxHp"].(int) {
					_ = StopWithEffects(repo, playerID)
					break
				}
			}
		}
		attrs, err = GetAttributesView(repo, playerID)
		if err != nil {
			t.Fatalf("get post-healing attributes failed: %v", err)
		}
		if attrs["currentMp"].(int) < attrs["trainedMaxMp"].(int) {
			if _, err := MeditateWithEffects(repo, playerID); err != nil {
				t.Fatalf("start recovery meditation failed: %v", err)
			}
			for i := 0; i < 100; i++ {
				if _, err := ProcessStatusTickWithEffects(repo); err != nil {
					t.Fatalf("process meditation recovery tick failed: %v", err)
				}
				current, _ := GetAttributesView(repo, playerID)
				if current["currentMp"].(int) >= current["trainedMaxMp"].(int) {
					_ = StopWithEffects(repo, playerID)
					break
				}
			}
		}
	}
}

func setupRepo(t *testing.T) *testHelper {
	t.Helper()
	repo := manager.NewRuntimeManager(t.TempDir())

	username := "tester"
	password := "pass123"
	if err := repo.Register(username, password); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := repo.Login(username, password); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	playerID, err := repo.CreateCharacter(username, "hero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	return &testHelper{repo: repo, username: username, playerID: playerID}
}

func TestReconnectSuccess(t *testing.T) {
	h := setupRepo(t)
	token, err := h.repo.Login(h.username, "pass123")
	if err != nil {
		t.Fatalf("login for token failed: %v", err)
	}

	res, err := Reconnect(h.repo, h.username, "", token, h.playerID)
	if err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}
	if res.UserID != h.username {
		t.Fatalf("unexpected user id: %s", res.UserID)
	}
	if res.PlayerID != h.playerID {
		t.Fatalf("unexpected player id: %s", res.PlayerID)
	}
}

func TestReconnectInvalidToken(t *testing.T) {
	h := setupRepo(t)
	_, err := Reconnect(h.repo, h.username, "", "bad-token", h.playerID)
	if err == nil {
		t.Fatalf("expected reconnect error")
	}
}

func TestMoveAndEnterWorldTransitions(t *testing.T) {
	h := setupRepo(t)

	moveRes, err := MoveWithTransition(h.repo, h.playerID, "north")
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}
	if moveRes.FromRoomID == moveRes.ToRoomID {
		t.Fatalf("expected different room after move")
	}

	enterRes, err := EnterWorldWithTransition(h.repo, h.playerID, "luoyang")
	if err != nil {
		t.Fatalf("enter world failed: %v", err)
	}
	if enterRes.ToRoomID != "ly_00" {
		t.Fatalf("unexpected target room: %s", enterRes.ToRoomID)
	}
}

func setupSecondPlayer(t *testing.T, username, charName string, repo RuntimeRepository) string {
	t.Helper()
	if err := repo.Register(username, "pass123"); err != nil {
		t.Fatalf("register %s failed: %v", username, err)
	}
	if _, err := repo.Login(username, "pass123"); err != nil {
		t.Fatalf("login %s failed: %v", username, err)
	}
	playerID, err := repo.CreateCharacter(username, charName, 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create %s character failed: %v", username, err)
	}
	return playerID
}

func movePlayersToDuelArea(t *testing.T, repo RuntimeRepository, ids ...string) {
	t.Helper()
	for _, id := range ids {
		if _, err := EnterWorldWithTransition(repo, id, "luoyang"); err != nil {
			t.Fatal(err)
		}
		moveToOutskirtsFromSpawn(t, repo, id)
	}
}

func TestStateAndCombatWorkflows(t *testing.T) {
	h := setupRepo(t)
	p2 := setupSecondPlayer(t, "tester2", "hero2", h.repo)
	p3 := setupSecondPlayer(t, "tester3", "hero3", h.repo)
	p4 := setupSecondPlayer(t, "tester4", "hero4", h.repo)

	if _, err := h.repo.Login(h.username, "pass123"); err != nil {
		t.Fatalf("refresh login failed: %v", err)
	}
	moveToOutskirtsFromSpawn(t, h.repo, h.playerID)
	actionRes, err := DoRoomActionWithEffects(h.repo, h.playerID, "gather")
	if err != nil {
		t.Fatalf("room action workflow failed: %v", err)
	}
	if actionRes.Message == "" {
		t.Fatalf("unexpected room action result: %+v", actionRes)
	}
	if _, err := HealWithEffects(h.repo, h.playerID); err == nil {
		t.Fatalf("heal should still reject when hp is full")
	}
	_ = StopWithEffects(h.repo, h.playerID)
	moveBackFromOutskirtsToSpawn(t, h.repo, h.playerID)
	movePlayersToDuelArea(t, h.repo, h.playerID, p2, p3, p4)
	combatRes, err := SparWithEffects(h.repo, h.playerID, p2)
	if err != nil {
		t.Fatalf("spar workflow failed: %v", err)
	}
	if !combatRes.NeedRoomMessage || !combatRes.NeedRoomUpdate {
		t.Fatalf("unexpected combat result: %+v", combatRes)
	}
	stopRes := StopWithEffects(h.repo, h.playerID)
	if stopRes.Message != "你正在战斗，待会再说" {
		t.Fatalf("unexpected stop message in combat: %+v", stopRes)
	}
	if stopRes.NeedRoomUpdate {
		t.Fatalf("stop in combat should not trigger room update")
	}

	killRes, err := KillWithEffects(h.repo, p3, p4)
	if err != nil {
		t.Fatalf("kill workflow failed: %v", err)
	}
	if killRes.Message == "" {
		t.Fatalf("kill message should not be empty")
	}
	deadID := ""
	for i := 0; i < 800; i++ {
		if _, err := ProcessCombatTickWithEffects(h.repo); err != nil {
			t.Fatalf("process combat tick failed: %v", err)
		}
		status4, err := GetPlayerStatusView(h.repo, p4)
		if err != nil {
			t.Fatalf("get player4 status failed: %v", err)
		}
		if state, _ := status4["state"].(string); state == "DEAD" {
			deadID = p4
			break
		}
		status3, err := GetPlayerStatusView(h.repo, p3)
		if err != nil {
			t.Fatalf("get player3 status failed: %v", err)
		}
		if state, _ := status3["state"].(string); state == "DEAD" {
			deadID = p3
			break
		}
	}
	if deadID == "" {
		t.Fatalf("expected one combatant dead after kill combat ticks")
	}

	reliveRes, err := ReliveWithEffects(h.repo, deadID)
	if err != nil {
		t.Fatalf("relive workflow failed: %v", err)
	}
	if reliveRes.Action != "refresh_room" {
		t.Fatalf("unexpected relive action: %s", reliveRes.Action)
	}
}

func TestShopAndInventoryWorkflows(t *testing.T) {
	h := setupRepo(t)

	moveToShopFromSpawn(t, h.repo, h.playerID)
	shop, err := GetShopView(h.repo, h.playerID, workflowShopNPCID)
	if err != nil {
		t.Fatalf("get shop view failed: %v", err)
	}
	if shop.NPCID != workflowShopNPCID || len(shop.Items) == 0 {
		t.Fatalf("unexpected shop payload: %+v", shop)
	}

	buyRes, err := BuyWithReceipt(h.repo, h.playerID, workflowShopNPCID, workflowShopItemID, 1)
	if err != nil {
		t.Fatalf("buy workflow failed: %v", err)
	}
	if buyRes.BoughtAmount != 1 || len(buyRes.Logs) == 0 {
		t.Fatalf("unexpected buy result: %+v", buyRes)
	}

	equipRes, err := EquipWithEffects(h.repo, h.playerID, workflowShopItemID)
	if err != nil {
		t.Fatalf("equip workflow failed: %v", err)
	}
	if equipRes.Message == "" || !equipRes.NeedRoomUpdate {
		t.Fatalf("unexpected equip result: %+v", equipRes)
	}

	unequipRes, err := UnequipWithEffects(h.repo, h.playerID, "BODY")
	if err != nil {
		t.Fatalf("unequip workflow failed: %v", err)
	}
	if unequipRes.Message == "" || !unequipRes.NeedRoomUpdate {
		t.Fatalf("unexpected unequip result: %+v", unequipRes)
	}

	if _, err := EnterWorldWithTransition(h.repo, h.playerID, "luoyang"); err != nil {
		t.Fatalf("reset to luoyang spawn failed: %v", err)
	}
	moveToTeacherFromSpawn(t, h.repo, h.playerID)
	learnRes, err := LearnWithEffects(h.repo, h.playerID, workflowTeacherNPCID, "basic_unarmed")
	if err != nil {
		t.Fatalf("learn workflow failed: %v", err)
	}
	if learnRes.Message == "" {
		t.Fatalf("unexpected learn result: %+v", learnRes)
	}
	_ = StopWithEffects(h.repo, h.playerID)

	discardMsg, err := DiscardWithMessage(h.repo, h.playerID, "iron_pickaxe", 1)
	if err != nil {
		t.Fatalf("discard workflow failed: %v", err)
	}
	if discardMsg == "" {
		t.Fatalf("discard message should not be empty")
	}

	if _, err := EnterWorldWithTransition(h.repo, h.playerID, "luoyang"); err != nil {
		t.Fatal(err)
	}
	moveToShopFromSpawn(t, h.repo, h.playerID)
	sellRes, err := SellWithReceipt(h.repo, h.playerID, workflowShopItemID, 1)
	if err != nil {
		t.Fatalf("sell workflow failed: %v", err)
	}
	if sellRes.Gain <= 0 || sellRes.Money <= 0 {
		t.Fatalf("unexpected sell result: %+v", sellRes)
	}
}

func TestDialogueAndQuestWorkflow(t *testing.T) {
	h := setupRepo(t)

	if _, err := MoveWithTransition(h.repo, h.playerID, "south"); err != nil {
		t.Fatalf("move to south gate failed: %v", err)
	}
	dialogue, err := TalkWithView(h.repo, h.playerID, "escort_bureau_chief")
	if err != nil {
		t.Fatalf("talk escort failed: %v", err)
	}
	if dialogue.Dialogue["id"] != "escort_intro" {
		t.Fatalf("unexpected dialogue: %+v", dialogue.Dialogue)
	}
	if _, err := TalkWithView(h.repo, h.playerID, "escort_bureau_chief"); err != nil {
		t.Fatalf("advance quest dialogue failed: %v", err)
	}
	quests, err := GetQuestsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get quests failed: %v", err)
	}
	questStatus := func(rows []map[string]interface{}) string {
		for _, row := range rows {
			if row["id"] == "escort_rumor_run" {
				return row["status"].(string)
			}
		}
		return ""
	}
	if len(quests.Quests) != 38 || questStatus(quests.Quests) != "active" {
		t.Fatal("expected active rumor quest within the directory")
	}

	if _, err := MoveWithTransition(h.repo, h.playerID, "west"); err != nil {
		t.Fatalf("move to inn failed: %v", err)
	}
	if _, err := TalkWithView(h.repo, h.playerID, "storyteller"); err != nil {
		t.Fatalf("open storyteller dialogue failed: %v", err)
	}
	if _, err := TalkWithView(h.repo, h.playerID, "storyteller"); err != nil {
		t.Fatalf("hear rumor dialogue failed: %v", err)
	}
	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move back to south gate failed: %v", err)
	}
	if _, err := TalkWithView(h.repo, h.playerID, "escort_bureau_chief"); err != nil {
		t.Fatalf("hear quest reminder dialogue failed: %v", err)
	}
	if _, err := TalkWithView(h.repo, h.playerID, "escort_bureau_chief"); err != nil {
		t.Fatalf("complete quest dialogue failed: %v", err)
	}
	quests, err = GetQuestsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get quests after completion failed: %v", err)
	}
	if len(quests.Quests) != 38 || questStatus(quests.Quests) != "completed" {
		t.Fatal("expected completed rumor quest within the directory")
	}
	attrs, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attrs after quest failed: %v", err)
	}
	if attrs["experience"].(int) < 30 || attrs["potential"].(int) < 525 {
		t.Fatalf("expected quest rewards in attrs, got %+v", attrs)
	}
	factions, err := GetFactionsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get factions after quest failed: %v", err)
	}
	foundEscortRep := false
	for _, faction := range factions.Factions {
		if faction["id"] == "luoyang_escort" && faction["reputation"] == 10 {
			foundEscortRep = true
		}
	}
	if !foundEscortRep {
		t.Fatalf("expected luoyang_escort reputation 10, got %+v", factions.Factions)
	}
	dialogue, err = TalkWithView(h.repo, h.playerID, "escort_bureau_chief")
	if err != nil {
		t.Fatalf("talk escort after reputation failed: %v", err)
	}
	hasTrustedOption := false
	switch options := dialogue.Dialogue["options"].(type) {
	case []map[string]interface{}:
		for _, option := range options {
			if option["id"] == "trusted_runner" {
				hasTrustedOption = true
				break
			}
		}
	case []interface{}:
		for _, raw := range options {
			option, _ := raw.(map[string]interface{})
			if option["id"] == "trusted_runner" {
				hasTrustedOption = true
				break
			}
		}
	}
	if !hasTrustedOption {
		t.Fatalf("expected reputation-gated dialogue option, got %+v", dialogue.Dialogue["options"])
	}
}

func TestQueryViewWorkflows(t *testing.T) {
	h := setupRepo(t)

	roomRes, err := GetRoomSnapshotView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get room snapshot view failed: %v", err)
	}
	if roomRes.Room["id"] == "" || roomRes.Player["id"] == "" {
		t.Fatalf("unexpected room snapshot payload: %+v", roomRes)
	}

	attrs, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attributes view failed: %v", err)
	}
	if attrs["currentHp"] == nil {
		t.Fatalf("attributes missing currentHp")
	}
	resources := attrs["resources"].(model.ResourceAttributes)
	if resources.MaxHP != 80 || attrs["currentHp"] != 148 || attrs["maxHp"] != 148 {
		t.Fatalf("expected base hp 80 and full effective hp 148, resources=%+v current=%v max=%v", resources, attrs["currentHp"], attrs["maxHp"])
	}

	bag, err := GetBackpackView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get backpack view failed: %v", err)
	}
	if len(bag.Items) == 0 || bag.Money <= 0 {
		t.Fatalf("unexpected backpack payload: %+v", bag)
	}

	skills, err := GetSkillsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get skills view failed: %v", err)
	}
	if len(skills) == 0 {
		t.Fatalf("skills should not be empty")
	}

	eq, err := GetEquipmentView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get equipment view failed: %v", err)
	}
	if len(eq) != 0 {
		t.Fatalf("new character should have empty equipment")
	}

	worlds := GetWorldsView(h.repo)
	if len(worlds) == 0 {
		t.Fatalf("world list should not be empty")
	}
}

func TestEffectiveAttributesUseSkillAndEquipmentBonuses(t *testing.T) {
	h := setupRepo(t)

	attrsBefore, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get initial attributes failed: %v", err)
	}
	initialMaxMP := attrsBefore["maxMp"].(int)

	moveToAcademyFromSpawn(t, h.repo, h.playerID)
	if _, err := LearnWithEffects(h.repo, h.playerID, "academy_teacher", "basic_neigong"); err != nil {
		t.Fatalf("learn basic neigong failed: %v", err)
	}
	runStatusTicks(t, h.repo, 30)
	_ = StopWithEffects(h.repo, h.playerID)

	attrsAfterSkill, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attributes after skill failed: %v", err)
	}
	if attrsAfterSkill["maxMp"].(int) <= initialMaxMP {
		t.Fatalf("expected basic_neigong to increase maxMp, before=%d after=%d", initialMaxMP, attrsAfterSkill["maxMp"].(int))
	}
	if attrsAfterSkill["trainedMaxMp"] != attrsBefore["trainedMaxMp"] {
		t.Fatalf("skill maxMp bonus must not raise trained maxMp, before=%v after=%v", attrsBefore["trainedMaxMp"], attrsAfterSkill["trainedMaxMp"])
	}
	if attrsAfterSkill["currentMp"] != attrsBefore["currentMp"] {
		t.Fatalf("skill maxMp bonus must not raise current mp, before=%v after=%v", attrsBefore["currentMp"], attrsAfterSkill["currentMp"])
	}
	if attrsAfterSkill["maxHp"] != attrsBefore["maxHp"] {
		t.Fatalf("hp conversion must use trained maxMp, not maxMp cap, before=%v after=%v", attrsBefore["maxHp"], attrsAfterSkill["maxHp"])
	}

	skills, err := GetSkillsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get skills failed: %v", err)
	}
	hasMaxMPBonus := false
	hasHPConversion := false
	for _, skill := range skills {
		if skill["id"] == "basic_neigong" {
			if bonusText, ok := skill["bonusText"].([]string); ok {
				for _, text := range bonusText {
					hasMaxMPBonus = hasMaxMPBonus || strings.Contains(text, "内力上限 +200")
					hasHPConversion = hasHPConversion || text == "已修内力转气血 +30个百分点"
				}
			}
		}
	}
	if !hasMaxMPBonus || !hasHPConversion {
		t.Fatalf("expected basic_neigong bonusText, got %+v", skills)
	}

	if _, err := EnterWorldWithTransition(h.repo, h.playerID, "luoyang"); err != nil {
		t.Fatalf("reset to luoyang failed: %v", err)
	}
	moveToShopFromSpawn(t, h.repo, h.playerID)
	attrsBeforeEquip, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attributes before equip failed: %v", err)
	}
	baseAttack := attrsBeforeEquip["attack"].(int)
	if _, err := BuyWithReceipt(h.repo, h.playerID, "heluo_blacksmith", "iron_sword", 1); err != nil {
		t.Fatalf("buy iron sword failed: %v", err)
	}
	if _, err := EquipWithEffects(h.repo, h.playerID, "iron_sword"); err != nil {
		t.Fatalf("equip iron sword failed: %v", err)
	}
	attrsAfterEquip, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attributes after equip failed: %v", err)
	}
	if attrsAfterEquip["attack"].(int) != baseAttack+6 {
		t.Fatalf("expected iron sword attack +6, before=%d after=%d", baseAttack, attrsAfterEquip["attack"].(int))
	}
}

func TestLearningContinuouslyConsumesPotentialAndRaisesSkill(t *testing.T) {
	h := setupRepo(t)
	moveToTeacherFromSpawn(t, h.repo, h.playerID)

	attrsBefore, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attributes before learning failed: %v", err)
	}
	skillsBefore, err := GetSkillsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get skills before learning failed: %v", err)
	}
	levelBefore := 0
	for _, skill := range skillsBefore {
		if skill["id"] == "basic_unarmed" {
			levelBefore = skill["level"].(int)
		}
	}

	if _, err := LearnWithEffects(h.repo, h.playerID, workflowTeacherNPCID, "basic_unarmed"); err != nil {
		t.Fatalf("start continuous learning failed: %v", err)
	}
	runStatusTicks(t, h.repo, 9)
	attrsBeforeSettlement, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attributes before first settlement failed: %v", err)
	}
	if attrsBeforeSettlement["potential"] != attrsBefore["potential"] {
		t.Fatalf("learning should not consume potential before 10 seconds")
	}

	firstTick, err := ProcessStatusTickWithEffects(h.repo)
	if err != nil {
		t.Fatalf("process first learning settlement failed: %v", err)
	}
	if len(firstTick.PlayerMessages) != 1 || !strings.Contains(firstTick.PlayerMessages[0].Message, "修炼至") {
		t.Fatalf("expected deterministic level-up feedback, got %+v", firstTick.PlayerMessages)
	}
	attrsAfterFirst, _ := GetAttributesView(h.repo, h.playerID)
	if attrsAfterFirst["potential"].(int) != attrsBefore["potential"].(int)-10 {
		t.Fatalf("expected one potential consumed, before=%v after=%v", attrsBefore["potential"], attrsAfterFirst["potential"])
	}

	runStatusTicks(t, h.repo, 20)
	skillsAfter, err := GetSkillsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get skills after learning failed: %v", err)
	}
	levelAfter := 0
	for _, skill := range skillsAfter {
		if skill["id"] == "basic_unarmed" {
			levelAfter = skill["level"].(int)
		}
	}
	if levelAfter != 20 || levelAfter <= levelBefore {
		t.Fatalf("expected low-level skill to rise at every settlement, before=%d after=%d", levelBefore, levelAfter)
	}
	status, err := GetPlayerStatusView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get learning status failed: %v", err)
	}
	if status["state"] != "IDLE" {
		t.Fatalf("expected learning to continue after level up, got %+v", status)
	}
}

func TestEventViewWorkflows(t *testing.T) {
	h := setupRepo(t)

	status, err := GetPlayerStatusView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get player status view failed: %v", err)
	}
	roomID, _ := status["roomId"].(string)
	if roomID == "" {
		t.Fatalf("status roomId should not be empty")
	}

	room, err := GetRoomPayloadView(h.repo, roomID)
	if err != nil {
		t.Fatalf("get room payload view failed: %v", err)
	}
	if room["id"] == "" {
		t.Fatalf("room payload id should not be empty")
	}

	update, err := GetRoomUpdateView(h.repo, roomID)
	if err != nil {
		t.Fatalf("get room update view failed: %v", err)
	}
	if len(update.Players) == 0 {
		t.Fatalf("room update players should not be empty")
	}

	rid2, rname2, err := GetPlayerRoomView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get player room view failed: %v", err)
	}
	if rid2 == "" || rname2 == "" {
		t.Fatalf("unexpected player room view: %s %s", rid2, rname2)
	}

	if !IsPlayerInRoomView(h.repo, h.playerID, rid2) {
		t.Fatalf("player should be in returned room")
	}
}

func TestStatusTickWorkflow(t *testing.T) {
	h := setupRepo(t)
	p2 := setupSecondPlayer(t, "ticker2", "ticker2hero", h.repo)

	moveToOutskirtsFromSpawn(t, h.repo, h.playerID)
	attrsBeforeGather, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attrs before gather failed: %v", err)
	}
	if _, err := DoRoomActionWithEffects(h.repo, h.playerID, "gather"); err != nil {
		t.Fatalf("gather failed: %v", err)
	}
	// 采集为忙碌状态制：30 秒倒数结束后由状态 tick 一次性结算收益
	runStatusTicks(t, h.repo, 31)
	attrsAfterGather, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attrs after gather failed: %v", err)
	}
	if attrsAfterGather["experience"].(int) <= attrsBeforeGather["experience"].(int) {
		t.Fatalf("expected gather to increase experience")
	}
	if attrsAfterGather["potential"].(int) <= attrsBeforeGather["potential"].(int) {
		t.Fatalf("expected gather to increase potential")
	}
	bagAfterGather, err := GetBackpackView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get backpack after gather failed: %v", err)
	}
	if len(bagAfterGather.Items) == 0 {
		t.Fatalf("expected gather to yield an item")
	}
	moveBackFromOutskirtsToSpawn(t, h.repo, h.playerID)
	attrsBeforeMeditate, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attrs before meditation failed: %v", err)
	}
	if _, err := MeditateWithEffects(h.repo, h.playerID); err != nil {
		t.Fatalf("meditate failed: %v", err)
	}
	runStatusTicks(t, h.repo, 10)
	attrsAfterMeditate, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attrs after meditation failed: %v", err)
	}
	if attrsAfterMeditate["trainedMaxMp"].(int) <= attrsBeforeMeditate["trainedMaxMp"].(int) {
		t.Fatalf("meditation should raise trained max mp, before=%+v after=%+v", attrsBeforeMeditate, attrsAfterMeditate)
	}
	if attrsAfterMeditate["maxHp"].(int) <= attrsBeforeMeditate["maxHp"].(int) {
		t.Fatalf("internal skill conversion should immediately raise max hp after meditation, before=%+v after=%+v", attrsBeforeMeditate, attrsAfterMeditate)
	}
	roomAfterMeditate, err := GetRoomSnapshotView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get room after meditation failed: %v", err)
	}
	roomPlayers, _ := roomAfterMeditate.Room["playersInRoom"].([]map[string]interface{})
	foundConvertedRoomHP := false
	for _, player := range roomPlayers {
		if player["id"] != h.playerID {
			continue
		}
		roomAttrs, _ := player["attributes"].(map[string]interface{})
		foundConvertedRoomHP = roomAttrs["maxHp"] == attrsAfterMeditate["maxHp"]
	}
	if !foundConvertedRoomHP {
		t.Fatalf("room player hp should use converted max hp, room=%+v attrs=%+v", roomAfterMeditate.Room, attrsAfterMeditate)
	}
	if stop := StopWithEffects(h.repo, h.playerID); stop.Message != "你停止了打坐。" {
		t.Fatalf("unexpected stop meditate message: %+v", stop)
	}
	// Meditation raised the effective HP cap without healing; spar requires actual full health.
	if _, err := HealWithEffects(h.repo, h.playerID); err != nil {
		t.Fatal(err)
	}
	runStatusTicks(t, h.repo, 10)
	StopWithEffects(h.repo, h.playerID)

	hpDropped := false
	for attempt := 0; attempt < 3 && !hpDropped; attempt++ {
		movePlayersToDuelArea(t, h.repo, h.playerID, p2)
		if _, err := SparWithEffects(h.repo, h.playerID, p2); err != nil {
			t.Fatalf("spar failed: %v", err)
		}
		for i := 0; i < 80; i++ {
			if _, err := ProcessCombatTickWithEffects(h.repo); err != nil {
				t.Fatalf("process combat tick failed: %v", err)
			}
			st1, err := GetPlayerStatusView(h.repo, h.playerID)
			if err != nil {
				t.Fatalf("get player1 status failed: %v", err)
			}
			attrsNow, err := GetAttributesView(h.repo, h.playerID)
			if err != nil {
				t.Fatalf("get attrs during combat tick failed: %v", err)
			}
			if attrsNow["currentHp"].(int) < attrsNow["maxHp"].(int) {
				hpDropped = true
			}
			if st1["state"] == "IDLE" {
				break
			}
		}
	}
	if !hpDropped {
		t.Fatalf("expected hp drop before heal test")
	}
	if _, err := HealWithEffects(h.repo, h.playerID); err != nil {
		t.Fatalf("heal failed: %v", err)
	}
	attrsBeforeHealTick, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attrs before heal tick failed: %v", err)
	}
	runStatusTicks(t, h.repo, 10)
	attrsAfterHealTick, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attrs after heal tick failed: %v", err)
	}
	if attrsAfterHealTick["currentHp"].(int) < attrsBeforeHealTick["currentHp"].(int) {
		t.Fatalf("heal tick should not decrease currentHp")
	}
}

func TestSparAutoStopWorkflow(t *testing.T) {
	h := setupRepo(t)
	p2 := setupSecondPlayer(t, "spar2", "spar2hero", h.repo)

	movePlayersToDuelArea(t, h.repo, h.playerID, p2)
	if _, err := SparWithEffects(h.repo, h.playerID, p2); err != nil {
		t.Fatalf("spar failed: %v", err)
	}

	idle1, idle2 := false, false
	for i := 0; i < 40; i++ {
		if _, err := ProcessCombatTickWithEffects(h.repo); err != nil {
			t.Fatalf("process combat tick failed: %v", err)
		}
		st1, err := GetPlayerStatusView(h.repo, h.playerID)
		if err != nil {
			t.Fatalf("get player1 status failed: %v", err)
		}
		st2, err := GetPlayerStatusView(h.repo, p2)
		if err != nil {
			t.Fatalf("get player2 status failed: %v", err)
		}
		idle1 = st1["state"] == "IDLE"
		idle2 = st2["state"] == "IDLE"
		if idle1 && idle2 {
			break
		}
	}
	if !idle1 || !idle2 {
		t.Fatalf("expected both players IDLE after spar ticks, got idle1=%v idle2=%v", idle1, idle2)
	}
}

func TestCombatTargetLossWorkflow(t *testing.T) {
	h := setupRepo(t)
	p2 := setupSecondPlayer(t, "loss2", "loss2hero", h.repo)

	movePlayersToDuelArea(t, h.repo, h.playerID, p2)
	if _, err := SparWithEffects(h.repo, h.playerID, p2); err != nil {
		t.Fatalf("spar failed: %v", err)
	}
	if err := h.repo.DeleteCharacter("loss2", p2); err != nil {
		t.Fatalf("delete opponent failed: %v", err)
	}

	tickRes, err := ProcessCombatTickWithEffects(h.repo)
	if err != nil {
		t.Fatalf("process combat tick failed: %v", err)
	}
	if len(tickRes.Messages) == 0 {
		t.Fatalf("expected target-loss combat message")
	}

	status, err := GetPlayerStatusView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get player status failed: %v", err)
	}
	if status["state"] != "IDLE" {
		t.Fatalf("expected attacker leave combat after target loss, got %v", status["state"])
	}
}

func TestDungeonInstanceLoadsNPCsAndSupportsKill(t *testing.T) {
	h := setupRepo(t)

	enterRes, err := EnterDungeonWithTransition(h.repo, h.playerID, "abandoned_temple_rain")
	if err != nil {
		t.Fatalf("enter dungeon failed: %v", err)
	}
	roomID, _ := enterRes.Room["id"].(string)
	if !strings.HasPrefix(roomID, "dg_abandoned_temple_rain_") {
		t.Fatalf("unexpected dungeon room id: %s", roomID)
	}
	if worldName, _ := enterRes.Room["worldName"].(string); worldName == "" {
		t.Fatalf("dungeon worldName should not be empty")
	}

	killRoomNPCByName(t, h.repo, h.playerID, "荒庙流寇")
	if _, err := MoveWithTransition(h.repo, h.playerID, "north"); err != nil {
		t.Fatalf("move to wolf room failed: %v", err)
	}
	snapshot, err := GetRoomSnapshotView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get room snapshot failed: %v", err)
	}
	npcs := extractRoomNPCs(snapshot.Room)
	if len(npcs) == 0 {
		t.Fatalf("expected dungeon NPCs in room payload, got %+v", snapshot.Room)
	}
	npcID, _ := npcs[0]["id"].(string)
	if npcID == "" {
		t.Fatalf("expected dungeon npc id in payload: %+v", npcs[0])
	}

	if _, err := KillWithEffects(h.repo, h.playerID, npcID); err != nil {
		t.Fatalf("kill dungeon npc should be supported, got: %v", err)
	}
}

func TestStaticNPCSparAndKillWorkflows(t *testing.T) {
	h := setupRepo(t)

	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move to east market failed: %v", err)
	}
	if _, err := MoveWithTransition(h.repo, h.playerID, "south"); err != nil {
		t.Fatalf("move to weapon school failed: %v", err)
	}
	if _, err := SparWithEffects(h.repo, h.playerID, "weapon_school_master"); err != nil {
		t.Fatalf("spar static npc failed: %v", err)
	}
	stopped := false
	for i := 0; i < 80; i++ {
		if _, err := ProcessCombatTickWithEffects(h.repo); err != nil {
			t.Fatalf("combat tick failed: %v", err)
		}
		status, err := GetPlayerStatusView(h.repo, h.playerID)
		if err != nil {
			t.Fatalf("get status failed: %v", err)
		}
		if status["state"] == "IDLE" {
			stopped = true
			break
		}
	}
	if !stopped {
		t.Fatalf("expected static npc spar to stop")
	}

	h = setupRepo(t)
	if _, err := MoveWithTransition(h.repo, h.playerID, "south"); err != nil {
		t.Fatalf("move to south gate failed: %v", err)
	}
	if _, err := MoveWithTransition(h.repo, h.playerID, "south"); err != nil {
		t.Fatalf("move to outskirts failed: %v", err)
	}
	if _, err := KillWithEffects(h.repo, h.playerID, "road_bandit"); err != nil {
		t.Fatalf("kill static npc failed: %v", err)
	}
	defeated := false
	for i := 0; i < 80; i++ {
		if _, err := ProcessCombatTickWithEffects(h.repo); err != nil {
			t.Fatalf("combat tick failed: %v", err)
		}
		snapshot, err := GetRoomSnapshotView(h.repo, h.playerID)
		if err != nil {
			t.Fatalf("get room snapshot failed: %v", err)
		}
		npcs := extractRoomNPCs(snapshot.Room)
		for _, npc := range npcs {
			if npc["id"] == "road_bandit" {
				attributes, _ := npc["attributes"].(map[string]interface{})
				if npc["state"] == "DEAD" && attributes["currentHp"] == 0 {
					defeated = true
				}
				break
			}
		}
		if defeated {
			break
		}
		status, err := GetPlayerStatusView(h.repo, h.playerID)
		if err != nil {
			t.Fatalf("get status after kill failed: %v", err)
		}
		if status["state"] == "DEAD" {
			t.Fatalf("player died before defeating road bandit")
		}
	}
	if !defeated {
		t.Fatalf("expected road bandit corpse to remain with zero HP")
	}
}

func TestDungeonAggressiveNPCWaitsForPlayerAction(t *testing.T) {
	h := setupRepo(t)

	if _, err := EnterDungeonWithTransition(h.repo, h.playerID, "abandoned_temple_rain"); err != nil {
		t.Fatalf("enter dungeon failed: %v", err)
	}
	tickRes, err := ProcessCombatTickWithEffects(h.repo)
	if err != nil {
		t.Fatalf("process combat tick failed: %v", err)
	}
	foundAggro := false
	for _, msg := range tickRes.Messages {
		if strings.Contains(msg.Message, "扑向") {
			foundAggro = true
			break
		}
	}
	if foundAggro {
		t.Fatalf("dungeon npc should wait for player action, got: %+v", tickRes.Messages)
	}
	status, err := GetPlayerStatusView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get player status failed: %v", err)
	}
	if status["state"] != "IDLE" {
		t.Fatalf("expected player to remain idle near hostile npc, got %+v", status)
	}
}
