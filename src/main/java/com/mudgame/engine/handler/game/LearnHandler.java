package com.mudgame.engine.handler.game;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.engine.manager.StatusManager;
import com.mudgame.engine.manager.NpcManager;
import com.mudgame.model.Npc;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.model.Room;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.Map;

public class LearnHandler implements CommandHandler {
    private final PlayerManager playerManager;
    private final RoomManager roomManager;
    private final StatusManager statusManager;
    private final NpcManager npcManager;

    public LearnHandler(PlayerManager playerManager, RoomManager roomManager, StatusManager statusManager, NpcManager npcManager) {
        this.playerManager = playerManager;
        this.roomManager = roomManager;
        this.statusManager = statusManager;
        this.npcManager = npcManager;
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "learn";
    }

    @Override
    public GameResult handle(CommandMessage command) {
        String playerId = command.getPlayerId();
        Player player = playerManager.getPlayer(playerId);
        
        if (player == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "player_not_found"), null);
        }
        
        if (player.getState() != PlayerState.IDLE) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "busy_state"), null);
        }
        
        Map<String, Object> data = command.getArgs();
        if (data == null || !data.containsKey("skillId") || !data.containsKey("npcId")) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "missing_params"), null);
        }
        
        String skillId = (String) data.get("skillId");
        String npcId = (String) data.get("npcId");
        if (skillId != null) {
            skillId = skillId.trim();
        }
        if (npcId != null) {
            npcId = npcId.trim();
        }

        if (skillId == null || skillId.isEmpty() || npcId == null || npcId.isEmpty()) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "missing_params"), null);
        }

        if (player.getAttributes().getPotential() <= 0) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "no_potential"), null);
        }
        
        // Check if NPC is in room
        Room room = roomManager.getRoom(player.getRoomId());
        if (room == null || room.getNpcs() == null || !room.getNpcs().contains(npcId)) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "npc_not_found"), null);
        }
        
        // Check if NPC teaches the skill
        Npc npc = npcManager.getNpc(npcId);
        if (npc == null) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "npc_not_found"), null);
        }
        
        if (npc.getSkills() == null || !npc.getSkills().containsKey(skillId)) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "skill_not_taught"), null);
        }

           if (player.getSkillLevel(skillId) >= 300) {
               return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "skill_level_capped"), null);
           }
        
        // Start learning
        statusManager.startLearning(player, skillId);
        
        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), null, null);
    }
}
