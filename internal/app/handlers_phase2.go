package app

import (
	"fmt"

	"mudgame/server/internal/exception"
	"mudgame/server/internal/protocol"

	"github.com/gorilla/websocket"
)

func (s *Server) requireSelectedPlayer(conn *websocket.Conn) (string, bool) {
	ss := s.session(conn)
	if ss.playerID == "" {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "error", false, map[string]interface{}{"error": exception.ErrPlayerNotSelected.Error()}, nil))
		return "", false
	}
	return ss.playerID, true
}

func (s *Server) handleSystemDeleteAccount(conn *websocket.Conn, cmd protocol.CommandMessage) {
	ss := s.session(conn)
	username := toString(cmd.Args["username"])
	password := toString(cmd.Args["password"])
	if err := s.store.DeleteAccount(username, password); err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "delete_account", false, map[string]interface{}{"error": err.Error()}, nil))
		return
	}
	if ss.userID == username {
		ss.userID = ""
		ss.playerID = ""
	}
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "delete_account", true, map[string]interface{}{"message": "账号已删除"}, nil))
}

func (s *Server) handleSystemDeleteCharacter(conn *websocket.Conn, cmd protocol.CommandMessage) {
	ss := s.session(conn)
	characterID := toString(cmd.Args["characterId"])
	if err := s.store.DeleteCharacter(ss.userID, characterID); err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "delete_character", false, map[string]interface{}{"error": err.Error()}, nil))
		return
	}
	if ss.playerID == characterID {
		ss.playerID = ""
	}
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "delete_character", true, map[string]interface{}{"message": "角色已删除"}, nil))
}

func (s *Server) handleSystemReconnect(conn *websocket.Conn, cmd protocol.CommandMessage) {
	ss := s.session(conn)
	res, err := s.store.Reconnect(
		toString(cmd.Args["userId"]),
		ss.userID,
		toString(cmd.Args["reconnectToken"]),
		toString(cmd.Args["playerId"]),
	)
	if err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeSystem, "reconnect", false, map[string]interface{}{"error": err.Error()}, nil))
		return
	}
	ss.userID = res.UserID
	if res.PlayerID != "" {
		s.handleCharacterRebind(res.PlayerID, conn)
		ss.playerID = res.PlayerID
		s.store.SetPlayerOnline(res.PlayerID, true)
		s.emitPlayerStatusChanged(res.PlayerID, "恢复连接", res.PlayerID)
	}
	results := map[string]interface{}{"message": "重新连接成功"}
	if ss.playerID != "" {
		results["playerId"] = ss.playerID
	}
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "reconnect", true, results, nil))
}

func (s *Server) handleSystemDeselectCharacter(conn *websocket.Conn, _ protocol.CommandMessage) {
	ss := s.session(conn)
	if ss.playerID != "" {
		oldPlayerID := ss.playerID
		s.store.SetPlayerOnline(ss.playerID, false)
		ss.playerID = ""
		s.emitPlayerStatusChanged(oldPlayerID, "断开了连接", oldPlayerID)
	}
	s.write(conn, protocol.NewResult(protocol.TypeSystem, "deselect_character", true, map[string]interface{}{"message": "已返回角色选择"}, nil))
}

func (s *Server) handleGameEnterWorld(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	worldID := toString(cmd.Args["worldId"])
	res, err := s.store.EnterWorldWithTransition(playerID, worldID)
	worldName := toString(res.Room["worldName"])
	if worldName == "" {
		worldName = worldID
	}
	s.replyCommand(conn, "enter_world", err, map[string]interface{}{"room": res.Room, "closeShop": true}, []string{fmt.Sprintf("你进入了%s。", worldName)})
	if err != nil {
		return
	}
	s.emitPlayerLeft(playerID, res.FromRoomID, res.FromRoomName)
	s.emitPlayerEntered(playerID, res.ToRoomID, res.ToRoomName)
	s.emitRoomUpdate(res.FromRoomID)
	s.emitRoomUpdate(res.ToRoomID)
}

func (s *Server) handleGameGo(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	direction := toString(cmd.Args["direction"])
	res, err := s.store.MoveWithTransition(playerID, direction)
	s.replyCommand(conn, "go", err, map[string]interface{}{"room": res.Room, "closeShop": true}, nil)
	if err != nil {
		return
	}
	s.emitPlayerLeft(playerID, res.FromRoomID, res.FromRoomName)
	s.emitPlayerEntered(playerID, res.ToRoomID, res.ToRoomName)
	s.emitRoomUpdate(res.FromRoomID)
	s.emitRoomUpdate(res.ToRoomID)
}

