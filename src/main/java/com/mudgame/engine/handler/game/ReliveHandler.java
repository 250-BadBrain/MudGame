package com.mudgame.engine.handler.game;

import com.mudgame.engine.handler.CommandHandler;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.model.Attributes;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.model.Room;
import com.mudgame.network.RoomBroadcaster;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.Map;

public class ReliveHandler implements CommandHandler {
    private final PlayerManager playerManager;
    private final RoomManager roomManager;

    public ReliveHandler(PlayerManager playerManager, RoomManager roomManager) {
        this.playerManager = playerManager;
        this.roomManager = roomManager;
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "relive";
    }

    @Override
    public GameResult handle(CommandMessage command) {
        String playerId = command.getPlayerId();
        Player player = playerManager.getPlayer(playerId);
        
        if (player == null) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "玩家不存在"), null);
        }
        
        if (player.getState() != PlayerState.DEAD) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "你还没死呢，复活什么？"), null);
        }

        // 恢复状态
        player.setState(PlayerState.IDLE);
        player.setTargetId(null);
        player.setCombatType(null);
        Attributes attr = player.getAttributes();
        attr.setCurrentHp(attr.getMaxHp());
        attr.setCurrentMp(attr.getTrainedMaxMp());
        
        // 移动到复活点 (假设是洛阳中心 ly_00)
        String oldRoomId = player.getRoomId();
        String respawnRoomId = "ly_00";
        
        Room oldRoom = roomManager.getRoom(oldRoomId);
        if (oldRoom != null) {
            oldRoom.removePlayer(playerId);
        }
        
        Room newRoom = roomManager.getRoom(respawnRoomId);
        if (newRoom != null) {
            newRoom.addPlayer(playerId);
            player.setRoomId(respawnRoomId);
        } else {
            // 如果找不到复活点，就在原地复活
            if (oldRoom != null) {
                oldRoom.addPlayer(playerId);
            }
        }
        
        playerManager.savePlayer(player);
        if (oldRoom != null) {
            roomManager.saveRoom(oldRoom);
        }
        if (newRoom != null && newRoom != oldRoom) {
            roomManager.saveRoom(newRoom);
        }
        
        // 广播
        RoomBroadcaster broadcaster = (RoomBroadcaster) command.getArgs().get("__broadcaster");
        if (broadcaster != null) {
            // 通知旧房间玩家离开
            if (oldRoom != null) {
                // 获取更新后的房间数据（不包含已离开的玩家）
                // 尸体消失不需要广播"离开"，只需要更新房间数据（移除尸体）
                broadcaster.broadcastRoomUpdate(oldRoomId);
            }
            // 通知新房间玩家进入
            if (newRoom != null && newRoom != oldRoom) {
                Map<String, Object> newRoomData = roomManager.getRoomDataForClient(respawnRoomId);
                broadcaster.broadcastPlayerEntered(respawnRoomId, playerId, player.getCharacterName(), newRoom.getName(), newRoom.getPlayersInRoom(), newRoomData);
                broadcaster.broadcastRoomUpdate(respawnRoomId);
            } else if (newRoom != null) {
                broadcaster.broadcastRoomUpdate(respawnRoomId);
            }
        }

        // 返回结果，前端应该处理这个结果并刷新页面或重新获取房间信息
        // 这里我们返回一个特殊的指令让前端重新获取房间
        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), 
            Map.of(
                "message", "你感觉一股暖流流遍全身，重新活了过来！",
                "action", "refresh_room" // 暗示前端刷新
            ), null);
    }
}
