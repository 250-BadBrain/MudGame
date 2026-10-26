package gamedata

import (
	"mudgame/server/internal/model"
	"testing"
)

func TestMartialCatalogAllowsExpansionBeyondFirst38(t *testing.T) {
	data, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := ResolveResourceDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"new_sword_art", "another_sword_art"} {
		art := data.Skills["thunder_sword"]
		art.ID = id
		art.Name = id
		art.ManualID = "manual_" + id
		art.Requirements = map[string]interface{}{"prerequisite": map[string]interface{}{"skill": "basic_sword", "level": float64(20)}}
		data.Skills[id] = art
		book := data.Items["manual_thunder_sword"]
		book.ID = art.ManualID
		book.Effects = map[string]interface{}{"learnSkill": id}
		data.Items[book.ID] = book
		npc := data.NPCs[art.TeacherID]
		npc.Skills[id] = 1
		data.NPCs[art.TeacherID] = npc
	}

	if len(data.Skills) != 40 {
		t.Fatal("expansion fixture is incomplete")
	}
	if err := validateMartialData(dir, data.Skills, data.Items, data.NPCs); err != nil {
		t.Fatalf("data-driven expansion rejected: %v", err)
	}
}

func TestMartialCatalogRejectsUntypedWeaponsAndOrphanManuals(t *testing.T) {
	for _, bad := range []string{"weapon", "book"} {
		t.Run(bad, func(t *testing.T) {
			data, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			dir, _ := ResolveResourceDir()
			if bad == "weapon" {
				item := data.Items["iron_sword"]
				item.WeaponType = ""
				data.Items[item.ID] = item
			} else {
				data.Items["orphan"] = model.ItemDef{ID: "orphan", Effects: map[string]interface{}{"learnSkill": "absent"}}
			}
			if err := validateMartialData(dir, data.Skills, data.Items, data.NPCs); err == nil {
				t.Fatal("invalid martial resource was accepted")
			}
		})
	}
}
