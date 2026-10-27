package gamedata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentValidationRejectsUnknownQuestGoalAndFreeRepeatableGift(t *testing.T) {
	dir, err := resolveResourceDir()
	if err != nil {
		t.Fatal(err)
	}
	data, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	q := data.Quests["escort_rumor_run"]
	q.Objectives[0].TargetID = "nonexistent_goal"
	data.Quests[q.ID] = q
	if err := validateContentData(dir, data.Rooms, data.NPCs, data.Dialogues, data.Quests); err == nil || !strings.Contains(err.Error(), "no reachable producer") {
		t.Fatal("bad quest goal accepted", err)
	}
	data, _ = Load()
	d := data.Dialogues["escort_intro"]
	d.Options[0].Repeatable = true
	data.Dialogues[d.ID] = d
	if err := validateContentData(dir, data.Rooms, data.NPCs, data.Dialogues, data.Quests); err == nil || !strings.Contains(err.Error(), "repeatable free reward") {
		t.Fatal("free repeatable option accepted", err)
	}
}

func TestContentValidationRejectsUnboundBriefing(t *testing.T) {
	dir, _ := resolveResourceDir()
	data, _ := Load()
	payload, err := os.ReadFile(filepath.Join(dir, "dungeons", "datang_gongbian.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]interface{}
	if err := json.Unmarshal(payload, &d); err != nil {
		t.Fatal(err)
	}
	room := d["rooms"].(map[string]interface{})[d["spawnRoomId"].(string)].(map[string]interface{})
	room["actions"] = []string{}
	temporary := t.TempDir()
	if err := os.Mkdir(filepath.Join(temporary, "dungeons"), 0700); err != nil {
		t.Fatal(err)
	}
	payload, _ = json.Marshal(d)
	if err := os.WriteFile(filepath.Join(temporary, "dungeons", "broken.json"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateContentData(temporary, data.Rooms, data.NPCs, data.Dialogues, data.Quests); err == nil || !strings.Contains(err.Error(), "unreachable dungeon action") {
		t.Fatal("unbound briefing accepted", err)
	}
}
