package com.mudgame.engine.handler.game;

import com.mudgame.engine.manager.ItemManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.exception.UnknownCommandException;
import com.mudgame.model.Item;
import com.mudgame.model.Player;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.utils.Logger;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GetBackpackHandler extends GameCommandHandler {
    private static final String subtype = "get_backpack";
    private final ItemManager itemManager;
    private final PlayerManager playerManager;

    public GetBackpackHandler(ItemManager itemManager, PlayerManager playerManager) {
        this.itemManager = itemManager;
        this.playerManager = playerManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info("engine: 接收到get_backpack指令，正在获取玩家背包");
        Player player;
        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();

        try {
            player = playerManager.getPlayer(message.getPlayerId());
        } catch (Exception e) {
            Logger.error("获取玩家失败: " + e.getMessage());
            logs.add("ERROR: Failed to get player info.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), null, logs);
        }

        Map<String, Integer> inventory = player.getItems();
        List<Map<String, Object>> backpackItems = new ArrayList<>();

        if (inventory != null) {
            for (Map.Entry<String, Integer> entry : inventory.entrySet()) {
                String itemId = entry.getKey();
                int count = entry.getValue();
                try {
                    Item item = itemManager.getItem(itemId);
                    if (item == null) continue;

                    if (item.isStackable()) {
                        Map<String, Object> itemMap = itemToMap(item);
                        itemMap.put("count", count);
                        itemMap.put("uniqueId", item.getId());
                        backpackItems.add(itemMap);
                    } else {
                        for (int i = 0; i < count; i++) {
                            Map<String, Object> itemMap = itemToMap(item);
                            itemMap.put("count", 1);
                            itemMap.put("uniqueId", item.getId() + "_" + i);
                            backpackItems.add(itemMap);
                        }
                    }
                } catch (Exception e) {
                    Logger.warn("获取物品失败: " + itemId + ", " + e.getMessage());
                }
            }
        }

        results.put("player", player);
        results.put("items", backpackItems);
        results.put("money", player.getMoney());

        Logger.info("engine: 成功获取玩家背包信息");
        logs.add("INFO: successfully get " + player.getId() + "‘s backpack");

        return new GameResult(type, subtype, true, System.currentTimeMillis(), results, logs);
    }

    @Override
    public String getType() {
        return type;
    }

    @Override
    public String getSubType() {
        return subtype;
    }

    private Map<String, Object> itemToMap(Item item) {
        Map<String, Object> map = new HashMap<>();
        map.put("id", item.getId());
        map.put("name", item.getName());
        map.put("description", item.getDescription());
        map.put("stackable", item.isStackable());
        map.put("sellable", item.isSellable());
        map.put("value", item.getValue());
        map.put("rarity", item.getRarity());
        map.put("unit", item.getUnit());
        map.put("equipmentSlot", item.getEquipmentSlot());
        return map;
    }
}
