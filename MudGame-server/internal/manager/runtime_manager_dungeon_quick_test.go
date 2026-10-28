package manager

import (
	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/model"
	"reflect"
	"testing"
)

// Flow tests verify real delivery, even when no random equipment drops. An
// omitted inventory update or a fabricated settlement both fail this check.
func leaveDungeonWithRewardCheck(t *testing.T, r *RuntimeManager, id string) (map[string]interface{}, error) {
	t.Helper()
	before, _ := r.loadPlayers()
	room, err := r.LeaveDungeon(id)
	if err != nil {
		return room, err
	}
	after, _ := r.loadPlayers()
	settlement, ok := room["dungeonSettlement"].(map[string]interface{})
	if !ok {
		t.Fatal("missing settlement")
	}
	received, ok := settlement["rewardItems"].(map[string]int)
	if !ok {
		t.Fatal("missing actual item results")
	}
	for item, amount := range received {
		if after[id].Backpack[item]-before[id].Backpack[item] != amount {
			t.Fatalf("settlement not delivered: %s", item)
		}
	}
	for item, amount := range after[id].Backpack {
		if gain := amount - before[id].Backpack[item]; gain > 0 && received[item] != gain {
			t.Fatalf("unreported reward: %s x%d", item, gain)
		}
	}
	if after[id].Money-before[id].Money != settlement["rewardMoney"] || after[id].Attributes.Experience-before[id].Attributes.Experience != settlement["rewardExperience"] || after[id].Attributes.Potential-before[id].Attributes.Potential != settlement["rewardPotential"] {
		t.Fatal("settlement currency was not delivered")
	}
	return room, nil
}

