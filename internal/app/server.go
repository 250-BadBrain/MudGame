package app

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"mudgame/server/internal/engine"
	"mudgame/server/internal/model"
	"mudgame/server/internal/protocol"

	"github.com/gorilla/websocket"
)

type connMessage struct {
	conn    *websocket.Conn
	message string
}

type sessionState struct {
	id       string
	userID   string
	playerID string
}

type commandHandler func(conn *websocket.Conn, cmd protocol.CommandMessage)

type Server struct {
	upgrader websocket.Upgrader
	repo     engine.RuntimeRepository

	mu       sync.RWMutex
	writeMu  sync.Mutex
	sessions map[*websocket.Conn]*sessionState

	handlersSys  map[string]commandHandler
	handlersGame map[string]commandHandler
	handlersChat map[string]commandHandler
}

func NewServer(repo engine.RuntimeRepository) *Server {
	s := &Server{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		repo:         repo,
		sessions:     map[*websocket.Conn]*sessionState{},
		handlersSys:  map[string]commandHandler{},
		handlersGame: map[string]commandHandler{},
		handlersChat: map[string]commandHandler{},
	}
	s.registerCommandHandlers()
	go s.runBackgroundTicker()
	return s
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ws", s.handleWS)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	s.mu.Lock()
	s.sessions[conn] = &sessionState{id: conn.RemoteAddr().String()}
	s.mu.Unlock()

	defer func() {
		s.handleSessionClosed(conn)
		s.mu.Lock()
		delete(s.sessions, conn)
		s.mu.Unlock()
	}()

	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var cmd protocol.CommandMessage
		if err := json.Unmarshal(payload, &cmd); err != nil {
			s.write(conn, protocol.NewResult(protocol.TypeSystem, "error", false, map[string]interface{}{"error": "无效请求"}, nil))
			continue
		}

		if cmd.Args == nil {
			cmd.Args = map[string]interface{}{}
		}

		s.dispatch(conn, cmd)
	}
}

func (s *Server) dispatch(conn *websocket.Conn, cmd protocol.CommandMessage) {
	var handlers map[string]commandHandler
	switch cmd.Type {
	case protocol.TypeSystem:
		handlers = s.handlersSys
	case protocol.TypeCommand:
		handlers = s.handlersGame
	case protocol.TypeChat:
		handlers = s.handlersChat
	default:
		s.writeUnknownCommand(conn, cmd)
		return
	}

	h, ok := handlers[cmd.Subtype]
	if !ok {
		s.writeUnknownCommand(conn, cmd)
		return
	}
	h(conn, cmd)
}

func (s *Server) handleSystem(conn *websocket.Conn, cmd protocol.CommandMessage) {
	s.writeUnknownCommand(conn, cmd)
}

func (s *Server) handleCommand(conn *websocket.Conn, cmd protocol.CommandMessage) {
	s.writeUnknownCommand(conn, cmd)
}

func (s *Server) handleChat(conn *websocket.Conn, cmd protocol.CommandMessage) {
	if cmd.Subtype == "public" {
		s.handleChatPublic(conn, cmd)
		return
	}
	s.writeUnknownCommand(conn, cmd)
}

func (s *Server) replyCommand(conn *websocket.Conn, subtype string, err error, okData map[string]interface{}, logs []string) {
	if err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeCommand, subtype, false, map[string]interface{}{"error": s.localizedError(err)}, nil))
		return
	}
	s.write(conn, protocol.NewResult(protocol.TypeCommand, subtype, true, okData, logs))
}

func (s *Server) session(conn *websocket.Conn) *sessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.sessions[conn]
	if !ok {
		ss = &sessionState{id: conn.RemoteAddr().String()}
		s.sessions[conn] = ss
	}
	return ss
}

func (s *Server) broadcast(result protocol.GameResult) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for c := range s.sessions {
		s.write(c, result)
	}
}

func (s *Server) handleSessionClosed(conn *websocket.Conn) {
	ss := s.session(conn)
	if ss.playerID != "" {
		s.repo.SetPlayerOnline(ss.playerID, false)
		s.emitPlayerStatusChanged(ss.playerID, "断开了连接", ss.playerID)
	}
}

