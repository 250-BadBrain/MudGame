package com.mudgame.storage;

import java.io.IOException;
import java.io.InputStream;
import java.util.Collections;
import java.util.Map;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.mudgame.exception.LoadException;
import com.mudgame.model.Item;
import com.mudgame.utils.Logger;

public class ItemStorage {
    private static final String ITEM_DIR="items.json";

    private final ObjectMapper objectMapper;

    public ItemStorage(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    public Map<String, Item> loadAllItems() throws LoadException {
        InputStream tempIs = ItemStorage.class.getClassLoader().getResourceAsStream(ITEM_DIR);
        if (tempIs == null) {
            tempIs = ItemStorage.class.getResourceAsStream("/" + ITEM_DIR);
        }

        if(tempIs == null) {
            Logger.error("storage：未找到物品配置文件: " + ITEM_DIR);
            throw new LoadException("未找到物品配置文件: " + ITEM_DIR);
        }

        try(InputStream is = tempIs) {
            TypeReference<Map<String, Item>> typeRef = new TypeReference<Map<String, Item>>() {};   
            Map<String, Item> items = objectMapper.readValue(is, typeRef);

            // Set the ID for each item based on the map key
            for (Map.Entry<String, Item> entry : items.entrySet()) {
                Item item = entry.getValue();
                if (item.getId() == null) {
                    item.setId(entry.getKey());
                }
            }

            Logger.info("storage：成功加载物品配置，数量: " + items.size());

            return Collections.unmodifiableMap(items);
        } catch (IOException e) {
            throw new LoadException("加载物品失败: " + e.getMessage(), e);
        }
    }
}
