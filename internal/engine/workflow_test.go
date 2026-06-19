package engine

import (
	"strings"
	"testing"

	"mudgame/server/internal/manager"
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
	npcs := extractRoomNPCs(snapshot.Room)
	targetID := ""
	for _, npc := range npcs {
		if npc["name"] == npcName {
			targetID, _ = npc["id"].(string)
			break
		}
	}
	if targetID == "" {
		t.Fatalf("expected npc %s in room, got %+v", npcName, npcs)
	}
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
			return
		}
	}
	status, _ := GetPlayerStatusView(repo, playerID)
	snapshot, _ = GetRoomSnapshotView(repo, playerID)
	t.Fatalf("combat with %s did not finish, status=%+v roomNPCs=%+v", npcName, status, extractRoomNPCs(snapshot.Room))
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
	if _, err := ChooseDialogueWithEffects(h.repo, h.playerID, "escort_bureau_chief", "ask_work"); err != nil {
		t.Fatalf("start quest dialogue failed: %v", err)
	}
	quests, err := GetQuestsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get quests failed: %v", err)
	}
	if len(quests.Quests) != 1 || quests.Quests[0]["status"] != "active" {
		t.Fatalf("expected active quest, got %+v", quests.Quests)
	}

	if _, err := MoveWithTransition(h.repo, h.playerID, "west"); err != nil {
		t.Fatalf("move to inn failed: %v", err)
	}
	if _, err := ChooseDialogueWithEffects(h.repo, h.playerID, "storyteller", "ask_rumor"); err != nil {
		t.Fatalf("hear rumor dialogue failed: %v", err)
	}
	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move back to south gate failed: %v", err)
	}
	if _, err := ChooseDialogueWithEffects(h.repo, h.playerID, "escort_bureau_chief", "turn_in_rumor"); err != nil {
		t.Fatalf("complete quest dialogue failed: %v", err)
	}
	quests, err = GetQuestsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get quests after completion failed: %v", err)
	}
	if len(quests.Quests) != 1 || quests.Quests[0]["status"] != "completed" {
		t.Fatalf("expected completed quest, got %+v", quests.Quests)
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
	_ = StopWithEffects(h.repo, h.playerID)

	attrsAfterSkill, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attributes after skill failed: %v", err)
	}
	if attrsAfterSkill["maxMp"].(int) <= initialMaxMP {
		t.Fatalf("expected basic_neigong to increase maxMp, before=%d after=%d", initialMaxMP, attrsAfterSkill["maxMp"].(int))
	}

	skills, err := GetSkillsView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get skills failed: %v", err)
	}
	hasBonusText := false
	for _, skill := range skills {
		if skill["id"] == "basic_neigong" {
			if text, ok := skill["bonusText"].([]string); ok && len(text) > 0 {
				hasBonusText = true
			}
		}
	}
	if !hasBonusText {
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
	attrsAfterGather, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attrs after gather failed: %v", err)
	}
	if attrsAfterGather["experience"].(int) <= attrsBeforeGather["experience"].(int) {
		t.Fatalf("expected gather to increase experience")
	}
	moveBackFromOutskirtsToSpawn(t, h.repo, h.playerID)
	if _, err := MeditateWithEffects(h.repo, h.playerID); err != nil {
		t.Fatalf("meditate failed: %v", err)
	}
	runStatusTicks(t, h.repo, 10)
	if stop := StopWithEffects(h.repo, h.playerID); stop.Message != "你停止了打坐。" {
		t.Fatalf("unexpected stop meditate message: %+v", stop)
	}

	hpDropped := false
	for attempt := 0; attempt < 3 && !hpDropped; attempt++ {
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
		stillPresent := false
		for _, npc := range npcs {
			if npc["id"] == "road_bandit" {
				stillPresent = true
				break
			}
		}
		if !stillPresent {
			defeated = true
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
		t.Fatalf("expected road bandit to be removed after kill")
	}
}

