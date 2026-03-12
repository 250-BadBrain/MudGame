package com.mudgame.engine.manager;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import com.mudgame.exception.LoadException;
import com.mudgame.model.Item;
import com.mudgame.storage.ItemStorage;
import com.mudgame.utils.Logger;

public class ItemManager {
    private final Map<String, Item> items = new ConcurrentHashMap<>();
    private volatile boolean loaded = false;
    private final ItemStorage itemStorage;

    public ItemManager(ItemStorage itemStorage){
        this.itemStorage = itemStorage;
    }

    public void loadAllItems() throws LoadException {
        if(loaded){
            throw new IllegalStateException("manager：物品已加载，不能重复加载");
        }
        Map<String, Item> loadedItems = itemStorage.loadAllItems();
        this.items.putAll(loadedItems);
        this.loaded = true;
        Logger.info("manager：加载所有物品成功");
    }

    public Item getItem(String itemId){
        if(!loaded){
            throw new IllegalStateException("manager：物品未加载，无法获取物品");
        }
        Item item=items.get(itemId);
        if(item==null){
            throw new IllegalArgumentException("manager: 物品参数错误");
        }
        return item;
    }
}
