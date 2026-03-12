package com.mudgame.network;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.mudgame.engine.HandleCommand;
import com.mudgame.engine.manager.NpcManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.network.Auth.AuthHandler;
import com.mudgame.network.Auth.ReconnectTokenService;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.model.User;
import com.mudgame.storage.UserStorage;
import com.mudgame.utils.Logger;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.CloseStatus;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;
import org.springframework.web.socket.handler.TextWebSocketHandler;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.List;

import com.mudgame.model.Player;
import com.mudgame.model.Room;

@Component
public class MudWebSocketHandler extends TextWebSocketHandler {

    // 绑定到 Session Attributes 的键
    private static final String ATTR_USER_ID = "userId";       // 账户ID（用户名）
    private static final String ATTR_PLAYER_ID = "playerId";   // 角色ID（已选择的角色ID）
    private static final String ARG_RECONNECT_TOKEN = "reconnectToken";
    private final ObjectMapper mapper;
    
    // 在线会话表
    // playerIdToSession 和 sessionIdToPlayer 现在绑定的是已选定角色（Player/Character ID）
    private final Map<String, WebSocketSession> playerIdToSession = new ConcurrentHashMap<>();
    private final Map<String, String> sessionIdToPlayer = new ConcurrentHashMap<>();

    // 子处理器
    private final HandleCommand handleCommand;
    private final AuthHandler authHandler;
    
    // 广播器
    private final RoomBroadcaster roomBroadcaster;

    // 依赖
    private final UserStorage userStorage;
    private final ReconnectTokenService reconnectTokenService;
    private final OnlineManager onlineManager;
    private final PlayerManager playerManager;
    private final RoomManager roomManager;
    private final NpcManager npcManager;

    public MudWebSocketHandler(
        HandleCommand handleCommand,
        AuthHandler authHandler,
        ObjectMapper objectMapper,
        RoomBroadcaster roomBroadcaster,
        UserStorage userStorage,
        ReconnectTokenService reconnectTokenService,
        OnlineManager onlineManager,
        PlayerManager playerManager,
        RoomManager roomManager,
        NpcManager npcManager) {
        this.handleCommand = handleCommand;
        this.authHandler = authHandler;
        this.mapper = objectMapper;
        this.roomBroadcaster = roomBroadcaster;
        this.userStorage = userStorage;
        this.reconnectTokenService = reconnectTokenService;
        this.onlineManager = onlineManager;
        this.playerManager = playerManager;
        this.roomManager = roomManager;
        this.npcManager = npcManager;
        this.roomBroadcaster.attachPlayerSessionMap(playerIdToSession);
    }

