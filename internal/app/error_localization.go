package app

import "strings"

var commandErrorMessages = map[string]string{
	"busy":                      "你现在正忙着呢。",
	"busy_state":                "你现在正忙着呢。",
	"dead_state":                "你已经重伤不起，先复活再说。",
	"healing_state":             "你正在疗伤中。",
	"meditating_state":          "你正在打坐中。",
	"mining_state":              "你正在挖矿中。",
	"learning_state":            "你正在学习中。",
	"combat_state":              "你正在战斗中，无法执行这个操作。",
	"world_not_found":           "没有找到这个世界。",
	"dungeon_not_found":         "没有找到这个副本。",
	"already_in_dungeon":        "你已经在副本中了。",
	"in_dungeon_state":          "你正在副本中，待会再说。",
	"not_in_dungeon":            "你当前不在副本中。",
	"dungeon_unavailable":       "副本系统暂不可用。",
	"dungeon_locked":            "你还没有找到进入这个秘境的线索。",
	"no_exit_in_that_direction": "这个方向走不通。",
	"shop_npc_not_in_room":      "这个商人不在当前房间。",
	"npc_not_shop":              "这个 NPC 不是商人。",
	"shop_not_found":            "商店不存在。",
	"sold_out":                  "该商品已售罄。",
	"item_not_found":            "没有找到该物品。",
	"money_not_enough":          "你的钱不够。",
	"no_npc_buyer":              "这里没有 NPC 会收购物品。",
	"item_not_sellable":         "该物品无法出售。",
	"missing_item_id":           "缺少物品信息。",
	"item_not_in_inventory":     "背包里没有这个物品。",
	"not_equippable":            "该物品无法装备。",
	"missing_slot":              "缺少装备槽位信息。",
	"slot_empty":                "该槽位没有装备。",
	"missing_params":            "参数不完整。",
	"no_potential":              "潜能不足，无法继续学习。",
	"npc_not_found":             "未找到该 NPC。",
	"skill_not_taught":          "该 NPC 不会传授这门技能。",
	"skill_level_capped":        "这门技能已达到当前上限。",
	"wrong_location":            "你不在可执行该操作的地点。",
	"no_pickaxe":                "你需要先装备铁镐。",
	"hp_full":                   "你的气血已满。",
	"mp_insufficient":           "你的内力不足。",
	"full_state":                "你的内力修为已达当前瓶颈。",
	"player_not_found":          "未找到角色。",
	"target_missing":            "缺少目标。",
	"self_target":               "不能以自己为目标。",
	"npc_combat_unsupported":    "暂不支持与 NPC 战斗。",
	"target_not_found":          "未找到目标。",
	"target_not_in_room":        "目标不在当前房间。",
	"target_is_dead":            "目标已经倒下了。",
	"target_busy":               "目标正忙着，无法交手。",
	"target_not_full":           "目标状态不佳，无法切磋。",
	"self_not_full":             "你当前状态不佳，无法切磋。",
	"target_in_combat":          "目标正在战斗中。",
	"not_dead":                  "你并没有死亡。",
}

func (s *Server) localizedError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if localized, ok := commandErrorMessages[msg]; ok {
		return localized
	}
	if strings.Contains(msg, "你") || strings.Contains(msg, "未") || strings.Contains(msg, "无") {
		return msg
	}
	return msg
}
