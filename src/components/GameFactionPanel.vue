<template>
  <div class="quest-panel-container">
    <div v-if="factions && factions.length > 0" class="quest-list">
      <div v-for="faction in factions" :key="faction.id" class="quest-entry">
        <div class="quest-title-row">
          <span class="quest-name">{{ faction.name }}</span>
          <span class="quest-status">声望 {{ faction.reputation || 0 }}</span>
        </div>
        <div class="quest-desc">{{ faction.description }}</div>
      </div>
    </div>
    <p v-else class="empty-msg">暂无势力记录。</p>
  </div>
</template>

<script setup>
defineProps({
  factions: { type: Array, default: () => [] }
})
</script>

<style scoped>
.quest-panel-container { display: flex; flex-direction: column; gap: 8px; }
.quest-list { display: flex; flex-direction: column; gap: 8px; }
.quest-entry { padding: 8px; border-bottom: 1px solid #2a2a2a; background: rgba(0,0,0,0.18); border-radius: 4px; }
.quest-title-row { display: flex; justify-content: space-between; gap: 8px; align-items: center; }
.quest-name { color: #f1c40f; font-weight: bold; font-size: 12px; overflow-wrap: anywhere; }
.quest-status { color: #2ecc71; font-size: 11px; white-space: nowrap; }
.quest-desc { color: #b0b0b0; font-size: 11px; margin-top: 4px; line-height: 1.4; }
.empty-msg { color: #666; text-align: center; padding: 15px; font-style: italic; font-size: 12px; }
</style>
