package com.mudgame.engine.handler.system;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.exception.UnknownCommandException;
import com.mudgame.model.Player;
import com.mudgame.model.Room;
import com.mudgame.network.RoomBroadcaster;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.utils.Logger;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GetRoomHandler extends SystemCommandHandler {
    private static final String subtype = "get_room";
    private final PlayerManager playerManager;
    private final RoomManager roomManager;

    public GetRoomHandler(PlayerManager playerManager, RoomManager roomManager) {
        this.playerManager = playerManager;
        this.roomManager = roomManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info(log_name + "接收到get_room指令，正在尝试获取玩家信息及其位置信息");
        Map<String, Object> results = new HashMap<>();
        List<String> logs = new ArrayList<>();

        // 检查 playerId 是否为 null
        String playerId = message.getPlayerId();
        if (playerId == null) {
            Logger.error(log_name + "get_room 指令的 playerId 为 null，无法继续处理！");
            throw new UnknownCommandException(type, subtype, "Player ID is missing or null.");
        }

        Player player = playerManager.getPlayer(playerId);
        Logger.info(log_name + "玩家" + player.getCharacterName() + "已加载");
        Room room = roomManager.getRoom(player.getRoomId());

        boolean isFirstTimeInRoom = !room.isPlayerInRoom(player.getId());
        
        playerManager.savePlayer(player);
        if (isFirstTimeInRoom) { 
            room.addPlayer(player.getId());
            roomManager.saveRoom(room);
        }

        Map<String, Object> roomDataForClient = roomManager.getRoomDataForClient(room.getId());
        if (roomDataForClient == null) {
            Logger.error(log_name + "房间 [" + room.getId() + "] 客户端数据获取失败！");
            throw new UnknownCommandException(type,subtype,"Room data for client not found.");
        }

        results.put("room", roomDataForClient);
        results.put("player", player);

        Logger.info(log_name + "获取完成，同时" + player.getId() + "处于场景" + room.getId());
        logs.add("你进入了" + room.getName());

        // 如果这是玩家首次进入房间，向其他玩家广播
        if (isFirstTimeInRoom) {
            RoomBroadcaster broadcaster = (RoomBroadcaster) message.getArg("__broadcaster");
            if (broadcaster != null) {
                broadcaster.broadcastPlayerEntered(
                    room.getId(),
                    playerId,
                    player.getCharacterName(),
                    room.getName(),
                    room.getPlayersInRoom(),
                    roomDataForClient
                );
                Logger.info(log_name + "已向房间 " + room.getId() + " 广播：玩家 " + player.getCharacterName() + " 进入了房间");
            }
        }

        return new GameResult(type,subtype, true, System.currentTimeMillis(), results, logs);
    }

    @Override
    public String getType() {
        return type;
    }

    @Override
    public String getSubType() {
        return subtype;
    }
}// 用于登陆后加载玩家信息，并加载玩家所在房间。
