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
          buyButtonText="请教入门 / 修炼"
          @close="closeLearn"
          @selectItem="selectLearnSkill"
          @buy="startLearning" />

        <div v-else class="room-entities-area">
          <GameRoomPanel
            :entities="combinedEntities" :safeZone="!!room?.safeZone"
            :selectedEntity="selectedEntityObj"
            @selectEntity="handleEntitySelect"
            @viewEntity="viewEntity"
            @talkEntity="talkEntity"
            @sparEntity="sparEntity"
            @killEntity="killEntity"
            @learnEntity="openLearn"
            @tradeEntity="openShop" />
        </div>

        <GameExitsGrid :room="room" @go="go" />
      </div>

      <div class="right-log-area">
        <GameLogPanel
          v-model:command="command"
          :logs="logs"
          :chat-messages="chatMessages"
          :active-leave-token="pendingDungeonLeave?.token || ''"
          @send="sendCommand"
          @choose="chooseDungeonAction"
          @choose-dialogue="chooseDialogueAction"
          @confirm-leave="confirmLeaveCurrentDungeon" />

        <!-- 右侧面板区域（包含信息面板） -->
        <div class="right-panels-wrapper">
            <GameInfoPanel
              :visible="showInfoPanel"
              :title="panelTitle"
              :currentPanel="currentPanel"
              :backpackItems="backpackItems"
              :selectedBackpackItem="selectedBackpackItem"
              :viewingItem="viewingItem"
              :enhancingItem="enhancingItem"
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
              @useItem="useBackpackItem"
              @sellItem="initiateSell"
              @enhanceItem="initiateEnhance"
              @confirmEnhance="confirmEnhance"
              @cancelEnhance="cancelEnhance"
              @changeDiscardAmount="changeDiscardAmount"
              @confirmDiscard="confirmDiscard"
              @cancelDiscard="cancelDiscard"
              @changeSellAmount="changeSellAmount"
              @confirmSell="confirmSell"
              @cancelSell="cancelSell"
              @changeSkillTab="(tab) => skillTab = tab"
              @selectSkill="selectSkill"
              @viewSkill="viewSkill"
              @equipSkill="equipSkill"
              @practiceSkill="practiceSkill"
              @unequipSkill="unequipSkill"
              @selectEquipSlot="selectEquipmentSlot"
              @unequipSlot="unequipItem"
              @selectWorld="selectWorld"
              @enterWorld="enterWorld"
              @selectDungeon="selectDungeon"
              @enterDungeon="enterDungeon" :quickCompleting="quickCompleting" :canQuickCompleteSelected="canQuickCompleteSelected" :quickCompleteStatusHint="quickCompleteStatusHint" @quickCompleteDungeon="quickCompleteDungeon" />

            <GameActionPanel
              :room-actions="roomActions"
              :is-in-dungeon="isInDungeon"
              :player-state="currentPlayer.state"
              @action="doAction"
              @room-action="doRoomAction"
              @leave-dungeon="leaveCurrentDungeon" />
        </div>

        <GameSideToolbar @toggle="onToolbarToggle" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { usePlayerStore } from '@/stores/player'
import { initWebSocket, removeMessageHandler, sendMessage } from '../utils/ws.js'
import GameShop from '../components/GameShop.vue'
import GameRoomPanel from '../components/GameRoomPanel.vue'
import GameInfoPanel from '../components/GameInfoPanel.vue'
import GameExitsGrid from '../components/GameExitsGrid.vue'
import GameLogPanel from '../components/GameLogPanel.vue'
import GameActionPanel from '../components/GameActionPanel.vue'
import GameSideToolbar from '../components/GameSideToolbar.vue'
import { useChatState } from '../composables/useChatState.js'
import { useCommandQueue } from '../composables/useCommandQueue.js'
import { useGameLog } from '../composables/useGameLog.js'
import { useEntityActions } from '../composables/useEntityActions.js'
import { useDungeonView } from '../composables/useDungeonView.js'
import { useInfoPanels } from '../composables/useInfoPanels.js'
import { useShopLearn } from '../composables/useShopLearn.js'
import { useBackpackActions } from '../composables/useBackpackActions.js'
import { useKeyNavigation } from '../composables/useKeyNavigation.js'
import { getMessage, describeCommandError } from '../game/messageText.js'

const router = useRouter()
const playerStore = usePlayerStore()

