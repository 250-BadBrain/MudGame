package com.mudgame.engine.handler.game;

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

import com.mudgame.model.PlayerState;

public class GoHandler extends GameCommandHandler {
    private static final String subtype = "go";
    private final PlayerManager playerManager;
    private final RoomManager roomManager;

    public GoHandler(PlayerManager playerManager, RoomManager roomManager) {
        this.playerManager = playerManager;
        this.roomManager = roomManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info(log_name + "接收到go指令，正在尝试移动");
        Player player;
        Room curRoom,tarRoom;
        Map<String, Object> results = new HashMap<>();

        player = playerManager.getPlayer(message.getPlayerId());
        
        // 【新增】检查玩家状态
        if (player.getState() != PlayerState.IDLE) {
            if (player.getState() == PlayerState.DEAD) {
                return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "dead_state"), null);
            }
            
            String errorState = "busy_state";
            switch (player.getState()) {
                case HEALING: errorState = "healing_state"; break;
                case MEDITATING: errorState = "meditating_state"; break;
                case MINING: errorState = "mining_state"; break;
                case LEARNING: errorState = "learning_state"; break;
                case COMBAT: errorState = "combat_state"; break;
                default: break;
            }
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", errorState), null);
        }

        curRoom = roomManager.getRoom(player.getRoomId());

        if (message.getArgs() == null || message.getArgs().isEmpty()) {
            Logger.error(log_name + "没有接收到所需参数");
            throw new UnknownCommandException(type,subtype,"No arguments passed");
        }

        String dir = (String) message.getArgs().getOrDefault("direction", "");
        if (dir.isEmpty()) {
            Logger.error(log_name + "参数名有误或方向为空");
            throw new UnknownCommandException(type,subtype,"missing or empty direction parameter (key should be 'direction')");
        }

        String roomId = curRoom.getExits().getOrDefault(dir, null);
        if (roomId == null || roomId.isEmpty()) {
            Logger.warn(log_name + player.getId() + "尝试向" + dir + "移动，但那里不存在房间！");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "no_exit_in_that_direction"), null);
        }
        
        tarRoom = roomManager.getRoom(roomId);
        if (tarRoom == null) {
            Logger.error(log_name + "移动失败，目标房间 [" + roomId + "] Room 对象获取失败！");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "target_room_not_found"), null);
        }

        // 【新增】记录旧房间的信息用于广播
        String oldRoomId = player.getRoomId();
        String oldRoomName = curRoom.getName();
        String playerName = player.getCharacterName();
        List<String> playersInOldRoom = new ArrayList<>(curRoom.getPlayersInRoom());

        player.setRoomId(roomId);
        curRoom.removePlayer(player.getId());
        tarRoom.addPlayer(player.getId());

        playerManager.savePlayer(player);
        roomManager.saveRoom(curRoom);
        roomManager.saveRoom(tarRoom);
        
        Map <String,Object>roomDataForClient = roomManager.getRoomDataForClient(roomId);

        if (roomDataForClient == null) {
            Logger.error(log_name + "移动失败，目标房间 [" + roomId + "] 数据获取失败！");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "room_data_error"), null);
        }

        results.put("player", player);
        results.put("room", roomDataForClient);

        Logger.info(log_name + player.getId() + "成功移动到" + tarRoom.getId());
        
        // 【新增】广播事件
        RoomBroadcaster broadcaster = (RoomBroadcaster) message.getArg("__broadcaster");
        if (broadcaster != null) {
            // 1. 向旧房间的其他玩家广播"玩家离开"事件
            if (!playersInOldRoom.isEmpty()) {
                Map<String, Object> oldRoomData = roomManager.getRoomDataForClient(oldRoomId);
                broadcaster.broadcastPlayerLeft(oldRoomId, player.getId(), playerName, oldRoomName, playersInOldRoom, oldRoomData);
            }
            
            // 2. 向新房间的其他玩家广播"玩家进入"事件
            List<String> playersInNewRoom = new ArrayList<>(tarRoom.getPlayersInRoom());
            if (!playersInNewRoom.isEmpty()) {
                broadcaster.broadcastPlayerEntered(roomId, player.getId(), playerName, tarRoom.getName(), playersInNewRoom, roomDataForClient);
            }
        }
        
        return new GameResult(type, subtype, true, System.currentTimeMillis(), results, null);
    }

    @Override
    public String getType() {
        return type;
    }

    @Override
    public String getSubType() {
        return subtype;
    }
}//使玩家移动