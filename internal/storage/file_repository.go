package storage

import (
	"crypto/hmac"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"mudgame/server/internal/exception"
	"mudgame/server/internal/gamedata"
	"mudgame/server/internal/model"

	"github.com/google/uuid"
)

type FileRepository struct {
	baseDir string
	mu      sync.Mutex
}

var (
	loadedStaticData = gamedata.MustLoad()
	fileWorldOrder   = append([]string{}, loadedStaticData.WorldOrder...)
	fileWorlds       = loadedStaticData.Worlds
	fileRooms        = loadedStaticData.Rooms
	fileItems        = loadedStaticData.Items
	fileSkills       = loadedStaticData.Skills
	fileNpcs         = loadedStaticData.NPCs
)

func NewFileRepository(baseDir string) *FileRepository {
	return &FileRepository{baseDir: baseDir}
}

func (r *FileRepository) Register(username, password string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if username == "" || password == "" {
		return errors.New("username_or_password_empty")
	}
	users, err := r.loadUsers()
	if err != nil {
		return err
	}
	if _, exists := users[username]; exists {
		return errors.New("username_exists")
	}
	users[username] = &model.User{Username: username, PasswordHash: hashPBKDF2Password(password)}
	return r.saveUsers(users)
}

func (r *FileRepository) Login(username, password string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.loadUsers()
	if err != nil {
		return "", err
	}
	u, exists := users[username]
	if !exists {
		return "", exception.ErrUserNotFound
	}
	if !matchesPassword(u, password) {
		return "", exception.ErrWrongPassword
	}
	u.ReconnectToken = uuid.NewString()
	if err := r.saveUsers(users); err != nil {
		return "", err
	}
	return u.ReconnectToken, nil
}

func (r *FileRepository) IsReconnectTokenValid(username, token string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.loadUsers()
	if err != nil {
		return false
	}
	u, ok := users[username]
	if !ok {
		return false
	}
	return token != "" && token == u.ReconnectToken
}

func (r *FileRepository) DeleteAccount(username, password string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.loadUsers()
	if err != nil {
		return err
	}
	u, exists := users[username]
	if !exists {
		return exception.ErrUserNotFound
	}
	if !matchesPassword(u, password) {
		return exception.ErrWrongPassword
	}

	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	for _, pid := range u.CharacterIDs {
		delete(players, pid)
	}
	delete(users, username)

	if err := r.savePlayers(players); err != nil {
		return err
	}
	return r.saveUsers(users)
}

func (r *FileRepository) CreateCharacter(username, name string, str, con, agi, intel int) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.loadUsers()
	if err != nil {
		return "", err
	}
	u, ok := users[username]
	if !ok {
		return "", exception.ErrUserNotFound
	}
	if len(name) < 2 || len(name) > 16 {
		return "", errors.New("invalid_character_name")
	}
	if str+con+agi+intel != 80 {
		return "", errors.New("invalid_attribute_sum")
	}

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}

	id := uuid.NewString()
	players[id] = &model.Player{
		ID:            id,
		UserID:        username,
		CharacterName: name,
		RoomID:        "ly_00",
		State:         "IDLE",
		Money:         5000,
		Attributes: model.Attributes{
			Experience: 0, Potential: 500,
			CurrentHP: 100, MaxHP: 100,
			CurrentMP: 80, TrainedMaxMP: 80, MaxMP: 200,
			Strength: str, Constitution: con, Agility: agi, Intelligence: intel,
			Appearance: 20, Luck: 20,
			Attack: 15, Defense: 15, HitRate: 100, Dodge: 100, Parry: 100, CritRate: 1, AttackSpeed: 4,
		},
		Skills:    map[string]int{"basic_fist": 1, "basic_parry": 1, "basic_internal": 1, "basic_light": 1},
		Backpack:  map[string]int{"iron_pickaxe": 1},
		Equipment: map[string]string{},
		Online:    false,
	}
	u.CharacterIDs = append(u.CharacterIDs, id)

	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	if err := r.saveUsers(users); err != nil {
		return "", err
	}
	return id, nil
}

func (r *FileRepository) ListCharacters(username string) []map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.loadUsers()
	if err != nil {
		return []map[string]string{}
	}
	u, exists := users[username]
	if !exists {
		return []map[string]string{}
	}
	players, err := r.loadPlayers()
	if err != nil {
		return []map[string]string{}
	}
	out := make([]map[string]string, 0, len(u.CharacterIDs))
	for _, id := range u.CharacterIDs {
		if p, ok := players[id]; ok {
			out = append(out, map[string]string{"id": p.ID, "name": p.CharacterName})
		}
	}
	return out
}

func (r *FileRepository) CharacterBelongsToUser(username, playerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return false
	}
	p, ok := players[playerID]
	if !ok {
		return false
	}
	return p.UserID == username
}

func (r *FileRepository) DeleteCharacter(username, playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok || p.UserID != username {
		return exception.ErrPlayerNotFound
	}
	delete(players, playerID)

	users, err := r.loadUsers()
	if err != nil {
		return err
	}
	u := users[username]
	kept := make([]string, 0, len(u.CharacterIDs))
	for _, id := range u.CharacterIDs {
		if id != playerID {
			kept = append(kept, id)
		}
	}
	u.CharacterIDs = kept

	if err := r.savePlayers(players); err != nil {
		return err
	}
	return r.saveUsers(users)
}

func (r *FileRepository) SetPlayerOnline(playerID string, online bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return
	}
	if p, ok := players[playerID]; ok {
		p.Online = online
		_ = r.savePlayers(players)
	}
}

