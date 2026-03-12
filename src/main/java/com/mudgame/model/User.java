package com.mudgame.model; 

import java.util.List;
import java.util.ArrayList;
import com.fasterxml.jackson.annotation.JsonIgnore;

public class User extends Models {
    // 每个账号的最大角色数上限
    public static final int MAX_CHARACTERS_PER_ACCOUNT = 3;
    
    private String username; 
    private String passwordHash; 
    private List<String> characterIds; 
    private long lastLogin;
    private String reconnectTokenHash;
    private long reconnectTokenExpiresAt;

    public User() {
        this.characterIds = new ArrayList<>();
        this.lastLogin = 0L;
    }

    public User(String username, String passwordHash) {
        this.username = username;
        this.passwordHash = passwordHash;
        this.characterIds = new ArrayList<>();
        this.lastLogin = 0L;
    }

    public String getUsername() {
        return username;
    }

    public void setUsername(String username) {
        this.username = username;
    }

    public String getPasswordHash() {
        return passwordHash;
    }

    public void setPasswordHash(String passwordHash) {
        this.passwordHash = passwordHash;
    }

    public List<String> getCharacterIds() { 
        return characterIds;
    }

    public void setCharacterIds(List<String> characterIds) { 
        this.characterIds = characterIds;
    }
    
    // 新增 lastLogin 的 Getter/Setter
    public long getLastLogin() {
        return lastLogin;
    }

    public void setLastLogin(long lastLogin) {
        this.lastLogin = lastLogin;
    }

    public String getReconnectTokenHash() {
        return reconnectTokenHash;
    }

    public void setReconnectTokenHash(String reconnectTokenHash) {
        this.reconnectTokenHash = reconnectTokenHash;
    }

    public long getReconnectTokenExpiresAt() {
        return reconnectTokenExpiresAt;
    }

    public void setReconnectTokenExpiresAt(long reconnectTokenExpiresAt) {
        this.reconnectTokenExpiresAt = reconnectTokenExpiresAt;
    }

    public void clearReconnectToken() {
        this.reconnectTokenHash = null;
        this.reconnectTokenExpiresAt = 0L;
    }

    /**
     * 添加一个新创建的角色ID到账户下
     * @param characterId 新的角色ID
     */
    public void addCharacterId(String characterId) { 
        if (this.characterIds == null) {
            this.characterIds = new ArrayList<>();
        }
        if (!this.characterIds.contains(characterId)) {
            this.characterIds.add(characterId);
        }
    }

    /**
     * 从账户中移除一个角色ID
     * @param characterId 要移除的角色ID
     */
    public void removeCharacterId(String characterId) {
        if (this.characterIds != null) {
            this.characterIds.remove(characterId);
        }
    }

    /**
     * 检查账户是否已达到角色数量上限
     * @return true 如果已达到上限，false 否则
     */
    @JsonIgnore
    public boolean isCharacterLimitReached() {
        return this.characterIds != null && this.characterIds.size() >= MAX_CHARACTERS_PER_ACCOUNT;
    }
}