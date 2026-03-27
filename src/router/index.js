import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'
import GameView from '../views/GameView.vue'
import CharacterSelectView from '../views/CharacterSelectView.vue'
import CharacterCreateView from '../views/CharacterCreateView.vue'
import { usePlayerStore } from '@/stores/player' 

const routes = [
  {
    path: '/',
    name: 'Login',
    component: LoginView
  },
  {
    path: '/register',
    name: 'Register',
    component: RegisterView
  },
  {
    path: '/select-character',
    name: 'CharacterSelect',
    component: CharacterSelectView,
    meta: { requiresAuth: true, requiresNoCharacter: true }
  },
  {
    path: '/create-character',
    name: 'CharacterCreate',
    component: CharacterCreateView,
    meta: { requiresAuth: true, requiresNoCharacter: true }
  },
  {
    path: '/game/:playerId',
    name: 'Game',
    component: GameView,
    props: true,
    meta: { requiresAuth: true, requiresCharacter: true } 
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 添加全局前置导航守卫
router.beforeEach((to, from, next) => {
  const playerStore = usePlayerStore()

  // 1. 检查是否已登录 (认证级别)
  if (!playerStore.isLoggedIn) {
    // 未登录：只能访问 Login 或 Register 路由
    if (to.name !== 'Login' && to.name !== 'Register') {
      next({ name: 'Login' })
    } else {
      next()
    }
    return
  }

  // 2. 已登录，检查角色选择状态 (游戏级别)
  if (playerStore.isLoggedIn) {
    if (!playerStore.isCharacterSelected) {
      // 未选角色：只能访问 CharacterSelect 路由
      if (to.name !== 'CharacterSelect' && to.name !== 'Login' && to.name !== 'CharacterCreate') {
        next({ name: 'CharacterSelect' })
      } else if (to.name === 'Login') {
        // 已登录，但跳转到 Login，重定向到角色选择
        next({ name: 'CharacterSelect' })
      } else {
        next()
      }
    } else {
      // 已选角色：只能访问 Game 路由
      if (to.name !== 'Game') {
        next({ name: 'Game', params: { playerId: playerStore.playerId } })
      } else {
        next()
      }
    }
    return
  }
  
  next()
})

export default router