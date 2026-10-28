package gamedata

import (
	"mudgame/server/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestConversionValueMustStrictlyDecrease(t *testing.T) {
	for _, output := range []int{100, 101} {
		items := map[string]model.ItemDef{"raw": {ID: "raw", Value: 100}, "made": {ID: "made", Value: output}}
		dialogues := map[string]model.DialogueDef{"craft": {ID: "craft", Options: []model.DialogueOption{{ID: "make", TakeItems: map[string]int{"raw": 1}, GiveItems: map[string]int{"made": 1}, TakeMoney: 500}}}}
		if err := validateEconomy(items, dialogues); err == nil {
			t.Fatal("fee concealed material value inversion")
		}
	}
	data, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if err := validateEconomy(data.Items, data.Dialogues); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, item := range data.Items {
		if item.EquipmentSlot != "" {
			if names[item.Name] {
				t.Fatal("duplicate equipment name")
			}
			names[item.Name] = true
		}
	}
	for old, target := range data.ItemAliases {
		if data.Items[old].ID != "" || data.Items[target].ID == "" {
			t.Fatal("bad alias")
		}
	}
}

func TestDungeonConversionsCannotBypassValueRule(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "dungeons"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "dungeons", "craft.json")
	for _, output := range []string{`"giveItems":{"made":1}`, `"giveMoney":100`} {
		payload := `{"id":"craft","puzzleActions":{"make":{"takeItems":{"raw":1},` + output + `}}}`
		if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
			t.Fatal(err)
		}
		if err := validateDungeonEconomy(dir, map[string]model.ItemDef{"raw": {ID: "raw", Value: 100}, "made": {ID: "made", Value: 100}}); err == nil {
			t.Fatal("dungeon conversion bypassed material value rule")
		}
	}
}
