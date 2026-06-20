<template>
  <div class="login-container">
    <h2>破碎江湖</h2>

    <div class="form-group">
      <label>用户名:</label>
      <input v-model="username" placeholder="请输入用户名" @keyup.enter="login" />
    </div>
    <div class="form-group">
      <label>密码:</label>
      <input type="password" v-model="password" placeholder="请输入密码" @keyup.enter="login" />
    </div>

    <div class="buttons">
      <button @click="login" :disabled="loading" class="btn-login">
        {{ loading ? '登录中...' : '登录' }}
      </button>
      <button @click="goToRegister" :disabled="loading" class="btn-register">
        注册账号
      </button>
    </div>

    <p v-if="error" class="error-msg">{{ error }}</p>

    <!-- 注销账号区域 -->
    <div class="logout-section">
      <p class="logout-hint">已有账号？</p>
      <button @click="openDeleteDialog" class="btn-delete-account">注销账号</button>
    </div>

    <!-- 删除确认对话框 -->
    <div v-if="showDeleteDialog" class="modal-overlay" @click="closeDeleteDialog">
      <div class="modal-content" @click.stop>
        <h3>注销账号</h3>
        <p>此操作将永久删除您的账号和所有相关角色数据，无法恢复！</p>
        <p class="confirm-text">请输入用户名 <strong>{{ username }}</strong> 来确认注销:</p>
        <input v-model="deleteConfirmInput" placeholder="输入用户名以确认" class="confirm-input" />
        <div class="modal-buttons">
          <button @click="closeDeleteDialog" class="btn-cancel">取消</button>
          <button 
            @click="confirmDelete" 
            :disabled="deleteConfirmInput !== username || loading"
            class="btn-confirm-delete"
          >
            {{ loading ? '删除中...' : '确认注销' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { initWebSocket, removeMessageHandler, sendMessage } from '../utils/ws.js'
import { createSystemMessage } from '../protocal/message.js'
import { usePlayerStore } from '@/stores/player' 

const router = useRouter()
const playerStore = usePlayerStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const showDeleteDialog = ref(false)
const deleteConfirmInput = ref('')

const handleLoginMessage = (msg) => {
  if (msg.type === 'system') {
      
    // --- 登录处理 ---
    if (msg.subtype === 'login') {
      loading.value = false
      if (msg.flag) {
        const userId = msg.results.userId
        const reconnectToken = msg.results.reconnectToken || null
        
        playerStore.setUserId(userId) 
        playerStore.setReconnectToken(reconnectToken)
        router.push({ name: 'CharacterSelect' }) 
        
      } else {
        error.value = `登录失败: ${msg.results.error || '用户名或密码错误'}`
      }
    }
    // --- 账号删除处理 ---
    else if (msg.subtype === 'delete_account') {
      loading.value = false
      if (msg.flag) {
        error.value = ''
        showDeleteDialog.value = false
        deleteConfirmInput.value = ''
        username.value = ''
        password.value = ''
        alert('账号已成功注销，所有相关数据已删除。')
        // 清空本地数据并返回登录
        playerStore.clearAuth()
        router.push({ name: 'Login' })
      } else {
        error.value = `注销失败: ${msg.results.error || '未知错误'}`
      }
    }
  }
}

onMounted(() => {
  initWebSocket(handleLoginMessage)
})

onUnmounted(() => {
  removeMessageHandler(handleLoginMessage)
})

const login = () => {
  error.value = ''
  if (!username.value || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }
  loading.value = true

  sendMessage(createSystemMessage('login', {
    username: username.value,
    password: password.value
  }))
}

const goToRegister = () => {
  router.push({ name: 'Register' })
}

const openDeleteDialog = () => {
  if (!username.value) {
    error.value = '请先输入用户名'
    return
  }
  showDeleteDialog.value = true
  deleteConfirmInput.value = ''
}

const closeDeleteDialog = () => {
  showDeleteDialog.value = false
  deleteConfirmInput.value = ''
}

const confirmDelete = () => {
  if (deleteConfirmInput.value !== username.value) {
    error.value = '用户名不匹配，请重试'
    return
  }
  
  loading.value = true
  error.value = ''

  sendMessage(createSystemMessage('delete_account', {
    username: username.value,
    password: password.value
  }))
}
</script>

<style scoped>
.login-container {
  width: 350px;
  margin: 60px auto;
  padding: 30px;
  background-color: var(--color-background-dark);
  color: var(--color-text-light);
  border: 4px solid var(--color-accent-gold);
  border-radius: 8px;
  box-shadow: 0 0 15px rgba(0, 0, 0, 0.5);
  text-align: center;
}

h2 {
  color: var(--color-accent-gold);
  margin-bottom: 25px;
  border-bottom: 2px solid var(--color-accent-gold);
  padding-bottom: 10px;
}

.form-group {
  margin-bottom: 20px;
  text-align: left;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
  font-weight: bold;
  color: var(--color-text-light);
}

.form-group input {
  width: 100%;
  padding: 10px;
  border: 1px solid var(--color-accent-gold);
  background-color: #1a1a1a;
  color: var(--color-text-light);
  border-radius: 4px;
  font-size: 14px;
}

.form-group input:focus {
  outline: none;
  border-color: var(--color-accent-blue);
  box-shadow: 0 0 5px rgba(100, 150, 200, 0.3);
}

.buttons {
  display: flex;
  justify-content: center;
  gap: 10px;
  margin-top: 25px;
}

.btn-login {
  flex: 1;
  background-color: var(--color-accent-gold);
  color: var(--color-background-dark);
  padding: 12px 20px;
  font-weight: bold;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: filter 0.3s;
}

.btn-login:hover:not(:disabled) {
  filter: brightness(1.2);
}

.btn-login:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-register {
  flex: 1;
  background-color: var(--color-accent-blue);
  color: var(--color-text-light);
  padding: 12px 20px;
  font-weight: bold;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: filter 0.3s;
}

.btn-register:hover:not(:disabled) {
  filter: brightness(1.2);
}

.btn-register:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.error-msg {
  color: #e74c3c;
  margin-top: 15px;
  font-weight: bold;
}

.logout-section {
  margin-top: 30px;
  padding-top: 20px;
  border-top: 1px dashed var(--color-border-subtle);
}

.logout-hint {
  color: var(--color-text-light);
  font-size: 12px;
  margin-bottom: 10px;
}

.btn-delete-account {
  background-color: #c0392b;
  color: var(--color-text-light);
  padding: 10px 20px;
  font-weight: bold;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: filter 0.3s;
  width: 100%;
}

.btn-delete-account:hover {
  filter: brightness(1.2);
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

.confirm-text {
  margin-top: 20px;
  font-weight: bold;
}

.confirm-input {
  width: 100%;
  padding: 10px;
  margin: 10px 0 20px 0;
  border: 1px solid var(--color-accent-blue);
  background-color: #1a1a1a;
  color: var(--color-text-light);
  border-radius: 4px;
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
