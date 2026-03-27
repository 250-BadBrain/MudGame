package engine

import "mudgame/server/internal/model"

type ReconnectResult struct {
	UserID   string
	PlayerID string
}

type TransitionResult struct {
	Room         map[string]interface{}
	FromRoomID   string
	FromRoomName string
	ToRoomID     string
	ToRoomName   string
}

type StateActionResult struct {
	Message        string
	RoomMessage    string
	NeedRoomUpdate bool
}

type MineActionResult struct {
	Experience     int
	Potential      int
	Logs           []string
	RoomMessage    string
	NeedRoomUpdate bool
}

type CombatActionResult struct {
	Message         string
	NeedRoomUpdate  bool
	NeedRoomMessage bool
}

type ReliveActionResult struct {
	Message        string
	Action         string
	NeedRoomUpdate bool
	FromRoomID     string
	FromRoomName   string
	ToRoomID       string
	ToRoomName     string
}

type ShopViewResult struct {
	Items    []map[string]interface{}
	NPCID    string
	NPCName  string
	NPCTitle string
}

type BuyActionResult struct {
	Money        int
	BoughtItemID string
	BoughtAmount int
	Logs         []string
}

type SellActionResult struct {
	Money   int
	Gain    int
	Message string
}

type RoomSnapshotResult struct {
	Room   map[string]interface{}
	Player map[string]interface{}
}

type BackpackResult struct {
	Items []map[string]interface{}
	Money int
}

type RoomUpdateResult struct {
	Players []map[string]interface{}
	NPCs    []map[string]interface{}
}

func (s *Service) Reconnect(userIDHint, sessionUserID, token, playerID string) (ReconnectResult, error) {
	userID := firstNonEmpty(userIDHint, sessionUserID)
	if !s.repo.IsReconnectTokenValid(userID, token) {
		return ReconnectResult{}, errReconnectTokenInvalid
	}
	res := ReconnectResult{UserID: userID}
	if playerID == "" {
		return res, nil
	}
	if !s.repo.CharacterBelongsToUser(userID, playerID) {
		return ReconnectResult{}, errReconnectPlayerInvalid
	}
	res.PlayerID = playerID
	return res, nil
}

func (s *Service) EnterWorldWithTransition(playerID, worldID string) (TransitionResult, error) {
	fromID, fromName, err := s.repo.GetPlayerRoom(playerID)
	if err != nil {
		return TransitionResult{}, err
	}
	room, err := s.repo.EnterWorld(playerID, worldID)
	if err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		Room:         room,
		FromRoomID:   fromID,
		FromRoomName: fromName,
		ToRoomID:     asString(room["id"]),
		ToRoomName:   asString(room["name"]),
	}, nil
}

func (s *Service) MoveWithTransition(playerID, direction string) (TransitionResult, error) {
	fromID, fromName, err := s.repo.GetPlayerRoom(playerID)
	if err != nil {
		return TransitionResult{}, err
	}
	room, err := s.repo.Move(playerID, direction)
	if err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{
		Room:         room,
		FromRoomID:   fromID,
		FromRoomName: fromName,
		ToRoomID:     asString(room["id"]),
		ToRoomName:   asString(room["name"]),
	}, nil
}

func (s *Service) HealWithEffects(playerID string) (StateActionResult, error) {
	msg, err := s.repo.Heal(playerID)
	if err != nil {
		return StateActionResult{}, err
	}
	name := s.playerName(playerID)
	roomMsg := ""
	if name != "" {
		roomMsg = name + "运功疗伤，气色看起来好多了。"
	}
	return StateActionResult{Message: msg, RoomMessage: roomMsg, NeedRoomUpdate: true}, nil
}

func (s *Service) MeditateWithEffects(playerID string) (StateActionResult, error) {
	msg, err := s.repo.Meditate(playerID)
	if err != nil {
		return StateActionResult{}, err
	}
	name := s.playerName(playerID)
	roomMsg := ""
	if name != "" {
		roomMsg = name + "盘膝而坐，吐纳调息。"
	}
	return StateActionResult{Message: msg, RoomMessage: roomMsg, NeedRoomUpdate: true}, nil
}

