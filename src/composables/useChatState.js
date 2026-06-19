import { nextTick, ref } from 'vue'

const MAX_CHAT_HISTORY = 200
const MAX_CHAT_LENGTH = 200

export function useChatState({ sendGameCommand, getPlayerId, addLog }) {
  const chatMessages = ref([])
  const command = ref('')
  const chatHistoryEl = ref(null)

  const scrollChatToBottom = () => {
    nextTick(() => {
      if (chatHistoryEl.value) {
        chatHistoryEl.value.scrollTop = chatHistoryEl.value.scrollHeight
      }
    })
  }

  const addChatMessage = ({ sender, message, ts }) => {
    chatMessages.value.push({ sender, message, ts })
    if (chatMessages.value.length > MAX_CHAT_HISTORY) {
      chatMessages.value.splice(0, chatMessages.value.length - MAX_CHAT_HISTORY)
    }
    scrollChatToBottom()
  }

  const sendChatCommand = () => {
    if (!command.value.trim()) return

    const input = command.value.trim()
    if (input.length > MAX_CHAT_LENGTH) {
      addLog("你要发送的内容太多了")
      return
    }

    sendGameCommand("chat", "public", getPlayerId(), { message: input })
    command.value = ''
  }

  return {
    chatMessages,
    command,
    chatHistoryEl,
    addChatMessage,
    sendChatCommand,
    scrollChatToBottom
  }
}
