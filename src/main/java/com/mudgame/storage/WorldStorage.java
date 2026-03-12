package com.mudgame.storage;

import java.io.IOException;
import java.io.InputStream;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.mudgame.exception.LoadException;
import com.mudgame.model.World;

public class WorldStorage {
    private static final String WORLDS_DIR = "worlds/";

    private final ObjectMapper objectMapper;

    public WorldStorage(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    public World loadWorld(String worldId) throws LoadException {
        String path = WORLDS_DIR + worldId + ".json";
        InputStream tempIs = WorldStorage.class.getClassLoader().getResourceAsStream(path);
        if (tempIs == null) {
            tempIs = WorldStorage.class.getResourceAsStream("/" + path);
        }
        
        if (tempIs == null) {
            throw new LoadException("storage：世界文件不存在: " + path);
        }

        try(InputStream is = tempIs) {
            return objectMapper.readValue(is, World.class);
        } catch (IOException e) {
            throw new LoadException("storage：世界加载失败: " + path, e);
        }
    }
}
