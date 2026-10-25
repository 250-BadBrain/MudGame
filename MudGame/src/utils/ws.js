import { createSystemMessage, createGameCommandMessage, createChatMessage } from '../protocal/message.js'

let socket = null
const messageHandlers = [] // 改为数组，支持多个处理器
let openCallbacks = [] // 连接成功后要执行的队列
let waitForReconnectRestore = false
let shouldReconnectRestoreOnOpen = false

const STORAGE_KEY_USER_ID = 'mud_userId'
const STORAGE_KEY_RECONNECT_TOKEN = 'mud_reconnectToken'
const SESSION_STORAGE_KEY_PLAYER_ID = 'mud_sessionPlayerId'

//  改进：WebSocket 重新连接配置
const RECONNECT_CONFIG = {
  maxRetries: 5,
  retryDelay: 3000, // 3秒
  backoffMultiplier: 1.5
}

// 心跳保活：定期发送 system/ping，服务端回 pong 刷新活跃时间；
// 超过阈值没有任何下行消息时视为连接已死，主动断开触发重连。
const HEARTBEAT_INTERVAL_MS = 25000
const HEARTBEAT_TIMEOUT_MS = 60000

let reconnectAttempts = 0
let reconnectTimeout = null
let heartbeatTimer = null
let lastActivityAt = 0

const DEFAULT_BACKEND_PORT = '4009'

// 初始化连接（仅在首次时创建连接）
export function initWebSocket(onMessage) {
  // 如果处理器已注册，直接返回
  if (onMessage && messageHandlers.includes(onMessage)) {
    return socket
  }

  // 如果处理器不为空，注册它
  if (onMessage && !messageHandlers.includes(onMessage)) {
    messageHandlers.push(onMessage)
  }

  // 如果连接已存在且打开，直接返回
  if (socket && socket.readyState !== WebSocket.CLOSED) {
    return socket
  }

  createWebSocket()
  return socket
}

function createWebSocket() {
  // 已有连接在手（连接中/已连接）时不得再建新连接，防止重连定时器与
  // sendMessage 恢复路径竞争产生两条并发连接
  if (socket && (socket.readyState === WebSocket.CONNECTING || socket.readyState === WebSocket.OPEN)) {
    return
  }

  try {
    const wsUrl = resolveWebSocketUrl()

    const ws = new WebSocket(wsUrl)
    socket = ws

    ws.onopen = () => {
      // 只处理当前连接的事件，忽略被替换掉的旧连接
      if (socket !== ws) return
      reconnectAttempts = 0 // 重置重连次数
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout)
        reconnectTimeout = null
      }
      lastActivityAt = Date.now()
      startHeartbeat()

      if (shouldReconnectRestoreOnOpen) {
        const reconnectMessage = buildReconnectMessage()
        if (reconnectMessage) {
          waitForReconnectRestore = true
          shouldReconnectRestoreOnOpen = false
          ws.send(JSON.stringify(reconnectMessage))
          return
        }
        shouldReconnectRestoreOnOpen = false
      }

      flushOpenCallbacks()
    }

    ws.onmessage = (event) => {
      if (socket !== ws) return
      lastActivityAt = Date.now()

      let msg = null
      try {
        msg = JSON.parse(event.data)
      } catch (e) {
        // 代理心跳帧/错误页等非 JSON 内容不应炸掉消息处理
        console.warn('[WebSocket] 收到无法解析的消息，已忽略:', e)
        return
      }

      const isReconnectResponse = msg?.type === 'system' && msg?.subtype === 'reconnect' && waitForReconnectRestore

      // 调用所有注册的处理器
      messageHandlers.forEach(handler => {
        try {
          handler(msg)
        } catch (e) {
          console.error("[WebSocket] 消息处理器执行错误:", e)
        }
      })

      if (isReconnectResponse) {
        waitForReconnectRestore = false
        if (msg.flag) {
          flushOpenCallbacks()
        } else {
          // 会话恢复失败：积压的指令属于已失效的会话，直接丢弃并提示
          console.warn("[WebSocket] 会话恢复失败，丢弃队列中的", openCallbacks.length, "条指令")
          openCallbacks = []
        }
      }
    }

    ws.onclose = (event) => {
      // 旧连接的关闭事件不触发重连（新连接可能已建立）
      if (socket !== ws) return
      waitForReconnectRestore = false
      stopHeartbeat()

      // 正常关闭（code 1000）不重连；心跳超时等主动断开使用 4000 等应用码
      if (event.code !== 1000) {
        attemptReconnect()
      }
    }

    ws.onerror = () => {
      // 仅记录错误：onclose 一定会随后触发，重连统一由 onclose 驱动，避免重复计数
      if (socket === ws) {
        console.error("[WebSocket] 出错")
      }
    }
  } catch (e) {
    console.error("[WebSocket] 创建连接失败:", e)
    attemptReconnect()
  }
}

