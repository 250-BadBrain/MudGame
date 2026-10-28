package gamedata

import (
	"encoding/json"
	"mudgame/server/internal/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConnectedEvidenceRejectsUnknownOrInvalidSkills(t *testing.T) {
	for _, required := range []map[string]int{{"missing": 1}, {"known": 0}, {"known": -1}} {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "dungeons"), 0755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(map[string]interface{}{"id": "test", "puzzleActions": map[string]interface{}{"observe": map[string]interface{}{"requiredSkills": required}}})
		if err := os.WriteFile(filepath.Join(dir, "dungeons", "test.json"), raw, 0644); err != nil {
			t.Fatal(err)
		}
		err := validateContentEvidence(dir, nil, map[string]model.SkillDef{"known": {ID: "known"}}, nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "invalid content skill") {
			t.Fatal("invalid exploration requirement accepted", required, err)
		}
	}
}

func TestConnectedJourneysHaveRealDistinctFollowups(t *testing.T) {
	data, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := ResolveResourceDir()
	count := 0
	for id, d := range data.Dialogues {
		if d.Journey == nil {
			continue
		}
		count++
		// Mutating a returned graph must not poison the process-wide cache.
		d.Journey.Branches[0].OptionID = "nonexistent_followup"
		data.Dialogues[id] = d
		if err := validateContentEvidence(dir, data.Items, data.Skills, data.Dialogues, data.NPCs, data.Rooms); err == nil {
			t.Fatal("broken follow-up accepted")
		}
		fresh, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if fresh.Dialogues[id].Journey.Branches[0].OptionID == "nonexistent_followup" {
			t.Fatal("journey clone shares mutable branches")
		}
		data = fresh
	}
	if count != 3 {
		t.Fatal("expected three connected city cases", count)
	}
}
