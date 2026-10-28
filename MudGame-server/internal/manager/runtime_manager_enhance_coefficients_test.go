package manager

import (
	"math"
	"mudgame/server/internal/model"
	"reflect"
	"testing"
)

func TestEnhancementCoefficientsAndPreview(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	before := effectiveAttributesForPlayerNoLock(p)
	def := fileItems["tangmen_baojian"]
	p.Equipment[def.EquipmentSlot] = def.ID
	base := effectiveAttributesForPlayerNoLock(p)
	p.EnhancedItems = map[string]int{def.ID: 5}
	after := effectiveAttributesForPlayerNoLock(p)
	if after.Attack-base.Attack != 5 || after.Defense-base.Defense != 5 || after.HitRate-base.HitRate != 5 || math.Abs(after.CritRate-base.CritRate-.5) > 1e-9 {
		t.Fatalf("bad coefficients: base=%+v after=%+v", base, after)
	}
	if before.Attack >= base.Attack {
		t.Fatal("base equipment missing")
	}
	bonus := getEnhanceBonus(def.ID, 1)
	if math.Abs(float64(bonus["critRate"])-.1) > 1e-9 || bonus["attack"] != 1 {
		t.Fatal("incorrect preview", bonus)
	}
	for key, want := range map[string]float64{"attack": 1, "defense": 1, "hitRate": 1, "dodge": 1, "parry": 1, "initiative": 1, "charm": 1, "maxHp": 5, "maxMp": 5, "critRate": .1} {
		got, ok := model.EnhancementStep(key)
		if !ok || got != want {
			t.Fatal(key, got)
		}
	}
	if _, ok := model.EnhancementStep("unknown"); ok {
		t.Fatal("unknown effect accepted")
	}
}