func (s *Server) handleCharacterRebind(playerID string, newConn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c, ss := range s.sessions {
		if c != newConn && ss.playerID == playerID {
			replaced := protocol.NewResult(protocol.TypeSystem, "session_replaced", false, map[string]interface{}{"error": "该角色已在另一个窗口中被选择，当前连接已断开"}, nil)
			s.write(c, replaced)
			_ = c.Close()
			ss.playerID = ""
		}
	}
}

func (s *Server) emitPlayerLeft(playerID, roomID, roomName string) {
	if roomID == "" {
		return
	}
	status, err := engine.GetPlayerStatusView(s.repo, playerID)
	if err != nil {
		return
	}
	room, err := engine.GetRoomPayloadView(s.repo, roomID)
	if err != nil {
		return
	}
	payload := protocol.NewResult(protocol.TypeEvent, "player_left", true, map[string]interface{}{
		"playerId":   status["playerId"],
		"playerName": status["playerName"],
		"roomId":     roomID,
		"roomName":   roomName,
		"room":       room,
	}, nil)
	s.broadcastToRoom(roomID, payload, playerID)
}

func (s *Server) emitPlayerEntered(playerID, roomID, roomName string) {
	if roomID == "" {
		return
	}
	status, err := engine.GetPlayerStatusView(s.repo, playerID)
	if err != nil {
		return
	}
	room, err := engine.GetRoomPayloadView(s.repo, roomID)
	if err != nil {
		return
	}
	payload := protocol.NewResult(protocol.TypeEvent, "player_entered", true, map[string]interface{}{
		"playerId":   status["playerId"],
		"playerName": status["playerName"],
		"roomId":     roomID,
		"roomName":   roomName,
		"room":       room,
	}, nil)
	s.broadcastToRoom(roomID, payload, playerID)
}

func (s *Server) emitPlayerStatusChanged(playerID, message, excludePlayerID string) {
	status, err := engine.GetPlayerStatusView(s.repo, playerID)
	if err != nil {
		return
	}
	roomID := toString(status["roomId"])
	payload := protocol.NewResult(protocol.TypeEvent, "player_status_changed", true, map[string]interface{}{
		"playerId":   status["playerId"],
		"playerName": status["playerName"],
		"status":     status["status"],
		"message":    message,
	}, nil)
	s.broadcastToRoom(roomID, payload, excludePlayerID)
}

func (s *Server) emitRoomUpdate(roomID string) {
	if roomID == "" {
		return
	}
	res, err := engine.GetRoomUpdateView(s.repo, roomID)
	if err != nil {
		return
	}
	payload := protocol.NewResult(protocol.TypeEvent, "room_update", true, map[string]interface{}{
		"players": res.Players,
		"npcs":    res.NPCs,
	}, nil)
	s.broadcastToRoom(roomID, payload, "")
}

func (s *Server) emitRoomUpdateByPlayer(playerID string) {
	roomID, _, err := engine.GetPlayerRoomView(s.repo, playerID)
	if err != nil {
		return
	}
	s.emitRoomUpdate(roomID)
}

func (s *Server) emitRoomMessageByPlayer(playerID, message string) {
	roomID, _, err := engine.GetPlayerRoomView(s.repo, playerID)
	if err != nil {
		return
	}
	s.emitRoomMessageToRoom(roomID, message)
}

func (s *Server) emitRoomMessageToOthersByPlayer(playerID, message string) {
	if message == "" {
		return
	}
	roomID, _, err := engine.GetPlayerRoomView(s.repo, playerID)
	if err != nil {
		return
	}
	payload := protocol.NewResult(protocol.TypeEvent, "room_message", true, map[string]interface{}{
		"message": message,
	}, nil)
	s.broadcastToRoom(roomID, payload, playerID)
}

