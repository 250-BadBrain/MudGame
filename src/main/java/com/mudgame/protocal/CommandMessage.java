package com.mudgame.protocal;

import java.util.Collections;
import java.util.HashMap;
import java.util.Map;

public class CommandMessage {
    private String type;        // system / command / chat
    private String subtype;     // login / go / say ...
    private String playerId;    // 哪个玩家发的
    private Long ts;            // 时间戳
    private Map<String, Object> args;  // 命令或附加参数

    // ===== 常量定义 =====
    public static final class Types {
        public static final String SYSTEM  = "system";
        public static final String COMMAND = "command";
        public static final String CHAT    = "chat";
        private Types() {}
    }

    public static final class Ops {
        // system
        public static final String LOGIN    = "login";
        public static final String REGISTER = "register";
        public static final String GET_ROOM = "get_room";

        // command
        public static final String GO       = "go";      
        public static final String TAKE     = "take";
        public static final String ATTACK   = "attack";
        public static final String USE      = "use";
        public static final String ATTRIBUTE = "attribute";
        public static final String BACKPACK  = "backpack";
        public static final String SKILLS    = "skills";
        public static final String EQUIP     = "equip";
        public static final String MINE      = "mine";
        public static final String GET_EQUIPMENT = "get_equipment";

        // chat
        public static final String SAY      = "say";      
        public static final String WHISPER  = "whisper";  

        private Ops() {}
    }

    // ===== 构造函数 =====
    public CommandMessage() {
        this.args = new HashMap<>();
    }

    public CommandMessage(String type, String subtype, String playerId, 
                            Map<String, Object> args, Long ts) {
        this.type = type;
        this.subtype = subtype;
        this.playerId = playerId;
        this.args = (args == null) ? new HashMap<>() : new HashMap<>(args);
        this.ts = ts;
    }

    // ===== Getter =====
    public String getType() { return type; }
    public String getSubtype() { return subtype; }
    public String getPlayerId() { return playerId; }
    public Long getTs() { return ts; }
    public Map<String, Object> getArgs() {
        return (args == null) ? Collections.emptyMap() : Collections.unmodifiableMap(args);
    }

    // ===== Setter =====
    public void setType(String type) { this.type = type; }
    public void setSubtype(String subtype) { this.subtype = subtype; }
    public void setPlayerId(String playerId) { this.playerId = playerId; }
    public void setTs(Long ts) { this.ts = ts; }
    public void setArgs(Map<String, Object> args) {
        this.args = (args == null) ? new HashMap<>() : new HashMap<>(args);
    }

    // ===== 工具方法（推荐） =====
    public void addArg(String key, Object value) {
        this.args.put(key, value);
    }

    public Object getArg(String key) {
        return this.args.get(key);
    }
}
