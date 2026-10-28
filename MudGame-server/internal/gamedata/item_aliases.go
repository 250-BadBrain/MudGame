package gamedata

import (
	"fmt"
	"mudgame/server/internal/model"
	"path/filepath"
)

func loadItemAliases(dir string, items map[string]model.ItemDef) (map[string]string, error) {
	aliases := map[string]string{}
	if err := readJSON(filepath.Join(dir, "item_aliases.json"), &aliases); err != nil {
		return nil, err
	}
	for old, target := range aliases {
		if old == "" || old == target || items[old].ID != "" || items[target].ID == "" || aliases[target] != "" {
			return nil, fmt.Errorf("invalid item alias %s/%s", old, target)
		}
	}
	names := map[string]string{}
	for id, item := range items {
		if item.EquipmentSlot == "" {
			continue
		}
		if prior := names[item.Name]; prior != "" {
			return nil, fmt.Errorf("duplicate equipment name %s: %s/%s", item.Name, prior, id)
		}
		names[item.Name] = id
	}
	return aliases, nil
}