    @SuppressWarnings("null")
    @Override
    public void handleTextMessage(WebSocketSession session, TextMessage message) {
        try {
            String payload = message.getPayload();
            CommandMessage cmd = mapper.readValue(payload, CommandMessage.class);
            
            GameResult result = null;
            String type = cmd.getType();
            String subType = cmd.getSubtype();
            Map<String, Object> args = cmd.getArgs() == null ? Map.of() : cmd.getArgs();

            if ("system".equals(type)) {
                // --- 系统指令：登录/注册 ---
                if ("login".equals(subType)) {
                    result = authHandler.handleLogin(cmd); // 调用 AuthHandler.handleLogin
                } else if ("register".equals(subType)) {
                    result = authHandler.handleRegister(cmd); // 调用 AuthHandler.handleRegister
                } else if ("delete_account".equals(subType)) {
                    // 💡 新增：处理账号删除请求
                    // 不需要检查 userId，因为前端会直接发送 username 和 password
                    result = handleCommand.handleCommand(cmd);
                } else if ("reconnect".equals(subType)) {
                    // 💡 新增：处理重新连接请求
                    // 前端在 WebSocket 连接建立后发送此消息，用来恢复之前的登录状态
                    String userId = (String) args.get("userId");
                    String playerId = (String) args.get("playerId");
                    String reconnectToken = (String) args.get(ARG_RECONNECT_TOKEN);
                    Logger.info("收到重新连接请求 - userId: " + userId + ", playerId: " + playerId);
                    
                    if (userId != null && !userId.isEmpty() && reconnectToken != null && !reconnectToken.isEmpty()) {
                        User user = userStorage.loadUser(userId);
                        if (!reconnectTokenService.isValid(user, reconnectToken)) {
                            result = sysFail("reconnect", "登录状态已失效，请重新登录");
                        } else {
                            if (playerId != null && !playerId.isEmpty() && !user.getCharacterIds().contains(playerId)) {
                                result = sysFail("reconnect", "角色信息无效，请重新选择角色");
                            } else {
                        // 将 userId 绑定到新的 session
                            session.getAttributes().put(ATTR_USER_ID, userId);
                            if (playerId != null && !playerId.isEmpty()) {
                                bindSessionToCharacter(session, playerId);
                            }
                            Logger.info("会话已绑定 userId: " + userId + "，会话ID: " + session.getId());
                            Map<String, Object> reconnectResults = new HashMap<>();
                            reconnectResults.put("message", "重新连接成功");
                            if (playerId != null && !playerId.isEmpty()) {
                                reconnectResults.put("playerId", playerId);
                            }
                            result = new GameResult(
                                "system",
                                "reconnect",
                                true,
                                System.currentTimeMillis(),
                                reconnectResults,
                                null
                            );
                            }
                        }
                    } else {
                        result = sysFail("reconnect", "缺少登录令牌，无法重新连接");
                    }
                } else {
                    // --- 系统指令：角色管理 (create_character, get_characters, select_character) 和游戏系统指令 (get_room) ---
                    // 获取当前会话绑定的 userId 和 playerId
                    String userId = (String) session.getAttributes().get(ATTR_USER_ID);
                    String playerId = (String) session.getAttributes().get(ATTR_PLAYER_ID);
                    
                    Logger.info("系统指令[" + subType + "] - 会话ID=" + session.getId() + ", userId=" + userId + ", playerId=" + playerId);
                    
                    if (userId == null) {
                        result = sysFail(subType, "未登录，无法进行该操作。请先登录。");
                    } else {
                        // 将 userId 注入到 command 中，供后端处理器使用
                        cmd.addArg("userId", userId);
                        cmd.addArg("__broadcaster", roomBroadcaster);
                        
                        // 对于需要 playerId 的系统指令（如 get_room），设置 playerId
                        if (playerId != null) {
                            cmd.setPlayerId(playerId);
                            Logger.info("系统指令[" + subType + "]已设置 playerId: " + playerId);
                        } else {
                            Logger.warn("系统指令[" + subType + "] 未设置 playerId（可能是游戏前的指令）");
                        }
                        
                        result = handleCommand.handleCommand(cmd);
                        
                        // 角色选择成功后，绑定 session 到角色 ID
                        if (result != null && result.getFlag() && "select_character".equals(subType)) {
                            String characterId = (String) result.getResults().get("characterId");
                            Logger.info("select_character 响应处理：flag=" + result.getFlag() + ", characterId=" + characterId);
                            if (characterId != null) {
                                bindSessionToCharacter(session, characterId);
                                Logger.info("会话绑定完成，会话ID=" + session.getId() + ", 角色ID=" + characterId);
                            } else {
                                Logger.warn("select_character 响应中 characterId 为 null");
                            }
                        }
                        
                        // 角色注销成功后，解绑角色
                        if (result != null && result.getFlag() && "deselect_character".equals(subType)) {
                            unbindCharacter(session);
                            Logger.info("会话[" + session.getId() + "] 已注销角色");
                        }
                    }
                }

                // --- 登录/注册成功后的会话绑定逻辑 ---
                if (result != null && result.getFlag()) {
                    if ("login".equals(subType) || "register".equals(subType)) {
                        // 注册/登录成功后，绑定 userId (即 username)
                        String userId = (String) result.getResults().get("userId");
                        if (userId != null) {
                            session.getAttributes().put(ATTR_USER_ID, userId);
                        }
                    }
                    // 【注意】不在此处绑定 playerId，因为它现在代表角色ID，需在角色选择后进行绑定
                }

            } else {
                // --- 游戏指令：command, chat ---
                // 检查是否已选择角色 (playerId 不为空)
                String playerId = (String) session.getAttributes().get(ATTR_PLAYER_ID);
                if (playerId == null) {
                    // 如果未绑定角色，返回未授权/未选角错误
                    result = sysFail("error", "未选择角色，请先登录并选择角色。");
                } else {
                    // 绑定角色ID，调用游戏命令处理器
                    cmd.setPlayerId(playerId);
                    
                    // 【新增】注入广播器到命令中
                    cmd.addArg("__broadcaster", roomBroadcaster);
                    
                    result = handleCommand.handleCommand(cmd);
                }
            }
            
            // 发送结果
            if (result != null) {
                sendJson(session, result);
            }

        } catch (Exception e) { 
            // 捕获所有后端异常（如 JSON 解析失败、UserStorage 初始/IO 异常）
            Logger.error("Error processing message for session " + session.getId() + ": " + e.getMessage());

            // 格式化错误信息为 system/error 消息，以便前端能够解析并解除卡死
            GameResult errorResult = new GameResult(
                "system", 
                "error", // 使用通用的 'error' 子类型
                false, 
                System.currentTimeMillis(), 
                Map.of("error", "服务器内部异常: " + e.getMessage() + "。请查看服务端日志。"), 
                null
            );
            sendJson(session, errorResult);
        }
    }

