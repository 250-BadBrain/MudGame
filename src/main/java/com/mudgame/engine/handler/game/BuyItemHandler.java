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

public class BuyItemHandler extends GameCommandHandler {
    private static final String subtype = "buy";
    private final ItemManager itemManager;
    private final NpcManager npcManager;
    private final PlayerManager playerManager;
    private final RoomManager roomManager;

    public BuyItemHandler(ItemManager itemManager, NpcManager npcManager, PlayerManager playerManager, RoomManager roomManager) {
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
            
            if (!room.getNpcs().contains(npcId)) {
                 return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("NPC不在房间内"));
            }

            Npc npc = npcManager.getNpc(npcId);
            List<String> shopItemIds = npc.getShopItems();
            
            if (shopItemIds == null || !shopItemIds.contains(itemId)) {
                return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("该NPC不出售此物品"));
            }

            Item item = itemManager.getItem(itemId);
            
            // Check NPC stock
            int stock = npc.getStock(itemId, item.getLimit());
            if (stock != -1) {
                if (amount > stock) {
                     return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of(npc.getName() + "没有那么多" + item.getName()));
                }
            }

            long totalCost = (long) item.getValue() * amount;

            if (player.getMoney() < totalCost) {
                return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("你的钱不够"));
            }

            // Deduct money
            player.setMoney(player.getMoney() - totalCost);
            
            // Reduce NPC stock
            if (stock != -1) {
                npc.reduceStock(itemId, amount);
            }
            
            // Add item(s)
            player.getItems().merge(itemId, amount, (a, b) -> a + b);
            
            playerManager.savePlayer(player);

            logs.add("你花费了" + formatMoney(totalCost) + "购买了" + amount + "个" + item.getName());
            results.put("money", player.getMoney());
            results.put("backpack", player.getItems());
            results.put("boughtItemId", itemId);
            results.put("boughtAmount", amount);

        } catch (Exception e) {
            Logger.error("Buy item failed: " + e.getMessage());
            return new GameResult(type, subtype, false, System.currentTimeMillis(), results, List.of("购买失败: " + e.getMessage()));
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
