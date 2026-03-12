package com.mudgame.engine.handler.game;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.manager.ItemManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.model.Item;
import com.mudgame.model.Player;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.List;
import java.util.Map;

public class UnequipHandler implements CommandHandler {
    private final ItemManager itemManager;
    private final PlayerManager playerManager;

    public UnequipHandler(ItemManager itemManager, PlayerManager playerManager) {
        this.itemManager = itemManager;
        this.playerManager = playerManager;
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "unequip";
    }

    @Override
    public GameResult handle(CommandMessage command) {
        String playerId = command.getPlayerId();
        Player player = playerManager.getPlayer(playerId);
        
        if (player == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "player_not_found"), null);
        }

        if (player.getState() != com.mudgame.model.PlayerState.IDLE) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "busy"), List.of("你现在正忙着呢。"));
        }

        String slot = (String) command.getArg("slot");
        if (slot == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "missing_slot"), null);
        }

        Map<String, String> equipment = player.getEquipment();
        if (equipment == null || !equipment.containsKey(slot)) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "slot_empty"), null);
        }

        String itemId = equipment.get(slot);
        Item item = itemManager.getItem(itemId);
        
        // Remove from slot
        equipment.remove(slot);

        // Add to inventory
        player.getItems().merge(itemId, 1, (a, b) -> a + b);

        playerManager.savePlayer(player);

        String itemName = (item != null) ? item.getName() : "未知物品";

        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), 
            Map.of("inventory", player.getItems(), "equipment", equipment), 
            List.of("你脱下了 " + itemName + "。"));
    }
}
