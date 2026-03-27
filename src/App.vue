<template>
  <div v-if="!isReady" class="loading-screen">
    <div class="loading-content">
      <div class="spinner"></div>
      <p>{{ loadingMessage }}</p>
      <div v-if="error" class="error-message">
        ⚠️ {{ error }}
      </div>
    </div>
  </div>
  <router-view v-else />
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { usePlayerStore } from '@/stores/player'

const isReady = ref(false)
const loadingMessage = ref('正在加载应用...')
const error = ref('')

onMounted(async () => {
  try {
    usePlayerStore()
    isReady.value = true
  } catch (e) {
    console.error('[App.vue] Error during initialization:', e)
    error.value = '应用初始化失败: ' + e.message
    isReady.value = true
  }
})
</script>

<style scoped>
.loading-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100vw;
  height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
}

.loading-content {
  text-align: center;
  color: #fff;
}

.spinner {
  width: 50px;
  height: 50px;
  margin: 0 auto 20px;
  border: 4px solid rgba(212, 175, 55, 0.3);
  border-top-color: #d4af37;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

p {
  font-size: 16px;
  margin: 0;
  color: #d4af37;
}

.error-message {
  margin-top: 20px;
  padding: 10px;
  background: rgba(244, 67, 54, 0.2);
  border: 1px solid #f44336;
  color: #ffcdd2;
  border-radius: 4px;
  font-size: 14px;
}
</style>
