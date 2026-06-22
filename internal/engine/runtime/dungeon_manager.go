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
	ID               string
	Name             string
	Description      string
	Exits            map[string]string
	NPCs             []string
	Actions          []string
	ActionLabels     map[string]string
	Checkpoint       bool
	StealthSafe      bool
	Noise            int
	ExitRequirements map[string]model.ExitRequirement
}

type PuzzleAction struct {
	ID                  string
	Label               string
	PuzzleID            string
	Operation           string
	Rooms               []string
	FromStates          []string
	NextState           string
	Message             string
	RepeatMessage       string
	FailureMessage      string
	RequiredItems       map[string]int
	RequiredDefeatNPCs  []string
	GiveItems           map[string]int
	TakeItems           map[string]int
	SetFlags            map[string]bool
	AlertDelta          int
	Choices             []ActionChoice
	DialogueNPCID       string
	DialogueLines       []string
	DialogueDoneMessage string
	MoveToRoomID        string
	RemoveNPCs          []string
	KillNPCs            []string
}

type ActionChoice struct {
	ID    string
	Label string
}

type PuzzleActionStateStatus string

const (
	PuzzleActionReady   PuzzleActionStateStatus = "ready"
	PuzzleActionRepeat  PuzzleActionStateStatus = "repeat"
	PuzzleActionBlocked PuzzleActionStateStatus = "blocked"
)

type Template struct {
	ID                  string
	Name                string
	Description         string
	Order               int
	ChainID             string
	Chapter             int
	SpawnRoomID         string
	Rooms               map[string]TemplateRoom
	ProgressOnKill      map[string]int
	Objectives          []Objective
	Rewards             model.Rewards
	UnlockConditions    []UnlockCondition
	PuzzleInitialStates map[string]string
	PuzzleActions       map[string]PuzzleAction
	Stealth             StealthConfig
	Patrols             []PatrolConfig
	TemporaryItems      []string
}

type StealthConfig struct {
	Enabled          bool
	AlertOnMove      int
	SafeRoomRecovery int
	MaxAlert         int
}

type PatrolConfig struct {
	NPCID string
	Route []string
}

type Objective struct {
	ID             string
	Type           string
	TargetID       string
	Required       int
	Description    string
	Optional       bool
	ExclusiveGroup string
	Weight         int
}

type UnlockCondition struct {
	Type     string
	TargetID string
	Value    string
	Min      int
}

type templateFile struct {
	ID                  string                      `json:"id"`
	Name                string                      `json:"name"`
	Description         string                      `json:"description"`
	Order               int                         `json:"order"`
	ChainID             string                      `json:"chainId"`
	Chapter             int                         `json:"chapter"`
	SpawnRoomID         string                      `json:"spawnRoomId"`
	Rooms               map[string]templateRoomFile `json:"rooms"`
	ProgressOnKill      map[string]int              `json:"progressOnKill"`
	Objectives          []objectiveFile             `json:"objectives"`
	Rewards             model.Rewards               `json:"rewards"`
	UnlockConditions    []unlockConditionFile       `json:"unlockConditions"`
	PuzzleInitialStates map[string]string           `json:"puzzleInitialStates"`
	PuzzleActions       map[string]puzzleActionFile `json:"puzzleActions"`
	Stealth             stealthFile                 `json:"stealth"`
	Patrols             []patrolFile                `json:"patrols"`
	TemporaryItems      []string                    `json:"temporaryItems"`
}

type templateRoomFile struct {
	Name             string                         `json:"name"`
	Description      string                         `json:"description"`
	Exits            map[string]string              `json:"exits"`
	NPCs             []string                       `json:"npcs"`
	Actions          []string                       `json:"actions"`
	Checkpoint       bool                           `json:"checkpoint"`
	StealthSafe      bool                           `json:"stealthSafe"`
	Noise            int                            `json:"noise"`
	ExitRequirements map[string]exitRequirementFile `json:"exitRequirements"`
}

