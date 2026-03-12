package com.mudgame.engine.manager;

import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;

import com.mudgame.exception.LoadException;
import com.mudgame.exception.SaveException;
import com.mudgame.model.ExitInfo;
import com.mudgame.model.Npc;
import com.mudgame.model.Player;
import com.mudgame.model.Room;
import com.mudgame.model.World;
import com.mudgame.network.OnlineManager;
import com.mudgame.storage.RoomRepository;
import com.mudgame.storage.RoomStorage;
import com.mudgame.utils.Logger;

public class RoomManager {
    private final Map<String,Room>rooms=new ConcurrentHashMap<>();
    private volatile boolean loaded = false;
    private final RoomStorage roomStorage;
    private final WorldManager worldManager;
    private final PlayerManager playerManager;
    private final OnlineManager onlineManager;
    private final NpcManager npcManager;

    public RoomManager(RoomStorage roomStorage, WorldManager worldManager, PlayerManager playerManager, OnlineManager onlineManager, NpcManager npcManager){
        this.roomStorage=roomStorage;
        this.worldManager = worldManager;
        this.playerManager = playerManager;
        this.onlineManager = onlineManager;
        this.npcManager = npcManager;
    }

    public void loadAllRooms() throws LoadException{
        if(loaded){
            throw new IllegalStateException("manager：房间已加载，不能重复加载");
        }
        Set<String> roomIds=RoomRepository.getRoomIds();
        for(String roomId:roomIds){
            Room room=this.roomStorage.loadRoom(roomId);
            rooms.put(roomId,room);
        }
        this.loaded=true;
        Logger.info("manager：加载所有房间成功");
    }

    public Room getRoom(String roomId){
        if(!loaded){
            throw new IllegalStateException("manager：物品未加载，无法获取房间");
        }
        Room room=rooms.get(roomId);
        if(room==null){
            throw new IllegalArgumentException("manager: 物品参数错误");
        }
        return room;
    }

    public void saveRoom(String roomId){
        try {
            roomStorage.saveRoom(getRoom(roomId));
        } catch (SaveException e) {
            Logger.error("manager：保存房间失败");
        }
    }

    public void saveRoom(Room room){
        try {
            roomStorage.saveRoom(room);
        } catch (SaveException e) {
            Logger.error("manager：保存房间失败");
        }
    }

    public Set<String> getLoadedRoomIds() {
        if (!loaded) {
            throw new IllegalStateException("manager：房间未加载，无法获取房间ID列表");
        }
        return Collections.unmodifiableSet(rooms.keySet());
    }

