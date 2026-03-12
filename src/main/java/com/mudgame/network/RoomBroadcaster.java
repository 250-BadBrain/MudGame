package com.mudgame.network;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.engine.manager.NpcManager;
import com.mudgame.model.Player;
import com.mudgame.model.PlayerState;
import com.mudgame.protocal.GameResult;
import com.mudgame.utils.Logger;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.WebSocketSession;
import org.springframework.web.socket.TextMessage;

import java.util.*;

/**
 * 房间广播工具类
 * 负责将游戏事件（如玩家离开房间）广播给房间内的所有玩家
 */
@Component
public class RoomBroadcaster {
    private final Map<String, WebSocketSession> playerIdToSession;
    private final ObjectMapper objectMapper;
    private final PlayerManager playerManager;
    private final RoomManager roomManager;
    private final NpcManager npcManager;
    private final OnlineManager onlineManager;

    public RoomBroadcaster(
        Map<String, WebSocketSession> playerIdToSession,
        ObjectMapper objectMapper,
        PlayerManager playerManager,
        RoomManager roomManager,
        NpcManager npcManager,
        OnlineManager onlineManager) {
        this.playerIdToSession = playerIdToSession;
        this.objectMapper = objectMapper;
        this.playerManager = playerManager;
        this.roomManager = roomManager;
        this.npcManager = npcManager;
        this.onlineManager = onlineManager;
    }

    public void attachPlayerSessionMap(Map<String, WebSocketSession> playerIdToSession) {
        this.playerIdToSession.clear();
        this.playerIdToSession.putAll(playerIdToSession);
    }

    private boolean isPlayerDead(String playerId) {
        try {
            Player p = playerManager.getPlayer(playerId);
            return p != null && p.getState() == PlayerState.DEAD;
        } catch (Exception e) {
            return false;
        }
    }

    /**
     * 广播玩家断线事件到房间内的所有其他玩家
     */
    public void broadcastPlayerOffline(String roomId, String offlinePlayerId, String offlinePlayerName, List<String> playersInRoom) {
        if (playersInRoom == null || playersInRoom.isEmpty()) return;

        Map<String, Object> results = new HashMap<>();
        results.put("playerId", offlinePlayerId);
        results.put("playerName", offlinePlayerName);
        results.put("roomId", roomId);

        // 使用统一的 player_status_changed 子类型，包含 status 字段
        results.put("status", "offline");
        GameResult broadcastMsg = new GameResult(
            "event",
            "player_status_changed",
            true,
            System.currentTimeMillis(),
            results,
            null
        );

        for (String playerId : playersInRoom) {
            if (playerId.equals(offlinePlayerId)) continue;
            if (isPlayerDead(playerId)) continue; // 死人无法感知
            WebSocketSession session = playerIdToSession.get(playerId);
            if (session != null && session.isOpen()) {
                try {
                    sendJson(session, broadcastMsg);
                } catch (Exception e) {
                    Logger.error("广播player_status_changed(offline)失败，目标玩家: " + playerId + "，原因: " + e.getMessage());
                }
            }
        }
    }

    /**
     * 广播玩家上线/恢复连接事件到房间内的所有其他玩家
     */
    public void broadcastPlayerOnline(String roomId, String onlinePlayerId, String onlinePlayerName, List<String> playersInRoom) {
        if (playersInRoom == null || playersInRoom.isEmpty()) return;

        Map<String, Object> results = new HashMap<>();
        results.put("playerId", onlinePlayerId);
        results.put("playerName", onlinePlayerName);
        results.put("roomId", roomId);
        // 使用统一的 player_status_changed 子类型，包含 status 字段
        results.put("status", "online");
        results.put("message", onlinePlayerName + "恢复连接");
        GameResult broadcastMsg = new GameResult(
            "event",
            "player_status_changed",
            true,
            System.currentTimeMillis(),
            results,
            null
        );

        for (String playerId : playersInRoom) {
            if (playerId.equals(onlinePlayerId)) continue;
            if (isPlayerDead(playerId)) continue; // 死人无法感知
            WebSocketSession session = playerIdToSession.get(playerId);
            if (session != null && session.isOpen()) {
                try {
                    sendJson(session, broadcastMsg);
                } catch (Exception e) {
                    Logger.error("广播player_status_changed(online)失败，目标玩家: " + playerId + "，原因: " + e.getMessage());
                }
            }
        }
    }