func (s *Server) handleGameHeal(conn *websocket.Conn, _ protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	res, err := s.store.HealWithEffects(playerID)
	s.replyCommand(conn, "heal", err, map[string]interface{}{}, nil)
	if err == nil && res.RoomMessage != "" {
		s.emitRoomMessageToOthersByPlayer(playerID, res.RoomMessage)
	}
	if err == nil && res.NeedRoomUpdate {
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleGameMeditate(conn *websocket.Conn, _ protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	res, err := s.store.MeditateWithEffects(playerID)
	s.replyCommand(conn, "meditate", err, map[string]interface{}{"message": res.Message}, nil)
	if err == nil && res.RoomMessage != "" {
		s.emitRoomMessageToOthersByPlayer(playerID, res.RoomMessage)
	}
	if err == nil && res.NeedRoomUpdate {
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleGameStop(conn *websocket.Conn, _ protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	res := s.store.StopWithEffects(playerID)
	s.write(conn, protocol.NewResult(protocol.TypeCommand, "stop", true, map[string]interface{}{}, nil))
	if res.Message != "" || res.RoomMessage != "" {
		s.emitScopedActionMessage(playerID, res.Message, res.RoomMessage)
	}
	if res.NeedRoomUpdate {
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleGameGetShop(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	npcID := toString(cmd.Args["npcId"])
	res, err := s.store.GetShopView(playerID, npcID)
	s.replyCommand(conn, "get_shop", err, map[string]interface{}{"shopItems": res.Items, "npcId": res.NPCID, "npcName": res.NPCName, "npcTitle": res.NPCTitle, "shopOpen": true, "hideRoomEntities": true}, nil)
}

func (s *Server) handleGameBuy(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	npcID := toString(cmd.Args["npcId"])
	itemID := toString(cmd.Args["itemId"])
	amount := toInt(cmd.Args["amount"], 1)
	res, err := s.store.BuyWithReceipt(playerID, npcID, itemID, amount)
	s.replyCommand(conn, "buy", err, map[string]interface{}{"money": res.Money, "boughtItemId": res.BoughtItemID, "boughtAmount": res.BoughtAmount}, res.Logs)
}

func (s *Server) handleGameDiscardItem(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	itemID := toString(cmd.Args["itemId"])
	amount := toInt(cmd.Args["amount"], 1)
	msg, err := s.store.DiscardWithMessage(playerID, itemID, amount)
	s.replyCommand(conn, "discard_item", err, map[string]interface{}{"message": msg}, nil)
}

func (s *Server) handleGameSell(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	itemID := toString(cmd.Args["itemId"])
	amount := toInt(cmd.Args["amount"], 1)
	res, err := s.store.SellWithReceipt(playerID, itemID, amount)
	s.replyCommand(conn, "sell", err, map[string]interface{}{"money": res.Money, "gain": res.Gain, "message": res.Message}, nil)
}

func (s *Server) handleGameEquip(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	itemID := toString(cmd.Args["itemId"])
	res, err := s.store.EquipWithEffects(playerID, itemID)
	s.replyCommand(conn, "equip", err, map[string]interface{}{"message": res.Message}, nil)
	if err == nil && res.NeedRoomUpdate {
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleGameUnequip(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	slot := toString(cmd.Args["slot"])
	res, err := s.store.UnequipWithEffects(playerID, slot)
	s.replyCommand(conn, "unequip", err, map[string]interface{}{"message": res.Message}, nil)
	if err == nil && res.NeedRoomUpdate {
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleGameLearn(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	npcID := toString(cmd.Args["npcId"])
	skillID := toString(cmd.Args["skillId"])
	res, err := s.store.LearnWithEffects(playerID, npcID, skillID)
	s.replyCommand(conn, "learn", err, map[string]interface{}{"message": res.Message}, nil)
	if err == nil && res.NeedRoomUpdate {
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleGameMine(conn *websocket.Conn, _ protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	res, err := s.store.MineWithEffects(playerID)
	s.replyCommand(conn, "mine", err, map[string]interface{}{"experience": res.Experience, "potential": res.Potential}, res.Logs)
	if err == nil && res.RoomMessage != "" {
		s.emitRoomMessageToOthersByPlayer(playerID, res.RoomMessage)
	}
	if err == nil && res.NeedRoomUpdate {
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleGameKill(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	targetID := toString(cmd.Args["targetId"])
	res, err := s.store.KillWithEffects(playerID, targetID)
	s.replyCommand(conn, "kill", err, map[string]interface{}{"message": res.Message}, nil)
	if err == nil && res.NeedRoomMessage {
		s.emitRoomMessageByPlayer(playerID, res.Message)
	}
	if err == nil && res.NeedRoomUpdate {
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleGameSpar(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	targetID := toString(cmd.Args["targetId"])
	res, err := s.store.SparWithEffects(playerID, targetID)
	s.replyCommand(conn, "spar", err, map[string]interface{}{"message": res.Message}, nil)
	if err == nil && res.NeedRoomMessage {
		s.emitRoomMessageByPlayer(playerID, res.Message)
	}
	if err == nil && res.NeedRoomUpdate {
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleGameRelive(conn *websocket.Conn, _ protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	res, err := s.store.ReliveWithEffects(playerID)
	s.replyCommand(conn, "relive", err, map[string]interface{}{"message": res.Message, "action": res.Action}, nil)
	if err == nil && res.NeedRoomUpdate {
		if res.FromRoomID != "" && res.FromRoomID != res.ToRoomID {
			s.emitPlayerLeft(playerID, res.FromRoomID, res.FromRoomName)
			s.emitRoomUpdate(res.FromRoomID)
		}
		if res.ToRoomID != "" && res.FromRoomID != res.ToRoomID {
			s.emitPlayerEntered(playerID, res.ToRoomID, res.ToRoomName)
			s.emitRoomUpdate(res.ToRoomID)
			return
		}
		s.emitRoomUpdateByPlayer(playerID)
	}
}

func (s *Server) handleChatPublic(conn *websocket.Conn, cmd protocol.CommandMessage) {
	playerID, ok := s.requireSelectedPlayer(conn)
	if !ok {
		return
	}
	message := toString(cmd.Args["message"])
	if message == "" {
		return
	}
	res, err := s.store.GetRoomSnapshotView(playerID)
	if err != nil {
		s.write(conn, protocol.NewResult(protocol.TypeChat, "public", false, map[string]interface{}{"error": err.Error()}, nil))
		return
	}
	sender := toString(res.Player["username"])
	payload := protocol.NewResult(protocol.TypeChat, "public", true, map[string]interface{}{"senderName": sender, "message": message, "roomId": res.Room["id"]}, nil)
	s.broadcastToRoom(toString(res.Room["id"]), payload, "")
}
