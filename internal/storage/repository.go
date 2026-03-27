package storage

import "mudgame/server/internal/model"

type Repository interface {
	Register(username, password string) error
	Login(username, password string) (string, error)
	IsReconnectTokenValid(username, token string) bool
	DeleteAccount(username, password string) error
	CreateCharacter(username, name string, str, con, agi, intel int) (string, error)
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
	EnterWorld(playerID, worldID string) (map[string]interface{}, error)
	Move(playerID, direction string) (map[string]interface{}, error)
	GetShop(playerID, npcID string) ([]map[string]interface{}, string, string, string, error)
	Buy(playerID, npcID, itemID string, amount int) (int, int, error)
	DiscardItem(playerID, itemID string, amount int) error
	SellItem(playerID, itemID string, amount int) (int, int, error)
	EquipItem(playerID, itemID string) error
	UnequipItem(playerID, slot string) error
	LearnSkill(playerID, npcID, skillID string) error
	Mine(playerID string) (int, int, string, error)
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
