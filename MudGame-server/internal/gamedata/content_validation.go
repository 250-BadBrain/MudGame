package gamedata

import (
	"fmt"
	"mudgame/server/internal/model"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type contentAction struct {
	Rooms         []string        `json:"rooms"`
	DialogueNPCID string          `json:"dialogueNpcId"`
	SetFlags      map[string]bool `json:"setFlags"`
	Choices       []struct {
		ID string `json:"id"`
	} `json:"choices"`
}
type contentDungeon struct {
	ID          string `json:"id"`
	SpawnRoomID string `json:"spawnRoomId"`
	Rooms       map[string]struct {
		NPCs    []string          `json:"npcs"`
		Actions []string          `json:"actions"`
		Exits   map[string]string `json:"exits"`
	} `json:"rooms"`
	PuzzleActions map[string]contentAction `json:"puzzleActions"`
	Rewards       model.Rewards            `json:"rewards"`
}

// Content must have actual UI/runtime entry points, not merely valid dictionary keys.
func validateContentData(dir string, rooms map[string]model.Room, npcs map[string]model.NPC, dialogues map[string]model.DialogueDef, quests map[string]model.QuestDef) error {
	publicNPCs := map[string]bool{}
	flags := map[string]bool{}
	dungeons := map[string]bool{}
	for _, room := range rooms {
		for _, nid := range room.NPCIDs {
			if !npcs[nid].InstanceOnly {
				publicNPCs[nid] = true
			}
		}
		if room.SearchFlag != "" {
			if room.SearchMessage == "" || !slices.Contains(room.Actions, "search") {
				return fmt.Errorf("search without entry/text: %s", room.ID)
			}
			flags[room.SearchFlag] = true
		}
		if len(room.Rumors) > 0 && !slices.Contains(room.Actions, "listen_rumor") {
			return fmt.Errorf("room rumor without entry: %s", room.ID)
		}
	}
	for nid := range publicNPCs {
		for _, opt := range dialogues[npcs[nid].DialogueID].Options {
			for flag, value := range opt.SetFlags {
				if value {
					flags[flag] = true
				}
			}
		}
	}
	entries, err := os.ReadDir(filepath.Join(dir, "dungeons"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var d contentDungeon
		if err := readJSON(filepath.Join(dir, "dungeons", entry.Name()), &d); err != nil {
			return err
		}
		dungeons[d.ID] = true
		flags["dungeon_cleared_"+d.ID] = true
		for flag, value := range d.Rewards.Flags {
			if value {
				flags[flag] = true
			}
		}
		reachableRooms := map[string]bool{d.SpawnRoomID: true}
		queue := []string{d.SpawnRoomID}
		for len(queue) > 0 {
			rid := queue[0]
			queue = queue[1:]
			for _, to := range d.Rooms[rid].Exits {
				if !reachableRooms[to] {
					reachableRooms[to] = true
					queue = append(queue, to)
				}
			}
		}
		reachableActions := map[string]bool{}
		for rid, room := range d.Rooms {
			if !reachableRooms[rid] {
				continue
			}
			for _, aid := range room.Actions {
				if a, ok := d.PuzzleActions[aid]; ok && (len(a.Rooms) == 0 || slices.Contains(a.Rooms, rid)) {
					reachableActions[aid] = true
				}
			}
			for aid, a := range d.PuzzleActions {
				if a.DialogueNPCID != "" && slices.Contains(a.Rooms, rid) && slices.Contains(room.NPCs, a.DialogueNPCID) && npcs[a.DialogueNPCID].DialogueID != "" {
					reachableActions[aid] = true
				}
			}
		}
		for changed := true; changed; {
			changed = false
			for aid := range reachableActions {
				for _, choice := range d.PuzzleActions[aid].Choices {
					if !reachableActions[choice.ID] {
						if _, ok := d.PuzzleActions[choice.ID]; !ok {
							return fmt.Errorf("unknown branch: %s/%s", d.ID, choice.ID)
						}
						reachableActions[choice.ID] = true
						changed = true
					}
				}
			}
		}
		for aid, a := range d.PuzzleActions {
			if !reachableActions[aid] {
				return fmt.Errorf("unreachable dungeon action: %s/%s", d.ID, aid)
			}
			for flag, value := range a.SetFlags {
				if value {
					flags[flag] = true
				}
			}
		}
	}
	for did, d := range dialogues {
		for _, opt := range d.Options {
			if opt.Repeatable && opt.TakeMoney <= 0 && len(opt.TakeItems) == 0 {
				return fmt.Errorf("repeatable free reward: %s/%s", did, opt.ID)
			}
			if opt.TakeMoney < 0 || opt.GiveMoney < 0 {
				return fmt.Errorf("negative dialogue money: %s/%s", did, opt.ID)
			}
			for _, values := range []map[string]int{opt.TakeItems, opt.GiveItems} {
				for _, n := range values {
					if n <= 0 {
						return fmt.Errorf("invalid dialogue amount: %s/%s", did, opt.ID)
					}
				}
			}
		}
	}
	names := map[string]bool{}
	for id, q := range quests {
		if q.AliasOf != "" {
			target, ok := quests[q.AliasOf]
			if !ok || target.AliasOf != "" || q.AliasOf == id {
				return fmt.Errorf("invalid quest alias: %s", id)
			}
			continue
		}
		if names[q.Name] {
			return fmt.Errorf("duplicate quest name: %s", q.Name)
		}
		names[q.Name] = true
		if !publicNPCs[q.GiverNPCID] || !publicNPCs[q.TurnInNPCID] {
			return fmt.Errorf("quest NPC has no public location: %s", id)
		}
		if q.SourceDungeon != "" && !dungeons[q.SourceDungeon] {
			return fmt.Errorf("missing quest source: %s", id)
		}
		hasStart, hasComplete := false, false
		for _, opt := range dialogues[npcs[q.GiverNPCID].DialogueID].Options {
			hasStart = hasStart || opt.StartQuestID == id
		}
		for _, opt := range dialogues[npcs[q.TurnInNPCID].DialogueID].Options {
			hasComplete = hasComplete || opt.CompleteQuestID == id
		}
		if !hasStart || !hasComplete {
			return fmt.Errorf("quest missing start/turn-in: %s", id)
		}
		if len(q.Objectives) == 0 {
			return fmt.Errorf("quest without objectives: %s", id)
		}
		for _, o := range q.Objectives {
			if o.Type != "flag" || !flags[o.TargetID] || o.Required != 1 {
				return fmt.Errorf("quest objective has no reachable producer: %s/%s", id, o.TargetID)
			}
			if strings.HasPrefix(o.TargetID, "dungeon_cleared_") && !dungeons[strings.TrimPrefix(o.TargetID, "dungeon_cleared_")] {
				return fmt.Errorf("missing quest dungeon: %s", id)
			}
		}
	}
	return nil
}
