<template>
  <div class="quest-panel-container">
    <div class="quest-filters">
      <button v-for="tab in tabs" :key="tab.id" :class="{ active: selectedTab === tab.id }" @click="selectedTab = tab.id">{{ tab.label }} {{ countFor(tab.id) }}</button>
    </div>
    <div v-if="visibleQuests.length" class="quest-list">
      <div v-for="quest in visibleQuests" :key="quest.id" class="quest-entry">
        <div class="quest-title-row">
          <span class="quest-name">{{ quest.name }}</span>
          <span class="quest-status">{{ quest.status === 'completed' ? '已完成' : quest.canTurnIn ? '可交付' : quest.status === 'active' ? `${quest.progress}%` : quest.locked ? '未解锁' : '可接取' }}</span>
        </div>
        <div class="quest-desc">{{ quest.description }}</div>
        <div class="quest-location">接取：{{ quest.giverLocation }}</div>
        <div class="quest-location">交付：{{ quest.turnInLocation }}</div>
        <div v-if="quest.lockHint" class="quest-lock">{{ quest.lockHint }}</div>
        <div
          v-for="objective in quest.objectives || []"
          :key="objective.targetId || objective.description"
          class="quest-objective">
          {{ objective.completed ? '✓' : '○' }} {{ objective.description }}
        </div>
      </div>
    </div>
    <p v-else class="empty-msg">此分类暂无任务。可切换查看其他委托。</p>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
const props = defineProps({
  quests: { type: Array, default: () => [] }
})
const selectedTab = ref('available')
const tabs = [{ id: 'available', label: '可接取' }, { id: 'active', label: '进行中' }, { id: 'locked', label: '未解锁' }, { id: 'completed', label: '已完成' }]
const inCategory = (quest, category) => category === 'available' ? quest.status === 'not_started' && !quest.locked : category === 'locked' ? quest.status === 'not_started' && quest.locked : quest.status === category
const visibleQuests = computed(() => props.quests.filter(quest => inCategory(quest, selectedTab.value)))
const countFor = (category) => props.quests.filter(quest => inCategory(quest, category)).length
</script>

<style scoped>
.quest-filters { display: flex; flex-wrap: wrap; gap: 5px; }
.quest-filters button { background: #171717; border: 1px solid #444; color: #aaa; padding: 4px 6px; font-size: 11px; border-radius: 3px; }
.quest-filters button.active { color: #f1c40f; border-color: #ac8122; }
.quest-location { color: #bbb; font-size: 11px; line-height: 1.5; margin-top: 4px; overflow-wrap: anywhere; }
.quest-lock { color: #aaa; font-size: 11px; margin-top: 4px; }
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
