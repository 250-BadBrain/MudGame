package gamedata

import (
	"encoding/json"
	"fmt"
	"mudgame/server/internal/model"
	"path/filepath"
)

func validateContentEvidence(dir string, items map[string]model.ItemDef, skills map[string]model.SkillDef, dialogues map[string]model.DialogueDef, npcs map[string]model.NPC, rooms map[string]model.Room) error {
	flags := map[string]bool{}
	for _, d := range dialogues {
		for _, o := range d.Options {
			for flag := range o.SetFlags {
				flags[flag] = true
			}
		}
	}
	type action struct {
		Rooms          []string
		RequiredSkills map[string]int
		PresentItems   map[string]int
		RequiredFlags  map[string]bool
		SetFlags       map[string]bool
	}
	actions := map[string]action{}
	publicNPCs := map[string]bool{}
	for _, room := range rooms {
		for _, id := range room.NPCIDs {
			publicNPCs[id] = true
		}
	}
	dungeonRooms := map[string]map[string]json.RawMessage{}
	paths, err := filepath.Glob(filepath.Join(dir, "dungeons", "*.json"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		var d struct {
			ID            string
			Rooms         map[string]json.RawMessage
			PuzzleActions map[string]action
		}
		if err := readJSON(path, &d); err != nil {
			return err
		}
		flags["dungeon_cleared_"+d.ID] = true
		dungeonRooms[d.ID] = d.Rooms
		for id, a := range d.PuzzleActions {
			actions[d.ID+"/"+id] = a
			for flag := range a.SetFlags {
				flags[flag] = true
			}
		}
	}
	validateSkills := func(label string, required map[string]int) error {
		for id, n := range required {
			if skills[id].ID == "" || n < 1 {
				return fmt.Errorf("invalid content skill requirement: %s/%s", label, id)
			}
		}
		return nil
	}
	validate := func(label string, proof map[string]int, required map[string]bool) error {
		for id, n := range proof {
			if items[id].ID == "" || n < 1 {
				return fmt.Errorf("invalid content proof: %s/%s", label, id)
			}
		}
		for flag := range required {
			if !flags[flag] {
				return fmt.Errorf("content evidence has no producer: %s/%s", label, flag)
			}
		}
		return nil
	}
	for _, d := range dialogues {
		for _, o := range d.Options {
			if err := validate(d.ID+"/"+o.ID, o.Requires.PresentItems, o.Requires.Flags); err != nil {
				return err
			}
			if err := validateSkills(o.ID, o.Requires.Skills); err != nil {
				return err
			}
		}
		if j := d.Journey; j != nil {
			if j.Title == "" || len(dungeonRooms[j.DungeonID]) == 0 || !flags[j.StartFlag] || items[j.ItemID].ID == "" || len(j.Witnesses) < 2 || len(j.Branches) != 2 {
				return fmt.Errorf("invalid journey: %s", d.ID)
			}
			seen := map[string]bool{}
			for _, id := range j.Witnesses {
				if !publicNPCs[id] || seen[id] || !flags["discovery_"+id+"_specialty"] {
					return fmt.Errorf("invalid journey witness: %s/%s", d.ID, id)
				}
				seen[id] = true
			}
			seen = map[string]bool{}
			for _, b := range j.Branches {
				a, exists := actions[j.DungeonID+"/"+b.ActionID]
				if !flags[b.Flag] || seen[b.Flag] || !publicNPCs[b.NPCID] || !exists || len(a.Rooms) == 0 || !a.RequiredFlags[b.Flag] || !a.RequiredFlags[b.FollowFlag] || !a.SetFlags[b.UsedFlag] || b.FollowFlag == "" || b.UsedFlag == "" {
					return fmt.Errorf("invalid journey branch: %s/%s", d.ID, b.ActionID)
				}
				seen[b.Flag] = true
				for _, rid := range a.Rooms {
					if dungeonRooms[j.DungeonID][rid] == nil {
						return fmt.Errorf("invalid journey room: %s", rid)
					}
				}
				follow := false
				for _, o := range dialogues[npcs[b.NPCID].DialogueID].Options {
					if o.ID == b.OptionID && o.Explicit && !o.Repeatable && o.Requires.Flags[b.Flag] && o.SetFlags[b.FollowFlag] {
						follow = true
					}
				}
				if !follow {
					return fmt.Errorf("invalid journey follow-up: %s/%s", d.ID, b.OptionID)
				}
			}
		}
	}
	for id, a := range actions {
		if err := validateSkills(id, a.RequiredSkills); err != nil {
			return err
		}
		if err := validate(id, a.PresentItems, a.RequiredFlags); err != nil {
			return err
		}
	}
	return nil
}
