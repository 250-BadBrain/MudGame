package app

import (
	"mudgame/server/internal/engine"
	"mudgame/server/internal/protocol"

	"github.com/gorilla/websocket"
)

func (s *Server) handleGameTalk(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	npcID := toString(cmd.Args["npcId"])
	res, err := engine.TalkWithView(s.repo, playerID, npcID)
	s.replyCommand(conn, "talk", err, map[string]interface{}{"dialogue": res.Dialogue}, nil)
}

func (s *Server) handleGameChooseDialogue(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	npcID := toString(cmd.Args["npcId"])
	optionID := toString(cmd.Args["optionId"])
	res, err := engine.ChooseDialogueWithEffects(s.repo, playerID, npcID, optionID)
	s.replyCommand(conn, "choose_dialogue", err, map[string]interface{}{"dialogue": res.Dialogue, "message": res.Message, "choices": res.Choices, "room": res.Room, "moved": res.Moved}, res.Logs)
}

func (s *Server) handleGameGetQuests(conn *websocket.Conn, _ protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	res, err := engine.GetQuestsView(s.repo, playerID)
	s.replyCommand(conn, "get_quests", err, map[string]interface{}{"quests": res.Quests}, nil)
}

func (s *Server) handleGameGetFactions(conn *websocket.Conn, _ protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	res, err := engine.GetFactionsView(s.repo, playerID)
	s.replyCommand(conn, "get_factions", err, map[string]interface{}{"factions": res.Factions}, nil)
}
