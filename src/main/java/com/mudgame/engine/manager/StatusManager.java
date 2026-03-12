package com.mudgame.engine.manager;

import com.mudgame.model.Attributes;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.network.RoomBroadcaster;
import com.mudgame.utils.Logger;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

public class StatusManager {
    private final ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(4);
    private final Map<String, ScheduledFuture<?>> statusTasks = new ConcurrentHashMap<>();
    private final RoomBroadcaster broadcaster;
    private final PlayerManager playerManager;
    private final ItemManager itemManager;
    private final SkillManager skillManager;

    public StatusManager(RoomBroadcaster broadcaster, PlayerManager playerManager, ItemManager itemManager, SkillManager skillManager) {
        this.broadcaster = broadcaster;
        this.playerManager = playerManager;
        this.itemManager = itemManager;
        this.skillManager = skillManager;
    }

    public void startHeal(Player player) {
        stopStatus(player, false); // Stop any existing status without manual stop prompt
        
        player.setState(PlayerState.HEALING);
        playerManager.savePlayer(player);
        
        // Broadcast start
        if (broadcaster != null) {
            broadcaster.broadcastToRoomExcept(player.getRoomId(), player.getCharacterName() + "运功疗伤，气色看起来好多了。", player.getId());
            broadcaster.broadcastRoomUpdate(player.getRoomId());
        }

        // Schedule periodic task (every 10 seconds)
        ScheduledFuture<?> task = scheduler.scheduleAtFixedRate(() -> {
            try {
                performHeal(player);
            } catch (Exception e) {
                Logger.error("Heal error: " + e.getMessage());
                stopStatus(player, false);
            }
        }, 10, 10, TimeUnit.SECONDS);

        statusTasks.put(player.getId(), task);
    }

    public void startMeditate(Player player) {
        stopStatus(player, false);
        
        player.setState(PlayerState.MEDITATING);
        playerManager.savePlayer(player);
        
        // Broadcast start
        if (broadcaster != null) {
            broadcaster.broadcastToRoomExcept(player.getRoomId(), player.getCharacterName() + "盘膝而坐，吐纳调息。", player.getId());
            broadcaster.broadcastRoomUpdate(player.getRoomId());
        }

        ScheduledFuture<?> task = scheduler.scheduleAtFixedRate(() -> {
            try {
                performMeditate(player);
            } catch (Exception e) {
                Logger.error("Meditate error: " + e.getMessage());
                stopStatus(player, false);
            }
        }, 10, 10, TimeUnit.SECONDS);

        statusTasks.put(player.getId(), task);
    }

    public void stopStatus(Player player) {
        stopStatus(player, true);
    }

