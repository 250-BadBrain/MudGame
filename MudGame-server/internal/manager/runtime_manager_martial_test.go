package manager

import (
	"bytes"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/model"
	"mudgame/server/internal/storage"
)

func v2Player(t *testing.T) (*RuntimeManager, string) {
	t.Helper()
	r, id := setupLearningManager(t)
	t.Cleanup(func() { _ = r.Close() })
	players, _ := r.loadPlayers()
	p := players[id]
	p.RoomID = "ly_weapon_school"
	p.Online = true
	if err := r.savePlayers(players); err != nil {
		t.Fatal(err)
	}
	return r, id
}

func grantArt(p *model.Player, id string, level int) {
	def := fileSkills[id]
	p.Skills[id] = level
	if p.Flags == nil {
		p.Flags = map[string]bool{}
	}
	if def.TeacherID != "" {
		p.Flags["martial_taught_"+id] = true
	}
	if def.SourceDungeon != "" {
		p.Flags["dungeon_cleared_"+def.SourceDungeon] = true
	}
}

func TestV2PracticeCarriesBudgetAndDoesNotSpendPastCap(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	players[id].Attributes.Potential = 20
	_ = r.savePlayers(players)
	if err := r.PracticeSkill(id, "basic_unarmed"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := r.ProcessStatusTick(); err != nil {
			t.Fatal(err)
		}
	}
	players, _ = r.loadPlayers()
	p := players[id]
	if p.Skills["basic_unarmed"] != 11 || p.Attributes.Potential != 10 {
		t.Fatalf("budget not carried: %+v", p)
	}
	for i := 0; i < 10; i++ {
		_, _ = r.ProcessStatusTick()
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Skills["basic_unarmed"] != 20 || p.Attributes.Potential != 1 || p.State != "IDLE" {
		t.Fatalf("cap consumed unused budget: level=%d potential=%d state=%s", p.Skills["basic_unarmed"], p.Attributes.Potential, p.State)
	}
}

func TestV2ManualFailsWithoutConsumingAndAllEntrancesShareGate(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Backpack["manual_csj_changshengjue"] = 1
	p.Skills["basic_neigong"] = 120
	p.Skills["basic_literacy"] = 80
	_ = r.savePlayers(players)
	if _, err := r.UseItem(id, "manual_csj_changshengjue"); err == nil {
		t.Fatal("missing transmission accepted")
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Backpack["manual_csj_changshengjue"] != 1 {
		t.Fatal("failed insight consumed book")
	}
	p.Flags["dungeon_cleared_datang_xinggong"] = true
	_ = r.savePlayers(players)
	if _, err := r.UseItem(id, "manual_csj_changshengjue"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.EquipSkill(id, "csj_changshengjue"); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	p.State = "COMBAT"
	_ = r.savePlayers(players)
	if _, err := r.UnequipSkill(id, "internal"); err == nil {
		t.Fatal("combat switch accepted")
	}
	players, _ = r.loadPlayers()
	p = players[id]
	p.State = "IDLE"
	p.Flags["dungeon_cleared_datang_xinggong"] = false
	_ = r.savePlayers(players)
	if err := r.PracticeSkill(id, "csj_changshengjue"); err == nil {
		t.Fatal("practice bypassed source gate")
	}
	if skillActiveReason(p, fileSkills["csj_changshengjue"]) == "" {
		t.Fatal("invalid skill still active")
	}
}

func TestV2OnlyCurrentWeaponContributesAndEnhancementRequiresEquipment(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Skills["basic_sword"] = 100
	p.Skills["basic_blade"] = 100
	p.Skills["basic_unarmed"] = 100
	grantArt(p, "thunder_sword", 100)
	p.ActiveArts["combat"] = "thunder_sword"
	p.Equipment["WEAPON"] = "iron_sword"
	before := effectiveAttributesForPlayerNoLock(p)
	p.Skills["basic_unarmed"] = 500
	p.Skills["basic_blade"] = 500
	p.EnhancedItems = map[string]int{"steel_blade": 5}
	after := effectiveAttributesForPlayerNoLock(p)
	if after.Attack != before.Attack {
		t.Fatalf("inactive skill or stored enhancement leaked: %d/%d", before.Attack, after.Attack)
	}
	p.Equipment["WEAPON"] = "steel_blade"
	if skillActiveReason(p, fileSkills["thunder_sword"]) == "" {
		t.Fatal("incompatible sword still active")
	}
}

func TestV2InnerCapOrderingAndSwitchDoesNotHealOrDestroyTraining(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Skills["basic_neigong"] = 120
	p.Skills["basic_literacy"] = 120
	grantArt(p, "csj_changshengjue", 100)
	p.ActiveArts["internal"] = "csj_changshengjue"
	p.Attributes.Resources.TrainedMaxMP = 2020
	p.Attributes.Resources.HP = 50
	p.Attributes.Resources.MP = 50
	eff := effectiveAttributesForPlayerNoLock(p)
	if eff.MaxMP != 2020 || eff.MaxHP != 1033 {
		t.Fatalf("unexpected caps: %+v", eff)
	}
	_ = r.savePlayers(players)
	if _, err := r.UnequipSkill(id, "internal"); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	eff = effectiveAttributesForPlayerNoLock(p)
	if eff.TrainedMaxMP != 1420 || eff.MaxHP != 550 || p.Attributes.Resources.TrainedMaxMP != 2020 {
		t.Fatalf("temporary cap polluted permanent progress: %+v", eff)
	}
	if _, err := r.EquipSkill(id, "csj_changshengjue"); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Attributes.Resources.HP != 50 || p.Attributes.Resources.MP != 50 {
		t.Fatal("switching healed resources")
	}
}

func TestV2RecoveryCooldownSharedAndFailedUseDoesNotConsume(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Attributes.Resources.HP = 1
	p.Backpack["simple_medicine"] = 2
	p.Backpack["dry_ration"] = 2
	_ = r.savePlayers(players)
	if _, err := r.UseItem(id, "simple_medicine"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.UseItem(id, "dry_ration"); err == nil {
		t.Fatal("food bypassed recovery cooldown")
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Backpack["dry_ration"] != 2 {
		t.Fatal("failed cooldown consumed food")
	}
	ready := p.RecoveryItemReadyAt
	r.SetPlayerOnline(id, false)
	players, _ = r.loadPlayers()
	if players[id].RecoveryItemReadyAt != ready {
		t.Fatal("disconnect reset cooldown")
	}
	players[id].RecoveryItemReadyAt = time.Now().UnixMilli() - 1
	_ = r.savePlayers(players)
	if _, err := r.UseItem(id, "dry_ration"); err != nil {
		t.Fatal(err)
	}
}

func TestV2SparseOrderStillRequiresPreviousExistingGroup(t *testing.T) {
	r := &RuntimeManager{}
	p := &model.Player{Flags: map[string]bool{}}
	templates := []engruntime.Template{{ID: "before", Order: 26}, {ID: "after", Order: 28}}
	if locked, _ := r.dungeonLockStateNoLock(p, templates[1], templates); !locked {
		t.Fatal("gap unlocked order28")
	}
	p.Flags["dungeon_cleared_before"] = true
	if locked, _ := r.dungeonLockStateNoLock(p, templates[1], templates); locked {
		t.Fatal("previous existing completion ignored")
	}
}

func TestV2ParryGrowthHasRealEffect(t *testing.T) {
	if math.Abs(boundedParryChance(56, 140)-.2857142857) > .00001 || math.Abs(boundedParryChance(78, 140)-.3577981651) > .00001 {
		t.Fatal("parry is still pinned to old floor")
	}
	if boundedChance(1, 1000) != .15 || boundedParryChance(1000, 1) != .60 {
		t.Fatal("incorrect independent probability bounds")
	}
}

func TestV2TransmissionAndWeaponRewardsDoNotDuplicate(t *testing.T) {
	p := &model.Player{Skills: map[string]int{}, Backpack: map[string]int{}, Flags: map[string]bool{}}
	s := engruntime.Settlement{Progress: 100, DungeonID: "datang_xinggong", RewardItems: map[string]int{"manual_csj_changshengjue": 1, "liangdao_spear": 1}}
	applyDungeonSettlementReward(p, s)
	applyDungeonSettlementReward(p, s)
	if p.Backpack["manual_csj_changshengjue"] != 1 {
		t.Fatal("FirstClaim item duplicated")
	}
	// liangdao_spear is probabilistic equipment (drop rate system)
}

func TestV2DefeatBossYieldsBeforeCounterattack(t *testing.T) {
	r, id, inst := createDungeonTestPlayer(t, "tiandao_trial_hall")
	bossID := ""
	for nid, n := range inst.NPCs {
		if n.TemplateID == "tt_song_phantom" {
			bossID = nid
		}
	}
	if bossID == "" {
		t.Fatal("missing boss")
	}
	boss := inst.NPCs[bossID]
	players, _ := r.loadPlayers()
	p := players[id]
	p.RoomID = boss.RoomID
	p.Attributes.Base.Strength = 1000
	p.Attributes.Resources.HP = 1
	boss.Attributes.Resources.HP = boss.Attributes.Resources.MaxHP/4 + 1
	boss.Attributes.Derived.Dodge = 1
	boss.Attributes.Derived.Parry = 0
	boss.Attributes.Derived.Initiative = 0
	boss.Attributes.Derived.Attack = 9999
	boss.Attributes.Derived.HitRate = 9999
	r.combatRandom = rand.New(rand.NewSource(1))
	_ = r.savePlayers(players)
	if _, err := r.Kill(id, bossID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ProcessCombatTick(); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if !boss.Defeated || !boss.Alive || boss.Attributes.Resources.HP != boss.Attributes.Resources.MaxHP/4 {
		t.Fatal("boss did not yield at quarter health")
	}
	if p.State == "DEAD" || p.Attributes.Resources.HP != 1 {
		t.Fatal("yielded boss still counterattacked")
	}
}

func TestV2OldSaveRequiresExplicitMigrationWithoutRewritingIt(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewFileStore(dir)
	old := &model.Player{ID: "old", Attributes: model.NewPlayerAttributes(16, 16, 16, 16, 16), Skills: map[string]int{"basic_unarmed": 1}}
	if err := store.SavePlayers(map[string]*model.Player{"old": old}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "players", "old.json")
	before, _ := os.ReadFile(path)
	panicked := false
	func() { defer func() { panicked = recover() != nil }(); _ = NewRuntimeManager(dir) }()
	if !panicked {
		t.Fatal("unmigrated save accepted")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("startup rewrote old player")
	}
}

func TestV2ActiveArtsAndRecoveryCooldownSurviveStorageRoundTrip(t *testing.T) {
	store := storage.NewFileStore(t.TempDir())
	p := &model.Player{ID: "v2", MartialVersion: model.MartialVersion, ActiveArts: map[string]string{"combat": "fuhu_jin"}, RecoveryItemReadyAt: 2000000000000,
		Attributes: model.NewPlayerAttributes(16, 16, 16, 16, 16), Skills: map[string]int{"fuhu_jin": 20}, Flags: map[string]bool{"martial_taught_fuhu_jin": true}}
	if err := store.SavePlayers(map[string]*model.Player{p.ID: p}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadPlayers()
	if err != nil {
		t.Fatal(err)
	}
	got := loaded[p.ID]
	if got.MartialVersion != model.MartialVersion || got.ActiveArts["combat"] != "fuhu_jin" || got.RecoveryItemReadyAt != p.RecoveryItemReadyAt {
		t.Fatalf("V2 state lost on restart: %+v", got)
	}
}
