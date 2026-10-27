package manager

import (
	"math/rand"
	"mudgame/server/internal/model"
	"testing"
)

func TestContentRecoveryRequiresFullInterval(t *testing.T) {
	for _, action := range []string{"heal", "meditate"} {
		t.Run(action, func(t *testing.T) {
			r, id := v2Player(t)
			players, _ := r.loadPlayers()
			p := players[id]
			p.Attributes.Resources.HP = 20
			p.Attributes.Resources.MP = 30
			if err := r.savePlayers(players); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 3; i++ {
				var err error
				if action == "heal" {
					_, err = r.Heal(id)
				} else {
					_, err = r.Meditate(id)
				}
				if err != nil {
					t.Fatal(err)
				}
				r.Stop(id)
			}
			players, _ = r.loadPlayers()
			p = players[id]
			if p.Attributes.Resources.HP != 20 || p.Attributes.Resources.MP != 30 {
				t.Fatalf("start/stop changed resources: %+v", p.Attributes.Resources)
			}
			if action == "heal" {
				_, _ = r.Heal(id)
			} else {
				_, _ = r.Meditate(id)
			}
			for i := 0; i < 9; i++ {
				_, _ = r.ProcessStatusTick()
			}
			players, _ = r.loadPlayers()
			if players[id].Attributes.Resources.HP != 20 || players[id].Attributes.Resources.MP != 30 {
				t.Fatal("early settlement")
			}
			_, _ = r.ProcessStatusTick()
			players, _ = r.loadPlayers()
			p = players[id]
			if action == "heal" && (p.Attributes.Resources.HP != 40 || p.Attributes.Resources.MP != 22) {
				t.Fatalf("wrong healing settlement: %+v", p.Attributes.Resources)
			}
			if action == "meditate" && p.Attributes.Resources.MP != 45 {
				t.Fatalf("wrong meditation: %+v", p.Attributes.Resources)
			}
		})
	}
}

func TestContentDeathCannotBeCancelledAndCombatCannotLeave(t *testing.T) {
	for _, state := range []string{"DEAD", "COMBAT"} {
		t.Run(state, func(t *testing.T) {
			r, id, _ := createDungeonTestPlayer(t, "abandoned_temple_rain")
			players, _ := r.loadPlayers()
			p := players[id]
			p.State = state
			if state == "DEAD" {
				p.Attributes.Resources.HP = 0
			}
			_ = r.savePlayers(players)
			r.Stop(id)
			players, _ = r.loadPlayers()
			if players[id].State != state {
				t.Fatal("stop bypassed state")
			}
			if _, err := r.LeaveDungeon(id); err == nil {
				t.Fatal("unsafe exit accepted")
			}
			players, _ = r.loadPlayers()
			if players[id].State != state {
				t.Fatal("exit changed state")
			}
		})
	}
}

func TestContentReliveCapsCurrentButKeepsPermanentMP(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.State = "DEAD"
	p.Attributes.Resources.HP = 0
	p.Attributes.Resources.TrainedMaxMP = 4000
	p.Attributes.Resources.MP = 0
	want := effectiveAttributesForPlayerNoLock(p).TrainedMaxMP / 2
	_ = r.savePlayers(players)
	if _, err := r.Relive(id); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Attributes.Resources.MP != want || p.Attributes.Resources.TrainedMaxMP != 4000 {
		t.Fatalf("wrong revived MP: %+v", p.Attributes.Resources)
	}
}

func TestContentSmokeEscapesWithoutFullClearOrHealing(t *testing.T) {
	r, id, inst := createDungeonTestPlayer(t, "abandoned_temple_rain")
	players, _ := r.loadPlayers()
	p := players[id]
	p.Backpack["smoke_powder"] = 2
	p.Attributes.Resources.HP = 23
	p.State = "COMBAT"
	p.TargetID = "enemy"
	p.CombatType = "KILL"
	_ = r.savePlayers(players)
	if _, err := r.UseItem(id, "smoke_powder"); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.RoomID != inst.EntryRoomID || p.State != "IDLE" || p.TargetID != "" || p.Attributes.Resources.HP != 23 || p.Backpack["smoke_powder"] != 1 || p.Flags["dungeon_cleared_abandoned_temple_rain"] {
		t.Fatalf("bad escape: %+v", p)
	}
	if _, exists := r.dungeonMgr.PlayerInstanceNoLock(id); exists {
		t.Fatal("escaped instance retained")
	}
	if _, err := r.UseItem(id, "smoke_powder"); err == nil {
		t.Fatal("noncombat powder consumed")
	}
	players, _ = r.loadPlayers()
	if players[id].Backpack["smoke_powder"] != 1 {
		t.Fatal("failed escape consumed powder")
	}
}