// 从 Pinia store 获取当前已选择的角色ID
const characterId = computed(() => playerStore.playerId)
const getPlayerId = () => characterId.value

// 由本视图直接维护、并被多个模块共享的基础状态
const room = ref(null)
const currentPlayer = ref({
  id: '',
  characterName: '未知玩家', // 用于前端显示
  userId: '',
  state: 'IDLE'
})
const backpackItems = ref([])
const playerMoney = ref(0)

// ------------------- 组合各功能模块 -------------------

const {
  logs,
  appendLogEntry,
  addLog,
  addDungeonLeaveButtonLog
} = useGameLog()

const {
  sendGameCommand,
  dispose: disposeCommandQueue
} = useCommandQueue({ addLog })

const {
  selectedEntityKey,
  combinedEntities,
  selectedEntityObj,
  handleEntitySelect,
  checkCanAct,
  doAction,
  doRoomAction,
  viewEntity,
  talkEntity,
  killEntity,
  sparEntity
} = useEntityActions({ room, currentPlayer, addLog, sendGameCommand, getPlayerId })

const {
  equipmentSlots,
  equipment,
  selectedEquipmentSlot,
  selectedBackpackItem,
  viewingItem,
  discardingItem,
  discardAmount,
  sellingItem,
  enhancingItem, initiateEnhance, confirmEnhance, cancelEnhance,
  sellAmount,
  selectEquipmentSlot,
  equipItem,
  useBackpackItem,
  unequipItem,
  selectBackpackItem,
  viewBackpackItem,
  initiateDiscard,
  initiateSell,
  changeDiscardAmount,
  confirmDiscard,
  changeSellAmount,
  maxSellCount,
  maxDiscardCount,
  discardShowAmount,
  confirmSell,
  cancelSell,
  cancelDiscard,
  resetSelection: resetBackpackSelection
} = useBackpackActions({ backpackItems, sendGameCommand, getPlayerId, checkCanAct })

const panels = useInfoPanels({ room, sendGameCommand, getPlayerId, checkCanAct })
const {
  showInfoPanel,
  currentPanel,
  panelTitle,
  skillsList,
  statusAttributes,
  worldsList,
  questsList,
  factionsList,
  selectedWorldId,
  selectedSkill,
  viewingSkill,
  skillTab,
  consumeSilentRefresh,
  resetPanelRequests,
  sendSilentRefresh,
  filteredSkills,
  selectSkill,
  viewSkill,
  sortedWorldsList,
  selectedWorld,
  selectWorld,
  enterWorld,
  toggleBackpack,
  toggleSkills,
  toggleEquipment,
  toggleStatus,
  toggleMaps,
  toggleDungeons,
  toggleQuests,
  toggleFactions,
  refreshOpenPanels
} = panels

// 关闭面板时同时清理背包相关的选择/操作状态
const closeInfoPanel = () => {
  panels.closeInfoPanel()
  resetBackpackSelection()
}

const {
  dungeonsList,
  selectedDungeonId,
  pendingDungeonLeave,
  sortedDungeonsList,
  selectedDungeon,
  isInDungeon,
  syncPendingDungeonLeavePrompt,
  syncDungeonSelectionFromRoom,
  applyRoomDungeonUpdate,
  selectDungeon,
  enterDungeon,
  canQuickCompleteSelected, quickCompleteStatusHint, quickCompleting, quickCompleteDungeon, finishQuickComplete, disposeDungeonView,
  leaveCurrentDungeon,
  confirmLeaveCurrentDungeon
} = useDungeonView({ room, addLog, addDungeonLeaveButtonLog, sendGameCommand, checkCanAct, getPlayerId,
  getPlayerState: () => currentPlayer.value.state,
  refreshAfterQuickComplete: () => { for (const subtype of ["get_dungeons", "get_attributes", "get_backpack", "get_quests", "get_factions"]) sendSilentRefresh(subtype) }
})

const {
  showShopPanel,
  shopItems,
  currentShopNpc,
  selectedShopItem,
  buyAmount,
  showLearnPanel,
  learnSkills,
  selectedLearnSkill,
  isMaxAmountReached,
  changeAmount,
  openShop,
  closeShop,
  openLearn,
  closeLearn,
  selectLearnSkill,
  startLearning,
  equipSkill,
  practiceSkill,
  unequipSkill,
  selectShopItem,
  buyItem
} = useShopLearn({ sendGameCommand, getPlayerId, checkCanAct, closeInfoPanel })

