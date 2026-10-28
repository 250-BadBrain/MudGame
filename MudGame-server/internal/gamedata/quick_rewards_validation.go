package gamedata

import (
	"fmt"
	"mudgame/server/internal/model"
	"os"
	"path/filepath"
	"strings"
)

// A quick reward must be a reviewed, permanent process reward. Never silently
// turn a temporary dungeon key or a paid recipe into a free repeatable gift.
func validateQuickRewards(dir string, items map[string]model.ItemDef) error {
	entries, err := os.ReadDir(filepath.Join(dir, "dungeons"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var d struct {
			ID        string        `json:"id"`
			Quick     model.Rewards `json:"quickRewards"`
			Temporary []string      `json:"temporaryItems"`
			Actions   map[string]struct {
				Items    map[string]int `json:"giveItems"`
				Money    int            `json:"giveMoney"`
				Take     map[string]int `json:"takeItems"`
				Required map[string]int `json:"requiredItems"`
				Choices  []interface{}  `json:"choices"`
			} `json:"puzzleActions"`
		}
		if err := readJSON(filepath.Join(dir, "dungeons", entry.Name()), &d); err != nil {
			return err
		}
		if d.Quick.Money < 0 || d.Quick.Experience != 0 || d.Quick.Potential != 0 || len(d.Quick.Flags) > 0 || len(d.Quick.Reputations) > 0 {
			return fmt.Errorf("invalid quick rewards: %s", d.ID)
		}
		possible := map[string]int{}
		money := 0
		for _, action := range d.Actions {
			if len(action.Take) > 0 || len(action.Required) > 0 || len(action.Choices) > 0 {
				continue
			}
			money += action.Money
			for id, amount := range action.Items {
				possible[id] += amount
			}
		}
		if d.Quick.Money > money {
			return fmt.Errorf("quick money has no process source: %s", d.ID)
		}
		for id, amount := range d.Quick.Items {
			if _, ok := items[id]; !ok || amount < 1 || amount > possible[id] {
				return fmt.Errorf("invalid quick item source: %s/%s", d.ID, id)
			}
			for _, temporary := range d.Temporary {
				if temporary == id {
					return fmt.Errorf("temporary quick reward: %s/%s", d.ID, id)
				}
			}
			if strings.TrimSpace(id) == "" {
				return fmt.Errorf("empty quick reward: %s", d.ID)
			}
		}
	}
	return nil
}
