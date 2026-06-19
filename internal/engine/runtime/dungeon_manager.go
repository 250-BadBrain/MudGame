package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"mudgame/server/internal/gamedata"
	"mudgame/server/internal/model"

	"github.com/google/uuid"
)

type TemplateRoom struct {
	ID          string
	Name        string
	Description string
	Exits       map[string]string
	NPCs        []string
	Actions     []string
}

type Template struct {
	ID               string
	Name             string
	Description      string
	Order            int
	ChainID          string
	Chapter          int
	SpawnRoomID      string
	Rooms            map[string]TemplateRoom
	ProgressOnKill   map[string]int
	Objectives       []Objective
	Rewards          model.Rewards
	UnlockConditions []UnlockCondition
}

type Objective struct {
	ID          string
	Type        string
	TargetID    string
	Required    int
	Description string
}

type UnlockCondition struct {
	Type     string
	TargetID string
	Value    string
	Min      int
}

type templateFile struct {
	ID               string                      `json:"id"`
	Name             string                      `json:"name"`
	Description      string                      `json:"description"`
	Order            int                         `json:"order"`
	ChainID          string                      `json:"chainId"`
	Chapter          int                         `json:"chapter"`
	SpawnRoomID      string                      `json:"spawnRoomId"`
	Rooms            map[string]templateRoomFile `json:"rooms"`
	ProgressOnKill   map[string]int              `json:"progressOnKill"`
	Objectives       []objectiveFile             `json:"objectives"`
	Rewards          model.Rewards               `json:"rewards"`
	UnlockConditions []unlockConditionFile       `json:"unlockConditions"`
}

type templateRoomFile struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
	NPCs        []string          `json:"npcs"`
	Actions     []string          `json:"actions"`
}

type objectiveFile struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	TargetID    string `json:"targetId"`
	Required    int    `json:"required"`
	Description string `json:"description"`
}

type unlockConditionFile struct {
	Type     string `json:"type"`
	TargetID string `json:"targetId"`
	Value    string `json:"value"`
	Min      int    `json:"min"`
}

type NPCState struct {
	ID            string
	TemplateID    string
	RoomID        string
	Name          string
	Title         string
	Description   string
	Aggressive    bool
	Attributes    model.Attributes
	ShopInventory map[string]int
	Skills        map[string]int
	IsShop        bool
	Alive         bool
}

type Instance struct {
	ID                string
	DungeonID         string
	Name              string
	Sequence          int
	ChainID           string
	Chapter           int
	Progress          int
	OwnerPlayer       string
	EntryRoomID       string
	SpawnRoomID       string
	Rooms             map[string]model.Room
	NPCs              map[string]*NPCState
	ProgressOnKill    map[string]int
	Objectives        []Objective
	ObjectiveProgress map[string]int
	Rewards           model.Rewards
}

type Settlement struct {
	DungeonID         string
	DungeonName       string
	Sequence          int
	ChainID           string
	Chapter           int
	Progress          int
	RewardExperience  int
	RewardPotential   int
	RewardMoney       int
	RewardItems       map[string]int
	RewardFlags       map[string]bool
	RewardReputations map[string]int
}

type Manager struct {
	npcTemplates   map[string]model.NPC
	templates      map[string]Template
	instances      map[string]*Instance
	playerInstance map[string]string
	roomInstance   map[string]string
	npcs           map[string]*NPCState
}

func NewManager(
	npcTemplates map[string]model.NPC,
	templates map[string]Template,
	instances map[string]*Instance,
	playerInstance map[string]string,
	roomInstance map[string]string,
	npcs map[string]*NPCState,
) *Manager {
	return &Manager{
		npcTemplates:   npcTemplates,
		templates:      templates,
		instances:      instances,
		playerInstance: playerInstance,
		roomInstance:   roomInstance,
		npcs:           npcs,
	}
}