func TestQuickCompletePersistsSequenceAndClaimRepair(t *testing.T) {
	dir := t.TempDir()
	r := NewRuntimeManager(dir)
	if err := r.Register("quick_reload", "pass123"); err != nil {
		t.Fatal(err)
	}
	id, err := r.CreateCharacter("quick_reload", "快通侠", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	players, _ := r.loadPlayers()
	p := players[id]
	p.Flags["dungeon_cleared_abandoned_temple_rain"] = true
	_ = r.savePlayers(players)
	if _, err := r.QuickCompleteDungeon(id, "abandoned_temple_rain", 0); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	r = NewRuntimeManager(dir)
	defer r.Close()
	players, _ = r.loadPlayers()
	before := players[id]
	if before.QuickCompleteSequence != 1 {
		t.Fatal("sequence lost on reload")
	}
	if _, err := r.QuickCompleteDungeon(id, "abandoned_temple_rain", 0); err == nil {
		t.Fatal("old request replayed after reload")
	}
	players, _ = r.loadPlayers()
	if !reflect.DeepEqual(before, players[id]) {
		t.Fatal("rejected replay mutated player")
	}
	if _, err := r.QuickCompleteDungeon(id, "abandoned_temple_rain", 1); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	if players[id].Backpack["sealed_letter"] != 1 {
		t.Fatal("one-time quest letter duplicated")
	}
}

func TestDungeonDropClaimOnlyAfterSuccessfulRoll(t *testing.T) {
	p := &model.Player{Backpack: map[string]int{}, Flags: map[string]bool{}, Skills: map[string]int{}}
	s := engruntime.Settlement{DungeonID: "datang_liangdao", Progress: 100, RewardItems: map[string]int{"liangdao_spear": 1, "bronze_fragment": 3}}
	applyDungeonSettlementRewardWithRoll(p, s, func() float64 { return .15 })
	if p.Backpack["liangdao_spear"] != 0 || p.Flags["item_claimed_liangdao_spear"] {
		t.Fatal("failed drop consumed claim")
	}
	result := applyDungeonSettlementRewardWithRoll(p, s, func() float64 { return .149999 })
	if p.Backpack["liangdao_spear"] != 1 || !p.Flags["item_claimed_liangdao_spear"] {
		t.Fatal("successful drop missing")
	}
	if p.Backpack["bronze_fragment"] != 5 || result.RewardItems["bronze_fragment"] != 5 {
		t.Fatal("bronze reward not additive")
	}
	applyDungeonSettlementRewardWithRoll(p, s, func() float64 { return 0 })
	if p.Backpack["liangdao_spear"] != 1 {
		t.Fatal("claimed weapon duplicated")
	}
}

func TestDungeonRepeatGuaranteeWithoutItemTable(t *testing.T) {
	p := &model.Player{Backpack: map[string]int{}, Flags: map[string]bool{"dungeon_cleared_datang_changan": true}}
	s := applyDungeonSettlementRewardWithRoll(p, engruntime.Settlement{DungeonID: "datang_changan", Progress: 100}, func() float64 { return 1 })
	if p.Backpack["bronze_fragment"] != 2 || s.RewardItems["bronze_fragment"] != 2 {
		t.Fatal("empty item table lost repeat guarantee")
	}
}

func TestDungeonDropLegacyRepairIsIdempotent(t *testing.T) {
	p := &model.Player{Backpack: map[string]int{"youlong_whip": 1}, Equipment: map[string]string{"WEAPON": "lieshi_mace"}, Flags: map[string]bool{"item_claimed_liangdao_spear": true, "item_claimed_youlong_whip": true, "item_claimed_lieshi_mace": true}}
	repairDungeonDropClaims(p)
	if p.Flags["item_claimed_liangdao_spear"] || !p.Flags["item_claimed_youlong_whip"] || !p.Flags["item_claimed_lieshi_mace"] {
		t.Fatal("repair lost ownership or missed failure")
	}
	p.Flags["item_claimed_liangdao_spear"] = true
	repairDungeonDropClaims(p)
	if !p.Flags["item_claimed_liangdao_spear"] {
		t.Fatal("repair ran twice")
	}
}

func TestQuickCompleteEligibilityAndExactlyOnce(t *testing.T) {
	r, id := v2Player(t)
	if _, err := r.QuickCompleteDungeon(id, "datang_changshengjue", 0); err == nil {
		t.Fatal("uncleared quick completion accepted")
	}
	players, _ := r.loadPlayers()
	p := players[id]
	p.Flags["dungeon_cleared_datang_changshengjue"] = true
	p.Attributes.Resources.HP = 30
	p.Attributes.Resources.MP = 20
	_ = r.savePlayers(players)
	beforeMoney, beforeExp, room := p.Money, p.Attributes.Experience, p.RoomID
	result, err := r.QuickCompleteDungeon(id, "datang_changshengjue", 0)
	if err != nil {
		t.Fatal(err)
	}
	if result["progress"] != 100 {
		t.Fatalf("result=%v", result)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Backpack["herb"] != 4 || p.Backpack["dry_ration"] < 2 || p.Backpack["bronze_fragment"] != 2 {
		t.Fatalf("missing fixed process loot: %v", p.Backpack)
	}
	if p.Money != beforeMoney+80 || p.Attributes.Experience != beforeExp+260 || p.RoomID != room || p.Attributes.Resources.HP != 30 || p.Attributes.Resources.MP != 20 || p.QuickCompleteSequence != 1 {
		t.Fatal("incorrect quick reward or side effect")
	}
	if _, err := r.QuickCompleteDungeon(id, "datang_changshengjue", 0); err == nil {
		t.Fatal("replay accepted")
	}
	players, _ = r.loadPlayers()
	if players[id].Money != p.Money || players[id].QuickCompleteSequence != 1 {
		t.Fatal("replay mutated state")
	}
	for _, state := range []string{"COMBAT", "DEAD", "LEARNING", "HEALING", "MEDITATING"} {
		players, _ = r.loadPlayers()
		players[id].State = state
		_ = r.savePlayers(players)
		if _, err := r.QuickCompleteDungeon(id, "datang_changshengjue", 1); err == nil {
			t.Fatalf("busy state accepted: %s", state)
		}
	}
	players, _ = r.loadPlayers()
	players[id].State = "IDLE"
	players[id].RoomID = "dg_fake_room"
	_ = r.savePlayers(players)
	if _, err := r.QuickCompleteDungeon(id, "datang_changshengjue", 1); err == nil {
		t.Fatal("instance quick accepted")
	}
}

func TestQuickCompleteAllCurrentDungeonsHaveFullRewards(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	for _, tpl := range r.dungeonTemplatesNoLock() {
		p.Flags["dungeon_cleared_"+tpl.ID] = true
	}
	_ = r.savePlayers(players)
	for _, tpl := range r.dungeonTemplatesNoLock() {
		players, _ = r.loadPlayers()
		before := players[id]
		result, err := r.QuickCompleteDungeon(id, tpl.ID, before.QuickCompleteSequence)
		if err != nil {
			t.Fatalf("%s: %v", tpl.ID, err)
		}
		players, _ = r.loadPlayers()
		after := players[id]
		if after.Money-before.Money != tpl.Rewards.Money+tpl.QuickRewards.Money || result["progress"] != 100 {
			t.Fatalf("%s lost full/process money", tpl.ID)
		}
		received := result["rewardItems"].(map[string]int)
		for item, amount := range received {
			if after.Backpack[item]-before.Backpack[item] != amount {
				t.Fatalf("%s/%s reward mismatch", tpl.ID, item)
			}
		}
		for item, amount := range tpl.QuickRewards.Items {
			if received[item] < amount {
				t.Fatalf("%s lost process item %s", tpl.ID, item)
			}
		}
		if received["bronze_fragment"] < 2 {
			t.Fatalf("%s lost bronze guarantee", tpl.ID)
		}
	}
}

func TestRestoredTransmissionAndProtectedNPC(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Skills["basic_neigong"] = 120
	p.Skills["basic_literacy"] = 80
	p.Backpack["manual_csj_changshengjue"] = 1
	p.RoomID = "ly_old_alley"
	_ = r.savePlayers(players)
	if err := r.LearnSkill(id, "ancient_scroll_keeper", "csj_changshengjue"); err == nil {
		t.Fatal("keeper bypassed transmission")
	}
	if _, err := r.UseItem(id, "manual_csj_changshengjue"); err == nil {
		t.Fatal("manual bypassed source dungeon")
	}
	players, _ = r.loadPlayers()
	p = players[id]
	p.Flags["dungeon_cleared_datang_xinggong"] = true
	p.Skills["csj_changshengjue"] = 100
	p.ActiveArts["internal"] = "csj_changshengjue"
	if reason := skillActiveReason(p, fileSkills["csj_changshengjue"]); reason != "" {
		t.Fatal("legitimate V2 art disabled:", reason)
	}
	r2, id2, inst := createDungeonTestPlayer(t, "datang_jingling")
	for _, npc := range inst.NPCs {
		if npc.TemplateID == "jl_kou_zhong" {
			moveDungeonTestPlayer(t, r2, id2, npc.RoomID)
			if _, err := r2.Kill(id2, npc.ID); err == nil {
				t.Fatal("protected protagonist kill accepted")
			}
			return
		}
	}
	t.Fatal("NPC missing")
}

func TestGongbianRequiresLedgers(t *testing.T) {
	r, id, inst := createDungeonTestPlayer(t, "datang_gongbian")
	room := func(k string) string { return r.dungeonMgr.InstanceRoomIDNoLock(inst, k) }
	if _, err := r.Move(id, "north"); err == nil {
		t.Fatal("contact gate absent")
	}
	if _, err := r.DoRoomAction(id, "find_contact"); err != nil {
		t.Fatal(err)
	}
	for _, npc := range inst.NPCs {
		if npc.TemplateID == "gb_pegasus_sergeant" || npc.TemplateID == "gb_dugu_ling" {
			r.dungeonMgr.DefeatNPCNoLock(npc.ID)
		}
	}
	moveDungeonTestPlayer(t, r, id, room("gb_03"))
	if _, err := r.Move(id, "east"); err == nil {
		t.Fatal("rockery gate absent")
	}
	if _, err := r.DoRoomAction(id, "examine_rockery"); err != nil {
		t.Fatal(err)
	}
	if inst.Progress >= 100 {
		t.Fatal("completed without ledger")
	}
	moveDungeonTestPlayer(t, r, id, room("gb_05"))
	for _, action := range []string{"take_ledgers", "open_ledgers"} {
		if _, err := r.DoRoomAction(id, action); err != nil {
			t.Fatal(err)
		}
	}
	if inst.Progress != 100 {
		t.Fatalf("complete story progress=%d", inst.Progress)
	}
}
