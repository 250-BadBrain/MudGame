package com.mudgame.engine.handler.game;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.manager.ItemManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.model.Item;
import com.mudgame.model.Player;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.HashMap;
import java.util.Map;

public class GetEquipmentHandler implements CommandHandler {
    private final ItemManager itemManager;
    private final PlayerManager playerManager;

    public GetEquipmentHandler(ItemManager itemManager, PlayerManager playerManager) {
        this.itemManager = itemManager;
        this.playerManager = playerManager;
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "get_equipment";
    }

    @Override
    public GameResult handle(CommandMessage command) {
        String playerId = command.getPlayerId();
        Player player = playerManager.getPlayer(playerId);
        
        if (player == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "player_not_found"), null);
        }

        Map<String, String> equipmentIds = player.getEquipment();
        if (equipmentIds == null) {
            equipmentIds = new HashMap<>();
        }
        Map<String, Item> equipmentItems = new HashMap<>();
        for (Map.Entry<String, String> entry : equipmentIds.entrySet()) {
            Item item = itemManager.getItem(entry.getValue());
            if (item != null) {
                equipmentItems.put(entry.getKey(), item);
            }
        }

        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), 
            Map.of("equipment", equipmentItems), null);
    }
}