func (m *Manager) LoadTemplatesNoLock() error {
	resourceDir, err := gamedata.ResolveResourceDir()
	if err != nil {
		return err
	}
	dungeonDir := filepath.Join(resourceDir, "dungeons")
	entries, err := os.ReadDir(dungeonDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		path := filepath.Join(dungeonDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(strings.TrimSpace(string(raw))) == 0 {
			continue
		}
		var df templateFile
		if err := json.Unmarshal(raw, &df); err != nil {
			return err
		}

		tplID := strings.TrimSpace(df.ID)
		if tplID == "" {
			tplID = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		}
		if tplID == "" || len(df.Rooms) == 0 {
			continue
		}

		progressOnKill := map[string]int{}
		for npcID, value := range df.ProgressOnKill {
			id := strings.TrimSpace(npcID)
			if id == "" || value <= 0 {
				continue
			}
			progressOnKill[id] = value
		}

		tpl := Template{
			ID:               tplID,
			Name:             strings.TrimSpace(df.Name),
			Description:      strings.TrimSpace(df.Description),
			Order:            df.Order,
			ChainID:          strings.TrimSpace(df.ChainID),
			Chapter:          df.Chapter,
			SpawnRoomID:      strings.TrimSpace(df.SpawnRoomID),
			Rooms:            map[string]TemplateRoom{},
			ProgressOnKill:   progressOnKill,
			Objectives:       loadObjectives(df.Objectives),
			Rewards:          normalizeDungeonRewards(df.Rewards),
			UnlockConditions: loadUnlockConditions(df.UnlockConditions),
		}
		if tpl.Name == "" {
			tpl.Name = tplID
		}

		roomIDs := make([]string, 0, len(df.Rooms))
		for roomID := range df.Rooms {
			roomID = strings.TrimSpace(roomID)
			if roomID == "" {
				continue
			}
			roomIDs = append(roomIDs, roomID)
		}
		sort.Strings(roomIDs)

		for _, roomID := range roomIDs {
			rf := df.Rooms[roomID]
			exits := map[string]string{}
			for dir, to := range rf.Exits {
				d := strings.TrimSpace(dir)
				t := strings.TrimSpace(to)
				if d == "" || t == "" {
					continue
				}
				exits[d] = t
			}
			npcs := append([]string{}, rf.NPCs...)
			name := strings.TrimSpace(rf.Name)
			if name == "" {
				name = roomID
			}
			tpl.Rooms[roomID] = TemplateRoom{
				ID:          roomID,
				Name:        name,
				Description: rf.Description,
				Exits:       exits,
				NPCs:        npcs,
				Actions:     append([]string{}, rf.Actions...),
			}
		}

		if tpl.SpawnRoomID == "" {
			for _, roomID := range roomIDs {
				if strings.HasSuffix(roomID, "_00") {
					tpl.SpawnRoomID = roomID
					break
				}
			}
		}
		if tpl.SpawnRoomID == "" && len(roomIDs) > 0 {
			tpl.SpawnRoomID = roomIDs[0]
		}

		m.templates[tpl.ID] = tpl
	}

	m.assignTemplateOrderNoLock()
	return nil
}

func (m *Manager) assignTemplateOrderNoLock() {
	ids := make([]string, 0, len(m.templates))
	for id := range m.templates {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	nextOrder := 1
	for _, id := range ids {
		tpl := m.templates[id]
		if tpl.Order > 0 && tpl.Order >= nextOrder {
			nextOrder = tpl.Order + 1
		}
	}

	for _, id := range ids {
		tpl := m.templates[id]
		if tpl.Order <= 0 {
			tpl.Order = nextOrder
			nextOrder++
			m.templates[id] = tpl
		}
	}
}

func loadObjectives(files []objectiveFile) []Objective {
	out := make([]Objective, 0, len(files))
	for _, obj := range files {
		id := strings.TrimSpace(obj.ID)
		typ := strings.TrimSpace(obj.Type)
		targetID := strings.TrimSpace(obj.TargetID)
		if id == "" || typ == "" {
			continue
		}
		required := obj.Required
		if required <= 0 {
			required = 1
		}
		out = append(out, Objective{
			ID:          id,
			Type:        typ,
			TargetID:    targetID,
			Required:    required,
			Description: strings.TrimSpace(obj.Description),
		})
	}
	return out
}

func loadUnlockConditions(files []unlockConditionFile) []UnlockCondition {
	out := make([]UnlockCondition, 0, len(files))
	for _, cond := range files {
		typ := strings.TrimSpace(cond.Type)
		targetID := strings.TrimSpace(cond.TargetID)
		if typ == "" || targetID == "" {
			continue
		}
		out = append(out, UnlockCondition{
			Type:     typ,
			TargetID: targetID,
			Value:    strings.TrimSpace(cond.Value),
			Min:      cond.Min,
		})
	}
	return out
}

func normalizeDungeonRewards(rewards model.Rewards) model.Rewards {
	if rewards.Items == nil {
		rewards.Items = map[string]int{}
	}
	if rewards.Flags == nil {
		rewards.Flags = map[string]bool{}
	}
	if rewards.Reputations == nil {
		rewards.Reputations = map[string]int{}
	}
	return rewards
}

func cloneObjectives(in []Objective) []Objective {
	if len(in) == 0 {
		return nil
	}
	out := make([]Objective, len(in))
	copy(out, in)
	return out
}

func cloneObjectivesForInstance(in []Objective, oldToNew map[string]string) []Objective {
	out := cloneObjectives(in)
	for i := range out {
		if out[i].Type == "enter_room" {
			if newID := oldToNew[out[i].TargetID]; newID != "" {
				out[i].TargetID = newID
			}
		}
	}
	return out
}

func cloneUnlockConditions(in []UnlockCondition) []UnlockCondition {
	if len(in) == 0 {
		return nil
	}
	out := make([]UnlockCondition, len(in))
	copy(out, in)
	return out
}

func cloneRewards(in model.Rewards) model.Rewards {
	return model.Rewards{
		Experience:  in.Experience,
		Potential:   in.Potential,
		Money:       in.Money,
		Items:       copyIntMap(in.Items),
		Flags:       copyBoolMap(in.Flags),
		Reputations: copyIntMap(in.Reputations),
	}
}

func (m *Manager) CreateInstanceNoLock(playerID, dungeonID, entryRoomID string) (*Instance, error) {
	if _, ok := m.playerInstance[playerID]; ok {
		return nil, errors.New("already_in_dungeon")
	}
	tpl, ok := m.templates[dungeonID]
	if !ok {
		return nil, errors.New("dungeon_not_found")
	}
	if len(tpl.Rooms) == 0 {
		return nil, errors.New("dungeon_not_found")
	}

	prefix := "dg_" + dungeonID + "_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	oldToNew := map[string]string{}
	roomIDs := make([]string, 0, len(tpl.Rooms))
	for roomID := range tpl.Rooms {
		roomIDs = append(roomIDs, roomID)
	}
	sort.Strings(roomIDs)
	for _, oldID := range roomIDs {
		oldToNew[oldID] = prefix + "_" + oldID
	}

	inst := &Instance{
		ID:                prefix,
		DungeonID:         dungeonID,
		Name:              tpl.Name,
		Sequence:          tpl.Order,
		ChainID:           tpl.ChainID,
		Chapter:           tpl.Chapter,
		Progress:          0,
		OwnerPlayer:       playerID,
		EntryRoomID:       entryRoomID,
		Rooms:             map[string]model.Room{},
		NPCs:              map[string]*NPCState{},
		ProgressOnKill:    copyIntMap(tpl.ProgressOnKill),
		Objectives:        cloneObjectivesForInstance(tpl.Objectives, oldToNew),
		ObjectiveProgress: map[string]int{},
		Rewards:           cloneRewards(tpl.Rewards),
	}

	for _, oldID := range roomIDs {
		tr := tpl.Rooms[oldID]
		newID := oldToNew[oldID]
		exits := map[string]model.Exit{}
		for dir, oldTargetID := range tr.Exits {
			newTargetID := oldToNew[oldTargetID]
			if newTargetID == "" {
				continue
			}
			targetRoom := tpl.Rooms[oldTargetID]
			targetName := targetRoom.Name
			if targetName == "" {
				targetName = oldTargetID
			}
			exits[dir] = model.Exit{Direction: dir, TargetRoomID: newTargetID, TargetRoomName: targetName}
		}
		inst.Rooms[newID] = model.Room{
			ID:          newID,
			WorldID:     "dungeon_" + dungeonID,
			WorldName:   tpl.Name,
			Name:        tr.Name,
			Description: tr.Description,
			Actions:     append([]string{}, tr.Actions...),
			Exits:       exits,
			NPCIDs:      []string{},
		}
	}

	for _, oldID := range roomIDs {
		tr := tpl.Rooms[oldID]
		roomID := oldToNew[oldID]
		room := inst.Rooms[roomID]
		for i, npcTemplateID := range tr.NPCs {
			npcTemplate, ok := m.npcTemplates[npcTemplateID]
			if !ok {
				continue
			}
			npcID := roomID + "_n" + strconv.Itoa(i+1)
			attrs := model.NormalizeAttributes(npcTemplate.Attributes)
			npcState := &NPCState{
				ID:            npcID,
				TemplateID:    npcTemplateID,
				RoomID:        roomID,
				Name:          npcTemplate.Name,
				Title:         npcTemplate.Title,
				Description:   npcTemplate.Description,
				Aggressive:    npcTemplate.Aggressive,
				Attributes:    attrs,
				ShopInventory: copyIntMap(npcTemplate.ShopInventory),
				Skills:        copyIntMap(npcTemplate.Skills),
				IsShop:        npcTemplate.IsShop,
				Alive:         true,
			}
			inst.NPCs[npcID] = npcState
			m.npcs[npcID] = npcState
			room.NPCIDs = append(room.NPCIDs, npcID)
		}
		inst.Rooms[roomID] = room
	}

	inst.SpawnRoomID = oldToNew[tpl.SpawnRoomID]
	if inst.SpawnRoomID == "" {
		inst.SpawnRoomID = oldToNew[roomIDs[0]]
	}

	m.instances[inst.ID] = inst
	m.playerInstance[playerID] = inst.ID
	for roomID := range inst.Rooms {
		m.roomInstance[roomID] = inst.ID
	}
	m.ApplyEnterRoomObjectiveNoLock(inst.SpawnRoomID)
	return inst, nil
}

func (m *Manager) DestroyInstanceNoLock(instanceID string) {
	inst, ok := m.instances[instanceID]
	if !ok || inst == nil {
		return
	}
	for roomID := range inst.Rooms {
		delete(m.roomInstance, roomID)
	}
	for npcID := range inst.NPCs {
		delete(m.npcs, npcID)
	}
	if inst.OwnerPlayer != "" {
		delete(m.playerInstance, inst.OwnerPlayer)
	}
	delete(m.instances, instanceID)
}

func (m *Manager) DestroyPlayerInstanceNoLock(playerID string) *Instance {
	instanceID, ok := m.playerInstance[playerID]
	if !ok {
		return nil
	}
	inst := m.instances[instanceID]
	m.DestroyInstanceNoLock(instanceID)
	return inst
}

func (m *Manager) RoomByIDNoLock(roomID string, staticRooms map[string]model.Room) (model.Room, bool) {
	if instanceID, ok := m.roomInstance[roomID]; ok {
		if inst, exists := m.instances[instanceID]; exists && inst != nil {
			room, ok := inst.Rooms[roomID]
			return room, ok
		}
	}
	if room, ok := staticRooms[roomID]; ok {
		return room, true
	}
	return model.Room{}, false
}

func (m *Manager) IsDungeonRoomIDNoLock(roomID string) bool {
	_, ok := m.roomInstance[roomID]
	return ok
}

func (m *Manager) DungeonNPCByIDNoLock(npcID string) (*NPCState, bool) {
	npc, ok := m.npcs[npcID]
	if !ok || npc == nil {
		return nil, false
	}
	return npc, true
}

func (m *Manager) RemoveDungeonNPCFromRoomNoLock(inst *Instance, npcID string) {
	if inst == nil {
		return
	}
	npc, ok := inst.NPCs[npcID]
	if !ok || npc == nil {
		return
	}
	room, ok := inst.Rooms[npc.RoomID]
	if !ok {
		return
	}
	filtered := make([]string, 0, len(room.NPCIDs))
	for _, id := range room.NPCIDs {
		if id != npcID {
			filtered = append(filtered, id)
		}
	}
	room.NPCIDs = filtered
	inst.Rooms[npc.RoomID] = room
}

func (m *Manager) DefeatNPCNoLock(npcID string) (*Instance, *NPCState, int, bool) {
	npc, ok := m.npcs[npcID]
	if !ok || npc == nil || !npc.Alive {
		return nil, nil, 0, false
	}
	instanceID, ok := m.roomInstance[npc.RoomID]
	if !ok {
		return nil, nil, 0, false
	}
	inst, ok := m.instances[instanceID]
	if !ok || inst == nil {
		return nil, nil, 0, false
	}

	m.RemoveDungeonNPCFromRoomNoLock(inst, npcID)
	npc.Alive = false

	oldProgress := inst.Progress
	progressGain := 0
	if len(inst.Objectives) > 0 {
		for _, obj := range inst.Objectives {
			if obj.Type != "kill" || obj.TargetID != npc.TemplateID {
				continue
			}
			inst.ObjectiveProgress[obj.ID]++
		}
		m.updateObjectiveProgressNoLock(inst)
		progressGain = inst.Progress - oldProgress
	} else if inst.ProgressOnKill != nil {
		progressGain = inst.ProgressOnKill[npc.TemplateID]
		if progressGain > 0 {
			inst.Progress += progressGain
			if inst.Progress > 100 {
				inst.Progress = 100
			}
		}
	}

	delete(inst.NPCs, npcID)
	delete(m.npcs, npcID)
	return inst, npc, progressGain, true
}

func (m *Manager) ApplyEnterRoomObjectiveNoLock(roomID string) int {
	instanceID, ok := m.roomInstance[roomID]
	if !ok {
		return 0
	}
	inst, ok := m.instances[instanceID]
	if !ok || inst == nil || len(inst.Objectives) == 0 {
		return 0
	}
	oldProgress := inst.Progress
	for _, obj := range inst.Objectives {
		if obj.Type != "enter_room" || obj.TargetID != roomID {
			continue
		}
		inst.ObjectiveProgress[obj.ID] = obj.Required
	}
	m.updateObjectiveProgressNoLock(inst)
	return inst.Progress - oldProgress
}

func (m *Manager) updateObjectiveProgressNoLock(inst *Instance) {
	if inst == nil || len(inst.Objectives) == 0 {
		return
	}
	completed := 0
	for _, obj := range inst.Objectives {
		required := obj.Required
		if required <= 0 {
			required = 1
		}
		if inst.ObjectiveProgress[obj.ID] >= required {
			completed++
		}
	}
	inst.Progress = completed * 100 / len(inst.Objectives)
	if completed == len(inst.Objectives) {
		inst.Progress = 100
	}
}

func (m *Manager) FirstAggressiveDungeonNPCInRoomNoLock(roomID string) (*NPCState, bool) {
	instanceID, ok := m.roomInstance[roomID]
	if !ok {
		return nil, false
	}
	inst, ok := m.instances[instanceID]
	if !ok || inst == nil {
		return nil, false
	}
	room, ok := inst.Rooms[roomID]
	if !ok {
		return nil, false
	}
	for _, npcID := range room.NPCIDs {
		npc, ok := inst.NPCs[npcID]
		if !ok || npc == nil || !npc.Alive {
			continue
		}
		if npc.Aggressive {
			return npc, true
		}
	}
	return nil, false
}

func (m *Manager) DungeonBaseRewardNoLock(inst *Instance) int {
	if inst == nil {
		return 0
	}
	sequence := inst.Sequence
	if sequence <= 0 {
		sequence = 1
	}
	return sequence * 100
}

func (m *Manager) BuildDungeonSettlementNoLock(inst *Instance) Settlement {
	if inst == nil {
		return Settlement{}
	}
	progress := inst.Progress
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	base := m.DungeonBaseRewardNoLock(inst)
	rewardExp := inst.Rewards.Experience
	if rewardExp <= 0 {
		rewardExp = base
	}
	rewardPotential := inst.Rewards.Potential
	if rewardPotential <= 0 {
		rewardPotential = base
	}
	rewardMoney := inst.Rewards.Money
	rewardItems := map[string]int{}
	rewardFlags := map[string]bool{}
	rewardReputations := map[string]int{}
	if progress >= 100 {
		rewardItems = copyIntMap(inst.Rewards.Items)
		rewardFlags = copyBoolMap(inst.Rewards.Flags)
		rewardReputations = copyIntMap(inst.Rewards.Reputations)
	}
	sequence := inst.Sequence
	if sequence <= 0 {
		sequence = 1
	}
	return Settlement{
		DungeonID:         inst.DungeonID,
		DungeonName:       inst.Name,
		Sequence:          sequence,
		ChainID:           inst.ChainID,
		Chapter:           inst.Chapter,
		Progress:          progress,
		RewardExperience:  rewardExp * progress / 100,
		RewardPotential:   rewardPotential * progress / 100,
		RewardMoney:       rewardMoney * progress / 100,
		RewardItems:       rewardItems,
		RewardFlags:       rewardFlags,
		RewardReputations: rewardReputations,
	}
}

func FormatDungeonSettlementMessage(s Settlement) string {
	prefix := "副本结算"
	if s.Progress >= 100 {
		prefix = "副本完成"
	}
	return fmt.Sprintf("%s：完成度%d%%，获得%d点经验，%d点潜能。", prefix, s.Progress, s.RewardExperience, s.RewardPotential)
}

func (m *Manager) ProcessDungeonAggroNoLock(players map[string]*model.Player, result *model.CombatTickResult, updatedRoom map[string]bool) {
	for _, p := range players {
		if p == nil || p.State != "IDLE" || p.RoomID == "" {
			continue
		}
		npc, ok := m.FirstAggressiveDungeonNPCInRoomNoLock(p.RoomID)
		if !ok || npc == nil || !npc.Alive {
			continue
		}
		p.State = "COMBAT"
		p.TargetID = npc.ID
		p.CombatType = "KILL"
		result.Messages = append(result.Messages, model.RoomMessage{RoomID: p.RoomID, Message: npc.Name + "猛地扑向" + p.CharacterName + "！"})
		updatedRoom[p.RoomID] = true
	}
}

func (m *Manager) DungeonTemplateIDsNoLock() []string {
	ids := make([]string, 0, len(m.templates))
	for id := range m.templates {
		ids = append(ids, id)
	}
	return ids
}

func (m *Manager) DungeonTemplateNoLock(id string) (Template, bool) {
	tpl, ok := m.templates[id]
	return tpl, ok
}

func (m *Manager) PlayerInstanceIDNoLock(playerID string) (string, bool) {
	instanceID, ok := m.playerInstance[playerID]
	return instanceID, ok
}

func (m *Manager) PlayerInstanceNoLock(playerID string) (*Instance, bool) {
	instanceID, ok := m.playerInstance[playerID]
	if !ok {
		return nil, false
	}
	inst, ok := m.instances[instanceID]
	if !ok || inst == nil {
		return nil, false
	}
	return inst, true
}

func (m *Manager) RoomInstanceIDNoLock(roomID string) (string, bool) {
	instanceID, ok := m.roomInstance[roomID]
	return instanceID, ok
}

func (m *Manager) DungeonInstanceNoLock(instanceID string) (*Instance, bool) {
	inst, ok := m.instances[instanceID]
	return inst, ok
}

func copyIntMap(in map[string]int) map[string]int {
	out := map[string]int{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyBoolMap(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
