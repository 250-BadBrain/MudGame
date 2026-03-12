package com.mudgame.engine.manager;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.UUID; 
import java.util.concurrent.ConcurrentHashMap;

import com.mudgame.exception.LoadException;
import com.mudgame.exception.SaveException;
import com.mudgame.model.Player;
import com.mudgame.model.User;
import com.mudgame.model.support.PlayerDefaults; 
import com.mudgame.storage.PlayerStorage;
import com.mudgame.storage.UserStorage;
import com.mudgame.utils.Logger;

public class PlayerManager {
    private final Map<String, Player> players = new ConcurrentHashMap<>();
    private final PlayerStorage playerStorage;
    private final UserStorage userStorage;

    public PlayerManager(PlayerStorage playerStorage, UserStorage userStorage){
        this.playerStorage = playerStorage;
        this.userStorage = userStorage;
    }

    /**
     * 根据角色ID加载玩家数据 (原playerId现为characterId)
     * @param characterId 角色ID
     * @return 玩家对象
     * @throws LoadException 如果加载失败
     */
    public Player loadPlayer(String characterId) throws LoadException{
        Player player=this.playerStorage.loadPlayer(characterId);
        players.put(characterId, player);
        return player;
    }

    /**
     * 获取在线或从文件中加载的玩家对象 (原playerId现为characterId)
     * @param characterId 角色ID
     * @return 玩家对象
     */
    public Player getPlayer(String characterId){
        Player player=players.get(characterId);
        if(player==null){
            try {
                player=loadPlayer(characterId); 
            } catch (LoadException e) {
                Logger.error("manager：manager状态异常，加载玩家失败：" + e.getMessage());
                throw new IllegalStateException("manager：manager状态异常，加载玩家失败");
            }
            if(player==null){
                throw new IllegalArgumentException("manager: 玩家参数错误");
            }
        }
        return player;
    }
    
    /**
     * 创建新角色，并更新用户账户的角色列表。
     * @param userId 账户的用户名（作为用户ID）
     * @param characterName 新角色的名称
     * @param strength 臂力
     * @param constitution 根骨
     * @param agility 身法
     * @param intelligence 悟性
     * @return 新创建的玩家对象
     * @throws Exception 抛出 LoadException, SaveException 或 IllegalArgumentException (角色名重复)
     */
    public Player createCharacter(String userId, String characterName, int strength, int constitution, int agility, int intelligence) throws Exception {
        // 1. 检查角色名是否已存在
        // 假设 PlayerStorage.characterNameExists(String) 已经实现
        if (playerStorage.characterNameExists(characterName)) {
            Logger.warn("manager：角色名已存在：" + characterName);
            // 抛出运行时异常，供上层 Command Handler 捕获并返回 GameResult
            throw new IllegalArgumentException("Character name already exists.");
        }
        
        // 2. 生成唯一的角色 ID
        String characterId = UUID.randomUUID().toString();
        
        // 3. 创建新 Player 对象 (使用新的 PlayerDefaults.create 签名)
        Player newPlayer = PlayerDefaults.create(characterId, characterName, userId, strength, constitution, agility, intelligence); 
        
        // 4. 保存新角色档案
        playerStorage.savePlayer(newPlayer);
        
        // 5. 更新 User 账户信息
        User user = userStorage.loadUser(userId);
        user.addCharacterId(characterId);
        userStorage.saveUser(user);
        
        // 6. 加入运行时缓存并记录日志
        players.put(characterId, newPlayer);
        Logger.info("manager：新角色已创建：角色名[" + characterName + "] ID: " + characterId + ", 账户: " + userId);
        
        return newPlayer;
    }
    
    /**
     * 将玩家对象持久化保存到存储层。
     * @param player 要保存的玩家对象
     */
    public void savePlayer(Player player){
        try {
            playerStorage.savePlayer(player);
        } catch (SaveException e) {
            Logger.error("manager：玩家数据持久化失败！"+e);
        }
    }

    /**
     * 从运行时缓存中移除指定角色，通常在删除角色或账户时调用。
     * 该操作不会触发持久化，只影响当前 JVM 中的缓存状态。
     */
    public void evictPlayer(String characterId) {
        if (characterId == null || characterId.isBlank()) {
            return;
        }
        players.remove(characterId);
    }

    public List<Player> getLoadedPlayersSnapshot() {
        return new ArrayList<>(players.values());
    }

    public void performMigration() {
        Logger.info("开始执行玩家数据迁移...");
        java.util.List<String> ids = playerStorage.getAllPlayerIds();
        int count = 0;
        for (String id : ids) {
            try {
                Player player = playerStorage.loadPlayer(id);
                boolean changed = false;
                
                // 1. Assign initial skills if missing or low level
                Map<String, Long> skills = player.getSkills();
                if (skills == null) {
                    skills = new java.util.HashMap<>();
                    player.setSkills(skills);
                }
                
                // Basic Internal Lv 10 (2500 potential)
                if (!skills.containsKey("basic_internal") || skills.get("basic_internal") < 2500) {
                    skills.put("basic_internal", 2500L);
                    changed = true;
                }
                
                // Basic Fist Lv 10 (2500 potential)
                if (!skills.containsKey("basic_fist") || skills.get("basic_fist") < 2500) {
                    skills.put("basic_fist", 2500L);
                    changed = true;
                }
                
                // 2. Recalculate Attributes
                // Always recalculate to ensure new formulas (MP/HP) are applied
                player.updateSkillBonuses();
                
                // 3. Fix Trained Max MP (Set to Limit if 0 or less than Limit for migration)
                // User requested to recalculate Trained MP. 
                // Since we are giving initial skills, we should ensure they have the MP to use them.
                if (player.getAttributes().getTrainedMaxMp() < player.getAttributes().getMaxMp()) {
                    player.getAttributes().setTrainedMaxMp(player.getAttributes().getMaxMp());
                    changed = true;
                }
                
                // 4. Reset HP/MP to full
                player.getAttributes().setCurrentMp(player.getAttributes().getTrainedMaxMp());
                player.getAttributes().setCurrentHp(player.getAttributes().getMaxHp());
                changed = true; // Always save to apply attribute formula updates
                
                if (changed) {
                    playerStorage.savePlayer(player);
                    count++;
                }
                
            } catch (Exception e) {
                Logger.error("迁移玩家 " + id + " 失败: " + e.getMessage());
            }
        }
        Logger.info("玩家数据迁移完成，共更新 " + count + " 个玩家档案。");
    }
}