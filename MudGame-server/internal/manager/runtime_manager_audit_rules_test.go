package manager

import (
	"encoding/json"
	"math/rand"
	"mudgame/server/internal/model"
	"reflect"
	"testing"
	"time"
)

func TestAuditCreationAndTransactions(t *testing.T) {
	r, id := v2Player(t)
	if _, err := r.CreateCharacter("learner", "越界侠", 1000, 1000, 1000, 1000, -3920); err == nil {
		t.Fatal("invalid attributes accepted")
	}
	if _, err := r.CreateCharacter("learner", "  江湖无名小卒  ", 16, 16, 16, 16, 16); err != nil {
		t.Fatal(err)
	}
	players, _ := r.loadPlayers()
	players[id].RoomID = "bs_market"
	_ = r.savePlayers(players)
	before, _ := r.loadPlayers()
	for _, n := range []int{-1, 0, 10001, 1 << 60} {
		if _, _, err := r.Buy(id, "bs_herbalist", "herb", n); err == nil {
			t.Fatalf("bad buy %d", n)
		}
		if _, _, err := r.SellItem(id, "herb", n); err == nil {
			t.Fatalf("bad sell %d", n)
		}
		if err := r.DiscardItem(id, "herb", n); err == nil {
			t.Fatalf("bad discard %d", n)
		}
	}
	after, _ := r.loadPlayers()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected buy changed state")
	}
	players, _ = r.loadPlayers()
	players[id].Backpack["herb"] = 2
	_ = r.savePlayers(players)
	gain, _, err := r.SellItem(id, "herb", 2)
	if err != nil || gain != 6 {
		t.Fatal("20% sale mismatch", gain, err)
	}
}

