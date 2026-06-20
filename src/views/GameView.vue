<template>
  <div class="game-container">
    
    <div class="top-info-area">
      <button @click="logout" class="logout-btn">退出游戏</button>
      <div class="room-header">
        <h3>{{ room?.worldName ? room.worldName + '-' : '' }}{{ room?.name || '加载中...' }}</h3>
        <p class="room-description">{{ room?.description || '' }}</p>
      </div>
    </div>

    <div class="middle-layout">
      
      <div class="left-content-area">
        <GameShop v-if="showShopPanel"
          :title="currentShopNpc.name + '正在售卖'"
          :items="shopItems"
          :selectedItem="selectedShopItem"
          :buyAmount="buyAmount"
          :isMaxReached="isMaxAmountReached"
          :playerMoney="playerMoney"
          @close="closeShop"
          @selectItem="selectShopItem"
          @changeAmount="changeAmount"
          @buy="buyItem" />

        <GameShop v-else-if="showLearnPanel"
          title="请教技能"
          :items="learnSkills"
          :selectedItem="selectedLearnSkill"
          :showMoney="false"
          teachingMode
          buyButtonText="开始学习"
          @close="closeLearn"
          @selectItem="selectLearnSkill"
          @buy="startLearning" />

        <GameDialoguePanel
          v-else-if="showDialoguePanel"
          :dialogue="activeDialogue"
          @close="closeDialogue"
          @choose="chooseDialogue" />

        <div v-else class="room-entities-area">
          <GameRoomPanel
            :entities="combinedEntities"
            :selectedEntity="selectedEntityObj"
            @selectEntity="handleEntitySelect"
            @viewEntity="viewEntity"
            @talkEntity="talkEntity"
            @sparEntity="sparEntity"
            @killEntity="killEntity"
            @learnEntity="openLearn"
            @tradeEntity="openShop" />
        </div>
        
        <div class="room-exits-grid">
          <h3>出口</h3>
          
          <div class="exits-map-grid">
            
            <div class="grid-cell">
                <template v-if="room?.exits?.northwest">
                    <button @click="go('northwest')">{{ room.exits.northwest.targetRoomName }}</button>
                </template>
            </div>
            <div class="grid-cell">
                <template v-if="room?.exits?.north">
                    <button @click="go('north')">{{ room.exits.north.targetRoomName }}</button>
                </template>
            </div>
            <div class="grid-cell">
                <template v-if="room?.exits?.northeast">
                    <button @click="go('northeast')">{{ room.exits.northeast.targetRoomName }}</button>
                </template>
            </div>

            <div class="grid-cell">
                <template v-if="room?.exits?.west">
                    <button @click="go('west')">{{ room.exits.west.targetRoomName }}</button>
                </template>
            </div>
            <div class="grid-cell current-room-center">
                <span class="room-abbr">{{ room?.name || '?' }}</span>
            </div>
            <div class="grid-cell">
                <template v-if="room?.exits?.east">
                    <button @click="go('east')">{{ room.exits.east.targetRoomName }}</button>
                </template>
            </div>

            <div class="grid-cell">
                <template v-if="room?.exits?.southwest">
                    <button @click="go('southwest')">{{ room.exits.southwest.targetRoomName }}</button>
                </template>
            </div>
            <div class="grid-cell">
                <template v-if="room?.exits?.south">
                    <button @click="go('south')">{{ room.exits.south.targetRoomName }}</button>
                </template>
            </div>
            <div class="grid-cell">
                <template v-if="room?.exits?.southeast">
                    <button @click="go('southeast')">{{ room.exits.southeast.targetRoomName }}</button>
                </template>
            </div>
          </div>
          
        </div>
        
      </div>
      
      <div class="right-log-area">
        <div class="log-content-wrapper">
            <!-- 聊天历史（移到游戏日志上方） -->
            <div class="chat-history-panel" v-if="true">
                <div class="chat-history-header">聊天</div>
                <div class="chat-history" ref="chatHistoryEl">
                    <div v-for="(c, idx) in chatMessages" :key="idx" class="chat-message">
                        <span class="chat-sender">{{ c.sender }}:</span>
                        <span class="chat-text">{{ c.message }}</span>
                    </div>
                </div>
            </div>

            <div class="game-log">
            <div v-for="(log, index) in logs" :key="index" :class="getLogClass(log)">
                <template v-if="isDungeonLeaveButtonLog(log)">
                    <button v-if="isActiveDungeonLeaveButtonLog(log)" @click="confirmLeaveCurrentDungeon">{{ getLogText(log) }}</button>
                    <template v-else>{{ getLogText(log) }}</template>
                </template>
                <template v-else>
                    {{ getLogText(log) }}
                </template>
            </div>
            </div>
            
            <div class="chat-input">
            <input v-model="command" @keyup.enter="sendCommand" placeholder="聊天..." />
            <button @click="sendCommand">发送</button>
            </div>
        </div>

        <!-- 右侧面板区域（包含信息面板） -->
        <div class="right-panels-wrapper">
            <GameInfoPanel
              :visible="showInfoPanel"
              :title="panelTitle"
              :currentPanel="currentPanel"
              :backpackItems="backpackItems"
              :selectedBackpackItem="selectedBackpackItem"
              :viewingItem="viewingItem"
              :discardingItem="discardingItem"
              :discardAmount="discardAmount"
              :discardMax="maxDiscardCount"
              :discardShowAmount="discardShowAmount"
              :sellingItem="sellingItem"
              :sellAmount="sellAmount"
              :sellMax="maxSellCount"
              :playerMoney="playerMoney"
              :hasShopNpc="room?.npcs?.some(n => n.isShop)"
              :skillTab="skillTab"
              :filteredSkills="filteredSkills"
              :selectedSkill="selectedSkill"
              :viewingSkill="viewingSkill"
              :equipmentSlotLabels="equipmentSlots"
              :equipment="equipment"
              :selectedEquipmentSlot="selectedEquipmentSlot"
              :statusAttributes="statusAttributes"
              :worldsList="sortedWorldsList"
              :selectedWorldId="selectedWorldId"
              :selectedWorld="selectedWorld"
              :dungeonsList="sortedDungeonsList"
              :selectedDungeonId="selectedDungeonId"
              :selectedDungeon="selectedDungeon"
              :isInDungeon="isInDungeon"
              :dungeonRoomId="room?.dungeon?.id"
              :dungeonProgress="room?.dungeon?.progress"
              :questsList="questsList"
              :factionsList="factionsList"
              @close="closeInfoPanel"
              @selectBackpackItem="selectBackpackItem"
              @viewItem="viewBackpackItem"
              @discardItem="initiateDiscard"
              @equipItem="equipItem"
              @sellItem="initiateSell"
              @changeDiscardAmount="changeDiscardAmount"
              @confirmDiscard="confirmDiscard"
              @cancelDiscard="cancelDiscard"
              @changeSellAmount="changeSellAmount"
              @confirmSell="confirmSell"
              @cancelSell="cancelSell"
              @changeSkillTab="(tab) => skillTab = tab"
              @selectSkill="selectSkill"
              @viewSkill="viewSkill"
              @selectEquipSlot="selectEquipmentSlot"
              @unequipSlot="unequipItem"
              @selectWorld="selectWorld"
              @enterWorld="enterWorld"
              @selectDungeon="selectDungeon"
              @enterDungeon="enterDungeon" />
            
            <!-- 动作按钮区域 -->
            <div class="action-panel">
                <div class="action-buttons-grid">
                    <button @click="doAction('meditate')" class="action-cmd-btn">打坐</button>
                    <button @click="doAction('heal')" class="action-cmd-btn">疗伤</button>
                    <button @click="doAction('stop')" class="action-cmd-btn">停止</button>
                    <button
                      v-for="action in roomActions"
                      :key="action.id"
                      @click="doRoomAction(action.id)"
                      class="action-cmd-btn">
                      {{ action.label }}
                    </button>
                    <button v-if="isInDungeon" @click="leaveCurrentDungeon" class="action-cmd-btn">离开副本</button>
                    <button v-if="currentPlayer.state === 'DEAD'" @click="doAction('relive')" class="action-cmd-btn relive-btn">复活</button>
                </div>
            </div>
        </div>

        <!-- 右侧工具栏 -->
        <div class="side-toolbar">
            <button @click="toggleStatus" class="tool-btn" title="属性">属</button>
            <button @click="toggleBackpack" class="tool-btn" title="背包">囊</button>
            <button @click="toggleSkills" class="tool-btn" title="技能">技</button>
            <button @click="toggleEquipment" class="tool-btn" title="装备">装</button>
            <button @click="toggleMaps" class="tool-btn" title="地图">图</button>
            <button @click="toggleDungeons" class="tool-btn" title="副本">境</button>
            <button @click="toggleQuests" class="tool-btn" title="任务">令</button>
            <button @click="toggleFactions" class="tool-btn" title="势力">势</button>
        </div>
      </div>
    </div>
    
    <!-- 商店面板 (已移动到 left-content-area 内部) -->
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { usePlayerStore } from '@/stores/player'
import { initWebSocket, removeMessageHandler, sendGameCommand as rawSendGameCommand, sendMessage } from '../utils/ws.js'
import GameShop from '../components/GameShop.vue'
import GameRoomPanel from '../components/GameRoomPanel.vue'
import GameInfoPanel from '../components/GameInfoPanel.vue'
import GameDialoguePanel from '../components/GameDialoguePanel.vue'
import { useChatState } from '../composables/useChatState.js'

const router = useRouter()
const playerStore = usePlayerStore()

// 从 Pinia store 获取当前已选择的角色ID
const characterId = computed(() => playerStore.playerId) 

// 游戏状态
const room = ref(null) // 现在使用 null 或更完整的默认对象
const currentPlayer = ref({
  id: '',
  characterName: '未知玩家', // 用于前端显示
  userId: '',
  state: 'IDLE' // Add state
})
const logs = ref([])
const pendingDungeonLeave = ref(null)
const DUNGEON_LEAVE_BUTTON_LOG_TYPE = 'dungeon_leave_button'

const COMMAND_LIMIT_PER_WINDOW = 15
const COMMAND_LIMIT_WINDOW_MS = 2000
const commandQueue = []
const commandSentTimestamps = []
let commandQueueTimer = null
let lastRateLimitNoticeAt = 0

// 面板状态
const showInfoPanel = ref(false)
const currentPanel = ref('') // 'backpack', 'skills', etc.
const panelTitle = ref('')
const backpackItems = ref([])
const playerMoney = ref(0) // Add player money state
const skillsList = ref([])
const statusAttributes = ref(null)
const worldsList = ref([])
const dungeonsList = ref([])
const questsList = ref([])
const factionsList = ref([])
const selectedEntityKey = ref(null)
const selectedWorldId = ref(null)
const selectedDungeonId = ref(null)
const suppressEquipmentPanelSwitchCount = ref(0)

const combinedEntities = computed(() => {
  const result = []
  if (room.value?.playersInRoom) {
    for (const p of room.value.playersInRoom) {
      result.push({
        id: p.id,
        name: p.name,
        isPlayer: true,
        isNpc: false,
        isItem: false,
        level: p.level,
        hp: p.attributes?.currentHp,
        maxHp: p.attributes?.maxHp,
        hostile: false,
        state: p.state,
        online: p.online,
        isShop: false,
        capabilities: [],
        description: '',
        attributes: p.attributes
      })
    }
  }
  if (room.value?.npcs) {
    for (const n of room.value.npcs) {
      result.push({
        id: n.id,
        name: n.name,
        isPlayer: false,
        isNpc: true,
        isItem: false,
        level: n.level,
        hp: n.attributes?.currentHp,
        maxHp: n.attributes?.maxHp,
        hostile: n.hostile || false,
        state: n.state,
        isShop: n.isShop || false,
        capabilities: n.capabilities || [],
        dialogueId: n.dialogueId || '',
        questIds: n.questIds || [],
        skills: n.skills || {},
        description: n.description || '',
        attributes: n.attributes
      })
    }
  }
  return result
})

