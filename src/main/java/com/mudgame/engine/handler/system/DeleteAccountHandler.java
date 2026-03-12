package com.mudgame.engine.handler.system;

import com.mudgame.exception.UnknownCommandException;
import com.mudgame.exception.LoadException;
import com.mudgame.exception.SaveException;
import com.mudgame.model.User;
import com.mudgame.network.Auth.PasswordHasher;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.storage.PlayerStorage;
import com.mudgame.storage.UserStorage;
import com.mudgame.utils.Logger;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.model.Player;
import com.mudgame.model.Room;
import com.mudgame.network.OnlineManager;
import com.mudgame.network.RoomBroadcaster;

/**
 * 处理删除账号的系统命令 (system.delete_account)。
 * 删除账号时需要验证密码，并同时删除该账号下的所有角色。
 */
public class DeleteAccountHandler extends SystemCommandHandler {
    private static final String subtype = "delete_account";
    
    private final UserStorage userStorage;
    private final PlayerStorage playerStorage;
    private final PasswordHasher passwordHasher;
    private final PlayerManager playerManager;
    private final RoomManager roomManager;
    private final OnlineManager onlineManager;

    public DeleteAccountHandler(
        UserStorage userStorage,
        PlayerStorage playerStorage,
        PasswordHasher passwordHasher,
        PlayerManager playerManager,
        RoomManager roomManager,
        OnlineManager onlineManager) {
        this.userStorage = userStorage;
        this.playerStorage = playerStorage;
        this.passwordHasher = passwordHasher;
        this.playerManager = playerManager;
        this.roomManager = roomManager;
        this.onlineManager = onlineManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info(log_name + "接收到delete_account指令，尝试删除账号");
        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();
        
        // 1. 获取必要的参数：username 和 password
        String username = (String) message.getArg("username");
        String password = (String) message.getArg("password");
        
        if (username == null || username.trim().isEmpty()) {
            logs.add("ERROR: Missing username.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "missing_username"), logs);
        }

        if (password == null || password.trim().isEmpty()) {
            logs.add("ERROR: Missing password.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "missing_password"), logs);
        }

        try {
            // 2. 验证用户身份（验证密码）
            User user = userStorage.loadUser(username);
            boolean isValid = passwordHasher.matches(password, user.getPasswordHash());
            if (!isValid) {
                logs.add("ERROR: Invalid username or password.");
                return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                    Map.of("error", "invalid_credentials"), logs);
            }

            // 获取广播器
            RoomBroadcaster broadcaster = (RoomBroadcaster) message.getArg("__broadcaster");

            // 3. 删除所有相关的角色
            for (String characterId : user.getCharacterIds()) {
                try {
                    // 清理角色数据（从房间移除、内存移除等）
                    removeCharacterFromAllRooms(characterId, broadcaster);

                    if (playerStorage.deletePlayer(characterId)) {
                        Logger.info(log_name + "删除角色文件：" + characterId);
                    }
                } catch (Exception e) {
                    Logger.error(log_name + "删除角色文件失败：" + characterId + "，原因：" + e.getMessage());
                }
            }

            // 4. 删除用户账户文件
            if (userStorage.deleteUser(username)) {
                Logger.info(log_name + "删除用户文件：" + username);
            }

            Logger.info(log_name + "成功删除账号：" + username);
            logs.add("INFO: Account deleted successfully.");
            results.put("username", username);
            
            return new GameResult(type, subtype, true, System.currentTimeMillis(), results, logs);

        } catch (LoadException e) {
            Logger.error(log_name + "删除账号失败，用户不存在：" + username);
            logs.add("ERROR: User not found.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "user_not_found"), logs);
        } catch (SaveException e) {
            Logger.error(log_name + "删除账号失败，文件系统异常，用户：" + username + "，原因：" + e.getMessage());
            logs.add("ERROR: Failed to delete persisted account data.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "storage_error"), logs);
        } catch (Exception e) {
            Logger.error(log_name + "删除账号失败，用户：" + username + "，原因：" + e.getMessage());
            logs.add("ERROR: Server failed to delete account due to internal error.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "server_error"), logs);
        }
    }

    /**
     * 从所有房间中移除指定的角色 (复用 DeleteCharacterHandler 的逻辑)
     */
    private void removeCharacterFromAllRooms(String characterId, RoomBroadcaster broadcaster) {
        Player player = null;
        try {
            player = playerManager.getPlayer(characterId);
        } catch (Exception e) {
            Logger.warn(log_name + "加载角色 " + characterId + " 失败，将继续根据房间记录清理：" + e.getMessage());
        }

        String playerName = (player != null && player.getCharacterName() != null)
            ? player.getCharacterName()
            : characterId;

        Set<String> roomIds;
        try {
            roomIds = roomManager.getLoadedRoomIds();
        } catch (Exception e) {
            Logger.error(log_name + "房间尚未加载，无法清理角色：" + e.getMessage());
            return;
        }

        for (String roomId : roomIds) {
            Room room;
            try {
                room = roomManager.getRoom(roomId);
            } catch (Exception e) {
                continue;
            }

            if (room != null && room.isPlayerInRoom(characterId)) {
                List<String> playersBeforeRemoval = new ArrayList<>(room.getPlayersInRoom());
                room.removePlayer(characterId);
                roomManager.saveRoom(room);
                Logger.info(log_name + "从房间 " + roomId + " 中移除角色 " + characterId);

                if (broadcaster != null && player != null && !playersBeforeRemoval.isEmpty()) {
                    Map<String, Object> updatedRoomData = roomManager.getRoomDataForClient(roomId);
                    if (updatedRoomData != null) {
                        broadcaster.broadcastPlayerLeft(
                            roomId,
                            characterId,
                            playerName,
                            room.getName(),
                            playersBeforeRemoval,
                            updatedRoomData
                        );
                    }
                }
            }
        }

        onlineManager.setOffline(characterId);
        playerManager.evictPlayer(characterId);
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
