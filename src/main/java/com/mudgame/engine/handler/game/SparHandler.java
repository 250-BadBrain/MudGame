package com.mudgame.engine.handler.game;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.manager.CombatManager;
import com.mudgame.engine.manager.NpcManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.model.Attributes;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.Map;

public class SparHandler implements CommandHandler {
    private final CombatManager combatManager;
    private final NpcManager npcManager;
    private final PlayerManager playerManager;

    public SparHandler(CombatManager combatManager, NpcManager npcManager, PlayerManager playerManager) {
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
        return "spar";
    }

    @Override
    public GameResult handle(CommandMessage command) {
        String playerId = command.getPlayerId();
        Player attacker = playerManager.getPlayer(playerId);
        
        if (attacker == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "玩家不存在"), null);
        }
        
        if (attacker.getState() == PlayerState.DEAD) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "你已经死了，无法切磋。"), null);
        }
        
        if (attacker.getState() != PlayerState.IDLE) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "busy"), null);
        }

        Attributes attackerAttr = attacker.getAttributes();
        if (attackerAttr.getCurrentHp() < attackerAttr.getMaxHp()) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "self_not_full"), null);
        }

        Map<String, Object> args = command.getArgs();
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

        if (target.getState() != PlayerState.IDLE) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "target_busy", "targetName", target.getCharacterName()), null);
        }

        Attributes targetAttr = target.getAttributes();
        if (targetAttr.getCurrentHp() < targetAttr.getMaxHp()) {
             return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "target_not_full", "targetName", target.getCharacterName()), null);
        }

        combatManager.startCombat(attacker, target, "SPAR");
        
        // 移除这里的返回消息，因为 CombatManager.startCombat 可能会广播消息，或者前端会收到 combat_start 事件
        // 如果前端只依赖这个返回消息来显示，那么保留它。
        // 用户反馈说显示了两次，说明 CombatManager 或者其他地方也发了消息。
        // 通常 startCombat 会广播 "xxx向xxx发起了切磋"。
        // 让我们检查一下 CombatManager。
        // 暂时先保留返回结果，但把 message 设为 null，或者修改前端不显示这个 message 如果有事件的话。
        // 但最简单的办法是这里不返回 message，让 CombatManager 处理通知。
        // 不过 GameResult 的 logs 字段通常用于回显给操作者。
        
        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), null, null);
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
