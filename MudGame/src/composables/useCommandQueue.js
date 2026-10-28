import { sendGameCommand as rawSendGameCommand } from '../utils/ws.js'

// 游戏指令的客户端限流队列：command 类指令每 2 秒最多 15 条，
// 超出的排队等待，避免玩家（或脚本）刷指令冲击服务端。
const COMMAND_LIMIT_PER_WINDOW = 15
const COMMAND_LIMIT_WINDOW_MS = 2000

export function useCommandQueue({ addLog }) {
  const commandQueue = []
  const commandSentTimestamps = []
  let commandQueueTimer = null
  let lastRateLimitNoticeAt = 0

  const trimCommandWindow = (now) => {
    while (commandSentTimestamps.length > 0 && (now - commandSentTimestamps[0]) >= COMMAND_LIMIT_WINDOW_MS) {
      commandSentTimestamps.shift()
    }
  }

  const scheduleCommandFlush = (delayMs) => {
    if (commandQueueTimer) {
      clearTimeout(commandQueueTimer)
    }
    commandQueueTimer = setTimeout(() => {
      commandQueueTimer = null
      flushCommandQueue()
    }, delayMs)
  }

  const flushCommandQueue = () => {
    if (commandQueue.length === 0) return

    while (commandQueue.length > 0) {
      const now = Date.now()
      trimCommandWindow(now)

      if (commandSentTimestamps.length >= COMMAND_LIMIT_PER_WINDOW) {
        if (now - lastRateLimitNoticeAt >= COMMAND_LIMIT_WINDOW_MS) {
          addLog("不要急，慢慢来")
          lastRateLimitNoticeAt = now
        }
        const waitMs = Math.max(1, COMMAND_LIMIT_WINDOW_MS - (now - commandSentTimestamps[0]))
        scheduleCommandFlush(waitMs)
        return
      }

      const next = commandQueue.shift()
      rawSendGameCommand(next.type, next.subtype, next.playerId, next.args)
      commandSentTimestamps.push(now)
    }
  }

  const sendGameCommand = (type, subtype, playerId, args = {}) => {
    if (type !== 'command') {
      rawSendGameCommand(type, subtype, playerId, args)
      return
    }

    commandQueue.push({ type, subtype, playerId, args })
    flushCommandQueue()
  }

  // sendImmediateCommand 施放指令专用：与原command共享15条/2秒额度，
  // 额度用尽立即返回false（调用方立即提示），绝不进入等待队列（实现规范§9）。
  const sendImmediateCommand = (type, subtype, playerId, args = {}) => {
    const now = Date.now()
    trimCommandWindow(now)
    if (commandSentTimestamps.length >= COMMAND_LIMIT_PER_WINDOW) {
      return false
    }
    rawSendGameCommand(type, subtype, playerId, args)
    commandSentTimestamps.push(now)
    return true
  }

  const dispose = () => {
    if (commandQueueTimer) {
      clearTimeout(commandQueueTimer)
      commandQueueTimer = null
    }
  }

  return { sendGameCommand, sendImmediateCommand, dispose }
}