func (r *FileRepository) GetRoomSnapshot(playerID string) (map[string]interface{}, map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, nil, exception.ErrPlayerNotFound
	}
	roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
	if err != nil {
		return nil, nil, err
	}
	playerPayload := map[string]interface{}{"id": p.ID, "username": p.CharacterName, "userId": p.UserID, "state": p.State}
	return roomPayload, playerPayload, nil
}
func (r *FileRepository) GetAttributes(playerID string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	a := p.Attributes
	return map[string]interface{}{
		"experience": a.Experience, "potential": a.Potential,
		"currentHp": a.CurrentHP, "maxHp": a.MaxHP,
		"currentMp": a.CurrentMP, "trainedMaxMp": a.TrainedMaxMP, "maxMp": a.MaxMP,
		"strength": a.Strength, "constitution": a.Constitution, "agility": a.Agility, "intelligence": a.Intelligence,
		"appearance": a.Appearance, "luck": a.Luck,
		"attack": a.Attack, "defense": a.Defense,
		"hitRate": a.HitRate, "dodge": a.Dodge, "parry": a.Parry, "critRate": a.CritRate, "attackSpeed": a.AttackSpeed,
		"postStrength": a.PostStrength, "postConstitution": a.PostConstitution, "postAgility": a.PostAgility, "postIntelligence": a.PostIntelligence,
	}, nil
}
func (r *FileRepository) GetBackpack(playerID string) ([]map[string]interface{}, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, 0, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, 0, exception.ErrPlayerNotFound
	}
	items := make([]map[string]interface{}, 0, len(p.Backpack))
	for id, count := range p.Backpack {
		def, ok := fileItems[id]
		if !ok || count <= 0 {
			continue
		}
		items = append(items, map[string]interface{}{
			"id": def.ID, "uniqueId": def.ID, "name": def.Name, "description": def.Description,
			"rarity": def.Rarity, "value": def.Value, "stackable": def.Stackable,
			"unit": def.Unit, "count": count, "equipmentSlot": def.EquipmentSlot, "sellable": def.Sellable,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i]["name"].(string) < items[j]["name"].(string) })
	return items, p.Money, nil
}
func (r *FileRepository) GetSkills(playerID string) ([]map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	out := make([]map[string]interface{}, 0, len(p.Skills))
	for id, level := range p.Skills {
		def, ok := fileSkills[id]
		if !ok {
			continue
		}
		out = append(out, map[string]interface{}{"id": id, "name": def.Name, "description": def.Description, "rarity": def.Rarity, "level": level})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
	return out, nil
}
func (r *FileRepository) GetEquipment(playerID string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	out := map[string]interface{}{}
	for slot, itemID := range p.Equipment {
		if itemID == "" {
			continue
		}
		def, ok := fileItems[itemID]
		if !ok {
			continue
		}
		out[slot] = map[string]interface{}{"id": def.ID, "name": def.Name, "description": def.Description, "rarity": def.Rarity}
	}
	return out, nil
}
func (r *FileRepository) GetWorlds() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(fileWorldOrder))
	for _, id := range fileWorldOrder {
		w := fileWorlds[id]
		out = append(out, map[string]interface{}{"id": w.ID, "name": w.Name, "description": w.Description})
	}
	return out
}
func (r *FileRepository) EnterWorld(playerID, worldID string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		if p.State == "DEAD" {
			return nil, errors.New("dead_state")
		}
		return nil, errors.New("busy_state")
	}
	w, ok := fileWorlds[worldID]
	if !ok {
		return nil, errors.New("world_not_found")
	}
	if p.RoomID == w.SpawnRoomID {
		roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
		return roomPayload, err
	}
	p.RoomID = w.SpawnRoomID
	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
	return roomPayload, err
}
func (r *FileRepository) Move(playerID, direction string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	rooms, err := r.loadRooms()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		if p.State == "DEAD" {
			return nil, errors.New("dead_state")
		}
		switch p.State {
		case "HEALING":
			return nil, errors.New("healing_state")
		case "MEDITATING":
			return nil, errors.New("meditating_state")
		case "MINING":
			return nil, errors.New("mining_state")
		case "LEARNING":
			return nil, errors.New("learning_state")
		case "COMBAT":
			return nil, errors.New("combat_state")
		default:
			return nil, errors.New("busy_state")
		}
	}
	room, ok := rooms[p.RoomID]
	if !ok {
		return nil, exception.ErrRoomNotFound
	}
	ex, ok := room.Exits[direction]
	if !ok {
		return nil, errors.New("no_exit_in_that_direction")
	}
	p.RoomID = ex.TargetRoomID
	if err := r.savePlayers(players); err != nil {
		return nil, err
	}
	roomPayload, _, err := r.buildRoomPayload(p.RoomID, players)
	return roomPayload, err
}
func (r *FileRepository) GetShop(playerID, npcID string) ([]map[string]interface{}, string, string, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, "", "", "", err
	}
	rooms, err := r.loadRooms()
	if err != nil {
		return nil, "", "", "", err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, "", "", "", exception.ErrPlayerNotFound
	}
	room, ok := rooms[p.RoomID]
	if !ok {
		return nil, "", "", "", exception.ErrRoomNotFound
	}
	npcAllowed := false
	for _, id := range room.NPCIDs {
		if id == npcID {
			npcAllowed = true
			break
		}
	}
	if !npcAllowed {
		return nil, "", "", "", errors.New("shop_npc_not_in_room")
	}
	npc, ok := fileNpcs[npcID]
	if !ok || !npc.IsShop {
		return nil, "", "", "", errors.New("npc_not_shop")
	}
	out := make([]map[string]interface{}, 0, len(npc.ShopInventory))
	for itemID, limit := range npc.ShopInventory {
		def, ok := fileItems[itemID]
		if !ok {
			continue
		}
		entry := map[string]interface{}{
			"id":          def.ID,
			"name":        def.Name,
			"description": def.Description,
			"rarity":      def.Rarity,
			"value":       def.Value,
			"stackable":   def.Stackable,
			"unit":        def.Unit,
		}
		if limit != -1 {
			entry["limit"] = limit
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["name"].(string) < out[j]["name"].(string) })
	return out, npc.ID, npc.Name, npc.Title, nil
}
func (r *FileRepository) Buy(playerID, npcID, itemID string, amount int) (int, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if amount <= 0 {
		amount = 1
	}
	players, err := r.loadPlayers()
	if err != nil {
		return 0, 0, err
	}
	rooms, err := r.loadRooms()
	if err != nil {
		return 0, 0, err
	}
	p, ok := players[playerID]
	if !ok {
		return 0, 0, exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return 0, 0, errors.New("busy")
	}
	room, ok := rooms[p.RoomID]
	if !ok {
		return 0, 0, exception.ErrRoomNotFound
	}
	npcInRoom := false
	for _, id := range room.NPCIDs {
		if id == npcID {
			npcInRoom = true
			break
		}
	}
	if !npcInRoom {
		return 0, 0, errors.New("shop_npc_not_in_room")
	}
	npc, ok := fileNpcs[npcID]
	if !ok || !npc.IsShop {
		return 0, 0, errors.New("shop_not_found")
	}
	limit, ok := npc.ShopInventory[itemID]
	if !ok {
		return 0, 0, errors.New("sold_out")
	}
	if limit != -1 && limit <= 0 {
		return 0, 0, errors.New("sold_out")
	}
	if limit != -1 && amount > limit {
		amount = limit
	}
	def, ok := fileItems[itemID]
	if !ok {
		return 0, 0, errors.New("item_not_found")
	}
	total := def.Value * amount
	if p.Money < total {
		return 0, 0, errors.New("money_not_enough")
	}
	p.Money -= total
	p.Backpack[itemID] += amount
	if limit != -1 {
		npc.ShopInventory[itemID] -= amount
	}
	if err := r.savePlayers(players); err != nil {
		return 0, 0, err
	}
	return amount, p.Money, nil
}
func (r *FileRepository) DiscardItem(playerID, itemID string, amount int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if amount <= 0 {
		amount = 1
	}
	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok {
		return exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return errors.New("busy")
	}
	have := p.Backpack[itemID]
	if have <= 0 {
		return errors.New("item_not_found")
	}
	if amount > have {
		amount = have
	}
	p.Backpack[itemID] -= amount
	if p.Backpack[itemID] <= 0 {
		delete(p.Backpack, itemID)
	}
	return r.savePlayers(players)
}
func (r *FileRepository) SellItem(playerID, itemID string, amount int) (int, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if amount <= 0 {
		amount = 1
	}
	players, err := r.loadPlayers()
	if err != nil {
		return 0, 0, err
	}
	rooms, err := r.loadRooms()
	if err != nil {
		return 0, 0, err
	}
	p, ok := players[playerID]
	if !ok {
		return 0, 0, exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return 0, 0, errors.New("busy")
	}
	room, ok := rooms[p.RoomID]
	if !ok {
		return 0, 0, exception.ErrRoomNotFound
	}
	if len(room.NPCIDs) == 0 {
		return 0, 0, errors.New("no_npc_buyer")
	}
	have := p.Backpack[itemID]
	if have <= 0 {
		return 0, 0, errors.New("item_not_found")
	}
	if amount > have {
		amount = have
	}
	def, ok := fileItems[itemID]
	if !ok {
		return 0, 0, errors.New("item_not_found")
	}
	if !def.Sellable {
		return 0, 0, errors.New("item_not_sellable")
	}
	p.Backpack[itemID] -= amount
	if p.Backpack[itemID] <= 0 {
		delete(p.Backpack, itemID)
	}
	gain := (def.Value / 2) * amount
	p.Money += gain
	if err := r.savePlayers(players); err != nil {
		return 0, 0, err
	}
	return gain, p.Money, nil
}
func (r *FileRepository) EquipItem(playerID, itemID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok {
		return exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return errors.New("busy")
	}
	if itemID == "" {
		return errors.New("missing_item_id")
	}
	if p.Backpack[itemID] <= 0 {
		return errors.New("item_not_in_inventory")
	}
	def, ok := fileItems[itemID]
	if !ok || def.EquipmentSlot == "" {
		return errors.New("not_equippable")
	}
	if p.Equipment == nil {
		p.Equipment = map[string]string{}
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	slot := def.EquipmentSlot
	if old := p.Equipment[slot]; old != "" {
		p.Backpack[old] += 1
	}
	p.Backpack[itemID] -= 1
	if p.Backpack[itemID] <= 0 {
		delete(p.Backpack, itemID)
	}
	p.Equipment[slot] = itemID
	return r.savePlayers(players)
}
func (r *FileRepository) UnequipItem(playerID, slot string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok {
		return exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return errors.New("busy")
	}
	if slot == "" {
		return errors.New("missing_slot")
	}
	if p.Equipment == nil {
		p.Equipment = map[string]string{}
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	itemID := p.Equipment[slot]
	if itemID == "" {
		return errors.New("slot_empty")
	}
	p.Backpack[itemID] += 1
	delete(p.Equipment, slot)
	return r.savePlayers(players)
}
func (r *FileRepository) LearnSkill(playerID, npcID, skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return err
	}
	rooms, err := r.loadRooms()
	if err != nil {
		return err
	}
	p, ok := players[playerID]
	if !ok {
		return exception.ErrPlayerNotFound
	}
	if p.State != "IDLE" {
		return errors.New("busy_state")
	}
	if npcID == "" || skillID == "" {
		return errors.New("missing_params")
	}
	if p.Attributes.Potential <= 0 {
		return errors.New("no_potential")
	}
	room, ok := rooms[p.RoomID]
	if !ok {
		return exception.ErrRoomNotFound
	}
	npcInRoom := false
	for _, id := range room.NPCIDs {
		if id == npcID {
			npcInRoom = true
			break
		}
	}
	if !npcInRoom {
		return errors.New("npc_not_found")
	}
	npc, ok := fileNpcs[npcID]
	if !ok {
		return errors.New("npc_not_found")
	}
	if _, ok := npc.Skills[skillID]; !ok {
		return errors.New("skill_not_taught")
	}
	if p.Skills[skillID] >= 300 {
		return errors.New("skill_level_capped")
	}
	if p.Skills == nil {
		p.Skills = map[string]int{}
	}
	p.Attributes.Potential -= 1
	p.Skills[skillID] += 1
	p.State = "LEARNING"
	return r.savePlayers(players)
}
func (r *FileRepository) Mine(playerID string) (int, int, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return 0, 0, "", err
	}
	p, ok := players[playerID]
	if !ok {
		return 0, 0, "", exception.ErrPlayerNotFound
	}
	if p.State == "DEAD" {
		return 0, 0, "", errors.New("dead_state")
	}
	if p.State != "IDLE" {
		return 0, 0, "", errors.New("busy")
	}
	if p.RoomID != "ly_mine" {
		return 0, 0, "", errors.New("wrong_location")
	}
	if p.Backpack == nil {
		p.Backpack = map[string]int{}
	}
	if p.Equipment["WEAPON"] != "iron_pickaxe" {
		return 0, 0, "", errors.New("no_pickaxe")
	}
	p.State = "MINING"
	if err := r.savePlayers(players); err != nil {
		return 0, 0, "", err
	}
	return p.Attributes.Experience, p.Attributes.Potential, "", nil
}
func (r *FileRepository) Heal(playerID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	if p.State == "COMBAT" {
		return "", errors.New("combat_state")
	}
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}
	if p.State == "HEALING" {
		return "", errors.New("healing_state")
	}
	if p.State == "MEDITATING" {
		return "", errors.New("meditating_state")
	}
	if p.State == "MINING" {
		return "", errors.New("mining_state")
	}
	if p.State == "LEARNING" {
		return "", errors.New("learning_state")
	}
	if p.Attributes.CurrentHP >= p.Attributes.MaxHP {
		return "", errors.New("hp_full")
	}
	if p.Attributes.CurrentMP < 10 {
		return "", errors.New("mp_insufficient")
	}
	p.Attributes.CurrentMP -= 10
	p.Attributes.CurrentHP += 20
	if p.Attributes.CurrentHP > p.Attributes.MaxHP {
		p.Attributes.CurrentHP = p.Attributes.MaxHP
	}
	p.State = "HEALING"
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你开始运功疗伤。", nil
}
func (r *FileRepository) Meditate(playerID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	if p.State == "COMBAT" {
		return "", errors.New("combat_state")
	}
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}
	if p.State == "HEALING" {
		return "", errors.New("healing_state")
	}
	if p.State == "MEDITATING" {
		return "", errors.New("meditating_state")
	}
	if p.State == "MINING" {
		return "", errors.New("mining_state")
	}
	if p.State == "LEARNING" {
		return "", errors.New("learning_state")
	}
	if p.Attributes.TrainedMaxMP >= p.Attributes.MaxMP && p.Attributes.CurrentMP >= p.Attributes.TrainedMaxMP {
		return "", errors.New("full_state")
	}
	p.Attributes.CurrentMP += 15
	if p.Attributes.CurrentMP > p.Attributes.TrainedMaxMP {
		p.Attributes.CurrentMP = p.Attributes.TrainedMaxMP
	}
	p.State = "MEDITATING"
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你开始打坐调息。", nil
}
func (r *FileRepository) Stop(playerID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err == nil {
		if p, ok := players[playerID]; ok {
			if p.State == "COMBAT" {
				return "你正在战斗，待会再说"
			}
			oldState := p.State
			p.State = "IDLE"
			p.TargetID = ""
			p.CombatType = ""
			_ = r.savePlayers(players)
			switch oldState {
			case "MINING":
				return "你收起铁镐，停止了挖矿。"
			case "HEALING":
				return "你停止了运功疗伤。"
			case "MEDITATING":
				return "你停止了打坐。"
			case "LEARNING":
				return "你停止了学习。"
			}
		}
	}
	return "你停止了动作。"
}
func (r *FileRepository) Spar(playerID, targetID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok1 := players[playerID]
	if !ok1 {
		return "", errors.New("player_not_found")
	}
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}
	if p.State != "IDLE" {
		return "", errors.New("busy")
	}
	if p.Attributes.CurrentHP < p.Attributes.MaxHP {
		return "", errors.New("self_not_full")
	}
	if targetID == "" {
		return "", errors.New("target_missing")
	}
	if p.ID == targetID {
		return "", errors.New("self_target")
	}
	if _, isNPC := fileNpcs[targetID]; isNPC {
		return "", errors.New("npc_combat_unsupported")
	}
	t, ok2 := players[targetID]
	if !ok2 {
		return "", errors.New("target_not_found")
	}
	if p.RoomID != t.RoomID {
		return "", errors.New("target_not_in_room")
	}
	if t.State == "DEAD" {
		return "", errors.New("target_is_dead")
	}
	if t.State != "IDLE" {
		return "", errors.New("target_busy")
	}
	if t.Attributes.CurrentHP < t.Attributes.MaxHP {
		return "", errors.New("target_not_full")
	}
	p.State = "COMBAT"
	p.TargetID = t.ID
	p.CombatType = "SPAR"
	t.State = "COMBAT"
	t.TargetID = p.ID
	t.CombatType = "SPAR"
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return p.CharacterName + "对" + t.CharacterName + "抱拳道：请赐教！", nil
}
func (r *FileRepository) Kill(playerID, targetID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok1 := players[playerID]
	if !ok1 {
		return "", errors.New("player_not_found")
	}
	if p.State == "DEAD" {
		return "", errors.New("dead_state")
	}
	if p.State != "IDLE" {
		return "", errors.New("busy_state")
	}
	if targetID == "" {
		return "", errors.New("target_missing")
	}
	if p.ID == targetID {
		return "", errors.New("self_target")
	}
	if _, isNPC := fileNpcs[targetID]; isNPC {
		return "", errors.New("npc_combat_unsupported")
	}
	t, ok2 := players[targetID]
	if !ok2 {
		return "", errors.New("target_not_found")
	}
	if p.RoomID != t.RoomID {
		return "", errors.New("target_not_in_room")
	}
	if t.State == "DEAD" {
		return "", errors.New("target_is_dead")
	}
	if t.State == "COMBAT" {
		return "", errors.New("target_in_combat")
	}
	p.State = "COMBAT"
	p.TargetID = t.ID
	p.CombatType = "KILL"
	t.State = "COMBAT"
	t.TargetID = p.ID
	t.CombatType = "KILL"
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return p.CharacterName + "对" + t.CharacterName + "喝道：纳命来！", nil
}
func (r *FileRepository) Relive(playerID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", exception.ErrPlayerNotFound
	}
	if p.State != "DEAD" {
		return "", errors.New("not_dead")
	}
	p.State = "IDLE"
	p.TargetID = ""
	p.CombatType = ""
	p.Attributes.CurrentHP = p.Attributes.MaxHP
	p.Attributes.CurrentMP = p.Attributes.TrainedMaxMP
	p.RoomID = "ly_00"
	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	return "你苏醒了过来。", nil
}

