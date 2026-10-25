<template>
  <div class="side-toolbar">
    <button v-for="tool in tools" :key="tool.key" @click="$emit('toggle', tool.key)" class="tool-btn" :title="tool.title">{{ tool.label }}</button>
  </div>
</template>

<script setup>
defineEmits(['toggle'])

const tools = [
  { key: 'status', label: '属', title: '属性' },
  { key: 'backpack', label: '囊', title: '背包' },
  { key: 'skills', label: '技', title: '技能' },
  { key: 'equipment', label: '装', title: '装备' },
  { key: 'maps', label: '图', title: '地图' },
  { key: 'dungeons', label: '境', title: '副本' },
  { key: 'quests', label: '令', title: '任务' },
  { key: 'factions', label: '势', title: '势力' }
]
</script>

<style scoped>
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
</style>
