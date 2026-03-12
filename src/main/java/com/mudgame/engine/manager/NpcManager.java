package com.mudgame.engine.manager;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import com.mudgame.exception.LoadException;
import com.mudgame.model.Npc;
import com.mudgame.storage.NpcStorage;
import com.mudgame.utils.Logger;

public class NpcManager {
    private final Map<String, Npc> npcs = new ConcurrentHashMap<>();
    private volatile boolean loaded = false;
    private final NpcStorage npcStorage;

    public NpcManager(NpcStorage npcStorage){
        this.npcStorage = npcStorage;
    }

    public void loadAllNpcs() throws LoadException{
        if(loaded){
            throw new IllegalStateException("NPC 已加载，不能重复加载");
        }
        Map<String, Npc> loadedNpcs = npcStorage.loadAllNpc();
        this.npcs.putAll(loadedNpcs);
        loaded = true;
        Logger.info("manager：加载所有 NPC 成功");
    }

    public Npc getNpc(String npcId){
        if(!loaded){
            throw new IllegalStateException("manager：NPC 未加载，无法获取 NPC");
        }
        Npc npc=npcs.get(npcId);
        if(npc==null){
            throw new IllegalArgumentException("manager: NPC 参数错误");
        }
        return npc;
    }
}
