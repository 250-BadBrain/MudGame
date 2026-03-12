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

public class DiscardItemHandler extends GameCommandHandler {
    private static final String subtype = "discard_item";
    private final ItemManager itemManager;
    private final PlayerManager playerManager;

    public DiscardItemHandler(ItemManager itemManager, PlayerManager playerManager) {
        this.itemManager = itemManager;
        this.playerManager = playerManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        String playerId = message.getPlayerId();
        Map<String, Object> args = message.getArgs();
        String itemId = (String) args.get("itemId");
        int amount = args.containsKey("amount") ? ((Number) args.get("amount")).intValue() : 1;

        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();

        if (amount <= 0) amount = 1;

        try {
            Player player = playerManager.getPlayer(playerId);
            
            if (player.getState() != com.mudgame.model.PlayerState.IDLE) {
                 return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("你现在正忙着呢。"));
            }

            Item item = itemManager.getItem(itemId);
            
            if (item == null) {
                return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("物品不存在"));
            }

            // Check if player has enough items
            int currentCount = player.getItems().getOrDefault(itemId, 0);
            if (currentCount < amount) {
                return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("你没有那么多" + item.getName()));
            }

            // Remove items
            if (currentCount == amount) {
                player.getItems().remove(itemId);
            } else {
                player.getItems().put(itemId, currentCount - amount);
            }
            
            playerManager.savePlayer(player);

            logs.add("你丢弃了" + amount + "个" + item.getName());
            results.put("backpack", player.getItems());

        } catch (Exception e) {
            Logger.error("Discard item failed: " + e.getMessage());
            return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("丢弃失败: " + e.getMessage()));
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
