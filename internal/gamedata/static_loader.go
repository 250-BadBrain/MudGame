package gamedata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"mudgame/server/internal/model"
)

type StaticData struct {
	WorldOrder []string
	Worlds     map[string]model.World
	Rooms      map[string]model.Room
	Items      map[string]model.ItemDef
	Skills     map[string]model.SkillDef
	NPCs       map[string]model.NPC
	Dialogues  map[string]model.DialogueDef
	Quests     map[string]model.QuestDef
	Factions   map[string]model.FactionDef
}

type rawItem struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Type          string                 `json:"type"`
	Subtype       string                 `json:"subtype"`
	Stackable     bool                   `json:"stackable"`
	Sellable      bool                   `json:"sellable"`
	Value         int                    `json:"value"`
	Rarity        int                    `json:"rarity"`
	Unit          string                 `json:"unit"`
	EquipmentSlot string                 `json:"equipmentSlot"`
	Limit         *int                   `json:"limit"`
	Bind          string                 `json:"bind"`
	QuestItem     bool                   `json:"questItem"`
	Usable        bool                   `json:"usable"`
	Effects       map[string]interface{} `json:"effects"`
	Source        string                 `json:"source"`
}

type rawSkill struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Rarity       int                    `json:"rarity"`
	Type         string                 `json:"type"`
	Category     string                 `json:"category"`
	School       string                 `json:"school"`
	Bonuses      map[string]float64     `json:"bonuses"`
	Requirements map[string]interface{} `json:"requirements"`
}

type rawNPC struct {
	Name         string           `json:"name"`
	Title        string           `json:"title"`
	Description  string           `json:"description"`
	Faction      string           `json:"faction"`
	Role         string           `json:"role"`
	Importance   string           `json:"importance"`
	DialogueID   string           `json:"dialogueId"`
	QuestIDs     []string         `json:"questIds"`
	InstanceOnly bool             `json:"instanceOnly"`
	SpawnRule    string           `json:"spawnRule"`
	Aggressive   bool             `json:"aggressive"`
	Attributes   model.Attributes `json:"attributes"`
	ShopItems    []string         `json:"shopItems"`
	Skills       map[string]int   `json:"skills"`
}

type rawWorldFile struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Order       int                `json:"order"`
	Category    string             `json:"category"`
	Visible     *bool              `json:"visible"`
	SpawnRoomID string             `json:"spawnRoomId"`
	Rooms       map[string]rawRoom `json:"rooms"`
}

type rawRoom struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Area        string            `json:"area"`
	Tags        []string          `json:"tags"`
	SafeZone    bool              `json:"safeZone"`
	Function    string            `json:"function"`
	Actions     []string          `json:"actions"`
	Exits       map[string]string `json:"exits"`
	NPCs        []string          `json:"npcs"`
}

var (
	loadOnce   sync.Once
	cachedData StaticData
	cachedErr  error
)

func MustLoad() StaticData {
	data, err := Load()
	if err != nil {
		panic(err)
	}
	return data
}

func Load() (StaticData, error) {
	loadOnce.Do(func() {
		cachedData, cachedErr = loadFromDisk()
	})
	if cachedErr != nil {
		return StaticData{}, cachedErr
	}
	return cloneStaticData(cachedData), nil
}

func loadFromDisk() (StaticData, error) {
	resourceDir, err := resolveResourceDir()
	if err != nil {
		return StaticData{}, err
	}

	items, itemLimits, err := loadItems(filepath.Join(resourceDir, "items.json"))
	if err != nil {
		return StaticData{}, err
	}
	skills, err := loadSkills(filepath.Join(resourceDir, "skills.json"))
	if err != nil {
		return StaticData{}, err
	}
	npcs, err := loadNPCs(filepath.Join(resourceDir, "npcs.json"), itemLimits, items)
	if err != nil {
		return StaticData{}, err
	}
	dialogues, err := loadDialogues(filepath.Join(resourceDir, "dialogues.json"))
	if err != nil {
		return StaticData{}, err
	}
	quests, err := loadQuests(filepath.Join(resourceDir, "quests.json"))
	if err != nil {
		return StaticData{}, err
	}
	factions, err := loadFactions(filepath.Join(resourceDir, "factions.json"))
	if err != nil {
		return StaticData{}, err
	}
	worldOrder, worlds, rooms, err := loadWorlds(filepath.Join(resourceDir, "worlds"))
	if err != nil {
		return StaticData{}, err
	}

	if len(items) == 0 || len(skills) == 0 || len(npcs) == 0 || len(worlds) == 0 || len(rooms) == 0 {
		return StaticData{}, errors.New("static resource files are loaded but contain empty core datasets")
	}

	return StaticData{
		WorldOrder: worldOrder,
		Worlds:     worlds,
		Rooms:      rooms,
		Items:      items,
		Skills:     skills,
		NPCs:       npcs,
		Dialogues:  dialogues,
		Quests:     quests,
		Factions:   factions,
	}, nil
}

