<template>
  <div class="page">
    <el-row :gutter="16">
      <el-col :span="14">
        <div class="card">
          <div class="card-title">选择股票</div>
          <el-table :data="quotes" highlight-current-row @current-change="onSelect" height="520">
            <el-table-column prop="code" label="代码" width="90" />
            <el-table-column prop="name" label="名称" width="110" />
            <el-table-column label="最新价" width="100">
              <template #default="{ row }">
                <span :class="chgClass(row.change_pct)">{{ row.current_price?.toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="涨跌幅" width="100">
              <template #default="{ row }">
                <span :class="chgClass(row.change_pct)">{{ row.change_pct?.toFixed(2) }}%</span>
              </template>
            </el-table-column>
            <el-table-column label="涨停" width="90">
              <template #default="{ row }"><span class="up">{{ row.limit_up?.toFixed(2) }}</span></template>
            </el-table-column>
            <el-table-column label="跌停" width="90">
              <template #default="{ row }"><span class="down">{{ row.limit_down?.toFixed(2) }}</span></template>
            </el-table-column>
            <el-table-column prop="volume" label="成交量" />
          </el-table>
        </div>
      </el-col>
      <el-col :span="10">
        <div class="card">
          <div class="card-title">限价委托 {{ selected ? `- ${selected.code} ${selected.name}` : '' }}</div>
          <el-form label-width="80px" :disabled="!selected">
            <el-form-item label="方向">
              <el-radio-group v-model="form.side">
                <el-radio-button label="buy">买入</el-radio-button>
                <el-radio-button label="sell">卖出</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="价格">
              <el-input-number v-model="form.price" :min="0.01" :step="0.01" :precision="2" style="width:100%" />
            </el-form-item>
            <el-form-item label="数量">
              <el-input-number v-model="form.quantity" :min="100" :step="100" style="width:100%" />
              <div class="tip">须为 100 股整数倍</div>
            </el-form-item>
            <el-form-item label="预估金额">
              <span class="amt">¥ {{ ((form.price || 0) * (form.quantity || 0)).toFixed(2) }}</span>
            </el-form-item>
            <el-button
              :type="form.side === 'buy' ? 'danger' : 'success'"
              style="width:100%"
              :loading="loading"
              @click="submit"
            >
              {{ form.side === 'buy' ? '买入' : '卖出' }}
            </el-button>
          </el-form>
        </div>

        <div class="card" style="margin-top:16px">
          <div class="card-title">五档盘口</div>
          <el-row :gutter="12">
            <el-col :span="12">
              <div class="depth-title up">买盘</div>
              <div v-for="(b, i) in book.bids" :key="'b'+i" class="depth-row">
                <span>{{ b.price?.toFixed(2) }}</span>
                <span>{{ b.quantity }}</span>
              </div>
              <div v-if="!book.bids?.length" class="empty">暂无</div>
            </el-col>
            <el-col :span="12">
              <div class="depth-title down">卖盘</div>
              <div v-for="(a, i) in book.asks" :key="'a'+i" class="depth-row">
                <span>{{ a.price?.toFixed(2) }}</span>
                <span>{{ a.quantity }}</span>
              </div>
              <div v-if="!book.asks?.length" class="empty">暂无</div>
            </el-col>
          </el-row>
        </div>

        <div class="card" style="margin-top:16px">
          <div class="card-title">我的持仓</div>
          <el-table :data="holdings" size="small" height="180">
            <el-table-column prop="stock_code" label="代码" width="80" />
            <el-table-column prop="stock_name" label="名称" width="90" />
            <el-table-column prop="quantity" label="持仓" width="70" />
            <el-table-column label="可用" width="70">
              <template #default="{ row }">{{ row.quantity - row.frozen_qty }}</template>
            </el-table-column>
            <el-table-column label="成本">
              <template #default="{ row }">{{ row.avg_cost?.toFixed(2) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const quotes = ref([])
const holdings = ref([])
const selected = ref(null)
const loading = ref(false)
const book = reactive({ bids: [], asks: [] })
const form = reactive({ side: 'buy', price: 0, quantity: 100 })
let timer

function chgClass(p) {
  if (p > 0) return 'up'
  if (p < 0) return 'down'
  return 'flat'
}

async function loadQuotes() {
  quotes.value = await http.get('/stocks/quotes')
}

async function loadHoldings() {
  holdings.value = await http.get('/holdings')
}

async function loadBook() {
  if (!selected.value) return
  const data = await http.get(`/stocks/${selected.value.id}/orderbook`)
  book.bids = data.bids || []
  book.asks = data.asks || []
}

function onSelect(row) {
  selected.value = row
  if (row) {
    form.price = Number(row.current_price.toFixed(2))
    loadBook()
  }
}

async function submit() {
  if (!selected.value) return
  loading.value = true
  try {
    await http.post('/orders', {
      stock_id: selected.value.id,
      side: form.side,
      price: form.price,
      quantity: form.quantity
    })
    ElMessage.success('委托已提交')
    await Promise.all([auth.refreshMe(), loadQuotes(), loadHoldings(), loadBook()])
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadQuotes(), loadHoldings()])
  timer = setInterval(async () => {
    await loadQuotes()
    await loadBook()
  }, 4000)
})

onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.card-title {
  font-weight: 700;
  margin-bottom: 12px;
}
.tip {
  color: var(--muted);
  font-size: 12px;
  margin-top: 4px;
}
.amt {
  color: #2563eb;
  font-weight: 700;
}
.depth-title {
  font-weight: 600;
  margin-bottom: 8px;
}
.depth-row {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
  border-bottom: 1px dashed #e6eaf2;
  font-size: 13px;
}
.empty {
  color: var(--muted);
  font-size: 13px;
}
</style>