func TestContentEnhanceRequiresOwnershipAndIdle(t *testing.T) {
	r, id := v2Player(t)
	for _, state := range []string{"IDLE", "COMBAT", "LEARNING", "DEAD"} {
		players, _ := r.loadPlayers()
		p := players[id]
		p.State = state
		p.Money = 5000
		p.Backpack = map[string]int{"bronze_fragment": 5}
		p.EnhancedItems = map[string]int{}
		p.Equipment = map[string]string{}
		if state != "IDLE" {
			p.Backpack["iron_sword"] = 1
		}
		_ = r.savePlayers(players)
		if _, err := r.EnhanceItem(id, "iron_sword"); err == nil {
			t.Fatalf("accepted %s or unowned item", state)
		}
		players, _ = r.loadPlayers()
		p = players[id]
		if p.Money != 5000 || p.Backpack["bronze_fragment"] != 5 || p.EnhancedItems["iron_sword"] != 0 {
			t.Fatal("failure charged")
		}
	}
}

func TestContentSmokeClearsBothSparringTargets(t *testing.T) {
	r, id := v2Player(t)
	other, err := r.CreateCharacter("learner", "对练者", 16, 16, 16, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	players, _ := r.loadPlayers()
	for _, pid := range []string{id, other} {
		p := players[pid]
		p.RoomID = "ly_weapon_school"
		p.State = "COMBAT"
		p.CombatType = "SPAR"
		p.Backpack = map[string]int{"smoke_powder": 1}
		p.Attributes.Resources.HP = 50
	}
	players[id].TargetID = other
	players[other].TargetID = id
	_ = r.savePlayers(players)
	if _, err := r.UseItem(id, "smoke_powder"); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	for _, p := range []*model.Player{players[id], players[other]} {
		if p.State != "IDLE" || p.TargetID != "" || p.Attributes.Resources.HP != 50 {
			t.Fatal("spar escape left combat or healed")
		}
	}
}

func TestContentSameCityReturnDoesNotChargeCarriage(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	players[id].Money = 0
	_ = r.savePlayers(players)
	if _, err := r.EnterWorld(id, "luoyang"); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	if players[id].Money != 0 || players[id].RoomID != "ly_00" {
		t.Fatal("same-city carriage changed funds")
	}
}

func TestContentPublicServicesRejectKillAndTeacherAllowsSpar(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Attributes.Resources.HP = effectiveAttributesForPlayerNoLock(p).MaxHP
	_ = r.savePlayers(players)
	if _, err := r.Kill(id, "weapon_school_master"); err == nil {
		t.Fatal("teacher can be killed")
	}
	if _, err := r.Spar(id, "weapon_school_master"); err != nil {
		t.Fatal("teacher spar denied", err)
	}
}

func TestContentSparUsesEffectiveHealthThreshold(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.Skills["basic_neigong"] = 120
	p.Attributes.Resources.TrainedMaxMP = 1200
	p.Attributes.Resources.HP = effectiveAttributesForPlayerNoLock(p).MaxHP / 5
	p.State = "COMBAT"
	p.TargetID = "weapon_school_master"
	p.CombatType = "SPAR"
	_ = r.savePlayers(players)
	npc, _ := r.staticNPCStateNoLock("weapon_school_master", p.RoomID)
	npc.Attributes.Derived.Attack = 1
	npc.Attributes.Derived.HitRate = 1
	r.combatRandom = rand.New(rand.NewSource(1))
	if _, err := r.ProcessCombatTick(); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	if players[id].State != "IDLE" {
		t.Fatal("spar ignored effective 20 percent threshold")
	}
}

func TestContentHealingCannotSpendHiddenExcessMP(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.State = "HEALING"
	p.StateTickCountdown = 1
	p.Attributes.Resources.HP = 20
	p.Attributes.Resources.TrainedMaxMP = 4000
	p.Attributes.Resources.MP = 4000
	allowed := effectiveAttributesForPlayerNoLock(p).MP
	_ = r.savePlayers(players)
	if _, err := r.ProcessStatusTick(); err != nil {
		t.Fatal(err)
	}
	players, _ = r.loadPlayers()
	if players[id].Attributes.Resources.MP != allowed-8 {
		t.Fatal("healing spent an invisible MP reservoir")
	}
}
