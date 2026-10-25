import { ref } from 'vue'

const MAX_LOG_ENTRIES = 500
export const DUNGEON_LEAVE_BUTTON_LOG_TYPE = 'dungeon_leave_button'

// 游戏日志：纯文本与结构化条目（副本离开按钮 / 剧情选项）的写入与裁剪。
// 滚动由展示组件（GameLogPanel）根据条目数量变化自行处理。
export function useGameLog() {
  const logs = ref([])

  const appendLogEntry = (entry) => {
    logs.value.push(entry)
    if (logs.value.length > MAX_LOG_ENTRIES) {
      logs.value.splice(0, logs.value.length - MAX_LOG_ENTRIES)
    }
  }

  const addLog = (message) => {
    if (!message) return
    if (typeof message !== 'string') {
      message = String(message)
    }
    // 移除 INFO/WARN/ERROR 前缀
    const cleanMessage = message.replace(/^(INFO|WARN|ERROR):\s*/, '')
    appendLogEntry(cleanMessage)
  }

  const addDungeonLeaveButtonLog = (token) => {
    appendLogEntry({
      type: DUNGEON_LEAVE_BUTTON_LOG_TYPE,
      text: '离开副本',
      token
    })
  }

  const getLogText = (log) => {
    if (typeof log === 'string') return log
    if (log && typeof log === 'object' && typeof log.text === 'string') return log.text
    return ''
  }

  const isDungeonLeaveButtonLog = (log) => {
    return !!(log && typeof log === 'object' && log.type === DUNGEON_LEAVE_BUTTON_LOG_TYPE)
  }

  const isDungeonChoiceLog = (log) => !!(log && typeof log === 'object' && log.type === 'dungeon_choice')

  const getLogClass = (log) => {
    const text = getLogText(log)
    if (text.startsWith('ERROR:')) return 'log-error'
    if (text.startsWith('WARN:')) return 'log-warn'
    if (text.startsWith('CHAT:')) return 'log-chat'
    return 'log-info'
  }

  return {
    logs,
    appendLogEntry,
    addLog,
    addDungeonLeaveButtonLog,
    getLogText,
    isDungeonLeaveButtonLog,
    isDungeonChoiceLog,
    getLogClass
  }
}
