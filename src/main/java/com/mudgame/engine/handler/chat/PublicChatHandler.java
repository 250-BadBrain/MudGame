package com.mudgame.engine.handler.chat;

import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.exception.UnknownCommandException;
import com.mudgame.model.Player;
import com.mudgame.network.RoomBroadcaster;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import com.mudgame.utils.Logger;

import java.util.Map;

public class PublicChatHandler extends ChatCommandHandler {
    private static final String subtype = "public";
    private final PlayerManager playerManager;

    public PublicChatHandler(PlayerManager playerManager) {
        this.playerManager = playerManager;
    }

    @Override
    public GameResult handle(CommandMessage message) throws UnknownCommandException {
        Logger.info(log_name + "接收到公开聊天请求");
        
        String playerId = message.getPlayerId();
        Player player = playerManager.getPlayer(playerId);
        
        if (player == null) {
             throw new UnknownCommandException(type, subtype, "Player not found");
        }

        Map<String, Object> args = message.getArgs();
        if (args == null || !args.containsKey("message")) {
             throw new UnknownCommandException(type, subtype, "Missing message content");
        }

        String chatContent = (String) args.get("message");
        if (chatContent == null || chatContent.trim().isEmpty()) {
             // 空消息不处理，或者返回错误
             return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "Empty message"), null);
        }

        RoomBroadcaster broadcaster = (RoomBroadcaster) args.get("__broadcaster");
        if (broadcaster != null) {
            broadcaster.broadcastPublicChat(playerId, player.getCharacterName(), chatContent);
        } else {
            Logger.error(log_name + "Broadcaster not found in command args");
            return new GameResult(type, subtype, false, System.currentTimeMillis(), Map.of("error", "Internal server error"), null);
        }

        // 返回给发送者的确认（可选，因为广播也会发给自己）
        // 这里返回 null，避免 MudWebSocketHandler 再次发送空消息给发送者，导致前端显示 "???"
        return null;
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
