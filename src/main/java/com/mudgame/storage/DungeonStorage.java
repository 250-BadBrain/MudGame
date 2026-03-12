package com.mudgame.storage;

import java.io.IOException;
import java.nio.charset.StandardCharsets;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.mudgame.model.Dungeon;

public class DungeonStorage {
    private static final String DUNGEONS_DIR = "dungeons/";

    private final ObjectMapper objectMapper;

    public DungeonStorage(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    public Dungeon loadDungeon(String dungeonId) throws IOException {
        String path = DUNGEONS_DIR + dungeonId + ".json";
        java.io.InputStream tempIs = DungeonStorage.class.getClassLoader().getResourceAsStream(path);
        if (tempIs == null) {
            tempIs = DungeonStorage.class.getResourceAsStream("/" + path);
        }
        
        if (tempIs == null) {
            throw new IOException("副本文件不存在: " + path);
        }

        try (java.io.InputStream is = tempIs) {
            byte[] rawBytes = is.readAllBytes();
            String json = new String(rawBytes, StandardCharsets.UTF_8).trim();
            if (json.isEmpty()) {
                return null;
            }
            return objectMapper.readValue(json, Dungeon.class);
        }
    }
}