const selectedEntityObj = computed(() => {
  if (!selectedEntityKey.value) return null
  return combinedEntities.value.find(e => `entity_${e.id}` === selectedEntityKey.value) || null
})

const handleEntitySelect = (entity) => {
  if (!entity) return
  const key = `entity_${entity.id}`
  if (selectedEntityKey.value === key) {
    selectedEntityKey.value = null
  } else {
    selectedEntityKey.value = key
  }
}

// Shop State
const showShopPanel = ref(false)
const shopItems = ref([])
const currentShopNpc = ref(null)
const selectedShopItem = ref(null)
const buyAmount = ref(1)

// Learn State
const showLearnPanel = ref(false)
const learnSkills = ref([])
const currentTeacher = ref(null)
const selectedLearnSkill = ref(null)

// Dialogue State
const showDialoguePanel = ref(false)
const activeDialogue = ref(null)
const activeDialogueNpcId = ref('')

// Backpack State
const selectedBackpackItem = ref(null)
const viewingItem = ref(null)
const discardingItem = ref(null)
const discardAmount = ref(1)
const sellingItem = ref(null)
const sellAmount = ref(1)

// Equipment State
const equipment = ref({})
const selectedEquipmentSlot = ref(null)
const equipmentSlots = {
    "WEAPON": "武器",
    "HEAD": "头部",
    "BODY": "衣服",
    "FEET": "鞋子",
    "NECKLACE": "项链",
    "RING": "戒指",
    "ACCESSORY": "配饰",
    "HIDDEN_WEAPON": "暗器"
}

// Skill State
const selectedSkill = ref(null)
const viewingSkill = ref(null)
const skillTab = ref('basic') // 'basic' or 'special'

const filteredSkills = computed(() => {
    if (!skillsList.value) return [];
    return skillsList.value.filter(skill => {
        if (skillTab.value === 'basic') {
            return skill.id.startsWith('basic_');
        } else {
            return !skill.id.startsWith('basic_');
        }
    });
});

const selectSkill = (skill) => {
    if (selectedSkill.value && selectedSkill.value.id === skill.id) {
        selectedSkill.value = null;
    } else {
        selectedSkill.value = skill;
        viewingSkill.value = null;
    }
}

const viewSkill = (skill) => {
    viewingSkill.value = viewingSkill.value?.id === skill.id ? null : skill;
}

const sortedWorldsList = computed(() => {
    if (!worldsList.value) return [];
    return [...worldsList.value];
});

const selectedWorld = computed(() => {
    if (!selectedWorldId.value || !worldsList.value) return null;
    return worldsList.value.find(w => w.id === selectedWorldId.value);
});

const sortedDungeonsList = computed(() => {
    if (!dungeonsList.value) return [];
    return [...dungeonsList.value].sort((a, b) => {
        const orderA = Number(a.order) || 999;
        const orderB = Number(b.order) || 999;
        if (orderA !== orderB) return orderA - orderB;
        return String(a.id || '').localeCompare(String(b.id || ''));
    });
});

const selectedDungeon = computed(() => {
    if (!selectedDungeonId.value || !dungeonsList.value) return null;
    return dungeonsList.value.find(d => d.id === selectedDungeonId.value) || null;
});

const roomActionLabels = {
    gather: '采集',
    work: '劳作',
    listen_rumor: '听传闻',
    search: '搜索',
    inspect_fragment: '辨认残简',
    inspect_token: '查验令牌',
    tame_horse: '驯马'
};

const roomActions = computed(() => {
    const actions = Array.isArray(room.value?.actions) ? room.value.actions : [];
    return actions.map(id => ({ id, label: roomActionLabels[id] || id }));
});

const isInDungeon = computed(() => {
    const worldId = room.value?.worldId;
    return typeof worldId === 'string' && worldId.startsWith('dungeon_');
});

const keepCurrentPlayerOnlineInRoom = () => {
    if (!room.value || !room.value.playersInRoom || !currentPlayer.value.id) return;
    room.value.playersInRoom = room.value.playersInRoom.map(p => (
        p.id === currentPlayer.value.id ? { ...p, online: true } : p
    ));
}

const applyDungeonProgressFromText = (message) => {
    if (!room.value || !message) return;
    const match = /当前完成度(\d+)%/.exec(message);
    if (!match) return;
    const progress = Math.max(0, Math.min(100, Number(match[1]) || 0));
    const dungeon = room.value.dungeon ? { ...room.value.dungeon } : {};
    dungeon.progress = progress;
    room.value = { ...room.value, dungeon };
    syncPendingDungeonLeavePrompt();
}

const formatDungeonSettlement = (settlement) => {
    if (!settlement) return '';
    const progress = Number(settlement.progress) || 0;
    const rewardExp = Number(settlement.rewardExperience) || 0;
    const rewardPotential = Number(settlement.rewardPotential) || 0;
    const prefix = progress >= 100 ? '副本完成' : '副本结算';
    return `${prefix}：完成度${progress}%，获得${rewardExp}点经验，${rewardPotential}点潜能。`;
}

const buildDungeonLeavePrompt = () => {
    if (!isInDungeon.value || !room.value) return null;
    const dungeonName = room.value?.dungeon?.name || room.value?.worldName || selectedDungeon.value?.name || '未知';
    const progress = Math.max(0, Math.min(100, Number(room.value?.dungeon?.progress ?? 0) || 0));
    return { name: dungeonName, progress, token: `${Date.now()}_${Math.random().toString(36).slice(2, 8)}` };
}

const syncPendingDungeonLeavePrompt = () => {
    if (!pendingDungeonLeave.value) return;
    const next = buildDungeonLeavePrompt();
    if (!next) {
        pendingDungeonLeave.value = null;
        return;
    }
    pendingDungeonLeave.value = next;
}

const syncDungeonSelectionFromRoom = () => {
    const currentDungeonId = room.value?.dungeon?.id;
    if (currentDungeonId) {
        selectedDungeonId.value = currentDungeonId;
    }
}

const trimCommandWindow = (now) => {
    while (commandSentTimestamps.length > 0 && (now - commandSentTimestamps[0]) >= COMMAND_LIMIT_WINDOW_MS) {
        commandSentTimestamps.shift();
    }
}

const scheduleCommandFlush = (delayMs) => {
    if (commandQueueTimer) {
        clearTimeout(commandQueueTimer)
    }
    commandQueueTimer = setTimeout(() => {
        commandQueueTimer = null
        flushCommandQueue()
    }, delayMs)
}

const flushCommandQueue = () => {
    if (commandQueue.length === 0) return

    while (commandQueue.length > 0) {
        const now = Date.now()
        trimCommandWindow(now)

        if (commandSentTimestamps.length >= COMMAND_LIMIT_PER_WINDOW) {
            if (now - lastRateLimitNoticeAt >= COMMAND_LIMIT_WINDOW_MS) {
                addLog("不要急，慢慢来")
                lastRateLimitNoticeAt = now
            }
            const waitMs = Math.max(1, COMMAND_LIMIT_WINDOW_MS - (now - commandSentTimestamps[0]))
            scheduleCommandFlush(waitMs)
            return
        }

        const next = commandQueue.shift()
        rawSendGameCommand(next.type, next.subtype, next.playerId, next.args)
        commandSentTimestamps.push(now)
    }
}

const sendGameCommand = (type, subtype, playerId, args = {}) => {
    if (type !== 'command') {
        rawSendGameCommand(type, subtype, playerId, args)
        return
    }

    commandQueue.push({ type, subtype, playerId, args })
    flushCommandQueue()
}

const {
    chatMessages,
    command,
    chatHistoryEl,
    addChatMessage,
    sendChatCommand
} = useChatState({
    sendGameCommand,
    getPlayerId: () => characterId.value,
    addLog: (message) => addLog(message)
})

const keyDirectionMap = {
    ArrowUp: 'north',
    ArrowDown: 'south',
    ArrowLeft: 'west',
    ArrowRight: 'east',
    Numpad8: 'north',
    Numpad2: 'south',
    Numpad4: 'west',
    Numpad6: 'east',
    Numpad3: 'southeast',
    Numpad9: 'northeast',
    Numpad1: 'southwest',
    Numpad7: 'northwest'
}

const isTypingElement = (el) => {
    if (!el || !(el instanceof HTMLElement)) return false
    const tag = el.tagName
    return el.isContentEditable || tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT'
}

const handleGlobalKeydown = (event) => {
    if (event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) return
    if (isTypingElement(event.target)) return

    const direction = keyDirectionMap[event.code]
    if (!direction) return

    event.preventDefault()
    if (document.activeElement instanceof HTMLElement && !isTypingElement(document.activeElement)) {
        document.activeElement.blur()
    }
    window.getSelection()?.removeAllRanges()
    go(direction)
}

const requestCoreGameState = () => {
    sendGameCommand("system", "get_room", characterId.value, {})
    sendGameCommand("command", "get_attributes", characterId.value, {})
}

const refreshOpenPanels = () => {
    if (!showInfoPanel.value) return

    if (currentPanel.value === 'backpack') {
        sendGameCommand("command", "get_backpack", characterId.value, {})
    } else if (currentPanel.value === 'skills') {
        sendGameCommand("command", "get_skills", characterId.value, {})
    } else if (currentPanel.value === 'equipment') {
        sendGameCommand("command", "get_equipment", characterId.value, {})
    } else if (currentPanel.value === 'status') {
        sendGameCommand("command", "get_attributes", characterId.value, {})
    } else if (currentPanel.value === 'maps') {
        sendGameCommand("command", "get_worlds", characterId.value, {})
    } else if (currentPanel.value === 'dungeons') {
        sendGameCommand("command", "get_dungeons", characterId.value, {})
    } else if (currentPanel.value === 'quests') {
        sendGameCommand("command", "get_quests", characterId.value, {})
    } else if (currentPanel.value === 'factions') {
        sendGameCommand("command", "get_factions", characterId.value, {})
    }
}

const isMaxAmountReached = computed(() => {
    if (!selectedShopItem.value) return true;
    if (selectedShopItem.value.limit) {
        return buyAmount.value >= selectedShopItem.value.limit;
    }
    return buyAmount.value >= 10000;
});

const changeAmount = (delta) => {
    let newAmount = buyAmount.value;
    
    if (delta === 'min') {
        newAmount = 1;
    } else if (delta === 'max') {
        if (selectedShopItem.value.limit) {
            newAmount = selectedShopItem.value.limit;
        } else {
            newAmount = 10000;
        }
    } else {
        newAmount += delta;
    }

    if (newAmount < 1) newAmount = 1;
    
    if (selectedShopItem.value.limit && newAmount > selectedShopItem.value.limit) {
        newAmount = selectedShopItem.value.limit;
    } else if (!selectedShopItem.value.limit && newAmount > 10000) {
        newAmount = 10000;
    }
    
    buyAmount.value = newAmount;
}

onMounted(() => {
  if (!playerStore.isCharacterSelected) {
    router.push({ name: 'CharacterSelect' })
    return
  }
  
  // 初始化 WebSocket 监听器
  initWebSocket(handleMessage);
    window.addEventListener('keydown', handleGlobalKeydown)

    if (playerStore.userId && playerStore.reconnectToken && characterId.value) {
        sendMessage({
            type: 'system',
            subtype: 'reconnect',
            args: {
                userId: playerStore.userId,
                playerId: characterId.value,
                reconnectToken: playerStore.reconnectToken,
            }
        })
    }
  
        requestCoreGameState()
})

onUnmounted(() => {
    if (commandQueueTimer) {
        clearTimeout(commandQueueTimer)
        commandQueueTimer = null
    }
    removeMessageHandler(handleMessage)
    window.removeEventListener('keydown', handleGlobalKeydown)
})

// ------------------- WebSocket 消息处理 -------------------

const describeCommandError = (error, fallback) => {
    if (typeof error === 'string' && /[\u3400-\u9fff]/.test(error)) return error;
    return fallback;
}

