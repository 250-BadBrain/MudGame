package manager

import (
	"errors"

	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"

	"github.com/google/uuid"
)

func (r *RuntimeManager) Register(username, password string) error {
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

func (r *RuntimeManager) Login(username, password string) (string, error) {
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
	token := uuid.NewString()
	u.ReconnectTokenHash = hashReconnectToken(token)
	if err := r.saveUsers(users); err != nil {
		return "", err
	}
	return token, nil
}

func (r *RuntimeManager) IsReconnectTokenValid(username, token string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users, err := r.loadUsers()
	if err != nil {
		return false
	}
	u, ok := users[username]
	if !ok {
		return false
	}
	if token == "" || u.ReconnectTokenHash == "" {
		return false
	}
	return u.ReconnectTokenHash == hashReconnectToken(token)
}

func (r *RuntimeManager) DeleteAccount(username, password string) error {
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

func (r *RuntimeManager) CreateCharacter(username, name string, str, con, agi, perception, willpower int) (string, error) {
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
	if str+con+agi+perception+willpower != 80 {
		return "", errors.New("invalid_attribute_sum")
	}

	players, err := r.loadPlayers()
	if err != nil {
		return "", err
	}

	for _, p := range players {
		if p != nil && p.CharacterName == name {
			return "", errors.New("character_name_exists")
		}
	}

	id := uuid.NewString()
	players[id] = &model.Player{
		ID:                    id,
		UserID:                username,
		CharacterName:         name,
		RoomID:                "ly_00",
		State:                 "IDLE",
		Money:                 5000,
		Attributes:            model.NewPlayerAttributes(str, con, agi, perception, willpower),
		Skills:                map[string]int{"basic_unarmed": 1, "basic_parry": 1, "basic_neigong": 1, "basic_qinggong": 1},
		SkillLearningProgress: map[string]int{},
		Backpack:              map[string]int{"iron_pickaxe": 1},
		Equipment:             map[string]string{},
		QuestStates:           map[string]model.QuestState{},
		Flags:                 map[string]bool{},
		DialogueHistory:       map[string]bool{},
		Reputations:           map[string]int{},
		Online:                false,
	}
	players[id].Attributes.Resources.HP = effectiveAttributesForPlayerNoLock(players[id]).MaxHP
	u.CharacterIDs = append(u.CharacterIDs, id)

	if err := r.savePlayers(players); err != nil {
		return "", err
	}
	if err := r.saveUsers(users); err != nil {
		return "", err
	}
	return id, nil
}

func (r *RuntimeManager) ListCharacters(username string) []map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

func (r *RuntimeManager) CharacterBelongsToUser(username, playerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

func (r *RuntimeManager) DeleteCharacter(username, playerID string) error {
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

func (r *RuntimeManager) SetPlayerOnline(playerID string, online bool) {
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