func ResolveResourceDir() (string, error) {
	return resolveResourceDir()
}

func resolveResourceDir() (string, error) {
	candidates := make([]string, 0, 5)
	if envDir := strings.TrimSpace(os.Getenv("MUD_RESOURCES_DIR")); envDir != "" {
		candidates = append(candidates, envDir)
	}

	if root := detectModuleRoot(); root != "" {
		candidates = append(candidates, filepath.Join(root, "resources"))
	}

	candidates = append(candidates,
		"resources",
		filepath.Join("..", "resources"),
		filepath.Join("..", "..", "resources"),
	)

	seen := map[string]bool{}
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		if looksLikeResourceDir(abs) {
			return abs, nil
		}
	}
	return "", fmt.Errorf("unable to locate resources directory, checked: %v", candidates)
}

func looksLikeResourceDir(dir string) bool {
	required := []string{
		filepath.Join(dir, "items.json"),
		filepath.Join(dir, "skills.json"),
		filepath.Join(dir, "npcs.json"),
		filepath.Join(dir, "worlds"),
	}
	for _, p := range required {
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	return true
}

func detectModuleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Dir(file)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func loadItems(path string) (map[string]model.ItemDef, map[string]int, error) {
	rawItems := map[string]rawItem{}
	if err := readJSON(path, &rawItems); err != nil {
		return nil, nil, err
	}

	items := map[string]model.ItemDef{}
	itemLimits := map[string]int{}
	for id, raw := range rawItems {
		if strings.TrimSpace(id) == "" || strings.TrimSpace(raw.Name) == "" {
			continue
		}
		limit := -1
		if raw.Limit != nil {
			limit = *raw.Limit
		}
		itemLimits[id] = limit
		items[id] = model.ItemDef{
			ID:            id,
			Name:          raw.Name,
			Description:   raw.Description,
			Type:          raw.Type,
			Subtype:       raw.Subtype,
			Rarity:        raw.Rarity,
			Value:         raw.Value,
			Stackable:     raw.Stackable,
			Unit:          raw.Unit,
			EquipmentSlot: raw.EquipmentSlot,
			Sellable:      raw.Sellable,
			Bind:          raw.Bind,
			QuestItem:     raw.QuestItem,
			Usable:        raw.Usable,
			Effects:       cloneInterfaceMap(raw.Effects),
			Source:        raw.Source,
		}
	}
	return items, itemLimits, nil
}

func loadSkills(path string) (map[string]model.SkillDef, error) {
	rawSkills := map[string]rawSkill{}
	if err := readJSON(path, &rawSkills); err != nil {
		return nil, err
	}

	skills := map[string]model.SkillDef{}
	for key, raw := range rawSkills {
		id := strings.TrimSpace(raw.ID)
		if id == "" {
			id = strings.TrimSpace(key)
		}
		if id == "" || strings.TrimSpace(raw.Name) == "" {
			continue
		}
		skills[id] = model.SkillDef{
			ID:           id,
			Name:         raw.Name,
			Description:  raw.Description,
			Rarity:       raw.Rarity,
			Type:         raw.Type,
			Category:     raw.Category,
			School:       raw.School,
			Bonuses:      copyFloatMap(raw.Bonuses),
			Requirements: cloneInterfaceMap(raw.Requirements),
		}
	}
	return skills, nil
}

func loadNPCs(path string, itemLimits map[string]int, items map[string]model.ItemDef) (map[string]model.NPC, error) {
	rawNPCs := map[string]rawNPC{}
	if err := readJSON(path, &rawNPCs); err != nil {
		return nil, err
	}

	npcs := map[string]model.NPC{}
	for id, raw := range rawNPCs {
		if strings.TrimSpace(id) == "" || strings.TrimSpace(raw.Name) == "" {
			continue
		}

		shopInventory := map[string]int{}
		for _, itemID := range raw.ShopItems {
			if _, ok := items[itemID]; !ok {
				continue
			}
			limit, ok := itemLimits[itemID]
			if !ok {
				limit = -1
			}
			shopInventory[itemID] = limit
		}

		skills := map[string]int{}
		for skillID, level := range raw.Skills {
			skills[skillID] = level
		}

		npcs[id] = model.NPC{
			ID:            id,
			Name:          raw.Name,
			Title:         raw.Title,
			Description:   raw.Description,
			Faction:       raw.Faction,
			Role:          raw.Role,
			Importance:    raw.Importance,
			DialogueID:    raw.DialogueID,
			QuestIDs:      append([]string{}, raw.QuestIDs...),
			InstanceOnly:  raw.InstanceOnly,
			SpawnRule:     raw.SpawnRule,
			Aggressive:    raw.Aggressive,
			IsShop:        len(shopInventory) > 0,
			ShopInventory: shopInventory,
			Skills:        skills,
			Attributes:    model.NormalizeAttributes(raw.Attributes),
		}
	}
	return npcs, nil
}

func loadDialogues(path string) (map[string]model.DialogueDef, error) {
	dialogues := map[string]model.DialogueDef{}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return dialogues, nil
	}
	raw := map[string]model.DialogueDef{}
	if err := readJSON(path, &raw); err != nil {
		return nil, err
	}
	for key, def := range raw {
		id := strings.TrimSpace(def.ID)
		if id == "" {
			id = strings.TrimSpace(key)
		}
		if id == "" {
			continue
		}
		def.ID = id
		dialogues[id] = def
	}
	return dialogues, nil
}

