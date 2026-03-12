package com.mudgame.engine.handler.game;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.StatusManager;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.Map;

public class MineHandler implements CommandHandler {
    private final PlayerManager playerManager;
    private final StatusManager statusManager;

    public MineHandler(PlayerManager playerManager, StatusManager statusManager) {
        this.playerManager = playerManager;
        this.statusManager = statusManager;
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "mine";
    }

    @Override
    public GameResult handle(CommandMessage command) {
        String playerId = command.getPlayerId();
        Player player = playerManager.getPlayer(playerId);
        
        if (player == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "player_not_found"), null);
        }
        
        if (player.getState() == PlayerState.DEAD) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "dead_state"), null);
        }

        if (player.getState() != PlayerState.IDLE) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "busy"), null);
        }

        // Check if in Mine room
        if (!"ly_mine".equals(player.getRoomId())) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "wrong_location"), null);
        }

        Map<String, String> equipment = player.getEquipment();
        String weaponId = equipment == null ? null : equipment.get("WEAPON");
        if (!"iron_pickaxe".equals(weaponId)) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "no_pickaxe"), null);
        }

        statusManager.startMining(player);

        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), null, null);
    }
}