    /**
     * 广播玩家离开房间事件到房间内的所有其他玩家
     * 
     * @param roomId 房间ID
     * @param leavingPlayerId 离开的玩家ID
     * @param leavingPlayerName 离开的玩家名称
     * @param roomName 房间名称
     * @param playersInRoom 房间内的玩家ID列表
     * @param updatedRoomData 更新后的房间数据（用于客户端同步）
     */
    public void broadcastPlayerLeft(String roomId, String leavingPlayerId, String leavingPlayerName, 
                                     String roomName, List<String> playersInRoom, Map<String, Object> updatedRoomData) {
        if (playersInRoom == null || playersInRoom.isEmpty()) {
            return;
        }

        // 创建广播消息
        Map<String, Object> results = new HashMap<>();
        results.put("playerName", leavingPlayerName);
        results.put("roomName", roomName);
        results.put("roomId", roomId);
        // 包含更新后的房间数据供前端同步
        if (updatedRoomData != null) {
            results.put("room", updatedRoomData);
        }

        GameResult broadcastMsg = new GameResult(
            "event",
            "player_left",
            true,
            System.currentTimeMillis(),
            results,
            null
        );

        // 广播给房间内的其他所有玩家
        for (String playerId : playersInRoom) {
            // 不发送给离开的玩家本身
            if (playerId.equals(leavingPlayerId)) {
                continue;
            }
            if (isPlayerDead(playerId)) continue; // 死人无法感知

            WebSocketSession session = playerIdToSession.get(playerId);
            if (session != null && session.isOpen()) {
                try {
                    sendJson(session, broadcastMsg);
                    Logger.info("广播玩家离开事件：" + leavingPlayerName + " 离开了 " + roomName + "，已发送给玩家 " + playerId);
                } catch (Exception e) {
                    Logger.error("广播消息发送失败，目标玩家: " + playerId + "，原因: " + e.getMessage());
                }
            }
        }
    }

    /**
     * 广播玩家进入房间事件到房间内的所有其他玩家
     * 
     * @param roomId 房间ID
     * @param enteringPlayerId 进入的玩家ID
     * @param enteringPlayerName 进入的玩家名称
     * @param roomName 房间名称
     * @param playersInRoom 房间内的玩家ID列表（包括新进入的玩家）
     * @param updatedRoomData 更新后的房间数据（用于客户端同步）
     */
    public void broadcastPlayerEntered(String roomId, String enteringPlayerId, String enteringPlayerName,
                                        String roomName, List<String> playersInRoom, Map<String, Object> updatedRoomData) {
        if (playersInRoom == null || playersInRoom.isEmpty()) {
            return;
        }

        // 创建广播消息
        Map<String, Object> results = new HashMap<>();
        results.put("playerName", enteringPlayerName);
        results.put("roomName", roomName);
        results.put("roomId", roomId);
        // 包含更新后的房间数据供前端同步
        if (updatedRoomData != null) {
            results.put("room", updatedRoomData);
        }

        GameResult broadcastMsg = new GameResult(
            "event",
            "player_entered",
            true,
            System.currentTimeMillis(),
            results,
            null
        );

        // 广播给房间内的其他所有玩家
        for (String playerId : playersInRoom) {
            // 不发送给进入的玩家本身
            if (playerId.equals(enteringPlayerId)) {
                continue;
            }
            if (isPlayerDead(playerId)) continue; // 死人无法感知

            WebSocketSession session = playerIdToSession.get(playerId);
            if (session != null && session.isOpen()) {
                try {
                    sendJson(session, broadcastMsg);
                    Logger.info("广播玩家进入事件：" + enteringPlayerName + " 进入了 " + roomName + "，已发送给玩家 " + playerId);
                } catch (Exception e) {
                    Logger.error("广播消息发送失败，目标玩家: " + playerId + "，原因: " + e.getMessage());
                }
            }
        }
    }

    /**
     * 广播公开聊天消息到所有在线玩家
     */
    public void broadcastPublicChat(String senderId, String senderName, String message) {
        Map<String, Object> results = new HashMap<>();
        results.put("senderName", senderName);
        results.put("message", message);

        GameResult broadcastMsg = new GameResult(
            "chat",
            "public",
            true,
            System.currentTimeMillis(),
            results,
            null
        );

        for (Map.Entry<String, WebSocketSession> entry : playerIdToSession.entrySet()) {
            if (isPlayerDead(entry.getKey())) continue; // 死人无法感知

            WebSocketSession session = entry.getValue();
            if (session != null && session.isOpen()) {
                try {
                    sendJson(session, broadcastMsg);
                } catch (Exception e) {
                    Logger.error("广播公开聊天失败，目标玩家: " + entry.getKey() + "，原因: " + e.getMessage());
                }
            }
        }
    }

    /**
     * 广播普通文本消息给房间内的所有玩家
     */
    public void broadcastToRoom(String roomId, String message) {
        broadcastToRoomExcept(roomId, message, null);
    }

    /**
     * 广播普通文本消息给房间内的所有玩家，排除指定玩家
     */
    public void broadcastToRoomExcept(String roomId, String message, String excludePlayerId) {
        try {
            com.mudgame.model.Room room = roomManager.getRoom(roomId);
            if (room == null) return;
            
            List<String> playersInRoom = room.getPlayersInRoom();
            if (playersInRoom == null || playersInRoom.isEmpty()) return;

            Map<String, Object> results = new HashMap<>();
            results.put("message", message);

            GameResult broadcastMsg = new GameResult(
                "event",
                "room_message", // 前端需要处理这个类型，或者复用 chat/room
                true,
                System.currentTimeMillis(),
                results,
                null
            );

            for (String playerId : playersInRoom) {
                if (excludePlayerId != null && playerId.equals(excludePlayerId)) {
                    continue;
                }
                if (isPlayerDead(playerId)) continue; // 死人无法感知

                WebSocketSession session = playerIdToSession.get(playerId);
                if (session != null && session.isOpen()) {
                    sendJson(session, broadcastMsg);
                }
            }
        } catch (Exception e) {
            Logger.error("广播房间消息失败: " + e.getMessage());
        }
    }

