<template>
  <div class="character-select-container">
    <h2>欢迎 {{ playerStore.userId }}，请选择你的角色</h2>

    <div class="character-list">
      
      <div v-if="loading" class="loading-msg">
        <p>正在加载角色列表...</p>
      </div>

      <div v-else-if="characters.length === 0" class="no-chars">
        <p>你还没有任何角色，请创建！</p>
      </div>

      <div v-else v-for="char in characters" :key="char.id" class="char-card" :class="{ 'selected': selectedCharId === char.id }" @click="highlightCharacter(char.id)">
        <p>
          <strong>{{ char.name }}</strong>
        </p>
      </div>
    </div>

    <div class="main-actions">
      <button @click="goToCreate" :disabled="loading" class="btn-create-nav">创建新角色</button>
      <button @click="enterGame" :disabled="!selectedCharId || loading" class="btn-enter">进入江湖</button>
      <button @click="openDeleteCharDialog" :disabled="!selectedCharId || loading" class="btn-delete">删除角色</button>
    </div>

    <p v-if="error" class="error-msg">{{ error }}</p>

    <div class="logout-btn">
      <button @click="logout" :disabled="loading" class="btn-logout">退出登录</button>
    </div>

    <!-- 删除角色确认对话框 -->
    <div v-if="showDeleteDialog" class="modal-overlay" @click="closeDeleteCharDialog">
      <div class="modal-content" @click.stop>
        <h3>删除角色</h3>
        <p>确定要删除角色 <strong>{{ getSelectedCharName() }}</strong> 吗？</p>
        <p class="warning-text">此操作无法撤销，角色的所有数据将被永久删除。</p>
        <div class="modal-buttons">
          <button @click="closeDeleteCharDialog" class="btn-cancel">取消</button>
          <button 
            @click="confirmDeleteChar" 
            :disabled="loading"
            class="btn-confirm-delete"
          >
            {{ loading ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { usePlayerStore } from '@/stores/player'
import { initWebSocket, removeMessageHandler, sendMessage } from '../utils/ws.js'
import { createSystemMessage } from '../protocal/message.js'

const router = useRouter()
const playerStore = usePlayerStore()

const loading = ref(false)
const error = ref('')
// 假设角色结构: { id: string, name: string, level: number }
// 初始化为 store 中已缓存的角色列表
const characters = ref(playerStore.characters) 
const selectedCharId = ref(null)
// 删除角色相关
const showDeleteDialog = ref(false)

// --- 动作函数 ---

const highlightCharacter = (id) => {
  if (selectedCharId.value === id) {
    selectedCharId.value = null
  } else {
    selectedCharId.value = id
  }
}

const getSelectedCharName = () => {
  const char = characters.value.find(c => c.id === selectedCharId.value)
  return char ? char.name : ''
}

const goToCreate = () => {
  router.push({ name: 'CharacterCreate' })
}

// 1. 发送获取角色列表请求
const fetchCharacters = () => {
  if (loading.value) return
  if (!playerStore.userId) {
    error.value = '错误：未获得用户信息'
    return
  }
  loading.value = true
  error.value = ''

  sendMessage(createSystemMessage('get_characters', { userId: playerStore.userId }))
}

// 3. 发送选择角色请求并跳转
const enterGame = () => {
  if (loading.value || !selectedCharId.value) return
  loading.value = true
  error.value = ''
  
  sendMessage(createSystemMessage('select_character', {
    userId: playerStore.userId,
    characterId: selectedCharId.value
  }))
}

// 4. 退出登录
const logout = () => {
  playerStore.clearAuth()
  router.push({ name: 'Login' })
}

// 5. 删除角色 - 打开对话框
const openDeleteCharDialog = () => {
  if (!selectedCharId.value) return
  showDeleteDialog.value = true
}

// 6. 删除角色 - 关闭对话框
const closeDeleteCharDialog = () => {
  showDeleteDialog.value = false
}

// 7. 删除角色 - 确认删除
const confirmDeleteChar = () => {
  if (!selectedCharId.value) return
  loading.value = true
  error.value = ''
  
  sendMessage(createSystemMessage('delete_character', {
    userId: playerStore.userId,
    characterId: selectedCharId.value
  }))
}

// --- WebSocket 消息处理 ---

const handleCharacterSelectMessage = (msg) => {
  if (msg.type === 'system') {
    if (msg.subtype === 'get_characters') {
      loading.value = false
      if (msg.flag) {
        characters.value = msg.results.characters || []
        playerStore.setCharacters(characters.value)
        error.value = ''
      } else {
        error.value = `获取角色列表失败: ${msg.results.error || '未知错误'}`
      }
    } else if (msg.subtype === 'reconnect') {
      if (msg.flag) {
        setTimeout(() => {
          fetchCharacters()
        }, 100)
      } else {
        playerStore.clearAuth()
        router.push({ name: 'Login' })
      }
    } else if (msg.subtype === 'select_character') {
      loading.value = false
      if (msg.flag) {
        const selectedId = msg.results.characterId
        playerStore.setPlayerId(selectedId)
        router.push({ name: 'Game' })
      } else {
        error.value = `选择角色失败: ${msg.results.error || '未知错误'}`
      }
    } else if (msg.subtype === 'delete_character') {
      loading.value = false
      if (msg.flag) {
        error.value = '角色已成功删除'
        selectedCharId.value = null
        closeDeleteCharDialog()
        setTimeout(() => {
          fetchCharacters()
        }, 500)
      } else {
        error.value = `删除角色失败: ${msg.results.error || '未知错误'}`
      }
    }
  }
}

onMounted(() => {
  // 注册 WebSocket 监听器，处理所有角色选择相关的系统消息
  initWebSocket(handleCharacterSelectMessage)
  
  // 如果页面刷新后 userId 存在，自动发送 reconnect 来恢复后端 session
  // reconnect成功后，reconnect处理器会自动调用fetchCharacters()
  // 不要在这里直接调用fetchCharacters()，否则会导致userId未被绑定到session
  if (playerStore.userId) {
    if (!playerStore.reconnectToken) {
      playerStore.clearAuth()
      router.push({ name: 'Login' })
      return
    }

    sendMessage(createSystemMessage('reconnect', {
      userId: playerStore.userId,
      reconnectToken: playerStore.reconnectToken
    }))
  } else {
    error.value = '错误：无法获取用户信息，请重新登录'
  }
})

onUnmounted(() => {
  removeMessageHandler(handleCharacterSelectMessage)
})
</script>

<style scoped>
.character-select-container {
  width: 500px;
  margin: 80px auto;
  padding: 30px;
  /* 沿用或调整武侠风格 */
  background-color: var(--color-background-dark); 
  color: var(--color-text-light);
  border: 4px solid var(--color-accent-gold); 
  border-radius: 8px;
  box-shadow: 0 0 15px rgba(0, 0, 0, 0.5);
  text-align: center;
}

h2, h3 {
  color: var(--color-accent-gold);
  margin-bottom: 20px;
  padding-bottom: 5px;
  border-bottom: 2px solid var(--color-accent-gold);
}

.character-list {
  margin-bottom: 30px;
  text-align: left;
  border: 1px solid var(--color-border-subtle);
  padding: 15px;
  border-radius: 5px;
  min-height: 100px; /* 确保有足够的空间 */
}

.loading-msg, .no-chars {
    text-align: center;
    padding: 20px 0;
    color: var(--color-mud-primary);
}

.char-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px;
  border-bottom: 1px dashed var(--color-border-subtle);
  cursor: pointer;
  transition: background-color 0.2s;
}

.char-card:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.char-card.selected {
  background-color: rgba(241, 196, 15, 0.2); /* Gold with opacity */
  border: 1px solid var(--color-accent-gold);
}

.main-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
  gap: 10px;
}

