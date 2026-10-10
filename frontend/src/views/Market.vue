<template>
  <div class="page">
    <div class="stat-grid">
      <div class="stat-item">
        <div class="label">成交额</div>
        <div class="value">{{ formatMoney(summary.total_amount) }}</div>
      </div>
      <div class="stat-item">
        <div class="label">成交量</div>
        <div class="value">{{ summary.total_volume || 0 }}</div>
      </div>
      <div class="stat-item">
        <div class="label">成交笔数</div>
        <div class="value">{{ summary.trade_count || 0 }}</div>
      </div>
      <div class="stat-item">
        <div class="label">涨 / 平 / 跌</div>
        <div class="value">
          <span class="up">{{ summary.up_count || 0 }}</span>
          /
          <span class="flat">{{ summary.flat_count || 0 }}</span>
          /
          <span class="down">{{ summary.down_count || 0 }}</span>
        </div>
      </div>
    </div>

    <div class="card" style="margin-bottom:16px">
      <div class="kline-head">
        <el-select v-model="klineStockId" placeholder="选择股票" style="width: 220px" filterable>
          <el-option
            v-for="q in quotes"
            :key="q.id"
            :label="`${q.code} ${q.name}`"
            :value="q.id"
          />
        </el-select>
      </div>
      <KlineChart :stock-id="klineStockId" :title="klineTitle" height="420px" />
    </div>

    <el-row :gutter="16">
      <el-col :span="14">
        <div class="card">
          <div class="card-title">当日分时成交量</div>
          <div ref="chartRef" class="chart"></div>
        </div>
      </el-col>
      <el-col :span="10">
        <div class="card">
          <div class="card-title">涨跌分布</div>
          <div ref="pieRef" class="chart"></div>
        </div>
      </el-col>
    </el-row>

    <div class="card" style="margin-top:16px">
      <div class="card-title">行情列表</div>
      <el-table
        ref="quoteTableRef"
        :data="quotes"
        row-key="id"
        stripe
        height="420"
        highlight-current-row
        @row-click="onQuoteClick"
      >
        <el-table-column prop="code" label="代码" width="100" />
        <el-table-column prop="name" label="名称" width="120" />
        <el-table-column label="最新价" width="110">
          <template #default="{ row }">
            <span :class="chgClass(row.change_pct)">{{ row.current_price?.toFixed(2) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="涨跌幅" width="110">
          <template #default="{ row }">
            <span :class="chgClass(row.change_pct)">{{ row.change_pct?.toFixed(2) }}%</span>
          </template>
        </el-table-column>
        <el-table-column prop="prev_close" label="昨收" width="100" :formatter="priceFmt" />
        <el-table-column prop="high_price" label="最高" width="100" :formatter="priceFmt" />
        <el-table-column prop="low_price" label="最低" width="100" :formatter="priceFmt" />
        <el-table-column prop="volume" label="成交量" />
        <el-table-column label="成交额">
          <template #default="{ row }">{{ formatMoney(row.amount) }}</template>
        </el-table-column>
        <el-table-column label="涨停/跌停" width="150">
          <template #default="{ row }">
            <span class="up">{{ row.limit_up?.toFixed(2) }}</span>
            /
            <span class="down">{{ row.limit_down?.toFixed(2) }}</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import * as echarts from 'echarts'
import http from '../api/http'
import { connectMarketStream } from '../api/marketStream'
import KlineChart from '../components/KlineChart.vue'

const quotes = ref([])
const klineStockId = ref('')
const quoteTableRef = ref()
const summary = reactive({
  total_amount: 0,
  total_volume: 0,
  trade_count: 0,
  up_count: 0,
  down_count: 0,
  flat_count: 0,
  hourly_volume: []
})
const chartRef = ref()
const pieRef = ref()
const streamStatus = ref('')
let chart, pie, disconnectStream, summaryTimer

const klineTitle = computed(() => {
  const q = quotes.value.find((i) => i.id === klineStockId.value)
  return q ? `${q.code} ${q.name} K线` : 'K线图'
})

async function restoreQuoteRow() {
  if (!klineStockId.value) return
  const row = quotes.value.find((q) => q.id === klineStockId.value)
  if (!row) return
  await nextTick()
  quoteTableRef.value?.setCurrentRow(row)
}

function onQuoteClick(row) {
  if (!row) return
  klineStockId.value = row.id
}

function formatMoney(v) {
  return Number(v || 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
function priceFmt(_r, _c, v) {
  return Number(v || 0).toFixed(2)
}
function chgClass(p) {
  if (p > 0) return 'up'
  if (p < 0) return 'down'
  return 'flat'
}

function renderCharts() {
  if (!chart) chart = echarts.init(chartRef.value)
  if (!pie) pie = echarts.init(pieRef.value)

  const hours = (summary.hourly_volume || []).map((i) => i.hour)
  const vols = (summary.hourly_volume || []).map((i) => i.volume)
  const amts = (summary.hourly_volume || []).map((i) => i.amount)

  chart.setOption({
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis' },
    legend: { data: ['成交量', '成交额'], textStyle: { color: '#6b7280' } },
    grid: { left: 50, right: 50, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: hours, axisLabel: { color: '#6b7280' } },
    yAxis: [
      { type: 'value', name: '量', axisLabel: { color: '#6b7280' }, splitLine: { lineStyle: { color: '#e6eaf2' } } },
      { type: 'value', name: '额', axisLabel: { color: '#6b7280' }, splitLine: { show: false } }
    ],
    series: [
      { name: '成交量', type: 'bar', data: vols, itemStyle: { color: '#3b6dff' } },
      { name: '成交额', type: 'line', yAxisIndex: 1, data: amts, itemStyle: { color: '#2563eb' }, smooth: true }
    ]
  })

  pie.setOption({
    backgroundColor: 'transparent',
    tooltip: { trigger: 'item' },
    series: [{
      type: 'pie',
      radius: ['40%', '68%'],
      label: { color: '#1f2a37' },
      data: [
        { name: '上涨', value: summary.up_count || 0, itemStyle: { color: '#f05656' } },
        { name: '平盘', value: summary.flat_count || 0, itemStyle: { color: '#94a3b8' } },
        { name: '下跌', value: summary.down_count || 0, itemStyle: { color: '#12b886' } }
      ]
    }]
  })
}

async function loadSummary() {
  const s = await http.get('/admin/market/summary')
  Object.assign(summary, s || {})
  renderCharts()
}

async function loadQuotesOnce() {
  quotes.value = (await http.get('/stocks/quotes')) || []
  if (!klineStockId.value && quotes.value.length) {
    klineStockId.value = quotes.value[0].id
  }
  await restoreQuoteRow()
}

function patchQuote(q) {
  if (!q?.id) return
  const idx = quotes.value.findIndex((i) => i.id === q.id)
  if (idx >= 0) {
    const next = quotes.value.slice()
    next[idx] = { ...next[idx], ...q }
    quotes.value = next
  } else {
    quotes.value = [...quotes.value, q]
  }
  restoreQuoteRow()
}

async function load() {
  await Promise.all([loadQuotesOnce(), loadSummary()])
}

onMounted(async () => {
  await load()
  disconnectStream = connectMarketStream({
    onStatus: (s) => { streamStatus.value = s },
    onTick: (payload) => {
      if (payload?.quote) patchQuote(payload.quote)
      // 有成交时刷新汇总图
      if (payload?.trades?.length) loadSummary()
    }
  })
  // 汇总兜底低频刷新（无成交时涨跌分布仍可更新）
  summaryTimer = setInterval(loadSummary, 30000)
  window.addEventListener('resize', () => {
    chart?.resize()
    pie?.resize()
  })
})

onUnmounted(() => {
  disconnectStream?.()
  clearInterval(summaryTimer)
  chart?.dispose()
  pie?.dispose()
})
</script>

<style scoped>
.card-title {
  font-weight: 700;
  margin-bottom: 10px;
}
.kline-head {
  margin-bottom: 8px;
}
.chart {
  height: 300px;
}
</style>
