package app

import (
	"mudgame/server/internal/engine"
	"mudgame/server/internal/protocol"

	"github.com/gorilla/websocket"
)

// 主动招式协议入口（实现规范§5.2）：只接受moveId/battleId/expectedSequence三个参数，
// 严格拒绝客户端声明的targetId、伤害、攻速、费用或CD参数。

func (s *Server) handleGameGetCombatMoves(conn *websocket.Conn, _ protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	view, err := engine.GetCombatMovesView(s.repo, playerID)
	if err != nil {
		s.replyCommand(conn, "get_combat_moves", err, nil, nil)
		return
	}
	s.replyCommand(conn, "get_combat_moves", nil, map[string]interface{}{"view": view}, nil)
}

func (s *Server) handleGameCastCombatMove(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	moveID := toString(cmd.Args["moveId"])
	battleID := toString(cmd.Args["battleId"])
	seq := int64(toInt(cmd.Args["expectedSequence"], -1))
	res, err := engine.CastCombatMoveWithEffects(s.repo, playerID, moveID, battleID, seq)
	if err != nil {
		s.replyCommand(conn, "cast_combat_move", err, nil, nil)
		return
	}
	// 战斗视图定向回本人；房间消息按现有事件通道广播
	logs := make([]string, 0, len(res.Effects.PlayerMessages))
	for _, pm := range res.Effects.PlayerMessages {
		if pm.PlayerID == playerID {
			logs = append(logs, pm.Message)
		}
	}
	s.replyCommand(conn, "cast_combat_move", nil, map[string]interface{}{
		"result": res,
		"view":   res.View,
	}, logs)
	if res.View != nil && res.View.BattleID != "" {
		for _, m := range res.Effects.Messages {
			s.emitRoomMessageByPlayer(playerID, m.Message)
		}
		s.emitRoomUpdateByPlayer(playerID)
	}
}
