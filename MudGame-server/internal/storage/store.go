package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	QuickCompleteSequence int                         `json:"quickCompleteSequence,omitempty"`
	MartialVersion        int                         `json:"martialVersion"`
	ActiveArts            map[string]string           `json:"activeArts,omitempty"`
	RecoveryItemReadyAt   int64                       `json:"recoveryItemReadyAt,omitempty"`
	ID                    string                      `json:"id"`
	CharacterName         string                      `json:"characterName"`
	UserID                string                      `json:"userId"`
	RoomID                string                      `json:"roomId"`
	Title                 interface{}                 `json:"title"`
	UnlockedTitles        interface{}                 `json:"unlockedTitles"`
	Money                 int                         `json:"money"`
	Attributes            model.Attributes            `json:"attributes"`
	Items                 map[string]int              `json:"items"`
	Equipment             map[string]string           `json:"equipment"`
	Skills                map[string]int              `json:"skills"`
	EquippedSkills        map[string]string           `json:"equippedSkills,omitempty"`
	EnhancedItems         map[string]int              `json:"enhancedItems,omitempty"`
	QuestStates           map[string]model.QuestState `json:"questStates"`
	Flags                 map[string]bool             `json:"flags"`
	DialogueHistory       map[string]bool             `json:"dialogueHistory"`
	Reputations           map[string]int              `json:"reputations"`
	State                 string                      `json:"state"`
	StateTickCountdown    int                         `json:"stateTickCountdown"`
	TargetID              string                      `json:"targetId"`
	LearningSkillID       interface{}                 `json:"learningSkillId"`
	LearningTeacherID     string                      `json:"learningTeacherId"`
	LearningTeacherLevel  int                         `json:"learningTeacherLevel"`
	SkillLearningProgress map[string]int              `json:"skillLearningProgress"`
	CombatType            string                      `json:"combatType"`
	WeakUntil             int64                       `json:"weakUntil,omitempty"`
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
		if err := readJSONFileTolerant(filepath.Join(s.usersDir(), entry.Name()), &du); err != nil {
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
	keep := map[string]bool{}
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
		name := u.Username + ".json"
		keep[name] = true
		if err := writeJSONFile(filepath.Join(s.usersDir(), name), d); err != nil {
			return err
		}
	}
	return removeJSONFilesNotIn(s.usersDir(), keep)
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
		if err := readJSONFileTolerant(filepath.Join(s.playersDir(), entry.Name()), &dp); err != nil {
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
		if dp.QuestStates == nil {
			dp.QuestStates = map[string]model.QuestState{}
		}
		if dp.Flags == nil {
			dp.Flags = map[string]bool{}
		}
		if dp.DialogueHistory == nil {
			dp.DialogueHistory = map[string]bool{}
		}
		if dp.Reputations == nil {
			dp.Reputations = map[string]int{}
		}
		if dp.SkillLearningProgress == nil {
			dp.SkillLearningProgress = map[string]int{}
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
		attrs := model.NormalizeAttributes(dp.Attributes)
		// 副本内死亡的角色落盘时会重置为 ly_00/IDLE 但保留 HP=0，
		// 加载时恢复气血，避免出现"活着但零血"的异常状态
		if state != "DEAD" && attrs.Resources.HP <= 0 {
			attrs.Resources.HP = attrs.Resources.MaxHP
		}
		players[dp.ID] = &model.Player{
			QuickCompleteSequence: dp.QuickCompleteSequence, MartialVersion: dp.MartialVersion, ActiveArts: dp.ActiveArts, RecoveryItemReadyAt: dp.RecoveryItemReadyAt,
			ID:                    dp.ID,
			UserID:                dp.UserID,
			CharacterName:         dp.CharacterName,
			RoomID:                roomID,
			State:                 state,
			StateTickCountdown:    dp.StateTickCountdown,
			LearningSkillID:       learningSkillID,
			LearningTeacherID:     dp.LearningTeacherID,
			LearningTeacherLevel:  dp.LearningTeacherLevel,
			SkillLearningProgress: dp.SkillLearningProgress,
			TargetID:              dp.TargetID,
			CombatType:            dp.CombatType,
			Money:                 dp.Money,
			Attributes:            attrs,
			Skills:                dp.Skills,
			EquippedSkills:        dp.EquippedSkills,
			EnhancedItems:         dp.EnhancedItems,
			Backpack:              dp.Items,
			Equipment:             dp.Equipment,
			QuestStates:           dp.QuestStates,
			Flags:                 dp.Flags,
			DialogueHistory:       dp.DialogueHistory,
			Reputations:           dp.Reputations,
			Online:                false,
			WeakUntil:             dp.WeakUntil,
		}
	}
	return players, nil
}

