package com.mudgame.engine.manager;

import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.network.RoomBroadcaster;
import com.mudgame.utils.Logger;

import java.util.List;
import java.util.Map;
import java.util.Random;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

public class CombatManager {
    // 使用多线程调度器，提高并发处理能力
    private final ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(Runtime.getRuntime().availableProcessors());
    private final Map<String, ScheduledFuture<?>> combatTasks = new ConcurrentHashMap<>();
    private final Random random = new Random();
    private final RoomBroadcaster broadcaster;
    private final PlayerManager playerManager;

    public CombatManager(RoomBroadcaster broadcaster, PlayerManager playerManager) {
        this.broadcaster = broadcaster;
        this.playerManager = playerManager;
    }

    public void startCombat(Player attacker, Player defender, String type) {
        if (attacker.getState() == PlayerState.COMBAT || defender.getState() == PlayerState.COMBAT) {
            // Already in combat
            return;
        }

        attacker.setState(PlayerState.COMBAT);
        attacker.setTargetId(defender.getId());
        attacker.setCombatType(type);

        defender.setState(PlayerState.COMBAT);
        defender.setTargetId(attacker.getId());
        defender.setCombatType(type);

        // Broadcast start
        String msg = type.equals("KILL") 
            ? attacker.getCharacterName() + "对" + defender.getCharacterName() + "喝道：“纳命来！”"
            : attacker.getCharacterName() + "对" + defender.getCharacterName() + "抱拳道：“请赐教！”";
        
        if (broadcaster != null) {
            broadcaster.broadcastToRoom(attacker.getRoomId(), msg);
            broadcaster.broadcastRoomUpdate(attacker.getRoomId());
        }

        // Schedule attacks
        scheduleAttack(attacker);
        scheduleAttack(defender);
    }

    public void stopCombat(Player player) {
        if (player == null) return;

        cancelCombatTask(player);

        // Reset state
        if (player.getState() != PlayerState.DEAD) {
            player.setState(PlayerState.IDLE);
        }
        player.setTargetId(null);
        player.setCombatType(null);
        
        playerManager.savePlayer(player);
    }

    public boolean tryEscape(Player player) {
        if (player == null || player.getState() != PlayerState.COMBAT) {
            return false;
        }

        Player target = null;
        try {
            String targetId = player.getTargetId();
            if (targetId != null && !targetId.isBlank()) {
                target = playerManager.getPlayer(targetId);
            }
        } catch (Exception ignored) {
            target = null;
        }

        if (target == null || target.getState() == PlayerState.DEAD || !player.getRoomId().equals(target.getRoomId())) {
            leaveCombat(player, null, player.getCharacterName() + "趁乱脱离了战斗。", null);
            return true;
        }

        int selfDodge = player.getAttributes().getDodge();
        int targetDodge = target.getAttributes().getDodge();
        double escapeChance = 0.35 + ((double) selfDodge - targetDodge) / 1000.0;
        escapeChance = Math.max(0.15, Math.min(0.85, escapeChance));

        if (random.nextDouble() >= escapeChance) {
            if (broadcaster != null) {
                broadcaster.broadcastToRoom(player.getRoomId(), player.getCharacterName() + "试图抽身后退，却被" + target.getCharacterName() + "死死缠住！");
                broadcaster.broadcastRoomUpdate(player.getRoomId());
            }
            return false;
        }

        leaveCombat(player, target, player.getCharacterName() + "虚晃一招，跳出了战圈。", "失去了目标。");
        return true;
    }

    private void scheduleAttack(Player attacker) {
        double speed = attacker.getAttributes().getAttackSpeed();
        if (speed <= 0) speed = 4.0; // Default

        ScheduledFuture<?> task = scheduler.schedule(() -> {
            try {
                performAttack(attacker);
            } catch (Exception e) {
                Logger.error("Combat error: " + e.getMessage());
                stopCombat(attacker);
            }
        }, (long) (speed * 1000), TimeUnit.MILLISECONDS);

        combatTasks.put(attacker.getId(), task);
    }

