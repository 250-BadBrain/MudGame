package com.mudgame.engine.manager;

import java.util.Collections;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import com.mudgame.exception.LoadException;
import com.mudgame.model.Dungeon;
import com.mudgame.storage.DungeonStorage;
import com.mudgame.utils.Logger;

public class DungeonManager {
    private final Map<String,Dungeon>dungeons=new ConcurrentHashMap<>();
    private volatile boolean loaded = false;
    private final DungeonStorage dungeonStorage;

    public DungeonManager(DungeonStorage dungeonStorage) {
        this.dungeonStorage = dungeonStorage;
    }

    public void loadAllDungeons(String[] dungeonIds) throws LoadException {
        if (loaded) {
            throw new IllegalStateException("manager：副本已加载，不能重复加载");
        }

        for (String dungeonId : dungeonIds) {
            try {
                Dungeon dungeon = dungeonStorage.loadDungeon(dungeonId);
                if (dungeon == null) {
                    Logger.warn("manager：副本资源为空，已跳过：" + dungeonId);
                    continue;
                }
                dungeons.put(dungeonId, dungeon);
                Logger.info("manager：加载副本成功：" + dungeonId);
            } catch (Exception e) {
                throw new LoadException("加载副本失败：" + dungeonId, e);
            }
        }

        loaded = true;
        Logger.info("manager：加载所有副本成功");
    }

    public Dungeon getDungeon(String dungeonId) {
        if(!loaded){
            throw new IllegalStateException("manager：副本未加载，无法获取副本");
        }
        Dungeon dungeon=dungeons.get(dungeonId);
        if(dungeon==null){
            throw new IllegalArgumentException("manager: 副本参数错误");
        }
        return dungeon;
    }

    public Map<String, Dungeon> getAllDungeons() {
        if (!loaded) {
            throw new IllegalStateException("manager：副本未加载，无法获取副本列表");
        }
        return Collections.unmodifiableMap(dungeons);
    }
}
