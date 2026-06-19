package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mudgame/server/internal/model"
)

type RoomDynamic struct {
	NPCs    []string `json:"npcs"`
	Players []string `json:"players"`
}

type DynamicStore interface {
	LoadUsers() (map[string]*model.User, error)
	SaveUsers(users map[string]*model.User) error
	LoadPlayers() (map[string]*model.Player, error)
	SavePlayers(players map[string]*model.Player) error
	LoadDynamicRooms() (map[string]RoomDynamic, error)
	SaveDynamicRooms(dynamic map[string]RoomDynamic) error
}

type FileStore struct {
	baseDir string
}

type diskUser struct {
	Username       string   `json:"username"`
	PasswordHash   string   `json:"passwordHash"`
	CharacterIDs   []string `json:"characterIds"`
	ReconnectToken string   `json:"reconnectToken,omitempty"`
	ReconnectHash  string   `json:"reconnectTokenHash,omitempty"`
}

type diskPlayer struct {
	ID                 string            `json:"id"`
	CharacterName      string            `json:"characterName"`
	UserID             string            `json:"userId"`
	RoomID             string            `json:"roomId"`
	Title              interface{}       `json:"title"`
	UnlockedTitles     interface{}       `json:"unlockedTitles"`
	Money              int               `json:"money"`
	Attributes         model.Attributes  `json:"attributes"`
	Items              map[string]int    `json:"items"`
	Equipment          map[string]string `json:"equipment"`
	Skills             map[string]int    `json:"skills"`
	State              string            `json:"state"`
	StateTickCountdown int               `json:"stateTickCountdown"`
	TargetID           string            `json:"targetId"`
	LearningSkillID    interface{}       `json:"learningSkillId"`
	CombatType         string            `json:"combatType"`
}

func NewFileStore(baseDir string) *FileStore {
	s := &FileStore{baseDir: baseDir}
	if err := s.ensureBaseDir(); err != nil {
		panic(err)
	}
	return s
}

func (s *FileStore) LoadUsers() (map[string]*model.User, error) {
	if err := s.ensureBaseDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.usersDir())
	if err != nil {
		return nil, err
	}
	users := map[string]*model.User{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var du diskUser
		if err := readJSONFile(filepath.Join(s.usersDir(), entry.Name()), &du); err != nil {
			return nil, err
		}
		if du.Username == "" {
			continue
		}
		reconnectHash := du.ReconnectHash
		if reconnectHash == "" && du.ReconnectToken != "" {
			reconnectHash = hashLegacyReconnectToken(du.ReconnectToken)
		}
		users[du.Username] = &model.User{
			Username:           du.Username,
			PasswordHash:       du.PasswordHash,
			CharacterIDs:       append([]string{}, du.CharacterIDs...),
			ReconnectTokenHash: reconnectHash,
		}
	}
	return users, nil
}

func (s *FileStore) SaveUsers(users map[string]*model.User) error {
	if err := s.ensureBaseDir(); err != nil {
		return err
	}
	if err := removeAllJSONFiles(s.usersDir()); err != nil {
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
			ReconnectToken: "",
			ReconnectHash:  u.ReconnectTokenHash,
		}
		if err := writeJSONFile(filepath.Join(s.usersDir(), u.Username+".json"), d); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStore) LoadPlayers() (map[string]*model.Player, error) {
	if err := s.ensureBaseDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.playersDir())
	if err != nil {
		return nil, err
	}
	players := map[string]*model.Player{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var dp diskPlayer
		if err := readJSONFile(filepath.Join(s.playersDir(), entry.Name()), &dp); err != nil {
			return nil, err
		}
		if dp.ID == "" {
			continue
		}
		roomID := dp.RoomID
		if roomID == "" || strings.HasPrefix(roomID, "dg_") {
			roomID = "ly_00"
		}
		if dp.Items == nil {
			dp.Items = map[string]int{}
		}
		if dp.Equipment == nil {
			dp.Equipment = map[string]string{}
		}
		if dp.Skills == nil {
			dp.Skills = map[string]int{"basic_unarmed": 1, "basic_parry": 1, "basic_neigong": 1, "basic_qinggong": 1}
		}
		state := dp.State
		if state == "" {
			state = "IDLE"
		}
		learningSkillID := ""
		switch v := dp.LearningSkillID.(type) {
		case string:
			learningSkillID = v
		}
		players[dp.ID] = &model.Player{
			ID:                 dp.ID,
			UserID:             dp.UserID,
			CharacterName:      dp.CharacterName,
			RoomID:             roomID,
			State:              state,
			StateTickCountdown: dp.StateTickCountdown,
			LearningSkillID:    learningSkillID,
			TargetID:           dp.TargetID,
			CombatType:         dp.CombatType,
			Money:              dp.Money,
			Attributes:         dp.Attributes,
			Skills:             dp.Skills,
			Backpack:           dp.Items,
			Equipment:          dp.Equipment,
			Online:             false,
		}
	}
	return players, nil
}

