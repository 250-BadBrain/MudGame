package engine

import (
	"mudgame/server/internal/model"
)

func EnterWorldWithTransition(repo RuntimeRepository, playerID, worldID string) (TransitionResult, error) {
	fromID, fromName, err := repo.GetPlayerRoom(playerID)
	if err != nil {
		return TransitionResult{}, err
	}
	room, err := repo.EnterWorld(playerID, worldID)
	if err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{Room: room, FromRoomID: fromID, FromRoomName: fromName, ToRoomID: asString(room["id"]), ToRoomName: asString(room["name"])}, nil
}

func EnterDungeonWithTransition(repo RuntimeRepository, playerID, dungeonID string) (TransitionResult, error) {
	fromID, fromName, err := repo.GetPlayerRoom(playerID)
	if err != nil {
		return TransitionResult{}, err
	}
	room, err := repo.EnterDungeon(playerID, dungeonID)
	if err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{Room: room, FromRoomID: fromID, FromRoomName: fromName, ToRoomID: asString(room["id"]), ToRoomName: asString(room["name"])}, nil
}

func LeaveDungeonWithTransition(repo RuntimeRepository, playerID string) (TransitionResult, error) {
	fromID, fromName, err := repo.GetPlayerRoom(playerID)
	if err != nil {
		return TransitionResult{}, err
	}
	room, err := repo.LeaveDungeon(playerID)
	if err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{Room: room, FromRoomID: fromID, FromRoomName: fromName, ToRoomID: asString(room["id"]), ToRoomName: asString(room["name"])}, nil
}

func MoveWithTransition(repo RuntimeRepository, playerID, direction string) (TransitionResult, error) {
	fromID, fromName, err := repo.GetPlayerRoom(playerID)
	if err != nil {
		return TransitionResult{}, err
	}
	room, err := repo.Move(playerID, direction)
	if err != nil {
		return TransitionResult{}, err
	}
	return TransitionResult{Room: room, FromRoomID: fromID, FromRoomName: fromName, ToRoomID: asString(room["id"]), ToRoomName: asString(room["name"])}, nil
}

func playerName(repo RuntimeRepository, playerID string) string {
	status, err := repo.GetPlayerStatus(playerID)
	if err != nil {
		return ""
	}
	return asString(status["playerName"])
}

func HealWithEffects(repo RuntimeRepository, playerID string) (StateActionResult, error) {
	msg, err := repo.Heal(playerID)
	if err != nil {
		return StateActionResult{}, err
	}
	name := playerName(repo, playerID)
	roomMsg := ""
	if name != "" {
		roomMsg = name + "运功疗伤，气色看起来好多了。"
	}
	return StateActionResult{Message: msg, RoomMessage: roomMsg, NeedRoomUpdate: true}, nil
}

func MeditateWithEffects(repo RuntimeRepository, playerID string) (StateActionResult, error) {
	msg, err := repo.Meditate(playerID)
	if err != nil {
		return StateActionResult{}, err
	}
	name := playerName(repo, playerID)
	roomMsg := ""
	if name != "" {
		roomMsg = name + "盘膝而坐，吐纳调息。"
	}
	return StateActionResult{Message: msg, RoomMessage: roomMsg, NeedRoomUpdate: true}, nil
}

