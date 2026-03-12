package com.mudgame.config;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import com.mudgame.engine.handler.UnknownCommandHandler;
import com.mudgame.engine.handler.chat.PublicChatHandler;
import com.mudgame.engine.handler.game.BuyItemHandler;
import com.mudgame.engine.handler.game.DiscardItemHandler;
import com.mudgame.engine.handler.game.EnterWorldHandler;
import com.mudgame.engine.handler.game.EquipHandler;
import com.mudgame.engine.handler.game.GetAttributesHandler;
import com.mudgame.engine.handler.game.GetBackpackHandler;
import com.mudgame.engine.handler.game.GetEquipmentHandler;
import com.mudgame.engine.handler.game.GetShopHandler;
import com.mudgame.engine.handler.game.GetSkillsHandler;
import com.mudgame.engine.handler.game.GetWorldsHandler;
import com.mudgame.engine.handler.game.GoHandler;
import com.mudgame.engine.handler.game.HealHandler;
import com.mudgame.engine.handler.game.KillHandler;
import com.mudgame.engine.handler.game.LearnHandler;
import com.mudgame.engine.handler.game.MeditateHandler;
import com.mudgame.engine.handler.game.MineHandler;
import com.mudgame.engine.handler.game.ReliveHandler;
import com.mudgame.engine.handler.game.SellItemHandler;
import com.mudgame.engine.handler.game.SparHandler;
import com.mudgame.engine.handler.game.StopHandler;
import com.mudgame.engine.handler.game.UnequipHandler;
import com.mudgame.engine.manager.CombatManager;
import com.mudgame.engine.manager.DungeonManager;
import com.mudgame.engine.manager.ItemManager;
import com.mudgame.engine.manager.NpcManager;
import com.mudgame.engine.manager.PlayerManager;
import com.mudgame.engine.manager.RoomManager;
import com.mudgame.engine.manager.SkillManager;
import com.mudgame.engine.manager.StatusManager;
import com.mudgame.engine.manager.WorldManager;
import com.mudgame.engine.handler.system.CreateCharacterHandler;
import com.mudgame.engine.handler.system.DeleteAccountHandler;
import com.mudgame.engine.handler.system.DeleteCharacterHandler;
import com.mudgame.engine.handler.system.DeselectCharacterHandler;
import com.mudgame.engine.handler.system.GetCharactersHandler;
import com.mudgame.engine.handler.system.GetRoomHandler;
import com.mudgame.engine.handler.system.SelectCharacterHandler;
import com.mudgame.network.RoomBroadcaster;
import com.mudgame.network.Auth.AuthHandler;
import com.mudgame.network.OnlineManager;
import com.mudgame.network.Auth.PasswordHasher;
import com.mudgame.network.Auth.Pbkdf2PasswordHasher;
import com.mudgame.network.Auth.ReconnectTokenService;
import com.mudgame.storage.ItemStorage;
import com.mudgame.storage.NpcStorage;
import com.mudgame.storage.PlayerStorage;
import com.mudgame.storage.RoomRepository;
import com.mudgame.storage.RoomStorage;
import com.mudgame.storage.SkillStorage;
import com.mudgame.storage.StorageManager;
import com.mudgame.storage.UserStorage;
import com.mudgame.storage.WorldStorage;
import com.mudgame.storage.DungeonStorage;

@Configuration
public class BackendBeansConfig {
    public static final class ManagerBootstrap {
    }

    @Bean
    public ObjectMapper objectMapper() {
        return new ObjectMapper();
    }

    @Bean
    public UserStorage userStorage(ObjectMapper objectMapper) {
        return new UserStorage(objectMapper);
    }

    @Bean
    public PlayerStorage playerStorage(ObjectMapper objectMapper) {
        return new PlayerStorage(objectMapper);
    }

    @Bean
    public RoomRepository roomRepository(ObjectMapper objectMapper) {
        return new RoomRepository(objectMapper);
    }

    @Bean
    public RoomStorage roomStorage(RoomRepository roomRepository) {
        return new RoomStorage(roomRepository);
    }

    @Bean
    public WorldStorage worldStorage(ObjectMapper objectMapper) {
        return new WorldStorage(objectMapper);
    }

    @Bean
    public ItemStorage itemStorage(ObjectMapper objectMapper) {
        return new ItemStorage(objectMapper);
    }

    @Bean
    public NpcStorage npcStorage(ObjectMapper objectMapper) {
        return new NpcStorage(objectMapper);
    }

    @Bean
    public SkillStorage skillStorage(ObjectMapper objectMapper) {
        return new SkillStorage(objectMapper);
    }

    @Bean
    public DungeonStorage dungeonStorage(ObjectMapper objectMapper) {
        return new DungeonStorage(objectMapper);
    }