func (s *Server) emitScopedActionMessage(playerID, selfMessage, othersMessage string) {
	if selfMessage == "" && othersMessage == "" {
		return
	}
	roomID, _, err := engine.GetPlayerRoomView(s.repo, playerID)
	if err != nil {
		return
	}
	recipients := s.connectionsWithScopedMessage(roomID, playerID, selfMessage, othersMessage)
	for _, recipient := range recipients {
		if recipient.message == "" {
			continue
		}
		s.write(recipient.conn, protocol.NewResult(protocol.TypeEvent, "room_message", true, map[string]interface{}{
			"message": recipient.message,
		}, nil))
	}
}

func (s *Server) connectionsWithScopedMessage(roomID, actorPlayerID, selfMessage, othersMessage string) []connMessage {
	if roomID == "" {
		return nil
	}
	out := make([]connMessage, 0)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for conn, ss := range s.sessions {
		if ss.playerID == "" {
			continue
		}
		if !engine.IsPlayerInRoomView(s.repo, ss.playerID, roomID) {
			continue
		}
		msg := othersMessage
		if ss.playerID == actorPlayerID {
			msg = selfMessage
		}
		if msg == "" {
			continue
		}
		out = append(out, connMessage{conn: conn, message: msg})
	}
	return out
}

func (s *Server) emitRoomMessageToRoom(roomID, message string) {
	payload := protocol.NewResult(protocol.TypeEvent, "room_message", true, map[string]interface{}{
		"message": message,
	}, nil)
	s.broadcastToRoom(roomID, payload, "")
}

func (s *Server) broadcastToRoom(roomID string, result protocol.GameResult, excludePlayerID string) {
	conns := s.connectionsByRoom(roomID, excludePlayerID)
	for _, c := range conns {
		s.write(c, result)
	}
}

func (s *Server) connectionsByRoom(roomID, excludePlayerID string) []*websocket.Conn {
	if roomID == "" {
		return nil
	}
	out := make([]*websocket.Conn, 0)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for c, ss := range s.sessions {
		if ss.playerID == "" {
			continue
		}
		if excludePlayerID != "" && ss.playerID == excludePlayerID {
			continue
		}
		if engine.IsPlayerInRoomView(s.repo, ss.playerID, roomID) {
			out = append(out, c)
		}
	}
	return out
}

func (s *Server) write(conn *websocket.Conn, result protocol.GameResult) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := conn.WriteJSON(result); err != nil {
		log.Printf("write failed: %v", err)
	}
}

func toString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func (s *Server) runBackgroundTicker() {
	statusTicker := time.NewTicker(1 * time.Second)
	combatTicker := time.NewTicker(2 * time.Second)
	defer statusTicker.Stop()
	defer combatTicker.Stop()

	for {
		select {
		case <-statusTicker.C:
			if statusRes, err := engine.ProcessStatusTickWithEffects(s.repo); err == nil {
				s.emitTickResult(statusRes)
			}
		case <-combatTicker.C:
			if combatRes, err := engine.ProcessCombatTickWithEffects(s.repo); err == nil {
				s.emitTickResult(combatRes)
			}
		}
	}
}

func (s *Server) emitTickResult(res model.CombatTickResult) {
	for _, pm := range res.PlayerMessages {
		if pm.PlayerID == "" || pm.Message == "" {
			continue
		}
		s.emitPersonalRoomMessage(pm.PlayerID, pm.Message)
	}
	for _, msg := range res.Messages {
		if msg.RoomID == "" || msg.Message == "" {
			continue
		}
		s.emitRoomMessageToRoom(msg.RoomID, msg.Message)
	}
	for _, roomID := range res.UpdatedRoomIDs {
		if roomID == "" {
			continue
		}
		s.emitRoomUpdate(roomID)
	}
}

func (s *Server) emitPersonalRoomMessage(playerID, message string) {
	if playerID == "" || message == "" {
		return
	}
	s.mu.RLock()
	conns := make([]*websocket.Conn, 0)
	for conn, ss := range s.sessions {
		if ss.playerID == playerID {
			conns = append(conns, conn)
		}
	}
	s.mu.RUnlock()
	if len(conns) == 0 {
		return
	}
	result := protocol.NewResult(protocol.TypeEvent, "room_message", true, map[string]interface{}{"message": message}, nil)
	for _, conn := range conns {
		s.write(conn, result)
	}
}

func toInt(v interface{}, fallback int) int {
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return fallback
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
