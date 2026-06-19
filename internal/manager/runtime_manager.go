package manager

import (
	"sync"
	"time"

	engruntime "mudgame/server/internal/engine/runtime"
	"mudgame/server/internal/gamedata"
	"mudgame/server/internal/model"
	"mudgame/server/internal/storage"
)

type RuntimeManager struct {
	store storage.DynamicStore
	mu    sync.RWMutex

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
	r := &RuntimeManager{
		store:         storage.NewFileStore(baseDir),
		usersCache:    map[string]*model.User{},
		playersCache:  map[string]*model.Player{},
		dynamicRooms:  map[string]diskRoomDynamic{},
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
