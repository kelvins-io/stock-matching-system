<template>
  <div class="page">
    <el-row :gutter="16">
      <el-col :span="14">
        <div class="card">
          <div class="head">
            <div class="card-title">我的委托</div>
            <el-button size="small" @click="load">刷新</el-button>
          </div>
          <el-table :data="orders" height="560">
            <el-table-column prop="stock_code" label="代码" width="90" />
            <el-table-column prop="stock_name" label="名称" width="100" />
            <el-table-column label="方向" width="70">
              <template #default="{ row }">
                <span :class="row.side === 'buy' ? 'up' : 'down'">{{ row.side === 'buy' ? '买' : '卖' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="价格" width="90">
              <template #default="{ row }">{{ row.price?.toFixed(2) }}</template>
            </el-table-column>
            <el-table-column prop="quantity" label="数量" width="80" />
            <el-table-column prop="filled_qty" label="成交" width="80" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">{{ statusText(row.status) }}</template>
            </el-table-column>
            <el-table-column label="时间" min-width="160">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button
                  v-if="row.status === 'pending' || row.status === 'partial'"
                  link
                  type="danger"
                  @click="cancel(row)"
                >撤单</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
      <el-col :span="10">
        <div class="card">
          <div class="card-title">我的成交</div>
          <el-table :data="trades" height="560">
            <el-table-column prop="stock_code" label="代码" width="90" />
            <el-table-column label="价格" width="90">
              <template #default="{ row }">{{ row.price?.toFixed(2) }}</template>
            </el-table-column>
            <el-table-column prop="quantity" label="数量" width="80" />
            <el-table-column label="金额" width="110">
              <template #default="{ row }">{{ row.amount?.toFixed(2) }}</template>
            </el-table-column>
            <el-table-column label="时间">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const orders = ref([])
const trades = ref([])

const statusMap = {
  pending: '待成交',
  partial: '部分成交',
  filled: '全部成交',
  cancelled: '已撤销',
  rejected: '已拒绝'
}

function statusText(s) {
  return statusMap[s] || s
}

function formatTime(t) {
  if (!t) return ''
  return new Date(t).toLocaleString('zh-CN')
}

async function load() {
  const [o, t] = await Promise.all([http.get('/orders'), http.get('/trades')])
  orders.value = o || []
  trades.value = t || []
}

async function cancel(row) {
  await ElMessageBox.confirm('确认撤销该委托？', '撤单', { type: 'warning' })
  await http.delete(`/orders/${row.id}`)
  ElMessage.success('已撤单')
  await Promise.all([load(), auth.refreshMe()])
}

onMounted(load)
</script>

<style scoped>
.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.card-title {
  font-weight: 700;
}
</style>