func (r *FileRepository) ProcessStatusTick() (model.CombatTickResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return model.CombatTickResult{}, err
	}

	result := model.CombatTickResult{}
	updatedRoom := map[string]bool{}

	for _, p := range players {
		roomID := p.RoomID
		switch p.State {
		case "HEALING":
			if p.Attributes.CurrentHP >= p.Attributes.MaxHP || p.Attributes.CurrentMP < 8 {
				p.State = "IDLE"
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: p.CharacterName + "停止了运功疗伤。"})
				updatedRoom[roomID] = true
				continue
			}
			beforeHP := p.Attributes.CurrentHP
			p.Attributes.CurrentMP -= 8
			p.Attributes.CurrentHP += 20
			if p.Attributes.CurrentHP > p.Attributes.MaxHP {
				p.Attributes.CurrentHP = p.Attributes.MaxHP
			}
			recovered := p.Attributes.CurrentHP - beforeHP
			if recovered > 0 {
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你恢复了" + strconv.Itoa(recovered) + "点气血。"})
			}
			updatedRoom[roomID] = true
		case "MEDITATING":
			if p.Attributes.CurrentMP < p.Attributes.TrainedMaxMP {
				beforeMP := p.Attributes.CurrentMP
				p.Attributes.CurrentMP += 20
				if p.Attributes.CurrentMP > p.Attributes.TrainedMaxMP {
					p.Attributes.CurrentMP = p.Attributes.TrainedMaxMP
				}
				recovered := p.Attributes.CurrentMP - beforeMP
				if recovered > 0 {
					result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你恢复了" + strconv.Itoa(recovered) + "点内力。"})
				}
				updatedRoom[roomID] = true
				continue
			}
			if p.Attributes.TrainedMaxMP >= p.Attributes.MaxMP {
				p.State = "IDLE"
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: p.CharacterName + "结束了打坐调息。"})
				updatedRoom[roomID] = true
				continue
			}
			beforeMaxMP := p.Attributes.TrainedMaxMP
			p.Attributes.TrainedMaxMP += 1
			if p.Attributes.TrainedMaxMP > p.Attributes.MaxMP {
				p.Attributes.TrainedMaxMP = p.Attributes.MaxMP
			}
			increased := p.Attributes.TrainedMaxMP - beforeMaxMP
			p.Attributes.CurrentMP = p.Attributes.TrainedMaxMP
			if increased > 0 {
				result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你增加了" + strconv.Itoa(increased) + "点最大内力。"})
			}
			updatedRoom[roomID] = true
		case "MINING":
			if p.Equipment["WEAPON"] != "iron_pickaxe" {
				p.State = "IDLE"
				result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: p.CharacterName + "收起铁镐，停止了挖矿。"})
				updatedRoom[roomID] = true
				continue
			}
			expGain := 20
			potGain := 40
			p.Attributes.Experience += expGain
			p.Attributes.Potential += potGain
			gemID := "cracked_red_gem"
			if _, ok := fileItems[gemID]; !ok {
				gemID = "gem_red"
			}
			if _, ok := fileItems[gemID]; ok {
				if p.Backpack == nil {
					p.Backpack = map[string]int{}
				}
				p.Backpack[gemID] += 1
			}
			result.PlayerMessages = append(result.PlayerMessages, model.PlayerMessage{PlayerID: p.ID, Message: "你获得了" + strconv.Itoa(expGain) + "点经验，" + strconv.Itoa(potGain) + "点潜能。"})
			updatedRoom[roomID] = true
		}
	}

	for roomID := range updatedRoom {
		result.UpdatedRoomIDs = append(result.UpdatedRoomIDs, roomID)
	}
	if err := r.savePlayers(players); err != nil {
		return model.CombatTickResult{}, err
	}
	return result, nil
}

