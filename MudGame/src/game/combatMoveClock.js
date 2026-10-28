// 主动招式冷却窗口的纯函数（实现规范§9）：进度=剩余窗口占比，1=刚进入冷却，0=可用。
// 所有时间戳以服务端毫秒为准；显示时钟由 GameView 用 serverTime 校正后传入。

export function cooldownFraction(window, now) {
  if (!window || window.readyAt <= now || window.readyAt <= window.startedAt) return 0
  return Math.min(1, Math.max(0, (window.readyAt - now) / (window.readyAt - window.startedAt)))
}

// 剩余秒数的显示规则：≤10秒保留一位小数并向上取整；>10秒向上取整到整秒；未结束不显示0。
export function cooldownRemainingSeconds(window, now) {
  if (!window || window.readyAt <= now) return null
  const remainingMs = window.readyAt - now
  if (remainingMs > 10000) return Math.ceil(remainingMs / 1000)
  return Math.min(9.9, Math.ceil(remainingMs / 100) / 10)
}

// 服务端校时：displayNow = serverTime + (performance.now() - receivedMono)。
export function createDisplayClock() {
  let serverTime = 0
  let receivedMono = 0
  return {
    apply(serverTimeMs, monoNow) {
      serverTime = serverTimeMs
      receivedMono = monoNow
    },
    now() {
      if (!serverTime) return Date.now()
      return serverTime + (performance.now() - receivedMono)
    },
  }
}