func TestDungeonAggressiveNPCStartsCombat(t *testing.T) {
	h := setupRepo(t)

	if _, err := EnterDungeonWithTransition(h.repo, h.playerID, "abandoned_temple_rain"); err != nil {
		t.Fatalf("enter dungeon failed: %v", err)
	}
	if _, err := MoveWithTransition(h.repo, h.playerID, "north"); err != nil {
		t.Fatalf("move to aggressive npc room failed: %v", err)
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
	if !foundAggro {
		t.Fatalf("expected aggressive dungeon npc message, got: %+v", tickRes.Messages)
	}
}

func TestXunqinDungeonUnlocksFromStorytellerAndRewardsItem(t *testing.T) {
	h := setupRepo(t)

	dungeons := GetDungeonsView(h.repo, h.playerID)
	for _, dungeon := range dungeons {
		if dungeon["id"] == "xunqin_ch01_bronze_gate" {
			t.Fatalf("xunqin dungeon should start locked, got %+v", dungeons)
		}
	}

	if _, err := MoveWithTransition(h.repo, h.playerID, "west"); err != nil {
		t.Fatalf("move to west market failed: %v", err)
	}
	if _, err := MoveWithTransition(h.repo, h.playerID, "south"); err != nil {
		t.Fatalf("move to inn failed: %v", err)
	}
	if _, err := ChooseDialogueWithEffects(h.repo, h.playerID, "storyteller", "ask_bronze_box"); err != nil {
		t.Fatalf("unlock xunqin dialogue failed: %v", err)
	}

	dungeons = GetDungeonsView(h.repo, h.playerID)
	found := false
	for _, dungeon := range dungeons {
		if dungeon["id"] == "xunqin_ch01_bronze_gate" {
			found = true
			if dungeon["chainId"] != "xunqin_legend" || dungeon["chapter"] != 1 {
				t.Fatalf("expected xunqin chain metadata, got %+v", dungeon)
			}
		}
	}
	if !found {
		t.Fatalf("expected xunqin dungeon after dialogue unlock, got %+v", dungeons)
	}

	enterRes, err := EnterDungeonWithTransition(h.repo, h.playerID, "xunqin_ch01_bronze_gate")
	if err != nil {
		t.Fatalf("enter xunqin dungeon failed: %v", err)
	}
	if progress, _ := enterRes.Room["dungeon"].(map[string]interface{})["progress"].(int); progress != 0 {
		t.Fatalf("expected initial xunqin progress 0, got %+v", enterRes.Room["dungeon"])
	}
	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move to bronze hall failed: %v", err)
	}
	moveRes, err := MoveWithTransition(h.repo, h.playerID, "east")
	if err != nil {
		t.Fatalf("move to bronze mechanism failed: %v", err)
	}
	actions, _ := moveRes.Room["actions"].([]string)
	hasSearch := false
	for _, action := range actions {
		if action == "search" {
			hasSearch = true
			break
		}
	}
	if !hasSearch {
		t.Fatalf("expected search action in bronze mechanism room, got %+v", moveRes.Room["actions"])
	}
	searchRes, err := DoRoomActionWithEffects(h.repo, h.playerID, "search")
	if err != nil {
		t.Fatalf("search bronze mechanism failed: %v", err)
	}
	if searchRes.Message == "" {
		t.Fatalf("search should return message")
	}

	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move to warring states rift failed: %v", err)
	}
	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move to village failed: %v", err)
	}
	killRoomNPCByName(t, h.repo, h.playerID, "荒村恶霸")

	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move to ambush road failed: %v", err)
	}
	killRoomNPCByName(t, h.repo, h.playerID, "赵国游骑")
	killRoomNPCByName(t, h.repo, h.playerID, "赵国游骑")

	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move to bamboo road failed: %v", err)
	}
	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move to abandoned camp failed: %v", err)
	}
	killRoomNPCByName(t, h.repo, h.playerID, "赵国游骑")
	killRoomNPCByName(t, h.repo, h.playerID, "赵府门客")

	moveRes, err = MoveWithTransition(h.repo, h.playerID, "east")
	if err != nil {
		t.Fatalf("move to fragment desk failed: %v", err)
	}
	actions, _ = moveRes.Room["actions"].([]string)
	hasInspect := false
	for _, action := range actions {
		if action == "inspect_fragment" {
			hasInspect = true
			break
		}
	}
	if !hasInspect {
		t.Fatalf("expected inspect_fragment action, got %+v", moveRes.Room["actions"])
	}
	inspectRes, err := DoRoomActionWithEffects(h.repo, h.playerID, "inspect_fragment")
	if err != nil {
		t.Fatalf("inspect fragment failed: %v", err)
	}
	if inspectRes.Message == "" {
		t.Fatalf("inspect fragment should return message")
	}

	if _, err := MoveWithTransition(h.repo, h.playerID, "east"); err != nil {
		t.Fatalf("move to broken bridge failed: %v", err)
	}
	killRoomNPCByName(t, h.repo, h.playerID, "门客剑士")

	moveRes, err = MoveWithTransition(h.repo, h.playerID, "east")
	if err != nil {
		t.Fatalf("move to bronze gate echo failed: %v", err)
	}
	dungeonPayload, _ := moveRes.Room["dungeon"].(map[string]interface{})
	if dungeonPayload["progress"] != 100 {
		t.Fatalf("expected xunqin progress 100 at bronze gate echo, got %+v", dungeonPayload)
	}

	backpack, err := GetBackpackView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get backpack failed: %v", err)
	}
	foundBronze := false
	foundZongheng := false
	for _, item := range backpack.Items {
		if item["id"] == "bronze_fragment" && item["count"] == 1 {
			foundBronze = true
		}
		if item["id"] == "zongheng_fragment" && item["count"] == 1 {
			foundZongheng = true
		}
	}
	if !foundBronze || !foundZongheng {
		t.Fatalf("expected xunqin clue items, bronze=%v zongheng=%v backpack=%+v", foundBronze, foundZongheng, backpack.Items)
	}

	if _, err := LeaveDungeonWithTransition(h.repo, h.playerID); err != nil {
		t.Fatalf("leave xunqin dungeon failed: %v", err)
	}
	status, err := GetPlayerStatusView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get status after leaving dungeon failed: %v", err)
	}
	if status["roomId"] != "ly_inn" {
		t.Fatalf("expected player to return to entry room after dungeon, got %+v", status)
	}
	attrs, err := GetAttributesView(h.repo, h.playerID)
	if err != nil {
		t.Fatalf("get attributes after dungeon failed: %v", err)
	}
	if attrs["experience"].(int) < 380 || attrs["potential"].(int) < 240 {
		t.Fatalf("expected xunqin settlement rewards, got %+v", attrs)
	}
}
