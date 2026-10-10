import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/login', name: 'login', component: () => import('../views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('../layouts/MainLayout.vue'),
    children: [
      { path: '', name: 'market', component: () => import('../views/Market.vue') },
      { path: 'trade', name: 'trade', component: () => import('../views/Trade.vue'), meta: { auth: true } },
      { path: 'orders', name: 'orders', component: () => import('../views/Orders.vue'), meta: { auth: true } },
      { path: 'admin', name: 'admin', component: () => import('../views/Admin.vue'), meta: { auth: true, admin: true } }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.auth && !auth.isLogin) return '/login'
  if (to.meta.admin && !auth.isAdmin) return '/'
  if (to.name === 'login' && auth.isLogin) return '/'
  return true
})

export default router
