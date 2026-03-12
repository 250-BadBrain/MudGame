package com.mudgame.engine.handler.system;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.exception.UnknownCommandException;
import com.mudgame.exception.LoadException;
import com.mudgame.model.Player;
import com.mudgame.model.User;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.storage.UserStorage;
import com.mudgame.utils.Logger;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * 处理创建新角色的系统命令 (system.create_character)。
 * 必须在用户已登录 (已绑定 userId) 但未选择角色时调用。
 */
public class CreateCharacterHandler extends SystemCommandHandler {
    private static final String subtype = "create_character";
    
    private final PlayerManager playerManager;
    private final UserStorage userStorage;

    public CreateCharacterHandler(UserStorage userStorage, PlayerManager playerManager) {
        this.userStorage = userStorage;
        this.playerManager = playerManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info(log_name + "接收到create_character指令，尝试创建新角色");
        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();
        
        // 1. 获取必要的参数：characterName 和 userId 
        String characterName = (String) message.getArg("characterName");
        if (characterName != null) {
            characterName = characterName.trim();
        }
        // 假设 MudWebSocketHandler 将会话中绑定的 userId 注入到 message 的 arguments 中
        String userId = (String) message.getArg("userId"); 
        
        if (userId == null || userId.trim().isEmpty()) {
            Logger.error(log_name + "会话未绑定 userId，无法创建角色。");
            // 抛出 UnknownCommandException 供上层捕获
            throw new UnknownCommandException(type, subtype, "User not authenticated or userId is missing.");
        }

        if (characterName == null || characterName.trim().isEmpty()) {
            logs.add("ERROR: Missing character name.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "missing_character_name"), logs);
        }
        
        // 简单限制
        if (characterName.length() < 2 || characterName.length() > 16) {
            logs.add("ERROR: Character name length invalid.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "name_length_invalid"), logs);
        }

        // 获取并验证先天属性
        int strength = getIntArg(message, "strength");
        int constitution = getIntArg(message, "constitution");
        int agility = getIntArg(message, "agility");
        int intelligence = getIntArg(message, "intelligence");

        if (!validateAttributes(strength, constitution, agility, intelligence)) {
            logs.add("ERROR: Invalid attributes. Each must be 15-30, sum must be 80.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "invalid_attributes"), logs);
        }

        try {
            // 2. 检查账号是否已达到角色数量上限
            User user = userStorage.loadUser(userId);
            if (user.isCharacterLimitReached()) {
                String errorMsg = "该账号下角色数量已达上限";
                logs.add("ERROR: " + errorMsg);
                return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                    Map.of("error", errorMsg), logs);
            }
            
            // 3. 调用 Manager 创建角色：Manager 负责检查名字唯一性、保存 Player 和更新 User
            Player newPlayer = playerManager.createCharacter(userId, characterName, strength, constitution, agility, intelligence);

            // 4. 返回成功结果，包含新角色的 ID 和名称
            results.put("characterId", newPlayer.getId());
            results.put("characterName", newPlayer.getCharacterName());

            Logger.info(log_name + "账户[" + userId + "] 成功创建角色：" + characterName + "(ID:" + newPlayer.getId() + ")");
            logs.add("INFO: Character '" + characterName + "' created successfully.");
            
            return new GameResult(type, subtype, true, System.currentTimeMillis(), results, logs);

        } catch (LoadException e) {
            Logger.error(log_name + "加载用户信息失败：" + userId);
            logs.add("ERROR: User not found.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "user_not_found"), logs);
        } catch (IllegalArgumentException e) {
            // 角色名已存在异常，由 PlayerManager 抛出
            logs.add("ERROR: Character name '" + characterName + "' already exists.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "name_taken"), logs);
        } catch (Exception e) {
            Logger.error(log_name + "创建角色失败，账户：" + userId + "，原因：" + e.getMessage());
            logs.add("ERROR: Server failed to create character due to internal error.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "server_error"), logs);
        }
    }

    @Override
    public String getType() {
        return type;
    }

    @Override
    public String getSubType() {
        return subtype;
    }

    private int getIntArg(CommandMessage message, String key) {
        Object val = message.getArg(key);
        if (val instanceof Number number) {
            return number.intValue();
        }
        // 如果是字符串尝试解析
        if (val instanceof String string) {
            try {
                return Integer.parseInt(string);
            } catch (NumberFormatException e) {
                return 0;
            }
        }
        return 0;
    }

    private boolean validateAttributes(int str, int con, int agi, int intel) {
        if (str < 15 || str > 30) return false;
        if (con < 15 || con > 30) return false;
        if (agi < 15 || agi > 30) return false;
        if (intel < 15 || intel > 30) return false;
        return (str + con + agi + intel) == 80;
    }
}