type exitRequirementFile struct {
	DefeatNPCs           []string          `json:"defeatNpcs"`
	RequiredItemID       string            `json:"requiredItemId"`
	RequiredAmount       int               `json:"requiredAmount"`
	ConsumeItem          bool              `json:"consumeItem"`
	RequiredPuzzleStates map[string]string `json:"requiredPuzzleStates"`
	MaxAlert             int               `json:"maxAlert"`
	Message              string            `json:"message"`
}

type puzzleActionFile struct {
	Label              string             `json:"label"`
	PuzzleID           string             `json:"puzzleId"`
	Operation          string             `json:"operation"`
	Rooms              []string           `json:"rooms"`
	FromStates         []string           `json:"fromStates"`
	NextState          string             `json:"nextState"`
	Message            string             `json:"message"`
	RepeatMessage      string             `json:"repeatMessage"`
	FailureMessage     string             `json:"failureMessage"`
	RequiredItems      map[string]int     `json:"requiredItems"`
	RequiredDefeatNPCs []string           `json:"requiredDefeatNpcs"`
	GiveItems          map[string]int     `json:"giveItems"`
	TakeItems          map[string]int     `json:"takeItems"`
	SetFlags           map[string]bool    `json:"setFlags"`
	AlertDelta         int                `json:"alertDelta"`
	Choices            []actionChoiceFile `json:"choices"`
	DialogueNPCID      string             `json:"dialogueNpcId"`
	MoveToRoomID       string             `json:"moveToRoomId"`
	RemoveNPCs         []string           `json:"removeNpcs"`
	KillNPCs           []string           `json:"killNpcs"`
}

type dungeonDialogueScript struct {
	Lines       []string `json:"lines"`
	DoneMessage string   `json:"doneMessage"`
}

type actionChoiceFile struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type stealthFile struct {
	Enabled          bool `json:"enabled"`
	AlertOnMove      int  `json:"alertOnMove"`
	SafeRoomRecovery int  `json:"safeRoomRecovery"`
	MaxAlert         int  `json:"maxAlert"`
}

type patrolFile struct {
	NPCID string   `json:"npcId"`
	Route []string `json:"route"`
}

type objectiveFile struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	TargetID       string `json:"targetId"`
	Required       int    `json:"required"`
	Description    string `json:"description"`
	Optional       bool   `json:"optional"`
	ExclusiveGroup string `json:"exclusiveGroup"`
	Weight         int    `json:"weight"`
}

type unlockConditionFile struct {
	Type     string `json:"type"`
	TargetID string `json:"targetId"`
	Value    string `json:"value"`
	Min      int    `json:"min"`
}

type NPCState struct {
	ID                string
	TemplateID        string
	RoomID            string
	Name              string
	Title             string
	Description       string
	DialogueID        string
	Aggressive        bool
	Attributes        model.Attributes
	ShopInventory     map[string]int
	Skills            map[string]int
	IsShop            bool
	Alive             bool
	WeaknessFlag      string
	WeaknessModifiers map[string]int
	WeaknessApplied   bool
	CombatMode        string
	Defeated          bool
}