    public Map<String, Object> getRoomDataForClient(String roomId) {
        Room rawRoom = getRoom(roomId);
        if (rawRoom == null) {
            return null;
        }

        // 1. 提取基础信息
        Map<String, Object> assembledData = new HashMap<>();
        assembledData.put("id", rawRoom.getId());
        assembledData.put("name", rawRoom.getName());
        assembledData.put("description", rawRoom.getDescription());
        
        // 查找所属世界名称
        String worldName = "未知世界";
        try {
            for (World world : worldManager.getAllWorlds().values()) {
                if (world.getRooms().containsKey(roomId)) {
                    worldName = world.getName();
                    break;
                }
            }
        } catch (Exception e) {
            Logger.warn("获取房间所属世界失败: " + e.getMessage());
        }
        assembledData.put("worldName", worldName);
        
        // 2. 转换 Exits: Map<String, String> -> Map<String, ExitInfo>
        Map<String, ExitInfo> exitsForClient = new HashMap<>();
        for (Map.Entry<String, String> entry : rawRoom.getExits().entrySet()) {
            String targetId = entry.getValue();
            Room targetRoom = getRoom(targetId); 
            
            String targetName = (targetRoom != null) ? targetRoom.getName() : "未知区域";
            
            exitsForClient.put(entry.getKey(), new ExitInfo(targetId, targetName));
        }
        assembledData.put("exits", exitsForClient);
        
        // 3. 转换玩家ID为完整Player对象
        List<Map<String, Object>> playersData = rawRoom.getPlayersInRoom().stream()
            .filter(playerId -> {
                try {
                    playerManager.getPlayer(playerId);
                    return true;
                } catch (Exception e) {
                    Logger.warn("房间[" + roomId + "]中的玩家[" + playerId + "]不存在，已过滤");
                    return false;
                }
            })
            .map(playerId -> {
                try {
                    Player player = playerManager.getPlayer(playerId);
                    Map<String, Object> playerInfo = new HashMap<>();
                    playerInfo.put("id", player.getId());
                    playerInfo.put("name", player.getCharacterName());
                    playerInfo.put("title", player.getTitle());
                    playerInfo.put("userId", player.getUserId());
                    playerInfo.put("state", player.getState());
                    // 在线状态
                    boolean online = onlineManager.isOnline(playerId);
                    playerInfo.put("online", online);
                    
                    // 属性信息
                    if (player.getAttributes() != null) {
                        Map<String, Object> attrs = new HashMap<>();
                        attrs.put("maxHp", player.getAttributes().getMaxHp());
                        attrs.put("currentHp", player.getAttributes().getCurrentHp());
                        attrs.put("maxMp", player.getAttributes().getMaxMp());
                        attrs.put("currentMp", player.getAttributes().getCurrentMp());
                        attrs.put("attack", player.getAttributes().getAttack());
                        attrs.put("defense", player.getAttributes().getDefense());
                        playerInfo.put("attributes", attrs);
                    }
                    
                    return playerInfo;
                } catch (Exception e) {
                    Logger.error("无法获取玩家[" + playerId + "]的信息: " + e.getMessage());
                    Map<String, Object> unknownPlayer = new HashMap<>();
                    unknownPlayer.put("id", playerId);
                    unknownPlayer.put("name", "未知玩家");
                    return unknownPlayer;
                }
            })
            .collect(Collectors.toList());
        assembledData.put("playersInRoom", playersData); 
        
        // 4. 转换 NPC ID为完整NPC对象
        List<Map<String, Object>> npcsData = rawRoom.getNpcs().stream()
            .filter(npcId -> {
                try {
                    npcManager.getNpc(npcId);
                    return true;
                } catch (Exception e) {
                    Logger.warn("房间[" + roomId + "]中的NPC[" + npcId + "]不存在，已过滤");
                    return false;
                }
            })
            .map(npcId -> {
                try {
                    Npc npc = npcManager.getNpc(npcId);
                    Map<String, Object> npcInfo = new HashMap<>();
                    npcInfo.put("id", npc.getId());
                    npcInfo.put("name", npc.getName());
                    npcInfo.put("title", npc.getTitle());
                    npcInfo.put("description", npc.getDescription());
                    npcInfo.put("isShop", npc.getShopItems() != null && !npc.getShopItems().isEmpty());
                    npcInfo.put("skills", npc.getSkills()); // Send skills map
                    
                    // 属性信息
                    if (npc.getAttributes() != null) {
                        Map<String, Object> attrs = new HashMap<>();
                        attrs.put("maxHp", npc.getAttributes().getMaxHp());
                        attrs.put("currentHp", npc.getAttributes().getCurrentHp());
                        attrs.put("maxMp", npc.getAttributes().getMaxMp());
                        attrs.put("currentMp", npc.getAttributes().getCurrentMp());
                        attrs.put("attack", npc.getAttributes().getAttack());
                        attrs.put("defense", npc.getAttributes().getDefense());
                        npcInfo.put("attributes", attrs);
                    }
                    
                    return npcInfo;
                } catch (Exception e) {
                    Logger.error("无法获取NPC[" + npcId + "]的信息: " + e.getMessage());
                    Map<String, Object> unknownNpc = new HashMap<>();
                    unknownNpc.put("id", npcId);
                    unknownNpc.put("name", "未知NPC");
                    return unknownNpc;
                }
            })
            .collect(Collectors.toList());
        assembledData.put("npcs", npcsData);

        return assembledData;
    }
}
