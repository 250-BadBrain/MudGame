package com.mudgame.engine.handler.game;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.manager.CombatManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.StatusManager;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.Map;

public class StopHandler implements CommandHandler {
    private final CombatManager combatManager;
    private final PlayerManager playerManager;
    private final StatusManager statusManager;

    public StopHandler(CombatManager combatManager, PlayerManager playerManager, StatusManager statusManager) {
        this.combatManager = combatManager;
        this.playerManager = playerManager;
        this.statusManager = statusManager;
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "stop";
    }

    @Override
    public GameResult handle(CommandMessage command) {
        String playerId = command.getPlayerId();
        Player player = playerManager.getPlayer(playerId);
        
        if (player == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "玩家不存在"), null);
        }
        
        if (player.getState() == PlayerState.HEALING || player.getState() == PlayerState.MEDITATING || player.getState() == PlayerState.MINING || player.getState() == PlayerState.LEARNING) {
            statusManager.stopStatus(player, true);
            // Return empty message to avoid duplication, as StatusManager sends the specific message
            return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), null, null);
        } else if (player.getState() == PlayerState.COMBAT) {
             boolean escaped = combatManager.tryEscape(player);
             if (escaped) {
                 return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), Map.of("message", "你试图抽身撤出战斗。"), null);
             }
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "escape_failed"), null);
        } else {
             return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), Map.of("message", "你什么也没做。"), null);
        }
    }
}
