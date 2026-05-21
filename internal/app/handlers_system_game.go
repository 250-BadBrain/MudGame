package app

import (
	"mudgame/server/internal/engine"
	"mudgame/server/internal/protocol"

	"github.com/gorilla/websocket"
)

func (s *Server) handleSystemRegister(conn *websocket.Conn, cmd protocol.CommandMessage) {
	ss := s.session(conn)
	username := toString(cmd.Args["username"])
	password := toString(cmd.Args["password"])
	if err := s.repo.Register(username, password); err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "register", false, map[string]interface{}{"error": err.Error()}, nil))
		return
	}
	ss.userID = username
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "register", true, map[string]interface{}{"message": "注册成功", "userId": username}, []string{"欢迎加入江湖。"}))
}

func (s *Server) handleSystemLogin(conn *websocket.Conn, cmd protocol.CommandMessage) {
	ss := s.session(conn)
	username := toString(cmd.Args["username"])
	password := toString(cmd.Args["password"])
	token, err := s.repo.Login(username, password)
	if err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "login", false, map[string]interface{}{"error": err.Error()}, nil))
		return
	}
	ss.userID = username
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "login", true, map[string]interface{}{"userId": username, "reconnectToken": token}, []string{"登录成功。"}))
}

func (s *Server) handleSystemCreateCharacter(conn *websocket.Conn, cmd protocol.CommandMessage) {
	ss := s.session(conn)
	username := firstNonEmpty(toString(cmd.Args["userId"]), ss.userID)
	charName := toString(cmd.Args["characterName"])
	str := toInt(cmd.Args["strength"], 20)
	con := toInt(cmd.Args["constitution"], 20)
	agi := toInt(cmd.Args["agility"], 20)
	intel := toInt(cmd.Args["intelligence"], 20)
	id, err := s.repo.CreateCharacter(username, charName, str, con, agi, intel)
	if err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "create_character", false, map[string]interface{}{"error": err.Error()}, nil))
		return
	}
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "create_character", true, map[string]interface{}{"characterId": id}, []string{"角色创建成功。"}))
}

func (s *Server) handleSystemGetCharacters(conn *websocket.Conn, cmd protocol.CommandMessage) {
	ss := s.session(conn)
	username := firstNonEmpty(toString(cmd.Args["userId"]), ss.userID)
	chars := s.repo.ListCharacters(username)
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "get_characters", true, map[string]interface{}{"characters": chars}, nil))
}

func (s *Server) handleSystemSelectCharacter(conn *websocket.Conn, cmd protocol.CommandMessage) {
	ss := s.session(conn)
	characterID := toString(cmd.Args["characterId"])
	if !s.repo.CharacterBelongsToUser(ss.userID, characterID) {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "select_character", false, map[string]interface{}{"error": "角色不存在"}, nil))
		return
	}
	s.handleCharacterRebind(characterID, conn)
	ss.playerID = characterID
	s.repo.SetPlayerOnline(characterID, true)
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "select_character", true, map[string]interface{}{"characterId": characterID}, nil))
	s.emitPlayerStatusChanged(characterID, "恢复连接", characterID)
}

func (s *Server) handleSystemGetRoom(conn *websocket.Conn, _ protocol.CommandMessage) {
	ss := s.session(conn)
	if ss.playerID == "" {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "get_room", false, map[string]interface{}{"error": "未选择角色"}, nil))
		return
	}
	res, err := engine.GetRoomSnapshotView(s.repo, ss.playerID)
	if err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "get_room", false, map[string]interface{}{"error": err.Error()}, nil))
		return
	}
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "get_room", true, map[string]interface{}{"room": res.Room, "player": res.Player}, nil))
}

func (s *Server) handleGameGetAttributes(conn *websocket.Conn, _ protocol.CommandMessage) {
	ss := s.session(conn)
	if ss.playerID == "" {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "error", false, map[string]interface{}{"error": "未选择角色，请先登录并选择角色"}, nil))
		return
	}
	attrs, err := engine.GetAttributesView(s.repo, ss.playerID)
	s.replyCommand(conn, "get_attributes", err, map[string]interface{}{"attributes": attrs}, nil)
}

func (s *Server) handleGameGetBackpack(conn *websocket.Conn, _ protocol.CommandMessage) {
	ss := s.session(conn)
	if ss.playerID == "" {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "error", false, map[string]interface{}{"error": "未选择角色，请先登录并选择角色"}, nil))
		return
	}
	res, err := engine.GetBackpackView(s.repo, ss.playerID)
	s.replyCommand(conn, "get_backpack", err, map[string]interface{}{"items": res.Items, "money": res.Money}, nil)
}

func (s *Server) handleGameGetSkills(conn *websocket.Conn, _ protocol.CommandMessage) {
	ss := s.session(conn)
	if ss.playerID == "" {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "error", false, map[string]interface{}{"error": "未选择角色，请先登录并选择角色"}, nil))
		return
	}
	skills, err := engine.GetSkillsView(s.repo, ss.playerID)
	s.replyCommand(conn, "get_skills", err, map[string]interface{}{"skills": skills}, nil)
}

func (s *Server) handleGameGetEquipment(conn *websocket.Conn, _ protocol.CommandMessage) {
	ss := s.session(conn)
	if ss.playerID == "" {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "error", false, map[string]interface{}{"error": "未选择角色，请先登录并选择角色"}, nil))
		return
	}
	eq, err := engine.GetEquipmentView(s.repo, ss.playerID)
	s.replyCommand(conn, "get_equipment", err, map[string]interface{}{"equipment": eq}, nil)
}

func (s *Server) handleGameGetWorlds(conn *websocket.Conn, _ protocol.CommandMessage) {
	ss := s.session(conn)
	if ss.playerID == "" {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "error", false, map[string]interface{}{"error": "未选择角色，请先登录并选择角色"}, nil))
		return
	}
	s.write(conn, protocol.NewResult(protocol.TypeCommand, "get_worlds", true, map[string]interface{}{"worlds": engine.GetWorldsView(s.repo)}, nil))
}

func (s *Server) handleGameGetDungeons(conn *websocket.Conn, _ protocol.CommandMessage) {
	ss := s.session(conn)
	if ss.playerID == "" {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "error", false, map[string]interface{}{"error": "未选择角色，请先登录并选择角色"}, nil))
		return
	}
	s.write(conn, protocol.NewResult(protocol.TypeCommand, "get_dungeons", true, map[string]interface{}{"dungeons": engine.GetDungeonsView(s.repo)}, nil))
}
