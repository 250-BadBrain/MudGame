package gamedata

import (
	"mudgame/server/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestEquipmentRequiresKnownEnhancementCoefficient(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "item_aliases.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	items := map[string]model.ItemDef{"unknown": {ID: "unknown", Name: "异物", EquipmentSlot: "RING", Effects: map[string]interface{}{"mysteryStat": 1}}}
	if _, err := loadItemAliases(dir, items); err == nil {
		t.Fatal("unknown equipment effect silently ignored")
	}
}