const {
  chatMessages,
  command,
  addChatMessage,
  sendChatCommand
} = useChatState({
  sendGameCommand,
  getPlayerId: () => characterId.value,
  addLog: (message) => addLog(message)
})

// ------------------- 视图内辅助 -------------------

const keepCurrentPlayerOnlineInRoom = () => {
  if (!room.value || !room.value.playersInRoom || !currentPlayer.value.id) return
  room.value.playersInRoom = room.value.playersInRoom.map(p => (
    p.id === currentPlayer.value.id ? { ...p, online: true } : p
  ))
}

const roomActionLabels = {
  gather: '采集',
  work: '劳作',
  listen_rumor: '听传闻',
  search: '搜索',
};

const roomActions = computed(() => {
  const actions = Array.isArray(room.value?.actions) ? room.value.actions : [];
  const labels = room.value?.actionLabels || {};
  return actions.map(id => ({ id, label: labels[id] || roomActionLabels[id] || id }));
});

const requestCoreGameState = () => {
  sendGameCommand("system", "get_room", characterId.value, {})
  sendSilentRefresh("get_attributes")
}

const go = (direction) => {
  if (!checkCanAct('go')) return
  sendGameCommand("command", "go", characterId.value, { direction: direction })
}

const sendCommand = sendChatCommand

const chooseDungeonAction = (log, actionId) => {
  if (!log || log.used) return;
  log.used = true;
  doRoomAction(actionId);
}

const appendDialogueChoices = (dialogue) => {
  if (!Array.isArray(dialogue?.options) || !dialogue.options.length) return
  for (const log of logs.value) if (log?.type === 'dialogue_choice') log.used = true
  appendLogEntry({ type: 'dialogue_choice', npcId: dialogue.npcId, options: dialogue.options, used: false })
}
const chooseDialogueAction = (log, option) => {
  if (log.used || option.available === false || !checkCanAct('talk')) return
  log.used = true
  sendGameCommand('command', 'choose_dialogue', characterId.value, { npcId: log.npcId, optionId: option.id })
}

const { handleGlobalKeydown } = useKeyNavigation({ onNavigate: go })

const toolbarToggles = {
  status: toggleStatus,
  backpack: toggleBackpack,
  skills: toggleSkills,
  equipment: toggleEquipment,
  maps: toggleMaps,
  dungeons: toggleDungeons,
  quests: toggleQuests,
  factions: toggleFactions
}

const onToolbarToggle = (key) => {
  toolbarToggles[key]?.()
}

const logout = () => {
  // 发送注销角色指令，通知后端广播离线
  sendGameCommand("system", "deselect_character", characterId.value, {});

  // 仅清除已选角色（保持登录状态），并跳转到角色选择界面
  // 不调用 clearAuth()，只清除 playerId
  playerStore.setPlayerId(null);
  router.push({ name: 'CharacterSelect' });
};

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
  disposeDungeonView()
    disposeCommandQueue()
    removeMessageHandler(handleMessage)
    window.removeEventListener('keydown', handleGlobalKeydown)
})

// ------------------- WebSocket 消息处理 -------------------