func TestAuditTradeOverflowIsAtomic(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.RoomID = "bs_market"
	p.Money = int(^uint(0) >> 1)
	p.Backpack["herb"] = 1
	_ = r.savePlayers(players)
	before, _ := r.loadPlayers()
	if _, _, err := r.SellItem(id, "herb", 1); err == nil {
		t.Fatal("sale overflow accepted")
	}
	after, _ := r.loadPlayers()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected sale changed state")
	}
	p.Backpack["herb"] = p.Money
	_ = r.savePlayers(players)
	before, _ = r.loadPlayers()
	if _, _, err := r.Buy(id, "bs_herbalist", "herb", 1); err == nil {
		t.Fatal("inventory overflow accepted")
	}
	after, _ = r.loadPlayers()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected purchase changed state")
	}
}
func TestAuditSparAndSafety(t *testing.T) {
	r, id := v2Player(t)
	npc, _ := r.staticNPCInRoomNoLock(fileRooms["ly_weapon_school"], "weapon_school_master")
	npc.Attributes.Resources.MaxHP = 1000000
	npc.Attributes.Resources.HP = 1000000
	npc.Attributes.Derived.Attack = 1000
	npc.Attributes.Derived.HitRate = 1000000
	r.combatRandom = rand.New(rand.NewSource(42))
	if _, err := r.Spar(id, npc.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if _, err := r.ProcessCombatTick(); err != nil {
			t.Fatal(err)
		}
		players, _ := r.loadPlayers()
		if players[id].State == "IDLE" {
			break
		}
	}
	players, _ := r.loadPlayers()
	p := players[id]
	if p.State != "IDLE" || p.Attributes.Resources.HP < 1 {
		t.Fatal("spar left invalid life state")
	}
	if npc.Attributes.Resources.HP != npc.Attributes.Resources.MaxHP {
		t.Fatal("NPC did not recover after win")
	}
	if _, err := r.Spar(id, npc.ID); err == nil {
		t.Fatal("injured player could spar")
	}
	other, err := r.CreateCharacter("learner", "离线侠", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	players[id].RoomID = "ly_00"
	players[id].Attributes.Resources.HP = effectiveAttributesForPlayerNoLock(players[id]).MaxHP
	_ = r.savePlayers(players)
	if _, err := r.Kill(id, other); err == nil {
		t.Fatal("safe-zone kill accepted")
	}
	if _, err := r.Spar(id, other); err == nil {
		t.Fatal("safe-zone player challenge accepted")
	}
	players, _ = r.loadPlayers()
	players[id].RoomID = "ly_outskirts"
	players[other].RoomID = "ly_outskirts"
	players[other].Online = false
	_ = r.savePlayers(players)
	if _, err := r.Kill(id, other); err != nil {
		t.Fatal("offline victim outside safe zone should be attackable", err)
	}
}

func TestAuditSparRequiresFullHealthTarget(t *testing.T) {
	r, id := v2Player(t)
	npc, _ := r.staticNPCInRoomNoLock(fileRooms["ly_weapon_school"], "weapon_school_master")
	npc.Attributes.Resources.HP--
	if _, err := r.Spar(id, npc.ID); err == nil {
		t.Fatal("injured NPC accepted spar")
	}
	other, err := r.CreateCharacter("learner", "受伤对手", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	players, _ := r.loadPlayers()
	players[id].RoomID = "ly_outskirts"
	players[other].RoomID = "ly_outskirts"
	players[other].Attributes.Resources.HP--
	_ = r.savePlayers(players)
	if _, err := r.Spar(id, other); err == nil {
		t.Fatal("injured player accepted spar")
	}
	players[other].Attributes.Resources.HP = effectiveAttributesForPlayerNoLock(players[other]).MaxHP
	_ = r.savePlayers(players)
	if _, err := r.Spar(id, other); err != nil {
		t.Fatal("full-health players could not spar", err)
	}
}
func TestAuditDeadDungeonRestart(t *testing.T) {
	dir := t.TempDir()
	r := NewRuntimeManager(dir)
	_ = r.Register("restart", "pass123")
	id, err := r.CreateCharacter("restart", "重启侠", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.EnterDungeon(id, "abandoned_temple_rain"); err != nil {
		t.Fatal(err)
	}
	players, _ := r.loadPlayers()
	players[id].State = "DEAD"
	players[id].Attributes.Resources.HP = 0
	_ = r.savePlayers(players)
	_ = r.Close()
	reopened := NewRuntimeManager(dir)
	defer reopened.Close()
	players, _ = reopened.loadPlayers()
	if players[id].State != "DEAD" || players[id].Attributes.Resources.HP != 0 {
		t.Fatal("restart revived dead player")
	}
}
func TestAuditAliasMergePreservesInvestment(t *testing.T) {
	p := &model.Player{Backpack: map[string]int{"langfan_jian": 2, "langfan_j": 1}, Equipment: map[string]string{"NECKLACE": "kongming_zhu"}, EnhancedItems: map[string]int{"langfan_jian": 2, "langfan_j": 3}, Flags: map[string]bool{}}
	changed, err := normalizePlayerItemAliases(p)
	if err != nil || !changed {
		t.Fatal(err)
	}
	if p.Backpack["langfan_j"] != 3 || p.Backpack["langfan_jian"] != 0 || p.Equipment["NECKLACE"] != "kongming_z" || p.EnhancedItems["langfan_j"] != 3 || p.Money != 600 || p.Backpack["bronze_fragment"] != 3 {
		t.Fatalf("bad alias merge: %+v", p)
	}
	before, _ := json.Marshal(p)
	changed, err = normalizePlayerItemAliases(p)
	after, _ := json.Marshal(p)
	if err != nil || changed || string(before) != string(after) {
		t.Fatal("alias repair not idempotent")
	}
}
func TestAuditInitiativeAndPassiveNPC(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	before := effectiveAttributesForPlayerNoLock(p)
	d := fileItems["tianma_ling"]
	p.Equipment[d.EquipmentSlot] = d.ID
	p.EnhancedItems = map[string]int{d.ID: 5}
	after := effectiveAttributesForPlayerNoLock(p)
	if after.Initiative != before.Initiative+8 {
		t.Fatal("equipment/upgrade initiative missing")
	}
	r2, id2, _ := createDungeonTestPlayer(t, "abandoned_temple_rain")
	for i := 0; i < 5; i++ {
		_, _ = r2.ProcessCombatTick()
	}
	ps, _ := r2.loadPlayers()
	if ps[id2].State != "IDLE" {
		t.Fatal("ordinary hostile NPC initiated combat")
	}
}

func TestAuditAliasRefundPersistsOnce(t *testing.T) {
	dir := t.TempDir()
	r := NewRuntimeManager(dir)
	_ = r.Register("aliases", "pass123")
	id, err := r.CreateCharacter("aliases", "旧装备侠", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	players, _ := r.loadPlayers()
	p := players[id]
	p.Backpack["langfan_jian"] = 2
	p.Backpack["langfan_j"] = 1
	p.EnhancedItems = map[string]int{"langfan_jian": 2, "langfan_j": 3}
	_ = r.savePlayers(players)
	_ = r.Close()
	for i := 0; i < 2; i++ {
		reopened := NewRuntimeManager(dir)
		players, _ = reopened.loadPlayers()
		p = players[id]
		if p.Money != 5600 || p.Backpack["bronze_fragment"] != 3 || p.Backpack["langfan_j"] != 3 || p.EnhancedItems["langfan_j"] != 3 {
			t.Fatalf("reload %d lost or repeated compensation: %+v", i, p)
		}
		if err := reopened.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAuditSparStopsAtOneAndCancelsCounter(t *testing.T) {
	r := &RuntimeManager{combatNow: func() time.Time { return time.UnixMilli(1000) }, combatRandom: rand.New(rand.NewSource(1))}
	r.initCombatStateNoLock()
	hpA, hpB := 100, 100
	attrs := model.EffectiveAttributes{Attack: 1000, HitRate: 1000, Dodge: 10, Parry: 0, CritRate: 0}
	entA := &combatEntity{key: ActorKey{Kind: "player", EntityID: "a"}, name: "甲", attrs: attrs, hp: &hpA, maxHP: 100, mode: "SPAR"}
	entB := &combatEntity{key: ActorKey{Kind: "player", EntityID: "b"}, name: "乙", attrs: attrs, hp: &hpB, maxHP: 100, mode: "SPAR"}
	// 切磋每次出手伤害上限为受击方MaxHP/5；到20%认输线立即终止，取消对方后续出手。
	for round := 0; round < 100 && hpB > 20; round++ {
		result := model.CombatTickResult{}
		r.runAutoRoundNoLock(entA, entB, "test", int64(1000+round), &result)
	}
	if hpB < 1 || hpB > 20 {
		t.Fatal("spar must stop at yield line and never below 1 HP", hpA, hpB)
	}
	if hpA < hpB {
		t.Fatal("striker swings must be cancelled once opponent reaches the line", hpA, hpB)
	}
	r, id := v2Player(t)
	npc, _ := r.staticNPCInRoomNoLock(fileRooms["ly_weapon_school"], "weapon_school_master")
	if _, err := r.Spar(id, npc.ID); err != nil {
		t.Fatal(err)
	}
	npc.Attributes.Resources.HP--
	players, _ := r.loadPlayers()
	players[id].Backpack["smoke_powder"] = 1
	_ = r.savePlayers(players)
	if _, err := r.UseItem(id, "smoke_powder"); err != nil {
		t.Fatal(err)
	}
	if npc.Attributes.Resources.HP != npc.Attributes.Resources.MaxHP {
		t.Fatal("spar escape left teacher injured")
	}
}
