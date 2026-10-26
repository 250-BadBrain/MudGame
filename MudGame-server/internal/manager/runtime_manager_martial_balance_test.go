package manager

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"mudgame/server/internal/model"
)

// These are rested single-boss encounters, not an entire campaign simulation.
// Sources are restricted to chapters before the target encounter. Mechanism weakness
// flags are supplied as if the player completed the chapter's puzzle before fighting.
func TestV2RoutesAgainstRepresentativeBosses(t *testing.T) {
	type stage struct {
		order, level, trained int
		dungeon, boss, armor  string
	}
	stages := []stage{
		{1, 20, 80, "abandoned_temple_rain", "temple_assassin", "common_clothes"},
		{8, 50, 160, "datang_jingling", "jl_vanguard_commander", "lanjiang_vest"},
		{19, 90, 400, "datang_baling", "db_zhang_xiu", "wagang_tiejia"},
		{38, 120, 800, "tiandao_trial_hall", "tt_song_phantom", "shujin_pifeng"},
		{47, 140, 1200, "datang_xinggong", "gb2_boss", "yunhe_pao"},
		{71, 170, 1600, "posui_tx3", "tx3_boss", "jueji_hu"},
	}
	type route struct {
		kind, root, early, late string
		artIDs                  []string
	}
	routes := []route{
		{"unarmed", "basic_unarmed", "", "", []string{"poxu_zhi", "fuhu_jin"}},
		{"blade", "basic_blade", "steel_blade", "langwang_dao", []string{"tiandao_blade", "xiulou_daofa", "bloody_blade"}},
		{"sword", "basic_sword", "iron_sword", "yanggong_sword", []string{"cihang_jiandian", "taiyi_jianfa", "thunder_sword"}},
		{"spear", "basic_spear", "training_spear", "liangdao_spear", []string{"bawang_qiang"}},
		{"whip", "basic_whip", "training_whip", "youlong_whip", []string{"youlong_bian"}},
		{"mace", "basic_mace", "training_mace", "lieshi_mace", []string{"lieshi_jian"}},
		{"hook", "basic_hook", "training_hook", "suolang_hook", []string{"suolang_gou"}},
		{"staff", "basic_staff", "iron_staff", "jinggang_staff", []string{"fengmo_gun"}},
		{"hidden_weapon", "basic_hidden_weapon", "darts", "tangmen_tili", []string{"mantian_huayu"}},
	}
	orders := map[string]int{}
	// Resolve source chronology from runtime templates rather than trusting display text.
	probe := NewRuntimeManager(t.TempDir())
	templates := probe.dungeonTemplatesNoLock()
	for _, tpl := range templates {
		orders[tpl.ID] = tpl.Order
	}
	_ = probe.Close()
	available := func(sid string, phase int) bool {
		d := fileSkills[sid]
		return d.SourceDungeon == "" || orders[d.SourceDungeon] < phase
	}
	type result struct {
		Order       int    `json:"order"`
		Route       string `json:"route"`
		Seed        int    `json:"seed"`
		Rounds      int    `json:"rounds"`
		Potions     int    `json:"potions"`
		RemainingHP int    `json:"remainingHp"`
	}
	records := []result{}
	for _, phase := range stages {
		for _, build := range routes {
			t.Run(fmt.Sprintf("%d/%s", phase.order, build.kind), func(t *testing.T) {
				r, id, initial := createDungeonTestPlayer(t, phase.dungeon)
				for seed := 1; seed <= 3; seed++ {
					inst := initial
					if seed > 1 {
						r.mu.Lock()
						r.dungeonMgr.DestroyInstanceNoLock(initial.ID)
						var err error
						inst, err = r.dungeonMgr.CreateInstanceNoLock(id, phase.dungeon, "ly_00")
						r.mu.Unlock()
						if err != nil {
							t.Fatal(err)
						}
						initial = inst
					}
					bossID := ""
					for nid, n := range inst.NPCs {
						if n.TemplateID == phase.boss {
							bossID = nid
							break
						}
					}
					if bossID == "" {
						t.Fatal("missing representative boss")
					}
					r.mu.Lock()
					r.combatRandom = rand.New(rand.NewSource(int64(seed)))
					players, _ := r.loadPlayers()
					p := players[id]
					p.Attributes = model.NewPlayerAttributes(16, 16, 16, 16, 16)
					root := (phase.level - 20) / 10
					p.Attributes.Experience = 40 * root * root * root
					p.Attributes.Resources.TrainedMaxMP = phase.trained
					p.Skills = map[string]int{}
					p.Flags = map[string]bool{}
					p.ActiveArts = map[string]string{}
					p.Equipment = map[string]string{"BODY": phase.armor}
					p.EnhancedItems = map[string]int{}
					p.Backpack = map[string]int{"xiaohuandan": 3}
					p.RecoveryItemReadyAt = 0
					p.State = "IDLE"
					p.TargetID = ""
					p.WeakUntil = 0
					p.Online = true
					p.RoomID = inst.NPCs[bossID].RoomID
					for sid, def := range fileSkills {
						if def.Type == "BASIC" {
							p.Skills[sid] = phase.level
						}
					}
					if build.early != "" {
						p.Equipment["WEAPON"] = build.early
					}
					if phase.order >= 38 && build.late != "" {
						p.Equipment["WEAPON"] = build.late
					}
					selectArt := func(candidates []string) {
						for _, sid := range candidates {
							if available(sid, phase.order) {
								grantArt(p, sid, min(100, phase.level))
								p.ActiveArts[fileSkills[sid].Slot] = sid
								break
							}
						}
					}
					selectArt(build.artIDs)
					selectArt([]string{"csj_changshengjue", "jyg_zhanshentu_dao", "dongting_xinf", "hunyuan_neigong"})
					selectArt([]string{"taxue_wuhen", "lingxu_bu", "talang_jue"})
					selectArt([]string{"jingang_buhuai", "tiebi_gong"})
					p.Flags[inst.NPCs[bossID].WeaknessFlag] = true
					eff := effectiveAttributesForPlayerNoLock(p)
					p.Attributes.Resources.HP = eff.MaxHP
					p.Attributes.Resources.MP = eff.TrainedMaxMP
					_ = r.savePlayers(players)
					r.mu.Unlock()
					if _, err := r.Kill(id, bossID); err != nil {
						t.Fatal(err)
					}
					rounds := 0
					lastHealRound := -5
					for ; rounds < 100; rounds++ {
						if _, err := r.ProcessCombatTick(); err != nil {
							t.Fatal(err)
						}
						players, _ = r.loadPlayers()
						p = players[id]
						if p.State != "COMBAT" {
							rounds++
							break
						}
						if (rounds+1)-lastHealRound >= 5 && p.Attributes.Resources.HP < effectiveAttributesForPlayerNoLock(p).MaxHP-50 && p.Backpack["xiaohuandan"] > 0 {
							// Advance only the synthetic recovery timestamp to model 5 x 2s rounds.
							lastHealRound = rounds + 1
							p.RecoveryItemReadyAt = time.Now().UnixMilli() - 1
							_ = r.savePlayers(players)
							if _, err := r.UseItem(id, "xiaohuandan"); err != nil {
								t.Fatal(err)
							}
						}
					}
					players, _ = r.loadPlayers()
					p = players[id]
					boss := inst.NPCs[bossID]
					if p.State == "DEAD" || p.Attributes.Resources.HP <= 0 || (boss.Alive && !boss.Defeated) {
						t.Fatalf("seed %d failure: rounds=%d potions=%d initialHP=%d attack=%d defense=%d dodge=%d parry=%d npcAttack=%d npcHit=%d weakness=%v bossHP=%d", seed, rounds, 3-p.Backpack["xiaohuandan"], eff.MaxHP, eff.Attack, eff.Defense, eff.Dodge, eff.Parry, boss.Attributes.Derived.Attack, boss.Attributes.Derived.HitRate, boss.WeaknessApplied, boss.Attributes.Resources.HP)
					}
					records = append(records, result{phase.order, build.kind, seed, rounds, 3 - p.Backpack["xiaohuandan"], p.Attributes.Resources.HP})
				}
			})
		}
	}
	if path := os.Getenv("MUD_BALANCE_REPORT"); path != "" {
		data, _ := json.MarshalIndent(records, "", "  ")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("Verified %d rested boss encounters across 9 routes and 6 stages; at most 3 potions per fight", len(records))
}
