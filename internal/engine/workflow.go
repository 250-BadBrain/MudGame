package engine

import "mudgame/server/internal/model"

type ReconnectResult struct {
	UserID   string
	PlayerID string
}

type TransitionResult struct {
	Room         map[string]interface{}
	FromRoomID   string
	FromRoomName string
	ToRoomID     string
	ToRoomName   string
}

type StateActionResult struct {
	Message        string
	RoomMessage    string
	NeedRoomUpdate bool
}

type MineActionResult struct {
	Experience     int
	Potential      int
	Logs           []string
	RoomMessage    string
	NeedRoomUpdate bool
}

type CombatActionResult struct {
	Message         string
	NeedRoomUpdate  bool
	NeedRoomMessage bool
}

type ReliveActionResult struct {
	Message        string
	Action         string
	NeedRoomUpdate bool
	FromRoomID     string
	FromRoomName   string
	ToRoomID       string
	ToRoomName     string
}

type ShopViewResult struct {
	Items    []map[string]interface{}
	NPCID    string
	NPCName  string
	NPCTitle string
}

type BuyActionResult struct {
	Money        int
	BoughtItemID string
	BoughtAmount int
	Logs         []string
}

type SellActionResult struct {
	Money   int
	Gain    int
	Message string
}

type RoomSnapshotResult struct {
	Room   map[string]interface{}
	Player map[string]interface{}
}

type BackpackResult struct {
	Items []map[string]interface{}
	Money int
}

type RoomUpdateResult struct {
	Players []map[string]interface{}
	NPCs    []map[string]interface{}
}

type DialogueResult struct {
	Dialogue map[string]interface{}
	Message  string
	Logs     []string
}

type QuestListResult struct {
	Quests []map[string]interface{}
}

type FactionListResult struct {
	Factions []map[string]interface{}
}

type RuntimeRepository interface {
	Register(username, password string) error
	Login(username, password string) (string, error)
	IsReconnectTokenValid(username, token string) bool
	DeleteAccount(username, password string) error
	CreateCharacter(username, name string, str, con, agi, perception, willpower int) (string, error)
	ListCharacters(username string) []map[string]string
	CharacterBelongsToUser(username, playerID string) bool
	DeleteCharacter(username, playerID string) error
	SetPlayerOnline(playerID string, online bool)
	GetRoomSnapshot(playerID string) (map[string]interface{}, map[string]interface{}, error)
	GetAttributes(playerID string) (map[string]interface{}, error)
	GetBackpack(playerID string) ([]map[string]interface{}, int, error)
	GetSkills(playerID string) ([]map[string]interface{}, error)
	GetEquipment(playerID string) (map[string]interface{}, error)
	GetWorlds() []map[string]interface{}
	GetDungeons() []map[string]interface{}
	EnterWorld(playerID, worldID string) (map[string]interface{}, error)
	EnterDungeon(playerID, dungeonID string) (map[string]interface{}, error)
	LeaveDungeon(playerID string) (map[string]interface{}, error)
	Move(playerID, direction string) (map[string]interface{}, error)
	GetShop(playerID, npcID string) ([]map[string]interface{}, string, string, string, error)
	Buy(playerID, npcID, itemID string, amount int) (int, int, error)
	DiscardItem(playerID, itemID string, amount int) error
	SellItem(playerID, itemID string, amount int) (int, int, error)
	EquipItem(playerID, itemID string) error
	UnequipItem(playerID, slot string) error
	LearnSkill(playerID, npcID, skillID string) error
	Mine(playerID string) (int, int, string, error)
	DoRoomAction(playerID, action string) (map[string]interface{}, error)
	Talk(playerID, npcID string) (map[string]interface{}, error)
	ChooseDialogue(playerID, npcID, optionID string) (map[string]interface{}, string, []string, error)
	GetQuests(playerID string) ([]map[string]interface{}, error)
	GetFactions(playerID string) ([]map[string]interface{}, error)
	Heal(playerID string) (string, error)
	Meditate(playerID string) (string, error)
	Stop(playerID string) string
	Spar(playerID, targetID string) (string, error)
	Kill(playerID, targetID string) (string, error)
	Relive(playerID string) (string, error)
	GetPlayerRoom(playerID string) (string, string, error)
	GetPlayerStatus(playerID string) (map[string]interface{}, error)
	GetRoomPayloadByID(roomID string) (map[string]interface{}, error)
	GetRoomUpdate(roomID string) ([]map[string]interface{}, []map[string]interface{}, error)
	IsPlayerInRoom(playerID, roomID string) bool
	ProcessCombatTick() (model.CombatTickResult, error)
	ProcessStatusTick() (model.CombatTickResult, error)
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
