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
}

type rawItem struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Stackable     bool   `json:"stackable"`
	Sellable      bool   `json:"sellable"`
	Value         int    `json:"value"`
	Rarity        int    `json:"rarity"`
	Unit          string `json:"unit"`
	EquipmentSlot string `json:"equipmentSlot"`
	Limit         *int   `json:"limit"`
}

type rawSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Rarity      int    `json:"rarity"`
}

type rawNPC struct {
	Name        string           `json:"name"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Attributes  model.Attributes `json:"attributes"`
	ShopItems   []string         `json:"shopItems"`
	Skills      map[string]int   `json:"skills"`
}

type rawWorldFile struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Rooms       map[string]rawRoom `json:"rooms"`
}

type rawRoom struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
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
	}, nil
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
			Rarity:        raw.Rarity,
			Value:         raw.Value,
			Stackable:     raw.Stackable,
			Unit:          raw.Unit,
			EquipmentSlot: raw.EquipmentSlot,
			Sellable:      raw.Sellable,
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
			ID:          id,
			Name:        raw.Name,
			Description: raw.Description,
			Rarity:      raw.Rarity,
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
			IsShop:        len(shopInventory) > 0,
			ShopInventory: shopInventory,
			Skills:        skills,
			Attributes:    raw.Attributes,
		}
	}
	return npcs, nil
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
				Exits:       exits,
				NPCIDs:      npcs,
			}
		}

		spawnRoomID := detectSpawnRoomID(roomIDs)
		worlds[worldID] = model.World{
			ID:          worldID,
			Name:        rawWorld.Name,
			Description: rawWorld.Description,
			SpawnRoomID: spawnRoomID,
		}
		worldIDs = append(worldIDs, worldID)
	}

	worldOrder := buildWorldOrder(worldIDs)
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

func buildWorldOrder(foundWorldIDs []string) []string {
	preferred := []string{"luoyang", "wudang", "shaolin", "emei", "huashan", "kunlun", "kongtong"}
	seen := map[string]bool{}
	out := make([]string, 0, len(foundWorldIDs))
	foundSet := map[string]bool{}
	for _, id := range foundWorldIDs {
		foundSet[id] = true
	}
	for _, id := range preferred {
		if foundSet[id] {
			out = append(out, id)
			seen[id] = true
		}
	}
	left := make([]string, 0)
	for _, id := range foundWorldIDs {
		if !seen[id] {
			left = append(left, id)
		}
	}
	sort.Strings(left)
	out = append(out, left...)
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
	}
	for id, w := range in.Worlds {
		out.Worlds[id] = w
	}
	for id, item := range in.Items {
		out.Items[id] = item
	}
	for id, skill := range in.Skills {
		out.Skills[id] = skill
	}
	for id, room := range in.Rooms {
		roomCopy := room
		roomCopy.NPCIDs = append([]string{}, room.NPCIDs...)
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
		out.NPCs[id] = npcCopy
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