type Instance struct {
	ID                   string
	DungeonID            string
	Name                 string
	Sequence             int
	ChainID              string
	Chapter              int
	Progress             int
	OwnerPlayer          string
	EntryRoomID          string
	SpawnRoomID          string
	Rooms                map[string]model.Room
	NPCs                 map[string]*NPCState
	ProgressOnKill       map[string]int
	Objectives           []Objective
	ObjectiveProgress    map[string]int
	PuzzleStates         map[string]string
	PuzzleAttempts       map[string]int
	CheckpointRoomID     string
	AlertLevel           int
	TurnCount            int
	PatrolRoutes         map[string][]string
	Stealth              StealthConfig
	TemporaryItems       []string
	PendingChoices       map[string]bool
	PendingChoicePrompt  string
	PendingChoiceOptions []ActionChoice
	DialogueProgress     map[string]int
	DialogueDoneMessages map[string]string
	DialogueHistory      map[string]bool
	Rewards              model.Rewards
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
	dialogueScripts, err := loadDungeonDialogueScripts(filepath.Join(resourceDir, "dungeon_dialogues.json"))
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
			ID:                  tplID,
			Name:                strings.TrimSpace(df.Name),
			Description:         strings.TrimSpace(df.Description),
			Order:               df.Order,
			ChainID:             strings.TrimSpace(df.ChainID),
			Chapter:             df.Chapter,
			SpawnRoomID:         strings.TrimSpace(df.SpawnRoomID),
			Rooms:               map[string]TemplateRoom{},
			ProgressOnKill:      progressOnKill,
			Objectives:          loadObjectives(df.Objectives),
			Rewards:             normalizeDungeonRewards(df.Rewards),
			UnlockConditions:    loadUnlockConditions(df.UnlockConditions),
			PuzzleInitialStates: copyStringMap(df.PuzzleInitialStates),
			PuzzleActions:       loadPuzzleActions(df.PuzzleActions),
			Stealth: StealthConfig{
				Enabled:          df.Stealth.Enabled,
				AlertOnMove:      df.Stealth.AlertOnMove,
				SafeRoomRecovery: df.Stealth.SafeRoomRecovery,
				MaxAlert:         df.Stealth.MaxAlert,
			},
			Patrols:        loadPatrols(df.Patrols),
			TemporaryItems: append([]string{}, df.TemporaryItems...),
		}
		for actionID, script := range dialogueScripts[tplID] {
			action, exists := tpl.PuzzleActions[actionID]
			if !exists {
				continue
			}
			action.DialogueLines = append([]string{}, script.Lines...)
			action.DialogueDoneMessage = strings.TrimSpace(script.DoneMessage)
			tpl.PuzzleActions[actionID] = action
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
			exitRequirements := map[string]model.ExitRequirement{}
			for direction, requirement := range rf.ExitRequirements {
				direction = strings.TrimSpace(direction)
				if direction == "" {
					continue
				}
				exitRequirements[direction] = model.ExitRequirement{
					DefeatNPCs:           append([]string{}, requirement.DefeatNPCs...),
					RequiredItemID:       strings.TrimSpace(requirement.RequiredItemID),
					RequiredAmount:       requirement.RequiredAmount,
					ConsumeItem:          requirement.ConsumeItem,
					RequiredPuzzleStates: copyStringMap(requirement.RequiredPuzzleStates),
					MaxAlert:             requirement.MaxAlert,
					Message:              strings.TrimSpace(requirement.Message),
				}
			}
			actionLabels := map[string]string{}
			for _, actionID := range rf.Actions {
				if action, ok := tpl.PuzzleActions[actionID]; ok && action.Label != "" {
					actionLabels[actionID] = action.Label
				}
			}
			tpl.Rooms[roomID] = TemplateRoom{
				ID:               roomID,
				Name:             name,
				Description:      rf.Description,
				Exits:            exits,
				NPCs:             npcs,
				Actions:          append([]string{}, rf.Actions...),
				ActionLabels:     actionLabels,
				Checkpoint:       rf.Checkpoint,
				StealthSafe:      rf.StealthSafe,
				Noise:            rf.Noise,
				ExitRequirements: exitRequirements,
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

func loadDungeonDialogueScripts(path string) (map[string]map[string]dungeonDialogueScript, error) {
	out := map[string]map[string]dungeonDialogueScript{}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func loadPatrols(files []patrolFile) []PatrolConfig {
	out := make([]PatrolConfig, 0, len(files))
	for _, patrol := range files {
		npcID := strings.TrimSpace(patrol.NPCID)
		if npcID == "" || len(patrol.Route) < 2 {
			continue
		}
		out = append(out, PatrolConfig{NPCID: npcID, Route: append([]string{}, patrol.Route...)})
	}
	return out
}

func loadPuzzleActions(files map[string]puzzleActionFile) map[string]PuzzleAction {
	out := map[string]PuzzleAction{}
	for id, action := range files {
		id = strings.TrimSpace(id)
		if id == "" || strings.TrimSpace(action.PuzzleID) == "" {
			continue
		}
		out[id] = PuzzleAction{
			ID:                 id,
			Label:              strings.TrimSpace(action.Label),
			PuzzleID:           strings.TrimSpace(action.PuzzleID),
			Operation:          strings.TrimSpace(action.Operation),
			Rooms:              append([]string{}, action.Rooms...),
			FromStates:         append([]string{}, action.FromStates...),
			NextState:          strings.TrimSpace(action.NextState),
			Message:            strings.TrimSpace(action.Message),
			RepeatMessage:      strings.TrimSpace(action.RepeatMessage),
			FailureMessage:     strings.TrimSpace(action.FailureMessage),
			RequiredItems:      copyIntMap(action.RequiredItems),
			RequiredDefeatNPCs: append([]string{}, action.RequiredDefeatNPCs...),
			GiveItems:          copyIntMap(action.GiveItems),
			TakeItems:          copyIntMap(action.TakeItems),
			SetFlags:           copyBoolMap(action.SetFlags),
			AlertDelta:         action.AlertDelta,
			Choices:            loadActionChoices(action.Choices),
			DialogueNPCID:      strings.TrimSpace(action.DialogueNPCID),
			MoveToRoomID:       strings.TrimSpace(action.MoveToRoomID),
			RemoveNPCs:         append([]string{}, action.RemoveNPCs...),
			KillNPCs:           append([]string{}, action.KillNPCs...),
		}
	}
	return out
}

func loadActionChoices(files []actionChoiceFile) []ActionChoice {
	out := make([]ActionChoice, 0, len(files))
	for _, choice := range files {
		id := strings.TrimSpace(choice.ID)
		label := strings.TrimSpace(choice.Label)
		if id != "" && label != "" {
			out = append(out, ActionChoice{ID: id, Label: label})
		}
	}
	return out
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
			ID:             id,
			Type:           typ,
			TargetID:       targetID,
			Required:       required,
			Description:    strings.TrimSpace(obj.Description),
			Optional:       obj.Optional,
			ExclusiveGroup: strings.TrimSpace(obj.ExclusiveGroup),
			Weight:         max(1, obj.Weight),
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
		ID:                   prefix,
		DungeonID:            dungeonID,
		Name:                 tpl.Name,
		Sequence:             tpl.Order,
		ChainID:              tpl.ChainID,
		Chapter:              tpl.Chapter,
		Progress:             0,
		OwnerPlayer:          playerID,
		EntryRoomID:          entryRoomID,
		Rooms:                map[string]model.Room{},
		NPCs:                 map[string]*NPCState{},
		ProgressOnKill:       copyIntMap(tpl.ProgressOnKill),
		Objectives:           cloneObjectivesForInstance(tpl.Objectives, oldToNew),
		ObjectiveProgress:    map[string]int{},
		PuzzleStates:         copyStringMap(tpl.PuzzleInitialStates),
		PuzzleAttempts:       map[string]int{},
		PatrolRoutes:         map[string][]string{},
		Stealth:              tpl.Stealth,
		TemporaryItems:       append([]string{}, tpl.TemporaryItems...),
		PendingChoices:       map[string]bool{},
		DialogueProgress:     map[string]int{},
		DialogueDoneMessages: map[string]string{},
		DialogueHistory:      map[string]bool{},
		Rewards:              cloneRewards(tpl.Rewards),
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
			ID:               newID,
			WorldID:          "dungeon_" + dungeonID,
			WorldName:        tpl.Name,
			Name:             tr.Name,
			Description:      tr.Description,
			StealthSafe:      tr.StealthSafe,
			Noise:            tr.Noise,
			Actions:          append([]string{}, tr.Actions...),
			ActionLabels:     copyStringMap(tr.ActionLabels),
			Exits:            exits,
			ExitRequirements: cloneExitRequirements(tr.ExitRequirements),
			NPCIDs:           []string{},
		}
		if tr.Checkpoint && inst.CheckpointRoomID == "" {
			inst.CheckpointRoomID = newID
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
				ID:                npcID,
				TemplateID:        npcTemplateID,
				RoomID:            roomID,
				Name:              npcTemplate.Name,
				Title:             npcTemplate.Title,
				Description:       npcTemplate.Description,
				DialogueID:        npcTemplate.DialogueID,
				Aggressive:        npcTemplate.Aggressive,
				Attributes:        attrs,
				ShopInventory:     copyIntMap(npcTemplate.ShopInventory),
				Skills:            copyIntMap(npcTemplate.Skills),
				IsShop:            npcTemplate.IsShop,
				Alive:             true,
				WeaknessFlag:      npcTemplate.WeaknessFlag,
				WeaknessModifiers: copyIntMap(npcTemplate.WeaknessModifiers),
				CombatMode:        npcTemplate.CombatMode,
			}
			inst.NPCs[npcID] = npcState
			m.npcs[npcID] = npcState
			room.NPCIDs = append(room.NPCIDs, npcID)
		}
		inst.Rooms[roomID] = room
	}
	for _, patrol := range tpl.Patrols {
		var npcInstanceID string
		for id, npc := range inst.NPCs {
			if npc.TemplateID == patrol.NPCID {
				npcInstanceID = id
				break
			}
		}
		if npcInstanceID == "" {
			continue
		}
		route := make([]string, 0, len(patrol.Route))
		for _, baseRoomID := range patrol.Route {
			if roomID := oldToNew[baseRoomID]; roomID != "" {
				route = append(route, roomID)
			}
		}
		if len(route) >= 2 {
			inst.PatrolRoutes[npcInstanceID] = route
		}
	}

	inst.SpawnRoomID = oldToNew[tpl.SpawnRoomID]
	if inst.CheckpointRoomID == "" {
		inst.CheckpointRoomID = inst.SpawnRoomID
	}
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

	if npc.CombatMode == "defeat" {
		npc.Defeated = true
		npc.Aggressive = false
		npc.Attributes.Resources.HP = max(1, npc.Attributes.Resources.MaxHP/4)
	} else {
		npc.Alive = false
	}

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

func (m *Manager) ApplyCollectObjectiveNoLock(playerID, itemID string) int {
	instanceID, ok := m.playerInstance[playerID]
	if !ok {
		return 0
	}
	inst, ok := m.instances[instanceID]
	if !ok || inst == nil || len(inst.Objectives) == 0 {
		return 0
	}
	oldProgress := inst.Progress
	for _, obj := range inst.Objectives {
		if obj.Type != "collect" || obj.TargetID != itemID {
			continue
		}
		inst.ObjectiveProgress[obj.ID]++
	}
	m.updateObjectiveProgressNoLock(inst)
	return inst.Progress - oldProgress
}

func (m *Manager) PuzzleActionNoLock(playerID, roomID, actionID string) (*Instance, PuzzleAction, bool) {
	inst, ok := m.PlayerInstanceNoLock(playerID)
	if !ok || inst == nil {
		return nil, PuzzleAction{}, false
	}
	if _, ok := inst.Rooms[roomID]; !ok {
		return nil, PuzzleAction{}, false
	}
	tpl, ok := m.templates[inst.DungeonID]
	if !ok {
		return nil, PuzzleAction{}, false
	}
	action, ok := tpl.PuzzleActions[actionID]
	if !ok {
		return nil, PuzzleAction{}, false
	}
	if len(action.Rooms) > 0 {
		allowed := false
		for _, baseRoomID := range action.Rooms {
			if strings.HasSuffix(roomID, "_"+baseRoomID) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, PuzzleAction{}, false
		}
	}
	return inst, action, true
}

func (m *Manager) PuzzleActionStateStatusNoLock(inst *Instance, action PuzzleAction) PuzzleActionStateStatus {
	if inst == nil {
		return PuzzleActionBlocked
	}
	current := inst.PuzzleStates[action.PuzzleID]
	for _, state := range action.FromStates {
		if current == state {
			return PuzzleActionReady
		}
	}
	if len(action.FromStates) == 0 {
		return PuzzleActionReady
	}
	if action.NextState == "" {
		return PuzzleActionBlocked
	}
	if current == action.NextState || m.puzzleStateReachableNoLock(inst.DungeonID, action.PuzzleID, action.NextState, current) {
		return PuzzleActionRepeat
	}
	return PuzzleActionBlocked
}

func (m *Manager) PendingActionChoicesNoLock(inst *Instance) []ActionChoice {
	if inst == nil {
		return nil
	}
	if len(inst.PendingChoiceOptions) > 0 {
		return append([]ActionChoice{}, inst.PendingChoiceOptions...)
	}
	tpl, ok := m.templates[inst.DungeonID]
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(inst.PendingChoices))
	for id, pending := range inst.PendingChoices {
		if pending {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]ActionChoice, 0, len(ids))
	for _, id := range ids {
		action, ok := tpl.PuzzleActions[id]
		if ok {
			out = append(out, ActionChoice{ID: id, Label: action.Label})
		}
	}
	return out
}