    private void performAttack(Player attacker) {
        if (attacker.getState() != PlayerState.COMBAT) return;

        String targetId = attacker.getTargetId();
        Player target;
        try {
            target = playerManager.getPlayer(targetId);
        } catch (Exception e) {
            target = null;
        }

        // 1. 检查目标是否存在以及是否在同一房间
        if (target == null || !target.getRoomId().equals(attacker.getRoomId())) {
            if (broadcaster != null) {
                broadcaster.broadcastToRoom(attacker.getRoomId(), attacker.getCharacterName() + "茫然地四处张望，找不到对手了。");
            }
            resolveCombatTargetLoss(attacker, targetId);
            if (target != null) {
                resolveCombatTargetLoss(target, attacker.getId());
            }
            resolveOtherCombatantsTargeting(attacker.getRoomId(), targetId, attacker.getId());
            broadcastRoomUpdate(attacker.getRoomId());
            return;
        }

        // 2. 使用 tryLock 机制代替固定顺序锁
        java.util.concurrent.locks.ReentrantLock lock1 = attacker.getLock();
        java.util.concurrent.locks.ReentrantLock lock2 = target.getLock();
        
        boolean locked1 = false;
        boolean locked2 = false;
        
        try {
            while (true) {
                if (lock1.tryLock(10, TimeUnit.MILLISECONDS)) {
                    locked1 = true;
                    if (lock2.tryLock(10, TimeUnit.MILLISECONDS)) {
                        locked2 = true;
                        break;
                    } else {
                        lock1.unlock();
                        locked1 = false;
                        Thread.sleep(random.nextInt(20) + 5);
                    }
                } else {
                    Thread.sleep(random.nextInt(20) + 5);
                }
            }
            
            if (attacker.getState() != PlayerState.COMBAT) return;
            if (target.getState() == PlayerState.DEAD) {
                resolveCombatTargetLoss(attacker, target.getId());
                broadcastRoomUpdate(attacker.getRoomId());
                return;
            }

            // Recalculate attributes for both
            // recalculateAttributes(attacker);
            // recalculateAttributes(target);

            com.mudgame.model.Attributes attAttr = attacker.getAttributes();
            com.mudgame.model.Attributes defAttr = target.getAttributes();

            // 3. Hit Check (命中判定)
            int hitRate = attAttr.getHitRate();
            int dodge = defAttr.getDodge();
            double hitChance = 0.0;
            
            if (hitRate > dodge * 0.5) {
                if (hitRate >= dodge * 2.0) {
                    hitChance = 1.0;
                } else {
                    double ratio = (double)hitRate / dodge;
                    hitChance = 0.1 + (ratio - 0.5) * 0.6;
                }
            }
            
            boolean isHit = random.nextDouble() < hitChance;
            if (!isHit) {
                String msg = attacker.getCharacterName() + "发起攻击，但是被" + target.getCharacterName() + "躲开了！";
                if (broadcaster != null) {
                    broadcaster.broadcastToRoom(attacker.getRoomId(), msg);
                }
                // Reschedule even if miss
                scheduleAttack(attacker);
                return;
            }

            // 4. Damage Calculation
            // 预期伤害
            int attack = attAttr.getAttack();
            int defense = defAttr.getDefense();
            double fluctuation = 0.9 + (random.nextDouble() * 0.2);
            int expectedDamage = (int)(attack * fluctuation - defense);

            // Crit Check
            boolean isCrit = random.nextInt(100) < attAttr.getCritRate();
            if (isCrit) {
                expectedDamage *= 2;
            }

            // Parry Check (招架判定)
            int parry = defAttr.getParry();
            double parryChance = 0.0;
            // Parry vs Attacker Hit Rate
            if (parry > hitRate * 0.5) {
                if (parry >= hitRate * 2.0) {
                    parryChance = 1.0;
                } else {
                    double ratio = (double)parry / hitRate;
                    parryChance = 0.1 + (ratio - 0.5) * 0.6;
                }
            }
            
            boolean isParried = random.nextDouble() < parryChance;
            if (isParried) {
                expectedDamage /= 2;
            }

            // Final Damage
            int finalDamage = expectedDamage;
            String damageMsg = "";
            
            if (finalDamage <= 0) {
                finalDamage = 0;
                damageMsg = "但是没有造成任何伤害";
            } else {
                damageMsg = "造成了 " + finalDamage + " 点伤害！";
            }

            // Apply Damage
            int currentHp = defAttr.getCurrentHp();
            currentHp -= finalDamage;
            if (currentHp < 0) currentHp = 0;
            defAttr.setCurrentHp(currentHp);

            // 5. Broadcast
            StringBuilder sb = new StringBuilder();
            sb.append(attacker.getCharacterName());
            if (isCrit) sb.append("发动了致命一击，");
            else sb.append("发起攻击，");
            
            if (isParried) {
                sb.append("被").append(target.getCharacterName()).append("招架了，");
            }
            
            sb.append("对").append(target.getCharacterName()).append(damageMsg);
            
            if (broadcaster != null) {
                broadcaster.broadcastToRoom(attacker.getRoomId(), sb.toString());
            }
            
            broadcastRoomUpdate(attacker.getRoomId());

            // 6. Check End
            if ("KILL".equals(attacker.getCombatType())) {
                if (currentHp <= 0) {
                    handleDeath(attacker, target);
                    return;
                }
            } else { // SPAR
                if (currentHp < defAttr.getMaxHp() * 0.2) {
                    handleSparEnd(attacker, target);
                    return;
                }
            }

        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        } finally {
            if (locked2) lock2.unlock();
            if (locked1) lock1.unlock();
        }

        // Reschedule
        scheduleAttack(attacker);
    }

