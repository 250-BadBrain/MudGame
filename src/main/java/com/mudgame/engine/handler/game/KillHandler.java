package com.mudgame.engine.handler.game;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.manager.CombatManager;
import com.mudgame.engine.manager.NpcManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.Map;

public class KillHandler implements CommandHandler {
    private final CombatManager combatManager;
    private final NpcManager npcManager;
    private final PlayerManager playerManager;

    public KillHandler(CombatManager combatManager, NpcManager npcManager, PlayerManager playerManager) {
        this.combatManager = combatManager;
        this.npcManager = npcManager;
        this.playerManager = playerManager;
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "kill";
    }

    @Override
    public GameResult handle(CommandMessage command) {
        String playerId = command.getPlayerId();
        Player attacker = playerManager.getPlayer(playerId);
        Map<String, Object> args = command.getArgs();
        
        if (attacker == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "player_not_found"), null);
        }
        
        if (attacker.getState() == PlayerState.DEAD) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "dead_state"), null);
        }
        
        if (attacker.getState() != PlayerState.IDLE) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "busy_state"), null);
        }

        if (args == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "target_missing"), null);
        }

        // 前端传来的可能是 targetId 或 id
        String targetId = (String) args.get("targetId");
        if (targetId == null) targetId = (String) args.get("id");
        
        if (targetId == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "target_missing"), null);
        }

        if (playerId.equals(targetId)) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "self_target"), null);
        }

        if (isNpcTarget(targetId)) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "npc_combat_unsupported"), null);
        }
        
        Player target = tryGetPlayer(targetId);
        if (target == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "target_not_found"), null);
        }
        
        if (!target.getRoomId().equals(attacker.getRoomId())) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "target_not_in_room"), null);
        }

        if (target.getState() == PlayerState.DEAD) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "target_is_dead"), null);
        }

        if (target.getState() == PlayerState.COMBAT) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "target_in_combat", "targetName", target.getCharacterName()), null);
        }

        combatManager.startCombat(attacker, target, "KILL");
        
        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), Map.of("message", "你对 " + target.getCharacterName() + " 下了杀手！"), null);
    }

            private boolean isNpcTarget(String targetId) {
                try {
                    npcManager.getNpc(targetId);
                    return true;
                } catch (RuntimeException ignored) {
                    return false;
                }
            }

            private Player tryGetPlayer(String targetId) {
                try {
                    return playerManager.getPlayer(targetId);
                } catch (IllegalArgumentException | IllegalStateException ignored) {
                    return null;
                }
            }
}
