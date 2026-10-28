package gamedata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestContentDepthPreservesEntityCatalogAndBudgets(t *testing.T) {
	dir, err := ResolveResourceDir()
	if err != nil {
		t.Fatal(err)
	}
	var baseline struct {
		Items, NPCs, Skills, Quests, Factions, Worlds, Rooms, Dungeons []string
		DungeonRooms                                                   map[string][]string
		Rewards, QuickRewards                                          map[string]json.RawMessage
	}
	raw, err := os.ReadFile("testdata/existing_content_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &baseline); err != nil {
		t.Fatal(err)
	}
	keys := func(m map[string]json.RawMessage) []string {
		out := []string{}
		for k := range m {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	for filename, want := range map[string][]string{"items": baseline.Items, "npcs": baseline.NPCs, "skills": baseline.Skills, "quests": baseline.Quests, "factions": baseline.Factions} {
		var catalog map[string]json.RawMessage
		if err := readJSON(filepath.Join(dir, filename+".json"), &catalog); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(keys(catalog), want) {
			t.Fatal("entity IDs changed", filename)
		}
	}
	locations := map[string]bool{}
	worldIDs, roomIDs, dungeonIDs := []string{}, []string{}, []string{}
	files, _ := filepath.Glob(filepath.Join(dir, "worlds", "*.json"))
	for _, path := range files {
		var w struct {
			ID    string
			Rooms map[string]struct {
				NPCs                      []string
				SearchFlag, SearchMessage string
				Actions                   []string
			}
		}
		if err := readJSON(path, &w); err != nil {
			t.Fatal(err)
		}
		worldIDs = append(worldIDs, w.ID)
		for id, room := range w.Rooms {
			roomIDs = append(roomIDs, id)
			for _, nid := range room.NPCs {
				locations[nid] = true
			}
			if room.SearchFlag == "" || room.SearchMessage == "" {
				t.Fatal("public scene lacks investigation", id)
			}
		}
	}
	files, _ = filepath.Glob(filepath.Join(dir, "dungeons", "*.json"))
	gates, epilogues, enriched := 0, 0, 0
	for _, path := range files {
		var d struct {
			ID, ContentStatus     string
			Rooms                 map[string]json.RawMessage
			Rewards, QuickRewards json.RawMessage
			ContentFeature        struct {
				Kind     string
				Solution []struct{ RoomID, ActionID string }
			}
			PuzzleActions map[string]struct{ SpawnNPCs, FailureSpawnNPCs []string }
		}
		if err := readJSON(path, &d); err != nil {
			t.Fatal(err)
		}
		dungeonIDs = append(dungeonIDs, d.ID)
		if !reflect.DeepEqual(keys(d.Rooms), baseline.DungeonRooms[d.ID]) {
			t.Fatal("dungeon rooms changed", d.ID)
		}
		for _, pair := range [][2]json.RawMessage{{d.Rewards, baseline.Rewards[d.ID]}, {d.QuickRewards, baseline.QuickRewards[d.ID]}} {
			var a, b interface{}
			if len(pair[0]) > 0 {
				_ = json.Unmarshal(pair[0], &a)
			} else {
				a = map[string]interface{}{}
			}
			_ = json.Unmarshal(pair[1], &b)
			if !reflect.DeepEqual(a, b) {
				t.Fatal("reward budget changed", d.ID)
			}
		}
		for _, room := range d.Rooms {
			var r struct{ NPCs []string }
			_ = json.Unmarshal(room, &r)
			for _, nid := range r.NPCs {
				locations[nid] = true
			}
		}
		for _, a := range d.PuzzleActions {
			for _, nid := range append(a.SpawnNPCs, a.FailureSpawnNPCs...) {
				locations[nid] = true
			}
		}
		if d.ContentFeature.Kind == "epilogue" {
			epilogues++
		} else if d.ContentFeature.Kind != "" {
			gates++
		} else {
			t.Fatal("missing chapter identity", d.ID)
		}
		if d.ContentStatus == "enriched" {
			enriched++
		}
	}
	if gates != 38 || epilogues != 40 || enriched != 40 {
		t.Fatal("incomplete coverage", gates, epilogues, enriched)
	}
	sort.Strings(worldIDs)
	sort.Strings(roomIDs)
	sort.Strings(dungeonIDs)
	if !reflect.DeepEqual(worldIDs, baseline.Worlds) || !reflect.DeepEqual(roomIDs, baseline.Rooms) || !reflect.DeepEqual(dungeonIDs, baseline.Dungeons) {
		t.Fatal("map/world IDs changed")
	}
	for _, id := range baseline.NPCs {
		if !locations[id] {
			t.Fatal("unplaced NPC", id)
		}
	}
	var items map[string]struct {
		ContentUse struct{ NPCID, ChapterID, AidDungeon, Reading string }
	}
	if err := readJSON(filepath.Join(dir, "items.json"), &items); err != nil {
		t.Fatal(err)
	}
	data, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for id, item := range items {
		u := item.ContentUse
		if !locations[u.NPCID] || u.Reading == "" || u.ChapterID == "" || u.AidDungeon == "" {
			t.Fatal("item lacks usable identity", id)
		}
		found := false
		for _, o := range data.Dialogues[data.NPCs[u.NPCID].DialogueID].Options {
			if o.ID == "content_show_"+id && o.Explicit && o.Requires.PresentItems[id] == 1 {
				found = true
			}
		}
		if !found {
			t.Fatal("missing explicit item presentation", id)
		}
	}
}
