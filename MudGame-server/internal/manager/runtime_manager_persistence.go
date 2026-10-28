package manager

import (
	"crypto/hmac"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mudgame/server/internal/model"
	"mudgame/server/internal/storage"
)

type diskRoomDynamic = storage.RoomDynamic

func (r *RuntimeManager) loadUsers() (map[string]*model.User, error) {
	if !r.usersLoaded {
		users, err := r.readUsersFromDiskNoLock()
		if err != nil {
			return nil, err
		}
		r.usersCache = users
		r.usersLoaded = true
	}
	return cloneUsersMap(r.usersCache), nil
}

func (r *RuntimeManager) saveUsers(users map[string]*model.User) error {
	r.usersCache = cloneUsersMap(users)
	r.usersLoaded = true
	r.usersDirty = true
	return nil
}

func (r *RuntimeManager) loadPlayers() (map[string]*model.Player, error) {
	if !r.playersLoaded {
		players, err := r.readPlayersFromDiskNoLock()
		if err != nil {
			return nil, err
		}
		r.playersCache = players
		r.playersLoaded = true
	}
	return clonePlayersMap(r.playersCache), nil
}

func (r *RuntimeManager) savePlayers(players map[string]*model.Player) error {
	r.playersCache = clonePlayersMap(players)
	r.playersLoaded = true
	r.playersDirty = true
	if err := r.syncDynamicRoomsNoLock(r.playersCache); err != nil {
		return err
	}
	return nil
}

func (r *RuntimeManager) loadDynamicRooms() (map[string]diskRoomDynamic, error) {
	if !r.roomsLoaded {
		out, err := r.readDynamicRoomsFromDiskNoLock()
		if err != nil {
			return nil, err
		}
		for roomID := range out {
			if strings.HasPrefix(roomID, "dg_") {
				delete(out, roomID)
			}
		}
		r.dynamicRooms = out
		r.roomsLoaded = true
	}
	cloned := cloneDiskRoomDynamicMap(r.dynamicRooms)
	for roomID := range cloned {
		if strings.HasPrefix(roomID, "dg_") {
			delete(cloned, roomID)
		}
	}
	return cloned, nil
}