.btn-enter, .btn-create-nav, .btn-delete {
    flex: 1;
    padding: 10px;
    font-weight: bold;
    border: none;
    cursor: pointer;
    border-radius: 4px;
}

.btn-enter {
    background-color: var(--color-accent-gold);
    color: var(--color-background-dark);
}

.btn-create-nav {
    background-color: var(--color-accent-blue);
    color: white;
}

.btn-delete {
    background-color: #e74c3c;
    color: white;
}

.btn-enter:disabled, .btn-delete:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.btn-enter:hover:not(:disabled), .btn-create-nav:hover, .btn-delete:hover:not(:disabled) {
    filter: brightness(1.2);
}

.logout-btn {
  margin-top: 40px;
}

.btn-logout {
  width: 100%;
  padding: 10px;
  background-color: #7f8c8d;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: bold;
}

.btn-logout:hover {
  background-color: #95a5a6;
}

/* 模态对话框 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background-color: var(--color-background-dark);
  border: 2px solid var(--color-accent-gold);
  border-radius: 8px;
  padding: 30px;
  max-width: 400px;
  box-shadow: 0 0 20px rgba(0, 0, 0, 0.8);
}

.modal-content h3 {
  color: #e74c3c;
  margin-bottom: 15px;
}

.modal-content p {
  color: var(--color-text-light);
  margin-bottom: 10px;
  font-size: 14px;
}

.warning-text {
  color: #f39c12;
  font-weight: bold;
}

.modal-buttons {
  display: flex;
  gap: 10px;
  margin-top: 20px;
}

.btn-cancel {
  flex: 1;
  background-color: var(--color-accent-blue);
  color: var(--color-text-light);
  padding: 10px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: filter 0.3s;
}

.btn-cancel:hover {
  filter: brightness(1.2);
}

.btn-confirm-delete {
  flex: 1;
  background-color: #c0392b;
  color: var(--color-text-light);
  padding: 10px;
  font-weight: bold;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: filter 0.3s;
}

.btn-confirm-delete:hover:not(:disabled) {
  filter: brightness(1.2);
}

.btn-confirm-delete:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

</style>