const getMessage = (msg) => {
    const { type, subtype, flag, results } = msg;
    
    if (type === 'command') {
        if (subtype === 'heal') {
            if (flag) return "你开始运功疗伤...";
            const err = results.error;
            if (err === 'combat_state') return "战斗中无法疗伤！";
            if (err === 'dead_state') return "死人是无法疗伤的。";
            if (err === 'learning_state') return "你正在学习，无法分心疗伤。";
            if (err === 'hp_full') return "你气血充沛，不需要疗伤";
            if (err === 'mp_insufficient') return "你的内力不足，无法疗伤。";
            return describeCommandError(err, "你现在无法运功疗伤。");
        }
        if (subtype === 'meditate') {
            if (flag) {
                // 如果有具体消息（如内力增加），则不显示默认的开始消息，避免重复或冗余
                return (results && results.message) ? null : "你盘膝而坐，开始吐纳调息...";
            }
            const err = results ? results.error : 'unknown_error';
            if (err === 'combat_state') return "战斗中无法打坐！";
            if (err === 'dead_state') return "死人是无法打坐的。";
            if (err === 'learning_state') return "你正在学习，无法同时打坐。";
            if (err === 'full_state') return "你现在精力充沛，无需打坐。";
            return describeCommandError(err, "你现在无法静心打坐。");
        }
        if (subtype === 'go') {
            if (flag) {
                if (results.room) return `你来到了${results.room.name}`;
                return null;
            }
            const err = results.error;
            if (err === 'combat_state') return "你正在战斗中，无法脱身！";
            if (err === 'dead_state') return "你已经死了，动弹不得。";
            if (err === 'learning_state') return "你正在学习，无法随意走动。";
            if (err === 'no_exit_in_that_direction') return "这个方向没有路。";
            return describeCommandError(err, "你现在无法向这个方向移动。");
        }
        if (subtype === 'spar') {
            if (flag) return results?.message || null;
            const err = results.error;
            if (err === 'combat_state') return "你正在战斗中。";
            if (err === 'dead_state') return "你已经死了，无法切磋。";
            if (err === 'busy') return "你现在正忙着呢。";
            if (err === 'self_not_full') return "你先调整好自己的状态再跟人家比试吧！";
            if (err === 'target_not_full') return `${results.targetName || '对方'}状态不好，你想趁人之危吗？`;
            if (err === 'target_missing') return "目标不能为空";
            if (err === 'self_target') return "和自己比试就不必了。";
            if (err === 'target_not_found') return "找不到目标";
            if (err === 'npc_combat_unsupported') return "当前版本暂不支持与 NPC 切磋。";
            if (err === 'target_not_in_room') return "目标不在当前房间";
            if (err === 'target_is_dead') return "你要比试谁？";
            if (err === 'target_busy') return `${results.targetName || '对方'}现在正忙着。`;
            return describeCommandError(err, "眼下并不适合切磋。");
        }
        if (subtype === 'kill') {
            if (flag && results?.message) return results.message;
            const err = results.error;
            if (err === 'busy_state') return "你现在正忙着呢。";
            if (err === 'dead_state') return "你已经死了，无法攻击。";
            if (err === 'target_missing') return "目标不能为空";
            if (err === 'self_target') return "你总不能对自己下手吧？";
            if (err === 'target_in_combat') return `${results.targetName || '对方'}正在战斗中。`;
            if (err === 'target_not_found') return "找不到目标";
            if (err === 'target_not_in_room') return "目标不在当前房间";
            if (err === 'npc_combat_unsupported') return "当前版本暂不支持与 NPC 战斗。";
            if (err === 'target_is_dead') return "你要击杀谁？";
            return describeCommandError(err, "你现在无法向对方出手。");
        }
        if (subtype === 'learn') {
            if (flag) return null;
            const err = results.error;
            if (err === 'busy_state') return "你现在正忙着呢。";
            if (err === 'no_potential') return "你的潜能不足，无法继续学习。";
            if (err === 'skill_not_taught') return "这位老师似乎不会这招。";
            if (err === 'npc_not_found') return "找不到这位老师。";
            if (typeof err === 'string' && err.includes('已经不输你师傅了')) return err;
            return describeCommandError(err, "你现在无法继续学习这门武功。");
        }
        if (subtype === 'stop') {
            if (flag) return results?.message || null;
            if (results?.error === 'escape_failed') return "你想抽身后退，却被对手缠得脱不开身。";
            return describeCommandError(results?.error, "你一时无法停下当前动作。");
        }
        if (subtype === 'enter_dungeon') {
            if (flag) return null;
            const err = results?.error;
            if (err === 'already_in_dungeon') return "你已经在副本中了。";
            if (err === 'dungeon_not_found') return "没有找到这个副本。";
            if (err === 'dungeon_unavailable') return "副本系统暂不可用。";
            if (err === 'busy_state') return "你现在正忙着呢。";
            if (err === 'dead_state') return "你已经重伤不起，先复活再说。";
            return describeCommandError(err, "你现在无法进入这处秘境。");
        }
        if (subtype === 'leave_dungeon') {
            if (flag) return null;
            const err = results?.error;
            if (err === 'not_in_dungeon') return "你当前不在副本中。";
            return describeCommandError(err, "你现在无法离开这处秘境。");
        }
    }
    return null;
}

