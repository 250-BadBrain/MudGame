package com.mudgame.engine.handler.game;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.StatusManager;
import com.mudgame.model.Attributes;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.Map;

public class HealHandler implements CommandHandler {
    private final PlayerManager playerManager;
    private final StatusManager statusManager;

    public HealHandler(PlayerManager playerManager, StatusManager statusManager) {
        this.playerManager = playerManager;
        this.statusManager = statusManager;
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "heal";
    }

    @Override
    public GameResult handle(CommandMessage command) {
        String playerId = command.getPlayerId();
        Player player = playerManager.getPlayer(playerId);
        
        if (player == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "player_not_found"), null);
        }
        
        if (player.getState() == PlayerState.COMBAT) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "combat_state"), null);
        }
        
        if (player.getState() == PlayerState.DEAD) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "dead_state"), null);
        }
        
        if (player.getState() == PlayerState.HEALING) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "healing_state"), null);
        }
        
        if (player.getState() == PlayerState.MEDITATING) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "meditating_state"), null);
        }
        
        if (player.getState() == PlayerState.MINING) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "mining_state"), null);
        }

        if (player.getState() == PlayerState.LEARNING) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "learning_state"), null);
        }

        Attributes attr = player.getAttributes();
        
        if (attr.getCurrentHp() >= attr.getMaxHp()) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "hp_full"), null);
        }

        int costMp = 10;
        if (attr.getCurrentMp() < costMp) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "mp_insufficient"), null);
        }

        // Start continuous healing
        statusManager.startHeal(player);

        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), null, null);
    }
}
