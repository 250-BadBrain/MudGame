package gamedata

import (
	"fmt"
	"mudgame/server/internal/model"
	"path/filepath"
)

func validateContentEvidence(dir string, items map[string]model.ItemDef, skills map[string]model.SkillDef, dialogues map[string]model.DialogueDef) error {
	flags := map[string]bool{}
	for _, d := range dialogues {
		for _, o := range d.Options {
			for flag := range o.SetFlags {
				flags[flag] = true
			}
		}
	}
	type action struct {
		PresentItems  map[string]int
		RequiredFlags map[string]bool
		SetFlags      map[string]bool
	}
	actions := map[string]action{}
	paths, err := filepath.Glob(filepath.Join(dir, "dungeons", "*.json"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		var d struct {
			ID            string
			PuzzleActions map[string]action
		}
		if err := readJSON(path, &d); err != nil {
			return err
		}
		flags["dungeon_cleared_"+d.ID] = true
		for id, a := range d.PuzzleActions {
			actions[d.ID+"/"+id] = a
			for flag := range a.SetFlags {
				flags[flag] = true
			}
		}
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
			for id, n := range o.Requires.Skills {
				if skills[id].ID == "" || n < 1 {
					return fmt.Errorf("invalid content skill requirement: %s/%s", o.ID, id)
				}
			}
		}
	}
	for id, a := range actions {
		if err := validate(id, a.PresentItems, a.RequiredFlags); err != nil {
			return err
		}
	}
	return nil
}
