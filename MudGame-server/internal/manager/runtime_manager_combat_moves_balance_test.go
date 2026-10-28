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

// 平衡验证（提案§10 / 实现规范§10）：同预算角色对代表性首领，
// 比较"仅普攻"与"主动+普攻"两种策略的胜率与战斗时长。
// 种子与配置固定，结果经 MUD_MOVES_BALANCE_REPORT 输出JSON供 balance_results.md 引用。
func TestV2MovesBalanceAgainstRepresentativeBosses(t *testing.T) {
	type stage struct {
		order, level, trained int
		dungeon, boss, armor  string
	}
	stages := []stage{
		{8, 50, 160, "datang_jingling", "jl_vanguard_commander", "lanjiang_vest"},
		{38, 120, 800, "tiandao_trial_hall", "tt_song_phantom", "shujin_pifeng"},
		{71, 170, 1600, "posui_tx3", "tx3_boss", "jueji_hu"},
	}
	routes := []struct {
		kind, root, early, late, art string
	}{
		{"unarmed", "basic_unarmed", "", "", "fuhu_jin"},
		{"blade", "basic_blade", "steel_blade", "langwang_dao", "xiulou_daofa"},
		{"spear", "basic_spear", "training_spear", "liangdao_spear", "bawang_qiang"},
	}
	orders := map[string]int{}
	probe := NewRuntimeManager(t.TempDir())
	for _, tpl := range probe.dungeonTemplatesNoLock() {
		orders[tpl.ID] = tpl.Order
	}
	_ = probe.Close()
	weaponOrder := map[string]int{"steel_blade": 0, "training_spear": 0, "langwang_dao": 15, "liangdao_spear": 18}

	type record struct {
		Order   int    `json:"order"`
		Route   string `json:"route"`
		Policy  string `json:"policy"`
		Seed    int    `json:"seed"`
		Win     bool   `json:"win"`
		Rounds  int    `json:"rounds"`
		Potions int    `json:"potions"`
	}
	records := []record{}
	for _, ph := range stages {
		for _, build := range routes {
			for _, policy := range []string{"auto_only", "moves"} {
				for seed := 1; seed <= 3; seed++ {
					r, id, inst := createDungeonTestPlayer(t, ph.dungeon)
					bossID := ""
					for nid, n := range inst.NPCs {
						if n.TemplateID == ph.boss {
							bossID = nid
							break
						}
					}
					if bossID == "" {
						t.Fatal("missing boss")
					}
					r.mu.Lock()
					r.combatRandom = rand.New(rand.NewSource(int64(seed)))
					players, _ := r.loadPlayers()
					p := players[id]
					p.Attributes = model.NewPlayerAttributes(16, 16, 16, 16, 16)
					root := (ph.level - 20) / 10
					p.Attributes.Experience = 40 * root * root * root
					p.Skills = map[string]int{}
					p.Flags = map[string]bool{}
					p.ActiveArts = map[string]string{}
					p.Equipment = map[string]string{"BODY": ph.armor}
					p.Backpack = map[string]int{"xiaohuandan": 3}
					p.State = "IDLE"
					p.Online = true
					for sid, def := range fileSkills {
						if def.Type == "BASIC" {
							p.Skills[sid] = ph.level
						}
					}
					if build.early != "" {
						p.Equipment["WEAPON"] = build.early
					}
					if weaponOrder[build.late] < ph.order {
						p.Equipment["WEAPON"] = build.late
					}
					grantArt(p, build.art, min(100, ph.level))
					p.ActiveArts[fileSkills[build.art].Slot] = build.art
					grantArt(p, "hunyuan_neigong", min(100, ph.level))
					p.ActiveArts["internal"] = "hunyuan_neigong"
					grantArt(p, "talang_jue", min(100, ph.level))
					p.ActiveArts["movement"] = "talang_jue"
					grantArt(p, "tiebi_gong", min(100, ph.level))
					p.ActiveArts["guard"] = "tiebi_gong"
					eff := effectiveAttributesForPlayerNoLock(p)
					p.Attributes.Resources.TrainedMaxMP = eff.MaxMP
					eff = effectiveAttributesForPlayerNoLock(p)
					p.Attributes.Resources.HP = eff.MaxHP
					p.Attributes.Resources.MP = eff.TrainedMaxMP
					_ = r.savePlayers(players)
					r.mu.Unlock()
					moveDungeonTestPlayer(t, r, id, inst.NPCs[bossID].RoomID)
					if _, err := r.Kill(id, bossID); err != nil {
						t.Fatal(err)
					}

					// 可注入时钟：每轮+2秒
					now := int64(1_000_000)
					r.combatNow = func() time.Time { return time.UnixMilli(now) }
					sequence := int64(0)
					moveCasts := 0
					battleID := ""
					rounds := 0
					lastHealRound := -5
					for ; rounds < 100; rounds++ {
						// 与既有平衡测试一致：每5轮按需用小还丹（恢复冷却对齐结算点）
						{
							pls, _ := r.loadPlayers()
							pl := pls[id]
							if pl.Attributes.Resources.HP < effectiveAttributesForPlayerNoLock(pl).MaxHP-50 && pl.Backpack["xiaohuandan"] > 0 && rounds-lastHealRound >= 5 {
								lastHealRound = rounds
								pl.RecoveryItemReadyAt = time.Now().UnixMilli() - 1
								_ = r.savePlayers(pls)
								_, _ = r.UseItem(id, "xiaohuandan")
							}
						}
						if policy == "moves" {
							view, err := r.GetCombatMoves(id)
							if err != nil {
								t.Fatal(err)
							}
							battleID = view.BattleID
							if battleID != "" {
								for _, mv := range view.Moves {
									if mv.UnavailableReason != "" || mv.Kind != "attack" {
										continue
									}
									actionEnd := view.ActionReadyAt
									skillEnd := mv.SkillReadyAt
									if max(actionEnd, skillEnd) > now {
										continue
									}
									res, _ := r.CastCombatMove(id, mv.ID, battleID, sequence)
									if res.Code == "" {
										sequence = res.AppliedSequence
										moveCasts++
									}
									break
								}
							}
						}
						now += 2000
						if _, err := r.ProcessCombatTick(); err != nil {
							t.Fatal(err)
						}
						pls, _ := r.loadPlayers()
						pl := pls[id]
						if pl.State != "COMBAT" {
							rounds++
							break
						}
					}
					pls, _ := r.loadPlayers()
					pl := pls[id]
					boss := inst.NPCs[bossID]
					// KILL型首领以Alive=false表达死亡；defeat型以Defeated表达认负
					bossDown := !boss.Alive || boss.Defeated
					win := pl.State != "DEAD" && pl.Attributes.Resources.HP > 0 && bossDown
					if !win && os.Getenv("MUD_MOVES_DEBUG") != "" {
						t.Logf("state: player_state=%s boss_alive=%v boss_defeated=%v", pl.State, boss.Alive, boss.Defeated)
						eff := effectiveAttributesForPlayerNoLock(pl)
						t.Logf("loss: order=%d %s %s seed=%d rounds=%d hp=%d maxhp=%d npc_hp=%d npc_maxhp=%d npc_atk=%d npc_hit=%d atk=%d def=%d dodge=%d potions_left=%d",
							ph.order, build.kind, policy, seed, rounds, pl.Attributes.Resources.HP, eff.MaxHP,
							boss.Attributes.Resources.HP, boss.Attributes.Resources.MaxHP,
							boss.Attributes.Derived.Attack, boss.Attributes.Derived.HitRate,
							eff.Attack, eff.Defense, eff.Dodge, pl.Backpack["xiaohuandan"])
					}
					records = append(records, record{ph.order, build.kind, policy, seed, win, rounds, 3 - pl.Backpack["xiaohuandan"]})
					if err := r.Close(); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
	// 汇总：胜率与平均轮次
	summarize := func(policy string, order int, kind string) (wins, total int, avgRounds float64) {
		w, n, rr := 0, 0, 0
		for _, rec := range records {
			if rec.Policy == policy && rec.Order == order && rec.Route == kind {
				n++
				rr += rec.Rounds
				if rec.Win {
					w++
				}
			}
		}
		if n == 0 {
			return 0, 0, 0
		}
		return w, n, float64(rr) / float64(n)
	}
	for _, ph := range stages {
		for _, build := range routes {
			w0, n0, r0 := summarize("auto_only", ph.order, build.kind)
			w1, n1, r1 := summarize("moves", ph.order, build.kind)
			if n0 == 0 || n1 == 0 {
				continue
			}
			t.Logf("order %d %s: auto_only %d/%d win avg %.1f rounds | moves %d/%d win avg %.1f rounds",
				ph.order, build.kind, w0, n0, r0, w1, n1, r1)
			// 发布阻断：主动不能把可通关配置打到胜率暴跌或无响应秒杀
			if w0 == n0 && w1 < n1 {
				t.Errorf("moves broke a winning config: order %d %s (%d/%d)", ph.order, build.kind, w1, n1)
			}
		}
	}
	if path := os.Getenv("MUD_MOVES_BALANCE_REPORT"); path != "" {
		data, _ := json.MarshalIndent(records, "", "  ")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	_ = fmt.Sprint()
}
