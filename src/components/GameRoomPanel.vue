<template>
  <div class="room-entities-comp">
    <div class="room-entities-section" v-if="entities?.length">
      <div v-for="entity in visibleEntities" :key="entity.id" class="entity-row"
        :class="{ 'entity-highlight': entity.isNpc && entity.hostile, 'entity-interactable': entity === selectedEntity }"
        @click="$emit('selectEntity', entity)">
        <span class="entity-icon">{{ entity.isNpc ? '👤' : entity.isPlayer ? '🧑' : '📦' }}</span>
        <span class="entity-name">{{ entity.name }}</span>
        <span v-if="entity.level" class="entity-level">Lv.{{ entity.level }}</span>
        <span v-if="entity.hp" class="entity-hp">HP:{{ entity.hp }}/{{ entity.maxHp }}</span>
        <span v-if="entity.isNpc && entity.hostile" class="entity-hostile">⚔敌对</span>
      </div>
    </div>

    <div class="room-entities-section" v-else-if="entities === null">
      <div class="entity-row hint-row">加载中...</div>
    </div>
    <div class="room-entities-section" v-else>
      <div class="entity-row hint-row">这里空无一人</div>
    </div>

    <div v-if="selectedEntity" class="entity-actions">
      <button @click="$emit('viewEntity', selectedEntity)" class="action-btn view-btn">查看</button>
      <button @click="$emit('sparEntity', selectedEntity)" class="action-btn spar-btn">比试</button>
      <button @click="$emit('killEntity', selectedEntity)" class="action-btn kill-btn">击杀</button>
      <button v-if="selectedEntity.isShop" @click="$emit('tradeEntity', selectedEntity)" class="action-btn trade-btn">交易</button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  entities: Array,
  selectedEntity: Object
})

defineEmits(['selectEntity', 'viewEntity', 'sparEntity', 'killEntity', 'tradeEntity'])

const visibleEntities = computed(() => {
  if (!props.entities) return []
  return props.entities.filter(e => e.isNpc || e.isItem || e.isPlayer)
})
</script>

<style scoped>
.room-entities-comp { height: 100%; }
.room-entities-section { }
.entity-row { display: flex; align-items: center; gap: 8px; padding: 6px 8px; cursor: pointer; border-bottom: 1px solid #2a2a2a; transition: background 0.15s; border-radius: 3px; }
.entity-row:hover { background: rgba(255,255,255,0.04); }
.entity-highlight { border-left: 3px solid #e74c3c; }
.entity-interactable { background: rgba(52,152,219,0.1); border-left: 3px solid #3498db; }
.entity-icon { font-size: 14px; }
.entity-name { font-weight: bold; flex: 1; }
.entity-level { color: #f1c40f; font-size: 12px; }
.entity-hp { color: #2ecc71; font-size: 12px; }
.entity-hostile { color: #e74c3c; font-size: 11px; font-weight: bold; }
.hint-row { color: #666; cursor: default; font-style: italic; }

.entity-actions {
  display: flex;
  gap: 6px;
  padding: 8px;
  flex-wrap: wrap;
  border-top: 1px solid #2a2a2a;
  margin-top: 4px;
}

.action-btn {
  padding: 5px 12px;
  border: 1px solid #444;
  background: #1e1e1e;
  color: #ccc;
  cursor: pointer;
  border-radius: 3px;
  font-size: 12px;
  transition: all 0.15s;
}

.action-btn:hover {
  background: #333;
  color: #fff;
}

.view-btn { border-color: #3498db; color: #3498db; }
.view-btn:hover { background: #3498db; color: #fff; }

.spar-btn { border-color: #f1c40f; color: #f1c40f; }
.spar-btn:hover { background: #f1c40f; color: #000; }

.kill-btn { border-color: #e74c3c; color: #e74c3c; }
.kill-btn:hover { background: #e74c3c; color: #fff; }

.trade-btn { border-color: #2ecc71; color: #2ecc71; }
.trade-btn:hover { background: #2ecc71; color: #000; }
</style>
