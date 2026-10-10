<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-brand">STOCK MATCHING</div>
      <h1>{{ tab === 'login' ? '登录' : '注册' }}</h1>
      <p class="auth-hint">
        {{ tab === 'login' ? '使用账号密码登录撮合交易系统' : '注册普通用户，初始资金 100 万' }}
      </p>
      <el-tabs v-model="tab" class="auth-tabs">
        <el-tab-pane label="登录" name="login">
          <el-form :model="form" @submit.prevent="onLogin" label-position="top">
            <el-form-item label="用户名" required>
              <el-input v-model="form.username" placeholder="用户名" autocomplete="username" />
            </el-form-item>
            <el-form-item label="密码" required>
              <el-input
                v-model="form.password"
                type="password"
                show-password
                placeholder="密码"
                autocomplete="current-password"
              />
            </el-form-item>
            <el-button type="primary" class="auth-submit" :loading="loading" @click="onLogin">
              登录
            </el-button>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="注册" name="register">
          <el-form :model="form" @submit.prevent="onRegister" label-position="top">
            <el-form-item label="用户名" required>
              <el-input v-model="form.username" placeholder="至少 3 位" autocomplete="username" />
            </el-form-item>
            <el-form-item label="密码" required>
              <el-input
                v-model="form.password"
                type="password"
                show-password
                placeholder="至少 6 位"
                autocomplete="new-password"
              />
            </el-form-item>
            <el-button type="primary" class="auth-submit" :loading="loading" @click="onRegister">
              注册并登录
            </el-button>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const tab = ref('login')
const loading = ref(false)
const form = reactive({ username: '', password: '' })

async function onLogin() {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    ElMessage.success('登录成功')
    router.push(auth.isAdmin ? '/' : '/trade')
  } finally {
    loading.value = false
  }
}

async function onRegister() {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await auth.register(form.username, form.password)
    ElMessage.success('注册成功')
    router.push('/trade')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  --bg: #f4f6fb;
  --panel: #ffffff;
  --line: #e6eaf2;
  --text: #1f2a37;
  --muted: #6b7280;
  --brand: #3b6dff;

  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  position: relative;
  color: var(--text);
  background:
    radial-gradient(1200px 600px at 10% -10%, rgba(59, 109, 255, 0.18), transparent 55%),
    radial-gradient(900px 500px at 100% 0%, rgba(17, 24, 39, 0.08), transparent 50%),
    var(--bg);
}

.auth-card {
  width: 100%;
  max-width: 400px;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 16px;
  padding: 28px 28px 24px;
  box-shadow: 0 12px 40px rgba(17, 24, 39, 0.06);
}

.auth-brand {
  color: var(--brand);
  font-weight: 700;
  letter-spacing: 0.04em;
  margin-bottom: 12px;
}

.auth-card h1 {
  margin: 0 0 6px;
  font-size: 24px;
}

.auth-hint {
  margin: 0 0 12px;
  color: var(--muted);
  font-size: 14px;
}

.auth-tabs :deep(.el-tabs__header) {
  margin-bottom: 12px;
}

.auth-submit {
  width: 100%;
}
</style>
