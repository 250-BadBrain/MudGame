<template>
  <div class="action-panel">
    <div class="action-buttons-grid">
      <button @click="$emit('action', 'meditate')" class="action-cmd-btn">打坐</button>
      <button @click="$emit('action', 'heal')" class="action-cmd-btn">疗伤</button>
      <button @click="$emit('action', 'stop')" class="action-cmd-btn">停止</button>
      <button
        v-for="action in roomActions"
        :key="action.id"
        @click="$emit('roomAction', action.id)"
        class="action-cmd-btn">
        {{ action.label }}
      </button>
      <button v-if="isInDungeon" @click="$emit('leaveDungeon')" class="action-cmd-btn">离开副本</button>
      <button v-if="playerState === 'DEAD'" @click="$emit('action', 'relive')" class="action-cmd-btn relive-btn">复活</button>
    </div>
  </div>
</template>

<script setup>
defineProps({
  roomActions: { type: Array, default: () => [] },
  isInDungeon: { type: Boolean, default: false },
  playerState: { type: String, default: 'IDLE' },
})

defineEmits(['action', 'roomAction', 'leaveDungeon'])
</script>

<style scoped>
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
</style>
