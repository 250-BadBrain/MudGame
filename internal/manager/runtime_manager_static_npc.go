package manager

import (
	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/model"
)

func (r *RuntimeManager) staticNPCStateNoLock(npcID, roomID string) (*engruntime.NPCState, bool) {
	npc, ok := fileNpcs[npcID]
	if !ok || npc.InstanceOnly {
		return nil, false
	}
	if r.staticNPCs == nil {
		r.staticNPCs = map[string]*engruntime.NPCState{}
	}
	if state, ok := r.staticNPCs[npcID]; ok && state != nil {
		return state, true
	}
	attrs := model.NormalizeAttributes(npc.Attributes)
	state := &engruntime.NPCState{
		ID:            npc.ID,
		TemplateID:    npc.ID,
		RoomID:        roomID,
		Name:          npc.Name,
		Title:         npc.Title,
		Description:   npc.Description,
		Aggressive:    npc.Aggressive,
		Attributes:    attrs,
		ShopInventory: copyIntMap(npc.ShopInventory),
		Skills:        copyIntMap(npc.Skills),
		IsShop:        npc.IsShop,
		Alive:         true,
	}
	r.staticNPCs[npcID] = state
	return state, true
}

func (r *RuntimeManager) staticNPCInRoomNoLock(room model.Room, npcID string) (*engruntime.NPCState, bool) {
	found := false
	for _, id := range room.NPCIDs {
		if id == npcID {
			found = true
			break
		}
	}
	if !found {
		return nil, false
	}
	state, ok := r.staticNPCStateNoLock(npcID, room.ID)
	if !ok || state == nil || !state.Alive {
		return nil, false
	}
	state.RoomID = room.ID
	return state, true
}

func copyIntMap(in map[string]int) map[string]int {
	out := map[string]int{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
