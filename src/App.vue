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
    // 🔧 修复：只初始化 store，不自动恢复 localStorage 中的登陆状态
    // 原因：
    // 1. 刚打开应用时，应该显示登录界面，而不是自动进入角色选择
    // 2. localStorage 的恢复应该只在明确需要时进行（如浏览器刷新）
    // 3. 路由守卫会根据 store 中的 userId 和 playerId 自动处理导航
    
    const playerStore = usePlayerStore()
    
    console.log('[App.vue] Mounted')
    console.log('[App.vue] Initial store state - userId:', playerStore.userId, 'playerId:', playerStore.playerId)
    
    // 短暂延迟确保 store 初始化完成
    await new Promise(resolve => setTimeout(resolve, 100))
    
    isReady.value = true
    console.log('[App.vue] Ready to render')
    
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