    /**
     * 广播房间数据更新（如战斗中血量变化）
     */
    public void broadcastRoomUpdate(String roomId) {
        try {
            com.mudgame.model.Room room = roomManager.getRoom(roomId);
            if (room == null) return;

            // 获取最新的房间数据（包含玩家列表和状态）
            // 这里我们需要构建类似于 get_room 指令返回的数据结构
            // 为了简化，我们假设前端只需要更新玩家列表的状态
            
            // 重新获取房间内的玩家详情
            List<Map<String, Object>> playersData = new ArrayList<>();
            List<String> invalidPlayerIds = new ArrayList<>();

            for (String pid : room.getPlayersInRoom()) {
                try {
                    com.mudgame.model.Player p = playerManager.getPlayer(pid);
                    if (p != null) {
                        Map<String, Object> pData = new HashMap<>();
                        pData.put("id", p.getId());
                        pData.put("name", p.getCharacterName());
                        pData.put("state", p.getState());
                        
                        // 构造 attributes 对象，以匹配前端 getHpPercent 的期望结构
                        Map<String, Object> attributes = new HashMap<>();
                        attributes.put("currentHp", p.getAttributes().getCurrentHp());
                        attributes.put("maxHp", p.getAttributes().getMaxHp());
                        attributes.put("currentMp", p.getAttributes().getCurrentMp());
                        attributes.put("maxMp", p.getAttributes().getMaxMp());
                        
                        pData.put("attributes", attributes);
                        
                        // 为了兼容性，保留外层的 hp/mp 字段（可选）
                        pData.put("hp", p.getAttributes().getCurrentHp());
                        pData.put("maxHp", p.getAttributes().getMaxHp());
                        pData.put("mp", p.getAttributes().getCurrentMp());
                        pData.put("maxMp", p.getAttributes().getMaxMp());
                        
                        playersData.add(pData);
                    }
                } catch (Exception e) {
                    // 捕获无效玩家ID异常，记录并标记移除
                    Logger.warn("广播更新时发现无效玩家ID: " + pid + "，将从房间移除");
                    invalidPlayerIds.add(pid);
                }
            }

            // 移除无效玩家并保存房间
            if (!invalidPlayerIds.isEmpty()) {
                room.getPlayersInRoom().removeAll(invalidPlayerIds);
                roomManager.saveRoom(room);
            }

            // 重新获取房间内的 NPC 详情
            List<Map<String, Object>> npcsData = new ArrayList<>();
            for (String npcId : room.getNpcs()) {
                try {
                    com.mudgame.model.Npc npc = npcManager.getNpc(npcId);
                    if (npc != null) {
                        Map<String, Object> nData = new HashMap<>();
                        nData.put("id", npc.getId());
                        nData.put("name", npc.getName());
                        
                        Map<String, Object> attributes = new HashMap<>();
                        if (npc.getAttributes() != null) {
                            attributes.put("currentHp", npc.getAttributes().getCurrentHp());
                            attributes.put("maxHp", npc.getAttributes().getMaxHp());
                            attributes.put("currentMp", npc.getAttributes().getCurrentMp());
                            attributes.put("maxMp", npc.getAttributes().getMaxMp());
                        }
                        nData.put("attributes", attributes);
                        
                        npcsData.add(nData);
                    }
                } catch (Exception e) {
                    // 忽略不存在的 NPC
                }
            }

            Map<String, Object> results = new HashMap<>();
            results.put("roomId", roomId);
            results.put("players", playersData);
            results.put("npcs", npcsData);

            GameResult broadcastMsg = new GameResult(
                "event",
                "room_update", 
                true,
                System.currentTimeMillis(),
                results,
                null
            );

            for (String playerId : room.getPlayersInRoom()) {
                WebSocketSession session = playerIdToSession.get(playerId);
                if (session != null && session.isOpen()) {
                    sendJson(session, broadcastMsg);
                }
            }
        } catch (Exception e) {
            Logger.error("广播房间更新失败: " + e.getMessage());
        }
    }

    /**
     * 发送消息给指定玩家
     */
    public void sendToPlayer(String playerId, GameResult message) {
        WebSocketSession session = playerIdToSession.get(playerId);
        if (session != null && session.isOpen()) {
            try {
                sendJson(session, message);
            } catch (Exception e) {
                Logger.error("发送消息给玩家失败: " + playerId + "，原因: " + e.getMessage());
            }
        }
    }

    private void sendJson(WebSocketSession session, Object payload) throws Exception {
        synchronized (session) {
            if (session.isOpen()) {
                session.sendMessage(new TextMessage(objectMapper.writeValueAsString(payload)));
            }
        }
    }
}
