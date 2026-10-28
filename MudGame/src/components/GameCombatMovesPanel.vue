<template>
  <div v-if="hasCapability && moves.length" class="moves-panel">
    <div class="moves-title">招式</div>
    <div v-for="group in groupedMoves" :key="group.slot" class="moves-group">
      <span class="moves-slot-label">{{ slotLabel(group.slot) }}</span>
      <button
        v-for="move in group.moves"
        :key="move.id"
        class="move-btn"
        :disabled="disabled || move.unavailableReason !== ''"
        :title="moveTip(move)"
        @click="$emit('cast', move)">
        <span class="move-overlay" :style="{ width: (fraction(move) * 100) + '%' }"></span>
        <span class="move-label">{{ move.name }}</span>
        <span v-if="remaining(move) !== null" class="move-remaining">{{ remaining(move) }}s</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { cooldownFraction, cooldownRemainingSeconds } from '../game/combatMoveClock.js'

const props = defineProps({
  view: { type: Object, required: true },
  displayNow: { type: Number, required: true },
  pending: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
})

defineEmits(['cast'])

const hasCapability = computed(() => (props.view.capabilities || []).includes('combat_moves_v1'))
const moves = computed(() => props.view.moves || [])

const groupedMoves = computed(() => {
  const groups = []
  const bySlot = {}
  for (const move of moves.value) {
    if (!bySlot[move.slot]) {
      bySlot[move.slot] = { slot: move.slot, moves: [] }
      groups.push(bySlot[move.slot])
    }
    bySlot[move.slot].moves.push(move)
  }
  return groups
})

const slotLabel = (slot) => {
  const labels = { combat: '战斗', internal: '内功', movement: '身法', guard: '护体', study: '学识' }
  return labels[slot] || slot
}

// 实际受限窗口 = max(出招CD, 本招技能CD)，两种CD不相加（实现规范§3.4）。
const blockedWindow = (move) => {
  const actionEnd = props.view.actionReadyAt || 0
  const skillEnd = move.skillReadyAt || 0
  const end = Math.max(actionEnd, skillEnd)
  if (end <= props.displayNow) return null
  const start = Math.min(
    skillEnd > props.displayNow ? (move.skillStartedAt || props.view.actionStartedAt || props.displayNow) : Number.MAX_SAFE_INTEGER,
    actionEnd > props.displayNow ? (props.view.actionStartedAt || props.displayNow) : Number.MAX_SAFE_INTEGER
  )
  return { startedAt: start === Number.MAX_SAFE_INTEGER ? props.displayNow : start, readyAt: end }
}

const fraction = (move) => {
  const w = blockedWindow(move)
  if (!w) return 0
  return cooldownFraction(w, props.displayNow)
}

const remaining = (move) => {
  const w = blockedWindow(move)
  if (!w) return null
  return cooldownRemainingSeconds(w, props.displayNow)
}

const moveTip = (move) => {
  const parts = [move.description || '']
  parts.push(`出招${Math.round((move.actionMs || 4000) / 100) / 10}秒（攻速后）`)
  parts.push(`技能CD ${move.cooldownSec}秒`)
  if ((move.mpCost || 0) > 0) parts.push(`内力 ${move.mpCost}`)
  if (move.unavailableReason) parts.push(`不可用：${move.unavailableReason}`)
  return parts.filter(Boolean).join('｜')
}
</script>

<style scoped>
.moves-panel {
  width: 100%;
  margin-top: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.moves-title {
  font-size: 0.8em;
  color: var(--color-accent-gold);
  margin-right: 4px;
}

.moves-group {
  display: flex;
  gap: 6px;
  align-items: center;
}

.moves-slot-label {
  font-size: 0.75em;
  color: var(--color-text-dim, #888);
}

.move-btn {
  position: relative;
  padding: 4px 8px;
  background-color: var(--color-background-secondary);
  border: 1px solid var(--color-border-subtle);
  color: var(--color-text-light);
  cursor: pointer;
  border-radius: 4px;
  font-size: 0.85em;
  overflow: hidden;
}

.move-btn:disabled {
  cursor: default;
  opacity: 0.7;
}

/* 高透明度合并进度条（20%不透明度），位于文字下方，不阻断点击 */
.move-overlay {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  background: var(--color-accent-gold);
  opacity: 0.2;
  pointer-events: none;
}

.move-label,
.move-remaining {
  position: relative;
}

.move-remaining {
  margin-left: 4px;
  font-size: 0.8em;
  color: var(--color-accent-gold);
}
</style>