func (m *Manager) PendingChoicePromptNoLock(inst *Instance) string {
	if inst == nil || len(inst.PendingChoices) == 0 {
		return ""
	}
	if inst.PendingChoicePrompt != "" {
		return inst.PendingChoicePrompt
	}
	tpl, ok := m.templates[inst.DungeonID]
	if !ok {
		return ""
	}
	for _, action := range tpl.PuzzleActions {
		if len(action.Choices) == 0 {
			continue
		}
		allPending := true
		for _, choice := range action.Choices {
			if !inst.PendingChoices[choice.ID] {
				allPending = false
				break
			}
		}
		if allPending {
			return action.Message
		}
	}
	return ""
}

func (m *Manager) RoomActionsForPayloadNoLock(roomID string, actions []string) []string {
	instanceID, ok := m.roomInstance[roomID]
	if !ok {
		return append([]string{}, actions...)
	}
	inst := m.instances[instanceID]
	if inst == nil {
		return append([]string{}, actions...)
	}
	tpl, ok := m.templates[inst.DungeonID]
	if !ok {
		return append([]string{}, actions...)
	}
	out := make([]string, 0, len(actions))
	for _, actionID := range actions {
		action, exists := tpl.PuzzleActions[actionID]
		if !exists || action.DialogueNPCID == "" {
			out = append(out, actionID)
		}
	}
	return out
}