    private void handleDeath(Player winner, Player loser) {
        cancelCombatTask(winner);
        cancelCombatTask(loser);

        loser.getAttributes().setCurrentHp(0);
        loser.setState(PlayerState.DEAD);
        loser.setTargetId(null);
        loser.setCombatType(null);
        playerManager.savePlayer(loser);

        resolveCombatTargetLoss(winner, loser.getId());
        resolveOtherCombatantsTargeting(winner.getRoomId(), loser.getId(), winner.getId());

        String msg = loser.getCharacterName() + "大叫一声，倒在地上，死了！";
        if (broadcaster != null) {
            broadcaster.broadcastToRoom(winner.getRoomId(), msg);
        }
        broadcastRoomUpdate(winner.getRoomId());
    }

    private void handleSparEnd(Player winner, Player loser) {
        leaveCombat(winner, loser, null, null);

        String msg;
        if (random.nextBoolean()) {
            msg = loser.getCharacterName() + "后退三尺，抱拳道：佩服、佩服！";
        } else {
            msg = winner.getCharacterName() + "抱拳道：承让！";
        }
        if (broadcaster != null) {
            broadcaster.broadcastToRoom(winner.getRoomId(), msg);
        }
        broadcastRoomUpdate(winner.getRoomId());
    }

    private void broadcastRoomUpdate(String roomId) {
        if (broadcaster != null) {
            broadcaster.broadcastRoomUpdate(roomId);
        }
    }

    private void cancelCombatTask(Player player) {
        if (player == null) {
            return;
        }
        ScheduledFuture<?> task = combatTasks.remove(player.getId());
        if (task != null) {
            task.cancel(false);
        }
    }

    private void leaveCombat(Player escapingPlayer, Player primaryOpponent, String roomMessage, String opponentMessage) {
        if (escapingPlayer == null) {
            return;
        }

        String roomId = escapingPlayer.getRoomId();
        stopCombat(escapingPlayer);
        if (primaryOpponent != null) {
            resolveCombatTargetLoss(primaryOpponent, escapingPlayer.getId());
        }
        resolveOtherCombatantsTargeting(roomId, escapingPlayer.getId(), primaryOpponent != null ? primaryOpponent.getId() : null);

        if (broadcaster != null && roomMessage != null) {
            broadcaster.broadcastToRoom(roomId, roomMessage);
        }
        if (broadcaster != null && opponentMessage != null && primaryOpponent != null) {
            broadcaster.broadcastToRoom(roomId, primaryOpponent.getCharacterName() + opponentMessage);
        }
        broadcastRoomUpdate(roomId);
    }

    private void resolveCombatTargetLoss(Player combatant, String lostTargetId) {
        if (combatant == null || lostTargetId == null || combatant.getState() != PlayerState.COMBAT) {
            return;
        }
        if (!lostTargetId.equals(combatant.getTargetId())) {
            return;
        }

        cancelCombatTask(combatant);
        Player replacement = findReplacementTarget(combatant, lostTargetId);
        if (replacement == null) {
            stopCombat(combatant);
            return;
        }

        combatant.setTargetId(replacement.getId());
        playerManager.savePlayer(combatant);
        scheduleAttack(combatant);

        if (broadcaster != null) {
            broadcaster.broadcastToRoom(combatant.getRoomId(), combatant.getCharacterName() + "转身盯上了" + replacement.getCharacterName() + "！");
        }
    }

    private void resolveOtherCombatantsTargeting(String roomId, String lostTargetId, String excludeCombatantId) {
        if (roomId == null || lostTargetId == null) {
            return;
        }

        List<Player> loadedPlayers = playerManager.getLoadedPlayersSnapshot();
        for (Player player : loadedPlayers) {
            if (player == null || player.getState() != PlayerState.COMBAT) {
                continue;
            }
            if (excludeCombatantId != null && excludeCombatantId.equals(player.getId())) {
                continue;
            }
            if (!roomId.equals(player.getRoomId())) {
                continue;
            }
            if (!lostTargetId.equals(player.getTargetId())) {
                continue;
            }
            resolveCombatTargetLoss(player, lostTargetId);
        }
    }

    private Player findReplacementTarget(Player combatant, String lostTargetId) {
        if (combatant == null || combatant.getRoomId() == null) {
            return null;
        }

        for (Player candidate : playerManager.getLoadedPlayersSnapshot()) {
            if (candidate == null) {
                continue;
            }
            if (candidate.getId().equals(combatant.getId()) || candidate.getId().equals(lostTargetId)) {
                continue;
            }
            if (candidate.getState() != PlayerState.COMBAT || candidate.getTargetId() == null) {
                continue;
            }
            if (!combatant.getRoomId().equals(candidate.getRoomId())) {
                continue;
            }
            if (!combatant.getId().equals(candidate.getTargetId())) {
                continue;
            }
            if (combatant.getCombatType() != null && candidate.getCombatType() != null
                && !combatant.getCombatType().equals(candidate.getCombatType())) {
                continue;
            }
            return candidate;
        }
        return null;
    }
}
