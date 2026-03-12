package com.mudgame.engine.handler.game;

import com.mudgame.engine.manager.WorldManager;
import com.mudgame.model.World;
import com.mudgame.protocal.CommandMessage;
import com.mudgame.protocal.GameResult;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GetWorldsHandler extends GameCommandHandler {
    private final WorldManager worldManager;

    public GetWorldsHandler(WorldManager worldManager) {
        this.worldManager = worldManager;
    }

    @Override
    public GameResult handle(CommandMessage message) {
        Map<String, World> worlds = worldManager.getAllWorlds();
        List<Map<String, String>> worldList = new ArrayList<>();
        
        for (World world : worlds.values()) {
            Map<String, String> w = new HashMap<>();
            w.put("id", world.getId());
            w.put("name", world.getName());
            w.put("description", world.getDescription());
            worldList.add(w);
        }
        
        Map<String, Object> results = new HashMap<>();
        results.put("worlds", worldList);
        
        return new GameResult(getType(), getSubType(), true, System.currentTimeMillis(), results, null);
    }

    @Override
    public String getType() {
        return "command";
    }

    @Override
    public String getSubType() {
        return "get_worlds";
    }
}
