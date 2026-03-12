package com.mudgame.engine.handler.system;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.exception.LoadException;
import com.mudgame.exception.UnknownCommandException;
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
 * 处理选择角色的系统命令 (system.select_character)。
 * 必须在用户已登录 (已绑定 userId) 但未选择角色时调用。
 * 成功后，需要 MudWebSocketHandler 将会话绑定到 characterId。
 */
public class SelectCharacterHandler extends SystemCommandHandler {
    private static final String subtype = "select_character";

    private final UserStorage userStorage;
    private final PlayerManager playerManager;

    public SelectCharacterHandler(UserStorage userStorage, PlayerManager playerManager) {
        this.userStorage = userStorage;
        this.playerManager = playerManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info(log_name + "接收到select_character指令，尝试选择角色");
        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();

        // 1. 获取必要的参数
        String characterId = (String) message.getArg("characterId");
        // 假设 MudWebSocketHandler 已经将会话中绑定的 userId 注入到 message 的 arguments 中
        String userId = (String) message.getArg("userId"); 

        if (userId == null || userId.trim().isEmpty()) {
            // 这不应该发生，但作为安全检查
            Logger.error(log_name + "会话未绑定 userId，无法选择角色。");
            throw new UnknownCommandException(type, subtype, "User not authenticated or userId is missing.");
        }
        
        if (characterId == null || characterId.trim().isEmpty()) {
            logs.add("ERROR: Missing character ID.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "missing_character_id"), logs);
        }

        try {
            // 2. 验证角色是否属于该用户
            User user = userStorage.loadUser(userId);
            if (!user.getCharacterIds().contains(characterId)) {
                Logger.warn(log_name + "账户[" + userId + "] 尝试选择不属于自己的角色ID：" + characterId);
                logs.add("ERROR: Character ID is invalid or does not belong to user.");
                return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "invalid_character_id"), logs);
            }
            
            // 3. 预加载玩家数据并校验
            // 虽然 PlayerManager.getPlayer 也会加载，但这里提前加载可以暴露文件错误
            playerManager.loadPlayer(characterId); 

            // 4. 选择成功，返回角色ID
            results.put("characterId", characterId);
            
            Logger.info(log_name + "账户[" + userId + "] 成功选择角色：" + characterId);
            logs.add("INFO: Character selected successfully. Ready to play.");
            
            return new GameResult(type, subtype, true, System.currentTimeMillis(), results, logs);

        } catch (LoadException e) {
            // 角色ID在用户的列表中，但文件丢失
            Logger.error(log_name + "选择角色失败，角色文件丢失：" + characterId + "，原因：" + e.getMessage());
            logs.add("ERROR: Character file not found on server.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "character_file_missing"), logs);
        } catch (Exception e) {
            Logger.error(log_name + "选择角色失败，账户：" + userId + "，原因：" + e.getMessage());
            logs.add("ERROR: Server failed to select character due to internal error.");
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
}