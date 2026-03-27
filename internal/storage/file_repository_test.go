package storage

import "testing"

const (
	shopNPCID    = "shopkeeper_jia"
	teacherNPCID = "zhao_lao_jiaoxi"
	shopItemID   = "common_clothes"
)

func moveToMineFromLuoyangSpawn(t *testing.T, repo *FileRepository, playerID string) {
	t.Helper()
	path := []string{"east", "east", "east", "east", "east", "south"}
	for _, dir := range path {
		if _, err := repo.Move(playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveBackToLuoyangSpawnFromMine(t *testing.T, repo *FileRepository, playerID string) {
	t.Helper()
	path := []string{"north", "west", "west", "west", "west", "west"}
	for _, dir := range path {
		if _, err := repo.Move(playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveToShopkeeperFromLuoyangSpawn(t *testing.T, repo *FileRepository, playerID string) {
	t.Helper()
	path := []string{"west", "south"}
	for _, dir := range path {
		if _, err := repo.Move(playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func moveToTeacherFromLuoyangSpawn(t *testing.T, repo *FileRepository, playerID string) {
	t.Helper()
	path := []string{"south", "west"}
	for _, dir := range path {
		if _, err := repo.Move(playerID, dir); err != nil {
			t.Fatalf("move %s failed: %v", dir, err)
		}
	}
}

func TestFileRepositoryUserAndCharacterFlow(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	if err := repo.Register("u1", "p1"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	token, err := repo.Login("u1", "p1")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if !repo.IsReconnectTokenValid("u1", token) {
		t.Fatalf("token should be valid")
	}

	pid, err := repo.CreateCharacter("u1", "hero", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}
	if !repo.CharacterBelongsToUser("u1", pid) {
		t.Fatalf("character should belong to user")
	}

	chars := repo.ListCharacters("u1")
	if len(chars) != 1 {
		t.Fatalf("expected 1 character, got %d", len(chars))
	}

	if err := repo.DeleteCharacter("u1", pid); err != nil {
		t.Fatalf("delete character failed: %v", err)
	}
	if got := len(repo.ListCharacters("u1")); got != 0 {
		t.Fatalf("expected 0 characters after delete, got %d", got)
	}
}

func TestFileRepositoryPersistenceAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	repo1 := NewFileRepository(dir)

	if err := repo1.Register("u2", "p2"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := repo1.Login("u2", "p2"); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	pid, err := repo1.CreateCharacter("u2", "hero2", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	repo2 := NewFileRepository(dir)
	chars := repo2.ListCharacters("u2")
	if len(chars) != 1 {
		t.Fatalf("expected persisted character, got %d", len(chars))
	}
	if !repo2.CharacterBelongsToUser("u2", pid) {
		t.Fatalf("persisted ownership check failed")
	}
}

func TestFileRepositoryRoomAndMoveFlow(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	if err := repo.Register("u3", "p3"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := repo.Login("u3", "p3"); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	pid, err := repo.CreateCharacter("u3", "hero3", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	roomID, roomName, err := repo.GetPlayerRoom(pid)
	if err != nil {
		t.Fatalf("get player room failed: %v", err)
	}
	if roomID == "" || roomName == "" {
		t.Fatalf("unexpected room info: %s %s", roomID, roomName)
	}

	worlds := repo.GetWorlds()
	if len(worlds) == 0 {
		t.Fatalf("worlds should not be empty")
	}

	if _, err := repo.Move(pid, "north"); err != nil {
		t.Fatalf("move north failed: %v", err)
	}
	roomID2, _, err := repo.GetPlayerRoom(pid)
	if err != nil {
		t.Fatalf("get room after move failed: %v", err)
	}
	if roomID2 != "ly_01" {
		t.Fatalf("expected ly_01 after north move, got %s", roomID2)
	}

	if _, err := repo.EnterWorld(pid, "wudang"); err != nil {
		t.Fatalf("enter world failed: %v", err)
	}
	roomID3, _, err := repo.GetPlayerRoom(pid)
	if err != nil {
		t.Fatalf("get room after enter world failed: %v", err)
	}
	if roomID3 != "wd_00" {
		t.Fatalf("expected wd_00 after enter world, got %s", roomID3)
	}

	roomPayload, playerPayload, err := repo.GetRoomSnapshot(pid)
	if err != nil {
		t.Fatalf("get room snapshot failed: %v", err)
	}
	if roomPayload["id"] == "" || playerPayload["id"] == "" {
		t.Fatalf("invalid snapshot payload")
	}

	status, err := repo.GetPlayerStatus(pid)
	if err != nil {
		t.Fatalf("get player status failed: %v", err)
	}
	if status["roomId"] != "wd_00" {
		t.Fatalf("unexpected status room: %v", status["roomId"])
	}
}

func TestFileRepositoryPanelsAndShopFlow(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	if err := repo.Register("u4", "p4"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := repo.Login("u4", "p4"); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	pid, err := repo.CreateCharacter("u4", "hero4", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	attrs, err := repo.GetAttributes(pid)
	if err != nil {
		t.Fatalf("get attributes failed: %v", err)
	}
	if attrs["currentHp"] == nil {
		t.Fatalf("attributes missing currentHp")
	}

	bag, money, err := repo.GetBackpack(pid)
	if err != nil {
		t.Fatalf("get backpack failed: %v", err)
	}
	if len(bag) == 0 || money <= 0 {
		t.Fatalf("unexpected backpack data")
	}

	skills, err := repo.GetSkills(pid)
	if err != nil {
		t.Fatalf("get skills failed: %v", err)
	}
	if len(skills) == 0 {
		t.Fatalf("skills should not be empty")
	}

	eq, err := repo.GetEquipment(pid)
	if err != nil {
		t.Fatalf("get equipment failed: %v", err)
	}
	if len(eq) != 0 {
		t.Fatalf("new character should have empty equipment")
	}

	moveToShopkeeperFromLuoyangSpawn(t, repo, pid)

	shopItems, npcID, _, _, err := repo.GetShop(pid, shopNPCID)
	if err != nil {
		t.Fatalf("get shop failed: %v", err)
	}
	if npcID != shopNPCID || len(shopItems) == 0 {
		t.Fatalf("unexpected shop response")
	}

	bought, money2, err := repo.Buy(pid, shopNPCID, shopItemID, 1)
	if err != nil {
		t.Fatalf("buy failed: %v", err)
	}
	if bought != 1 || money2 >= money {
		t.Fatalf("buy result unexpected: bought=%d money2=%d money=%d", bought, money2, money)
	}

	gain, money3, err := repo.SellItem(pid, shopItemID, 1)
	if err != nil {
		t.Fatalf("sell failed: %v", err)
	}
	if gain <= 0 || money3 <= money2 {
		t.Fatalf("sell result unexpected: gain=%d money3=%d money2=%d", gain, money3, money2)
	}

	if err := repo.DiscardItem(pid, "iron_pickaxe", 1); err != nil {
		t.Fatalf("discard failed: %v", err)
	}
}

func TestFileRepositoryActionsFlow(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	if err := repo.Register("u5", "p5"); err != nil {
		t.Fatalf("register u5 failed: %v", err)
	}
	if _, err := repo.Login("u5", "p5"); err != nil {
		t.Fatalf("login u5 failed: %v", err)
	}
	pid1, err := repo.CreateCharacter("u5", "hero5", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create character u5 failed: %v", err)
	}

	if err := repo.Register("u6", "p6"); err != nil {
		t.Fatalf("register u6 failed: %v", err)
	}
	if _, err := repo.Login("u6", "p6"); err != nil {
		t.Fatalf("login u6 failed: %v", err)
	}
	pid2, err := repo.CreateCharacter("u6", "hero6", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create character u6 failed: %v", err)
	}
	if err := repo.Register("u7", "p7"); err != nil {
		t.Fatalf("register u7 failed: %v", err)
	}
	if _, err := repo.Login("u7", "p7"); err != nil {
		t.Fatalf("login u7 failed: %v", err)
	}
	pid3, err := repo.CreateCharacter("u7", "hero7", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create character u7 failed: %v", err)
	}
	if err := repo.Register("u8", "p8"); err != nil {
		t.Fatalf("register u8 failed: %v", err)
	}
	if _, err := repo.Login("u8", "p8"); err != nil {
		t.Fatalf("login u8 failed: %v", err)
	}
	pid4, err := repo.CreateCharacter("u8", "hero8", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create character u8 failed: %v", err)
	}

	moveToShopkeeperFromLuoyangSpawn(t, repo, pid1)
	if _, _, err := repo.Buy(pid1, shopNPCID, shopItemID, 1); err != nil {
		t.Fatalf("buy shop item failed: %v", err)
	}
	if err := repo.EquipItem(pid1, shopItemID); err != nil {
		t.Fatalf("equip failed: %v", err)
	}
	eq, err := repo.GetEquipment(pid1)
	if err != nil {
		t.Fatalf("get equipment failed: %v", err)
	}
	if _, ok := eq["BODY"]; !ok {
		t.Fatalf("expected BODY equipment slot after equip")
	}
	if err := repo.UnequipItem(pid1, "BODY"); err != nil {
		t.Fatalf("unequip failed: %v", err)
	}
	eq2, err := repo.GetEquipment(pid1)
	if err != nil {
		t.Fatalf("get equipment after unequip failed: %v", err)
	}
	if len(eq2) != 0 {
		t.Fatalf("expected empty equipment after unequip")
	}

	skills1, err := repo.GetSkills(pid1)
	if err != nil {
		t.Fatalf("get skills before learn failed: %v", err)
	}
	before := 0
	for _, s := range skills1 {
		if id, ok := s["id"].(string); ok && id == "basic_fist" {
			if lvl, ok := s["level"].(int); ok {
				before = lvl
			}
		}
	}
	if _, err := repo.EnterWorld(pid1, "luoyang"); err != nil {
		t.Fatalf("reset to luoyang spawn failed: %v", err)
	}
	moveToTeacherFromLuoyangSpawn(t, repo, pid1)
	if err := repo.LearnSkill(pid1, teacherNPCID, "basic_fist"); err != nil {
		t.Fatalf("learn skill failed: %v", err)
	}
	_ = repo.Stop(pid1)
	skills2, err := repo.GetSkills(pid1)
	if err != nil {
		t.Fatalf("get skills after learn failed: %v", err)
	}
	after := 0
	for _, s := range skills2 {
		if id, ok := s["id"].(string); ok && id == "basic_fist" {
			if lvl, ok := s["level"].(int); ok {
				after = lvl
			}
		}
	}
	if after != before+1 {
		t.Fatalf("expected basic_fist level +1, before=%d after=%d", before, after)
	}
	if err := repo.EquipItem(pid1, "iron_pickaxe"); err != nil {
		t.Fatalf("equip pickaxe failed: %v", err)
	}
	if _, err := repo.EnterWorld(pid1, "luoyang"); err != nil {
		t.Fatalf("reset to luoyang spawn failed: %v", err)
	}
	moveToMineFromLuoyangSpawn(t, repo, pid1)
	attrsBeforeMine, err := repo.GetAttributes(pid1)
	if err != nil {
		t.Fatalf("get attrs before mine failed: %v", err)
	}
	exp, pot, gemName, err := repo.Mine(pid1)
	if err != nil {
		t.Fatalf("mine failed: %v", err)
	}
	if exp != attrsBeforeMine["experience"].(int) || pot != attrsBeforeMine["potential"].(int) {
		t.Fatalf("mine start should not immediately settle: exp=%d pot=%d", exp, pot)
	}
	if gemName != "" {
		t.Fatalf("mine start should not immediately produce gem, got %s", gemName)
	}
	_ = repo.Stop(pid1)
	moveBackToLuoyangSpawnFromMine(t, repo, pid1)

	if _, err := repo.Spar(pid1, pid2); err != nil {
		t.Fatalf("spar failed: %v", err)
	}
	status1, err := repo.GetPlayerStatus(pid1)
	if err != nil {
		t.Fatalf("get status1 failed: %v", err)
	}
	if state, _ := status1["state"].(string); state != "COMBAT" {
		t.Fatalf("expected player1 COMBAT after spar, got %v", status1["state"])
	}
	stopCombatMsg := repo.Stop(pid1)
	if stopCombatMsg != "你正在战斗，待会再说" {
		t.Fatalf("unexpected stop message in combat: %s", stopCombatMsg)
	}
	status1c, err := repo.GetPlayerStatus(pid1)
	if err != nil {
		t.Fatalf("get status1 after stop-combat failed: %v", err)
	}
	if state, _ := status1c["state"].(string); state != "COMBAT" {
		t.Fatalf("expected player1 remain COMBAT after stop in combat, got %v", status1c["state"])
	}

	if _, err := repo.Kill(pid3, pid4); err != nil {
		t.Fatalf("kill failed: %v", err)
	}
	deadID := ""
	for i := 0; i < 800; i++ {
		if _, err := repo.ProcessCombatTick(); err != nil {
			t.Fatalf("process combat tick failed: %v", err)
		}
		status4, err := repo.GetPlayerStatus(pid4)
		if err != nil {
			t.Fatalf("get pid4 status failed: %v", err)
		}
		if state, _ := status4["state"].(string); state == "DEAD" {
			deadID = pid4
			break
		}
		status3, err := repo.GetPlayerStatus(pid3)
		if err != nil {
			t.Fatalf("get pid3 status failed: %v", err)
		}
		if state, _ := status3["state"].(string); state == "DEAD" {
			deadID = pid3
			break
		}
	}
	if deadID == "" {
		t.Fatalf("expected one combatant DEAD after kill combat ticks")
	}

	reliveMsg, err := repo.Relive(deadID)
	if err != nil {
		t.Fatalf("relive failed: %v", err)
	}
	if reliveMsg == "" {
		t.Fatalf("relive message should not be empty")
	}
	status2b, err := repo.GetPlayerStatus(deadID)
	if err != nil {
		t.Fatalf("get target status after relive failed: %v", err)
	}
	if state, _ := status2b["state"].(string); state != "IDLE" {
		t.Fatalf("expected target IDLE after relive, got %v", status2b["state"])
	}
	if roomID4, _, err := repo.GetPlayerRoom(deadID); err != nil || roomID4 != "ly_00" {
		t.Fatalf("expected relive respawn room ly_00, got room=%s err=%v", roomID4, err)
	}
	healMsg, err := repo.Heal(deadID)
	if err == nil || err.Error() != "hp_full" {
		t.Fatalf("heal should return hp_full after relive, got msg=%q err=%v", healMsg, err)
	}
	_ = repo.Stop(deadID)
	meditateMsg, err := repo.Meditate(deadID)
	if err != nil {
		t.Fatalf("meditate failed: %v", err)
	}
	if meditateMsg == "" {
		t.Fatalf("meditate message should not be empty")
	}
	stopMsg := repo.Stop(deadID)
	if stopMsg == "" {
		t.Fatalf("stop message should not be empty")
	}
	status1b, err := repo.GetPlayerStatus(deadID)
	if err != nil {
		t.Fatalf("get status1 after stop failed: %v", err)
	}
	if state, _ := status1b["state"].(string); state != "IDLE" {
		t.Fatalf("expected player IDLE after stop, got %v", status1b["state"])
	}
}

func TestFileRepositoryStatusTickFlow(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	if err := repo.Register("u9", "p9"); err != nil {
		t.Fatalf("register u9 failed: %v", err)
	}
	if _, err := repo.Login("u9", "p9"); err != nil {
		t.Fatalf("login u9 failed: %v", err)
	}
	p1, err := repo.CreateCharacter("u9", "hero9", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create p1 failed: %v", err)
	}

	if err := repo.Register("u10", "p10"); err != nil {
		t.Fatalf("register u10 failed: %v", err)
	}
	if _, err := repo.Login("u10", "p10"); err != nil {
		t.Fatalf("login u10 failed: %v", err)
	}
	p2, err := repo.CreateCharacter("u10", "hero10", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create p2 failed: %v", err)
	}

	if err := repo.EquipItem(p1, "iron_pickaxe"); err != nil {
		t.Fatalf("equip pickaxe failed: %v", err)
	}
	moveToMineFromLuoyangSpawn(t, repo, p1)
	if _, _, _, err := repo.Mine(p1); err != nil {
		t.Fatalf("start mine failed: %v", err)
	}
	attrsBeforeMine, err := repo.GetAttributes(p1)
	if err != nil {
		t.Fatalf("get attrs before mine tick failed: %v", err)
	}
	if _, err := repo.ProcessStatusTick(); err != nil {
		t.Fatalf("process status tick for mining failed: %v", err)
	}
	attrsAfterMine, err := repo.GetAttributes(p1)
	if err != nil {
		t.Fatalf("get attrs after mine tick failed: %v", err)
	}
	if attrsAfterMine["experience"].(int) <= attrsBeforeMine["experience"].(int) {
		t.Fatalf("expected experience increase on mining tick")
	}
	if msg := repo.Stop(p1); msg != "你收起铁镐，停止了挖矿。" {
		t.Fatalf("unexpected stop mining message: %s", msg)
	}
	moveBackToLuoyangSpawnFromMine(t, repo, p1)
	if _, err := repo.Meditate(p1); err != nil {
		t.Fatalf("start meditate failed: %v", err)
	}
	attrsBeforeMeditate, err := repo.GetAttributes(p1)
	if err != nil {
		t.Fatalf("get attrs before meditate tick failed: %v", err)
	}
	if _, err := repo.ProcessStatusTick(); err != nil {
		t.Fatalf("process status tick for meditate failed: %v", err)
	}
	attrsAfterMeditate, err := repo.GetAttributes(p1)
	if err != nil {
		t.Fatalf("get attrs after meditate tick failed: %v", err)
	}
	if attrsAfterMeditate["trainedMaxMp"].(int) < attrsBeforeMeditate["trainedMaxMp"].(int) {
		t.Fatalf("trainedMaxMp should not decrease after meditate tick")
	}
	if msg := repo.Stop(p1); msg != "你停止了打坐。" {
		t.Fatalf("unexpected stop meditate message: %s", msg)
	}

	hpDropped := false
	for attempt := 0; attempt < 3 && !hpDropped; attempt++ {
		if _, err := repo.Spar(p1, p2); err != nil {
			t.Fatalf("spar failed: %v", err)
		}
		for i := 0; i < 80; i++ {
			if _, err := repo.ProcessCombatTick(); err != nil {
				t.Fatalf("process combat tick failed: %v", err)
			}
			st1, err := repo.GetPlayerStatus(p1)
			if err != nil {
				t.Fatalf("get player1 status failed: %v", err)
			}
			attrsNow, err := repo.GetAttributes(p1)
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
	if _, err := repo.Heal(p1); err != nil {
		t.Fatalf("start heal failed: %v", err)
	}
	attrsBeforeHealTick, err := repo.GetAttributes(p1)
	if err != nil {
		t.Fatalf("get attrs before heal tick failed: %v", err)
	}
	if _, err := repo.ProcessStatusTick(); err != nil {
		t.Fatalf("process status tick for heal failed: %v", err)
	}
	attrsAfterHealTick, err := repo.GetAttributes(p1)
	if err != nil {
		t.Fatalf("get attrs after heal tick failed: %v", err)
	}
	if attrsAfterHealTick["currentHp"].(int) < attrsBeforeHealTick["currentHp"].(int) {
		t.Fatalf("currentHp should not decrease on heal tick")
	}
}

func TestFileRepositorySparAutoStopByTick(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	if err := repo.Register("u11", "p11"); err != nil {
		t.Fatalf("register u11 failed: %v", err)
	}
	if _, err := repo.Login("u11", "p11"); err != nil {
		t.Fatalf("login u11 failed: %v", err)
	}
	p1, err := repo.CreateCharacter("u11", "hero11", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create p1 failed: %v", err)
	}

	if err := repo.Register("u12", "p12"); err != nil {
		t.Fatalf("register u12 failed: %v", err)
	}
	if _, err := repo.Login("u12", "p12"); err != nil {
		t.Fatalf("login u12 failed: %v", err)
	}
	p2, err := repo.CreateCharacter("u12", "hero12", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create p2 failed: %v", err)
	}

	if _, err := repo.Spar(p1, p2); err != nil {
		t.Fatalf("spar failed: %v", err)
	}

	idle1, idle2 := false, false
	for i := 0; i < 40; i++ {
		if _, err := repo.ProcessCombatTick(); err != nil {
			t.Fatalf("process combat tick failed: %v", err)
		}
		st1, err := repo.GetPlayerStatus(p1)
		if err != nil {
			t.Fatalf("get player1 status failed: %v", err)
		}
		st2, err := repo.GetPlayerStatus(p2)
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
		t.Fatalf("expected both players return to IDLE after spar ticks, got idle1=%v idle2=%v", idle1, idle2)
	}
}

func TestFileRepositoryCombatTargetLoss(t *testing.T) {
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	if err := repo.Register("u13", "p13"); err != nil {
		t.Fatalf("register u13 failed: %v", err)
	}
	if _, err := repo.Login("u13", "p13"); err != nil {
		t.Fatalf("login u13 failed: %v", err)
	}
	p1, err := repo.CreateCharacter("u13", "hero13", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create p1 failed: %v", err)
	}

	if err := repo.Register("u14", "p14"); err != nil {
		t.Fatalf("register u14 failed: %v", err)
	}
	if _, err := repo.Login("u14", "p14"); err != nil {
		t.Fatalf("login u14 failed: %v", err)
	}
	p2, err := repo.CreateCharacter("u14", "hero14", 20, 20, 20, 20)
	if err != nil {
		t.Fatalf("create p2 failed: %v", err)
	}

	if _, err := repo.Spar(p1, p2); err != nil {
		t.Fatalf("spar failed: %v", err)
	}
	if err := repo.DeleteCharacter("u14", p2); err != nil {
		t.Fatalf("delete opponent failed: %v", err)
	}

	tickRes, err := repo.ProcessCombatTick()
	if err != nil {
		t.Fatalf("process combat tick failed: %v", err)
	}
	if len(tickRes.Messages) == 0 {
		t.Fatalf("expected target-loss combat message")
	}

	status, err := repo.GetPlayerStatus(p1)
	if err != nil {
		t.Fatalf("get player status failed: %v", err)
	}
	if status["state"] != "IDLE" {
		t.Fatalf("expected attacker leave combat after target loss, got %v", status["state"])
	}
}
