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

public class EquipHandler implements CommandHandler {
    private final ItemManager itemManager;
    private final PlayerManager playerManager;

    public EquipHandler(ItemManager itemManager, PlayerManager playerManager) {
        this.itemManager = itemManager;
        this.playerManager = playerManager;
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "equip";
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

        String itemId = (String) command.getArg("itemId");
        if (itemId == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "missing_item_id"), null);
        }

        Map<String, Integer> inventory = player.getItems();
        if (!inventory.containsKey(itemId) || inventory.get(itemId) <= 0) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "item_not_in_inventory"), null);
        }

        Item item = itemManager.getItem(itemId);
        if (item == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "item_not_found"), null);
        }

        String slot = item.getEquipmentSlot();
        if (slot == null || slot.isEmpty()) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "not_equippable"), null);
        }

        Map<String, String> equipment = player.getEquipment();
        if (equipment == null) {
            equipment = new java.util.HashMap<>();
            player.setEquipment(equipment);
        }
        String oldItemId = equipment.get(slot);

        // Remove new item from inventory
        int count = inventory.get(itemId);
        if (count == 1) {
            inventory.remove(itemId);
        } else {
            inventory.put(itemId, count - 1);
        }

        // If there was an item in the slot, move it to inventory
        if (oldItemId != null) {
            inventory.merge(oldItemId, 1, (a, b) -> a + b);
        }

        // Put new item in slot
        equipment.put(slot, itemId);

        playerManager.savePlayer(player);

        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), 
            Map.of("inventory", inventory, "equipment", equipment), 
            List.of("你装备了 " + item.getName() + "。"));
    }
}