func TestUnlimitedSmokeAcrossShoppers(t *testing.T) {
	r, id := v2Player(t)
	count := 0
	for _, npc := range fileNpcs {
		if stock, ok := npc.ShopInventory["smoke_powder"]; ok {
			count++
			if stock != -1 {
				t.Fatal("limited merchant", npc.ID, stock)
			}
		}
	}
	if count != 4 {
		t.Fatal("unexpected number of merchants", count)
	}
	other, err := r.CreateCharacter("learner", "买烟侠", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	for _, playerID := range []string{id, other} {
		players, _ := r.loadPlayers()
		p := players[playerID]
		p.RoomID = "ly_west_market"
		p.Money = 5000
		_ = r.savePlayers(players)
		if _, _, err := r.Buy(playerID, "black_market_broker", "smoke_powder", 6); err != nil {
			t.Fatal(err)
		}
		players, _ = r.loadPlayers()
		if players[playerID].Money != 3500 || players[playerID].Backpack["smoke_powder"] != 6 {
			t.Fatal("bad purchase")
		}
	}
	if r.staticNPCs["black_market_broker"].ShopInventory["smoke_powder"] != -1 {
		t.Fatal("unlimited stock depleted")
	}
}

func TestLegacyEnhancementResetIsOptionalAndOnce(t *testing.T) {
	dir := t.TempDir()
	r := NewRuntimeManager(dir)
	_ = r.Register("legacycoeff", "pass123")
	id, err := r.CreateCharacter("legacycoeff", "旧强化侠", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	players, _ := r.loadPlayers()
	p := players[id]
	p.Backpack["iron_sword"] = 1
	p.EnhancedItems = map[string]int{"iron_sword": 5}
	delete(p.Flags, enhancementRulesFlag)
	_ = r.savePlayers(players)
	_ = r.Close()
	r = NewRuntimeManager(dir)
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Money != 5000 || p.EnhancedItems["iron_sword"] != 5 || !enhancementInfo(p, "iron_sword")["canReset"].(bool) {
		t.Fatal("legacy investment lost or prematurely refunded")
	}
	if _, err = r.ResetEnhancement(id, "iron_sword"); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Money != 8000 || p.Backpack["bronze_fragment"] != 15 || p.EnhancedItems["iron_sword"] != 0 {
		t.Fatal("bad refund", p)
	}
	before, _ := r.loadPlayers()
	if _, err = r.ResetEnhancement(id, "iron_sword"); err == nil {
		t.Fatal("replay accepted")
	}
	after, _ := r.loadPlayers()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("replay changed inventory")
	}
	_ = r.Close()
	r = NewRuntimeManager(dir)
	defer r.Close()
	players, _ = r.loadPlayers()
	p = players[id]
	if enhancementInfo(p, "iron_sword")["canReset"].(bool) || p.Money != 8000 {
		t.Fatal("restart repeated eligibility/refund")
	}
	if _, err = r.EnhanceItem(id, "iron_sword"); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ResetEnhancement(id, "iron_sword"); err == nil {
		t.Fatal("re-enhancement renewed eligibility")
	}
}

func TestEnhancementResetRejectsInvalidStateWithoutMutation(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Backpack["iron_sword"] = 1
	p.EnhancedItems = map[string]int{"iron_sword": 2}
	if enhancementInfo(p, "iron_sword")["canReset"].(bool) {
		t.Fatal("new player got legacy reset")
	}
	p.Flags[enhancementResetFlag("iron_sword")] = true
	for _, scenario := range []string{"busy", "unowned", "overflow"} {
		p.State = "IDLE"
		p.Backpack["iron_sword"] = 1
		p.Money = 5000
		switch scenario {
		case "busy":
			p.State = "MEDITATING"
		case "unowned":
			delete(p.Backpack, "iron_sword")
		case "overflow":
			p.Money = int(^uint(0) >> 1)
		}
		_ = r.savePlayers(players)
		before, _ := r.loadPlayers()
		if _, err := r.ResetEnhancement(id, "iron_sword"); err == nil {
			t.Fatal("accepted", scenario)
		}
		after, _ := r.loadPlayers()
		if !reflect.DeepEqual(before, after) {
			t.Fatal("changed state", scenario)
		}
	}
}

func TestEnhancementResetClampsWithoutHealingOrErasingTraining(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Equipment["NECKLACE"] = "cangming_juan"
	p.EnhancedItems = map[string]int{"cangming_juan": 5}
	p.Flags[enhancementResetFlag("cangming_juan")] = true
	withBonus := effectiveAttributesForPlayerNoLock(p)
	p.Attributes.Resources.TrainedMaxMP = withBonus.MaxMP
	p.Attributes.Resources.MP = withBonus.MaxMP
	withBonus = effectiveAttributesForPlayerNoLock(p)
	p.Attributes.Resources.HP = withBonus.MaxHP
	permanent := p.Attributes.Resources.TrainedMaxMP
	_ = r.savePlayers(players)
	if _, err := r.ResetEnhancement(id, "cangming_juan"); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	after := effectiveAttributesForPlayerNoLock(p)
	// Removing 25 MP also removes 8 HP from the existing 30% internal conversion.
	if withBonus.MaxHP-after.MaxHP != 33 || withBonus.MaxMP-after.MaxMP != 25 || p.Attributes.Resources.HP != after.MaxHP || p.Attributes.Resources.MP != after.TrainedMaxMP || p.Attributes.Resources.TrainedMaxMP != permanent {
		t.Fatal("bad cap/training handling", withBonus, after, p.Attributes.Resources)
	}
}

func TestAliasRefundAndResetDoNotDoubleCount(t *testing.T) {
	p := &model.Player{Money: 5000, Backpack: map[string]int{"langfan_jian": 1, "langfan_j": 1}, EnhancedItems: map[string]int{"langfan_jian": 2, "langfan_j": 3}, Flags: map[string]bool{}}
	if _, err := normalizePlayerItemAliases(p); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareEnhancementReset(p); err != nil {
		t.Fatal(err)
	}
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p.ID = id
	p.State = "IDLE"
	p.Attributes = players[id].Attributes
	players[id] = p
	_ = r.savePlayers(players)
	if _, err := r.ResetEnhancement(id, "langfan_jian"); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Money != 6800 || p.Backpack["bronze_fragment"] != 9 || p.Backpack["langfan_j"] != 2 {
		t.Fatal("merged/refunded wrong investments", p)
	}
}
