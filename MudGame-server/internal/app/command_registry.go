package app

import (
	"mudgame/server/internal/exception"
	"mudgame/server/internal/protocol"

	"github.com/gorilla/websocket"
)

func (s *Server) registerCommandHandlers() {
	// system commands
	s.handlersSys["register"] = s.handleSystemRegister
	s.handlersSys["login"] = s.handleSystemLogin
	s.handlersSys["delete_account"] = s.handleSystemDeleteAccount
	s.handlersSys["create_character"] = s.handleSystemCreateCharacter
	s.handlersSys["get_characters"] = s.handleSystemGetCharacters
	s.handlersSys["select_character"] = s.handleSystemSelectCharacter
	s.handlersSys["delete_character"] = s.handleSystemDeleteCharacter
	s.handlersSys["reconnect"] = s.handleSystemReconnect
	s.handlersSys["deselect_character"] = s.handleSystemDeselectCharacter
	s.handlersSys["get_room"] = s.handleSystemGetRoom
	s.handlersSys["ping"] = s.handleSystemPing

	// game commands (legacy wire type is command)
	s.handlersGame["get_attributes"] = s.handleGameGetAttributes
	s.handlersGame["get_backpack"] = s.handleGameGetBackpack
	s.handlersGame["get_skills"] = s.handleGameGetSkills
	s.handlersGame["get_equipment"] = s.handleGameGetEquipment
	s.handlersGame["get_worlds"] = s.handleGameGetWorlds
	s.handlersGame["get_dungeons"] = s.handleGameGetDungeons
	s.handlersGame["enter_world"] = s.handleGameEnterWorld
	s.handlersGame["enter_dungeon"] = s.handleGameEnterDungeon
	s.handlersGame["leave_dungeon"] = s.handleGameLeaveDungeon
	s.handlersGame["quick_complete_dungeon"] = s.handleGameQuickCompleteDungeon
	s.handlersGame["go"] = s.handleGameGo
	s.handlersGame["get_shop"] = s.handleGameGetShop
	s.handlersGame["buy"] = s.handleGameBuy
	s.handlersGame["discard_item"] = s.handleGameDiscardItem
	s.handlersGame["sell"] = s.handleGameSell
	s.handlersGame["equip"] = s.handleGameEquip
	s.handlersGame["unequip"] = s.handleGameUnequip
	s.handlersGame["learn"] = s.handleGameLearn
	s.handlersGame["practice"] = s.handleGamePractice
	s.handlersGame["unequip_skill"] = s.handleGameUnequipSkill
	s.handlersGame["equip_skill"] = s.handleGameEquipSkill
	s.handlersGame["enhance_item"] = s.handleGameEnhanceItem
	s.handlersGame["reset_enhancement"] = s.handleGameResetEnhancement
	s.handlersGame["use_item"] = s.handleGameUseItem
	s.handlersGame["do_action"] = s.handleGameDoAction
	s.handlersGame["talk"] = s.handleGameTalk
	s.handlersGame["choose_dialogue"] = s.handleGameChooseDialogue
	s.handlersGame["get_quests"] = s.handleGameGetQuests
	s.handlersGame["get_factions"] = s.handleGameGetFactions
	s.handlersGame["heal"] = s.handleGameHeal
	s.handlersGame["meditate"] = s.handleGameMeditate
	s.handlersGame["stop"] = s.handleGameStop
	s.handlersGame["kill"] = s.handleGameKill
	s.handlersGame["spar"] = s.handleGameSpar
	s.handlersGame["relive"] = s.handleGameRelive

	// chat commands
	s.handlersChat["public"] = s.handleChatPublic
}

func (s *Server) writeUnknownCommand(conn *websocket.Conn, cmd protocol.CommandMessage) {
	msgType := cmd.Type
	if msgType == "" {
		msgType = protocol.TypeSystem
	}
	s.write(conn, protocol.NewResult(msgType, cmd.Subtype, false, map[string]interface{}{"error": exception.ErrUnknownCommand.Error()}, nil))
}