func (s *FileStore) SavePlayers(players map[string]*model.Player) error {
	if err := s.ensureBaseDir(); err != nil {
		return err
	}
	keep := map[string]bool{}
	for _, p := range players {
		if p == nil || p.ID == "" {
			continue
		}
		roomID := p.RoomID
		state := p.State
		targetID := p.TargetID
		combatType := p.CombatType
		learningSkillID := interface{}(p.LearningSkillID)
		learningTeacherID := p.LearningTeacherID
		learningTeacherLevel := p.LearningTeacherLevel
		if strings.HasPrefix(roomID, "dg_") {
			roomID = "ly_00"
			state = "IDLE"
			targetID = ""
			combatType = ""
			learningSkillID = ""
			learningTeacherID = ""
			learningTeacherLevel = 0
		}
		d := diskPlayer{
			QuickCompleteSequence: p.QuickCompleteSequence, MartialVersion: p.MartialVersion, ActiveArts: p.ActiveArts, RecoveryItemReadyAt: p.RecoveryItemReadyAt,
			ID:                    p.ID,
			CharacterName:         p.CharacterName,
			UserID:                p.UserID,
			RoomID:                roomID,
			Title:                 nil,
			UnlockedTitles:        nil,
			Money:                 p.Money,
			Attributes:            model.NormalizeAttributes(p.Attributes),
			Items:                 p.Backpack,
			Equipment:             p.Equipment,
			Skills:                p.Skills,
			EquippedSkills:        p.EquippedSkills,
			EnhancedItems:         p.EnhancedItems,
			QuestStates:           p.QuestStates,
			Flags:                 p.Flags,
			DialogueHistory:       p.DialogueHistory,
			Reputations:           p.Reputations,
			State:                 state,
			StateTickCountdown:    p.StateTickCountdown,
			TargetID:              targetID,
			LearningSkillID:       learningSkillID,
			LearningTeacherID:     learningTeacherID,
			LearningTeacherLevel:  learningTeacherLevel,
			SkillLearningProgress: p.SkillLearningProgress,
			CombatType:            combatType,
			WeakUntil:             p.WeakUntil,
		}
		name := p.ID + ".json"
		keep[name] = true
		if err := writeJSONFile(filepath.Join(s.playersDir(), name), d); err != nil {
			return err
		}
	}
	return removeJSONFilesNotIn(s.playersDir(), keep)
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
		if err := readJSONFileTolerant(filepath.Join(s.roomsDir(), entry.Name()), &dr); err != nil {
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
	keep := map[string]bool{}
	for roomID, d := range dynamic {
		if strings.HasPrefix(roomID, "dg_") {
			continue
		}
		name := roomID + ".json"
		keep[name] = true
		if err := writeJSONFile(filepath.Join(s.roomsDir(), name), d); err != nil {
			return err
		}
	}
	return removeJSONFilesNotIn(s.roomsDir(), keep)
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

// removeJSONFilesNotIn 删除目录中不再需要的 .json 与 .tmp 文件。
// 写入采用 tmp+rename 原子替换，因此先写后清理：即使中途崩溃也只留下旧文件或完整新文件。
func removeJSONFilesNotIn(dir string, keep map[string]bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if keep[name] {
			continue
		}
		if filepath.Ext(name) == ".json" || strings.HasSuffix(name, ".tmp") {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				return err
			}
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
		return errEmptyFile
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%w: %v", errUnparsableFile, err)
	}
	return nil
}

var errEmptyFile = errors.New("empty data file")

var errUnparsableFile = errors.New("unparsable data file")

// readJSONFileTolerant 读取数据文件；文件损坏（空文件/非法 JSON）时将其重命名为
// <name>.corrupt 隔离并记日志，跳过该文件继续加载，而不是让整个服务无法启动，
// 也避免后续 removeJSONFilesNotIn 把坏文件当作冗余直接删除。
func readJSONFileTolerant(path string, out interface{}) error {
	err := readJSONFile(path, out)
	if err == nil {
		return nil
	}
	if !errors.Is(err, errEmptyFile) && !errors.Is(err, errUnparsableFile) {
		return err
	}
	corruptPath := path + ".corrupt"
	if renameErr := os.Rename(path, corruptPath); renameErr != nil {
		// 重命名失败（如被占用）则保留原文件并跳过，绝不能让坏文件阻断启动
		log.Printf("[storage] %v, 隔离失败(%v)，跳过该文件: %s", err, renameErr, path)
		return nil
	}
	log.Printf("[storage] %v, 已隔离为 %s", err, corruptPath)
	return nil
}

func writeJSONFile(path string, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
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