func (s *Service) StopWithEffects(playerID string) StateActionResult {
	status, err := s.repo.GetPlayerStatus(playerID)
	if err == nil && asString(status["state"]) == "COMBAT" {
		return StateActionResult{Message: "你正在战斗，待会再说", NeedRoomUpdate: false}
	}
	name := s.playerName(playerID)
	msg := s.repo.Stop(playerID)
	roomMsg := ""
	if name != "" {
		switch msg {
		case "你收起铁镐，停止了挖矿。":
			roomMsg = name + "收起铁镐，停止了挖矿。"
		case "你停止了运功疗伤。":
			roomMsg = name + "停止了运功疗伤。"
		case "你停止了打坐。":
			roomMsg = name + "停止了打坐。"
		case "你停止了学习。":
			roomMsg = name + "停止了学习。"
		default:
			roomMsg = name + "停止了动作。"
		}
	}
	return StateActionResult{Message: msg, RoomMessage: roomMsg, NeedRoomUpdate: true}
}

func (s *Service) MineWithEffects(playerID string) (MineActionResult, error) {
	exp, potential, gemName, err := s.repo.Mine(playerID)
	if err != nil {
		return MineActionResult{}, err
	}
	logs := []string{"你挥舞着铁镐，开始卖力地挖矿。"}
	if gemName != "" {
		logs = append(logs, "你在矿脉里挖出了"+gemName)
	}
	name := s.playerName(playerID)
	roomMsg := ""
	if name != "" {
		roomMsg = name + "挥舞着铁镐，开始卖力地挖矿。"
	}
	return MineActionResult{Experience: exp, Potential: potential, Logs: logs, RoomMessage: roomMsg, NeedRoomUpdate: true}, nil
}

func (s *Service) KillWithEffects(playerID, targetID string) (CombatActionResult, error) {
	msg, err := s.repo.Kill(playerID, targetID)
	if err != nil {
		return CombatActionResult{}, err
	}
	return CombatActionResult{Message: msg, NeedRoomUpdate: true, NeedRoomMessage: true}, nil
}

func (s *Service) SparWithEffects(playerID, targetID string) (CombatActionResult, error) {
	msg, err := s.repo.Spar(playerID, targetID)
	if err != nil {
		return CombatActionResult{}, err
	}
	return CombatActionResult{Message: msg, NeedRoomUpdate: true, NeedRoomMessage: true}, nil
}

func (s *Service) ReliveWithEffects(playerID string) (ReliveActionResult, error) {
	fromID, fromName, err := s.repo.GetPlayerRoom(playerID)
	if err != nil {
		return ReliveActionResult{}, err
	}
	msg, err := s.repo.Relive(playerID)
	if err != nil {
		return ReliveActionResult{}, err
	}
	toID, toName, err := s.repo.GetPlayerRoom(playerID)
	if err != nil {
		return ReliveActionResult{}, err
	}
	return ReliveActionResult{
		Message:        msg,
		Action:         "refresh_room",
		NeedRoomUpdate: true,
		FromRoomID:     fromID,
		FromRoomName:   fromName,
		ToRoomID:       toID,
		ToRoomName:     toName,
	}, nil
}

func (s *Service) GetShopView(playerID, npcID string) (ShopViewResult, error) {
	items, rid, rname, rtitle, err := s.repo.GetShop(playerID, npcID)
	if err != nil {
		return ShopViewResult{}, err
	}
	return ShopViewResult{Items: items, NPCID: rid, NPCName: rname, NPCTitle: rtitle}, nil
}

func (s *Service) BuyWithReceipt(playerID, npcID, itemID string, amount int) (BuyActionResult, error) {
	bought, money, err := s.repo.Buy(playerID, npcID, itemID, amount)
	if err != nil {
		return BuyActionResult{}, err
	}
	return BuyActionResult{
		Money:        money,
		BoughtItemID: itemID,
		BoughtAmount: bought,
		Logs:         []string{"购买完成。"},
	}, nil
}