    // 绑定 session 到角色 ID (playerId)
    @SuppressWarnings("null")
    private void bindSessionToCharacter(WebSocketSession session, String characterId) {
        // 检查该角色是否已在其他会话中
        WebSocketSession existingSession = playerIdToSession.get(characterId);
        if (existingSession != null && existingSession.isOpen() && !existingSession.equals(session)) {
            // 旧会话存在且未关闭，发送 SESSION_REPLACED 事件给旧会话
            Logger.warn("角色[" + characterId + "]在新会话中被选择，将断开旧会话[" + existingSession.getId() + "]");
            try {
                GameResult replacedNotice = new GameResult(
                    "system",
                    "session_replaced",
                    false,
                    System.currentTimeMillis(),
                    Map.of("error", "该角色已在另一个窗口中被选择，当前连接已断开"),
                    null
                );
                sendJson(existingSession, replacedNotice);
                // 立即关闭旧会话
                existingSession.close(CloseStatus.NORMAL);
            } catch (Exception e) {
                Logger.error("关闭旧会话失败: " + e.getMessage());
            }
        }
        
        session.getAttributes().put(ATTR_PLAYER_ID, characterId);
        sessionIdToPlayer.put(session.getId(), characterId);
        playerIdToSession.put(characterId, session);
        Logger.info("会话[" + session.getId() + "] 已选择角色: " + characterId);

        // 标记为在线并广播给同房间玩家，通知恢复连接
        try {
            onlineManager.setOnline(characterId);

            // 获取玩家所在房间和姓名
            Player player = playerManager.getPlayer(characterId);
            String roomId = player.getRoomId();
            String playerName = player.getCharacterName();
            if (roomId != null && !roomId.isEmpty()) {
                Room room = roomManager.getRoom(roomId);
                if (room != null) {
                    List<String> playersInRoom = room.getPlayersInRoom();
                    roomBroadcaster.broadcastPlayerOnline(roomId, characterId, playerName, playersInRoom);
                }
            }
        } catch (Exception e) {
            Logger.error("bindSessionToCharacter 广播上线失败: " + e.getMessage());
        }
    }
    
    // 解绑角色（但不解绑用户，用于返回角色选择界面）
    private void unbindCharacter(WebSocketSession session) {
        String pid = sessionIdToPlayer.remove(session.getId());
        if (pid != null) {
            // 在移除映射之前，广播该玩家离线并标记为离线
            try {
                onlineManager.setOffline(pid);
                Player player = playerManager.getPlayer(pid);
                String roomId = player.getRoomId();
                String playerName = player.getCharacterName();
                if (roomId != null && !roomId.isEmpty()) {
                    Room room = roomManager.getRoom(roomId);
                    if (room != null) {
                        List<String> playersInRoom = room.getPlayersInRoom();
                        roomBroadcaster.broadcastPlayerOffline(roomId, pid, playerName, playersInRoom);
                    }
                }
            } catch (Exception e) {
                Logger.error("unbindCharacter 广播离线失败: " + e.getMessage());
            }

            playerIdToSession.remove(pid);
        }
        session.getAttributes().remove(ATTR_PLAYER_ID);
    }

    private void unbindSession(WebSocketSession session) {
        unbindCharacter(session);
        session.getAttributes().remove(ATTR_USER_ID); // 清理 userId
    }

    @SuppressWarnings("null")
    @Override
    public void afterConnectionClosed(WebSocketSession session, CloseStatus status) throws Exception {
        unbindSession(session);
        super.afterConnectionClosed(session, status);
    }

    @SuppressWarnings("null")
    @Override
    public void handleTransportError(WebSocketSession session, Throwable exception) throws Exception {
        Logger.error("WebSocket transport error for session " + session.getId() + ": " + exception.getMessage());
        unbindSession(session);
        super.handleTransportError(session, exception);
    }

    @SuppressWarnings("null")
    public void sendJson(WebSocketSession session, Object obj) {
        try {
            synchronized (session) {
                if (session.isOpen()) {
                    session.sendMessage(new TextMessage(mapper.writeValueAsString(obj)));
                }
            }
        } catch (Exception e) {
            // 记录错误但不打印堆栈，避免日志刷屏
            Logger.error("发送消息失败: " + e.getMessage());
        }
    }

    public void sendToPlayer(String playerId, Object payload) {
        WebSocketSession s = playerIdToSession.get(playerId);
        if (s != null && s.isOpen()) sendJson(s, payload);
    }
    
    // 公开方法：获取广播器实例
    public RoomBroadcaster getRoomBroadcaster() {
        return roomBroadcaster;
    }
    
    private static GameResult sysFail(String subType, String error) {
        // 创建一个通用的失败 GameResult
        return new GameResult("system", subType, false, System.currentTimeMillis(),
                Map.of("error", error), null);
    }
}