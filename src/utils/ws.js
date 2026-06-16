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

let reconnectAttempts = 0
let reconnectTimeout = null

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
  try {
    const wsUrl = resolveWebSocketUrl()

    socket = new WebSocket(wsUrl)

    socket.onopen = () => {
      reconnectAttempts = 0 // 重置重连次数
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout)
        reconnectTimeout = null
      }

      if (shouldReconnectRestoreOnOpen) {
        const reconnectMessage = buildReconnectMessage()
        if (reconnectMessage) {
          waitForReconnectRestore = true
          shouldReconnectRestoreOnOpen = false
          socket.send(JSON.stringify(reconnectMessage))
          return
        }
        shouldReconnectRestoreOnOpen = false
      }
      
      flushOpenCallbacks()
    }

    socket.onmessage = (event) => {
      const msg = JSON.parse(event.data)

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
          openCallbacks = []
        }
      }
    }

    socket.onclose = (event) => {
      waitForReconnectRestore = false
      
      // 🔧 改进：非正常关闭时尝试重新连接
      if (event.code !== 1000) {
        attemptReconnect()
      }
    }

    socket.onerror = (error) => {
      console.error("[WebSocket] 出错:", error)
      attemptReconnect()
    }
  } catch (e) {
    console.error("[WebSocket] 创建连接失败:", e)
    attemptReconnect()
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

// 🔧 新增：重新连接逻辑
function attemptReconnect() {
  if (reconnectAttempts >= RECONNECT_CONFIG.maxRetries) {
    console.error("[WebSocket] 达到最大重连次数，放弃重连")
    return
  }

  reconnectAttempts++
  shouldReconnectRestoreOnOpen = hasReconnectSession()
  const delay = RECONNECT_CONFIG.retryDelay * Math.pow(RECONNECT_CONFIG.backoffMultiplier, reconnectAttempts - 1)

  
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout)
  }
  
  reconnectTimeout = setTimeout(() => {
    createWebSocket()
  }, delay)
}

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
    console.error("[WebSocket] WebSocket 未连接或已关闭，无法发送:", message)
    if (!socket || socket.readyState === WebSocket.CLOSED) {
      initWebSocket(null)
      openCallbacks.push(send)
    }
  }
}
// 发送命令的封装（用于登录/注册等不需要 playerId 的阶段）
export function sendCommandMessage(type, subtype, args = {}) {
    const message = createSystemMessage(subtype, args)
    message.type = type
    message.ts = Date.now()
    sendMessage(message)
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
