package manager

import (
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/gamedata"
	"mudgame/server/internal/model"
	"mudgame/server/internal/storage"
)

type RuntimeManager struct {
	combatRandom *rand.Rand
	store        storage.DynamicStore
	mu           sync.RWMutex

	usersCache    map[string]*model.User
	playersCache  map[string]*model.Player
	dynamicRooms  map[string]diskRoomDynamic
	usersLoaded   bool
	playersLoaded bool
	roomsLoaded   bool
	usersDirty    bool
	playersDirty  bool
	roomsDirty    bool
	autosaveEvery time.Duration
	autosaveStop  chan struct{}
	autosaveDone  chan struct{}

	dungeonLoadErr error
	dungeonMgr     *engruntime.Manager
	// staticNPCs 会在 RLock 路径（房间快照/更新）中懒加载写入，
	// 单独用一把锁保护 map 本身，避免两个读锁持有者并发写 map 导致 fatal
	staticNPCsMu sync.Mutex
	staticNPCs   map[string]*engruntime.NPCState

	// 主动招式战斗会话（实现规范§3）：mu保护；临时效果只在内存，CD随玩家存档持久化
	combatBindings map[string]*combatBinding  // battleID → 绑定
	actorBattles   map[string]string          // actorKey → battleID
	combatEffects  map[string][]*CombatEffect // ownerKey → 效果列表
	npcIntents     map[string]*npcIntentState // npcKey → 公开意图
	combatRevision map[string]int64           // playerID → 视图修订号
	combatEpoch    string                     // 进程纪元：重启即换，旧视图作废
	combatNow      func() time.Time           // 可注入时钟，测试用
}

var (
	loadedStaticData = gamedata.MustLoad()
	fileWorldOrder   = append([]string{}, loadedStaticData.WorldOrder...)
	fileWorlds       = loadedStaticData.Worlds
	fileRooms        = loadedStaticData.Rooms
	fileItems        = loadedStaticData.Items
	fileSkills       = loadedStaticData.Skills
	fileNpcs         = loadedStaticData.NPCs
	fileDialogues    = loadedStaticData.Dialogues
	fileQuests       = loadedStaticData.Quests
	fileFactions     = loadedStaticData.Factions
)

func NewRuntimeManager(baseDir string) *RuntimeManager {
	if _, err := os.Stat(filepath.Join(baseDir, ".martial-migration.lock")); err == nil {
		panic("武学迁移尚未结束，拒绝启动")
	}
	r := &RuntimeManager{
		store:         storage.NewFileStore(baseDir),
		combatRandom:  rand.New(rand.NewSource(time.Now().UnixNano())),
		usersCache:    map[string]*model.User{},
		playersCache:  map[string]*model.Player{},
		dynamicRooms:  map[string]diskRoomDynamic{},
		staticNPCs:    map[string]*engruntime.NPCState{},
		combatNow:     time.Now,
		autosaveEvery: 30 * time.Second,
		autosaveStop:  make(chan struct{}),
		autosaveDone:  make(chan struct{}),
	}
	dungeonTemplates := map[string]engruntime.Template{}
	dungeonInstances := map[string]*engruntime.Instance{}
	playerDungeonInstance := map[string]string{}
	roomDungeonInstance := map[string]string{}
	dungeonNPCs := map[string]*engruntime.NPCState{}
	r.dungeonMgr = engruntime.NewManager(
		fileNpcs,
		dungeonTemplates,
		dungeonInstances,
		playerDungeonInstance,
		roomDungeonInstance,
		dungeonNPCs,
	)
	if _, err := r.loadUsers(); err != nil {
		panic(err)
	}
	if _, err := r.loadPlayers(); err != nil {
		panic(err)
	}
	if _, err := r.loadDynamicRooms(); err != nil {
		panic(err)
	}
	r.dungeonLoadErr = r.loadDungeonTemplatesNoLock()
	r.resetCombatStateAfterRestartNoLock()
	r.startAutosaveLoop()
	return r
}

func (r *RuntimeManager) startAutosaveLoop() {
	if r.autosaveEvery <= 0 {
		close(r.autosaveDone)
		return
	}
	go func() {
		ticker := time.NewTicker(r.autosaveEvery)
		defer ticker.Stop()
		defer close(r.autosaveDone)
		for {
			select {
			case <-ticker.C:
				r.flushAllDirty()
			case <-r.autosaveStop:
				return
			}
		}
	}()
}

func (r *RuntimeManager) flushAllDirty() {
	r.mu.Lock()
	defer r.mu.Unlock()
	_ = r.flushDirtyNoLock()
}

func (r *RuntimeManager) flushDirtyNoLock() error {
	if r.usersDirty {
		if err := r.writeUsersToDiskNoLock(r.usersCache); err != nil {
			return err
		}
		r.usersDirty = false
	}
	if r.playersDirty {
		if err := r.writePlayersToDiskNoLock(r.playersCache); err != nil {
			return err
		}
		r.playersDirty = false
	}
	if r.roomsDirty {
		if err := r.writeDynamicRoomsToDiskNoLock(r.dynamicRooms); err != nil {
			return err
		}
		r.roomsDirty = false
	}
	return nil
}

func (r *RuntimeManager) Close() error {
	close(r.autosaveStop)
	<-r.autosaveDone
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.flushDirtyNoLock()
}
