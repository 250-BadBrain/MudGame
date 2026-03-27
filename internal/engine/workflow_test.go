package engine

import (
	"testing"

	"mudgame/server/internal/storage"
)

const (
	workflowShopNPCID    = "shopkeeper_jia"
	workflowTeacherNPCID = "zhao_lao_jiaoxi"
	workflowShopItemID   = "common_clothes"
)

func moveToMineFromSpawn(t *testing.T, svc *Service, playerID string) {
	t.Helper()
	path := []string{"east", "east", "east", "east", "east", "south"}
	for _, dir := range path {
		if _, err := svc.MoveWithTransition(playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveBackFromMineToSpawn(t *testing.T, svc *Service, playerID string) {
	t.Helper()
	path := []string{"north", "west", "west", "west", "west", "west"}
	for _, dir := range path {
		if _, err := svc.MoveWithTransition(playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveToShopFromSpawn(t *testing.T, svc *Service, playerID string) {
	t.Helper()
	path := []string{"west", "south"}
	for _, dir := range path {
		if _, err := svc.MoveWithTransition(playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveToTeacherFromSpawn(t *testing.T, svc *Service, playerID string) {
	t.Helper()
	path := []string{"south", "west"}
	for _, dir := range path {
		if _, err := svc.MoveWithTransition(playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func setupService(t *testing.T) (*Service, string, string) {
	t.Helper()
	repo := storage.NewFileRepository(t.TempDir())
	svc := NewService(repo)

	username := "tester"
	password := "pass123"
	if err := svc.Register(username, password); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := svc.Login(username, password); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	playerID, err := svc.CreateCharacter(username, "hero", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	return svc, username, playerID
}

func TestReconnectSuccess(t *testing.T) {
	svc, username, playerID := setupService(t)
	token, err := svc.Login(username, "pass123")
	if err != nil {
		t.Fatalf("login for token failed: %v", err)
	}

	res, err := svc.Reconnect(username, "", token, playerID)
	if err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}
	if res.UserID != username {
		t.Fatalf("unexpected user id: %s", res.UserID)
	}
	if res.PlayerID != playerID {
		t.Fatalf("unexpected player id: %s", res.PlayerID)
	}
}

func TestReconnectInvalidToken(t *testing.T) {
	svc, username, playerID := setupService(t)
	_, err := svc.Reconnect(username, "", "bad-token", playerID)
	if err == nil {
		t.Fatalf("expected reconnect error")
	}
}

func TestMoveAndEnterWorldTransitions(t *testing.T) {
	svc, _, playerID := setupService(t)

	moveRes, err := svc.MoveWithTransition(playerID, "north")
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}
	if moveRes.FromRoomID == moveRes.ToRoomID {
		t.Fatalf("expected different room after move")
	}

	enterRes, err := svc.EnterWorldWithTransition(playerID, "wudang")
	if err != nil {
		t.Fatalf("enter world failed: %v", err)
	}
	if enterRes.ToRoomID != "wd_00" {
		t.Fatalf("unexpected target room: %s", enterRes.ToRoomID)
	}
}

func TestStateAndCombatWorkflows(t *testing.T) {
	svc, username, p1 := setupService(t)

	if err := svc.Register("tester2", "pass123"); err != nil {
		t.Fatalf("register tester2 failed: %v", err)
	}
	if _, err := svc.Login("tester2", "pass123"); err != nil {
		t.Fatalf("login tester2 failed: %v", err)
	}
	p2, err := svc.CreateCharacter("tester2", "hero2", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create tester2 character failed: %v", err)
	}
	if err := svc.Register("tester3", "pass123"); err != nil {
		t.Fatalf("register tester3 failed: %v", err)
	}
	if _, err := svc.Login("tester3", "pass123"); err != nil {
		t.Fatalf("login tester3 failed: %v", err)
	}
	p3, err := svc.CreateCharacter("tester3", "hero3", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create tester3 character failed: %v", err)
	}
	if err := svc.Register("tester4", "pass123"); err != nil {
		t.Fatalf("register tester4 failed: %v", err)
	}
	if _, err := svc.Login("tester4", "pass123"); err != nil {
		t.Fatalf("login tester4 failed: %v", err)
	}
	p4, err := svc.CreateCharacter("tester4", "hero4", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create tester4 character failed: %v", err)
	}

	if _, err := svc.Login(username, "pass123"); err != nil {
		t.Fatalf("refresh login failed: %v", err)
	}
	if _, err := svc.EquipWithEffects(p1, "iron_pickaxe"); err != nil {
		t.Fatalf("equip pickaxe failed: %v", err)
	}

	moveToMineFromSpawn(t, svc, p1)
	mineRes, err := svc.MineWithEffects(p1)
	if err != nil {
		t.Fatalf("mine workflow failed: %v", err)
	}
	if len(mineRes.Logs) == 0 {
		t.Fatalf("unexpected mine result: %+v", mineRes)
	}

	_, err = svc.HealWithEffects(p1)
	if err == nil || err.Error() != "mining_state" {
		t.Fatalf("heal should return mining_state while mining, got: %v", err)
	}
	if stop := svc.StopWithEffects(p1); stop.Message != "你收起铁镐，停止了挖矿。" {
		t.Fatalf("unexpected stop mining message: %+v", stop)
	}

	moveBackFromMineToSpawn(t, svc, p1)
	combatRes, err := svc.SparWithEffects(p1, p2)
	if err != nil {
		t.Fatalf("spar workflow failed: %v", err)
	}
	if !combatRes.NeedRoomMessage || !combatRes.NeedRoomUpdate {
		t.Fatalf("unexpected combat result: %+v", combatRes)
	}
	stopRes := svc.StopWithEffects(p1)
	if stopRes.Message != "你正在战斗，待会再说" {
		t.Fatalf("unexpected stop message in combat: %+v", stopRes)
	}
	if stopRes.NeedRoomUpdate {
		t.Fatalf("stop in combat should not trigger room update")
	}

	killRes, err := svc.KillWithEffects(p3, p4)
	if err != nil {
		t.Fatalf("kill workflow failed: %v", err)
	}
	if killRes.Message == "" {
		t.Fatalf("kill message should not be empty")
	}
	deadID := ""
	for i := 0; i < 800; i++ {
		if _, err := svc.ProcessCombatTickWithEffects(); err != nil {
			t.Fatalf("process combat tick failed: %v", err)
		}
		status4, err := svc.GetPlayerStatusView(p4)
		if err != nil {
			t.Fatalf("get player4 status failed: %v", err)
		}
		if state, _ := status4["state"].(string); state == "DEAD" {
			deadID = p4
			break
		}
		status3, err := svc.GetPlayerStatusView(p3)
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

	reliveRes, err := svc.ReliveWithEffects(deadID)
	if err != nil {
		t.Fatalf("relive workflow failed: %v", err)
	}
	if reliveRes.Action != "refresh_room" {
		t.Fatalf("unexpected relive action: %s", reliveRes.Action)
	}
}

func TestShopAndInventoryWorkflows(t *testing.T) {
	svc, _, playerID := setupService(t)

	moveToShopFromSpawn(t, svc, playerID)
	shop, err := svc.GetShopView(playerID, workflowShopNPCID)
	if err != nil {
		t.Fatalf("get shop view failed: %v", err)
	}
	if shop.NPCID != workflowShopNPCID || len(shop.Items) == 0 {
		t.Fatalf("unexpected shop payload: %+v", shop)
	}

	buyRes, err := svc.BuyWithReceipt(playerID, workflowShopNPCID, workflowShopItemID, 1)
	if err != nil {
		t.Fatalf("buy workflow failed: %v", err)
	}
	if buyRes.BoughtAmount != 1 || len(buyRes.Logs) == 0 {
		t.Fatalf("unexpected buy result: %+v", buyRes)
	}

	equipRes, err := svc.EquipWithEffects(playerID, workflowShopItemID)
	if err != nil {
		t.Fatalf("equip workflow failed: %v", err)
	}
	if equipRes.Message == "" || !equipRes.NeedRoomUpdate {
		t.Fatalf("unexpected equip result: %+v", equipRes)
	}

	unequipRes, err := svc.UnequipWithEffects(playerID, "BODY")
	if err != nil {
		t.Fatalf("unequip workflow failed: %v", err)
	}
	if unequipRes.Message == "" || !unequipRes.NeedRoomUpdate {
		t.Fatalf("unexpected unequip result: %+v", unequipRes)
	}

	if _, err := svc.EnterWorldWithTransition(playerID, "luoyang"); err != nil {
		t.Fatalf("reset to luoyang spawn failed: %v", err)
	}
	moveToTeacherFromSpawn(t, svc, playerID)
	learnRes, err := svc.LearnWithEffects(playerID, workflowTeacherNPCID, "basic_fist")
	if err != nil {
		t.Fatalf("learn workflow failed: %v", err)
	}
	if learnRes.Message == "" {
		t.Fatalf("unexpected learn result: %+v", learnRes)
	}
	_ = svc.StopWithEffects(playerID)

	discardMsg, err := svc.DiscardWithMessage(playerID, "iron_pickaxe", 1)
	if err != nil {
		t.Fatalf("discard workflow failed: %v", err)
	}
	if discardMsg == "" {
		t.Fatalf("discard message should not be empty")
	}

	sellRes, err := svc.SellWithReceipt(playerID, workflowShopItemID, 1)
	if err != nil {
		t.Fatalf("sell workflow failed: %v", err)
	}
	if sellRes.Gain <= 0 || sellRes.Money <= 0 {
		t.Fatalf("unexpected sell result: %+v", sellRes)
	}
}

func TestQueryViewWorkflows(t *testing.T) {
	svc, _, playerID := setupService(t)

	roomRes, err := svc.GetRoomSnapshotView(playerID)
	if err != nil {
		t.Fatalf("get room snapshot view failed: %v", err)
	}
	if roomRes.Room["id"] == "" || roomRes.Player["id"] == "" {
		t.Fatalf("unexpected room snapshot payload: %+v", roomRes)
	}

	attrs, err := svc.GetAttributesView(playerID)
	if err != nil {
		t.Fatalf("get attributes view failed: %v", err)
	}
	if attrs["currentHp"] == nil {
		t.Fatalf("attributes missing currentHp")
	}

	bag, err := svc.GetBackpackView(playerID)
	if err != nil {
		t.Fatalf("get backpack view failed: %v", err)
	}
	if len(bag.Items) == 0 || bag.Money <= 0 {
		t.Fatalf("unexpected backpack payload: %+v", bag)
	}

	skills, err := svc.GetSkillsView(playerID)
	if err != nil {
		t.Fatalf("get skills view failed: %v", err)
	}
	if len(skills) == 0 {
		t.Fatalf("skills should not be empty")
	}

	eq, err := svc.GetEquipmentView(playerID)
	if err != nil {
		t.Fatalf("get equipment view failed: %v", err)
	}
	if len(eq) != 0 {
		t.Fatalf("new character should have empty equipment")
	}

	worlds := svc.GetWorldsView()
	if len(worlds) == 0 {
		t.Fatalf("world list should not be empty")
	}
}

func TestEventViewWorkflows(t *testing.T) {
	svc, _, playerID := setupService(t)

	status, err := svc.GetPlayerStatusView(playerID)
	if err != nil {
		t.Fatalf("get player status view failed: %v", err)
	}
	roomID, _ := status["roomId"].(string)
	if roomID == "" {
		t.Fatalf("status roomId should not be empty")
	}

	room, err := svc.GetRoomPayloadView(roomID)
	if err != nil {
		t.Fatalf("get room payload view failed: %v", err)
	}
	if room["id"] == "" {
		t.Fatalf("room payload id should not be empty")
	}

	update, err := svc.GetRoomUpdateView(roomID)
	if err != nil {
		t.Fatalf("get room update view failed: %v", err)
	}
	if len(update.Players) == 0 {
		t.Fatalf("room update players should not be empty")
	}

	rid2, rname2, err := svc.GetPlayerRoomView(playerID)
	if err != nil {
		t.Fatalf("get player room view failed: %v", err)
	}
	if rid2 == "" || rname2 == "" {
		t.Fatalf("unexpected player room view: %s %s", rid2, rname2)
	}

	if !svc.IsPlayerInRoomView(playerID, rid2) {
		t.Fatalf("player should be in returned room")
	}
}

func TestStatusTickWorkflow(t *testing.T) {
	svc, _, p1 := setupService(t)

	if err := svc.Register("ticker2", "pass123"); err != nil {
		t.Fatalf("register ticker2 failed: %v", err)
	}
	if _, err := svc.Login("ticker2", "pass123"); err != nil {
		t.Fatalf("login ticker2 failed: %v", err)
	}
	p2, err := svc.CreateCharacter("ticker2", "ticker2hero", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create ticker2 character failed: %v", err)
	}

	if _, err := svc.EquipWithEffects(p1, "iron_pickaxe"); err != nil {
		t.Fatalf("equip pickaxe failed: %v", err)
	}
	moveToMineFromSpawn(t, svc, p1)
	if _, err := svc.MineWithEffects(p1); err != nil {
		t.Fatalf("mine failed: %v", err)
	}
	attrsBeforeMine, err := svc.GetAttributesView(p1)
	if err != nil {
		t.Fatalf("get attrs before mine tick failed: %v", err)
	}
	if _, err := svc.ProcessStatusTickWithEffects(); err != nil {
		t.Fatalf("process status tick for mine failed: %v", err)
	}
	attrsAfterMine, err := svc.GetAttributesView(p1)
	if err != nil {
		t.Fatalf("get attrs after mine tick failed: %v", err)
	}
	if attrsAfterMine["experience"].(int) <= attrsBeforeMine["experience"].(int) {
		t.Fatalf("expected mine tick to increase experience")
	}
	if stop := svc.StopWithEffects(p1); stop.Message != "你收起铁镐，停止了挖矿。" {
		t.Fatalf("unexpected stop mining message: %+v", stop)
	}

	moveBackFromMineToSpawn(t, svc, p1)
	if _, err := svc.MeditateWithEffects(p1); err != nil {
		t.Fatalf("meditate failed: %v", err)
	}
	if _, err := svc.ProcessStatusTickWithEffects(); err != nil {
		t.Fatalf("process status tick for meditate failed: %v", err)
	}
	if stop := svc.StopWithEffects(p1); stop.Message != "你停止了打坐。" {
		t.Fatalf("unexpected stop meditate message: %+v", stop)
	}

	hpDropped := false
	for attempt := 0; attempt < 3 && !hpDropped; attempt++ {
		if _, err := svc.SparWithEffects(p1, p2); err != nil {
			t.Fatalf("spar failed: %v", err)
		}
		for i := 0; i < 80; i++ {
			if _, err := svc.ProcessCombatTickWithEffects(); err != nil {
				t.Fatalf("process combat tick failed: %v", err)
			}
			st1, err := svc.GetPlayerStatusView(p1)
			if err != nil {
				t.Fatalf("get player1 status failed: %v", err)
			}
			attrsNow, err := svc.GetAttributesView(p1)
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
	if _, err := svc.HealWithEffects(p1); err != nil {
		t.Fatalf("heal failed: %v", err)
	}
	attrsBeforeHealTick, err := svc.GetAttributesView(p1)
	if err != nil {
		t.Fatalf("get attrs before heal tick failed: %v", err)
	}
	if _, err := svc.ProcessStatusTickWithEffects(); err != nil {
		t.Fatalf("process status tick for heal failed: %v", err)
	}
	attrsAfterHealTick, err := svc.GetAttributesView(p1)
	if err != nil {
		t.Fatalf("get attrs after heal tick failed: %v", err)
	}
	if attrsAfterHealTick["currentHp"].(int) < attrsBeforeHealTick["currentHp"].(int) {
		t.Fatalf("heal tick should not decrease currentHp")
	}
}

func TestSparAutoStopWorkflow(t *testing.T) {
	svc, _, p1 := setupService(t)

	if err := svc.Register("spar2", "pass123"); err != nil {
		t.Fatalf("register spar2 failed: %v", err)
	}
	if _, err := svc.Login("spar2", "pass123"); err != nil {
		t.Fatalf("login spar2 failed: %v", err)
	}
	p2, err := svc.CreateCharacter("spar2", "spar2hero", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create spar2 character failed: %v", err)
	}

	if _, err := svc.SparWithEffects(p1, p2); err != nil {
		t.Fatalf("spar failed: %v", err)
	}

	idle1, idle2 := false, false
	for i := 0; i < 40; i++ {
		if _, err := svc.ProcessCombatTickWithEffects(); err != nil {
			t.Fatalf("process combat tick failed: %v", err)
		}
		st1, err := svc.GetPlayerStatusView(p1)
		if err != nil {
			t.Fatalf("get player1 status failed: %v", err)
		}
		st2, err := svc.GetPlayerStatusView(p2)
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
	svc, _, p1 := setupService(t)

	if err := svc.Register("loss2", "pass123"); err != nil {
		t.Fatalf("register loss2 failed: %v", err)
	}
	if _, err := svc.Login("loss2", "pass123"); err != nil {
		t.Fatalf("login loss2 failed: %v", err)
	}
	p2, err := svc.CreateCharacter("loss2", "loss2hero", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create loss2 character failed: %v", err)
	}

	if _, err := svc.SparWithEffects(p1, p2); err != nil {
		t.Fatalf("spar failed: %v", err)
	}
	if err := svc.DeleteCharacter("loss2", p2); err != nil {
		t.Fatalf("delete opponent failed: %v", err)
	}

	tickRes, err := svc.ProcessCombatTickWithEffects()
	if err != nil {
		t.Fatalf("process combat tick failed: %v", err)
	}
	if len(tickRes.Messages) == 0 {
		t.Fatalf("expected target-loss combat message")
	}

	status, err := svc.GetPlayerStatusView(p1)
	if err != nil {
		t.Fatalf("get player status failed: %v", err)
	}
	if status["state"] != "IDLE" {
		t.Fatalf("expected attacker leave combat after target loss, got %v", status["state"])
	}
}
