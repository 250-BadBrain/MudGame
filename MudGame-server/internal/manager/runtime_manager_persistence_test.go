package manager

import (
	"reflect"
	"testing"

	"mudgame/server/internal/model"
)

func newFullyPopulatedPlayer(id string) *model.Player {
	player := &model.Player{
		MartialVersion: model.MartialVersion, ActiveArts: map[string]string{"combat": "tiandao_blade"}, RecoveryItemReadyAt: 2000000000000,
		ID:                    id,
		UserID:                id + "-user",
		CharacterName:         "clone_hero",
		RoomID:                "ly_00",
		State:                 "LEARNING",
		StateTickCountdown:    7,
		LearningSkillID:       "basic_sword",
		LearningTeacherID:     "weapon_school_master",
		LearningTeacherLevel:  9,
		SkillLearningProgress: map[string]int{"basic_sword": 3},
		TargetID:              "target-1",
		CombatType:            "SPAR",
		Money:                 1234,
		Attributes:            model.NormalizeAttributes(model.Attributes{}),
		Skills:                map[string]int{"basic_unarmed": 2, "basic_parry": 1},
		EquippedSkills:        map[string]string{"basic_blade": "tiandao_blade"},
		EnhancedItems:         map[string]int{"iron_sword": 2},
		Backpack:              map[string]int{"dry_ration": 2, "iron_sword": 1},
		Equipment:             map[string]string{"WEAPON": "iron_sword"},
		QuestStates:           map[string]model.QuestState{"escort_rumor_run": {ID: "escort_rumor_run", Status: "active", Progress: 1}},
		Flags:                 map[string]bool{"heard_storyteller_rumor": true},
		DialogueHistory:       map[string]bool{"storyteller|opening": true},
		Reputations:           map[string]int{"luoyang_escort": 5},
		Online:                true,
		WeakUntil:             2000000000,
	}
	player.Attributes.Level = 3
	player.Attributes.Realm = "trainee"
	player.Attributes.Experience = 900
	player.Attributes.Potential = 470
	player.Attributes.Base = model.BaseAttributes{Strength: 16, Constitution: 16, Agility: 16, Perception: 16, Willpower: 16, Charm: 5, Luck: 6}
	player.Attributes.Growth = model.GrowthAttributes{PostStrength: 1, PostConstitution: 2, PostAgility: 3, PostPerception: 4, PostWillpower: 5}
	player.Attributes.Resources = model.ResourceAttributes{HP: 60, MaxHP: 80, MP: 40, TrainedMaxMP: 88, MaxMP: 200, Spirit: 30, MaxSpirit: 100}
	player.Attributes.Derived = model.DerivedAttributes{Attack: 42, Defense: 37, HitRate: 82, Dodge: 62, Parry: 58, CritRate: 3, AttackSpeed: 4, Initiative: 6, GuardBreak: 1, DamageReduction: 2}
	player.Attributes.Social = model.SocialAttributes{Fame: 10, Notoriety: 1, Authority: 2, Credit: 3}
	player.Attributes.Resistances = model.ResistanceAttributes{Poison: 4, Bleed: 5, Stun: 6, Fear: 7, Charm: 8, Seal: 9}
	return player
}

