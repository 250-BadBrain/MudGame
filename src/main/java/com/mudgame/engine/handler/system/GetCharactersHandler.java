package com.mudgame.engine.handler.system;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.exception.LoadException;
import com.mudgame.exception.UnknownCommandException;
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
 * 处理获取角色列表的系统命令 (system.get_characters)。
 * 用于在角色选择界面展示当前账户下的所有角色。
 */
public class GetCharactersHandler extends SystemCommandHandler {
    private static final String subtype = "get_characters";

    // 依赖项
    // 假设 UserStorage 负责加载包含 characterIds 的 User 对象
    private final UserStorage userStorage;
    private final PlayerManager playerManager;

    public GetCharactersHandler(UserStorage userStorage, PlayerManager playerManager) {
        this.userStorage = userStorage;
        this.playerManager = playerManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info(log_name + "接收到get_characters指令，尝试获取角色列表");
        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();

        // 1. 获取必要的参数：userId (账户名)
        String userId = (String) message.getArg("userId");
        if (userId == null || userId.trim().isEmpty()) {
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "missing_user_id"), logs);
        }

        try {
            // 2. 加载 User 对象，获取角色 ID 列表
            User user = userStorage.loadUser(userId);
            List<String> characterIds = user.getCharacterIds();
            
            // 3. 遍历角色 ID，加载每个角色的摘要信息
            List<Map<String, Object>> characterSummaries = new ArrayList<>();
            for (String characterId : characterIds) {
                try {
                    // 从存储加载 Player 对象 (PlayerManager 内部调用 PlayerStorage)
                    Player player = playerManager.loadPlayer(characterId);
                    
                    // 构造客户端需要的角色摘要信息
                    Map<String, Object> summary = new HashMap<>();
                    summary.put("id", player.getId());
                    
                    // 【关键修改】：使用 Player.getCharacterName() 获取角色名称
                    summary.put("name", player.getCharacterName()); 
                    // 【注】暂不返回等级，后续统一实现角色属性系统时再添加

                    characterSummaries.add(summary);
                } catch (LoadException e) {
                    // 角色文件丢失，跳过该角色
                    Logger.warn(log_name + "角色文件丢失，ID：" + characterId + "，已跳过。原因：" + e.getMessage());
                    logs.add("WARN: Character file missing for ID " + characterId + ", skipped.");
                }
            }

            // 4. 返回成功结果
            results.put("characters", characterSummaries);

            Logger.info(log_name + "账户[" + userId + "] 成功获取角色列表，共 " + characterSummaries.size() + " 个角色。");
            
            return new GameResult(type, subtype, true, System.currentTimeMillis(), results, logs);

        } catch (LoadException e) {
            // User 文件不存在或加载失败
            Logger.error(log_name + "加载用户失败，账户：" + userId + "，原因：" + e.getMessage());
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "user_load_failed"), logs);
        } catch (Exception e) {
            Logger.error(log_name + "获取角色列表失败，账户：" + userId + "，原因：" + e);
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