func (r *FileRepository) ProcessCombatTick() (model.CombatTickResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return model.CombatTickResult{}, err
	}

	result := model.CombatTickResult{}
	updatedRoom := map[string]bool{}
	processed := map[string]bool{}

	for _, attacker := range players {
		if attacker.State != "COMBAT" || attacker.TargetID == "" || processed[attacker.ID] {
			continue
		}
		target, ok := players[attacker.TargetID]
		if !ok || target.State != "COMBAT" || target.TargetID != attacker.ID || target.RoomID != attacker.RoomID {
			roomID := attacker.RoomID
			lostTargetID := attacker.TargetID
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: attacker.CharacterName + "茫然地四处张望，找不到对手了。"})
			r.resolveCombatTargetLossNoLock(players, attacker, lostTargetID, roomID, &result)
			if ok {
				r.resolveCombatTargetLossNoLock(players, target, attacker.ID, target.RoomID, &result)
			}
			r.resolveOtherCombatantsTargetingNoLock(players, roomID, lostTargetID, attacker.ID, &result)
			updatedRoom[roomID] = true
			if ok && target.RoomID != roomID {
				updatedRoom[target.RoomID] = true
			}
			continue
		}

		processed[attacker.ID] = true
		processed[target.ID] = true
		roomID := attacker.RoomID
		combatType := attacker.CombatType
		if combatType == "" {
			combatType = target.CombatType
		}
		if combatType == "" {
			combatType = "SPAR"
		}

		strikeAToB := r.resolveCombatStrike(attacker, target, combatType)
		strikeBToA := r.resolveCombatStrike(target, attacker, combatType)

		target.Attributes.CurrentHP -= strikeAToB.Damage
		if target.Attributes.CurrentHP < 0 {
			target.Attributes.CurrentHP = 0
		}
		attacker.Attributes.CurrentHP -= strikeBToA.Damage
		if attacker.Attributes.CurrentHP < 0 {
			attacker.Attributes.CurrentHP = 0
		}

		if strikeAToB.Message != "" {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: strikeAToB.Message})
		}
		if strikeBToA.Message != "" {
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: strikeBToA.Message})
		}
		updatedRoom[roomID] = true

		if combatType == "KILL" {
			if attacker.Attributes.CurrentHP == 0 || target.Attributes.CurrentHP == 0 {
				r.resolveKillEndNoLock(attacker, target, roomID, &result)
			}
			continue
		}

		attackerLimit := attacker.Attributes.MaxHP / 5
		targetLimit := target.Attributes.MaxHP / 5
		attackerLow := attacker.Attributes.CurrentHP <= attackerLimit
		targetLow := target.Attributes.CurrentHP <= targetLimit
		if attackerLow || targetLow {
			r.clearCombatNoLock(attacker)
			r.clearCombatNoLock(target)
			result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: r.sparEndMessage(attacker, target, attackerLow, targetLow)})
			updatedRoom[roomID] = true
		}
	}

	for roomID := range updatedRoom {
		result.UpdatedRoomIDs = append(result.UpdatedRoomIDs, roomID)
	}
	if err := r.savePlayers(players); err != nil {
		return model.CombatTickResult{}, err
	}
	return result, nil
}