func TestClonePlayersMapPreservesAllFields(t *testing.T) {
	src := map[string]*model.Player{"p1": newFullyPopulatedPlayer("p1")}
	cloned := clonePlayersMap(src)
	if len(cloned) != 1 {
		t.Fatalf("expected one cloned player, got %d", len(cloned))
	}
	orig, ok := src["p1"]
	if !ok || orig == nil {
		t.Fatalf("missing source player")
	}
	clone, ok := cloned["p1"]
	if !ok || clone == nil {
		t.Fatalf("missing cloned player")
	}

	origValue := reflect.ValueOf(*orig)
	cloneValue := reflect.ValueOf(*clone)
	structType := origValue.Type()
	// 手工克隆清单与 model.Player 字段必须同步维护：
	// 给 model.Player 新增字段时，clonePlayersMap 与本测试都要同步更新，否则存档会静默丢字段。
	if structType.NumField() != 28 {
		t.Fatalf("model.Player field count changed to %d: update clonePlayersMap, newFullyPopulatedPlayer and this guard", structType.NumField())
	}
	for i := 0; i < structType.NumField(); i++ {
		name := structType.Field(i).Name
		if reflect.DeepEqual(origValue.Field(i).Interface(), cloneValue.Field(i).Interface()) {
			continue
		}
		t.Fatalf("clonePlayersMap did not preserve field %s", name)
	}

	// 克隆必须是深拷贝：修改克隆方的 map 不得影响源数据
	clone.ActiveArts["combat"] = "other"
	if orig.ActiveArts["combat"] != "tiandao_blade" {
		t.Fatal("active arts are not deep copied")
	}
	clone.Backpack["dry_ration"] = 99
	clone.Flags["heard_storyteller_rumor"] = false
	if orig.Backpack["dry_ration"] != 2 || !orig.Flags["heard_storyteller_rumor"] {
		t.Fatalf("clone shares map state with source: %+v %+v", orig.Backpack, orig.Flags)
	}
}

func TestCloneUsersMapPreservesAllFields(t *testing.T) {
	src := map[string]*model.User{
		"u1": {Username: "u1", PasswordHash: "pbkdf2$1$aa$bb", CharacterIDs: []string{"p1", "p2"}, ReconnectTokenHash: "tokenv1$abc"},
	}
	cloned := cloneUsersMap(src)
	orig := src["u1"]
	clone := cloned["u1"]
	if clone == nil {
		t.Fatalf("missing cloned user")
	}
	if clone.Username != orig.Username || clone.PasswordHash != orig.PasswordHash || clone.ReconnectTokenHash != orig.ReconnectTokenHash {
		t.Fatalf("cloneUsersMap did not preserve fields: %+v", clone)
	}
	if !reflect.DeepEqual(clone.CharacterIDs, orig.CharacterIDs) {
		t.Fatalf("cloneUsersMap did not preserve CharacterIDs: %+v", clone.CharacterIDs)
	}
	clone.CharacterIDs[0] = "mutated"
	if orig.CharacterIDs[0] != "p1" {
		t.Fatalf("clone shares CharacterIDs slice with source")
	}
}