func (m *Manager) ExitVisibleNoLock(roomID string, requirement model.ExitRequirement) bool {
	if len(requirement.RequiredPuzzleStates) == 0 {
		return true
	}
	instanceID, ok := m.roomInstance[roomID]
	if !ok {
		return true
	}
	inst := m.instances[instanceID]
	if inst == nil {
		return true
	}
	for puzzleID, expected := range requirement.RequiredPuzzleStates {
		if inst.PuzzleStates[puzzleID] != expected {
			return false
		}
	}
	return true
}

func (m *Manager) DialogueActionsNoLock(playerID, roomID, npcTemplateID string) []PuzzleAction {
	all := m.RoomDialogueActionsNoLock(playerID, roomID, npcTemplateID)
	out := make([]PuzzleAction, 0, len(all))
	inst, _ := m.PlayerInstanceNoLock(playerID)
	for _, action := range all {
		status := m.PuzzleActionStateStatusNoLock(inst, action)
		if status == PuzzleActionReady || status == PuzzleActionRepeat && len(inst.PendingChoices) > 0 {
			out = append(out, action)
		}
	}
	return out
}

func (m *Manager) RoomDialogueActionsNoLock(playerID, roomID, npcTemplateID string) []PuzzleAction {
	inst, ok := m.PlayerInstanceNoLock(playerID)
	if !ok || inst == nil {
		return nil
	}
	tpl, ok := m.templates[inst.DungeonID]
	if !ok {
		return nil
	}
	out := []PuzzleAction{}
	for _, action := range tpl.PuzzleActions {
		if action.DialogueNPCID != npcTemplateID {
			continue
		}
		allowedRoom := len(action.Rooms) == 0
		for _, baseRoomID := range action.Rooms {
			if strings.HasSuffix(roomID, "_"+baseRoomID) {
				allowedRoom = true
			}
		}
		if !allowedRoom {
			continue
		}
		out = append(out, action)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *Manager) InstanceRoomIDNoLock(inst *Instance, baseRoomID string) string {
	if inst == nil || baseRoomID == "" {
		return ""
	}
	for roomID := range inst.Rooms {
		if strings.HasSuffix(roomID, "_"+baseRoomID) {
			return roomID
		}
	}
	return ""
}

func (m *Manager) ApplyActionNPCEffectsNoLock(inst *Instance, roomID string, removeTemplates, killTemplates []string) {
	if inst == nil {
		return
	}
	removeSet := map[string]bool{}
	killSet := map[string]bool{}
	for _, id := range removeTemplates {
		removeSet[id] = true
	}
	for _, id := range killTemplates {
		killSet[id] = true
	}
	room := inst.Rooms[roomID]
	remaining := make([]string, 0, len(room.NPCIDs))
	for _, npcID := range room.NPCIDs {
		npc := inst.NPCs[npcID]
		if npc == nil {
			continue
		}
		if removeSet[npc.TemplateID] {
			delete(inst.NPCs, npcID)
			delete(m.npcs, npcID)
			continue
		}
		if killSet[npc.TemplateID] {
			npc.Alive = false
			npc.Defeated = false
			npc.Aggressive = false
			npc.Attributes.Resources.HP = 0
		}
		remaining = append(remaining, npcID)
	}
	room.NPCIDs = remaining
	inst.Rooms[roomID] = room
}

func (m *Manager) puzzleStateReachableNoLock(dungeonID, puzzleID, start, target string) bool {
	if start == "" || target == "" || start == target {
		return start == target
	}
	tpl, ok := m.templates[dungeonID]
	if !ok {
		return false
	}
	seen := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]
		for _, candidate := range tpl.PuzzleActions {
			if candidate.PuzzleID != puzzleID || candidate.NextState == "" {
				continue
			}
			fromCurrent := len(candidate.FromStates) == 0
			for _, from := range candidate.FromStates {
				if from == state {
					fromCurrent = true
					break
				}
			}
			if !fromCurrent || seen[candidate.NextState] {
				continue
			}
			if candidate.NextState == target {
				return true
			}
			seen[candidate.NextState] = true
			queue = append(queue, candidate.NextState)
		}
	}
	return false
}