func (r *FileRepository) resolveKillEndNoLock(a, b *model.Player, roomID string, result *model.CombatTickResult) {
	if a.Attributes.CurrentHP == 0 && b.Attributes.CurrentHP == 0 {
		a.State = "DEAD"
		a.TargetID = ""
		a.CombatType = ""
		b.State = "DEAD"
		b.TargetID = ""
		b.CombatType = ""
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: a.CharacterName + "与" + b.CharacterName + "同归于尽。"})
		return
	}
	if b.Attributes.CurrentHP == 0 {
		b.State = "DEAD"
		b.TargetID = ""
		b.CombatType = ""
		r.clearCombatNoLock(a)
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: b.CharacterName + "大叫一声，倒在地上，死了！"})
		return
	}
	if a.Attributes.CurrentHP == 0 {
		a.State = "DEAD"
		a.TargetID = ""
		a.CombatType = ""
		r.clearCombatNoLock(b)
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: a.CharacterName + "大叫一声，倒在地上，死了！"})
	}
}

func (r *FileRepository) clearCombatNoLock(p *model.Player) {
	if p.State != "DEAD" {
		p.State = "IDLE"
	}
	p.TargetID = ""
	p.CombatType = ""
}

func (r *FileRepository) resolveCombatTargetLossNoLock(players map[string]*model.Player, combatant *model.Player, lostTargetID, roomID string, result *model.CombatTickResult) {
	if combatant == nil || combatant.State != "COMBAT" || combatant.TargetID != lostTargetID {
		return
	}
	replacement := r.findReplacementTargetNoLock(players, combatant, lostTargetID)
	if replacement == nil {
		r.clearCombatNoLock(combatant)
		return
	}
	combatant.TargetID = replacement.ID
	result.Messages = append(result.Messages, model.RoomMessage{RoomID: roomID, Message: combatant.CharacterName + "转身盯上了" + replacement.CharacterName + "！"})
}