func loadQuests(path string) (map[string]model.QuestDef, error) {
	quests := map[string]model.QuestDef{}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return quests, nil
	}
	raw := map[string]model.QuestDef{}
	if err := readJSON(path, &raw); err != nil {
		return nil, err
	}
	for key, def := range raw {
		id := strings.TrimSpace(def.ID)
		if id == "" {
			id = strings.TrimSpace(key)
		}
		if id == "" || strings.TrimSpace(def.Name) == "" {
			continue
		}
		def.ID = id
		if def.Rewards.Items == nil {
			def.Rewards.Items = map[string]int{}
		}
		if def.Rewards.Flags == nil {
			def.Rewards.Flags = map[string]bool{}
		}
		if def.Rewards.Reputations == nil {
			def.Rewards.Reputations = map[string]int{}
		}
		quests[id] = def
	}
	return quests, nil
}

func loadFactions(path string) (map[string]model.FactionDef, error) {
	factions := map[string]model.FactionDef{}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return factions, nil
	}
	raw := map[string]model.FactionDef{}
	if err := readJSON(path, &raw); err != nil {
		return nil, err
	}
	for key, def := range raw {
		id := strings.TrimSpace(def.ID)
		if id == "" {
			id = strings.TrimSpace(key)
		}
		if id == "" || strings.TrimSpace(def.Name) == "" {
			continue
		}
		def.ID = id
		factions[id] = def
	}
	return factions, nil
}

func loadWorlds(worldDir string) ([]string, map[string]model.World, map[string]model.Room, error) {
	entries, err := os.ReadDir(worldDir)
	if err != nil {
		return nil, nil, nil, err
	}

	worldIDs := make([]string, 0, len(entries))
	worlds := map[string]model.World{}
	rooms := map[string]model.Room{}

	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		path := filepath.Join(worldDir, entry.Name())
		var rawWorld rawWorldFile
		if err := readJSON(path, &rawWorld); err != nil {
			return nil, nil, nil, err
		}

		worldID := strings.TrimSpace(rawWorld.ID)
		if worldID == "" {
			worldID = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		}
		if worldID == "" {
			continue
		}

		roomNames := map[string]string{}
		roomIDs := make([]string, 0, len(rawWorld.Rooms))
		for roomID, room := range rawWorld.Rooms {
			roomID = strings.TrimSpace(roomID)
			if roomID == "" {
				continue
			}
			name := strings.TrimSpace(room.Name)
			if name == "" {
				name = roomID
			}
			roomNames[roomID] = name
			roomIDs = append(roomIDs, roomID)
		}
		sort.Strings(roomIDs)

		for _, roomID := range roomIDs {
			rawRoom := rawWorld.Rooms[roomID]
			exits := map[string]model.Exit{}
			for dir, targetRoomID := range rawRoom.Exits {
				targetName := roomNames[targetRoomID]
				if targetName == "" {
					targetName = targetRoomID
				}
				exits[dir] = model.Exit{Direction: dir, TargetRoomID: targetRoomID, TargetRoomName: targetName}
			}
			npcs := append([]string{}, rawRoom.NPCs...)
			rooms[roomID] = model.Room{
				ID:          roomID,
				WorldID:     worldID,
				WorldName:   rawWorld.Name,
				Name:        roomNames[roomID],
				Description: rawRoom.Description,
				Area:        rawRoom.Area,
				Tags:        append([]string{}, rawRoom.Tags...),
				SafeZone:    rawRoom.SafeZone,
				Function:    rawRoom.Function,
				Actions:     append([]string{}, rawRoom.Actions...),
				Exits:       exits,
				NPCIDs:      npcs,
			}
		}

		spawnRoomID := strings.TrimSpace(rawWorld.SpawnRoomID)
		if spawnRoomID == "" {
			spawnRoomID = detectSpawnRoomID(roomIDs)
		}
		visible := true
		if rawWorld.Visible != nil {
			visible = *rawWorld.Visible
		}
		worlds[worldID] = model.World{
			ID:          worldID,
			Name:        rawWorld.Name,
			Description: rawWorld.Description,
			Order:       rawWorld.Order,
			Category:    rawWorld.Category,
			Visible:     visible,
			SpawnRoomID: spawnRoomID,
		}
		worldIDs = append(worldIDs, worldID)
	}

	worldOrder := buildWorldOrder(worldIDs, worlds)
	return worldOrder, worlds, rooms, nil
}