    @Bean
    public StorageManager storageManager(
        PlayerStorage playerStorage,
        WorldStorage worldStorage,
        DungeonStorage dungeonStorage,
        ItemStorage itemStorage,
        NpcStorage npcStorage,
        RoomStorage roomStorage) {
        return new StorageManager(playerStorage, worldStorage, dungeonStorage, itemStorage, npcStorage, roomStorage);
    }

    @Bean
    public PasswordHasher passwordHasher() {
        return new Pbkdf2PasswordHasher();
    }

    @Bean
    public ReconnectTokenService reconnectTokenService() {
        return new ReconnectTokenService();
    }

    @Bean
    public PlayerManager playerManager(PlayerStorage playerStorage, UserStorage userStorage) {
        return new PlayerManager(playerStorage, userStorage);
    }

    @Bean
    public RoomManager roomManager(
        RoomStorage roomStorage,
        WorldManager worldManager,
        PlayerManager playerManager,
        OnlineManager onlineManager,
        NpcManager npcManager) {
        return new RoomManager(roomStorage, worldManager, playerManager, onlineManager, npcManager);
    }

    @Bean
    public WorldManager worldManager(WorldStorage worldStorage) {
        return new WorldManager(worldStorage);
    }

    @Bean
    public ItemManager itemManager(ItemStorage itemStorage) {
        return new ItemManager(itemStorage);
    }

    @Bean
    public DungeonManager dungeonManager(DungeonStorage dungeonStorage) {
        return new DungeonManager(dungeonStorage);
    }

    @Bean
    public NpcManager npcManager(NpcStorage npcStorage) {
        return new NpcManager(npcStorage);
    }

    @Bean
    public SkillManager skillManager(SkillStorage skillStorage) {
        return new SkillManager(skillStorage);
    }

    @Bean
    public CombatManager combatManager(RoomBroadcaster roomBroadcaster, PlayerManager playerManager) {
        return new CombatManager(roomBroadcaster, playerManager);
    }

    @Bean
    public StatusManager statusManager(RoomBroadcaster roomBroadcaster, PlayerManager playerManager, ItemManager itemManager, SkillManager skillManager) {
        return new StatusManager(roomBroadcaster, playerManager, itemManager, skillManager);
    }

    @Bean
    public OnlineManager onlineManager() {
        return new OnlineManager();
    }

    @Bean
    public ManagerBootstrap managerBootstrap(
        PlayerManager playerManager,
        RoomManager roomManager,
        WorldManager worldManager,
        ItemManager itemManager,
        DungeonManager dungeonManager,
        NpcManager npcManager,
        SkillManager skillManager,
        CombatManager combatManager,
        StatusManager statusManager,
        OnlineManager onlineManager) {
        return new ManagerBootstrap();
    }

    @Bean
    public AuthHandler authHandler(UserStorage userStorage, PasswordHasher passwordHasher, ReconnectTokenService reconnectTokenService) {
        return new AuthHandler(userStorage, passwordHasher, reconnectTokenService);
    }

    @Bean
    public UnknownCommandHandler unknownCommandHandler(ManagerBootstrap managerBootstrap) {
        return new UnknownCommandHandler();
    }

    @Bean
    public GetRoomHandler getRoomHandler(PlayerManager playerManager, RoomManager roomManager, ManagerBootstrap managerBootstrap) {
        return new GetRoomHandler(playerManager, roomManager);
    }

    @Bean
    public CreateCharacterHandler createCharacterHandler(UserStorage userStorage, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new CreateCharacterHandler(userStorage, playerManager);
    }

    @Bean
    public SelectCharacterHandler selectCharacterHandler(UserStorage userStorage, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new SelectCharacterHandler(userStorage, playerManager);
    }

    @Bean
    public GetCharactersHandler getCharactersHandler(UserStorage userStorage, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new GetCharactersHandler(userStorage, playerManager);
    }

    @Bean
    public DeleteCharacterHandler deleteCharacterHandler(
        UserStorage userStorage,
        PlayerStorage playerStorage,
        PlayerManager playerManager,
        RoomManager roomManager,
        OnlineManager onlineManager,
        ManagerBootstrap managerBootstrap) {
        return new DeleteCharacterHandler(userStorage, playerStorage, playerManager, roomManager, onlineManager);
    }

    @Bean
    public DeleteAccountHandler deleteAccountHandler(
        UserStorage userStorage,
        PlayerStorage playerStorage,
        PasswordHasher passwordHasher,
        PlayerManager playerManager,
        RoomManager roomManager,
        OnlineManager onlineManager,
        ManagerBootstrap managerBootstrap) {
        return new DeleteAccountHandler(userStorage, playerStorage, passwordHasher, playerManager, roomManager, onlineManager);
    }