func (r *FileRepository) resolveOtherCombatantsTargetingNoLock(players map[string]*model.Player, roomID, lostTargetID, excludeCombatantID string, result *model.CombatTickResult) {
	if roomID == "" || lostTargetID == "" {
		return
	}
	for _, p := range players {
		if p.State != "COMBAT" {
			continue
		}
		if excludeCombatantID != "" && p.ID == excludeCombatantID {
			continue
		}
		if p.RoomID != roomID || p.TargetID != lostTargetID {
			continue
		}
		r.resolveCombatTargetLossNoLock(players, p, lostTargetID, roomID, result)
	}
}

func (r *FileRepository) findReplacementTargetNoLock(players map[string]*model.Player, combatant *model.Player, lostTargetID string) *model.Player {
	if combatant == nil || combatant.RoomID == "" {
		return nil
	}
	for _, candidate := range players {
		if candidate == nil || candidate.ID == combatant.ID || candidate.ID == lostTargetID {
			continue
		}
		if candidate.State != "COMBAT" || candidate.RoomID != combatant.RoomID || candidate.TargetID != combatant.ID {
			continue
		}
		if combatant.CombatType != "" && candidate.CombatType != "" && combatant.CombatType != candidate.CombatType {
			continue
		}
		return candidate
	}
	return nil
}

func (r *FileRepository) sparEndMessage(attacker, target *model.Player, attackerLow, targetLow bool) string {
	if attackerLow && targetLow {
		return "双方点到为止，抱拳收招。"
	}
	winner := attacker
	loser := target
	if attackerLow && !targetLow {
		winner = target
		loser = attacker
	}
	if rand.Intn(2) == 0 {
		return loser.CharacterName + "后退三尺，抱拳道：佩服、佩服！"
	}
	return winner.CharacterName + "抱拳道：承让！"
}

type combatStrikeResult struct {
	Damage  int
	Message string
}

func (r *FileRepository) resolveCombatStrike(attacker, defender *model.Player, combatType string) combatStrikeResult {
	hitChance := boundedChance(attacker.Attributes.HitRate, defender.Attributes.Dodge)
	if rand.Float64() > hitChance {
		return combatStrikeResult{Damage: 0, Message: attacker.CharacterName + "发起攻击，但被" + defender.CharacterName + "躲开了！"}
	}

	base := attacker.Attributes.Attack - defender.Attributes.Defense/3
	if combatType == "KILL" {
		base += 14
	} else {
		base += 8
	}
	if base < 2 {
		base = 2
	}

	fluctuation := 90 + rand.Intn(21)
	damage := base * fluctuation / 100

	crit := rand.Intn(100) < attacker.Attributes.CritRate
	if crit {
		damage *= 2
	}

	parryChance := boundedChance(defender.Attributes.Parry, attacker.Attributes.HitRate)
	parried := rand.Float64() < parryChance
	if parried {
		damage /= 2
	}

	msg := attacker.CharacterName + "一招击中" + defender.CharacterName + "，造成" + strconv.Itoa(damage) + "点伤害"
	if crit {
		msg = attacker.CharacterName + "发动致命一击，重创" + defender.CharacterName + "，造成" + strconv.Itoa(damage) + "点伤害"
	}
	if parried {
		msg += "（被招架减伤）"
	}
	if damage <= 0 {
		msg = attacker.CharacterName + "一招击中" + defender.CharacterName + "，但是没有造成任何伤害。"
		return combatStrikeResult{Damage: 0, Message: msg}
	}
	msg += "。"
	return combatStrikeResult{Damage: damage, Message: msg}
}

func boundedChance(attackerStat, defenderStat int) float64 {
	den := attackerStat + defenderStat
	if den <= 0 {
		return 0.7
	}
	raw := float64(attackerStat) / float64(den)
	if raw < 0.45 {
		return 0.45
	}
	if raw > 0.95 {
		return 0.95
	}
	return raw
}
func (r *FileRepository) GetPlayerRoom(playerID string) (string, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", "", err
	}
	rooms, err := r.loadRooms()
	if err != nil {
		return "", "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", "", exception.ErrPlayerNotFound
	}
	room, ok := rooms[p.RoomID]
	if !ok {
		return "", "", exception.ErrRoomNotFound
	}
	return room.ID, room.Name, nil
}
func (r *FileRepository) GetPlayerStatus(playerID string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	rooms, err := r.loadRooms()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	room, ok := rooms[p.RoomID]
	if !ok {
		return nil, exception.ErrRoomNotFound
	}
	return map[string]interface{}{
		"playerId":   p.ID,
		"playerName": p.CharacterName,
		"status":     onlineLabel(p.Online),
		"state":      p.State,
		"online":     p.Online,
		"roomId":     room.ID,
		"roomName":   room.Name,
	}, nil
}
func (r *FileRepository) GetRoomPayloadByID(roomID string) (map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	roomPayload, _, err := r.buildRoomPayload(roomID, players)
	return roomPayload, err
}
func (r *FileRepository) GetRoomUpdate(roomID string) ([]map[string]interface{}, []map[string]interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, nil, err
	}
	playersInRoom, npcs, err := r.buildRoomEntities(roomID, players)
	if err != nil {
		return nil, nil, err
	}
	return playersInRoom, npcs, nil
}
func (r *FileRepository) IsPlayerInRoom(playerID, roomID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	players, err := r.loadPlayers()
	if err != nil {
		return false
	}
	p, ok := players[playerID]
	if !ok {
		return false
	}
	return p.RoomID == roomID
}