func StopWithEffects(repo RuntimeRepository, playerID string) StateActionResult {
	status, err := repo.GetPlayerStatus(playerID)
	if err == nil && asString(status["state"]) == "COMBAT" {
		return StateActionResult{Message: "你正在战斗，待会再说", NeedRoomUpdate: false}
	}
	name := playerName(repo, playerID)
	msg := repo.Stop(playerID)
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

func MineWithEffects(repo RuntimeRepository, playerID string) (MineActionResult, error) {
	exp, potential, gemName, err := repo.Mine(playerID)
	if err != nil {
		return MineActionResult{}, err
	}
	logs := []string{"你挥舞着铁镐，开始卖力地挖矿。"}
	if gemName != "" {
		logs = append(logs, "你在矿脉里挖出了"+gemName)
	}
	name := playerName(repo, playerID)
	roomMsg := ""
	if name != "" {
		roomMsg = name + "挥舞着铁镐，开始卖力地挖矿。"
	}
	return MineActionResult{Experience: exp, Potential: potential, Logs: logs, RoomMessage: roomMsg, NeedRoomUpdate: true}, nil
}

func DoRoomActionWithEffects(repo RuntimeRepository, playerID, action string) (StateActionResult, error) {
	res, err := repo.DoRoomAction(playerID, action)
	if err != nil {
		return StateActionResult{}, err
	}
	msg := asString(res["message"])
	room, _, err := repo.GetRoomSnapshot(playerID)
	if err != nil {
		return StateActionResult{}, err
	}
	return StateActionResult{Message: msg, Room: room, NeedRoomUpdate: true}, nil
}

func TalkWithView(repo RuntimeRepository, playerID, npcID string) (DialogueResult, error) {
	dialogue, err := repo.Talk(playerID, npcID)
	if err != nil {
		return DialogueResult{}, err
	}
	return DialogueResult{Dialogue: dialogue}, nil
}

func ChooseDialogueWithEffects(repo RuntimeRepository, playerID, npcID, optionID string) (DialogueResult, error) {
	dialogue, message, logs, err := repo.ChooseDialogue(playerID, npcID, optionID)
	if err != nil {
		return DialogueResult{}, err
	}
	if message != "" {
		logs = append([]string{message}, logs...)
	}
	return DialogueResult{Dialogue: dialogue, Message: message, Logs: logs}, nil
}

func GetQuestsView(repo RuntimeRepository, playerID string) (QuestListResult, error) {
	quests, err := repo.GetQuests(playerID)
	if err != nil {
		return QuestListResult{}, err
	}
	return QuestListResult{Quests: quests}, nil
}

func GetFactionsView(repo RuntimeRepository, playerID string) (FactionListResult, error) {
	factions, err := repo.GetFactions(playerID)
	if err != nil {
		return FactionListResult{}, err
	}
	return FactionListResult{Factions: factions}, nil
}

func KillWithEffects(repo RuntimeRepository, playerID, targetID string) (CombatActionResult, error) {
	msg, err := repo.Kill(playerID, targetID)
	if err != nil {
		return CombatActionResult{}, err
	}
	return CombatActionResult{Message: msg, NeedRoomUpdate: true, NeedRoomMessage: true}, nil
}

func SparWithEffects(repo RuntimeRepository, playerID, targetID string) (CombatActionResult, error) {
	msg, err := repo.Spar(playerID, targetID)
	if err != nil {
		return CombatActionResult{}, err
	}
	return CombatActionResult{Message: msg, NeedRoomUpdate: true, NeedRoomMessage: true}, nil
}

func ReliveWithEffects(repo RuntimeRepository, playerID string) (ReliveActionResult, error) {
	fromID, fromName, err := repo.GetPlayerRoom(playerID)
	if err != nil {
		return ReliveActionResult{}, err
	}
	msg, err := repo.Relive(playerID)
	if err != nil {
		return ReliveActionResult{}, err
	}
	toID, toName, err := repo.GetPlayerRoom(playerID)
	if err != nil {
		return ReliveActionResult{}, err
	}
	return ReliveActionResult{Message: msg, Action: "refresh_room", NeedRoomUpdate: true, FromRoomID: fromID, FromRoomName: fromName, ToRoomID: toID, ToRoomName: toName}, nil
}

func GetShopView(repo RuntimeRepository, playerID, npcID string) (ShopViewResult, error) {
	items, rid, rname, rtitle, err := repo.GetShop(playerID, npcID)
	if err != nil {
		return ShopViewResult{}, err
	}
	return ShopViewResult{Items: items, NPCID: rid, NPCName: rname, NPCTitle: rtitle}, nil
}

func BuyWithReceipt(repo RuntimeRepository, playerID, npcID, itemID string, amount int) (BuyActionResult, error) {
	bought, money, err := repo.Buy(playerID, npcID, itemID, amount)
	if err != nil {
		return BuyActionResult{}, err
	}
	return BuyActionResult{Money: money, BoughtItemID: itemID, BoughtAmount: bought, Logs: []string{"购买完成。"}}, nil
}

func SellWithReceipt(repo RuntimeRepository, playerID, itemID string, amount int) (SellActionResult, error) {
	gain, money, err := repo.SellItem(playerID, itemID, amount)
	if err != nil {
		return SellActionResult{}, err
	}
	return SellActionResult{Money: money, Gain: gain, Message: "出售成功"}, nil
}

func DiscardWithMessage(repo RuntimeRepository, playerID, itemID string, amount int) (string, error) {
	if err := repo.DiscardItem(playerID, itemID, amount); err != nil {
		return "", err
	}
	return "已丢弃物品", nil
}

func EquipWithEffects(repo RuntimeRepository, playerID, itemID string) (StateActionResult, error) {
	if err := repo.EquipItem(playerID, itemID); err != nil {
		return StateActionResult{}, err
	}
	return StateActionResult{Message: "装备成功", NeedRoomUpdate: true}, nil
}

func UnequipWithEffects(repo RuntimeRepository, playerID, slot string) (StateActionResult, error) {
	if err := repo.UnequipItem(playerID, slot); err != nil {
		return StateActionResult{}, err
	}
	return StateActionResult{Message: "已脱下装备", NeedRoomUpdate: true}, nil
}

func LearnWithEffects(repo RuntimeRepository, playerID, npcID, skillID string) (StateActionResult, error) {
	if err := repo.LearnSkill(playerID, npcID, skillID); err != nil {
		return StateActionResult{}, err
	}
	return StateActionResult{Message: "你开始专心学习。", NeedRoomUpdate: true}, nil
}

func GetRoomSnapshotView(repo RuntimeRepository, playerID string) (RoomSnapshotResult, error) {
	room, player, err := repo.GetRoomSnapshot(playerID)
	if err != nil {
		return RoomSnapshotResult{}, err
	}
	return RoomSnapshotResult{Room: room, Player: player}, nil
}

func GetAttributesView(repo RuntimeRepository, playerID string) (map[string]interface{}, error) {
	return repo.GetAttributes(playerID)
}

func GetBackpackView(repo RuntimeRepository, playerID string) (BackpackResult, error) {
	items, money, err := repo.GetBackpack(playerID)
	if err != nil {
		return BackpackResult{}, err
	}
	return BackpackResult{Items: items, Money: money}, nil
}

func GetSkillsView(repo RuntimeRepository, playerID string) ([]map[string]interface{}, error) {
	return repo.GetSkills(playerID)
}

func GetEquipmentView(repo RuntimeRepository, playerID string) (map[string]interface{}, error) {
	return repo.GetEquipment(playerID)
}

func GetWorldsView(repo RuntimeRepository) []map[string]interface{} {
	return repo.GetWorlds()
}

func GetDungeonsView(repo RuntimeRepository, playerID string) []map[string]interface{} {
	return repo.GetDungeons(playerID)
}

func GetPlayerStatusView(repo RuntimeRepository, playerID string) (map[string]interface{}, error) {
	return repo.GetPlayerStatus(playerID)
}

func GetRoomPayloadView(repo RuntimeRepository, roomID string) (map[string]interface{}, error) {
	return repo.GetRoomPayloadByID(roomID)
}

func GetRoomUpdateView(repo RuntimeRepository, roomID string) (RoomUpdateResult, error) {
	players, npcs, err := repo.GetRoomUpdate(roomID)
	if err != nil {
		return RoomUpdateResult{}, err
	}
	return RoomUpdateResult{Players: players, NPCs: npcs}, nil
}

func GetPlayerRoomView(repo RuntimeRepository, playerID string) (string, string, error) {
	return repo.GetPlayerRoom(playerID)
}

func IsPlayerInRoomView(repo RuntimeRepository, playerID, roomID string) bool {
	return repo.IsPlayerInRoom(playerID, roomID)
}

func ProcessCombatTickWithEffects(repo RuntimeRepository) (model.CombatTickResult, error) {
	return repo.ProcessCombatTick()
}

func ProcessStatusTickWithEffects(repo RuntimeRepository) (model.CombatTickResult, error) {
	return repo.ProcessStatusTick()
}
