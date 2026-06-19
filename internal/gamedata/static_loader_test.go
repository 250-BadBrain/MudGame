package gamedata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticResourcesAreConsistent(t *testing.T) {
	data, err := Load()
	if err != nil {
		t.Fatalf("load static data failed: %v", err)
	}
	if len(data.WorldOrder) == 0 {
		t.Fatalf("world order should not be empty")
	}
	for _, world := range data.Worlds {
		if strings.TrimSpace(world.ID) == "" {
			t.Fatalf("world id should not be empty")
		}
		if world.SpawnRoomID == "" {
			t.Fatalf("world %s missing spawn room", world.ID)
		}
		spawn, ok := data.Rooms[world.SpawnRoomID]
		if !ok {
			t.Fatalf("world %s spawn room %s does not exist", world.ID, world.SpawnRoomID)
		}
		if spawn.WorldID != world.ID {
			t.Fatalf("world %s spawn room %s belongs to %s", world.ID, world.SpawnRoomID, spawn.WorldID)
		}
	}
	for roomID, room := range data.Rooms {
		if strings.TrimSpace(roomID) == "" || strings.TrimSpace(room.ID) == "" {
			t.Fatalf("room id should not be empty")
		}
		for dir, ex := range room.Exits {
			if strings.TrimSpace(dir) == "" {
				t.Fatalf("room %s has empty exit direction", roomID)
			}
			target, ok := data.Rooms[ex.TargetRoomID]
			if !ok {
				t.Fatalf("room %s exit %s points to missing room %s", roomID, dir, ex.TargetRoomID)
			}
			if target.WorldID != room.WorldID {
				t.Fatalf("room %s exit %s crosses world boundary to %s", roomID, dir, ex.TargetRoomID)
			}
		}
		for _, npcID := range room.NPCIDs {
			if _, ok := data.NPCs[npcID]; !ok {
				t.Fatalf("room %s references missing npc %s", roomID, npcID)
			}
		}
	}
	for npcID, npc := range data.NPCs {
		if npc.Faction != "" {
			if _, ok := data.Factions[npc.Faction]; !ok {
				t.Fatalf("npc %s references missing faction %s", npcID, npc.Faction)
			}
		}
		if npc.DialogueID != "" {
			if _, ok := data.Dialogues[npc.DialogueID]; !ok {
				t.Fatalf("npc %s references missing dialogue %s", npcID, npc.DialogueID)
			}
		}
		for _, questID := range npc.QuestIDs {
			if _, ok := data.Quests[questID]; !ok {
				t.Fatalf("npc %s references missing quest %s", npcID, questID)
			}
		}
		for itemID := range npc.ShopInventory {
			if _, ok := data.Items[itemID]; !ok {
				t.Fatalf("npc %s shop references missing item %s", npcID, itemID)
			}
		}
		for skillID := range npc.Skills {
			if _, ok := data.Skills[skillID]; !ok {
				t.Fatalf("npc %s teaches missing skill %s", npcID, skillID)
			}
		}
	}
	validSlots := map[string]bool{
		"": true, "WEAPON": true, "HEAD": true, "BODY": true, "FEET": true,
		"NECKLACE": true, "RING": true, "ACCESSORY": true, "HIDDEN_WEAPON": true,
	}
	for itemID, item := range data.Items {
		if strings.TrimSpace(item.Name) == "" {
			t.Fatalf("item %s missing name", itemID)
		}
		if !validSlots[item.EquipmentSlot] {
			t.Fatalf("item %s has invalid equipment slot %s", itemID, item.EquipmentSlot)
		}
	}
	for key, skill := range data.Skills {
		if skill.ID != key {
			t.Fatalf("skill key %s does not match id %s", key, skill.ID)
		}
		if strings.TrimSpace(skill.Name) == "" {
			t.Fatalf("skill %s missing name", key)
		}
	}
	for dialogueID, dialogue := range data.Dialogues {
		if dialogue.NPCID != "" {
			if _, ok := data.NPCs[dialogue.NPCID]; !ok {
				t.Fatalf("dialogue %s references missing npc %s", dialogueID, dialogue.NPCID)
			}
		}
		for _, option := range dialogue.Options {
			if option.Requires.ReputationID != "" {
				if _, ok := data.Factions[option.Requires.ReputationID]; !ok {
					t.Fatalf("dialogue %s option %s requires missing reputation %s", dialogueID, option.ID, option.Requires.ReputationID)
				}
			}
			if option.StartQuestID != "" {
				if _, ok := data.Quests[option.StartQuestID]; !ok {
					t.Fatalf("dialogue %s option %s starts missing quest %s", dialogueID, option.ID, option.StartQuestID)
				}
			}
			if option.CompleteQuestID != "" {
				if _, ok := data.Quests[option.CompleteQuestID]; !ok {
					t.Fatalf("dialogue %s option %s completes missing quest %s", dialogueID, option.ID, option.CompleteQuestID)
				}
			}
		}
	}
	for questID, quest := range data.Quests {
		if quest.GiverNPCID != "" {
			if _, ok := data.NPCs[quest.GiverNPCID]; !ok {
				t.Fatalf("quest %s references missing giver npc %s", questID, quest.GiverNPCID)
			}
		}
		if quest.TurnInNPCID != "" {
			if _, ok := data.NPCs[quest.TurnInNPCID]; !ok {
				t.Fatalf("quest %s references missing turn-in npc %s", questID, quest.TurnInNPCID)
			}
		}
		for itemID := range quest.Rewards.Items {
			if _, ok := data.Items[itemID]; !ok {
				t.Fatalf("quest %s rewards missing item %s", questID, itemID)
			}
		}
		for factionID := range quest.Rewards.Reputations {
			if _, ok := data.Factions[factionID]; !ok {
				t.Fatalf("quest %s rewards missing faction reputation %s", questID, factionID)
			}
		}
	}
	validateDungeonResources(t, data)
}

