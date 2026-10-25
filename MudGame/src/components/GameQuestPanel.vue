<template>
  <div class="quest-panel-container">
    <div v-if="quests && quests.length > 0" class="quest-list">
      <div v-for="quest in quests" :key="quest.id" class="quest-entry">
        <div class="quest-title-row">
          <span class="quest-name">{{ quest.name }}</span>
          <span class="quest-status">{{ quest.status === 'completed' ? '已完成' : '进行中' }}</span>
        </div>
        <div class="quest-desc">{{ quest.description }}</div>
        <div
          v-for="objective in quest.objectives || []"
          :key="objective.targetId || objective.description"
          class="quest-objective">
          {{ objective.description }}
        </div>
      </div>
    </div>
    <p v-else class="empty-msg">暂无任务。</p>
  </div>
</template>

<script setup>
defineProps({
  quests: { type: Array, default: () => [] }
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
.quest-objective { color: #ecf0f1; font-size: 11px; margin-top: 5px; padding-left: 8px; border-left: 2px solid #444; }
.empty-msg { color: #666; text-align: center; padding: 15px; font-style: italic; font-size: 12px; }
</style>
