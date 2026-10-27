package manager

import (
	"errors"
	"fmt"
	"mudgame/server/internal/model"
)

// The caller owns r.mu and its cloned player transaction. Escape never heals.
func (r *RuntimeManager) escapeCombatNoLock(players map[string]*model.Player, p *model.Player, itemID string) (string, error) {
	if p.State != "COMBAT" {
		return "", errors.New("烟尘粉只能在战斗中用于撤离")
	}
	if other := players[p.TargetID]; other != nil && other.State == "COMBAT" && other.TargetID == p.ID {
		r.clearCombatNoLock(other)
	}
	p.Backpack[itemID]--
	if p.Backpack[itemID] <= 0 {
		delete(p.Backpack, itemID)
	}
	message := "你撒出烟尘粉，趁视线受阻撤离了战斗。"
	if _, inDungeon := r.dungeonMgr.PlayerInstanceIDNoLock(p.ID); inDungeon {
		room, err := r.leaveDungeonNoLock(players, p)
		if err != nil {
			return "", err
		}
		if settlement, ok := room["dungeonSettlement"].(map[string]interface{}); ok {
			message += fmt.Sprintf("本次副本按%d%%完成度结算。", settlement["progress"])
		}
		return message, nil
	}
	if room, ok := fileRooms[p.RoomID]; ok {
		if world, exists := fileWorlds[room.WorldID]; exists {
			p.RoomID = world.SpawnRoomID
		}
	}
	r.clearCombatNoLock(p)
	p.StateTickCountdown = 0
	return message, r.savePlayers(players)
}
