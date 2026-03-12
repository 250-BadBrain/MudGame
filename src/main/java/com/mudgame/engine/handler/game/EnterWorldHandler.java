package com.mudgame.engine.handler.game;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.engine.manager.WorldManager;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.model.Room;
import com.mudgame.model.World;
import com.mudgame.network.RoomBroadcaster;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class EnterWorldHandler extends GameCommandHandler {
    private final PlayerManager playerManager;
    private final RoomManager roomManager;
    private final WorldManager worldManager;

    public EnterWorldHandler(PlayerManager playerManager, RoomManager roomManager, WorldManager worldManager) {
        this.playerManager = playerManager;
        this.roomManager = roomManager;
        this.worldManager = worldManager;
    }

    @Override
    public GameResult handle(CommandMessage message) {
        String playerId = message.getPlayerId();
        Map<String, Object> args = message.getArgs();
        String worldId = args == null ? null : (String) args.get("worldId");
        
        if (worldId == null || worldId.isEmpty()) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "世界ID不能为空"), null);
        }
        
        try {
            World world = worldManager.getWorld(worldId);
            
            String prefix = "";
            if (worldId.equals("luoyang")) prefix = "ly";
            else if (worldId.equals("wudang")) prefix = "wd";
            else if (worldId.equals("shaolin")) prefix = "sl";
            else if (worldId.equals("emei")) prefix = "em";
            else if (worldId.equals("huashan")) prefix = "hs";
            else if (worldId.equals("kunlun")) prefix = "kl";
            else if (worldId.equals("kongtong")) prefix = "kt";
            else prefix = worldId; // Fallback
            
            String targetRoomId = prefix + "_00";
            
            Room targetRoom = roomManager.getRoom(targetRoomId);
            if (targetRoom == null) {
                 return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "无法找到该世界的入口: " + targetRoomId), null);
            }
            
            Player player = playerManager.getPlayer(playerId);
            if (player.getState() != PlayerState.IDLE) {
                if (player.getState() == PlayerState.DEAD) {
                    return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "dead_state"), null);
                }

                return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", "busy_state"), null);
            }

            String oldRoomId = player.getRoomId();
            if (targetRoomId.equals(oldRoomId)) {
                Map<String, Object> currentRoomData = roomManager.getRoomDataForClient(targetRoomId);
                return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), Map.of("room", currentRoomData, "player", player), List.of("你已经在" + world.getName() + "了。"));
            }
            
            // Leave old room
            RoomBroadcaster broadcaster = (RoomBroadcaster) message.getArgs().get("__broadcaster");
            if (oldRoomId != null) {
                Room oldRoom = roomManager.getRoom(oldRoomId);
                if (oldRoom != null) {
                    oldRoom.removePlayer(playerId);
                    roomManager.saveRoom(oldRoom);
                    // Broadcast leave
                    if (broadcaster != null) {
                        Map<String, Object> oldRoomData = roomManager.getRoomDataForClient(oldRoomId);
                        broadcaster.broadcastPlayerLeft(oldRoomId, playerId, player.getCharacterName(), oldRoom.getName(), new ArrayList<>(oldRoom.getPlayersInRoom()), oldRoomData);
                    }
                }
            }
            
            // Enter new room
            player.setRoomId(targetRoomId);
            targetRoom.addPlayer(playerId);
            playerManager.savePlayer(player);
            roomManager.saveRoom(targetRoom);
            
            // Prepare client data
            Map<String, Object> roomDataForClient = roomManager.getRoomDataForClient(targetRoomId);
            
            if (broadcaster != null) {
                broadcaster.broadcastPlayerEntered(targetRoomId, playerId, player.getCharacterName(), targetRoom.getName(), targetRoom.getPlayersInRoom(), roomDataForClient);
            }
            
            List<String> logs = new ArrayList<>();
            logs.add("你来到了" + world.getName() + "。");
            
            Map<String, Object> results = new HashMap<>();
            results.put("room", roomDataForClient);
            results.put("player", player);
            
            return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), results, logs);
            
        } catch (Exception e) {
            return new GameResult(getType(), getSubType(), false, System.currentTimeMillis(), Map.of("error", e.getMessage()), null);
        }
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "enter_world";
    }
}
