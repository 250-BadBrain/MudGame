package com.mudgame.engine.handler.system;

import com.mudgame.exception.UnknownCommandException;
import com.mudgame.exception.LoadException;
import com.mudgame.exception.SaveException;
import com.mudgame.model.User;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.storage.PlayerStorage;
import com.mudgame.storage.UserStorage;
import com.mudgame.utils.Logger;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.model.Player;
import com.mudgame.model.Room;
import com.mudgame.network.OnlineManager;
import com.mudgame.network.RoomBroadcaster;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * 处理删除角色的系统命令 (system.delete_character)。
 */
public class DeleteCharacterHandler extends SystemCommandHandler {
    private static final String subtype = "delete_character";
    
    private final UserStorage userStorage;
    private final PlayerStorage playerStorage;
    private final PlayerManager playerManager;
    private final RoomManager roomManager;
    private final OnlineManager onlineManager;

    public DeleteCharacterHandler(
        UserStorage userStorage,
        PlayerStorage playerStorage,
        PlayerManager playerManager,
        RoomManager roomManager,
        OnlineManager onlineManager) {
        this.userStorage = userStorage;
        this.playerStorage = playerStorage;
        this.playerManager = playerManager;
        this.roomManager = roomManager;
        this.onlineManager = onlineManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info(log_name + "接收到delete_character指令，尝试删除角色");
        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();
        
        // 1. 获取必要的参数：characterId 和 userId
        String characterId = (String) message.getArg("characterId");
        String userId = (String) message.getArg("userId");
        
        if (userId == null || userId.trim().isEmpty()) {
            Logger.error(log_name + "会话未绑定 userId，无法删除角色。");
            throw new UnknownCommandException(type, subtype, "User not authenticated or userId is missing.");
        }

        if (characterId == null || characterId.trim().isEmpty()) {
            logs.add("ERROR: Missing character ID.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "missing_character_id"), logs);
        }

        try {
            // 2. 加载用户信息
            User user = userStorage.loadUser(userId);
            
            // 3. 验证角色是否属于该用户
            if (!user.getCharacterIds().contains(characterId)) {
                logs.add("ERROR: Character does not belong to this user.");
                return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                    Map.of("error", "character_not_owned"), logs);
            }
            
            // 4. 从所有房间中移除该角色并广播
            RoomBroadcaster broadcaster = (RoomBroadcaster) message.getArg("__broadcaster");
            removeCharacterFromAllRooms(characterId, broadcaster, logs);
            
            // 5. 删除角色文件
            if (!playerStorage.deletePlayer(characterId)) {
                logs.add("ERROR: Character file not found.");
                return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                    Map.of("error", "character_file_not_found"), logs);
            }
            
            // 6. 从用户的角色列表中移除该角色
            user.removeCharacterId(characterId);
            userStorage.saveUser(user);

            Logger.info(log_name + "账户[" + userId + "] 成功删除角色：" + characterId);
            logs.add("INFO: Character deleted successfully.");
            results.put("characterId", characterId);
            
            return new GameResult(type, subtype, true, System.currentTimeMillis(), results, logs);

        } catch (LoadException e) {
            Logger.error(log_name + "用户不存在：" + userId);
            logs.add("ERROR: User not found.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "user_not_found"), logs);
        } catch (SaveException e) {
            Logger.error(log_name + "删除角色文件失败，账户：" + userId + "，角色ID：" + characterId + "，原因：" + e.getMessage());
            logs.add("ERROR: Failed to delete character file.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "delete_failed"), logs);
        } catch (Exception e) {
            Logger.error(log_name + "删除角色失败，账户：" + userId + "，角色ID：" + characterId + "，原因：" + e.getMessage());
            logs.add("ERROR: Server failed to delete character due to internal error.");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), 
                Map.of("error", "server_error"), logs);
        }
    }

    /**
     * 从所有房间中移除指定的角色
     */
    private void removeCharacterFromAllRooms(String characterId, RoomBroadcaster broadcaster, List<String> logs) {
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

        boolean removedFromAnyRoom = false;

        for (String roomId : roomIds) {
            Room room;
            try {
                room = roomManager.getRoom(roomId);
            } catch (Exception e) {
                Logger.warn(log_name + "获取房间 " + roomId + " 时出错，跳过清理：" + e.getMessage());
                continue;
            }

            if (room != null && room.isPlayerInRoom(characterId)) {
                removedFromAnyRoom = true;
                List<String> playersBeforeRemoval = new ArrayList<>(room.getPlayersInRoom());
                room.removePlayer(characterId);
                roomManager.saveRoom(room);
                Logger.info(log_name + "从房间 " + roomId + " 中移除角色 " + characterId);
                if (logs != null) {
                    logs.add("INFO: Character removed from room " + roomId + ".");
                }

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

        if (!removedFromAnyRoom) {
            Logger.warn(log_name + "未在任何房间中找到角色 " + characterId + " 的残留记录。");
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