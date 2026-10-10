<template>
  <div class="page">
    <el-row :gutter="16">
      <el-col :span="12">
        <div class="card">
          <div class="card-title">发行股票</div>
          <el-form label-width="100px">
            <el-form-item label="股票代码">
              <el-input v-model="issue.code" placeholder="如 601318" />
            </el-form-item>
            <el-form-item label="股票名称">
              <el-input v-model="issue.name" placeholder="如 中国平安" />
            </el-form-item>
            <el-form-item label="总股本">
              <el-input-number v-model="issue.total_shares" :min="1000" :step="1000" style="width:100%" />
            </el-form-item>
            <el-form-item label="发行价">
              <el-input-number v-model="issue.issue_price" :min="0.01" :step="0.01" :precision="2" style="width:100%" />
            </el-form-item>
            <el-form-item label="涨跌幅%">
              <el-input-number v-model="issue.price_limit_pct" :min="1" :max="100" :step="1" style="width:100%" />
            </el-form-item>
            <el-button type="primary" :loading="issuing" @click="doIssue">发行</el-button>
          </el-form>
        </div>
      </el-col>
      <el-col :span="12">
        <div class="card">
          <div class="card-title">开市 / 闭市规则</div>
          <el-form label-width="110px">
            <el-form-item label="上午开盘">
              <el-input v-model="rule.open_am" placeholder="09:30" />
            </el-form-item>
            <el-form-item label="上午收盘">
              <el-input v-model="rule.close_am" placeholder="11:30" />
            </el-form-item>
            <el-form-item label="下午开盘">
              <el-input v-model="rule.open_pm" placeholder="13:00" />
            </el-form-item>
            <el-form-item label="下午收盘">
              <el-input v-model="rule.close_pm" placeholder="15:00" />
            </el-form-item>
            <el-form-item label="交易日">
              <el-input v-model="rule.trading_days" placeholder="1,2,3,4,5 (周一到周五)" />
            </el-form-item>
            <el-form-item label="强制开市">
              <el-switch v-model="rule.force_open" />
            </el-form-item>
            <el-form-item label="强制休市">
              <el-switch v-model="rule.force_closed" />
            </el-form-item>
            <el-button type="warning" :loading="savingRule" @click="saveRule">保存规则</el-button>
          </el-form>
        </div>
      </el-col>
    </el-row>

    <div class="card" style="margin-top:16px">
      <div class="card-title">股票管理（设置涨跌幅 / 停牌）</div>
      <el-table :data="stocks" height="420">
        <el-table-column prop="code" label="代码" width="100" />
        <el-table-column prop="name" label="名称" width="120" />
        <el-table-column label="现价" width="100">
          <template #default="{ row }">{{ row.current_price?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="涨跌幅%" width="140">
          <template #default="{ row }">
            <el-input-number v-model="row.price_limit_pct" :min="1" :max="100" size="small" />
          </template>
        </el-table-column>
        <el-table-column label="状态" width="140">
          <template #default="{ row }">
            <el-select v-model="row.status" size="small">
              <el-option label="正常" value="active" />
              <el-option label="停牌" value="suspended" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="操作">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="updateStock(row)">保存</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'

const stocks = ref([])
const issuing = ref(false)
const savingRule = ref(false)
const issue = reactive({
  code: '',
  name: '',
  total_shares: 100000000,
  issue_price: 10,
  price_limit_pct: 10
})
const rule = reactive({
  open_am: '09:30',
  close_am: '11:30',
  open_pm: '13:00',
  close_pm: '15:00',
  trading_days: '1,2,3,4,5',
  force_open: true,
  force_closed: false
})

async function load() {
  stocks.value = await http.get('/stocks')
  const r = await http.get('/admin/market/rules')
  Object.assign(rule, r)
}

async function doIssue() {
  issuing.value = true
  try {
    await http.post('/admin/stocks', { ...issue })
    ElMessage.success('发行成功')
    issue.code = ''
    issue.name = ''
    await load()
  } finally {
    issuing.value = false
  }
}

async function saveRule() {
  savingRule.value = true
  try {
    await http.put('/admin/market/rules', { ...rule })
    ElMessage.success('规则已更新')
  } finally {
    savingRule.value = false
  }
}

async function updateStock(row) {
  await http.put(`/admin/stocks/${row.id}`, {
    name: row.name,
    price_limit_pct: row.price_limit_pct,
    status: row.status
  })
  ElMessage.success('已保存')
  await load()
}

onMounted(load)
</script>

<style scoped>
.card-title {
  font-weight: 700;
  margin-bottom: 12px;
}
</style>
