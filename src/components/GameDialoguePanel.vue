<template>
  <div class="dialogue-panel">
    <div class="dialogue-header">
      <div>
        <h3>{{ dialogue?.npcName || '交谈' }}</h3>
        <span>{{ dialogue?.title || '' }}</span>
      </div>
      <button @click="$emit('close')" class="dialogue-close-btn">×</button>
    </div>
    <div class="dialogue-body">
      <p class="dialogue-opening">{{ dialogue?.opening }}</p>
      <div class="dialogue-options">
        <button
          v-for="option in options"
          :key="option.id"
          @click="$emit('choose', option.id)"
          class="dialogue-option-btn">
          {{ option.text }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  dialogue: Object
})

defineEmits(['close', 'choose'])

const options = computed(() => {
  return Array.isArray(props.dialogue?.options) ? props.dialogue.options : []
})
</script>

<style scoped>
.dialogue-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    overflow: hidden;
    background-color: var(--color-background-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 6px;
}

.dialogue-header {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    padding: 10px;
    border-bottom: 1px solid var(--color-accent-gold);
}

.dialogue-header h3 {
    margin: 0;
    color: var(--color-accent-gold);
    font-size: 1em;
}

.dialogue-header span {
    color: var(--color-text-light);
    font-size: 0.85em;
    opacity: 0.8;
}

.dialogue-close-btn {
    flex: 0 0 auto;
    width: 24px;
    height: 24px;
    border: 1px solid var(--color-border-subtle);
    border-radius: 3px;
    background: var(--color-background-dark);
    color: var(--color-text-light);
    cursor: pointer;
}

.dialogue-body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 12px;
    -ms-overflow-style: none;
    scrollbar-width: none;
}

.dialogue-body::-webkit-scrollbar {
    display: none;
}

.dialogue-opening {
    margin: 0 0 12px;
    color: var(--color-text-light);
    line-height: 1.6;
    overflow-wrap: anywhere;
}

.dialogue-options {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.dialogue-option-btn {
    width: 100%;
    text-align: left;
    padding: 8px 10px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid var(--color-border-subtle);
    border-radius: 4px;
    color: var(--color-text-light);
    cursor: pointer;
    font-family: inherit;
    line-height: 1.4;
    overflow-wrap: anywhere;
}

.dialogue-option-btn:hover {
    border-color: var(--color-accent-gold);
    color: var(--color-accent-gold);
}
</style>