const handleMessage = (msg) => {
  // 优先使用前端生成的消息
  const generatedMsg = getMessage(msg);
  if (generatedMsg) {
      addLog(generatedMsg);
  }

    if (msg.type === 'system' && msg.subtype === 'get_room') {
    if (msg.flag) {
      room.value = msg.results.room
            closeShop();
            closeLearn();
            closeDialogue();
      
      // 注意：后端 Player 对象现在包含 characterName/username
      // 根据您的 Player.java 结构，它有一个 getUsername()，我们假设在后端
      // GetRoomHandler 中将其映射为 characterName 或直接使用 username
      if (msg.results.player) {
          // 假设后端返回的 player 对象包含 characterName 字段
          currentPlayer.value = { 
            id: msg.results.player.id,
            characterName: msg.results.player.username || msg.results.player.characterName, 
            userId: msg.results.player.userId, // 如果后端返回
            state: msg.results.player.state || 'IDLE'
          };
                    keepCurrentPlayerOnlineInRoom();
                    syncDungeonSelectionFromRoom();
                    syncPendingDungeonLeavePrompt();
      }
            syncPendingDungeonLeavePrompt();
      
      (msg.logs || []).forEach(l => addLog(l));
    } else {
      addLog(describeCommandError(msg.results.error, '眼前景象一时难以辨清。'))
    }
  } else if (msg.type === 'command' && msg.subtype === 'go') {
    if (msg.flag) {
      room.value = msg.results.room
      // 切换房间时清除选中状态
      selectedEntityKey.value = null;
    closeShop();
    closeLearn();
            closeDialogue();
            keepCurrentPlayerOnlineInRoom();
            syncDungeonSelectionFromRoom();
            syncPendingDungeonLeavePrompt();
      // 玩家移动成功，通常只更新日志和房间
      (msg.logs || []).forEach(l => addLog(l));
    } else {
      if (!generatedMsg) addLog(describeCommandError(msg.results.error, '你现在无法向这个方向移动。'))
    }
  } else if (msg.type === 'command' && msg.subtype === 'get_backpack') {
      // 处理背包信息
      if (msg.flag) {
          backpackItems.value = msg.results.items || [];
          playerMoney.value = msg.results.money || 0; // Update money
          // 如果面板未打开或当前不是背包，则打开/切换到背包
          if (!showInfoPanel.value || currentPanel.value !== 'backpack') {
              showInfoPanel.value = true;
              currentPanel.value = 'backpack';
              panelTitle.value = '背包';
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你一时无法查看背包。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'get_skills') {
      // 处理技能信息
      if (msg.flag) {
          skillsList.value = msg.results.skills || [];
          if (!showInfoPanel.value || currentPanel.value !== 'skills') {
              showInfoPanel.value = true;
              currentPanel.value = 'skills';
              panelTitle.value = '技能';
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你一时无法查看所学武功。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'get_attributes') {
      // 处理属性信息
      if (msg.flag) {
          statusAttributes.value = msg.results.attributes || {};
          if (!showInfoPanel.value || currentPanel.value !== 'status') {
              showInfoPanel.value = true;
              currentPanel.value = 'status';
              panelTitle.value = '属性';
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你一时无法察看自身状态。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'get_equipment') {
      if (msg.flag) {
          equipment.value = msg.results.equipment || {};
          if (suppressEquipmentPanelSwitchCount.value > 0) {
              suppressEquipmentPanelSwitchCount.value--;
          } else if (!showInfoPanel.value || currentPanel.value !== 'equipment') {
              showInfoPanel.value = true;
              currentPanel.value = 'equipment';
              panelTitle.value = '装备';
          }
      } else {
          if (suppressEquipmentPanelSwitchCount.value > 0) {
              suppressEquipmentPanelSwitchCount.value--;
          }
          addLog(describeCommandError(msg.results.error, '你一时无法查看随身装备。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'equip') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          suppressEquipmentPanelSwitchCount.value++;
          sendGameCommand("command", "get_backpack", characterId.value, {});
          sendGameCommand("command", "get_equipment", characterId.value, {});
      } else {
          addLog(describeCommandError(msg.results.error, '这件物品现在无法装备。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'unequip') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          // Refresh both equipment and backpack
          sendGameCommand("command", "get_backpack", characterId.value, {});
          sendGameCommand("command", "get_equipment", characterId.value, {});
      } else {
          addLog(describeCommandError(msg.results.error, '这件装备现在无法脱下。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'mine') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          if (msg.results && msg.results.experience !== undefined && statusAttributes.value) {
              statusAttributes.value.experience = msg.results.experience;
              statusAttributes.value.potential = msg.results.potential;
          }
          if (showInfoPanel.value && currentPanel.value === 'backpack') {
              sendGameCommand("command", "get_backpack", characterId.value, {});
          }
      } else {
          if (msg.results.error === 'no_pickaxe') {
              addLog("挖矿不用稿子可不行");
          } else if (msg.results.error === 'wrong_location') {
              addLog("这里没有矿可挖。");
          } else if (msg.results.error === 'busy') {
              addLog("你现在正忙着，没法挖矿。");
          } else if (msg.results.error === 'dead_state') {
              addLog("你已经死了，没法挖矿。");
          } else {
              addLog(describeCommandError(msg.results.error, '你现在无法在此采矿。'));
          }
      }
  } else if (msg.type === 'command' && msg.subtype === 'get_worlds') {
      // 处理世界列表
      if (msg.flag) {
          worldsList.value = msg.results.worlds || [];
          if (!showInfoPanel.value || currentPanel.value !== 'maps') {
              showInfoPanel.value = true;
              currentPanel.value = 'maps';
              panelTitle.value = '世界';
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你一时无法查阅地图。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'get_dungeons') {
      if (msg.flag) {
          dungeonsList.value = msg.results.dungeons || [];
          if (!showInfoPanel.value || currentPanel.value !== 'dungeons') {
              showInfoPanel.value = true;
              currentPanel.value = 'dungeons';
              panelTitle.value = '副本';
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你一时无法探查附近秘境。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'enter_world') {
      // 处理进入世界
      if (msg.flag) {
          room.value = msg.results.room;
          // 清除选中状态
          selectedEntityKey.value = null;
          (msg.logs || []).forEach(l => addLog(l));
          closeInfoPanel();
          closeShop();
          closeLearn();
            closeDialogue();
          keepCurrentPlayerOnlineInRoom();
          syncDungeonSelectionFromRoom();
          pendingDungeonLeave.value = null;
      } else {
          addLog(describeCommandError(msg.results.error, '你现在无法前往那处地方。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'enter_dungeon') {
      if (msg.flag) {
          room.value = msg.results.room;
          selectedEntityKey.value = null;
          (msg.logs || []).forEach(l => addLog(l));
          closeInfoPanel();
          closeShop();
          closeLearn();
            closeDialogue();
          keepCurrentPlayerOnlineInRoom();
          syncDungeonSelectionFromRoom();
          pendingDungeonLeave.value = null;
      } else if (!generatedMsg) {
          addLog(describeCommandError(msg.results.error, '你现在无法进入这处秘境。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'leave_dungeon') {
      if (msg.flag) {
          room.value = msg.results.room;
          selectedEntityKey.value = null;
          (msg.logs || []).forEach(l => addLog(l));
          closeInfoPanel();
          closeShop();
          closeLearn();
            closeDialogue();
          keepCurrentPlayerOnlineInRoom();
          syncDungeonSelectionFromRoom();
          pendingDungeonLeave.value = null;
          sendGameCommand("command", "get_attributes", characterId.value, {});
      } else if (!generatedMsg) {
          pendingDungeonLeave.value = null;
          addLog(describeCommandError(msg.results.error, '你现在无法离开这处秘境。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'get_shop') {
      if (msg.flag) {
          shopItems.value = msg.results.shopItems || [];
          if (msg.results.money !== undefined) {
              playerMoney.value = msg.results.money;
          }
          currentShopNpc.value = {
              id: msg.results.npcId,
              name: msg.results.npcName,
              title: msg.results.npcTitle
          };
          showLearnPanel.value = false;
          showShopPanel.value = msg.results?.shopOpen !== false;
          selectedShopItem.value = null;
          buyAmount.value = 1;
      } else {
          addLog(describeCommandError(msg.results.error, '对方此刻无意做买卖。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'buy') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          // Update money and backpack if returned
          if (msg.results.money !== undefined) playerMoney.value = msg.results.money;
          
          // Update shop item limit locally
          if (msg.results.boughtItemId && msg.results.boughtAmount) {
              const boughtId = msg.results.boughtItemId;
              const amount = msg.results.boughtAmount;
              
              const item = shopItems.value.find(i => i.id === boughtId);
              if (item && item.limit !== undefined) {
                  item.limit = Math.max(0, item.limit - amount);
                  // If currently selected item is the one bought, update it too (it should be reactive, but just in case)
                  if (selectedShopItem.value && selectedShopItem.value.id === boughtId) {
                      // Vue reactivity should handle this if selectedShopItem is a reference to the object in shopItems
                      // But if it's a copy, we might need to update it. 
                      // In selectShopItem we do: selectedShopItem.value = item; so it is a reference.
                      if (item.limit === 0) {
                          selectedShopItem.value = null;
                      }
                  }
              }
          }

          // We might want to refresh backpack if it's open, but the command returns it?
          // The handler returns "backpack" (list of IDs).
          // But get_backpack returns list of Item objects.
          // So we can't directly update backpackItems with msg.results.backpack (IDs).
          // We should probably trigger a get_backpack if the panel is open.
          if (showInfoPanel.value && currentPanel.value === 'backpack') {
              sendGameCommand("command", "get_backpack", characterId.value, {});
          }
      } else {
          addLog(describeCommandError(msg.results.error, '这桩买卖没有谈成。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'discard_item') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          // Refresh backpack immediately
          sendGameCommand("command", "get_backpack", characterId.value, {});
      } else {
          addLog(describeCommandError(msg.results.error, '这件物品现在无法丢弃。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'sell') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          // Refresh backpack immediately
          sendGameCommand("command", "get_backpack", characterId.value, {});
      } else {
          addLog(describeCommandError(msg.results.error, '对方没有收下这件物品。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'talk') {
      if (msg.flag) {
          activeDialogue.value = msg.results.dialogue || null;
          activeDialogueNpcId.value = activeDialogue.value?.npcId || activeDialogueNpcId.value;
          showShopPanel.value = false;
          showLearnPanel.value = false;
          showDialoguePanel.value = !!activeDialogue.value;
      } else {
          addLog(describeCommandError(msg.results.error, '对方此刻无意交谈。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'choose_dialogue') {
      if (msg.flag) {
          activeDialogue.value = msg.results.dialogue || activeDialogue.value;
          showDialoguePanel.value = !!activeDialogue.value;
          (msg.logs || []).forEach(l => addLog(l));
          sendGameCommand("command", "get_quests", characterId.value, {});
          sendGameCommand("command", "get_attributes", characterId.value, {});
          if (showInfoPanel.value && currentPanel.value === 'factions') {
              sendGameCommand("command", "get_factions", characterId.value, {});
          }
          if (showInfoPanel.value && currentPanel.value === 'backpack') {
              sendGameCommand("command", "get_backpack", characterId.value, {});
          }
      } else {
          addLog(describeCommandError(msg.results.error, '对方没有回应这句话。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'get_quests') {
      if (msg.flag) {
          questsList.value = msg.results.quests || [];
          if (!showInfoPanel.value || currentPanel.value !== 'quests') {
              showInfoPanel.value = true;
              currentPanel.value = 'quests';
              panelTitle.value = '任务';
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你一时无法理清身上的事务。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'get_factions') {
      if (msg.flag) {
          factionsList.value = msg.results.factions || [];
          if (!showInfoPanel.value || currentPanel.value !== 'factions') {
              showInfoPanel.value = true;
              currentPanel.value = 'factions';
              panelTitle.value = '势力';
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你一时无法查明江湖势力。'));
      }
  } else if (msg.type === 'chat') {
      // 聊天消息处理（公开聊天）
      addChatMessage({
          sender: msg.results?.senderName || '???',
          message: msg.results?.message || '',
          ts: msg.ts || Date.now()
      });
  } else if (msg.type === 'system' && msg.subtype === 'session_replaced') {
    // 【新增】其他会话选择了该角色，当前连接被替换
    addLog(`ERROR: ${msg.results.error}`);
    // 延迟1秒后自动退出游戏，返回角色选择界面（保持登录状态）
    setTimeout(() => {
      playerStore.setPlayerId(null);
      // 不调用 clearAuth()，保持 userId 以便继续使用该账户
      router.push({ name: 'CharacterSelect' });
    }, 1000);
    } else if (msg.type === 'system' && msg.subtype === 'reconnect') {
        if (msg.flag) {
            if (msg.results?.playerId) {
                playerStore.setPlayerId(msg.results.playerId)
            }
            requestCoreGameState()
            refreshOpenPanels()
        } else {
            addLog(`ERROR: ${msg.results?.error || '重连失败'}`)
            playerStore.clearAuth()
            router.push({ name: 'Login' })
        }
  } else if (msg.type === 'event' && msg.subtype === 'player_left') {
    // 广播：其他玩家离开了房间
    const leftPlayer = msg.results?.playerName || '未知玩家';
    const leftRoom = msg.results?.roomName || '未知房间';
    addLog(`${leftPlayer}离开了${leftRoom}`);
    // 更新房间信息
    if (msg.results?.room) {
      room.value = msg.results.room;
            keepCurrentPlayerOnlineInRoom();
            syncDungeonSelectionFromRoom();
        syncPendingDungeonLeavePrompt();
    }
  } else if (msg.type === 'event' && msg.subtype === 'player_entered') {
    // 【新增】广播：其他玩家进入了房间
    const enteredPlayer = msg.results?.playerName || '未知玩家';
    const enteredRoom = msg.results?.roomName || '未知房间';
    addLog(`${enteredPlayer}进入了${enteredRoom}`);
    // 更新房间信息
    if (msg.results?.room) {
      room.value = msg.results.room;
            keepCurrentPlayerOnlineInRoom();
            syncDungeonSelectionFromRoom();
        syncPendingDungeonLeavePrompt();
    }
  } else if (msg.type === 'event' && msg.subtype === 'player_status_changed') {
    const pid = msg.results?.playerId;
    const pname = msg.results?.playerName || '未知玩家';
    const status = msg.results?.online ?? msg.results?.status;
    const message = msg.results?.message;
    handlePlayerStatusChanged(pid, pname, status, message);
  } else if (msg.type === 'event' && msg.subtype === 'room_message') {
    // 房间广播消息（战斗信息等）
    const message = msg.results?.message;
    if (message) {
        addLog(message);
    }
  } else if (msg.type === 'event' && msg.subtype === 'room_update') {
    // 房间数据更新（血量变化等）
    if (msg.results?.players && room.value) {
        // 更新房间内的玩家列表数据
        const updatedPlayers = msg.results.players;
        if (room.value.playersInRoom) {
            room.value.playersInRoom = room.value.playersInRoom.map(p => {
                const update = updatedPlayers.find(u => u.id === p.id);
                if (update) {
                    // 如果是自己，更新自己的状态
                    if (p.id === currentPlayer.value.id) {
                        currentPlayer.value.state = update.state;
                    }
                    const merged = { ...p, ...update };
                    if (p.id === currentPlayer.value.id) {
                        merged.online = true;
                    } else if (update.online === undefined) {
                        merged.online = p.online;
                    }
                    return merged;
                }
                return p;
            });
        }
        
        // 更新房间内的 NPC 列表数据
        const updatedNpcs = msg.results.npcs;
        if (updatedNpcs && Array.isArray(updatedNpcs)) {
            const oldNpcMap = new Map((room.value.npcs || []).map(n => [n.id, n]));
            room.value.npcs = updatedNpcs.map(n => ({ ...(oldNpcMap.get(n.id) || {}), ...n }));
        }
        syncPendingDungeonLeavePrompt();
        if (showInfoPanel.value && currentPanel.value === 'skills') {
            sendGameCommand("command", "get_skills", characterId.value, {});
        } else if (showInfoPanel.value && currentPanel.value === 'status') {
            sendGameCommand("command", "get_attributes", characterId.value, {});
        }
    }
  } else if (msg.type === 'command' && msg.subtype === 'do_action') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          if (showInfoPanel.value && currentPanel.value === 'backpack') {
              sendGameCommand("command", "get_backpack", characterId.value, {});
          }
          if (showInfoPanel.value && currentPanel.value === 'status') {
              sendGameCommand("command", "get_attributes", characterId.value, {});
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你现在无法做出这个动作。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'relive') {
      if (msg.flag) {
          addLog(msg.results.message);
          if (msg.results.action === 'refresh_room') {
              sendGameCommand("system", "get_room", characterId.value, {});
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你的神志尚未恢复。'));
      }
  } else if (msg.type === 'command') {
      // 通用命令处理（处理 kill, spar, heal, meditate 等没有特殊 UI 逻辑的命令）
      if (['heal', 'meditate', 'spar', 'kill'].includes(msg.subtype)) {
          // 已由 getMessage 处理
          if (msg.flag && msg.logs) (msg.logs || []).forEach(l => addLog(l));
          if (msg.flag && msg.results && msg.results.message) addLog(msg.results.message); // spar returns message on success
          if (!msg.flag && !generatedMsg) addLog(describeCommandError(msg.results.error, '你现在无法做出这个动作。'));
      } else {
          if (msg.flag) {
              if (msg.results && msg.results.message) {
                  addLog(msg.results.message);
              }
              if (msg.logs && msg.logs.length > 0) {
                 (msg.logs || []).forEach(l => addLog(l));
              }
          } else {
              addLog(describeCommandError(msg.results.error, '你现在无法做出这个动作。'));
          }
      }
  } else {
        return
  }
}

// 统一处理玩家在线/离线状态变化
const handlePlayerStatusChanged = (playerId, playerName, status, message) => {
  if (!room.value || !room.value.playersInRoom) return;
  const p = room.value.playersInRoom.find(x => x.id === playerId);
  const online = status === true || status === 'online' || status === '在线';
    if (p) {
            if (playerId === currentPlayer.value.id) {
                    p.online = true;
            } else {
                    p.online = online;
            }
    }
  if (online) {
    addLog(message || `${playerName}恢复连接`);
  } else {
        if (playerId === currentPlayer.value.id) return;
    addLog(`${playerName}断开了连接`);
  }
}

// ------------------- 面板控制 -------------------

const toggleBackpack = () => {
    if (showInfoPanel.value && currentPanel.value === 'backpack') {
        closeInfoPanel();
    } else {
        // 发送获取背包命令
        sendGameCommand("command", "get_backpack", characterId.value, {});
    }
}

const toggleSkills = () => {
    if (showInfoPanel.value && currentPanel.value === 'skills') {
        closeInfoPanel();
    } else {
        // 发送获取技能命令
        sendGameCommand("command", "get_skills", characterId.value, {});
    }
}

const toggleEquipment = () => {
    if (showInfoPanel.value && currentPanel.value === 'equipment') {
        closeInfoPanel();
    } else {
        sendGameCommand("command", "get_equipment", characterId.value, {});
    }
}

const toggleStatus = () => {
    if (showInfoPanel.value && currentPanel.value === 'status') {
        closeInfoPanel();
    } else {
        // 发送获取属性命令
        sendGameCommand("command", "get_attributes", characterId.value, {});
    }
}

const toggleMaps = () => {
    if (showInfoPanel.value && currentPanel.value === 'maps') {
        closeInfoPanel();
    } else {
        // 发送获取世界列表命令
        sendGameCommand("command", "get_worlds", characterId.value, {});
        
        // 尝试推断当前世界并设置为默认选中
        if (room.value && room.value.id) {
            const prefix = room.value.id.split('_')[0];
            const prefixMap = {
                'ly': 'luoyang'
            };
            selectedWorldId.value = prefixMap[prefix] || 'luoyang';
        } else {
            selectedWorldId.value = 'luoyang';
        }
    }
}

const toggleDungeons = () => {
    if (showInfoPanel.value && currentPanel.value === 'dungeons') {
        closeInfoPanel();
    } else {
        sendGameCommand("command", "get_dungeons", characterId.value, {});
    }
}

const toggleQuests = () => {
    if (showInfoPanel.value && currentPanel.value === 'quests') {
        closeInfoPanel();
    } else {
        sendGameCommand("command", "get_quests", characterId.value, {});
    }
}

const toggleFactions = () => {
    if (showInfoPanel.value && currentPanel.value === 'factions') {
        closeInfoPanel();
    } else {
        sendGameCommand("command", "get_factions", characterId.value, {});
    }
}

const selectWorld = (worldId) => {
    selectedWorldId.value = selectedWorldId.value === worldId ? null : worldId;
}

const selectDungeon = (dungeonId) => {
    selectedDungeonId.value = selectedDungeonId.value === dungeonId ? null : dungeonId;
}

const enterWorld = () => {
    if (!checkCanAct('enter_world')) return;
    if (!selectedWorldId.value) return;
    sendGameCommand("command", "enter_world", characterId.value, { worldId: selectedWorldId.value });
}

const enterDungeon = () => {
    if (!checkCanAct('enter_dungeon')) return;
    if (!selectedDungeonId.value) return;
    sendGameCommand("command", "enter_dungeon", characterId.value, { dungeonId: selectedDungeonId.value });
}

const leaveCurrentDungeon = () => {
    if (!checkCanAct('leave_dungeon')) return;
    if (pendingDungeonLeave.value) return;
    const prompt = buildDungeonLeavePrompt();
    if (!prompt) {
        pendingDungeonLeave.value = null;
        return;
    }
    pendingDungeonLeave.value = prompt;
    addLog(`确认离开${prompt.name}副本？`);
    addLog(`当前副本完成度${prompt.progress}%`);
    addDungeonLeaveButtonLog(prompt.token);
}

const confirmLeaveCurrentDungeon = () => {
    if (!pendingDungeonLeave.value) return;
    if (!checkCanAct('leave_dungeon')) return;
    pendingDungeonLeave.value = null;
    sendGameCommand("command", "leave_dungeon", characterId.value, {});
}

const closeInfoPanel = () => {
    showInfoPanel.value = false;
    currentPanel.value = '';
    // Reset backpack item view state
    viewingItem.value = null;
    discardingItem.value = null;
    sellingItem.value = null;
    selectedBackpackItem.value = null;
}

const closeDialogue = () => {
    showDialoguePanel.value = false;
    activeDialogue.value = null;
    activeDialogueNpcId.value = '';
}

// ------------------- 实体交互 -------------------

const getEntityKey = (id, index, type) => {
    if (type === 'player') {
        return `player_${id}`;
    } else {
        return `npc_${index}_${id}`;
    }
}

const toggleEntity = (id, type, index = null) => {
    if (type === 'player' && id === currentPlayer.value.id) return; // Don't toggle self
    
    const key = getEntityKey(id, index, type);
    
    // 实现再次点击收起功能
    if (selectedEntityKey.value === key) {
        selectedEntityKey.value = null;
    } else {
        selectedEntityKey.value = key;
    }
}

// 状态名称映射，用于显示友好的状态提示
const stateNames = {
    'HEALING': '疗伤',
    'MEDITATING': '打坐',
    'MINING': '挖矿',
    'COMBAT': '战斗',
    'LEARNING': '学习'
};

// 检查当前状态是否允许执行动作
const checkCanAct = (actionType) => {
    // 停止动作总是允许的
    if (actionType === 'stop' || actionType === 'leave_dungeon') return true;
    
    const state = currentPlayer.value.state;
    
    // 如果是死亡状态，仅允许复活（或其他特定逻辑，这里暂不处理，由后端或getMessage处理）
    if (state === 'DEAD') {
        return true; 
    }

    // 检查是否在忙碌状态
    const stateName = stateNames[state];
    if (stateName) {
        addLog(`你正在${stateName}，待会再说`);
        return false;
    }
    
    return true;
}

const doAction = (actionType) => {
    if (!checkCanAct(actionType)) return;

    if (actionType === 'relive') {
        sendGameCommand("command", "relive", characterId.value, {});
    } else {
        sendGameCommand("command", actionType, characterId.value, {});
    }
}

const doRoomAction = (actionType) => {
    if (!checkCanAct(actionType)) return;
    sendGameCommand("command", "do_action", characterId.value, { action: actionType });
}

const getHealthStatusDescription = (currentHp, maxHp) => {
    if (!maxHp) return '看起来深不可测。';
    const percent = (currentHp / maxHp) * 100;
    if (percent >= 100) return '看起来气血充盈，精力旺盛。';
    if (percent >= 80) return '看起来受了点轻伤，不过并无大碍。';
    if (percent >= 60) return '看起来受伤不轻，动作有些迟缓。';
    if (percent >= 40) return '看起来伤痕累累，状况不太妙。';
    if (percent >= 20) return '看起来摇摇欲坠，随时可能倒下。';
    return '看起来已经奄奄一息，离死不远了。';
}

const viewEntity = (entity) => {
    if (entity.state === 'DEAD') {
        const name = entity.name + '的尸体';
        addLog(`${name}\n${name}，静静地躺在这里`);
        return;
    }

    let info = `${entity.name}\n`;
    if (entity.description) {
        info += `${entity.description}\n`;
    }
    
    if (entity.attributes) {
        const hpDesc = getHealthStatusDescription(entity.attributes.currentHp, entity.attributes.maxHp);
        info += `${entity.name}${hpDesc}`;
    } else {
        info += `${entity.name}看起来深不可测。`;
    }
    
    addLog(info);
}

const talkEntity = (entity) => {
    if (!checkCanAct('talk')) return;
    if (!entity?.isNpc) {
        addLog('只能和 NPC 交谈。');
        return;
    }
    activeDialogueNpcId.value = entity.id;
    sendGameCommand("command", "talk", characterId.value, { npcId: entity.id });
}

const chooseDialogue = (optionId) => {
    if (!activeDialogueNpcId.value || !optionId) return;
    sendGameCommand("command", "choose_dialogue", characterId.value, {
        npcId: activeDialogueNpcId.value,
        optionId
    });
}

const killEntity = (entity) => {
    if (!checkCanAct('kill')) return;
    if (!entity) {
        addLog('找不到目标。');
        return;
    }
    sendGameCommand("command", "kill", characterId.value, { targetId: entity.id });
}

const sparEntity = (entity) => {
    if (!checkCanAct('spar')) return;
    if (!entity || entity.state === undefined) {
        addLog('当前版本暂不支持与 NPC 切磋。');
        return;
    }
    sendGameCommand("command", "spar", characterId.value, { targetId: entity.id });
}

const getHpPercent = (entity) => {
    if (!entity || !entity.attributes) return 100;
    if (entity.state === 'DEAD') return 0;
    const max = entity.attributes.maxHp || 100;
    // 修复：如果 currentHp 为 0，原逻辑 (0 || max) 会返回 max，导致显示满血
    const current = (entity.attributes.currentHp !== undefined && entity.attributes.currentHp !== null) 
        ? entity.attributes.currentHp 
        : max;
    return Math.max(0, Math.min(100, (current / max) * 100));
}

const getItemColor = (rarity) => {
    switch (rarity) {
        case 1: return '#D3D3D3'; // Light Gray
        case 2: return '#2ecc71'; // Green
        case 3: return '#3498db'; // Blue
        case 4: return '#f1c40f'; // Yellow
        case 5: return '#9b59b6'; // Purple
        case 6: return '#e67e22'; // Orange
        case 7: return '#e74c3c'; // Red
        default: return '#D3D3D3';
    }
}

const getMpPercent = (entity) => {
    if (!entity || !entity.attributes) return 100;
    const max = entity.attributes.trainedMaxMp || entity.attributes.maxMp || 100;
    const current = (entity.attributes.currentMp !== undefined && entity.attributes.currentMp !== null) 
        ? entity.attributes.currentMp 
        : max;
    return Math.max(0, Math.min(100, (current / max) * 100));
}

// ------------------- 命令发送 -------------------

const sendCommand = sendChatCommand

const go = (direction) => {
  if (!checkCanAct('go')) return;
  // 使用新的 sendGameCommand，传入 characterId
  sendGameCommand("command", "go", characterId.value, { direction: direction })
}

// ------------------- 辅助功能 -------------------

const addLog = (message) => {
    if (!message) return;
    if (typeof message !== 'string') {
        message = String(message);
    }
    // 移除 INFO/WARN/ERROR 前缀
    const cleanMessage = message.replace(/^(INFO|WARN|ERROR):\s*/, '');
    applyDungeonProgressFromText(cleanMessage);
    logs.value.push(cleanMessage);
    scrollToBottom();
}

const addDungeonLeaveButtonLog = (token) => {
    logs.value.push({
        type: DUNGEON_LEAVE_BUTTON_LOG_TYPE,
        text: '离开副本',
        token
    });
    scrollToBottom();
}

const getLogText = (log) => {
    if (typeof log === 'string') return log;
    if (log && typeof log === 'object' && typeof log.text === 'string') return log.text;
    return '';
}

const isDungeonLeaveButtonLog = (log) => {
    return !!(log && typeof log === 'object' && log.type === DUNGEON_LEAVE_BUTTON_LOG_TYPE);
}

const isActiveDungeonLeaveButtonLog = (log) => {
    if (!isDungeonLeaveButtonLog(log)) return false;
    return !!pendingDungeonLeave.value && pendingDungeonLeave.value.token === log.token;
}

const getLogClass = (log) => {
  const text = getLogText(log)
  if (text.startsWith('ERROR:')) return 'log-error'
  if (text.startsWith('WARN:')) return 'log-warn'
  if (text.startsWith('CHAT:')) return 'log-chat'
  return 'log-info'
}

const scrollToBottom = () => {
  // 简单的滚动到底部实现
  const logContainer = document.querySelector('.game-log');
  if (logContainer) {
    // 延迟是为了确保 DOM 已经更新
    setTimeout(() => {
        logContainer.scrollTop = logContainer.scrollHeight;
    }, 0);
  }
}

const logout = () => {
  // 发送注销角色指令，通知后端广播离线
  sendGameCommand("system", "deselect_character", characterId.value, {});

  // 仅清除已选角色（保持登录状态），并跳转到角色选择界面
  // 不调用 clearAuth()，只清除 playerId
  playerStore.setPlayerId(null);
  router.push({ name: 'CharacterSelect' });
};

const openShop = (npc) => {
    if (!checkCanAct('open_shop')) return;
    closeDialogue();
    sendGameCommand("command", "get_shop", characterId.value, { npcId: npc.id });
}

const closeShop = () => {
    showShopPanel.value = false;
    currentShopNpc.value = null;
    shopItems.value = [];
}

const openLearn = (npc) => {
    if (!checkCanAct('learn')) return;
    closeDialogue();
    currentTeacher.value = npc;
    // Convert skills map to array for display
    // npc.skills is { "skillId": level }
    // We might want skill names. But client doesn't have skill DB.
    // We can use a simple mapping or just show ID for now, or ask server for names?
    // The server sends `skills` map.
    // Let's use a hardcoded map for now or just display ID.
    // Better: The server should probably send skill names in get_room? No, that's too much data.
    // But wait, `get_skills` returns skill names.
    // Maybe we can just show the ID or a formatted name if we know it.
    // For now, let's just list them.
    
    const skillNames = {
        "basic_unarmed": "基本拳脚",
        "basic_sword": "基本剑法",
        "basic_blade": "基本刀法",
        "basic_staff": "基本棍法",
        "basic_parry": "基本招架",
        "basic_dodge": "基本躲闪",
        "basic_neigong": "基本内功",
        "basic_qinggong": "基本轻功",
        "basic_literacy": "读书识文"
    };

    learnSkills.value = Object.entries(npc.skills || {}).map(([id, level]) => ({
        id,
        name: skillNames[id] || id,
        level
    }));
    
    showLearnPanel.value = true;
    showShopPanel.value = false; // Close shop if open
    closeInfoPanel(); // Close info panel if open
}

const closeLearn = () => {
    showLearnPanel.value = false;
    currentTeacher.value = null;
    learnSkills.value = [];
    selectedLearnSkill.value = null;
}

const selectLearnSkill = (skill) => {
    selectedLearnSkill.value = skill;
}

const startLearning = () => {
    if (!checkCanAct('learn')) return;
    if (!currentTeacher.value || !selectedLearnSkill.value) return;
    
    sendGameCommand("command", "learn", characterId.value, {
        npcId: currentTeacher.value.id,
        skillId: selectedLearnSkill.value.id
    });
    showLearnPanel.value = false;
}


const selectShopItem = (item) => {
    if (item.limit === 0) return;
    selectedShopItem.value = selectedShopItem.value?.id === item.id ? null : item;
    buyAmount.value = 1;
}

const selectEquipmentSlot = (slot) => {
    if (selectedEquipmentSlot.value === slot) {
        selectedEquipmentSlot.value = null;
    } else {
        selectedEquipmentSlot.value = slot;
    }
}

const equipItem = (item) => {
    if (!checkCanAct('equip')) return;
    sendGameCommand("command", "equip", characterId.value, { itemId: item.id });
    viewingItem.value = null;
    selectedBackpackItem.value = null;
}

const unequipItem = (slot) => {
    if (!checkCanAct('unequip')) return;
    sendGameCommand("command", "unequip", characterId.value, { slot: slot });
    selectedEquipmentSlot.value = null;
}

const buyItem = () => {
    if (!checkCanAct('buy')) return;
    if (!selectedShopItem.value || !currentShopNpc.value) return;
    sendGameCommand("command", "buy", characterId.value, { 
        npcId: currentShopNpc.value.id, 
        itemId: selectedShopItem.value.id,
        amount: buyAmount.value
    });
}

// Backpack Actions
const selectBackpackItem = (item) => {
    const currentId = selectedBackpackItem.value?.uniqueId || selectedBackpackItem.value?.id;
    const targetId = item.uniqueId || item.id;

    if (currentId === targetId) {
        selectedBackpackItem.value = null;
        viewingItem.value = null;
        discardingItem.value = null;
        sellingItem.value = null;
    } else {
        selectedBackpackItem.value = item;
        viewingItem.value = null;
        discardingItem.value = null;
        sellingItem.value = null;
    }
}

const viewBackpackItem = (item) => {
    const currentId = viewingItem.value?.uniqueId || viewingItem.value?.id;
    const targetId = item.uniqueId || item.id;
    viewingItem.value = currentId === targetId ? null : item;
    discardingItem.value = null;
    sellingItem.value = null;
}

const initiateDiscard = (item) => {
    discardingItem.value = item;
    viewingItem.value = null;
    sellingItem.value = null;
    discardAmount.value = 1;
}

const initiateSell = (item) => {
    sellingItem.value = item;
    viewingItem.value = null;
    discardingItem.value = null;
    // Default to max amount for selling
    sellAmount.value = item.count || 1;
}

const changeDiscardAmount = (delta) => {
    let newAmount = discardAmount.value + delta;
    if (newAmount < 1) newAmount = 1;
    
    // Count items in backpack
    // Since we now group items, item.count should be available
    const count = discardingItem.value.count || backpackItems.value.filter(i => i.id === discardingItem.value.id).length;
    if (newAmount > count) newAmount = count;
    
    discardAmount.value = newAmount;
}

const confirmDiscard = () => {
    if (!checkCanAct('discard')) return;
    if (!discardingItem.value) return;
    sendGameCommand("command", "discard_item", characterId.value, { 
        itemId: discardingItem.value.id, 
        amount: discardAmount.value 
    });
    discardingItem.value = null;
    selectedBackpackItem.value = null;
}

const changeSellAmount = (delta) => {
    let newAmount = sellAmount.value;
    const max = sellingItem.value.count || 1;

    if (delta === 'min') {
        newAmount = 1;
    } else if (delta === 'max') {
        newAmount = max;
    } else {
        newAmount += delta;
    }

    if (newAmount < 1) newAmount = 1;
    if (newAmount > max) newAmount = max;
    
    sellAmount.value = newAmount;
}

const isMaxSellReached = computed(() => {
    if (!sellingItem.value) return true;
    const max = sellingItem.value.count || 1;
    return sellAmount.value >= max;
});

const maxSellCount = computed(() => {
    if (!sellingItem.value) return 1;
    return sellingItem.value.count || 1;
});

const maxDiscardCount = computed(() => {
    if (!discardingItem.value) return 1;
    return discardingItem.value.count || backpackItems.value.filter(i => i.id === discardingItem.value.id).length || 1;
});

const discardShowAmount = computed(() => {
    if (!discardingItem.value) return false;
    return backpackItems.value.filter(i => i.id === discardingItem.value.id).length > 1 || (discardingItem.value.stackable && discardingItem.value.count > 1);
});

const confirmSell = () => {
    if (!checkCanAct('sell')) return;
    if (!sellingItem.value) return;
    sendGameCommand("command", "sell", characterId.value, { 
        itemId: sellingItem.value.id, 
        amount: sellAmount.value 
    });
    sellingItem.value = null;
    selectedBackpackItem.value = null;
}

const cancelSell = () => {
    sellingItem.value = null;
}

const cancelDiscard = () => {
    discardingItem.value = null;
}

const isMaxDiscardReached = computed(() => {
    if (!discardingItem.value) return true;
    const count = discardingItem.value.count || backpackItems.value.filter(i => i.id === discardingItem.value.id).length;
    return discardAmount.value >= count;
});

const formatPrice = (money) => {
    if (money === 0) return "0枚铜板";
    const gold = Math.floor(money / 10000);
    const silver = Math.floor((money % 10000) / 100);
    const copper = Math.floor(money % 100);
    
    let str = "";
    if (gold > 0) str += `${gold}两黄金`;
    if (silver > 0) str += `${silver}两银子`;
    if (copper > 0) str += `${copper}枚铜板`;
    
    return str;
}

const formatMoney = (money) => {
    if (!money) return "你身无分文";
    const gold = Math.floor(money / 10000);
    const silver = Math.floor((money % 10000) / 100);
    const copper = Math.floor(money % 100);
    
    let str = "你身上有";
    if (gold > 0) str += `${gold}两黄金`;
    if (silver > 0) str += `${silver}两银子`;
    if (copper > 0) str += `${copper}枚铜板`;
    
    return str;
}
</script>

<style scoped>
/* -----------------------------------
   样式部分 - 保持不变
   ----------------------------------- */
.game-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
    width: 100vw;
    padding: 20px;
    box-sizing: border-box;
    gap: 15px;
    background-color: var(--color-background-dark);
}

.top-info-area {
    background-color: var(--color-background-secondary);
    border: 2px solid var(--color-accent-gold);
    padding: 8px 15px;
    border-radius: 8px;
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
    position: relative;
}

.top-info-area h2 {
    color: var(--color-mud-primary);
    margin-top: 0;
    margin-bottom: 5px;
    border-bottom: none;
    font-size: 1.5em;
}

.room-header {
    margin-top: 5px;
}

.room-header h3 {
    color: var(--color-accent-gold);
    margin: 0 0 4px 0;
    font-size: 1.2em;
}

.room-description {
    color: var(--color-text-light);
    margin: 5px 0 0 0;
    line-height: 1.2;
}

.logout-btn {
  position: absolute;
  top: 10px;
  right: 15px;
  background-color: #f44336;
  border-color: #f44336;
  padding: 5px 10px;
  font-size: 0.9em;
}

.middle-layout {
    display: flex;
    flex-grow: 1;
    gap: 15px;
    overflow: hidden;
    min-height: 0;
}

.left-content-area {
    flex: 0 0 260px; /* Increased width to accommodate shop controls */
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-height: 0;
    overflow: hidden;
}

.room-entities-area {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    padding: 10px;
}

.room-exits-grid {
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-border-subtle);
    padding: 8px;
    border-radius: 8px;
    flex: 0 0 auto;
    margin-top: auto;
    overflow: hidden;
}

.room-exits-grid h3 {
    margin: 0 0 6px;
    padding-bottom: 4px;
    font-size: 0.95em;
}

.room-entities {
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-border-subtle);
    padding: 15px;
    border-radius: 8px;
    flex: 1;
    overflow-y: auto;
}

.room-entities h3 {
    margin-top: 0;
    margin-bottom: 10px;
    color: var(--color-accent-gold);
}

.entities-section {
    margin-bottom: 15px;
}

.entities-section h4 {
    margin: 8px 0 5px 0;
    color: var(--color-mud-primary);
    font-size: 0.95em;
}

.entities-section ul {
    margin: 5px 0;
    padding-left: 4px; /* 减少左侧缩进，仅留少量空隙 */
    list-style: none;
}

.entities-section li {
    color: var(--color-text-light);
    padding: 1px 0;
    font-size: 0.9em;
    line-height: 1.2;
}

.entities-section li:before {
    content: none;
}

.current-player {
    color: var(--color-accent-gold);
    font-weight: bold;
}

.player-offline {
  color: #888; /* 变浅的颜色 */
}

.offline-tag {
  color: #ff4d4f; /* 醒目的断线提示色，或者也用浅色 */
  font-size: 0.8em;
  margin-left: 5px;
}

.status-tag {
    font-size: 0.8em;
    margin-left: 5px;
}

.status-tag.healing {
    color: #2ecc71; /* Green */
}

.status-tag.meditating {
    color: #2ecc71; /* Green */
}

.status-tag.mining {
    color: #2ecc71; /* Green */
}

.status-tag.learning {
    color: #2ecc71; /* Green */
}

/* Entity Interaction Styles */
.entity-item {
    margin-bottom: 5px;
}

.entity-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 4px 5px;
    border-radius: 4px;
    transition: background-color 0.2s;
}

.entity-row.clickable {
    cursor: pointer;
}

.entity-row.clickable:hover {
    background-color: rgba(255, 255, 255, 0.1);
}

.entity-info {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    margin-right: 10px;
}

.entity-bars {
    display: flex;
    flex-direction: column;
    gap: 2px;
    width: 60px; /* Fixed width for bars */
}

.bar-container {
    width: 100%;
    height: 4px;
    background-color: #333;
    border-radius: 2px;
    overflow: hidden;
}

.hp-bar .bar-fill {
    height: 100%;
    background-color: #e74c3c; /* Red for HP */
    transition: width 0.3s ease;
}

.mp-bar .bar-fill {
    height: 100%;
    background-color: #3498db; /* Blue for MP */
    transition: width 0.3s ease;
}

.status-section-title {
    color: var(--color-mud-primary);
    font-weight: bold;
    margin: 5px 0;
    font-size: 0.9em;
}

.status-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 5px 15px;
}

.status-item.full-width {
    grid-column: span 2;
}

.right-log-area {
    flex: 1;
    display: flex;
    flex-direction: row;
    gap: 10px;
    padding: 10px; /* 调整内边距 */
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    overflow: hidden;
}

.log-content-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    min-width: 0; /* 防止 flex 子项溢出 */
}

.game-log {
    flex-grow: 1;
    overflow-y: auto;
    overflow-x: hidden;
    margin-bottom: 5px;
    padding-right: 0;
    line-height: 1.2;
    white-space: pre-wrap; /* Preserve newlines */
}

/* 隐藏滚动条 */
.game-log::-webkit-scrollbar {
    display: none;
}

.game-log {
    -ms-overflow-style: none;
    scrollbar-width: none;
}

.log-info { color: var(--color-mud-primary); }
.log-warn { color: #f39c12; }
.log-error { color: #e74c3c; }
.log-chat { color: var(--color-accent-blue); }


.chat-input {
    display: flex;
    width: 100%;
}

.chat-input input {
    flex-grow: 1;
    padding: 10px;
    border: 1px solid var(--color-border-subtle);
    border-radius: 4px 0 0 4px;
    background-color: var(--color-background-dark);
    color: var(--color-text-light);
}

.chat-input input:focus {
    outline: none;
    border-color: var(--color-border-subtle);
    box-shadow: none;
}

.chat-input button {
    flex: 0 0 auto;
    background-color: var(--color-accent-gold);
    color: var(--color-background-dark);
    border: none;
    padding: 10px 15px;
    border-radius: 0 4px 4px 0;
    cursor: pointer;
}

/* Exits Map Grid 样式 */
.exits-map-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    grid-template-rows: repeat(3, 1fr);

    gap: 3px;
    border: none;

    width: 100%;
    max-width: none;
    margin: 0 auto;
}

.grid-cell {
    min-height: 28px;

    display: flex;
    justify-content: center;
    align-items: center;
    padding: 0;
    box-sizing: border-box;
    border: none;
    background-color: transparent;
}

.grid-cell button {
    width: 100%;
    height: 100%;

    padding: 2px 2px;
    min-height: 28px;

    cursor: pointer;
    white-space: normal;
    text-align: center;
    border: none;
    background-color: var(--color-background-dark, #111);
    color: var(--color-text-light, #fff);
    font-size: 0.78em;
    line-height: 1.1;

    border-radius: 5px;
    border: 1px solid var(--color-border-subtle);
}

.current-room-center {
    width: 95%;
    height: 95%;
    min-height: 28px;

    background-color: var(--color-background-dark);
    color: var(--color-accent-gold);
    border: 2px solid var(--color-accent-gold);
    border-radius: 5px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
}

.room-abbr {
    font-size: 0.78em;
    word-break: break-word;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    max-width: 80%;
}

/* 新增布局样式 */
.info-panel {
    width: 100%; /* 适应父容器 */
    background-color: var(--color-background-dark);
    border: 1px solid var(--color-accent-gold);
    border-radius: 4px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: slideIn 0.3s ease;
    flex: 1; /* Take available space */
    min-height: 0;
    box-sizing: border-box;
}

@keyframes slideIn {
    from { opacity: 0; transform: translateY(-10px); }
    to { opacity: 1; transform: translateY(0); }
}

.panel-header {
    background-color: var(--color-background-secondary);
    padding: 8px 10px;
    border-bottom: 1px solid var(--color-border-subtle);
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.panel-header h3 {
    margin: 0;
    font-size: 1em;
    color: var(--color-accent-gold);
}

.close-btn {
    background: none;
    border: none;
    color: var(--color-text-light);
    cursor: pointer;
    font-size: 1.2em;
    padding: 0 5px;
}

.panel-content {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 10px;
}

/* 隐藏面板内容的滚动条 */
.panel-content::-webkit-scrollbar {
    display: none;
}
.panel-content {
    -ms-overflow-style: none;
    scrollbar-width: none;
}

.side-toolbar {
    width: 40px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    align-items: center;
    padding-top: 5px;
    border-left: 1px solid var(--color-border-subtle);
    padding-left: 10px;
}

.tool-btn {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    border: 1px solid var(--color-accent-gold);
    background-color: var(--color-background-secondary);
    color: var(--color-text-light);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.2em;
    transition: all 0.2s;
}

.tool-btn:hover {
    background-color: var(--color-accent-gold);
    color: var(--color-background-dark);
    transform: scale(1.1);
}

.item-entry {
    display: flex;
    flex-direction: column;
    padding: 3px 0; /* Compact padding */
    border-bottom: 1px solid var(--color-border-subtle);
}

.skill-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.skill-level-tag {
    font-size: 0.85em;
    color: var(--color-accent-gold);
}

.item-name {
    color: var(--color-accent-blue);
    font-weight: bold;
}

.item-desc {
    font-size: 0.85em;
    color: var(--color-text-light);
    opacity: 0.8;
}

.empty-msg {
    color: var(--color-text-light);
    font-style: italic;
    text-align: center;
    margin-top: 20px;
}

.status-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.status-item {
    display: flex;
    justify-content: space-between;
    font-size: 0.95em;
}

.status-item .label {
    color: var(--color-text-light);
}

.status-item .value {
    color: var(--color-accent-gold);
    font-weight: bold;
}

.status-divider {
    border: 0;
    border-top: 1px dashed var(--color-border-subtle);
    margin: 5px 0;
}

.right-panels-wrapper {
    display: flex;
    flex-direction: column;
    gap: 10px;
    /* 固定宽度用于右侧工具栏，但允许高度伸展 */
    width: 250px;
    min-height: 0; /* 允许子元素使用 flex:1 并正确滚动 */
    height: 100%;
    box-sizing: border-box;
}

.action-panel {
    width: 100%;
    background-color: var(--color-background-dark);
    border: 1px solid var(--color-accent-gold);
    border-radius: 4px;
    padding: 10px;
    flex: 0 0 auto;
    margin-top: auto; /* 确保始终位于底部 */
    box-sizing: border-box;
}

.action-buttons-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    justify-content: flex-start;
}

.action-cmd-btn {
    padding: 5px 10px;
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-light);
    cursor: pointer;
    border-radius: 4px;
    transition: all 0.2s;
    font-size: 0.9em;
    width: auto; /* 恰好容纳文字 */
    flex: 0 0 auto;
}

.action-cmd-btn:hover {
    background-color: var(--color-accent-gold);
    color: var(--color-background-dark);
    border-color: var(--color-accent-gold);
}

.relive-btn {
    background-color: #e74c3c;
    border-color: #c0392b;
    color: white;
    animation: pulse 2s infinite;
}

.relive-btn:hover {
    background-color: #c0392b;
    color: white;
}

@keyframes pulse {
    0% { box-shadow: 0 0 0 0 rgba(231, 76, 60, 0.7); }
    70% { box-shadow: 0 0 0 10px rgba(231, 76, 60, 0); }
    100% { box-shadow: 0 0 0 0 rgba(231, 76, 60, 0); }
}

.action-btn {
    font-size: 0.8em;
    padding: 2px 6px;
    margin-right: 5px;
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-accent-gold);
    color: var(--color-text-light);
    border-radius: 3px;
    cursor: pointer;
}

.action-btn:hover {
    background-color: var(--color-accent-gold);
    color: var(--color-background-dark);
}

/* 聊天历史面板样式 - 固定高度 */
.chat-history-panel {
    width: 100%;
    margin-top: 0;
    background-color: var(--color-background-dark);
    border: 1px solid var(--color-border-subtle);
    border-radius: 6px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    flex: 0 0 140px; /* 固定高度，约4-5行 */
    min-height: 0;
    margin-bottom: 10px;
}

.chat-history-header {
    padding: 8px 10px;
    background-color: var(--color-background-secondary);
    color: var(--color-accent-gold);
    font-weight: bold;
    border-bottom: 1px solid var(--color-border-subtle);
}

.chat-history {
    /* remove fixed max-height so it can stretch; use flex to fill */ 
    flex: 1 1 auto;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 5px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-height: 0; /* 允许正确计算高度在 flex 容器内 */
}

/* 隐藏原生滚动条，但保留滚动功能 */
.chat-history::-webkit-scrollbar { width: 0; height: 0; }
.chat-history { -ms-overflow-style: none; scrollbar-width: none; }    

.chat-message {
    display: flex;
    align-items: flex-start;
    min-width: 0;
    font-size: 0.9em;
    line-height: 1.4;
}

.chat-sender {
    flex: 0 0 auto;
    max-width: 42%;
    color: var(--color-accent-blue);
    font-weight: bold;
    margin-right: 6px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.chat-text {
    flex: 1 1 auto;
    min-width: 0;
    color: var(--color-text-light);
    white-space: normal;
    overflow-wrap: anywhere;
    word-break: break-word;
}

.player-offline {
    color: #888; /* 变浅的颜色 */
}

.offline-tag {
    color: #ff4d4f; /* 醒目的断线提示色，或者也用浅色 */
    font-size: 0.8em;
    margin-left: 5px;
}
.map-entry {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 0;
}

/* Maps Panel Styles */
.maps-panel-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
}

.map-list-area {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    padding-right: 5px;
    /* Hide scrollbar */
    -ms-overflow-style: none;
    scrollbar-width: none;
}
.map-list-area::-webkit-scrollbar {
    display: none;
}

.map-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

.map-item {
    padding: 10px;
    border-bottom: 1px solid var(--color-border-subtle);
    cursor: pointer;
    transition: background-color 0.2s;
    color: var(--color-text-light);
}

.map-item:hover {
    background-color: rgba(255, 255, 255, 0.05);
}

.map-item.selected {
    background-color: rgba(255, 215, 0, 0.15);
    color: var(--color-accent-gold);
    border-left: 3px solid var(--color-accent-gold);
}

.map-name {
    font-weight: bold;
}

.map-description-area {
    flex: 0 0 auto; /* Fixed height based on content */
    border-top: 1px solid var(--color-accent-gold);
    padding: 10px;
    background-color: var(--color-background-secondary);
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.map-desc-text {
    font-size: 0.9em;
    color: var(--color-text-light);
    line-height: 1.4;
    min-height: 60px; /* Ensure some space for text */
    max-height: 100px;
    overflow-y: auto;
    overflow-x: hidden;
}

.enter-world-btn {
    width: 100%;
    padding: 8px;
    background-color: var(--color-accent-gold);
    color: var(--color-background-dark);
    border: none;
    border-radius: 4px;
    font-weight: bold;
    cursor: pointer;
    transition: opacity 0.2s;
}

.enter-world-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background-color: #555;
    color: #888;
}

/* Shop Panel Styles */
.shop-panel {
    /* 移除绝对定位，使其嵌入文档流 */
    /* position: absolute; */
    /* top: 0; */
    /* left: 0; */
    /* width: 220px; */
    /* z-index: 100; */
    
    flex: 1; /* 占据 room-entities 的空间 */
    background-color: var(--color-background-secondary); /* 与 room-entities 一致 */
    border: 1px solid var(--color-border-subtle); /* 与 room-entities 一致 */
    border-radius: 8px; /* 与 room-entities 一致 */
    
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.shop-header {
    padding: 10px;
    background-color: var(--color-background-secondary);
    border-bottom: 1px solid var(--color-accent-gold);
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.shop-header h3 {
    margin: 0;
    color: var(--color-accent-gold);
    font-size: 1em;
}

.shop-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 10px;
    overflow-y: auto;
    overflow-x: hidden;
}

.money-display {
    padding: 5px;
    margin-bottom: 10px;
    color: var(--color-accent-gold);
    font-weight: bold;
    border-bottom: 1px dashed var(--color-border-subtle);
    font-size: 0.9em;
}

.shop-items-list {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    /* Hide scrollbar */
    -ms-overflow-style: none;
    scrollbar-width: none;
}

.shop-items-list::-webkit-scrollbar {
    display: none;
}

.shop-item {
    padding: 4px 6px; /* Compact padding */
    border: 1px solid var(--color-border-subtle);
    margin-bottom: 2px; /* Compact margin */
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    justify-content: space-between;
    align-items: stretch;
    font-size: 0.9em;
}

.shop-item:hover {
    background-color: rgba(255, 255, 255, 0.05);
}

.item-selected-indicator {
    color: var(--color-accent-gold);
}

.detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 5px;
}

.detail-name {
    font-weight: bold;
    font-size: 1.1em;
}

.detail-sell-price {
    color: var(--color-accent-gold);
    font-size: 0.9em;
}

.item-actions-row {
    margin-top: 10px;
    display: flex;
    justify-content: flex-end;
}

.item-details {
    display: flex;
    flex-direction: column;
}

.item-name {
    color: var(--color-text-light);
    font-weight: bold;
}

.item-price {
    color: #f39c12;
    font-size: 0.85em;
}

/* Equipment Panel Styles */
.equipment-panel-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow-y: auto;
}

.equipment-slots {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.equipment-slot {
    display: flex;
    align-items: center;
    padding: 8px;
    background-color: rgba(255, 255, 255, 0.05);
    border-radius: 4px;
    border: 1px solid var(--color-border-subtle);
}

.slot-label {
    width: 60px;
    color: var(--color-text-light);
    font-weight: bold;
    font-size: 0.9em;
}

.slot-content {
    flex: 1;
    display: flex;
    justify-content: flex-end;
}

.equipped-item {
    font-weight: bold;
}

.empty-slot {
    color: #666;
    font-style: italic;
}

.detail-slot {
    margin-top: 5px;
    color: var(--color-accent-gold);
    font-size: 0.9em;
}

.buy-controls {
    margin-top: 10px;
    padding-top: 10px;
    border-top: 1px solid var(--color-border-subtle);
}

.selected-item-info {
    margin-bottom: 10px;
    display: flex;
    justify-content: space-between;
    gap: 8px;
    flex-wrap: wrap;
    font-size: 0.9em;
}

.amount-selector {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-wrap: wrap;
    gap: 4px;
    margin-bottom: 10px;
}

.amount-selector button {
    width: 36px;
    height: 30px;
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-light);
    cursor: pointer;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 0;
    font-size: 0.9em;
}

/* Skill Tabs */
.skill-tabs {
    display: flex;
    border-bottom: 1px solid var(--color-border-subtle);
    margin-bottom: 10px;
}

.skill-tab {
    flex: 1;
    padding: 8px 0;
    text-align: center;
    cursor: pointer;
    color: var(--color-text-light);
    background-color: transparent;
    border: none;
    border-bottom: 2px solid transparent;
    transition: all 0.2s;
}

.skill-tab.active {
    color: var(--color-accent-gold);
    border-bottom-color: var(--color-accent-gold);
    font-weight: bold;
}

.skill-tab:hover {
    background-color: rgba(255, 255, 255, 0.05);
}

.skills-panel-container {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.skills-list {
    flex: 1;
    overflow-y: auto;
    padding-right: 2px;
    -ms-overflow-style: none;
    scrollbar-width: none;
}

.skills-list::-webkit-scrollbar {
    display: none;
}

.skill-detail-area {
    flex: 0 0 auto;
    min-height: 100px;
    border-top: 1px solid var(--color-accent-gold);
    background-color: rgba(0, 0, 0, 0.2);
    padding: 10px;
    margin-top: 5px;
    display: flex;
    flex-direction: column;
    gap: 5px;
}

.skill-bonus-list {
    margin-top: 5px;
    font-size: 0.85em;
    color: #2ecc71; /* Green for bonuses */
}

.skill-bonus-item {
    margin-bottom: 2px;
}

/* Hide spinner */
.amount-selector input::-webkit-outer-spin-button,
.amount-selector input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.amount-selector input[type=number] {
  -moz-appearance: textfield;
  appearance: textfield;
}

.amount-selector input {
    width: 42px;
    text-align: center;
    background-color: var(--color-background-dark);
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-light);
    height: 28px;
    margin: 0;
}

.buy-btn {
    width: 100%;
    padding: 8px;
    background-color: var(--color-accent-gold);
    color: var(--color-background-dark);
    border: none;
    border-radius: 4px;
    font-weight: bold;
    cursor: pointer;
}

.buy-btn:hover {
    opacity: 0.9;
}

/* Money display in backpack */
.money-footer {
    margin-top: auto;
    padding-top: 10px;
    border-top: 1px solid var(--color-accent-gold);
    color: var(--color-accent-gold);
    font-weight: bold;
    text-align: center;
    font-size: 0.95em;
}

/* Backpack panel container to support sticky footer */
.backpack-panel-container {
    display: flex;
    flex-direction: column;
    height: 100%;
}

/* New Styles for Shop and Backpack Improvements */

/* Highlight selected items */
.shop-item.selected,
.item-entry.selected {
    background-color: rgba(255, 215, 0, 0.15);
    border-color: var(--color-accent-gold);
}

/* Compact buttons for shop amount selector */
.small-btn {
    font-size: 0.8em !important;
    padding: 0 5px !important;
    min-width: 25px;
}

/* Backpack Layout */
.money-header {
    padding: 8px;
    margin-bottom: 5px;
    background-color: rgba(0, 0, 0, 0.2);
    border-bottom: 1px solid var(--color-border-subtle);
    color: var(--color-accent-gold);
    font-weight: bold;
    text-align: center;
    font-size: 0.95em;
    flex: 0 0 auto;
}

.backpack-list {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    padding-right: 2px;
    /* Hide scrollbar */
    -ms-overflow-style: none;
    scrollbar-width: none;
}

.backpack-list::-webkit-scrollbar {
    display: none;
}

.backpack-list ul {
    list-style: none;
    padding: 0;
    margin: 0;
}

.item-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
}

.item-entry {
    cursor: pointer;
    padding: 8px 5px;
    transition: background-color 0.2s;
}

.item-entry:hover {
    background-color: rgba(255, 255, 255, 0.05);
}

.item-actions {
    display: flex;
    justify-content: flex-end;
    flex-wrap: wrap;
    margin-top: 5px;
    gap: 5px;
    padding-top: 5px;
    border-top: 1px dashed rgba(255, 255, 255, 0.1);
}

.discard-btn {
    border-color: #e74c3c;
    color: #e74c3c;
}

.discard-btn:hover {
    background-color: #e74c3c;
    color: white;
}

.backpack-bottom-area {
    flex: 0 0 auto;
    min-height: 80px;
    border-top: 1px solid var(--color-accent-gold);
    background-color: rgba(0, 0, 0, 0.2);
    padding: 10px;
    margin-top: 5px;
}

.item-details-view {
    display: flex;
    flex-direction: column;
    gap: 5px;
}

.detail-name {
    color: var(--color-accent-gold);
    font-weight: bold;
    font-size: 1.1em;
}

.detail-desc {
    color: var(--color-text-light);
    font-size: 0.9em;
    line-height: 1.4;
}

.discard-prompt {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
}

.discard-info {
    color: #e74c3c;
    font-weight: bold;
}

.discard-actions {
    display: flex;
    gap: 10px;
    width: 100%;
    justify-content: center;
}

.confirm-btn {
    background-color: #e74c3c;
    color: white;
    border: none;
    padding: 5px 15px;
    border-radius: 4px;
    cursor: pointer;
}

.cancel-btn {
    background-color: transparent;
    border: 1px solid var(--color-text-light);
    color: var(--color-text-light);
    padding: 5px 15px;
    border-radius: 4px;
    cursor: pointer;
}

/* 新增：物品数量和库存信息样式 */
.item-row-content {
    display: flex;
    justify-content: space-between;
    width: 100%;
}

.item-count {
    color: var(--color-text-light);
    font-size: 0.9em;
}

.stock-info {
    text-align: center;
    color: var(--color-text-light);
    font-size: 0.9em;
    margin: 5px 0;
}

.sell-price-info, .total-price-info {
    text-align: center;
    color: var(--color-accent-gold);
    margin: 5px 0;
}

/* Equipment Panel Styles */
.equipment-panel-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
}

.equipment-slots {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    padding-bottom: 10px;
    /* Hide scrollbar */
    -ms-overflow-style: none;
    scrollbar-width: none;
}

.equipment-slots::-webkit-scrollbar {
    display: none;
}

.equipment-slot-wrapper {
    display: flex;
    flex-direction: column;
    border-bottom: 1px solid var(--color-border-subtle);
}

.slot-actions {
    padding: 5px 10px;
    background-color: rgba(0, 0, 0, 0.2);
    display: flex;
    justify-content: flex-end;
}

.unequip-btn {
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-accent-gold);
    color: var(--color-text-light);
    padding: 2px 8px;
    border-radius: 3px;
    cursor: pointer;
    font-size: 0.85em;
}

.unequip-btn:hover {
    background-color: var(--color-accent-gold);
    color: var(--color-background-dark);
}

/* Ensure backpack-bottom-area is fixed at bottom */
.backpack-bottom-area {
    flex: 0 0 auto;
    border-top: 1px solid var(--color-accent-gold);
    background-color: var(--color-background-dark);
    z-index: 10;
}
</style>
/* Skills Panel Styles */
.skills-panel-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    gap: 0; /* Remove gap as we use padding/borders */
}

.skill-tabs {
    display: flex;
    justify-content: center; /* Center tabs */
    gap: 20px;
    padding: 10px 0;
    border-bottom: 1px solid var(--color-border-subtle);
    flex: 0 0 auto;
}

.skill-tabs span {
    cursor: pointer;
    color: var(--color-text-light);
    font-size: 1em;
    padding-bottom: 2px;
    transition: all 0.2s;
}

.skill-tabs span:hover {
    color: var(--color-accent-gold);
}

.skill-tabs span.active {
    color: var(--color-accent-gold);
    border-bottom: 2px solid var(--color-accent-gold);
    font-weight: bold;
}

.tab-separator {
    color: var(--color-border-subtle);
    cursor: default !important;
}

.skills-list-container {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    /* Hide scrollbar */
    -ms-overflow-style: none;
    scrollbar-width: none;
}

.skills-list-container::-webkit-scrollbar {
    display: none;
}

.skills-list {
    list-style: none;
    padding: 0;
    margin: 0;
}

.skill-item {
    display: flex;
    flex-direction: column;
    padding: 8px 5px;
    cursor: pointer;
    border-bottom: 1px solid var(--color-border-subtle);
    transition: background-color 0.2s;
}

.skill-item:hover {
    background-color: rgba(255, 255, 255, 0.05);
}

.skill-item.selected {
    background-color: rgba(255, 215, 0, 0.15);
    border-left: 3px solid var(--color-accent-gold);
}

.skill-row-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
}

.skill-name {
    font-weight: bold;
}

.skill-level {
    color: var(--color-accent-gold);
    font-size: 0.9em;
}

.skill-details-view {
    display: flex;
    flex-direction: column;
    gap: 5px;
}

.detail-bonuses {
    margin-top: 10px;
    background-color: rgba(0, 0, 0, 0.2);
    padding: 8px;
    border-radius: 4px;
}

.bonus-title {
    color: #f1c40f;
    margin-bottom: 5px;
    font-weight: bold;
    font-size: 0.9em;
}

.bonus-item {
    color: #2ecc71;
    font-size: 0.85em;
    margin-bottom: 2px;
}