func (s *Service) SellWithReceipt(playerID, itemID string, amount int) (SellActionResult, error) {
	gain, money, err := s.repo.SellItem(playerID, itemID, amount)
	if err != nil {
		return SellActionResult{}, err
	}
	return SellActionResult{Money: money, Gain: gain, Message: "出售成功"}, nil
}

func (s *Service) DiscardWithMessage(playerID, itemID string, amount int) (string, error) {
	if err := s.repo.DiscardItem(playerID, itemID, amount); err != nil {
		return "", err
	}
	return "已丢弃物品", nil
}

func (s *Service) EquipWithEffects(playerID, itemID string) (StateActionResult, error) {
	if err := s.repo.EquipItem(playerID, itemID); err != nil {
		return StateActionResult{}, err
	}
	return StateActionResult{Message: "装备成功", NeedRoomUpdate: true}, nil
}

func (s *Service) UnequipWithEffects(playerID, slot string) (StateActionResult, error) {
	if err := s.repo.UnequipItem(playerID, slot); err != nil {
		return StateActionResult{}, err
	}
	return StateActionResult{Message: "已脱下装备", NeedRoomUpdate: true}, nil
}

func (s *Service) LearnWithEffects(playerID, npcID, skillID string) (StateActionResult, error) {
	if err := s.repo.LearnSkill(playerID, npcID, skillID); err != nil {
		return StateActionResult{}, err
	}
	return StateActionResult{Message: "学习有所得", NeedRoomUpdate: false}, nil
}

func (s *Service) GetRoomSnapshotView(playerID string) (RoomSnapshotResult, error) {
	room, player, err := s.repo.GetRoomSnapshot(playerID)
	if err != nil {
		return RoomSnapshotResult{}, err
	}
	return RoomSnapshotResult{Room: room, Player: player}, nil
}

func (s *Service) GetAttributesView(playerID string) (map[string]interface{}, error) {
	return s.repo.GetAttributes(playerID)
}

func (s *Service) GetBackpackView(playerID string) (BackpackResult, error) {
	items, money, err := s.repo.GetBackpack(playerID)
	if err != nil {
		return BackpackResult{}, err
	}
	return BackpackResult{Items: items, Money: money}, nil
}

func (s *Service) GetSkillsView(playerID string) ([]map[string]interface{}, error) {
	return s.repo.GetSkills(playerID)
}

func (s *Service) GetEquipmentView(playerID string) (map[string]interface{}, error) {
	return s.repo.GetEquipment(playerID)
}

func (s *Service) GetWorldsView() []map[string]interface{} {
	return s.repo.GetWorlds()
}

func (s *Service) GetPlayerStatusView(playerID string) (map[string]interface{}, error) {
	return s.repo.GetPlayerStatus(playerID)
}

func (s *Service) GetRoomPayloadView(roomID string) (map[string]interface{}, error) {
	return s.repo.GetRoomPayloadByID(roomID)
}

func (s *Service) GetRoomUpdateView(roomID string) (RoomUpdateResult, error) {
	players, npcs, err := s.repo.GetRoomUpdate(roomID)
	if err != nil {
		return RoomUpdateResult{}, err
	}
	return RoomUpdateResult{Players: players, NPCs: npcs}, nil
}

func (s *Service) GetPlayerRoomView(playerID string) (string, string, error) {
	return s.repo.GetPlayerRoom(playerID)
}

func (s *Service) IsPlayerInRoomView(playerID, roomID string) bool {
	return s.repo.IsPlayerInRoom(playerID, roomID)
}

func (s *Service) ProcessCombatTickWithEffects() (model.CombatTickResult, error) {
	return s.repo.ProcessCombatTick()
}

func (s *Service) ProcessStatusTickWithEffects() (model.CombatTickResult, error) {
	return s.repo.ProcessStatusTick()
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (s *Service) playerName(playerID string) string {
	status, err := s.repo.GetPlayerStatus(playerID)
	if err != nil {
		return ""
	}
	name, _ := status["playerName"].(string)
	return name
}