func (m *Manager) ApplyPuzzleObjectiveNoLock(inst *Instance, puzzleID, state string) int {
	if inst == nil {
		return 0
	}
	for _, obj := range inst.Objectives {
		if obj.Type != "puzzle" || obj.TargetID != puzzleID+":"+state {
			continue
		}
		inst.ObjectiveProgress[obj.ID] = obj.Required
	}
	m.updateObjectiveProgressNoLock(inst)
	return inst.Progress
}

func (m *Manager) UpdateCheckpointNoLock(roomID string) {
	instanceID, ok := m.roomInstance[roomID]
	if !ok {
		return
	}
	inst := m.instances[instanceID]
	if inst == nil {
		return
	}
	tpl := m.templates[inst.DungeonID]
	for baseRoomID, room := range tpl.Rooms {
		if room.Checkpoint && strings.HasSuffix(roomID, "_"+baseRoomID) {
			inst.CheckpointRoomID = roomID
			return
		}
	}
}

func (m *Manager) AdvanceStealthNoLock(playerRoomID string) {
	instanceID, ok := m.roomInstance[playerRoomID]
	if !ok {
		return
	}
	inst := m.instances[instanceID]
	if inst == nil || !inst.Stealth.Enabled {
		return
	}
	inst.TurnCount++
	room := inst.Rooms[playerRoomID]
	if room.StealthSafe {
		inst.AlertLevel -= inst.Stealth.SafeRoomRecovery
	} else {
		inst.AlertLevel += inst.Stealth.AlertOnMove + room.Noise
	}
	for npcID, route := range inst.PatrolRoutes {
		npc := inst.NPCs[npcID]
		if npc == nil || !npc.Alive || len(route) < 2 {
			continue
		}
		oldRoom := inst.Rooms[npc.RoomID]
		oldRoom.NPCIDs = removeString(oldRoom.NPCIDs, npcID)
		inst.Rooms[npc.RoomID] = oldRoom
		newRoomID := route[inst.TurnCount%len(route)]
		newRoom := inst.Rooms[newRoomID]
		if !containsString(newRoom.NPCIDs, npcID) {
			newRoom.NPCIDs = append(newRoom.NPCIDs, npcID)
		}
		inst.Rooms[newRoomID] = newRoom
		npc.RoomID = newRoomID
		if newRoomID == playerRoomID {
			inst.AlertLevel++
		}
	}
	if inst.AlertLevel < 0 {
		inst.AlertLevel = 0
	}
	if inst.Stealth.MaxAlert > 0 && inst.AlertLevel > inst.Stealth.MaxAlert {
		inst.AlertLevel = inst.Stealth.MaxAlert
	}
}

