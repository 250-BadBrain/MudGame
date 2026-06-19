<template>
  <div class="register-container">
    <h2>创建新账号</h2>

    <div class="form-group">
      <label>用户名:</label>
      <input v-model="username" placeholder="请输入用户名" @keyup.enter="register" />
    </div>
    <div class="form-group">
      <label>密码:</label>
      <input type="password" v-model="password" placeholder="请输入密码" @keyup.enter="register" />
    </div>
    <div class="form-group">
      <label>确认密码:</label>
      <input type="password" v-model="confirmPassword" placeholder="请再次输入密码" @keyup.enter="register" />
    </div>

    <div class="buttons">
      <button @click="register" :disabled="loading" class="btn-register">
        {{ loading ? '注册中...' : '注册账号' }}
      </button>
      <button @click="backToLogin" :disabled="loading" class="btn-back">
        返回登录
      </button>
    </div>

    <p v-if="error" class="error-msg">{{ error }}</p>
    <p v-if="success" class="success-msg">{{ success }}</p>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { initWebSocket, removeMessageHandler, sendMessage } from '../utils/ws.js'
import { createSystemMessage } from '../protocal/message.js'

const router = useRouter()

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')
const success = ref('')

const handleRegisterMessage = (msg) => {
  if (msg.type === 'system' && msg.subtype === 'register') {
    loading.value = false
    if (msg.flag) {
      const registeredUsername = msg.results.userId || username.value
      success.value = `注册成功，用户：${registeredUsername}。请返回登录。`
      // 3秒后自动返回登录页
      setTimeout(() => {
        router.push({ name: 'Login' })
      }, 2000)
    } else {
      error.value = `注册失败: ${msg.results.error || '未知错误'}`
    }
  }
}

onMounted(() => {
  initWebSocket(handleRegisterMessage)
})

onUnmounted(() => {
  removeMessageHandler(handleRegisterMessage)
})

const register = () => {
  error.value = ''
  success.value = ''
  
  // 验证
  if (!username.value.trim()) {
    error.value = '请输入用户名'
    return
  }
  if (username.value.trim().length < 3) {
    error.value = '用户名至少 3 个字符'
    return
  }
  if (!password.value) {
    error.value = '请输入密码'
    return
  }
  if (password.value.length < 6) {
    error.value = '密码至少 6 个字符'
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = '两次输入的密码不一致'
    return
  }

  loading.value = true

  sendMessage(createSystemMessage('register', {
    username: username.value.trim(),
    password: password.value
  }))
}

const backToLogin = () => {
  router.push({ name: 'Login' })
}
</script>

<style scoped>
.register-container {
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

.btn-register {
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

.btn-register:hover:not(:disabled) {
  filter: brightness(1.2);
}

.btn-register:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-back {
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

.btn-back:hover:not(:disabled) {
  filter: brightness(1.2);
}

.btn-back:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.error-msg {
  color: #e74c3c;
  margin-top: 15px;
  font-weight: bold;
}

.success-msg {
  color: #27ae60;
  margin-top: 15px;
  font-weight: bold;
}
</style>
