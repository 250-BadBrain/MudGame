import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    proxy: {
      '/ws': {
        target: 'ws://localhost:4009',
        ws: true,
        changeOrigin: true
      }
    },
    // 允许 ngrok 等内网穿透工具的 host
    allowedHosts: true 
  }
})