func (m *Manager) updateObjectiveProgressNoLock(inst *Instance) {
	if inst == nil || len(inst.Objectives) == 0 {
		return
	}
	completed := 0
	total := 0
	groupCompleted := map[string]bool{}
	groupWeights := map[string]int{}
	for _, obj := range inst.Objectives {
		if obj.Optional {
			continue
		}
		required := obj.Required
		if required <= 0 {
			required = 1
		}
		isComplete := inst.ObjectiveProgress[obj.ID] >= required
		weight := max(1, obj.Weight)
		if obj.ExclusiveGroup != "" {
			if weight > groupWeights[obj.ExclusiveGroup] {
				groupWeights[obj.ExclusiveGroup] = weight
			}
			if isComplete {
				groupCompleted[obj.ExclusiveGroup] = true
			}
			continue
		}
		total += weight
		if isComplete {
			completed += weight
		}
	}
	for group, weight := range groupWeights {
		total += weight
		if groupCompleted[group] {
			completed += weight
		}
	}
	if total == 0 {
		inst.Progress = 100
		return
	}
	inst.Progress = completed * 100 / total
	if completed == total {
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

func (m *Manager) ProcessDungeonAggroNoLock(_ map[string]*model.Player, _ *model.CombatTickResult, _ map[string]bool) {
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

func cloneExitRequirements(in map[string]model.ExitRequirement) map[string]model.ExitRequirement {
	out := map[string]model.ExitRequirement{}
	for direction, requirement := range in {
		requirement.DefeatNPCs = append([]string{}, requirement.DefeatNPCs...)
		requirement.RequiredPuzzleStates = copyStringMap(requirement.RequiredPuzzleStates)
		out[direction] = requirement
	}
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func removeString(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
