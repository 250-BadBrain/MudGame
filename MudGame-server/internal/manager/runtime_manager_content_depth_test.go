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

type depthChapter struct {
	ID             string
	ContentFeature struct {
		Kind     string
		Solution []struct {
			RoomID   string `json:"roomId"`
			ActionID string `json:"actionId"`
		}
		Mistake string
	}
}

func TestEveryExpandedChapterSolvesAndRejectsForgedBranches(t *testing.T) {
	dir, _ := gamedata.ResolveResourceDir()
	paths, _ := filepath.Glob(filepath.Join(dir, "dungeons", "*.json"))
	for _, path := range paths {
		raw, _ := os.ReadFile(path)
		var d depthChapter
		_ = json.Unmarshal(raw, &d)
		t.Run(d.ID, func(t *testing.T) {
			r, id, inst := createDungeonTestPlayer(t, d.ID)
			// Defeat existing actors through the objective API; preserve protected witnesses.
			for nid, npc := range inst.NPCs {
				if npc.CombatMode != "protected" {
					_, _, _, _ = r.dungeonMgr.DefeatNPCNoLock(nid)
				}
			}
			if d.ContentFeature.Kind != "epilogue" {
				gateRoom := d.ContentFeature.Solution[len(d.ContentFeature.Solution)-1].RoomID
				moveDungeonTestPlayer(t, r, id, r.dungeonMgr.InstanceRoomIDNoLock(inst, gateRoom))
				if _, err := r.DoRoomAction(id, "content_decide_good"); err == nil {
					t.Fatal("forged branch accepted")
				}
				result, err := r.DoRoomAction(id, "open_gate")
				if err != nil || result["actionStatus"] != "blocked" {
					t.Fatal("opened without observations", result, err)
				}
			}
			for _, step := range d.ContentFeature.Solution {
				moveDungeonTestPlayer(t, r, id, r.dungeonMgr.InstanceRoomIDNoLock(inst, step.RoomID))
				if step.ActionID == "content_decide_good" {
					result, err := r.DoRoomAction(id, d.ContentFeature.Mistake)
					if err != nil || result["puzzleState"] != "closed" {
						t.Fatal("wrong choice did not recover safely", result, err)
					}
					if _, err = r.DoRoomAction(id, "open_gate"); err != nil {
						t.Fatal(err)
					}
				}
				result, err := r.DoRoomAction(id, step.ActionID)
				if err != nil || result["actionStatus"] != "success" {
					t.Fatalf("%s failed: %+v %v", step.ActionID, result, err)
				}
			}
			if d.ContentFeature.Kind != "epilogue" && inst.PuzzleStates["gate"] != "open" {
				t.Fatal("solution did not open gate")
			}
			journal, err := r.GetJournal(id)
			if err != nil || len(journal) == 0 {
				t.Fatal("chapter conclusion not recorded", err)
			}
		})
	}
}

func TestEvidenceRequiresRealPossessionAndExplicitChoice(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.RoomID = "ly_east_market"
	p.Backpack["iron_sword"] = 1
	_ = r.savePlayers(players)
	actor := ""
	for _, d := range fileDialogues {
		for _, o := range d.Options {
			if o.ID == "content_show_iron_sword" {
				actor = d.NPCID
			}
		}
	}
	// Move to the actual existing location of this item's designated specialist.
	for rid, room := range fileRooms {
		for _, nid := range room.NPCIDs {
			if nid == actor {
				p.RoomID = rid
			}
		}
	}
	_ = r.savePlayers(players)
	for i := 0; i < 6; i++ {
		if _, err := r.Talk(id, actor); err != nil {
			t.Fatal(err)
		}
	}
	players, _ = r.loadPlayers()
	if players[id].Flags["discovery_item_iron_sword"] {
		t.Fatal("Talk automatically chose evidence")
	}
	if err := r.EquipItem(id, "iron_sword"); err != nil {
		t.Fatal(err)
	}
	before, _ := r.loadPlayers()
	if _, _, _, _, err := r.ChooseDialogue(id, actor, "content_show_iron_sword"); err != nil {
		t.Fatal("equipped evidence refused", err)
	}
	after, _ := r.loadPlayers()
	if !reflect.DeepEqual(before[id].Backpack, after[id].Backpack) || !reflect.DeepEqual(before[id].Equipment, after[id].Equipment) {
		t.Fatal("presentation consumed gear")
	}
	if _, _, _, _, err := r.ChooseDialogue(id, actor, "content_show_iron_sword"); err == nil {
		t.Fatal("one-time presentation replay accepted")
	}
	journal, _ := r.GetJournal(id)
	seen := false
	for _, entry := range journal {
		if entry["id"] == "discovery_item_iron_sword" {
			seen = true
		}
		if strings.HasPrefix(entry["id"], "discovery_item_") && entry["id"] != "discovery_item_iron_sword" {
			t.Fatal("unseen item leaked into journal")
		}
	}
	if !seen {
		t.Fatal("evidence absent from journal")
	}
	players, _ = r.loadPlayers()
	delete(players[id].Equipment, "WEAPON")
	delete(players[id].DialogueHistory, actor+"|content_show_iron_sword")
	_ = r.savePlayers(players)
	if _, _, _, _, err := r.ChooseDialogue(id, actor, "content_show_iron_sword"); err == nil {
		t.Fatal("unowned evidence accepted")
	}
}

