// 玩家指令
export function createCommandMessage(playerId, content) {
  return {
    type: "command",
    playerId,
    content,
    timestamp: Date.now()
  }
}

// 聊天
export function createChatMessage(playerId, content) {
  return {
    type: "chat",
    playerId,
    content,
    timestamp: Date.now()
  }
}

// 系统消息
export function createSystemMessage(content) {
  return {
    type: "system",
    content,
    timestamp: Date.now()
  }
}