func validateDungeonResources(t *testing.T, data StaticData) {
	t.Helper()
	resourceDir, err := ResolveResourceDir()
	if err != nil {
		t.Fatalf("resolve resource dir failed: %v", err)
	}
	dungeonDir := filepath.Join(resourceDir, "dungeons")
	entries, err := os.ReadDir(dungeonDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read dungeon dir failed: %v", err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		path := filepath.Join(dungeonDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read dungeon %s failed: %v", entry.Name(), err)
		}
		var dungeon struct {
			ID          string `json:"id"`
			SpawnRoomID string `json:"spawnRoomId"`
			Rooms       map[string]struct {
				Name  string            `json:"name"`
				Exits map[string]string `json:"exits"`
				NPCs  []string          `json:"npcs"`
			} `json:"rooms"`
			ProgressOnKill map[string]int `json:"progressOnKill"`
			Objectives     []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				TargetID string `json:"targetId"`
				Required int    `json:"required"`
			} `json:"objectives"`
			Rewards struct {
				Items map[string]int `json:"items"`
			} `json:"rewards"`
		}
		if err := json.Unmarshal(raw, &dungeon); err != nil {
			t.Fatalf("parse dungeon %s failed: %v", entry.Name(), err)
		}
		if dungeon.ID == "" {
			t.Fatalf("dungeon %s missing id", entry.Name())
		}
		if seen[dungeon.ID] {
			t.Fatalf("duplicate dungeon id %s", dungeon.ID)
		}
		seen[dungeon.ID] = true
		if len(dungeon.Rooms) == 0 {
			t.Fatalf("dungeon %s has no rooms", dungeon.ID)
		}
		if dungeon.SpawnRoomID == "" {
			t.Fatalf("dungeon %s missing spawnRoomId", dungeon.ID)
		}
		if _, ok := dungeon.Rooms[dungeon.SpawnRoomID]; !ok {
			t.Fatalf("dungeon %s spawn room %s does not exist", dungeon.ID, dungeon.SpawnRoomID)
		}
		dungeonNPCs := map[string]bool{}
		for roomID, room := range dungeon.Rooms {
			if strings.TrimSpace(roomID) == "" || strings.TrimSpace(room.Name) == "" {
				t.Fatalf("dungeon %s has room with empty id or name", dungeon.ID)
			}
			for dir, targetID := range room.Exits {
				if strings.TrimSpace(dir) == "" {
					t.Fatalf("dungeon %s room %s has empty exit direction", dungeon.ID, roomID)
				}
				if _, ok := dungeon.Rooms[targetID]; !ok {
					t.Fatalf("dungeon %s room %s exit %s points to missing room %s", dungeon.ID, roomID, dir, targetID)
				}
			}
			for _, npcID := range room.NPCs {
				if _, ok := data.NPCs[npcID]; !ok {
					t.Fatalf("dungeon %s room %s references missing npc %s", dungeon.ID, roomID, npcID)
				}
				dungeonNPCs[npcID] = true
			}
		}
		for npcID, progress := range dungeon.ProgressOnKill {
			if progress <= 0 {
				t.Fatalf("dungeon %s progressOnKill %s should be positive", dungeon.ID, npcID)
			}
			if !dungeonNPCs[npcID] {
				t.Fatalf("dungeon %s progressOnKill references npc %s not present in dungeon rooms", dungeon.ID, npcID)
			}
		}
		for _, obj := range dungeon.Objectives {
			if strings.TrimSpace(obj.ID) == "" || strings.TrimSpace(obj.Type) == "" {
				t.Fatalf("dungeon %s has objective with empty id or type", dungeon.ID)
			}
			if obj.Required <= 0 {
				t.Fatalf("dungeon %s objective %s should have positive required", dungeon.ID, obj.ID)
			}
			switch obj.Type {
			case "kill", "talk":
				if !dungeonNPCs[obj.TargetID] {
					t.Fatalf("dungeon %s objective %s references missing npc %s", dungeon.ID, obj.ID, obj.TargetID)
				}
			case "enter_room":
				if _, ok := dungeon.Rooms[obj.TargetID]; !ok {
					t.Fatalf("dungeon %s objective %s references missing room %s", dungeon.ID, obj.ID, obj.TargetID)
				}
			case "collect":
				if _, ok := data.Items[obj.TargetID]; !ok {
					t.Fatalf("dungeon %s objective %s references missing item %s", dungeon.ID, obj.ID, obj.TargetID)
				}
			default:
				t.Fatalf("dungeon %s objective %s has unsupported type %s", dungeon.ID, obj.ID, obj.Type)
			}
		}
		for itemID, amount := range dungeon.Rewards.Items {
			if amount <= 0 {
				t.Fatalf("dungeon %s reward item %s should be positive", dungeon.ID, itemID)
			}
			if _, ok := data.Items[itemID]; !ok {
				t.Fatalf("dungeon %s reward references missing item %s", dungeon.ID, itemID)
			}
		}
	}
}