func (s *FileStore) SavePlayers(players map[string]*model.Player) error {
	if err := s.ensureBaseDir(); err != nil {
		return err
	}
	if err := removeAllJSONFiles(s.playersDir()); err != nil {
		return err
	}
	for _, p := range players {
		if p == nil || p.ID == "" {
			continue
		}
		roomID := p.RoomID
		state := p.State
		targetID := p.TargetID
		combatType := p.CombatType
		learningSkillID := interface{}(p.LearningSkillID)
		if strings.HasPrefix(roomID, "dg_") {
			roomID = "ly_00"
			state = "IDLE"
			targetID = ""
			combatType = ""
			learningSkillID = ""
		}
		d := diskPlayer{
			ID:                 p.ID,
			CharacterName:      p.CharacterName,
			UserID:             p.UserID,
			RoomID:             roomID,
			Title:              nil,
			UnlockedTitles:     nil,
			Money:              p.Money,
			Attributes:         p.Attributes,
			Items:              p.Backpack,
			Equipment:          p.Equipment,
			Skills:             p.Skills,
			State:              state,
			StateTickCountdown: p.StateTickCountdown,
			TargetID:           targetID,
			LearningSkillID:    learningSkillID,
			CombatType:         combatType,
		}
		if err := writeJSONFile(filepath.Join(s.playersDir(), p.ID+".json"), d); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStore) LoadDynamicRooms() (map[string]RoomDynamic, error) {
	if err := s.ensureBaseDir(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.roomsDir())
	if err != nil {
		return nil, err
	}
	out := map[string]RoomDynamic{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		roomID := strings.TrimSuffix(entry.Name(), ".json")
		if strings.HasPrefix(roomID, "dg_") {
			continue
		}
		var dr RoomDynamic
		if err := readJSONFile(filepath.Join(s.roomsDir(), entry.Name()), &dr); err != nil {
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

func (s *FileStore) SaveDynamicRooms(dynamic map[string]RoomDynamic) error {
	if err := s.ensureBaseDir(); err != nil {
		return err
	}
	if err := removeAllJSONFiles(s.roomsDir()); err != nil {
		return err
	}
	for roomID, d := range dynamic {
		if strings.HasPrefix(roomID, "dg_") {
			continue
		}
		if err := writeJSONFile(filepath.Join(s.roomsDir(), roomID+".json"), d); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStore) usersDir() string {
	return filepath.Join(s.baseDir, "users")
}

func (s *FileStore) playersDir() string {
	return filepath.Join(s.baseDir, "players")
}

func (s *FileStore) roomsDir() string {
	return filepath.Join(s.baseDir, "rooms")
}

func (s *FileStore) ensureBaseDir() error {
	if err := os.MkdirAll(s.usersDir(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.playersDir(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.roomsDir(), 0o755); err != nil {
		return err
	}
	return nil
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

func hashLegacyReconnectToken(token string) string {
	if token == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte("mudgame-reconnect-token"))
	h.Write([]byte(token))
	dk := h.Sum(nil)
	return fmt.Sprintf("tokenv1$%s", base64.RawURLEncoding.EncodeToString(dk))
}
