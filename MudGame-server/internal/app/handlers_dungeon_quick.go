package app

import (
	"fmt"
	"github.com/gorilla/websocket"
	"mudgame/server/internal/protocol"
	"sort"
	"strings"
)

func (s *Server) handleGameQuickCompleteDungeon(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	result, err := s.repo.QuickCompleteDungeon(playerID, toString(cmd.Args["dungeonId"]), toInt(cmd.Args["expectedSequence"], -1))
	if err != nil {
		s.replyCommand(conn, "quick_complete_dungeon", err, nil, nil)
		return
	}
	names, _ := result["rewardItemNames"].(map[string]string)
	items := []string{}
	if rewards, ok := result["rewardItems"].(map[string]int); ok {
		ids := make([]string, 0, len(rewards))
		for id := range rewards {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			name := id
			if names[id] != "" {
				name = names[id]
			}
			items = append(items, fmt.Sprintf("%s×%d", name, rewards[id]))
		}
	}
	text := fmt.Sprintf("快速完成《%s》：获得%d经验、%d潜能、%d铜钱", toString(result["dungeonName"]), toInt(result["rewardExperience"], 0), toInt(result["rewardPotential"], 0), toInt(result["rewardMoney"], 0))
	if len(items) > 0 {
		text += "，" + strings.Join(items, "、")
	}
	s.replyCommand(conn, "quick_complete_dungeon", nil, map[string]interface{}{"settlement": result}, []string{text + "。"})
}
