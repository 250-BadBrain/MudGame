<template>
  <div class="container">
    <h2>WebSocket 测试</h2>
    <input v-model="inputMsg" placeholder="输入消息..." />
    <button @click="sendMessage">发送</button>

    <h3>消息记录：</h3>
    <ul>
      <li v-for="(msg, index) in messages" :key="index">{{ msg }}</li>
    </ul>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const inputMsg = ref('')
const messages = ref([])
let ws = null

onMounted(() => {
  // 连接后端 WebSocket
  ws = new WebSocket('ws://localhost:8080/ws')

  ws.onopen = () => {
    messages.value.push('✅ 已连接服务器')
  }

  ws.onmessage = (event) => {
    messages.value.push('📩 收到: ' + event.data)
  }

  ws.onerror = (error) => {
    console.error('WebSocket 错误:', error)
    messages.value.push('❌ WebSocket 出错')
  }

  ws.onclose = () => {
    messages.value.push('🔌 连接已关闭')
  }
})

function sendMessage() {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(inputMsg.value)
    messages.value.push('➡️ 发送: ' + inputMsg.value)
    inputMsg.value = ''
  } else {
    messages.value.push('⚠️ 连接未建立')
  }
}
</script>

<style scoped>
.container {
  padding: 20px;
  font-family: Arial, sans-serif;
}
input {
  padding: 5px;
  margin-right: 10px;
}
button {
  padding: 5px 10px;
}
ul {
  margin-top: 10px;
  padding: 0;
  list-style: none;
}
li {
  margin: 3px 0;
}
</style>