func TestShopInventoryIsIsolatedFromStaticTemplate(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("shop_user", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("shop_user", "shop_hero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	r.mu.Lock()
	players, loadErr := r.loadPlayers()
	if loadErr != nil {
		r.mu.Unlock()
		t.Fatalf("load players failed: %v", loadErr)
	}
	players[playerID].RoomID = "ly_west_market"
	players[playerID].Money = 100000
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save players failed: %v", err)
	}
	r.mu.Unlock()

	templateStock := fileNpcs["black_market_broker"].ShopInventory["smoke_powder"]
	if templateStock != 5 {
		t.Fatalf("expected static template stock 5 for smoke_powder, got %d", templateStock)
	}

	if _, _, err := r.Buy(playerID, "black_market_broker", "smoke_powder", 2); err != nil {
		t.Fatalf("buy failed: %v", err)
	}

	if after := fileNpcs["black_market_broker"].ShopInventory["smoke_powder"]; after != 5 {
		t.Fatalf("static template stock was polluted by buy: got %d, want 5", after)
	}
	r.mu.RLock()
	state, ok := r.staticNPCs["black_market_broker"]
	r.mu.RUnlock()
	if !ok || state == nil {
		t.Fatalf("expected runtime shop state for black_market_broker")
	}
	if state.ShopInventory["smoke_powder"] != 3 {
		t.Fatalf("expected runtime stock 3 after buying 2, got %d", state.ShopInventory["smoke_powder"])
	}

	items, _, _, _, err := r.GetShop(playerID, "black_market_broker")
	if err != nil {
		t.Fatalf("get shop failed: %v", err)
	}
	found := false
	for _, item := range items {
		if item["id"] == "smoke_powder" {
			found = true
			if item["limit"] != 3 {
				t.Fatalf("expected displayed limit 3 after purchase, got %+v", item)
			}
		}
	}
	if !found {
		t.Fatalf("smoke_powder missing from shop payload: %+v", items)
	}
}

func TestUseItemConsumesAndRestoresHP(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("item_user", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("item_user", "item_hero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	if _, err := r.UseItem(playerID, "dry_ration"); err == nil {
		t.Fatalf("expected error when item missing from backpack")
	}

	r.mu.Lock()
	players, loadErr := r.loadPlayers()
	if loadErr != nil {
		r.mu.Unlock()
		t.Fatalf("load players failed: %v", loadErr)
	}
	players[playerID].Backpack["dry_ration"] = 2
	players[playerID].Attributes.Resources.HP = 30
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save players failed: %v", err)
	}
	r.mu.Unlock()

	message, err := r.UseItem(playerID, "dry_ration")
	if err != nil {
		t.Fatalf("use item failed: %v", err)
	}
	if message == "" {
		t.Fatalf("expected use item message")
	}
	r.mu.RLock()
	backpackCount := r.playersCache[playerID].Backpack["dry_ration"]
	currentHP := r.playersCache[playerID].Attributes.Resources.HP
	r.mu.RUnlock()
	if backpackCount != 1 {
		t.Fatalf("expected one dry_ration consumed, got %d left", backpackCount)
	}
	if currentHP != 35 {
		t.Fatalf("expected hp 35 after using dry_ration(recoverHp 5), got %d", currentHP)
	}

	// 满血时应拒绝并保留物品
	r.mu.Lock()
	players, loadErr = r.loadPlayers()
	if loadErr != nil {
		r.mu.Unlock()
		t.Fatalf("reload players failed: %v", loadErr)
	}
	hpAfterRecovery := players[playerID]
	hpAfterRecovery.Attributes.Resources.HP = effectiveAttributesForPlayerNoLock(hpAfterRecovery).MaxHP
	if err := r.savePlayers(players); err != nil {
		r.mu.Unlock()
		t.Fatalf("save players failed: %v", err)
	}
	r.mu.Unlock()
	if _, err := r.UseItem(playerID, "dry_ration"); err == nil {
		t.Fatalf("expected hp_full error when using item at full hp")
	}
	r.mu.RLock()
	backpackCount = r.playersCache[playerID].Backpack["dry_ration"]
	r.mu.RUnlock()
	if backpackCount != 1 {
		t.Fatalf("item should not be consumed at full hp, got %d left", backpackCount)
	}
}

func TestCompleteQuestValidatesTurnInNPC(t *testing.T) {
	r := NewRuntimeManager(t.TempDir())
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Register("quest_user", "pass123"); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	playerID, err := r.CreateCharacter("quest_user", "quest_hero", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatalf("create character failed: %v", err)
	}

	r.mu.Lock()
	players, loadErr := r.loadPlayers()
	if loadErr != nil {
		r.mu.Unlock()
		t.Fatalf("load players failed: %v", loadErr)
	}
	p := players[playerID]
	p.QuestStates["escort_rumor_run"] = model.QuestState{ID: "escort_rumor_run", Status: "active"}
	p.Flags["heard_storyteller_rumor"] = true

	// 交给错误的 NPC 应被拒绝，任务保持进行中
	if _, err := r.completeQuestNoLock(p, "escort_rumor_run", "storyteller"); err == nil {
		r.mu.Unlock()
		t.Fatalf("expected quest_wrong_npc error when turning in at wrong npc")
	} else if err.Error() != "quest_wrong_npc" {
		r.mu.Unlock()
		t.Fatalf("expected quest_wrong_npc error, got %v", err)
	}
	if p.QuestStates["escort_rumor_run"].Status != "active" {
		r.mu.Unlock()
		t.Fatalf("quest status should remain active after rejected turn-in")
	}

	logs, err := r.completeQuestNoLock(p, "escort_rumor_run", "escort_bureau_chief")
	if err != nil {
		r.mu.Unlock()
		t.Fatalf("turn in at correct npc failed: %v", err)
	}
	if len(logs) == 0 {
		r.mu.Unlock()
		t.Fatalf("expected completion logs")
	}
	if p.QuestStates["escort_rumor_run"].Status != "completed" {
		r.mu.Unlock()
		t.Fatalf("expected completed quest state")
	}
	r.mu.Unlock()
}