func TestConditionalAidHiddenAndStillValidatedByServer(t *testing.T) {
	r, id, inst := createDungeonTestPlayer(t, "datang_yunhe")
	room := r.dungeonMgr.InstanceRoomIDNoLock(inst, "dat_01")
	moveDungeonTestPlayer(t, r, id, room)
	payload, _, err := r.GetRoomSnapshot(id)
	if err != nil {
		t.Fatal(err)
	}
	for _, aid := range payload["actions"].([]string) {
		if strings.HasPrefix(aid, "content_aid_") {
			t.Fatal("unavailable item aid cluttered room", aid)
		}
	}
	tpl, _ := r.dungeonMgr.DungeonTemplateNoLock(inst.DungeonID)
	for aid := range tpl.PuzzleActions {
		if strings.HasPrefix(aid, "content_aid_") {
			result, err := r.DoRoomAction(id, aid)
			if err != nil || result["actionStatus"] != "blocked" {
				t.Fatal("unowned aid accepted", aid, result, err)
			}
			return
		}
	}
	t.Fatal("no item aid to validate")
}

func TestCityCaseNeedsDistinctWitnessesAndLocksOneConclusion(t *testing.T) {
	r, id := v2Player(t)
	players, _ := r.loadPlayers()
	p := players[id]
	p.RoomID = "ly_north_office"
	p.Backpack["sealed_letter"] = 1
	_ = r.savePlayers(players)
	if _, _, _, _, err := r.ChooseDialogue(id, "office_clerk", "content_case_left"); err == nil {
		t.Fatal("case skipped witnesses")
	}
	p.Flags["discovery_qingyi_spy_specialty"] = true
	p.Flags["discovery_luoyang_guard_specialty"] = true
	_ = r.savePlayers(players)
	if _, _, _, _, err := r.ChooseDialogue(id, "office_clerk", "content_case_left"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := r.ChooseDialogue(id, "office_clerk", "content_case_right"); err == nil {
		t.Fatal("contradictory conclusion accepted")
	}
	players, _ = r.loadPlayers()
	p = players[id]
	if p.Backpack["sealed_letter"] != 1 {
		t.Fatal("case consumed original proof")
	}
	p.RoomID = "ly_refugee_camp"
	_ = r.savePlayers(players)
	if _, _, _, _, err := r.ChooseDialogue(id, "old_refugee", "content_after_office_clerk_left"); err != nil {
		t.Fatal("aftermath inaccessible", err)
	}
	entries, _ := r.GetJournal(id)
	has := false
	for _, e := range entries {
		has = has || e["id"] == "discovery_aftermath_office_clerk_left"
	}
	if !has {
		t.Fatal("aftermath missing from journal")
	}
}

func TestAllItemInterpretationsAreActuallyCallable(t *testing.T) {
	r, id := v2Player(t)
	dir, _ := gamedata.ResolveResourceDir()
	raw, _ := os.ReadFile(filepath.Join(dir, "items.json"))
	var items map[string]struct{ ContentUse struct{ NPCID string } }
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatal(err)
	}
	players, _ := r.loadPlayers()
	p := players[id]
	for iid := range items {
		p.Backpack[iid] = 1
	}
	for iid, item := range items {
		actor := item.ContentUse.NPCID
		found := false
		for rid, room := range fileRooms {
			for _, npc := range room.NPCIDs {
				if npc == actor {
					p.RoomID = rid
					found = true
				}
			}
		}
		if !found {
			t.Fatal("item specialist not publicly reachable", iid, actor)
		}
		_ = r.savePlayers(players)
		before, _ := r.loadPlayers()
		if _, _, _, _, err := r.ChooseDialogue(id, actor, "content_show_"+iid); err != nil {
			t.Fatalf("%s/%s: %v", iid, actor, err)
		}
		after, _ := r.loadPlayers()
		if !reflect.DeepEqual(before[id].Backpack, after[id].Backpack) || before[id].Money != after[id].Money {
			t.Fatal("interpretation changed item/currency ownership", iid)
		}
		players = after
		p = players[id]
	}
}

func TestAllEightCityCasesHaveDifferentWitnessGates(t *testing.T) {
	count := 0
	for _, dialogue := range fileDialogues {
		for _, option := range dialogue.Options {
			if option.ID != "content_case_left" {
				continue
			}
			count++
			t.Run(dialogue.NPCID, func(t *testing.T) {
				r, id := v2Player(t)
				players, _ := r.loadPlayers()
				p := players[id]
				for rid, room := range fileRooms {
					for _, npc := range room.NPCIDs {
						if npc == dialogue.NPCID {
							p.RoomID = rid
						}
					}
				}
				for iid, n := range option.Requires.PresentItems {
					p.Backpack[iid] = n
				}
				_ = r.savePlayers(players)
				if _, _, _, _, err := r.ChooseDialogue(id, dialogue.NPCID, option.ID); err == nil {
					t.Fatal("missing witness requirements accepted")
				}
				for flag, value := range option.Requires.Flags {
					if value {
						p.Flags[flag] = true
					}
				}
				_ = r.savePlayers(players)
				if _, _, _, _, err := r.ChooseDialogue(id, dialogue.NPCID, option.ID); err != nil {
					t.Fatal(err)
				}
				if _, _, _, _, err := r.ChooseDialogue(id, dialogue.NPCID, "content_case_right"); err == nil {
					t.Fatal("mutually exclusive outcome accepted")
				}
			})
		}
	}
	if count != 8 {
		t.Fatal("city coverage incomplete", count)
	}
}