    @Bean
    public DeselectCharacterHandler deselectCharacterHandler(ManagerBootstrap managerBootstrap) {
        return new DeselectCharacterHandler();
    }

    @Bean
    public GoHandler goHandler(PlayerManager playerManager, RoomManager roomManager, ManagerBootstrap managerBootstrap) {
        return new GoHandler(playerManager, roomManager);
    }

    @Bean
    public GetBackpackHandler getBackpackHandler(ItemManager itemManager, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new GetBackpackHandler(itemManager, playerManager);
    }

    @Bean
    public GetSkillsHandler getSkillsHandler(PlayerManager playerManager, SkillManager skillManager, ManagerBootstrap managerBootstrap) {
        return new GetSkillsHandler(playerManager, skillManager);
    }

    @Bean
    public GetAttributesHandler getAttributesHandler(PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new GetAttributesHandler(playerManager);
    }

    @Bean
    public GetWorldsHandler getWorldsHandler(WorldManager worldManager, ManagerBootstrap managerBootstrap) {
        return new GetWorldsHandler(worldManager);
    }

    @Bean
    public EnterWorldHandler enterWorldHandler(PlayerManager playerManager, RoomManager roomManager, WorldManager worldManager, ManagerBootstrap managerBootstrap) {
        return new EnterWorldHandler(playerManager, roomManager, worldManager);
    }

    @Bean
    public GetShopHandler getShopHandler(ItemManager itemManager, NpcManager npcManager, PlayerManager playerManager, RoomManager roomManager, ManagerBootstrap managerBootstrap) {
        return new GetShopHandler(itemManager, npcManager, playerManager, roomManager);
    }

    @Bean
    public BuyItemHandler buyItemHandler(ItemManager itemManager, NpcManager npcManager, PlayerManager playerManager, RoomManager roomManager, ManagerBootstrap managerBootstrap) {
        return new BuyItemHandler(itemManager, npcManager, playerManager, roomManager);
    }

    @Bean
    public KillHandler killHandler(CombatManager combatManager, NpcManager npcManager, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new KillHandler(combatManager, npcManager, playerManager);
    }

    @Bean
    public SparHandler sparHandler(CombatManager combatManager, NpcManager npcManager, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new SparHandler(combatManager, npcManager, playerManager);
    }

    @Bean
    public HealHandler healHandler(PlayerManager playerManager, StatusManager statusManager, ManagerBootstrap managerBootstrap) {
        return new HealHandler(playerManager, statusManager);
    }

    @Bean
    public MeditateHandler meditateHandler(PlayerManager playerManager, StatusManager statusManager, ManagerBootstrap managerBootstrap) {
        return new MeditateHandler(playerManager, statusManager);
    }

    @Bean
    public ReliveHandler reliveHandler(PlayerManager playerManager, RoomManager roomManager, ManagerBootstrap managerBootstrap) {
        return new ReliveHandler(playerManager, roomManager);
    }

    @Bean
    public StopHandler stopHandler(CombatManager combatManager, PlayerManager playerManager, StatusManager statusManager, ManagerBootstrap managerBootstrap) {
        return new StopHandler(combatManager, playerManager, statusManager);
    }

    @Bean
    public DiscardItemHandler discardItemHandler(ItemManager itemManager, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new DiscardItemHandler(itemManager, playerManager);
    }

    @Bean
    public SellItemHandler sellItemHandler(ItemManager itemManager, PlayerManager playerManager, RoomManager roomManager, ManagerBootstrap managerBootstrap) {
        return new SellItemHandler(itemManager, playerManager, roomManager);
    }

    @Bean
    public EquipHandler equipHandler(ItemManager itemManager, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new EquipHandler(itemManager, playerManager);
    }

    @Bean
    public UnequipHandler unequipHandler(ItemManager itemManager, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new UnequipHandler(itemManager, playerManager);
    }

    @Bean
    public MineHandler mineHandler(PlayerManager playerManager, StatusManager statusManager, ManagerBootstrap managerBootstrap) {
        return new MineHandler(playerManager, statusManager);
    }

    @Bean
    public GetEquipmentHandler getEquipmentHandler(ItemManager itemManager, PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new GetEquipmentHandler(itemManager, playerManager);
    }

    @Bean
    public LearnHandler learnHandler(PlayerManager playerManager, RoomManager roomManager, StatusManager statusManager, NpcManager npcManager, ManagerBootstrap managerBootstrap) {
        return new LearnHandler(playerManager, roomManager, statusManager, npcManager);
    }

    @Bean
    public PublicChatHandler publicChatHandler(PlayerManager playerManager, ManagerBootstrap managerBootstrap) {
        return new PublicChatHandler(playerManager);
    }
}