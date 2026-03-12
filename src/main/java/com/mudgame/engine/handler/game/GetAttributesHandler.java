package com.mudgame.engine.handler.game;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.exception.UnknownCommandException;
import com.mudgame.model.Player;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.utils.Logger;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GetAttributesHandler extends GameCommandHandler {
    private static final String subtype = "get_attributes";
    private final PlayerManager playerManager;

    public GetAttributesHandler(PlayerManager playerManager) {
        this.playerManager = playerManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info("engine: 接收到get_attributes指令，正在获取玩家属性");
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

        results.put("player", player);
        results.put("attributes", player.getAttributes());

        Logger.info("engine: 成功获取玩家属性信息");
        logs.add("INFO: successfully get " + player.getId() + "‘s attributes");

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