func (r *FileRepository) usersDir() string {
	return filepath.Join(r.baseDir, "users")
}

func (r *FileRepository) playersDir() string {
	return filepath.Join(r.baseDir, "players")
}

func (r *FileRepository) roomsDir() string {
	return filepath.Join(r.baseDir, "rooms")
}

func (r *FileRepository) ensureBaseDir() error {
	if err := os.MkdirAll(r.usersDir(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(r.playersDir(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(r.roomsDir(), 0o755); err != nil {
		return err
	}
	return nil
}

type diskUser struct {
	Username       string   `json:"username"`
	PasswordHash   string   `json:"passwordHash"`
	CharacterIDs   []string `json:"characterIds"`
	ReconnectToken string   `json:"reconnectToken,omitempty"`
	ReconnectHash  string   `json:"reconnectTokenHash,omitempty"`
}

type diskPlayer struct {
	ID              string            `json:"id"`
	CharacterName   string            `json:"characterName"`
	UserID          string            `json:"userId"`
	RoomID          string            `json:"roomId"`
	Title           interface{}       `json:"title"`
	UnlockedTitles  interface{}       `json:"unlockedTitles"`
	Money           int               `json:"money"`
	Attributes      model.Attributes  `json:"attributes"`
	Items           map[string]int    `json:"items"`
	Equipment       map[string]string `json:"equipment"`
	Skills          map[string]int    `json:"skills"`
	State           string            `json:"state"`
	TargetID        string            `json:"targetId"`
	LearningSkillID interface{}       `json:"learningSkillId"`
	CombatType      string            `json:"combatType"`
}

type diskRoomDynamic struct {
	NPCs    []string `json:"npcs"`
	Players []string `json:"players"`
}

func (r *FileRepository) loadUsers() (map[string]*model.User, error) {
	if err := r.ensureBaseDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(r.usersDir())
	if err != nil {
		return nil, err
	}
	users := map[string]*model.User{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var du diskUser
		if err := readJSONFile(filepath.Join(r.usersDir(), entry.Name()), &du); err != nil {
			return nil, err
		}
		if du.Username == "" {
			continue
		}
		users[du.Username] = &model.User{
			Username:       du.Username,
			PasswordHash:   du.PasswordHash,
			CharacterIDs:   append([]string{}, du.CharacterIDs...),
			ReconnectToken: du.ReconnectToken,
		}
	}
	return users, nil
}

func (r *FileRepository) saveUsers(users map[string]*model.User) error {
	if err := r.ensureBaseDir(); err != nil {
		return err
	}
	if err := removeAllJSONFiles(r.usersDir()); err != nil {
		return err
	}
	for _, u := range users {
		if u == nil || u.Username == "" {
			continue
		}
		d := diskUser{
			Username:       u.Username,
			PasswordHash:   u.PasswordHash,
			CharacterIDs:   append([]string{}, u.CharacterIDs...),
			ReconnectToken: u.ReconnectToken,
			ReconnectHash:  "",
		}
		if err := writeJSONFile(filepath.Join(r.usersDir(), u.Username+".json"), d); err != nil {
			return err
		}
	}
	return nil
}

func (r *FileRepository) loadPlayers() (map[string]*model.Player, error) {
	if err := r.ensureBaseDir(); err != nil {
		return nil, err
	}
	rooms, err := r.loadRooms()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(r.playersDir())
	if err != nil {
		return nil, err
	}
	players := map[string]*model.Player{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var dp diskPlayer
		if err := readJSONFile(filepath.Join(r.playersDir(), entry.Name()), &dp); err != nil {
			return nil, err
		}
		if dp.ID == "" {
			continue
		}
		if dp.Items == nil {
			dp.Items = map[string]int{}
		}
		if dp.Equipment == nil {
			dp.Equipment = map[string]string{}
		}
		if dp.Skills == nil {
			dp.Skills = map[string]int{"basic_fist": 1, "basic_parry": 1, "basic_internal": 1, "basic_light": 1}
		}
		if dp.RoomID == "" || rooms[dp.RoomID].ID == "" {
			dp.RoomID = "ly_00"
		}
		state := dp.State
		if state == "" {
			state = "IDLE"
		}
		players[dp.ID] = &model.Player{
			ID:            dp.ID,
			UserID:        dp.UserID,
			CharacterName: dp.CharacterName,
			RoomID:        dp.RoomID,
			State:         state,
			TargetID:      dp.TargetID,
			CombatType:    dp.CombatType,
			Money:         dp.Money,
			Attributes:    dp.Attributes,
			Skills:        dp.Skills,
			Backpack:      dp.Items,
			Equipment:     dp.Equipment,
			Online:        false,
		}
	}
	return players, nil
}

func (r *FileRepository) savePlayers(players map[string]*model.Player) error {
	if err := r.ensureBaseDir(); err != nil {
		return err
	}
	if err := removeAllJSONFiles(r.playersDir()); err != nil {
		return err
	}
	for _, p := range players {
		if p == nil || p.ID == "" {
			continue
		}
		d := diskPlayer{
			ID:              p.ID,
			CharacterName:   p.CharacterName,
			UserID:          p.UserID,
			RoomID:          p.RoomID,
			Title:           nil,
			UnlockedTitles:  nil,
			Money:           p.Money,
			Attributes:      p.Attributes,
			Items:           p.Backpack,
			Equipment:       p.Equipment,
			Skills:          p.Skills,
			State:           p.State,
			TargetID:        p.TargetID,
			LearningSkillID: nil,
			CombatType:      p.CombatType,
		}
		if err := writeJSONFile(filepath.Join(r.playersDir(), p.ID+".json"), d); err != nil {
			return err
		}
	}
	if err := r.syncDynamicRooms(players); err != nil {
		return err
	}
	return nil
}

func (r *FileRepository) loadDynamicRooms() (map[string]diskRoomDynamic, error) {
	if err := r.ensureBaseDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(r.roomsDir())
	if err != nil {
		return nil, err
	}
	out := map[string]diskRoomDynamic{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		roomID := strings.TrimSuffix(entry.Name(), ".json")
		var dr diskRoomDynamic
		if err := readJSONFile(filepath.Join(r.roomsDir(), entry.Name()), &dr); err != nil {
			return nil, err
		}
		if dr.NPCs == nil {
			dr.NPCs = []string{}
		}
		if dr.Players == nil {
			dr.Players = []string{}
		}
		out[roomID] = dr
	}
	return out, nil
}

func (r *FileRepository) loadRooms() (map[string]model.Room, error) {
	dynamic, err := r.loadDynamicRooms()
	if err != nil {
		return nil, err
	}
	rooms := map[string]model.Room{}
	for id, room := range fileRooms {
		rm := room
		rm.NPCIDs = append([]string{}, room.NPCIDs...)
		rm.Exits = cloneExits(room.Exits)
		if d, ok := dynamic[id]; ok {
			rm.NPCIDs = append([]string{}, d.NPCs...)
		}
		rooms[id] = rm
	}
	for id, d := range dynamic {
		if _, ok := rooms[id]; ok {
			continue
		}
		npcs := append([]string{}, d.NPCs...)
		rooms[id] = model.Room{ID: id, Name: id, Description: "", WorldID: "", WorldName: "", Exits: map[string]model.Exit{}, NPCIDs: npcs}
	}
	return rooms, nil
}

func cloneExits(in map[string]model.Exit) map[string]model.Exit {
	out := map[string]model.Exit{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (r *FileRepository) syncDynamicRooms(players map[string]*model.Player) error {
	dynamic, err := r.loadDynamicRooms()
	if err != nil {
		return err
	}
	for id, d := range dynamic {
		d.Players = []string{}
		dynamic[id] = d
	}
	for id, sr := range fileRooms {
		if _, ok := dynamic[id]; !ok {
			dynamic[id] = diskRoomDynamic{NPCs: append([]string{}, sr.NPCIDs...), Players: []string{}}
		}
	}
	for _, p := range players {
		if p == nil || p.RoomID == "" {
			continue
		}
		d := dynamic[p.RoomID]
		if d.NPCs == nil {
			d.NPCs = []string{}
		}
		d.Players = append(d.Players, p.ID)
		dynamic[p.RoomID] = d
	}
	for roomID, d := range dynamic {
		sort.Strings(d.NPCs)
		sort.Strings(d.Players)
		d.NPCs = uniqueStrings(d.NPCs)
		d.Players = uniqueStrings(d.Players)
		if err := writeJSONFile(filepath.Join(r.roomsDir(), roomID+".json"), d); err != nil {
			return err
		}
	}
	return nil
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	out := make([]string, 0, len(in))
	last := ""
	for i, v := range in {
		if i == 0 || v != last {
			out = append(out, v)
			last = v
		}
	}
	return out
}

func matchesPassword(u *model.User, plain string) bool {
	if u == nil {
		return false
	}
	if u.PasswordHash != "" {
		if verifyPBKDF2Password(plain, u.PasswordHash) {
			return true
		}
	}
	return u.Password == plain
}

func hashPBKDF2Password(password string) string {
	iter := 210000
	salt := make([]byte, 16)
	if _, err := cryptorand.Read(salt); err != nil {
		return password
	}
	dk := pbkdf2SHA256([]byte(password), salt, iter, 32)
	return fmt.Sprintf("pbkdf2$%d$%s$%s", iter, base64.RawURLEncoding.EncodeToString(salt), base64.RawURLEncoding.EncodeToString(dk))
}

func verifyPBKDF2Password(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter <= 0 {
		return false
	}
	salt, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got := pbkdf2SHA256([]byte(password), salt, iter, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	hLen := 32
	numBlocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, numBlocks*hLen)
	for block := 1; block <= numBlocks; block++ {
		buf := make([]byte, len(salt)+4)
		copy(buf, salt)
		buf[len(salt)] = byte(block >> 24)
		buf[len(salt)+1] = byte(block >> 16)
		buf[len(salt)+2] = byte(block >> 8)
		buf[len(salt)+3] = byte(block)

		h := hmac.New(sha256.New, password)
		h.Write(buf)
		u := h.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iter; i++ {
			h = hmac.New(sha256.New, password)
			h.Write(u)
			u = h.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func removeAllJSONFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func readJSONFile(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

func writeJSONFile(path string, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (r *FileRepository) buildRoomPayload(roomID string, players map[string]*model.Player) (map[string]interface{}, []map[string]interface{}, error) {
	rooms, err := r.loadRooms()
	if err != nil {
		return nil, nil, err
	}
	room, ok := rooms[roomID]
	if !ok {
		return nil, nil, exception.ErrRoomNotFound
	}
	playersInRoom, npcs, err := r.buildRoomEntities(roomID, players)
	if err != nil {
		return nil, nil, err
	}
	exits := map[string]map[string]interface{}{}
	for dir, ex := range room.Exits {
		exits[dir] = map[string]interface{}{"targetRoomId": ex.TargetRoomID, "targetRoomName": ex.TargetRoomName}
	}
	roomPayload := map[string]interface{}{
		"id":            room.ID,
		"worldId":       room.WorldID,
		"worldName":     room.WorldName,
		"name":          room.Name,
		"description":   room.Description,
		"playersInRoom": playersInRoom,
		"npcs":          npcs,
		"items":         []interface{}{},
		"exits":         exits,
	}
	return roomPayload, playersInRoom, nil
}

func (r *FileRepository) buildRoomEntities(roomID string, players map[string]*model.Player) ([]map[string]interface{}, []map[string]interface{}, error) {
	rooms, err := r.loadRooms()
	if err != nil {
		return nil, nil, err
	}
	room, ok := rooms[roomID]
	if !ok {
		return nil, nil, exception.ErrRoomNotFound
	}
	playersInRoom := make([]map[string]interface{}, 0)
	for _, p := range players {
		if p.RoomID == room.ID {
			playersInRoom = append(playersInRoom, map[string]interface{}{
				"id":     p.ID,
				"name":   p.CharacterName,
				"title":  "",
				"state":  p.State,
				"online": p.Online,
				"attributes": map[string]interface{}{
					"currentHp":    p.Attributes.CurrentHP,
					"maxHp":        p.Attributes.MaxHP,
					"currentMp":    p.Attributes.CurrentMP,
					"trainedMaxMp": p.Attributes.TrainedMaxMP,
				},
			})
		}
	}
	npcs := make([]map[string]interface{}, 0, len(room.NPCIDs))
	for _, npcID := range room.NPCIDs {
		npc, ok := fileNpcs[npcID]
		if !ok {
			continue
		}
		npcs = append(npcs, map[string]interface{}{
			"id":          npc.ID,
			"name":        npc.Name,
			"title":       npc.Title,
			"description": npc.Description,
		})
	}
	return playersInRoom, npcs, nil
}

func onlineLabel(online bool) string {
	if online {
		return "在线"
	}
	return "离线"
}
