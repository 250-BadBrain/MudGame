package com.mudgame.engine.handler.game;

import com.mudgame.engine.manager.ItemManager;
import com.mudgame.engine.manager.NpcManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.exception.UnknownCommandException;
import com.mudgame.model.Item;
import com.mudgame.model.Npc;
import com.mudgame.model.Player;
import com.mudgame.model.Room;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.utils.Logger;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GetShopHandler extends GameCommandHandler {
    private static final String subtype = "get_shop";
    private final ItemManager itemManager;
    private final NpcManager npcManager;
    private final PlayerManager playerManager;
    private final RoomManager roomManager;

    public GetShopHandler(ItemManager itemManager, NpcManager npcManager, PlayerManager playerManager, RoomManager roomManager) {
        this.itemManager = itemManager;
        this.npcManager = npcManager;
        this.playerManager = playerManager;
        this.roomManager = roomManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        String playerId = message.getPlayerId();
        Map<String, Object> args = message.getArgs();
        String npcId = (String) args.get("npcId");

        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();

        if (npcId == null) {
            return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("NPC ID is required"));
        }

        try {
            Player player = playerManager.getPlayer(playerId);
            Room room = roomManager.getRoom(player.getRoomId());
            
            // Verify NPC is in the room
            if (!room.getNpcs().contains(npcId)) {
                 return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("NPC not found in this room"));
            }

            Npc npc = npcManager.getNpc(npcId);
            List<String> shopItemIds = npc.getShopItems();
            
            if (shopItemIds == null || shopItemIds.isEmpty()) {
                return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("This NPC is not a merchant"));
            }

            List<Map<String, Object>> shopItems = new ArrayList<>();
            for (String itemId : shopItemIds) {
                Item item = itemManager.getItem(itemId);
                if (item != null) {
                    Logger.info("GetShopHandler: item=" + item.getName() + ", limit=" + item.getLimit());
                    Map<String, Object> itemData = new HashMap<>();
                    itemData.put("id", itemId);
                    itemData.put("name", item.getName());
                    itemData.put("description", item.getDescription());
                    itemData.put("value", item.getValue());
                    itemData.put("stackable", item.isStackable());
                    itemData.put("rarity", item.getRarity());
                    itemData.put("unit", item.getUnit());
                    
                    // Use NPC stock instead of player limit
                    int stock = npc.getStock(itemId, item.getLimit());
                    if (stock != -1) {
                        itemData.put("limit", stock);
                    }
                    shopItems.add(itemData);
                }
            }

            results.put("shopItems", shopItems);
            results.put("npcName", npc.getName());
            results.put("npcTitle", npc.getTitle());
            results.put("npcId", npc.getId());

        } catch (Exception e) {
            Logger.error("Get shop failed: " + e.getMessage());
            return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("Error getting shop info"));
        }

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
}
