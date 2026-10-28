package gamedata

import (
	"fmt"
	"mudgame/server/internal/model"
	"os"
	"path/filepath"
)

func validateEconomy(items map[string]model.ItemDef, dialogues map[string]model.DialogueDef) error {
	value := func(quantities map[string]int) (int, error) {
		sum := 0
		for id, n := range quantities {
			item, ok := items[id]
			if !ok || n < 1 {
				return 0, fmt.Errorf("invalid conversion item %s", id)
			}
			v, err := model.CheckedProduct(item.Value, n)
			if err != nil {
				return 0, err
			}
			sum, err = model.CheckedAdd(sum, v)
			if err != nil {
				return 0, err
			}
		}
		return sum, nil
	}
	for id, item := range items {
		if item.Value < 0 {
			return fmt.Errorf("negative value: %s", id)
		}
	}
	for _, d := range dialogues {
		for _, o := range d.Options {
			if o.StartQuestID != "" || o.CompleteQuestID != "" {
				continue
			}
			if len(o.TakeItems) == 0 && o.TakeMoney == 0 {
				continue
			}
			if len(o.GiveItems) == 0 && o.GiveMoney == 0 {
				continue
			}
			in, err := value(o.TakeItems)
			if err != nil {
				return err
			}
			out, err := value(o.GiveItems)
			if err != nil {
				return err
			}
			out, err = model.CheckedAdd(out, o.GiveMoney)
			if err != nil {
				return err
			}
			if (len(o.TakeItems) > 0 && out >= in) || (len(o.TakeItems) == 0 && out > o.TakeMoney) {
				return fmt.Errorf("conversion value must decrease: %s/%s (%d -> %d)", d.ID, o.ID, in, out)
			}
		}
	}
	return nil
}

func validateDungeonEconomy(dir string, items map[string]model.ItemDef) error {
	entries, err := os.ReadDir(filepath.Join(dir, "dungeons"))
	if err != nil {
		return err
	}
	conversions := map[string]model.DialogueDef{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var dungeon struct {
			ID      string `json:"id"`
			Actions map[string]struct {
				TakeItems map[string]int `json:"takeItems"`
				GiveItems map[string]int `json:"giveItems"`
				GiveMoney int            `json:"giveMoney"`
			} `json:"puzzleActions"`
		}
		if err := readJSON(filepath.Join(dir, "dungeons", entry.Name()), &dungeon); err != nil {
			return err
		}
		dialogue := model.DialogueDef{ID: dungeon.ID}
		for id, action := range dungeon.Actions {
			dialogue.Options = append(dialogue.Options, model.DialogueOption{ID: id, TakeItems: action.TakeItems, GiveItems: action.GiveItems, GiveMoney: action.GiveMoney})
		}
		conversions[dungeon.ID] = dialogue
	}
	return validateEconomy(items, conversions)
}
