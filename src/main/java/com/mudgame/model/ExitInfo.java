package com.mudgame.model;

public class ExitInfo {
    private String targetRoomId;
    private String targetRoomName;

    public ExitInfo() {

    }

    public ExitInfo(String targetRoomId, String targetRoomName) {
        this.targetRoomId = targetRoomId;
        this.targetRoomName = targetRoomName;
    }

    public String getTargetRoomId() {
        return targetRoomId;
    }
    
    public String getTargetRoomName() {
        return targetRoomName;
    }

    // Setter 方法是必需的，尤其是当 Jackson 用于反序列化时
    public void setTargetRoomId(String targetRoomId) {
        this.targetRoomId = targetRoomId;
    }

    public void setTargetRoomName(String targetRoomName) {
        this.targetRoomName = targetRoomName;
    }
}