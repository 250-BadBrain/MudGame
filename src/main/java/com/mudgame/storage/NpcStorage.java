package com.mudgame.storage;

import java.io.IOException;
import java.io.InputStream;
import java.util.Collections;
import java.util.Map;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.mudgame.exception.LoadException;
import com.mudgame.model.Npc;
import com.mudgame.utils.Logger;

public class NpcStorage {
    private static final String NPC_DIR = "npcs.json";

    private final ObjectMapper objectMapper;

    public NpcStorage(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    public Map<String, Npc> loadAllNpc() throws LoadException {
        InputStream tempIs = NpcStorage.class.getClassLoader().getResourceAsStream(NPC_DIR);
        if (tempIs == null) {
            tempIs = NpcStorage.class.getResourceAsStream("/" + NPC_DIR);
        }

        if (tempIs == null) {
            Logger.error("storage：未找到NPC配置文件: " + NPC_DIR);
            throw new LoadException("未找到NPC配置文件: " + NPC_DIR);
        }

        try (InputStream is = tempIs) {
            TypeReference<Map<String, Npc>> typeRef = new TypeReference<Map<String, Npc>>() {};
            Map<String, Npc> npcs = objectMapper.readValue(is, typeRef);
            
            // Set ID for each NPC based on the map key
            for (Map.Entry<String, Npc> entry : npcs.entrySet()) {
                if (entry.getValue().getId() == null) {
                    entry.getValue().setId(entry.getKey());
                }
            }

            Logger.info("storage：成功加载NPC配置，数量: " + npcs.size());
            
            return Collections.unmodifiableMap(npcs);
        } catch (IOException e) {
            throw new LoadException("加载NPC失败: " + e.getMessage(), e);
        }
    }
}
