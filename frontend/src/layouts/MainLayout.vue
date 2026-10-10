<template>
  <el-container class="layout">
    <el-header class="header">
      <div class="brand">
        <span class="logo">SMS</span>
        <div>
          <div class="title">A股撮合交易系统</div>
          <div class="sub">Stock Matching System</div>
        </div>
      </div>
      <el-menu
        mode="horizontal"
        :default-active="active"
        router
        class="nav"
        background-color="transparent"
        text-color="#3b4a63"
        active-text-color="#3b6dff"
      >
        <el-menu-item index="/">交易大盘</el-menu-item>
        <el-menu-item index="/trade">买卖下单</el-menu-item>
        <el-menu-item index="/orders">我的委托</el-menu-item>
        <el-menu-item v-if="auth.isAdmin" index="/admin">系统管理</el-menu-item>
      </el-menu>
      <div class="user">
        <el-tag :type="marketOpen ? 'success' : 'info'" size="small" class="mkt">
          {{ marketOpen ? '开市中' : '休市' }}
        </el-tag>
        <template v-if="auth.isLogin">
          <span class="uname">{{ auth.user?.username }}</span>
          <span class="bal">¥ {{ formatMoney(auth.user?.balance) }}</span>
          <el-button size="small" @click="refresh">刷新</el-button>
          <el-button size="small" text type="danger" @click="onLogout">退出</el-button>
        </template>
        <el-button v-else type="primary" size="small" @click="$router.push('/login')">登录</el-button>
      </div>
    </el-header>
    <el-main class="main">
      <router-view />
    </el-main>
  </el-container>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const active = computed(() => route.path.startsWith('/admin') ? '/admin' : route.path)
const marketOpen = ref(false)
let timer

function formatMoney(v) {
  return Number(v || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

async function loadStatus() {
  try {
    const st = await http.get('/market/status')
    marketOpen.value = !!st.is_open
  } catch (_) {}
}

async function refresh() {
  if (auth.isLogin) await auth.refreshMe()
  await loadStatus()
}

function onLogout() {
  auth.logout()
  router.push('/login')
}

onMounted(() => {
  refresh()
  timer = setInterval(loadStatus, 5000)
})

onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.layout {
  min-height: 100vh;
  background: transparent;
}
.header {
  height: 64px !important;
  display: flex;
  align-items: center;
  gap: 20px;
  border-bottom: 1px solid #e2e8f0;
  background: #fff;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 220px;
}
.logo {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  font-weight: 800;
  color: #fff;
  background: linear-gradient(135deg, #3b6dff, #2563eb);
}
.title {
  font-weight: 700;
  line-height: 1.2;
  color: #1f2a37;
}
.sub {
  font-size: 12px;
  color: #6b7280;
}
.nav {
  flex: 1;
  border-bottom: none !important;
}
.user {
  display: flex;
  align-items: center;
  gap: 10px;
  white-space: nowrap;
}
.uname {
  color: #334155;
}
.bal {
  color: #2563eb;
  font-weight: 600;
}
.main {
  padding: 0;
  background: transparent;
}
:deep(.el-menu--horizontal > .el-menu-item.is-active) {
  border-bottom-color: #3b6dff !important;
  font-weight: 600;
}
:deep(.el-menu--horizontal > .el-menu-item:hover) {
  background: rgba(59, 109, 255, 0.08) !important;
}
</style>
