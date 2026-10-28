package manager

import (
	"mudgame/server/internal/exception"
	"mudgame/server/internal/model"
	"sort"
	"strings"
)

func heldItemCount(p *model.Player, id string) int {
	id = canonicalItemID(id)
	count := p.Backpack[id]
	def, ok := fileItems[id]
	if ok && def.EquipmentSlot != "" && p.Equipment[def.EquipmentSlot] == id {
		count++
	}
	return count
}

func contentEvidenceMet(p *model.Player, flags map[string]bool, items map[string]int) bool {
	for key, want := range flags {
		if p.Flags[key] != want {
			return false
		}
	}
	for id, n := range items {
		if n < 1 || heldItemCount(p, id) < n {
			return false
		}
	}
	return true
}

// Journal entries are derived only from discoveries actually recorded in saves.
func (r *RuntimeManager) GetJournal(playerID string) ([]map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	players, err := r.loadPlayers()
	if err != nil {
		return nil, err
	}
	p, ok := players[playerID]
	if !ok {
		return nil, exception.ErrPlayerNotFound
	}
	out := []map[string]string{}
	for _, d := range fileDialogues {
		for _, o := range d.Options {
			for flag, value := range o.SetFlags {
				if value && strings.HasPrefix(flag, "discovery_") && p.Flags[flag] {
					out = append(out, map[string]string{"id": flag, "title": fileNpcs[d.NPCID].Name + " · " + o.Text, "text": o.Response})
					break
				}
			}
		}
	}
	for _, room := range fileRooms {
		if room.SearchFlag != "" && p.Flags[room.SearchFlag] {
			out = append(out, map[string]string{"id": room.SearchFlag, "title": room.Name + " · 在地调查", "text": room.SearchMessage})
		}
	}
	if r.dungeonMgr != nil {
		for _, id := range r.dungeonMgr.DungeonTemplateIDsNoLock() {
			tpl, _ := r.dungeonMgr.DungeonTemplateNoLock(id)
			for _, action := range tpl.PuzzleActions {
				for flag, value := range action.SetFlags {
					if value && strings.HasPrefix(flag, "discovery_") && p.Flags[flag] {
						out = append(out, map[string]string{"id": flag, "title": tpl.Name + " · " + action.Label, "text": action.Message})
						break
					}
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["id"] < out[j]["id"] })
	return out, nil
}
