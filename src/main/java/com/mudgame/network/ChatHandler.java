package com.mudgame.network;

import com.mudgame.protocal.ChatEvent;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;
import org.springframework.web.socket.WebSocketSession;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

public class ChatHandler {

    private final Map<String, WebSocketSession> playerIdToSession;
    private final Map<String, Long> lastChatAt = new ConcurrentHashMap<>();

    public ChatHandler(Map<String, WebSocketSession> playerIdToSession) {
        this.playerIdToSession = playerIdToSession;
    }

    public void handleChat(String senderId, CommandMessage cmd, MudWebSocketHandler sender) {
        String type = cmd.getType();
        String op = cmd.getSubtype();
        if (op == null && type != null && type.startsWith("chat.")) {
            op = type.substring("chat.".length());
        }

        String text = safeTrim((String) cmd.getArg("text"));
        if (text == null || text.isEmpty()) {
            sender.sendToPlayer(senderId, chatFail("empty_text"));
            return;
        }

        long now = System.currentTimeMillis();
        Long prev = lastChatAt.get(senderId);
        if (prev != null && now - prev < 300) {
            sender.sendToPlayer(senderId, chatFail("rate_limited"));
            return;
        }
        lastChatAt.put(senderId, now);

        if ("whisper".equalsIgnoreCase(op)) {
            // 目标玩家从 args["to"] 读取
            Object toObj = cmd.getArgs().get("to");
            String to = toObj == null ? null : String.valueOf(toObj).trim();
            if (to == null || to.isEmpty()) {
                sender.sendToPlayer(senderId, chatFail("missing_target"));
                return;
            }
            WebSocketSession target = playerIdToSession.get(to);
            if (target == null || !target.isOpen()) {
                sender.sendToPlayer(senderId, chatFail("target_offline"));
                return;
            }
            // 对方收到：msg
            GameResult toResult = new GameResult(
                    "chat", "msg", true, now,
                    Map.of("event", ChatEvent.whisper(senderId, to, text)), null);
            sender.sendJson(target, toResult);
            // 自己回显：sent
            GameResult echo = new GameResult(
                    "chat", "sent", true, now,
                    Map.of("event", ChatEvent.whisper(senderId, to, text)), null);
            sender.sendToPlayer(senderId, echo);
            return;
        }

        if ("shout".equalsIgnoreCase(op)) {
            GameResult evt = new GameResult(
                    "chat", "msg", true, now,
                    Map.of("event", ChatEvent.shout(senderId, text)), null);
            for (WebSocketSession s : playerIdToSession.values()) {
                if (s != null && s.isOpen()) sender.sendJson(s, evt);
            }
            return;
        }

        sender.sendToPlayer(senderId, chatFail("bad_op"));
    }

    private static GameResult chatFail(String errorCode) {
        return new GameResult("chat", "err", false, System.currentTimeMillis(),
                Map.of("error", errorCode), null);
    }

    private static String safeTrim(String s) {
        return s == null ? null : s.trim();
    }
}



