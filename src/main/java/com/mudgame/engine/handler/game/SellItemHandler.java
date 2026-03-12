package com.mudgame.engine.handler.game;

import com.mudgame.engine.manager.ItemManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.exception.UnknownCommandException;
import com.mudgame.model.Item;
import com.mudgame.model.Player;
import com.mudgame.model.Room;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.utils.Logger;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class SellItemHandler extends GameCommandHandler {
    private static final String subtype = "sell";
    private final ItemManager itemManager;
    private final PlayerManager playerManager;
    private final RoomManager roomManager;

    public SellItemHandler(ItemManager itemManager, PlayerManager playerManager, RoomManager roomManager) {
        this.itemManager = itemManager;
        this.playerManager = playerManager;
        this.roomManager = roomManager;
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

            Room room = roomManager.getRoom(player.getRoomId());
            
            if (room.getNpcs() == null || room.getNpcs().isEmpty()) {
                 return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("这里没有NPC可以收购你的物品"));
            }

            Item item = itemManager.getItem(itemId);
            if (item == null) {
                return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("物品不存在"));
            }

            if (!item.isSellable()) {
                return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("这个物品不能卖"));
            }

            // Check if player has enough items
            int currentCount = player.getItems().getOrDefault(itemId, 0);
            if (currentCount < amount) {
                return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("你没有那么多" + item.getName()));
            }

            long unitPrice = item.getValue() / 2;
            long totalMoney = unitPrice * amount;

            // Remove items
            if (currentCount == amount) {
                player.getItems().remove(itemId);
            } else {
                player.getItems().put(itemId, currentCount - amount);
            }
            
            // Add money
            player.setMoney(player.getMoney() + totalMoney);
            
            playerManager.savePlayer(player);

            logs.add("你卖掉了" + amount + "个" + item.getName() + "，获得了" + formatMoney(totalMoney));
            results.put("money", player.getMoney());
            results.put("backpack", player.getItems());

        } catch (Exception e) {
            Logger.error("Sell item failed: " + e.getMessage());
            return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("出售失败: " + e.getMessage()));
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

    private String formatMoney(long money) {
        if (money == 0) return "0枚铜板";
        long gold = money / 10000;
        long silver = (money % 10000) / 100;
        long copper = money % 100;
        
        StringBuilder sb = new StringBuilder();
        if (gold > 0) sb.append(gold).append("两黄金");
        if (silver > 0) sb.append(silver).append("两银子");
        if (copper > 0) sb.append(copper).append("枚铜板");
        return sb.toString();
    }
}
