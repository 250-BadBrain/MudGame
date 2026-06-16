export function createSystemMessage(subtype, args = {}) {
  return {
    type: "system",
    subtype,
    args
  }
}

export function createCommandMessage(subtype, args = {}) {
  return {
    type: "command",
    subtype,
    args
  }
}

export function createGameCommandMessage(subtype, playerId, args = {}) {
  return {
    type: "command",
    subtype,
    playerId,
    ts: Date.now(),
    args
  }
}

export function createChatMessage(playerId, content) {
  return {
    type: "chat",
    subtype: "public",
    playerId,
    ts: Date.now(),
    args: { message: content }
  }
}
