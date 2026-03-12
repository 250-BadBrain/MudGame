package com.mudgame.protocal;

import com.fasterxml.jackson.annotation.JsonInclude;

@JsonInclude(JsonInclude.Include.NON_NULL)
public final class ChatEvent {
    private static final int MAX_TEXT_LEN = 300;

    private final String kind;         // "whisper" | "shout"
    private final String fromPlayerId; // 发送者
    private final String toPlayerId;   // 私聊目标(whisper)；shout为null
    private final String text;         // 文本（已trim与限长）
    private final long ts;             // 服务器时间戳(ms)

    private ChatEvent(String kind, String from, String to, String text, long ts) {
        this.kind = kind; this.fromPlayerId = from; this.toPlayerId = to; this.text = text; this.ts = ts;
    }

    public static ChatEvent whisper(String fromPlayerId, String toPlayerId, String text) {
        return new ChatEvent("whisper", fromPlayerId, toPlayerId, sanitize(text), System.currentTimeMillis());
    }
    public static ChatEvent shout(String fromPlayerId, String text) {
        return new ChatEvent("shout", fromPlayerId, null, sanitize(text), System.currentTimeMillis());
    }

    public String getKind() { return kind; }
    public String getFromPlayerId() { return fromPlayerId; }
    public String getToPlayerId() { return toPlayerId; }
    public String getText() { return text; }
    public long getTs() { return ts; }

    private static String sanitize(String s) {
        if (s == null) return "";
        s = s.trim();
        return s.length() > MAX_TEXT_LEN ? s.substring(0, MAX_TEXT_LEN) : s;
    }
}
