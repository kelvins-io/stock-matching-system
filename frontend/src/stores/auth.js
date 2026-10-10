import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import http from '../api/http'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))

  const isLogin = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function setSession(t, u) {
    token.value = t
    user.value = u
    localStorage.setItem('token', t)
    localStorage.setItem('user', JSON.stringify(u))
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  async function login(username, password) {
    const data = await http.post('/auth/login', { username, password })
    setSession(data.token, data.user)
    return data
  }

  async function register(username, password) {
    const data = await http.post('/auth/register', { username, password })
    setSession(data.token, data.user)
    return data
  }

  async function refreshMe() {
    if (!token.value) return
    const data = await http.get('/auth/me')
    user.value = data
    localStorage.setItem('user', JSON.stringify(data))
  }

  return { token, user, isLogin, isAdmin, login, register, logout, refreshMe, setSession }
})
