<template>
  <div class="log-content-wrapper">
    <!-- 聊天历史（游戏日志上方） -->
    <div class="chat-history-panel">
      <div class="chat-history-header">聊天</div>
      <div class="chat-history" ref="chatHistoryEl">
        <div v-for="(c, idx) in chatMessages" :key="idx" class="chat-message">
          <span class="chat-sender">{{ c.sender }}:</span>
          <span class="chat-text">{{ c.message }}</span>
        </div>
      </div>
    </div>

    <div class="game-log" ref="gameLogEl">
      <div v-for="(log, index) in logs" :key="index" :class="getLogClass(log)">
        <template v-if="isDungeonLeaveButtonLog(log)">
          <button v-if="isActiveLeaveButtonLog(log)" @click="$emit('confirmLeave')">{{ getLogText(log) }}</button>
          <template v-else>{{ getLogText(log) }}</template>
        </template>
        <template v-else-if="isDungeonChoiceLog(log)">
          <span class="dungeon-choice-buttons">
            <button v-for="choice in log.choices" :key="choice.id" :disabled="log.used" @click="$emit('choose', log, choice.id)">{{ choice.label }}</button>
          </span>
        </template>
        <template v-else-if="log?.type === 'dialogue_choice'">
          <span class="dialogue-choice-buttons">
            <button v-for="option in log.options" :key="option.id" :disabled="log.used || option.available === false" @click="$emit('chooseDialogue', log, option)">{{ option.text }}</button>
          </span>
        </template>
        <template v-else>
          {{ getLogText(log) }}
        </template>
      </div>
    </div>

    <div class="chat-input">
      <input v-model="command" @keyup.enter="$emit('send')" placeholder="聊天..." />
      <button @click="$emit('send')">发送</button>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { DUNGEON_LEAVE_BUTTON_LOG_TYPE } from '../composables/useGameLog.js'

const command = defineModel('command', { type: String, default: '' })

defineEmits(['send', 'choose', 'confirmLeave', 'chooseDialogue'])

const props = defineProps({
  logs: { type: Array, default: () => [] },
  chatMessages: { type: Array, default: () => [] },
  activeLeaveToken: { type: String, default: '' }
})

const chatHistoryEl = ref(null)
const gameLogEl = ref(null)

const getLogText = (log) => {
  if (typeof log === 'string') return log
  if (log && typeof log === 'object' && typeof log.text === 'string') return log.text
  return ''
}

const isDungeonLeaveButtonLog = (log) => {
  return !!(log && typeof log === 'object' && log.type === DUNGEON_LEAVE_BUTTON_LOG_TYPE)
}

const isDungeonChoiceLog = (log) => !!(log && typeof log === 'object' && log.type === 'dungeon_choice')

const isActiveLeaveButtonLog = (log) => {
  if (!isDungeonLeaveButtonLog(log)) return false
  return !!props.activeLeaveToken && props.activeLeaveToken === log.token
}

const getLogClass = (log) => {
  const text = getLogText(log)
  if (text.startsWith('ERROR:')) return 'log-error'
  if (text.startsWith('WARN:')) return 'log-warn'
  if (text.startsWith('CHAT:')) return 'log-chat'
  return 'log-info'
}

// 新消息到达时自动滚到底部
watch(() => props.chatMessages.length, () => {
  const el = chatHistoryEl.value
  if (el) el.scrollTop = el.scrollHeight
}, { flush: 'post' })

watch(() => props.logs.length, () => {
  const el = gameLogEl.value
  if (el) el.scrollTop = el.scrollHeight
}, { flush: 'post' })
</script>

<style scoped>
.dialogue-choice-buttons { display: flex; flex-wrap: wrap; gap: 5px; }
.dialogue-choice-buttons button { max-width: 100%; white-space: normal; text-align: left; padding: 5px 8px; background: #1b1b1b; color: #e5c46e; border: 1px solid #6a562b; border-radius: 3px; font-size: 12px; }
.dialogue-choice-buttons button:disabled { color: #777; border-color: #333; cursor: default; }
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

.dungeon-choice-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 4px;
}

.dungeon-choice-buttons button {
    border: 1px solid #8d6a23;
    background: #17140d;
    color: #e5c568;
    cursor: pointer;
}

.dungeon-choice-buttons button:disabled {
    color: #777;
    border-color: #444;
    cursor: default;
}

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
</style>
