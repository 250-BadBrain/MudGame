package com.mudgame.engine.manager;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import com.mudgame.exception.LoadException;
import com.mudgame.model.World;
import com.mudgame.storage.RoomRepository;
import com.mudgame.storage.WorldStorage;
import com.mudgame.utils.Logger;

public class WorldManager {
    private final Map<String,World>worlds=new ConcurrentHashMap<>();
    private volatile boolean loaded = false;
    private final WorldStorage worldStorage;

    public WorldManager(WorldStorage worldStorage){
        this.worldStorage = worldStorage;
    }
    
    public void loadAllWorlds(String[] worldIds) throws LoadException{
        if (loaded) {
            throw new IllegalStateException("manager：世界已加载，不能重复加载");
        }
        for(String worldId:worldIds){
            World world=worldStorage.loadWorld(worldId);
            worlds.put(worldId,world);
            Logger.info("manager：加载世界成功："+worldId);
            RoomRepository.loadStaticRooms(worldId);
            Logger.info("manager：加载静态房间成功："+worldId);
        }
        loaded=true;
        Logger.info("manager：加载所有世界成功");
    }
    public World getWorld(String worldId){
        if(!loaded){
            throw new IllegalStateException("manager：世界未加载，无法获取世界");
        }
        World world=worlds.get(worldId);
        if(world==null){
            throw new IllegalArgumentException("manager: 世界参数错误");
        }
        return world;
    }
    public Map<String, World> getAllWorlds() {
        if (!loaded) {
            throw new IllegalStateException("manager：世界未加载，无法获取世界列表");
        }
        return worlds;
    }
}
