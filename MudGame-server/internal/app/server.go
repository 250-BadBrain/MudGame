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

const (
	// 单条上行消息的最大字节数，超限连接将被服务端断开
	maxMessageBytes = 1 << 20
	// 单次下行写的超时，防止僵死客户端无限期占住全局写锁
	writeWait = 10 * time.Second
)

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
	// 限制单条消息大小，防止恶意超大帧耗尽内存
	conn.SetReadLimit(maxMessageBytes)

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

func (s *Server) replyCommand(conn *websocket.Conn, subtype string, err error, okData map[string]interface{}, logs []string) {
	if err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeCommand, subtype, false, map[string]interface{}{"error": s.localizedError(err)}, nil))
		return
	}
	s.write(conn, protocol.NewResult(protocol.TypeCommand, subtype, true, okData, logs))
}

// sessionState 的字段会被各连接读循环与后台 ticker 并发读写，
// 读写必须经以下带锁访问器，避免无锁写与 RLock 读形成数据竞争。

func (s *Server) sessionUserID(conn *websocket.Conn) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ss, ok := s.sessions[conn]; ok {
		return ss.userID
	}
	return ""
}

func (s *Server) sessionPlayerID(conn *websocket.Conn) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ss, ok := s.sessions[conn]; ok {
		return ss.playerID
	}
	return ""
}

// bindSessionUser 将会话切换到指定账号，并解除该会话上残留的旧角色绑定，
// 防止换账号登录后继续以他人角色身份执行指令。
func (s *Server) bindSessionUser(conn *websocket.Conn, userID string) {
	s.mu.Lock()
	ss, ok := s.sessions[conn]
	if !ok {
		s.mu.Unlock()
		return
	}
	oldPlayer := ss.playerID
	ss.userID = userID
	ss.playerID = ""
	s.mu.Unlock()
	if oldPlayer != "" {
		s.repo.SetPlayerOnline(oldPlayer, false)
		s.emitPlayerStatusChanged(oldPlayer, "断开了连接", oldPlayer)
	}
}

// setSessionPlayerID 绑定当前会话的角色，返回旧的角色 ID（可为空）。
func (s *Server) setSessionPlayerID(conn *websocket.Conn, playerID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss, ok := s.sessions[conn]
	if !ok {
		return ""
	}
	old := ss.playerID
	ss.playerID = playerID
	return old
}

func (s *Server) handleSessionClosed(conn *websocket.Conn) {
	playerID := s.sessionPlayerID(conn)
	if playerID != "" {
		s.repo.SetPlayerOnline(playerID, false)
		s.emitPlayerStatusChanged(playerID, "断开了连接", playerID)
	}
}

func (s *Server) handleCharacterRebind(playerID string, newConn *websocket.Conn) {
	// 先在锁内摘除旧会话的角色绑定并收集连接，再在锁外写消息/关闭，
	// 避免向可能停滞的连接写入时长期持有 sessions 写锁。
	type victim struct {
		conn *websocket.Conn
	}
	victims := make([]victim, 0)
	s.mu.Lock()
	for c, ss := range s.sessions {
		if c != newConn && ss.playerID == playerID {
			ss.playerID = ""
			victims = append(victims, victim{conn: c})
		}
	}
	s.mu.Unlock()
	for _, v := range victims {
		replaced := protocol.NewResult(protocol.TypeSystem, "session_replaced", false, map[string]interface{}{"error": "该角色已在另一个窗口中被选择，当前连接已断开"}, nil)
		s.write(v.conn, replaced)
		_ = v.conn.Close()
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
		"online":     status["online"],
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
	payload := map[string]interface{}{
		"players": res.Players,
		"npcs":    res.NPCs,
	}
	if res.Dungeon != nil {
		payload["dungeon"] = res.Dungeon
	}
	result := protocol.NewResult(protocol.TypeEvent, "room_update", true, payload, nil)
	s.broadcastToRoom(roomID, result, "")
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
	// 写超时兜底：客户端停止读取时 WriteJSON 会永久阻塞并占住全局写锁，冻结全服下发
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteJSON(result); err != nil {
		log.Printf("write failed: %v", err)
	}
	_ = conn.SetWriteDeadline(time.Time{})
}

func toString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func (s *Server) runBackgroundTicker() {
	statusTicker := time.NewTicker(1 * time.Second)
	combatTicker := time.NewTicker(2 * time.Second)
	moveTicker := time.NewTicker(100 * time.Millisecond)
	defer statusTicker.Stop()
	defer combatTicker.Stop()
	defer moveTicker.Stop()

	for {
		select {
		case <-moveTicker.C:
			// 100ms调度仅处理NPC意图的到点释放，不在该精度结算普攻/流血（实现规范§8）
			if moveRes, err := engine.ProcessCombatMovesTickWithEffects(s.repo); err != nil {
				log.Printf("combat moves tick failed: %v", err)
			} else {
				s.emitTickResult(moveRes)
			}
		case <-statusTicker.C:
			if statusRes, err := engine.ProcessStatusTickWithEffects(s.repo); err != nil {
				log.Printf("status tick failed: %v", err)
			} else {
				s.emitTickResult(statusRes)
			}
		case <-combatTicker.C:
			if combatRes, err := engine.ProcessCombatTickWithEffects(s.repo); err != nil {
				log.Printf("combat tick failed: %v", err)
			} else {
				s.emitTickResult(combatRes)
			}
		}
	}
}

func (s *Server) emitTickResult(res model.CombatTickResult) {
	// 主动招式私有视图：只定向发本人，不混入公共房间payload（实现规范§5）
	for _, upd := range res.CombatMoves {
		if upd.PlayerID == "" {
			continue
		}
		s.emitPersonalEvent(upd.PlayerID, "combat_moves", map[string]interface{}{"view": upd.View})
	}
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

// emitPersonalEvent 向指定玩家的全部连接定向推送一条事件。
func (s *Server) emitPersonalEvent(playerID, subtype string, data map[string]interface{}) {
	if playerID == "" || subtype == "" {
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
	result := protocol.NewResult(protocol.TypeEvent, subtype, true, data, nil)
	for _, conn := range conns {
		s.write(conn, result)
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
