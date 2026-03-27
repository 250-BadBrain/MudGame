package engine

import (
	"mudgame/server/internal/model"
	"mudgame/server/internal/storage"
)

type Service struct {
	repo storage.Repository
}

func NewService(repo storage.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(username, password string) error {
	return s.repo.Register(username, password)
}
func (s *Service) Login(username, password string) (string, error) {
	return s.repo.Login(username, password)
}
func (s *Service) IsReconnectTokenValid(username, token string) bool {
	return s.repo.IsReconnectTokenValid(username, token)
}
func (s *Service) DeleteAccount(username, password string) error {
	return s.repo.DeleteAccount(username, password)
}
func (s *Service) CreateCharacter(username, name string, str, con, agi, intel int) (string, error) {
	return s.repo.CreateCharacter(username, name, str, con, agi, intel)
}
func (s *Service) ListCharacters(username string) []map[string]string {
	return s.repo.ListCharacters(username)
}
func (s *Service) CharacterBelongsToUser(username, playerID string) bool {
	return s.repo.CharacterBelongsToUser(username, playerID)
}
func (s *Service) DeleteCharacter(username, playerID string) error {
	return s.repo.DeleteCharacter(username, playerID)
}
func (s *Service) SetPlayerOnline(playerID string, online bool) {
	s.repo.SetPlayerOnline(playerID, online)
}
func (s *Service) GetRoomSnapshot(playerID string) (map[string]interface{}, map[string]interface{}, error) {
	return s.repo.GetRoomSnapshot(playerID)
}
func (s *Service) GetAttributes(playerID string) (map[string]interface{}, error) {
	return s.repo.GetAttributes(playerID)
}
func (s *Service) GetBackpack(playerID string) ([]map[string]interface{}, int, error) {
	return s.repo.GetBackpack(playerID)
}
func (s *Service) GetSkills(playerID string) ([]map[string]interface{}, error) {
	return s.repo.GetSkills(playerID)
}
func (s *Service) GetEquipment(playerID string) (map[string]interface{}, error) {
	return s.repo.GetEquipment(playerID)
}
func (s *Service) GetWorlds() []map[string]interface{} { return s.repo.GetWorlds() }
func (s *Service) EnterWorld(playerID, worldID string) (map[string]interface{}, error) {
	return s.repo.EnterWorld(playerID, worldID)
}
func (s *Service) Move(playerID, direction string) (map[string]interface{}, error) {
	return s.repo.Move(playerID, direction)
}
func (s *Service) GetShop(playerID, npcID string) ([]map[string]interface{}, string, string, string, error) {
	return s.repo.GetShop(playerID, npcID)
}
func (s *Service) Buy(playerID, npcID, itemID string, amount int) (int, int, error) {
	return s.repo.Buy(playerID, npcID, itemID, amount)
}
func (s *Service) DiscardItem(playerID, itemID string, amount int) error {
	return s.repo.DiscardItem(playerID, itemID, amount)
}
func (s *Service) SellItem(playerID, itemID string, amount int) (int, int, error) {
	return s.repo.SellItem(playerID, itemID, amount)
}
func (s *Service) EquipItem(playerID, itemID string) error { return s.repo.EquipItem(playerID, itemID) }
func (s *Service) UnequipItem(playerID, slot string) error { return s.repo.UnequipItem(playerID, slot) }
func (s *Service) LearnSkill(playerID, npcID, skillID string) error {
	return s.repo.LearnSkill(playerID, npcID, skillID)
}
func (s *Service) Mine(playerID string) (int, int, string, error) { return s.repo.Mine(playerID) }
func (s *Service) Heal(playerID string) (string, error)           { return s.repo.Heal(playerID) }
func (s *Service) Meditate(playerID string) (string, error)       { return s.repo.Meditate(playerID) }
func (s *Service) Stop(playerID string) string                    { return s.repo.Stop(playerID) }
func (s *Service) Spar(playerID, targetID string) (string, error) {
	return s.repo.Spar(playerID, targetID)
}
func (s *Service) Kill(playerID, targetID string) (string, error) {
	return s.repo.Kill(playerID, targetID)
}
func (s *Service) Relive(playerID string) (string, error) { return s.repo.Relive(playerID) }
func (s *Service) GetPlayerRoom(playerID string) (string, string, error) {
	return s.repo.GetPlayerRoom(playerID)
}
func (s *Service) GetPlayerStatus(playerID string) (map[string]interface{}, error) {
	return s.repo.GetPlayerStatus(playerID)
}
func (s *Service) GetRoomPayloadByID(roomID string) (map[string]interface{}, error) {
	return s.repo.GetRoomPayloadByID(roomID)
}
func (s *Service) GetRoomUpdate(roomID string) ([]map[string]interface{}, []map[string]interface{}, error) {
	return s.repo.GetRoomUpdate(roomID)
}
func (s *Service) IsPlayerInRoom(playerID, roomID string) bool {
	return s.repo.IsPlayerInRoom(playerID, roomID)
}
func (s *Service) ProcessCombatTick() (model.CombatTickResult, error) {
	return s.repo.ProcessCombatTick()
}
func (s *Service) ProcessStatusTick() (model.CombatTickResult, error) {
	return s.repo.ProcessStatusTick()
}
