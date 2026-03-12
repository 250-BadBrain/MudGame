package com.mudgame.network;

import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

public class OnlineManager {
    private final Set<String> onlinePlayers = ConcurrentHashMap.newKeySet();

    public OnlineManager() {}

    public void setOnline(String playerId) {
        if (playerId != null) onlinePlayers.add(playerId);
    }

    public void setOffline(String playerId) {
        if (playerId != null) onlinePlayers.remove(playerId);
    }

    public boolean isOnline(String playerId) {
        if (playerId == null) return false;
        return onlinePlayers.contains(playerId);
    }

    public Set<String> getOnlinePlayers() {
        return onlinePlayers;
    }
}
