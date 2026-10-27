package manager

import (
	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
)

func (r *RuntimeManager) GetRoomSnapshot(playerID string) (map[string]interface{}, map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

func (r *RuntimeManager) GetPlayerRoom(playerID string) (string, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players, err := r.loadPlayers()
	if err != nil {
		return "", "", err
	}
	staticRooms, err := r.loadRooms()
	if err != nil {
		return "", "", err
	}
	p, ok := players[playerID]
	if !ok {
		return "", "", exception.ErrPlayerNotFound
	}
	room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
	if !ok {
		return "", "", exception.ErrRoomNotFound
	}
	return room.ID, room.Name, nil
}

func (r *RuntimeManager) GetPlayerStatus(playerID string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	staticRooms, err := r.loadRooms()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	room, ok := r.roomByIDNoLock(p.RoomID, staticRooms)
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

func (r *RuntimeManager) GetRoomPayloadByID(roomID string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	roomPayload, _, err := r.buildRoomPayload(roomID, players)
	return roomPayload, err
}

func (r *RuntimeManager) GetRoomUpdate(roomID string) ([]map[string]interface{}, []map[string]interface{}, map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players, err := r.loadPlayers()
	if err != nil {
		return nil, nil, nil, err
	}
	playersInRoom, npcs, err := r.buildRoomEntities(roomID, players)
	if err != nil {
		return nil, nil, nil, err
	}
	dungeon, _ := r.dungeonPayloadByRoomIDNoLock(roomID)
	return playersInRoom, npcs, dungeon, nil
}

func (r *RuntimeManager) IsPlayerInRoom(playerID, roomID string) bool {
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
	return p.RoomID == roomID
}

func (r *RuntimeManager) buildRoomPayload(roomID string, players map[string]*model.Player) (map[string]interface{}, []map[string]interface{}, error) {
	staticRooms, err := r.loadRooms()
	if err != nil {
		return nil, nil, err
	}
	room, ok := r.roomByIDNoLock(roomID, staticRooms)
	if !ok {
		return nil, nil, exception.ErrRoomNotFound
	}
	playersInRoom, npcs, err := r.buildRoomEntities(roomID, players)
	if err != nil {
		return nil, nil, err
	}
	exits := map[string]map[string]interface{}{}
	for dir, ex := range room.Exits {
		if requirement, exists := room.ExitRequirements[dir]; exists && r.dungeonMgr != nil && !r.dungeonMgr.ExitVisibleNoLock(room.ID, requirement) {
			continue
		}
		exits[dir] = map[string]interface{}{"targetRoomId": ex.TargetRoomID, "targetRoomName": ex.TargetRoomName}
	}
	actions := room.Actions
	if r.dungeonMgr != nil {
		actions = r.dungeonMgr.RoomActionsForPayloadNoLock(room.ID, room.Actions)
	}
	roomPayload := map[string]interface{}{
		"id":            room.ID,
		"worldId":       room.WorldID,
		"worldName":     room.WorldName,
		"name":          room.Name,
		"description":   room.Description,
		"area":          room.Area,
		"tags":          room.Tags,
		"safeZone":      room.SafeZone,
		"function":      room.Function,
		"actions":       actions,
		"actionLabels":  room.ActionLabels,
		"playersInRoom": playersInRoom,
		"npcs":          npcs,
		"items":         []interface{}{},
		"exits":         exits,
	}
	if dungeon, ok := r.dungeonPayloadByRoomIDNoLock(room.ID); ok {
		roomPayload["dungeon"] = dungeon
	}
	return roomPayload, playersInRoom, nil
}

func (r *RuntimeManager) buildRoomEntities(roomID string, players map[string]*model.Player) ([]map[string]interface{}, []map[string]interface{}, error) {
	staticRooms, err := r.loadRooms()
	if err != nil {
		return nil, nil, err
	}
	room, ok := r.roomByIDNoLock(roomID, staticRooms)
	if !ok {
		return nil, nil, exception.ErrRoomNotFound
	}
	playersInRoom := make([]map[string]interface{}, 0)
	for _, p := range players {
		if p.RoomID == room.ID {
			eff := effectiveAttributesForPlayerNoLock(p)
			playersInRoom = append(playersInRoom, map[string]interface{}{
				"id":     p.ID,
				"name":   p.CharacterName,
				"title":  "",
				"state":  p.State,
				"online": p.Online,
				"attributes": map[string]interface{}{
					"currentHp":    eff.HP,
					"maxHp":        eff.MaxHP,
					"currentMp":    eff.MP,
					"trainedMaxMp": eff.TrainedMaxMP,
					"spirit":       eff.Spirit,
					"maxSpirit":    eff.MaxSpirit,
				},
			})
		}
	}
	npcs := make([]map[string]interface{}, 0, len(room.NPCIDs))
	for _, npcID := range room.NPCIDs {
		npcIDValue := npcID
		npcName := ""
		npcTitle := ""
		npcDescription := ""
		npcIsShop := false
		npcAggressive := false
		npcState := "IDLE"
		combatMode := "kill"
		dialogueID := ""
		questIDs := []string{}
		skills := map[string]int{}
		npcAttributes := map[string]interface{}{}
		if npc, ok := fileNpcs[npcID]; ok {
			staticState, _ := r.staticNPCStateNoLock(npcID, room.ID)
			if staticState != nil && !staticState.Alive {
				npcState = "DEAD"
			}
			attrs := model.NormalizeAttributes(npc.Attributes)
			if staticState != nil {
				attrs = model.NormalizeAttributes(staticState.Attributes)
			}
			eff := model.BuildEffectiveAttributes(attrs)
			npcIDValue = npc.ID
			npcName = npc.Name
			npcTitle = npc.Title
			npcDescription = npc.Description
			npcIsShop = npc.IsShop
			npcAggressive = npc.Aggressive
			if npc.CombatMode != "" {
				combatMode = npc.CombatMode
			}
			if staticNPCProtectedFromKill(npc) {
				combatMode = "spar"
				if npc.NonCombat || npc.CombatMode == "protected" {
					combatMode = "protected"
				}
			}
			dialogueID = npc.DialogueID
			questIDs = append([]string{}, npc.QuestIDs...)
			for skillID, level := range npc.Skills {
				skills[skillID] = level
			}
			npcAttributes = map[string]interface{}{
				"currentHp":    eff.HP,
				"maxHp":        eff.MaxHP,
				"currentMp":    eff.MP,
				"trainedMaxMp": eff.TrainedMaxMP,
				"spirit":       eff.Spirit,
				"maxSpirit":    eff.MaxSpirit,
			}
		} else if dnpc, ok := r.dungeonNPCByIDNoLock(npcID); ok {
			if !dnpc.Alive {
				npcState = "DEAD"
			}
			attrs := model.NormalizeAttributes(dnpc.Attributes)
			eff := model.BuildEffectiveAttributes(attrs)
			npcIDValue = dnpc.ID
			npcName = dnpc.Name
			npcTitle = dnpc.Title
			npcDescription = dnpc.Description
			npcIsShop = dnpc.IsShop
			npcAggressive = dnpc.Aggressive
			if dnpc.CombatMode != "" {
				combatMode = dnpc.CombatMode
			}
			dialogueID = dnpc.DialogueID
			for skillID, level := range dnpc.Skills {
				skills[skillID] = level
			}
			npcAttributes = map[string]interface{}{
				"currentHp":    eff.HP,
				"maxHp":        eff.MaxHP,
				"currentMp":    eff.MP,
				"trainedMaxMp": eff.TrainedMaxMP,
				"spirit":       eff.Spirit,
				"maxSpirit":    eff.MaxSpirit,
			}
		} else {
			continue
		}
		skillNames := map[string]string{}
		for skillID := range skills {
			if def, ok := fileSkills[skillID]; ok {
				skillNames[skillID] = def.Name
			}
		}
		capabilities := make([]string, 0, 2)
		if npcState != "DEAD" {
			if npcIsShop {
				capabilities = append(capabilities, "shop")
			}
			if len(skills) > 0 {
				capabilities = append(capabilities, "teach")
			}
			if dialogueID != "" {
				capabilities = append(capabilities, "talk")
			}
		} else {
			npcAggressive = false
		}
		shopable := 0
		if npcIsShop {
			shopable = 1
		}
		teachable := 0
		if len(skills) > 0 {
			teachable = 1
		}
		npcs = append(npcs, map[string]interface{}{
			"id":           npcIDValue,
			"name":         npcName,
			"title":        npcTitle,
			"description":  npcDescription,
			"attributes":   npcAttributes,
			"aggressive":   npcAggressive,
			"hostile":      npcAggressive,
			"isShop":       npcIsShop,
			"shopable":     shopable,
			"teachable":    teachable,
			"dialogueId":   dialogueID,
			"questIds":     questIDs,
			"capabilities": capabilities,
			"skills":       skills,
			"skillNames":   skillNames,
			"state":        npcState,
			"combatMode":   combatMode,
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