func detectSpawnRoomID(roomIDs []string) string {
	if len(roomIDs) == 0 {
		return ""
	}
	for _, roomID := range roomIDs {
		if strings.HasSuffix(roomID, "_00") {
			return roomID
		}
	}
	return roomIDs[0]
}

func buildWorldOrder(foundWorldIDs []string, worlds map[string]model.World) []string {
	out := append([]string{}, foundWorldIDs...)
	sort.Slice(out, func(i, j int) bool {
		a := worlds[out[i]]
		b := worlds[out[j]]
		if a.Order > 0 && b.Order > 0 && a.Order != b.Order {
			return a.Order < b.Order
		}
		if a.Order > 0 && b.Order <= 0 {
			return true
		}
		if a.Order <= 0 && b.Order > 0 {
			return false
		}
		return out[i] < out[j]
	})
	return out
}

func cloneStaticData(in StaticData) StaticData {
	out := StaticData{
		WorldOrder: append([]string{}, in.WorldOrder...),
		Worlds:     map[string]model.World{},
		Rooms:      map[string]model.Room{},
		Items:      map[string]model.ItemDef{},
		Skills:     map[string]model.SkillDef{},
		NPCs:       map[string]model.NPC{},
		Dialogues:  map[string]model.DialogueDef{},
		Quests:     map[string]model.QuestDef{},
		Factions:   map[string]model.FactionDef{},
	}
	for id, w := range in.Worlds {
		out.Worlds[id] = w
	}
	for id, item := range in.Items {
		item.Effects = cloneInterfaceMap(item.Effects)
		out.Items[id] = item
	}
	for id, skill := range in.Skills {
		skill.Bonuses = copyFloatMap(skill.Bonuses)
		skill.Requirements = cloneInterfaceMap(skill.Requirements)
		out.Skills[id] = skill
	}
	for id, room := range in.Rooms {
		roomCopy := room
		roomCopy.NPCIDs = append([]string{}, room.NPCIDs...)
		roomCopy.Tags = append([]string{}, room.Tags...)
		roomCopy.Actions = append([]string{}, room.Actions...)
		roomCopy.Exits = map[string]model.Exit{}
		for dir, ex := range room.Exits {
			roomCopy.Exits[dir] = ex
		}
		out.Rooms[id] = roomCopy
	}
	for id, npc := range in.NPCs {
		npcCopy := npc
		npcCopy.ShopInventory = map[string]int{}
		for itemID, limit := range npc.ShopInventory {
			npcCopy.ShopInventory[itemID] = limit
		}
		npcCopy.Skills = map[string]int{}
		for skillID, level := range npc.Skills {
			npcCopy.Skills[skillID] = level
		}
		npcCopy.QuestIDs = append([]string{}, npc.QuestIDs...)
		out.NPCs[id] = npcCopy
	}
	for id, dialogue := range in.Dialogues {
		dialogueCopy := dialogue
		dialogueCopy.Options = append([]model.DialogueOption{}, dialogue.Options...)
		out.Dialogues[id] = dialogueCopy
	}
	for id, quest := range in.Quests {
		questCopy := quest
		questCopy.Objectives = append([]model.Objective{}, quest.Objectives...)
		questCopy.Rewards.Items = map[string]int{}
		for itemID, amount := range quest.Rewards.Items {
			questCopy.Rewards.Items[itemID] = amount
		}
		questCopy.Rewards.Flags = map[string]bool{}
		for flag, value := range quest.Rewards.Flags {
			questCopy.Rewards.Flags[flag] = value
		}
		questCopy.Rewards.Reputations = map[string]int{}
		for factionID, value := range quest.Rewards.Reputations {
			questCopy.Rewards.Reputations[factionID] = value
		}
		out.Quests[id] = questCopy
	}
	for id, faction := range in.Factions {
		out.Factions[id] = faction
	}
	return out
}

func copyFloatMap(in map[string]float64) map[string]float64 {
	if in == nil {
		return nil
	}
	out := map[string]float64{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneInterfaceMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := map[string]interface{}{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func readJSON(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	return json.Unmarshal(data, out)
}
