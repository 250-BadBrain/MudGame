package com.mudgame.config;

import org.springframework.boot.ApplicationArguments;
import org.springframework.boot.ApplicationRunner;
import org.springframework.stereotype.Component;

import com.mudgame.engine.manager.ItemManager;
import com.mudgame.engine.manager.NpcManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.engine.manager.SkillManager;
import com.mudgame.engine.manager.DungeonManager;
import com.mudgame.engine.manager.WorldManager;
import com.mudgame.exception.LoadException;
import com.mudgame.utils.Logger;

@Component
public class GameDataInitializer implements ApplicationRunner {
    private static final String[] WORLDS = {"luoyang", "wudang", "shaolin", "emei", "huashan", "kunlun", "kongtong"};
    private static final String[] DUNGEONS = {"dungeon1", "dongeon2"};

    private final WorldManager worldManager;
    private final RoomManager roomManager;
    private final ItemManager itemManager;
    private final SkillManager skillManager;
    private final NpcManager npcManager;
    private final DungeonManager dungeonManager;
    private final PlayerManager playerManager;

    public GameDataInitializer(
        WorldManager worldManager,
        RoomManager roomManager,
        ItemManager itemManager,
        SkillManager skillManager,
        NpcManager npcManager,
        DungeonManager dungeonManager,
        PlayerManager playerManager) {
        this.worldManager = worldManager;
        this.roomManager = roomManager;
        this.itemManager = itemManager;
        this.skillManager = skillManager;
        this.npcManager = npcManager;
        this.dungeonManager = dungeonManager;
        this.playerManager = playerManager;
    }

    @Override
    public void run(ApplicationArguments args) throws Exception {
        Logger.info("Spring Boot WebSocket 服务器已启动，开始加载游戏数据...");
        try {
            worldManager.loadAllWorlds(WORLDS);
            roomManager.loadAllRooms();
            itemManager.loadAllItems();
            skillManager.loadAllSkills();
            npcManager.loadAllNpcs();
            dungeonManager.loadAllDungeons(DUNGEONS);
            playerManager.performMigration();
        } catch (LoadException e) {
            Logger.error("游戏启动失败，数据加载异常！" + e.getMessage());
            throw new IllegalStateException("游戏数据加载失败", e);
        } catch (IllegalStateException e) {
            Logger.error("游戏启动失败，Manager状态异常！" + e.getMessage());
            throw e;
        } catch (Exception e) {
            Logger.error("游戏启动失败，出现未知异常！" + e.getMessage());
            throw e;
        }
        Logger.info("游戏启动完成，数据加载完毕！");
    }
}