// 确保尽快建立新连接（复用/打断排队的重连），并带上会话恢复握手
function ensureConnecting() {
  if (socket && (socket.readyState === WebSocket.CONNECTING || socket.readyState === WebSocket.OPEN)) {
    return
  }
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout)
    reconnectTimeout = null
  }
  shouldReconnectRestoreOnOpen = hasReconnectSession()
  createWebSocket()
}

function startHeartbeat() {
  stopHeartbeat()
  heartbeatTimer = setInterval(() => {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return
    }
    const now = Date.now()
    if (now - lastActivityAt > HEARTBEAT_TIMEOUT_MS) {
      // 长时间没有任何下行消息，视为连接已死，主动断开以触发重连。
      // 必须用非 1000 的应用关闭码：默认码会被 onclose 当作正常关闭而跳过重连
      console.warn("[WebSocket] 心跳超时，主动断开以重连")
      try {
        socket.close(4000, 'heartbeat timeout')
      } catch (e) {
        console.error("[WebSocket] 关闭失活连接失败:", e)
      }
      return
    }
    try {
      socket.send(JSON.stringify(createSystemMessage('ping')))
    } catch (e) {
      console.error("[WebSocket] 心跳发送失败:", e)
    }
  }, HEARTBEAT_INTERVAL_MS)
}

function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
}

function resolveWebSocketUrl() {
  const envUrl = import.meta.env.VITE_WS_URL?.trim()
  if (envUrl) {
    return envUrl
  }

  // 生产域名兜底：避免未注入环境变量时误连到前端自身的 :4009
  if (window.location.hostname === 'mud.game.h2seo4.win') {
    return 'wss://api-mud.h2seo4.win/ws'
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const hostname = window.location.hostname || 'localhost'
  return `${protocol}//${hostname}:${DEFAULT_BACKEND_PORT}/ws`
}

// 重新连接逻辑
function attemptReconnect() {
  if (reconnectTimeout) {
    // 已有重连在排队，避免 onclose / 异常路径重复触发导致计数翻倍
    return
  }
  if (reconnectAttempts >= RECONNECT_CONFIG.maxRetries) {
    console.error("[WebSocket] 达到最大重连次数，暂停重连（切回页面后会再次尝试）")
    return
  }

  reconnectAttempts++
  shouldReconnectRestoreOnOpen = hasReconnectSession()
  const delay = RECONNECT_CONFIG.retryDelay * Math.pow(RECONNECT_CONFIG.backoffMultiplier, reconnectAttempts - 1)

  reconnectTimeout = setTimeout(() => {
    reconnectTimeout = null
    createWebSocket()
  }, delay)
}

// 标签页切到后台时浏览器会节流定时器导致心跳停发、连接被误判断线；
// 切回前台时若连接已断，重置计数并立即重连
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState !== 'visible') return
  if (socket && (socket.readyState === WebSocket.CONNECTING || socket.readyState === WebSocket.OPEN)) return
  if (reconnectTimeout) return
  reconnectAttempts = 0
  ensureConnecting()
})

function buildReconnectMessage() {
  const userId = localStorage.getItem(STORAGE_KEY_USER_ID)
  const reconnectToken = localStorage.getItem(STORAGE_KEY_RECONNECT_TOKEN)
  const playerId = sessionStorage.getItem(SESSION_STORAGE_KEY_PLAYER_ID)

  if (!userId || !reconnectToken) {
    return null
  }

  const args = { userId, reconnectToken }
  if (playerId) {
    args.playerId = playerId
  }

  return {
    type: 'system',
    subtype: 'reconnect',
    args,
  }
}

function hasReconnectSession() {
  return !!buildReconnectMessage()
}

function flushOpenCallbacks() {
  const queuedCallbacks = openCallbacks
  openCallbacks = []
  queuedCallbacks.forEach(cb => cb())
}

// 移除指定的消息处理器
export function removeMessageHandler(handler) {
  const index = messageHandlers.indexOf(handler)
  if (index > -1) {
    messageHandlers.splice(index, 1)
  }
}

// 安全发送消息：如果未连接，先放入队列等待连接成功再发送
export function sendMessage(message) {
  const send = () => {
    if (socket && socket.readyState === WebSocket.OPEN && !waitForReconnectRestore) {
      socket.send(JSON.stringify(message))
    } else {
      console.error("[WebSocket] WebSocket 状态异常，无法发送:", message)
    }
  }

  if (socket && socket.readyState === WebSocket.OPEN && !waitForReconnectRestore) {
    send()
  } else if ((socket && socket.readyState === WebSocket.CONNECTING) || waitForReconnectRestore) {
    openCallbacks.push(send)
  } else {
    // 已关闭/正在关闭：立即建立新连接（走会话恢复握手），并把指令排队
    ensureConnecting()
    openCallbacks.push(send)
  }
}

// 供游戏内使用的指令发送，需要 playerId
export function sendGameCommand(type, subtype, playerId, args = {}) {
    let message
    if (type === 'command') {
        message = createGameCommandMessage(subtype, playerId, args)
    } else if (type === 'chat') {
        message = createChatMessage(playerId, args?.message || args?.content || '')
    } else {
        message = createSystemMessage(subtype, args)
        message.ts = Date.now()
    }
    sendMessage(message)
}
