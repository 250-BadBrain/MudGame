// 📄 文件: stores/player.js

import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const usePlayerStore = defineStore('player', () => {
    
    // localStorage 的键名常量
    const STORAGE_KEY_USER_ID = 'mud_userId'
    const STORAGE_KEY_PLAYER_ID = 'mud_playerId'
    const STORAGE_KEY_CHARACTERS = 'mud_characters'  // 角色列表缓存
    const STORAGE_KEY_RECONNECT_TOKEN = 'mud_reconnectToken'
    const SESSION_STORAGE_KEY_PLAYER_ID = 'mud_sessionPlayerId'  // sessionStorage 中的 playerId
    
    // 🔧 修复：Store 初始化策略
    // localStorage: 
    //   - userId（登录状态）- 跨浏览器标签页持久化
    //   - characters（角色列表缓存）- 跨浏览器标签页持久化
    // sessionStorage: 
    //   - playerId（当前会话的角色）- 只在当前标签页持久化，刷新时恢复
    // 💡 新增选项：是否在刷新时恢复 playerId
    // 如果设为 false，则刷新时总是进入角色选择界面
    const RESTORE_PLAYERID_ON_REFRESH = false  // ⚙️ 设置为 false：刷新时重置 playerId
    
    function initFromLocalStorage() {
      const userIdValue = localStorage.getItem(STORAGE_KEY_USER_ID)
      const charactersValue = localStorage.getItem(STORAGE_KEY_CHARACTERS)
      const reconnectTokenValue = localStorage.getItem(STORAGE_KEY_RECONNECT_TOKEN)
      // 从 sessionStorage 恢复 playerId（仅在当前标签页刷新时）
      // 但可以通过 RESTORE_PLAYERID_ON_REFRESH 控制是否恢复
      const playerIdValue = RESTORE_PLAYERID_ON_REFRESH ? sessionStorage.getItem(SESSION_STORAGE_KEY_PLAYER_ID) : null
      let parsedCharacters = []
      if (charactersValue) {
        try {
          parsedCharacters = JSON.parse(charactersValue)
        } catch {
          localStorage.removeItem(STORAGE_KEY_CHARACTERS)
        }
      }
      
      return {
        userId: (userIdValue && userIdValue !== '') ? userIdValue : null,
        playerId: (playerIdValue && playerIdValue !== '') ? playerIdValue : null,
        characters: parsedCharacters,
        reconnectToken: (reconnectTokenValue && reconnectTokenValue !== '') ? reconnectTokenValue : null
      }
    }
    
    const initialState = initFromLocalStorage()
    const userId = ref(initialState.userId)
    const playerId = ref(initialState.playerId)
    const characters = ref(initialState.characters)
    const reconnectToken = ref(initialState.reconnectToken)

    // Actions/Mutations
    function setUserId(id) {
      userId.value = id
      // 持久化到 localStorage
      if (id) {
        localStorage.setItem(STORAGE_KEY_USER_ID, id)
      } else {
        localStorage.removeItem(STORAGE_KEY_USER_ID)
      }
    }
    
    function setPlayerId(id) {
      playerId.value = id
      // 💡 重要：保存 playerId 到 sessionStorage（不是 localStorage）
      // sessionStorage 只在当前浏览器标签页有效
      // 刷新页面时会恢复，但关闭标签页后会清空
      // 这样可以：
      // 1. 刷新页面时保持游戏状态
      // 2. 避免旧的 playerId 污染其他会话
      if (id) {
        sessionStorage.setItem(SESSION_STORAGE_KEY_PLAYER_ID, id)
      } else {
        sessionStorage.removeItem(SESSION_STORAGE_KEY_PLAYER_ID)
      }
    }

    function setReconnectToken(token) {
      reconnectToken.value = token
      if (token) {
        localStorage.setItem(STORAGE_KEY_RECONNECT_TOKEN, token)
      } else {
        localStorage.removeItem(STORAGE_KEY_RECONNECT_TOKEN)
      }
    }

    // 新增：保存角色列表缓存
    function setCharacters(charList) {
      characters.value = charList
      // 💡 重要：保存角色列表到 localStorage
      // 这样刷新页面时可以立即显示角色列表，不需要重新请求
      if (charList && charList.length > 0) {
        localStorage.setItem(STORAGE_KEY_CHARACTERS, JSON.stringify(charList))
      } else {
        localStorage.removeItem(STORAGE_KEY_CHARACTERS)
      }
    }

    // 新增：清空角色列表缓存
    function clearCharacters() {
      characters.value = []
      localStorage.removeItem(STORAGE_KEY_CHARACTERS)
    }

    // 退出登录时调用
    function clearAuth() {
      userId.value = null
      playerId.value = null
      characters.value = []
      // 清空 localStorage 和 sessionStorage
      localStorage.removeItem(STORAGE_KEY_USER_ID)
      localStorage.removeItem(STORAGE_KEY_PLAYER_ID)  // 保险起见
      localStorage.removeItem(STORAGE_KEY_CHARACTERS)
      localStorage.removeItem(STORAGE_KEY_RECONNECT_TOKEN)
      sessionStorage.removeItem(SESSION_STORAGE_KEY_PLAYER_ID)
      reconnectToken.value = null
    }

    // Getter
    const isLoggedIn = computed(() => !!userId.value) // 使用 userId 判断是否登录
    const isCharacterSelected = computed(() => !!playerId.value) // 判断是否选角

    return { 
      userId, setUserId,
      reconnectToken, setReconnectToken,
      playerId, setPlayerId, 
      characters, setCharacters, clearCharacters,
      clearAuth, isLoggedIn, isCharacterSelected 
    }
})