package com.mudgame.storage;

public class StorageManager {

    private final PlayerStorage playerStorage;
    private final WorldStorage worldStorage;
    private final DungeonStorage dungeonStorage;
    private final ItemStorage itemStorage;
    private final NpcStorage npcStorage;
    private final RoomStorage roomStorage;

    public StorageManager(
        PlayerStorage playerStorage,
        WorldStorage worldStorage,
        DungeonStorage dungeonStorage,
        ItemStorage itemStorage,
        NpcStorage npcStorage,
        RoomStorage roomStorage) {
        this.playerStorage = playerStorage;
        this.worldStorage = worldStorage;
        this.dungeonStorage = dungeonStorage;
        this.itemStorage = itemStorage;
        this.npcStorage = npcStorage;
        this.roomStorage = roomStorage;
    }

    public PlayerStorage players() {
        return playerStorage;
    }

    public WorldStorage worlds() {
        return worldStorage;
    }

    public DungeonStorage dungeons() {
        return dungeonStorage;
    }

    public ItemStorage items() {
        return itemStorage;
    }

    public NpcStorage npcs() {
        return npcStorage;
    }
    public RoomStorage rooms() {
        return roomStorage;
    }
}