    public void stopStatus(Player player, boolean notifyPlayer) {
        ScheduledFuture<?> task = statusTasks.remove(player.getId());
        if (task != null) {
            task.cancel(false);
        }
        
        if (player.getState() == PlayerState.HEALING || player.getState() == PlayerState.MEDITATING || player.getState() == PlayerState.MINING || player.getState() == PlayerState.LEARNING) {
            PlayerState oldState = player.getState();
            player.setState(PlayerState.IDLE);
            player.setLearningSkillId(null);
            playerManager.savePlayer(player);
            
            if (broadcaster != null) {
                String selfMsg = "你停止了动作。";
                String roomMsg = player.getCharacterName() + "停止了动作。";

                if (oldState == PlayerState.MINING) {
                    selfMsg = "你收起铁镐，停止了挖矿。";
                    roomMsg = player.getCharacterName() + "收起铁镐，停止了挖矿。";
                } else if (oldState == PlayerState.HEALING) {
                    selfMsg = "你停止了运功疗伤。";
                    roomMsg = player.getCharacterName() + "停止了运功疗伤。";
                } else if (oldState == PlayerState.MEDITATING) {
                    selfMsg = "你停止了打坐。";
                    roomMsg = player.getCharacterName() + "停止了打坐。";
                } else if (oldState == PlayerState.LEARNING) {
                    selfMsg = "你停止了学习。";
                    roomMsg = player.getCharacterName() + "停止了学习。";
                }

                broadcaster.broadcastToRoomExcept(player.getRoomId(), roomMsg, player.getId());
                if (notifyPlayer) {
                    broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "stop", true, System.currentTimeMillis(), java.util.Map.of("message", selfMsg), null));
                }
                broadcaster.broadcastRoomUpdate(player.getRoomId());
            }
        }
    }

    public void startMining(Player player) {
        stopStatus(player, false);
        
        // Check if player has pickaxe equipped
        Map<String, String> equipment = player.getEquipment();
        if (equipment == null) {
             if (broadcaster != null) {
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "mine", false, System.currentTimeMillis(), java.util.Map.of("error", "no_pickaxe"), java.util.List.of("你需要装备铁镐才能挖矿。")));
            }
            return;
        }
        String weaponId = equipment.get("WEAPON");
        if (weaponId == null || !weaponId.equals("iron_pickaxe")) {
             if (broadcaster != null) {
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "mine", false, System.currentTimeMillis(), java.util.Map.of("error", "no_pickaxe"), java.util.List.of("你需要装备铁镐才能挖矿。")));
            }
            return;
        }

        player.setState(PlayerState.MINING);
        playerManager.savePlayer(player);
        
        if (broadcaster != null) {
            broadcaster.broadcastToRoomExcept(player.getRoomId(), player.getCharacterName() + "挥舞着铁镐，开始卖力地挖矿。", player.getId());
            broadcaster.broadcastRoomUpdate(player.getRoomId());
            broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "mine", true, System.currentTimeMillis(), null, java.util.List.of("你开始挖矿...")));
        }

        ScheduledFuture<?> task = scheduler.scheduleAtFixedRate(() -> {
            try {
                performMining(player);
            } catch (Exception e) {
                Logger.error("Mining error: " + e.getMessage());
                stopStatus(player, false);
            }
        }, 10, 10, TimeUnit.SECONDS);

        statusTasks.put(player.getId(), task);
    }

    private void performMining(Player player) {
        if (player.getState() != PlayerState.MINING) {
            stopStatus(player, false);
            return;
        }
        
        // Check pickaxe again
        Map<String, String> equipment = player.getEquipment();
        if (equipment == null) {
             if (broadcaster != null) {
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "mine", false, System.currentTimeMillis(), java.util.Map.of("error", "no_pickaxe"), java.util.List.of("你的铁镐不见了，无法继续挖矿。")));
            }
            stopStatus(player, false);
            return;
        }
        String weaponId = equipment.get("WEAPON");
        if (weaponId == null || !weaponId.equals("iron_pickaxe")) {
             if (broadcaster != null) {
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "mine", false, System.currentTimeMillis(), java.util.Map.of("error", "no_pickaxe"), java.util.List.of("你的铁镐不见了，无法继续挖矿。")));
            }
            stopStatus(player, false);
            return;
        }

        Attributes attr = player.getAttributes();
        java.util.Random rand = new java.util.Random();
        
        int expGain = 20 + rand.nextInt(6); // 20-25
        int potGain = 40 + rand.nextInt(6); // 40-45
        
        attr.setExperience(attr.getExperience() + expGain);
        attr.setPotential(attr.getPotential() + potGain);
        
        String log = "你获得了" + expGain + "点经验，" + potGain + "点潜能";
        
        // Gem drop logic
        String gemDrop = null;
        if (rand.nextInt(100) < 10) { // 10% chance for gem
            String[] gems = {"cracked_yellow_gem", "cracked_blue_gem", "cracked_red_gem", "cracked_green_gem", "cracked_purple_gem"};
            String gemId = gems[rand.nextInt(gems.length)];
            player.getItems().merge(gemId, 1, (a, b) -> a + b);
            
            com.mudgame.model.Item gem = itemManager.getItem(gemId);
            if (gem != null) {
                gemDrop = "你挖到了一块" + gem.getName();
            }
        }
        
        playerManager.savePlayer(player);
        
        if (broadcaster != null) {
            java.util.List<String> logs = new java.util.ArrayList<>();
            logs.add(log);
            if (gemDrop != null) {
                logs.add(gemDrop);
            }
            
            broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "mine", true, System.currentTimeMillis(), 
                java.util.Map.of("experience", attr.getExperience(), "potential", attr.getPotential(), "inventory", player.getItems()), 
                logs));
        }
    }

    private void performHeal(Player player) {
        if (player.getState() != PlayerState.HEALING) {
            stopStatus(player, false);
            return;
        }

        Attributes attr = player.getAttributes();
        
        // Check full HP
        if (attr.getCurrentHp() >= attr.getMaxHp()) {
            if (broadcaster != null) {
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "heal", true, System.currentTimeMillis(), java.util.Map.of("message", "你的气血已满，无需继续疗伤。"), null));
            }
            stopStatus(player, false);
            return;
        }

        int healHp = 20; // Base heal amount
        // Calculate actual recovery possible
        int actualRecover = Math.min(attr.getMaxHp() - attr.getCurrentHp(), healHp);
        
        // Calculate MP cost: 40% of recovered HP
        int costMp = (int) (actualRecover * 0.4);
        if (costMp < 1) costMp = 1; // Minimum 1 MP

        if (attr.getCurrentMp() < costMp) {
            // Not enough MP
            if (broadcaster != null) {
                  broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "heal", false, System.currentTimeMillis(), java.util.Map.of("error", "mp_insufficient"), java.util.List.of("你的内力不足，无法继续疗伤。")));
                  broadcaster.broadcastToRoom(player.getRoomId(), player.getCharacterName() + "内力耗尽，不得不停止疗伤。");
            }
            stopStatus(player, false);
            return;
        }

        attr.setCurrentMp(attr.getCurrentMp() - costMp);
        attr.setCurrentHp(attr.getCurrentHp() + actualRecover);
        
        playerManager.savePlayer(player);
        
        if (broadcaster != null) {
            // 发送个人日志
            broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "heal", true, System.currentTimeMillis(), java.util.Map.of("message", "你运功疗伤，消耗了 " + costMp + " 点内力，恢复了 " + actualRecover + " 点气血。"), null));
            // 更新房间状态（血条）
            broadcaster.broadcastRoomUpdate(player.getRoomId());
        }
        
        if (attr.getCurrentHp() >= attr.getMaxHp()) {
             stopStatus(player, false);
        }
    }

    private void performMeditate(Player player) {
        if (player.getState() != PlayerState.MEDITATING) {
            stopStatus(player, false);
            return;
        }

        Attributes attr = player.getAttributes();
        
        // Check if reached limit
        if (attr.getTrainedMaxMp() >= attr.getMaxMp() && attr.getCurrentMp() >= attr.getTrainedMaxMp()) {
            if (broadcaster != null) {
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "meditate", true, System.currentTimeMillis(), java.util.Map.of("message", "你的内力修为已达瓶颈，无法继续精进。"), null));
            }
            stopStatus(player, false);
            return;
        }

        // If current MP < Trained Max MP, recover MP
        if (attr.getCurrentMp() < attr.getTrainedMaxMp()) {
            int recoverMp = 20; // Base recovery or formula? User didn't specify recovery formula, only max increase.
            // Keeping existing recovery logic but capping at TrainedMaxMp
            int oldMp = attr.getCurrentMp();
            int newMp = Math.min(attr.getTrainedMaxMp(), attr.getCurrentMp() + recoverMp);
            int recoveredMp = newMp - oldMp;
            attr.setCurrentMp(newMp);
            
            playerManager.savePlayer(player);
            
            if (broadcaster != null) {
                String msg = "你打坐吐纳，恢复了 " + recoveredMp + " 点内力。";
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "meditate", true, System.currentTimeMillis(), java.util.Map.of("message", msg), null));
                broadcaster.broadcastRoomUpdate(player.getRoomId());
            }
        } else {
            // Increase Trained Max MP
            // Formula: (Pre-natal Bone * 0.2 * (1 + Post-natal Bone * 0.01) + Flat Meditation Bonus) * % Meditation Bonus
            double increase = (attr.getConstitution() * 0.2 * (1 + attr.getPostConstitution() * 0.01) + attr.getFlatMeditationBonus()) * attr.getPctMeditationBonus();
            int increaseInt = (int) Math.ceil(increase);
            
            if (increaseInt < 1) increaseInt = 1;
            
            int oldMax = attr.getTrainedMaxMp();
            int newMax = Math.min(attr.getMaxMp(), oldMax + increaseInt);
            int actualIncrease = newMax - oldMax;
            
            if (actualIncrease > 0) {
                attr.setTrainedMaxMp(newMax);
                attr.setCurrentMp(newMax); // Keep current MP full
                
                // Recalculate HP (since HP depends on Max MP)
                // User said: "HP is calculated from Real Max Internal Force"
                // This is handled in recalculateDerivedAttributes? No, recalculateDerivedAttributes sets maxHp?
                // I need to check Attributes.recalculateDerivedAttributes.
                // It doesn't seem to set MaxHp based on Mp currently.
                // I should update Attributes.recalculateDerivedAttributes to set MaxHp based on TrainedMaxMp if that's the rule.
                // Or just update it here.
                // "玩家的气血是由实时的最大内力计算得出的" -> "Player's HP is calculated from real-time Max Internal Force".
                // I'll update recalculateDerivedAttributes later to include this.
                player.getAttributes().recalculateDerivedAttributes(); // This will update MaxHp if I add the formula there.
                
                playerManager.savePlayer(player);
                
                if (broadcaster != null) {
                    String msg = "你的最大内力增加了 " + actualIncrease + " 点。";
                    broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "meditate", true, System.currentTimeMillis(), java.util.Map.of("message", msg), null));
                    broadcaster.broadcastRoomUpdate(player.getRoomId());
                }
            } else {
                 if (broadcaster != null) {
                    broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "meditate", true, System.currentTimeMillis(), java.util.Map.of("message", "你的内力修为已达瓶颈。"), null));
                }
                stopStatus(player, false);
            }
        }
    }

    public void startLearning(Player player, String skillId) {
        stopStatus(player, false);
        
        player.setState(PlayerState.LEARNING);
        player.setLearningSkillId(skillId);
        playerManager.savePlayer(player);
        
        // Broadcast start
        if (broadcaster != null) {
            broadcaster.broadcastToRoomExcept(player.getRoomId(), player.getCharacterName() + "正在向赵老教习请教武艺。", player.getId());
            broadcaster.broadcastRoomUpdate(player.getRoomId());
            broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "learn", true, System.currentTimeMillis(), null, java.util.List.of("你开始向赵老教习请教...")));
        }

        ScheduledFuture<?> task = scheduler.scheduleAtFixedRate(() -> {
            try {
                performLearning(player);
            } catch (Exception e) {
                Logger.error("Learning error: " + e.getMessage());
                stopStatus(player, false);
            }
        }, 10, 10, TimeUnit.SECONDS);

        statusTasks.put(player.getId(), task);
    }

    private void performLearning(Player player) {
        if (player.getState() != PlayerState.LEARNING) {
            stopStatus(player, false);
            return;
        }
        
        String skillId = player.getLearningSkillId();
        if (skillId == null) {
            stopStatus(player, false);
            return;
        }

        Attributes attr = player.getAttributes();
        
        // Calculate potential cost
        // Formula: (Pre-natal Intelligence * 100 * (1 + Post-natal Intelligence * 0.1)) * % Learning Efficiency
        double cost = (attr.getIntelligence() * 100 * (1 + attr.getPostIntelligence() * 0.1)) * attr.getPctLearningBonus();
        int costInt = (int) Math.ceil(cost);
        
        if (attr.getPotential() <= 0) {
             if (broadcaster != null) {
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "learn", false, System.currentTimeMillis(), java.util.Map.of("error", "no_potential"), java.util.List.of("你的潜能不足，无法继续学习。")));
            }
            stopStatus(player, false);
            return;
        }
        
        long actualCost = Math.min(attr.getPotential(), costInt);
        
        // Capture old level
        int oldLevel = player.getSkillLevel(skillId);

        attr.setPotential(attr.getPotential() - actualCost);
        
        // Add to skill potential
        Map<String, Long> skills = player.getSkills();
        long currentSkillPot = skills.getOrDefault(skillId, 0L);
        long newSkillPot = currentSkillPot + actualCost;
        skills.put(skillId, newSkillPot);
        
        // Check level cap (300 from NPC)
        int currentLevel = player.getSkillLevel(skillId);
        if (currentLevel >= 300) {
             if (broadcaster != null) {
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "learn", true, System.currentTimeMillis(), java.util.Map.of("message", "这项技能你已经不输你的师傅了。"), null));
            }
            stopStatus(player, false);
            return;
        }
        
        playerManager.savePlayer(player);
        
        if (broadcaster != null) {
            String skillName = skillManager.getSkillName(skillId);
            String msg;
            if (currentLevel > oldLevel) {
                msg = "你的" + skillName + "等级提升了！";
            } else {
                msg = "你对" + skillName + "略有所悟";
            }
            broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "learn", true, System.currentTimeMillis(), java.util.Map.of("message", msg, "potential", attr.getPotential()), null));
        }
        
        if (attr.getPotential() <= 0) {
             if (broadcaster != null) {
                broadcaster.sendToPlayer(player.getId(), new com.mudgame.protocal.GameResult("command", "learn", true, System.currentTimeMillis(), java.util.Map.of("message", "你的潜能耗尽了。"), null));
            }
            stopStatus(player, false);
        }
    }
}
