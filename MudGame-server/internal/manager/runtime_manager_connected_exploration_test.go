package manager

import (
	"encoding/json"
	"mudgame/server/internal/gamedata"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Continue the original ordinary solution after a substitute step. This catches
// an alternate state that looks successful but strands the rest of the graph.
func finishConnectedChapter(t *testing.T, r *RuntimeManager, id string) {
	t.Helper()
	inst, _ := r.dungeonMgr.PlayerInstanceNoLock(id)
	tpl, _ := r.dungeonMgr.DungeonTemplateNoLock(inst.DungeonID)
	dir, _ := gamedata.ResolveResourceDir()
	raw, err := os.ReadFile(filepath.Join(dir, "dungeons", inst.DungeonID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var d depthChapter
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	for nid, npc := range inst.NPCs {
		if npc.CombatMode != "protected" {
			_, _, _, _ = r.dungeonMgr.DefeatNPCNoLock(nid)
		}
	}
	for _, step := range d.ContentFeature.Solution {
		a := tpl.PuzzleActions[step.ActionID]
		state := inst.PuzzleStates[a.PuzzleID]
		if state == a.NextState || (a.PuzzleID == "content_support" && state == "ready") {
			continue
		}
		moveDungeonTestPlayer(t, r, id, r.dungeonMgr.InstanceRoomIDNoLock(inst, step.RoomID))
		res, err := r.DoRoomAction(id, step.ActionID)
		if err != nil || res["actionStatus"] != "success" {
			t.Fatal("substitute stranded normal continuation", step.ActionID, res, err)
		}
	}
	if inst.PuzzleStates["gate"] != "open" {
		t.Fatal("substitute did not lead to open final gate")
	}
}

func TestConnectedSixMartialRoutesRequireRealLearning(t *testing.T) {
	cases := []struct{ chapter, room, skill, puzzle, state string }{
		{"datang_yunhe", "dat_02", "talang_jue", "content_compare", "checked"},
		{"dongming_chuanfen", "cf_02", "hunyuan_neigong", "content_support", "ready"},
		{"jiangnan_jianshou", "jia_01", "thunder_sword", "content_trace", "known"},
		{"datang_xb2", "xb2_02", "bloody_blade", "content_detail", "known"},
		{"posui_zhongjuan", "pos_02", "wuzhen_pian", "content_detail", "known"},
		{"bashu_tangmen_trial", "bas_01", "tiebi_gong", "content_trace", "known"},
	}
	for _, c := range cases {
		t.Run(c.chapter, func(t *testing.T) {
			r, id, inst := createDungeonTestPlayer(t, c.chapter)
			moveDungeonTestPlayer(t, r, id, r.dungeonMgr.InstanceRoomIDNoLock(inst, c.room))
			players, _ := r.loadPlayers()
			p := players[id]
			p.Skills = map[string]int{}
			_ = r.savePlayers(players)
			beforeState := inst.PuzzleStates[c.puzzle]
			result, err := r.DoRoomAction(id, "connection_skill")
			if err != nil || result["actionStatus"] != "blocked" || inst.PuzzleStates[c.puzzle] != beforeState {
				t.Fatal("unlearned route not safely rejected", result, err)
			}
			payload, _, _ := r.GetRoomSnapshot(id)
			for _, aid := range payload["actions"].([]string) {
				if aid == "connection_skill" {
					t.Fatal("unlearned route visible")
				}
			}
			players, _ = r.loadPlayers()
			p = players[id]
			p.Skills[c.skill] = 1
			_ = r.savePlayers(players)
			result, err = r.DoRoomAction(id, "connection_skill")
			if err != nil || result["actionStatus"] != "blocked" {
				t.Fatal("forged skill bypassed original qualification", result, err)
			}
			players, _ = r.loadPlayers()
			p = players[id]
			def := fileSkills[c.skill]
			p.Skills[skillPrerequisiteID(def)] = skillPrerequisiteLevel(def)
			p.Flags["martial_taught_"+c.skill] = true
			_ = r.savePlayers(players)
			if c.chapter == "datang_yunhe" || c.chapter == "dongming_chuanfen" {
				result, err = r.DoRoomAction(id, "connection_skill")
				if err != nil || result["actionStatus"] != "blocked" {
					t.Fatal("missing scene prerequisite accepted", result, err)
				}
				if c.chapter == "datang_yunhe" {
					inst.PuzzleStates["content_trace"] = "known"
				} else {
					inst.PuzzleStates["content_detail"] = "known"
				}
			}
			before, _ := r.loadPlayers()
			result, err = r.DoRoomAction(id, "connection_skill")
			if err != nil || result["actionStatus"] != "success" || inst.PuzzleStates[c.puzzle] != c.state || inst.PuzzleStates["gate"] != "closed" {
				t.Fatal("learned route failed or skipped final decision", result, err)
			}
			after, _ := r.loadPlayers()
			if before[id].Money != after[id].Money || !reflect.DeepEqual(before[id].Backpack, after[id].Backpack) || !reflect.DeepEqual(before[id].Skills, after[id].Skills) || before[id].Attributes.Resources.MP != after[id].Attributes.Resources.MP {
				t.Fatal("exploration changed resources or learning")
			}
			finishConnectedChapter(t, r, id)
			moveDungeonTestPlayer(t, r, id, r.dungeonMgr.InstanceRoomIDNoLock(inst, c.room))
			after, _ = r.loadPlayers()
			delete(after[id].Skills, c.skill)
			_ = r.savePlayers(after)
			result, err = r.DoRoomAction(id, "connection_skill")
			if err != nil || result["actionStatus"] != "blocked" {
				t.Fatal("completed ordinary state leaked an unlearned route", result, err)
			}
			tpl, _ := r.dungeonMgr.DungeonTemplateNoLock(c.chapter)
			moveDungeonTestPlayer(t, r, id, r.dungeonMgr.InstanceRoomIDNoLock(inst, tpl.SpawnRoomID))
			if _, err := r.DoRoomAction(id, "connection_skill"); err == nil {
				t.Fatal("wrong room accepted")
			}
		})
	}
}

// Read the actual dialogue graph, so every configured branch must be callable.
func TestConnectedCityBranchesContinueOldChoices(t *testing.T) {
	dir, _ := gamedata.ResolveResourceDir()
	raw, _ := os.ReadFile(filepath.Join(dir, "dialogues.json"))
	var defs map[string]struct {
		NPCID   string
		Journey *struct {
			DungeonID string
			Branches  []struct{ Flag, NPCID, OptionID, FollowFlag, ActionID, UsedFlag string }
		}
	}
	_ = json.Unmarshal(raw, &defs)
	count := 0
	for _, d := range defs {
		if d.Journey == nil {
			continue
		}
		for _, b := range d.Journey.Branches {
			count++
			t.Run(d.NPCID+"/"+b.OptionID, func(t *testing.T) {
				r, id := v2Player(t)
				players, _ := r.loadPlayers()
				p := players[id]
				for rid, room := range fileRooms {
					for _, nid := range room.NPCIDs {
						if nid == b.NPCID {
							p.RoomID = rid
						}
					}
				}
				_ = r.savePlayers(players)
				if _, _, _, _, err := r.ChooseDialogue(id, b.NPCID, b.OptionID); err == nil {
					t.Fatal("follow-up skipped city choice")
				}
				// This is an old save with an existing conclusion but no new follow-up.
				p.Flags[b.Flag] = true
				_ = r.savePlayers(players)
				for i := 0; i < 4; i++ {
					_, _ = r.Talk(id, b.NPCID)
				}
				players, _ = r.loadPlayers()
				if players[id].Flags[b.FollowFlag] {
					t.Fatal("Talk auto-followed")
				}
				if _, _, _, _, err := r.ChooseDialogue(id, b.NPCID, b.OptionID); err != nil {
					t.Fatal(err)
				}
				players, _ = r.loadPlayers()
				if !players[id].Flags[b.FollowFlag] {
					t.Fatal("follow-up missing")
				}
				// Start a separate instance, carrying only the earned persistent flags.
				r2, id2, inst := createDungeonTestPlayer(t, d.Journey.DungeonID)
				p2s, _ := r2.loadPlayers()
				p2 := p2s[id2]
				for flag, value := range players[id].Flags {
					p2.Flags[flag] = value
				}
				_ = r2.savePlayers(p2s)
				tpl, _ := r2.dungeonMgr.DungeonTemplateNoLock(d.Journey.DungeonID)
				a, ok := tpl.PuzzleActions[b.ActionID]
				if !ok {
					t.Fatal("missing consequence action")
				}
				moveDungeonTestPlayer(t, r2, id2, r2.dungeonMgr.InstanceRoomIDNoLock(inst, a.Rooms[0]))
				for k, v := range a.RequiredPuzzleStates {
					inst.PuzzleStates[k] = v
				}
				// The opposite/unfollowed path cannot reuse the visible action ID.
				p2s, _ = r2.loadPlayers()
				delete(p2s[id2].Flags, b.FollowFlag)
				_ = r2.savePlayers(p2s)
				res, err := r2.DoRoomAction(id2, b.ActionID)
				if err != nil || res["actionStatus"] != "blocked" {
					t.Fatal("unfollowed path accepted", res, err)
				}
				p2s, _ = r2.loadPlayers()
				p2s[id2].Flags[b.FollowFlag] = true
				_ = r2.savePlayers(p2s)
				res, err = r2.DoRoomAction(id2, b.ActionID)
				if err != nil || res["actionStatus"] != "success" || inst.PuzzleStates[a.PuzzleID] != a.NextState {
					t.Fatal(res, err)
				}
				p2s, _ = r2.loadPlayers()
				if !p2s[id2].Flags[b.UsedFlag] {
					t.Fatal("actual use not recorded")
				}
				if row := connectedJourney(t, r2, id2, d.NPCID); row == nil || row["state"] != "已在现场互证" {
					t.Fatal("used path guidance missing", row)
				}
				finishConnectedChapter(t, r2, id2)
				moveDungeonTestPlayer(t, r2, id2, r2.dungeonMgr.InstanceRoomIDNoLock(inst, a.Rooms[0]))
				delete(p2s[id2].Flags, b.FollowFlag)
				_ = r2.savePlayers(p2s)
				res, err = r2.DoRoomAction(id2, b.ActionID)
				if err != nil || res["actionStatus"] != "blocked" {
					t.Fatal("completed state bypassed branch proof", res, err)
				}
			})
		}
	}
	if count != 6 {
		t.Fatal("expected six branch consequences", count)
	}
}

func connectedJourney(t *testing.T, r *RuntimeManager, id, actor string) map[string]string {
	t.Helper()
	rows, err := r.GetJournal(id)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row["id"] == "journey_"+actor {
			return row
		}
	}
	return nil
}

func TestConnectedJournalShowsOnlyKnownNextStep(t *testing.T) {
	r, id := v2Player(t)
	if connectedJourney(t, r, id, "office_clerk") != nil {
		t.Fatal("unseen event leaked")
	}
	players, _ := r.loadPlayers()
	p := players[id]
	p.Flags["discovery_qingyi_spy_specialty"] = true
	_ = r.savePlayers(players)
	row := connectedJourney(t, r, id, "office_clerk")
	if row == nil || !strings.Contains(row["next"], "洛阳府卫") || strings.Contains(row["next"], "隐去受胁者") {
		t.Fatal("missing-witness hint incorrect", row)
	}
	p.Flags["discovery_luoyang_guard_specialty"] = true
	_ = r.savePlayers(players)
	if row = connectedJourney(t, r, id, "office_clerk"); !strings.Contains(row["next"], "染血密信") {
		t.Fatal("missing proof hint", row)
	}
	p.Backpack["sealed_letter"] = 1
	_ = r.savePlayers(players)
	if row = connectedJourney(t, r, id, "office_clerk"); !strings.Contains(row["next"], "韩录事") {
		t.Fatal("case destination missing", row)
	}
	p.Flags["discovery_case_office_clerk_left"] = true
	_ = r.savePlayers(players)
	if row = connectedJourney(t, r, id, "office_clerk"); !strings.Contains(row["next"], "秦三娘") || strings.Contains(row["next"], "原卷") {
		t.Fatal("branch guidance leaks opposite path", row)
	}
	j := fileDialogues[fileNpcs["office_clerk"].DialogueID].Journey
	p.Flags[j.Branches[0].FollowFlag] = true
	_ = r.savePlayers(players)
	if row = connectedJourney(t, r, id, "office_clerk"); row["progress"] != "3/4" || row["state"] != "等待章节解锁" {
		t.Fatal("unavailable chapter advertised ready", row)
	}
	for _, cid := range r.dungeonMgr.DungeonTemplateIDsNoLock() {
		p.Flags["dungeon_cleared_"+cid] = true
	}
	delete(p.Flags, "dungeon_cleared_datang_yunhe")
	_ = r.savePlayers(players)
	if row = connectedJourney(t, r, id, "office_clerk"); row["state"] != "可在现场互证" || !strings.Contains(row["next"], "断缆纤道") {
		t.Fatal("ready scene location missing", row)
	}
	p.Flags["discovery_chapter_datang_yunhe"] = true
	_ = r.savePlayers(players)
	if row = connectedJourney(t, r, id, "office_clerk"); row["state"] != "已自行解开现场" || row["progress"] != "4/4" {
		t.Fatal("normal solution requires unnecessary replay", row)
	}
}
