<template>
  <div class="room-entities-comp">
    <div class="room-entities-section" v-if="entities?.length">
      <div v-for="entity in visibleEntities" :key="entity.id" class="entity-block">
        <div class="entity-row"
          :class="{ 'entity-highlight': entity.isNpc && entity.hostile, 'entity-interactable': entity === selectedEntity }"
          @click="$emit('selectEntity', entity)">
          <span class="entity-icon">{{ entity.isNpc ? '客' : entity.isPlayer ? '人' : '物' }}</span>
          <span class="entity-main">
            <span class="entity-name">{{ entity.name }}</span>
            <span v-if="isOffline(entity)" class="entity-status entity-status-offline">&lt;断线&gt;</span>
            <span v-if="stateLabel(entity)" class="entity-status entity-status-active">&lt;{{ stateLabel(entity) }}&gt;</span>
          </span>
          <span v-if="entity.level" class="entity-level">Lv.{{ entity.level }}</span>
          <span v-if="!isCorpse(entity) && entity.hp !== undefined && entity.hp !== null" class="entity-hp">HP:{{ entity.hp }}/{{ entity.maxHp }}</span>
          <span v-if="!isCorpse(entity) && entity.isNpc && entity.hostile" class="entity-hostile">敌对</span>
        </div>
        <div v-if="entity === selectedEntity" class="entity-actions">
          <button @click.stop="$emit('viewEntity', entity)" class="action-btn view-btn">查看</button>
          <button v-if="!isCorpse(entity) && canTalk(entity)" @click.stop="$emit('talkEntity', entity)" class="action-btn talk-btn">交谈</button>
          <button v-if="!isCorpse(entity)" @click.stop="$emit('sparEntity', entity)" class="action-btn spar-btn">比试</button>
          <button v-if="!isCorpse(entity)" @click.stop="$emit('killEntity', entity)" class="action-btn kill-btn">击杀</button>
          <button v-if="!isCorpse(entity) && canTeach(entity)" @click.stop="$emit('learnEntity', entity)" class="action-btn learn-btn">请教</button>
          <button v-if="!isCorpse(entity) && entity.isShop" @click.stop="$emit('tradeEntity', entity)" class="action-btn trade-btn">交易</button>
        </div>
      </div>
    </div>

    <div class="room-entities-section" v-else-if="entities === null">
      <div class="entity-row hint-row">加载中...</div>
    </div>
    <div class="room-entities-section" v-else>
      <div class="entity-row hint-row">这里空无一人</div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  entities: Array,
  selectedEntity: Object
})

defineEmits(['selectEntity', 'viewEntity', 'talkEntity', 'sparEntity', 'killEntity', 'learnEntity', 'tradeEntity'])

const visibleEntities = computed(() => {
  if (!props.entities) return []
  return props.entities.filter(e => e.isNpc || e.isItem || e.isPlayer)
})

const canTeach = (entity) => {
  return !!(entity?.capabilities?.includes('teach') || Object.keys(entity?.skills || {}).length > 0)
}

const canTalk = (entity) => {
  return !!entity?.capabilities?.includes('talk')
}

const isCorpse = (entity) => entity?.isCorpse || entity?.state === 'DEAD'

const stateNames = {
  HEALING: '疗伤',
  MEDITATING: '打坐',
  MINING: '挖矿',
  COMBAT: '战斗',
  LEARNING: '学习',
  DEAD: '死亡'
}

const stateLabel = (entity) => {
  const state = entity?.state
  if (!state || state === 'IDLE' || state === 'DEAD') return ''
  return stateNames[state] || state
}

const isOffline = (entity) => {
  return !!(entity?.isPlayer && entity.online === false)
}
</script>

<style scoped>
.room-entities-comp { height: 100%; overflow: hidden; min-width: 0; }
.room-entities-section { overflow-y: auto; overflow-x: hidden; max-height: 100%; min-width: 0; }
.entity-row { display: flex; align-items: center; gap: 8px; padding: 6px 8px; cursor: pointer; border-bottom: 1px solid #2a2a2a; transition: background 0.15s; border-radius: 3px; min-width: 0; }
.entity-row:hover { background: rgba(255,255,255,0.04); }
.entity-highlight { border-left: 3px solid #e74c3c; }
.entity-interactable { background: rgba(52,152,219,0.1); border-left: 3px solid #3498db; }
.entity-icon { font-size: 14px; }
.entity-main { flex: 1; min-width: 0; display: flex; align-items: center; gap: 4px; overflow: hidden; }
.entity-name { color: #fff; font-weight: bold; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.entity-status { flex: 0 0 auto; font-size: 11px; font-weight: bold; white-space: nowrap; }
.entity-status-active { color: #2ecc71; }
.entity-status-offline { color: #e74c3c; }
.entity-level { color: #f1c40f; font-size: 12px; }
.entity-hp { color: #2ecc71; font-size: 12px; }
.entity-hostile { color: #e74c3c; font-size: 11px; font-weight: bold; }
.hint-row { color: #666; cursor: default; font-style: italic; }

.entity-block { }

.entity-actions {
  display: flex;
  gap: 4px;
  padding: 2px 8px 6px;
  flex-wrap: wrap;
  overflow: hidden;
}

.action-btn {
  padding: 1px 6px;
  border: 1px solid #444;
  background: #1e1e1e;
  color: #ccc;
  cursor: pointer;
  border-radius: 2px;
  font-size: 11px;
  line-height: 1.4;
  transition: all 0.15s;
}

.action-btn:hover {
  background: #333;
  color: #fff;
}

.view-btn { border-color: #3498db; color: #3498db; }
.view-btn:hover { background: #3498db; color: #fff; }

.talk-btn { border-color: #1abc9c; color: #1abc9c; }
.talk-btn:hover { background: #1abc9c; color: #000; }

.spar-btn { border-color: #f1c40f; color: #f1c40f; }
.spar-btn:hover { background: #f1c40f; color: #000; }

.kill-btn { border-color: #e74c3c; color: #e74c3c; }
.kill-btn:hover { background: #e74c3c; color: #fff; }

.learn-btn { border-color: #9b59b6; color: #c39bd3; }
.learn-btn:hover { background: #9b59b6; color: #fff; }

.trade-btn { border-color: #2ecc71; color: #2ecc71; }
.trade-btn:hover { background: #2ecc71; color: #000; }
</style>
