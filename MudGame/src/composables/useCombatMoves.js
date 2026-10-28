// 主动招式的前端状态（实现规范§9）：
// 只依赖注入对象和Vue，不导入别的composable；由GameView装配。
// 技能可用性由服务端权威判定，前端倒计时只负责显示，不在浏览器授权施放。

import { ref } from 'vue'
import { createDisplayClock } from '../game/combatMoveClock.js'

const EMPTY_VIEW = () => ({
  capabilities: [],
  serverEpoch: '',
  revision: 0,
  serverTime: 0,
  battleId: '',
  sequence: 0,
  mp: 0,
  effectiveActionSpeed: 4,
  actionStartedAt: 0,
  actionReadyAt: 0,
  actionDurationMs: 4000,
  moves: [],
})

export function useCombatMoves({ sendImmediateCommand, addLog, getPlayerId, monotonicNow }) {
  const view = ref(EMPTY_VIEW())
  const pending = ref(false)
  const clock = createDisplayClock()
  const displayNow = ref(0)
  let displayTimer = null
  let confirmedEpoch = ''

  const hasCapability = () => (view.value.capabilities || []).includes('combat_moves_v1')

  const startDisplayTick = () => {
    if (displayTimer) return
    displayTimer = setInterval(() => {
      displayNow.value = clock.now()
    }, 100)
  }
  const stopDisplayTick = () => {
    if (displayTimer) {
      clearInterval(displayTimer)
      displayTimer = null
    }
  }

  // applySnapshot 服务端权威视图：跨epoch仅在已确认新连接上接受并reset，旧socket包不触发反向切换。
  const applySnapshot = (snapshot) => {
    if (!snapshot || typeof snapshot !== 'object') return
    const epoch = snapshot.serverEpoch || ''
    if (epoch && confirmedEpoch && epoch !== confirmedEpoch) {
      // 旧epoch的包：忽略，不倒退视图
      return
    }
    confirmedEpoch = epoch
    const prevRevision = view.value.revision || 0
    if (snapshot.revision < prevRevision && snapshot.serverEpoch === view.value.serverEpoch) {
      return // 同epoch旧revision：不倒退
    }
    view.value = snapshot
    clock.apply(snapshot.serverTime || 0, monotonicNow ? monotonicNow() : performance.now())
    displayNow.value = clock.now()
    startDisplayTick()
  }

  const requestRefresh = () => {
    const playerId = getPlayerId()
    if (!playerId) return
    sendImmediateCommand('command', 'get_combat_moves', playerId, {})
  }

  // castMove 施放：不进排队队列；立即发送或立即拒绝，禁止并发多重pending。
  const castMove = (moveId, battleId) => {
    if (pending.value) return false
    if (!hasCapability()) return false
    if (!battleId) {
      addLog('未处于战斗中，无法出招。')
      return false
    }
    const playerId = getPlayerId()
    if (!playerId) return false
    pending.value = true
    const ok = sendImmediateCommand('command', 'cast_combat_move', playerId, {
      moveId,
      battleId,
      expectedSequence: view.value.sequence || 0,
    })
    if (!ok) {
      pending.value = false
      addLog('操作过快，请稍后重试。')
      return false
    }
    // 发出后等权威回包；不自行扣MP或开CD
    return true
  }

  const settlePending = () => {
    pending.value = false
  }

  // reset 断线/换角色：清pending与视图，下次重连重新查询，不重发旧cast。
  const reset = () => {
    pending.value = false
    confirmedEpoch = ''
    view.value = EMPTY_VIEW()
    stopDisplayTick()
  }

  const dispose = () => {
    stopDisplayTick()
  }

  return { view, pending, displayNow, hasCapability, applySnapshot, requestRefresh, castMove, settlePending, reset, dispose }
}