const handleMessage = (msg) => {
  const silentRefresh = msg.type === 'command' ? consumeSilentRefresh(msg.subtype) : true;
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
          // 用户主动查看时才打开/切换到背包，内部刷新不弹面板
          if (!silentRefresh && (!showInfoPanel.value || currentPanel.value !== 'backpack')) {
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
          if (selectedSkill.value) selectedSkill.value = skillsList.value.find(s => s.id === selectedSkill.value.id) || null;
          if (viewingSkill.value) viewingSkill.value = skillsList.value.find(s => s.id === viewingSkill.value.id) || null;
          if (!silentRefresh && (!showInfoPanel.value || currentPanel.value !== 'skills')) {
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
          if (!silentRefresh && (!showInfoPanel.value || currentPanel.value !== 'status')) {
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
          if (!silentRefresh && (!showInfoPanel.value || currentPanel.value !== 'equipment')) {
              showInfoPanel.value = true;
              currentPanel.value = 'equipment';
              panelTitle.value = '装备';
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你一时无法查看随身装备。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'equip') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          sendSilentRefresh("get_backpack");
          sendSilentRefresh("get_equipment");
      } else {
          addLog(describeCommandError(msg.results.error, '这件物品现在无法装备。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'unequip') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          // Refresh both equipment and backpack
          sendSilentRefresh("get_backpack");
          sendSilentRefresh("get_equipment");
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
          if (!silentRefresh && (!showInfoPanel.value || currentPanel.value !== 'maps')) {
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
          if (!silentRefresh && (!showInfoPanel.value || currentPanel.value !== 'dungeons')) {
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
          keepCurrentPlayerOnlineInRoom();
          syncDungeonSelectionFromRoom();
          pendingDungeonLeave.value = null;
      } else if (!generatedMsg) {
          addLog(describeCommandError(msg.results.error, '你现在无法进入这处秘境。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'quick_complete_dungeon') {
      finishQuickComplete();
      if (msg.flag) (msg.logs || []).forEach(l => addLog(l));
      else addLog(describeCommandError(msg.results?.error, '当前无法快速完成副本。'));
  } else if (msg.type === 'command' && msg.subtype === 'leave_dungeon') {
      if (msg.flag) {
          room.value = msg.results.room;
          selectedEntityKey.value = null;
          (msg.logs || []).forEach(l => addLog(l));
          closeInfoPanel();
          closeShop();
          closeLearn();
          keepCurrentPlayerOnlineInRoom();
          syncDungeonSelectionFromRoom();
          pendingDungeonLeave.value = null;
          sendSilentRefresh("get_attributes");
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
              sendSilentRefresh("get_backpack");
          }
      } else {
          addLog(describeCommandError(msg.results.error, '这桩买卖没有谈成。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'discard_item') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          // Refresh backpack immediately
          sendSilentRefresh("get_backpack");
      } else {
          addLog(describeCommandError(msg.results.error, '这件物品现在无法丢弃。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'sell') {
      if (msg.flag) {
          (msg.logs || []).forEach(l => addLog(l));
          // Refresh backpack immediately
          sendSilentRefresh("get_backpack");
      } else {
          addLog(describeCommandError(msg.results.error, '对方没有收下这件物品。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'talk') {
      if (msg.flag) {
          appendDialogueChoices(msg.results?.dialogue);
          showShopPanel.value = false;
          showLearnPanel.value = false;
          (msg.logs || []).forEach(l => addLog(l));
          if (msg.results?.room) {
              room.value = msg.results.room;
              keepCurrentPlayerOnlineInRoom();
          }
          if (Array.isArray(msg.results?.choices) && msg.results.choices.length) {
              appendLogEntry({ type: 'dungeon_choice', text: '', choices: msg.results.choices, used: false });
          }
          sendSilentRefresh("get_quests");
          sendSilentRefresh("get_attributes");
          if (showInfoPanel.value && currentPanel.value === 'backpack') {
              sendSilentRefresh("get_backpack");
          }
      } else {
          addLog(describeCommandError(msg.results.error, '对方此刻无意交谈。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'choose_dialogue') {
      if (msg.flag) {
          appendDialogueChoices(msg.results?.dialogue);
          (msg.logs || []).forEach(l => addLog(l));
          if (msg.results?.room) {
              room.value = msg.results.room;
              keepCurrentPlayerOnlineInRoom();
          }
          if (Array.isArray(msg.results?.choices) && msg.results.choices.length) {
              appendLogEntry({ type: 'dungeon_choice', text: '', choices: msg.results.choices, used: false });
          }
          sendSilentRefresh("get_quests");
          sendSilentRefresh("get_attributes");
          if (showInfoPanel.value && currentPanel.value === 'factions') {
              sendSilentRefresh("get_factions");
          }
          if (showInfoPanel.value && currentPanel.value === 'backpack') {
              sendSilentRefresh("get_backpack");
          }
      } else {
          addLog(describeCommandError(msg.results.error, '对方没有回应这句话。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'get_quests') {
      if (msg.flag) {
          questsList.value = msg.results.quests || [];
          if (!silentRefresh && (!showInfoPanel.value || currentPanel.value !== 'quests')) {
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
          if (!silentRefresh && (!showInfoPanel.value || currentPanel.value !== 'factions')) {
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
            disposeDungeonView()
            resetPanelRequests()
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
        // 结构化同步副本进度（服务端在 room_update 中携带 dungeon 字段）
        if (msg.results?.dungeon) {
            applyRoomDungeonUpdate(msg.results.dungeon);
        }
        syncPendingDungeonLeavePrompt();
        if (showInfoPanel.value && currentPanel.value === 'skills') {
            sendSilentRefresh("get_skills");
        } else if (showInfoPanel.value && currentPanel.value === 'status') {
            sendSilentRefresh("get_attributes");
        }
    }
  } else if (msg.type === 'command' && msg.subtype === 'do_action') {
      if (msg.flag) {
          if (msg.results?.room) {
              room.value = msg.results.room;
              keepCurrentPlayerOnlineInRoom();
          }
          (msg.logs || []).forEach(l => addLog(l));
          if (Array.isArray(msg.results?.choices) && msg.results.choices.length) {
              appendLogEntry({ type: 'dungeon_choice', text: '', choices: msg.results.choices, used: false });
          }
          if (showInfoPanel.value && currentPanel.value === 'backpack') {
              sendSilentRefresh("get_backpack");
          }
          if (showInfoPanel.value && currentPanel.value === 'status') {
              sendSilentRefresh("get_attributes");
          }
      } else {
          addLog(describeCommandError(msg.results.error, '你现在无法做出这个动作。'));
      }
  } else if (msg.type === 'command' && ['practice', 'equip_skill', 'unequip_skill', 'learn', 'stop'].includes(msg.subtype)) {
      if (msg.flag) {
          if (msg.results?.message) addLog(msg.results.message);
          sendSilentRefresh('get_skills');
          sendSilentRefresh('get_attributes');
      } else {
          addLog(describeCommandError(msg.results.error, '当前无法修炼或调整武学。'));
      }
  } else if (msg.type === 'command' && msg.subtype === 'enhance_item') {
      if (msg.flag) {
          addLog(msg.results?.message);
          sendSilentRefresh('get_backpack'); sendSilentRefresh('get_equipment'); sendSilentRefresh('get_attributes');
      } else { addLog(describeCommandError(msg.results?.error, '当前无法强化这件装备。')); }
  } else if (msg.type === 'command' && msg.subtype === 'use_item') {
      if (msg.flag) {
          if (msg.results?.room) {
              room.value = msg.results.room;
              selectedEntityKey.value = null;
              keepCurrentPlayerOnlineInRoom(); syncDungeonSelectionFromRoom();
              pendingDungeonLeave.value = null;
              closeShop(); closeLearn();
              sendSilentRefresh('get_dungeons'); sendSilentRefresh('get_quests');
          }
          (msg.logs || []).forEach(l => addLog(l));
          if (msg.results && msg.results.message) addLog(msg.results.message);
          sendSilentRefresh("get_backpack");
          sendSilentRefresh("get_attributes");
      } else {
          addLog(describeCommandError(msg.results.error, '这个物品现在无法使用。'));
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
          if (msg.subtype !== 'kill' && msg.flag && msg.results && msg.results.message) addLog(msg.results.message); // spar returns message on success
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
    background-color: transparent;
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

















/* Entity Interaction Styles */












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



/* 隐藏滚动条 */








/* Exits Map Grid 样式 */





/* 新增布局样式 */

@keyframes slideIn {
    from { opacity: 0; transform: translateY(-10px); }
    to { opacity: 1; transform: translateY(0); }
}




/* 隐藏面板内容的滚动条 */







.game-info-panel {
    width: 100%;
    border: 1px solid var(--color-accent-gold);
    border-radius: 4px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: slideIn 0.3s ease;
    flex: 1;
    min-height: 0;
    box-sizing: border-box;
}

@keyframes slideIn {
    from { opacity: 0; transform: translateY(-10px); }
    to { opacity: 1; transform: translateY(0); }
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













/* 聊天历史面板样式 - 固定高度 */



/* 隐藏原生滚动条，但保留滚动功能 */

/* Maps Panel Styles */










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
















/* Equipment Panel Styles */











/* Skill Tabs */









/* Hide spinner */
.amount-selector input::-webkit-outer-spin-button,
.amount-selector input[type=number] {
  -moz-appearance: textfield;
  appearance: textfield;
}



/* Money display in backpack */
/* Backpack panel container to support sticky footer */
/* New Styles for Shop and Backpack Improvements */

/* Compact buttons for shop amount selector */
/* Backpack Layout */


















/* 新增：物品数量和库存信息样式 */



/* Equipment Panel Styles */






/* Ensure backpack-bottom-area is fixed at bottom */
</style>