func (r *RuntimeManager) loadRooms() (map[string]model.Room, error) {
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

func (r *RuntimeManager) syncDynamicRoomsNoLock(players map[string]*model.Player) error {
	dynamic, err := r.loadDynamicRooms()
	if err != nil {
		return err
	}
	for id, d := range dynamic {
		if strings.HasPrefix(id, "dg_") {
			delete(dynamic, id)
			continue
		}
		d.Players = []string{}
		dynamic[id] = d
	}
	for id, sr := range fileRooms {
		if strings.HasPrefix(id, "dg_") {
			continue
		}
		if _, ok := dynamic[id]; !ok {
			dynamic[id] = diskRoomDynamic{NPCs: append([]string{}, sr.NPCIDs...), Players: []string{}}
		}
	}
	for _, p := range players {
		if p == nil || p.RoomID == "" {
			continue
		}
		if strings.HasPrefix(p.RoomID, "dg_") {
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
		dynamic[roomID] = d
	}
	r.dynamicRooms = dynamic
	r.roomsLoaded = true
	r.roomsDirty = true
	return nil
}

func (r *RuntimeManager) readUsersFromDiskNoLock() (map[string]*model.User, error) {
	return r.store.LoadUsers()
}

func (r *RuntimeManager) writeUsersToDiskNoLock(users map[string]*model.User) error {
	return r.store.SaveUsers(users)
}

func (r *RuntimeManager) readPlayersFromDiskNoLock() (map[string]*model.Player, error) {
	players, err := r.store.LoadPlayers()
	if err != nil {
		return nil, err
	}
	for _, p := range players {
		changed, err := normalizePlayerItemAliases(p)
		if err != nil {
			return nil, err
		}
		if changed {
			r.playersDirty = true
		}
		if repairDungeonDropClaims(p) {
			r.playersDirty = true
		}
		prepared, err := prepareEnhancementReset(p)
		if err != nil {
			return nil, err
		}
		if prepared {
			r.playersDirty = true
		}
		if p.MartialVersion != model.MartialVersion {
			return nil, fmt.Errorf("角色%s尚未完成武学V2迁移；请停服运行resources/migrations/migrate_martial_v2.py", p.ID)
		}
		if p.State == "LEARNING" {
			stopLearningNoLock(p)
		}
	}
	return players, nil
}

func (r *RuntimeManager) writePlayersToDiskNoLock(players map[string]*model.Player) error {
	return r.store.SavePlayers(players)
}

func (r *RuntimeManager) readDynamicRoomsFromDiskNoLock() (map[string]diskRoomDynamic, error) {
	raw, err := r.store.LoadDynamicRooms()
	if err != nil {
		return nil, err
	}
	out := make(map[string]diskRoomDynamic, len(raw))
	for roomID, d := range raw {
		out[roomID] = diskRoomDynamic{NPCs: append([]string{}, d.NPCs...), Players: append([]string{}, d.Players...)}
	}
	return out, nil
}

func (r *RuntimeManager) writeDynamicRoomsToDiskNoLock(dynamic map[string]diskRoomDynamic) error {
	raw := make(map[string]storage.RoomDynamic, len(dynamic))
	for roomID, d := range dynamic {
		raw[roomID] = storage.RoomDynamic{NPCs: append([]string{}, d.NPCs...), Players: append([]string{}, d.Players...)}
	}
	return r.store.SaveDynamicRooms(raw)
}

func cloneUsersMap(src map[string]*model.User) map[string]*model.User {
	out := make(map[string]*model.User, len(src))
	for k, v := range src {
		if v == nil {
			continue
		}
		out[k] = &model.User{Username: v.Username, PasswordHash: v.PasswordHash, CharacterIDs: append([]string{}, v.CharacterIDs...), ReconnectTokenHash: v.ReconnectTokenHash}
	}
	return out
}

func clonePlayersMap(src map[string]*model.Player) map[string]*model.Player {
	out := make(map[string]*model.Player, len(src))
	for k, v := range src {
		if v == nil {
			continue
		}
		backpack := map[string]int{}
		for bk, bv := range v.Backpack {
			backpack[bk] = bv
		}
		equipment := map[string]string{}
		for ek, ev := range v.Equipment {
			equipment[ek] = ev
		}
		skills := map[string]int{}
		for sk, sv := range v.Skills {
			skills[sk] = sv
		}
		enhancedItems := map[string]int{}
		for ei, ev := range v.EnhancedItems {
			enhancedItems[ei] = ev
		}
		combatMoves := model.CombatMoveState{Sequence: v.CombatMoves.Sequence, Action: v.CombatMoves.Action}
		if v.CombatMoves.Skills != nil {
			combatMoves.Skills = map[string]model.CooldownWindow{}
			for mk, mv := range v.CombatMoves.Skills {
				combatMoves.Skills[mk] = mv
			}
		}
		if v.CombatMoves.Blocks != nil {
			combatMoves.Blocks = map[string]model.CooldownWindow{}
			for mk, mv := range v.CombatMoves.Blocks {
				combatMoves.Blocks[mk] = mv
			}
		}
		equippedSkills := map[string]string{}
		for ek, ev := range v.EquippedSkills {
			equippedSkills[ek] = ev
		}
		learningProgress := map[string]int{}
		for sk, sv := range v.SkillLearningProgress {
			learningProgress[sk] = sv
		}
		questStates := map[string]model.QuestState{}
		for qk, qv := range v.QuestStates {
			questStates[qk] = qv
		}
		flags := map[string]bool{}
		for fk, fv := range v.Flags {
			flags[fk] = fv
		}
		dialogueHistory := map[string]bool{}
		for dk, dv := range v.DialogueHistory {
			dialogueHistory[dk] = dv
		}
		reputations := map[string]int{}
		for rk, rv := range v.Reputations {
			reputations[rk] = rv
		}
		out[k] = &model.Player{QuickCompleteSequence: v.QuickCompleteSequence, MartialVersion: v.MartialVersion, ActiveArts: cloneStringMap(v.ActiveArts), RecoveryItemReadyAt: v.RecoveryItemReadyAt, ID: v.ID, UserID: v.UserID, CharacterName: v.CharacterName, RoomID: v.RoomID, State: v.State, StateTickCountdown: v.StateTickCountdown, LearningSkillID: v.LearningSkillID, LearningTeacherID: v.LearningTeacherID, LearningTeacherLevel: v.LearningTeacherLevel, SkillLearningProgress: learningProgress, TargetID: v.TargetID, CombatType: v.CombatType, Money: v.Money, Attributes: v.Attributes, Skills: skills, EquippedSkills: equippedSkills, EnhancedItems: enhancedItems, CombatMoves: combatMoves, Backpack: backpack, Equipment: equipment, QuestStates: questStates, Flags: flags, DialogueHistory: dialogueHistory, Reputations: reputations, Online: v.Online, WeakUntil: v.WeakUntil}
	}
	return out
}

func cloneDiskRoomDynamicMap(src map[string]diskRoomDynamic) map[string]diskRoomDynamic {
	out := make(map[string]diskRoomDynamic, len(src))
	for k, v := range src {
		npcs := append([]string{}, v.NPCs...)
		players := append([]string{}, v.Players...)
		out[k] = diskRoomDynamic{NPCs: npcs, Players: players}
	}
	return out
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
	if u == nil || u.PasswordHash == "" {
		return false
	}
	return verifyPBKDF2Password(plain, u.PasswordHash)
}

func hashReconnectToken(token string) string {
	if token == "" {
		return ""
	}
	h := hmac.New(sha256.New, []byte("mudgame-reconnect-token"))
	h.Write([]byte(token))
	dk := h.Sum(nil)
	return fmt.Sprintf("tokenv1$%s", base64.RawURLEncoding.EncodeToString(dk))
}

func hashPBKDF2Password(password string) (string, error) {
	iter := 210000
	salt := make([]byte, 16)
	if _, err := cryptorand.Read(salt); err != nil {
		// 绝不能回退为明文落盘：直接报错让注册失败
		return "", fmt.Errorf("generate salt: %w", err)
	}
	dk := pbkdf2SHA256([]byte(password), salt, iter, 32)
	return fmt.Sprintf("pbkdf2$%d$%s$%s", iter, base64.RawURLEncoding.EncodeToString(salt), base64.RawURLEncoding.EncodeToString(dk)), nil
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
