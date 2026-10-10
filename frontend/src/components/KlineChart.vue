<template>
  <div class="kline-wrap">
    <div class="kline-toolbar">
      <span class="kline-title">{{ title }}</span>
      <div class="toolbar-right">
        <el-tag v-if="live" size="small" type="success" effect="plain">实时</el-tag>
        <el-radio-group v-model="interval" size="small" @change="reload">
          <el-radio-button label="1m">1分</el-radio-button>
          <el-radio-button label="5m">5分</el-radio-button>
          <el-radio-button label="15m">15分</el-radio-button>
          <el-radio-button label="1d">日K</el-radio-button>
        </el-radio-group>
      </div>
    </div>
    <div ref="chartRef" class="kline-chart" :style="{ height }"></div>
    <div v-if="!stockId" class="kline-empty">请选择股票查看 K 线</div>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import http from '../api/http'
import { connectMarketStream } from '../api/marketStream'

const props = defineProps({
  stockId: { type: String, default: '' },
  title: { type: String, default: 'K线图' },
  height: { type: String, default: '360px' },
  defaultInterval: { type: String, default: '1d' }
})

const chartRef = ref()
const interval = ref(props.defaultInterval)
const live = ref(false)
const bars = ref([])
let chart
let disconnectStream

function formatAxis(ts, iv) {
  const d = new Date(ts)
  if (iv === '1d') {
    return `${d.getMonth() + 1}/${d.getDate()}`
  }
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function render() {
  if (!chartRef.value) return
  if (!chart) chart = echarts.init(chartRef.value)
  const list = bars.value || []

  const categories = list.map((b) => formatAxis(b.open_time, interval.value))
  const candle = list.map((b) => [b.open, b.close, b.low, b.high])
  const volumes = list.map((b, i) => ({
    value: b.volume,
    itemStyle: {
      color: list[i].close >= list[i].open ? '#f05656' : '#12b886'
    }
  }))

  chart.setOption({
    animation: false,
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' },
      formatter(params) {
        const p = params.find((x) => x.seriesType === 'candlestick')
        if (!p) return ''
        const [o, c, l, h] = p.data
        const vol = params.find((x) => x.seriesName === '成交量')
        return [
          p.axisValue,
          `开：${Number(o).toFixed(2)}`,
          `高：${Number(h).toFixed(2)}`,
          `低：${Number(l).toFixed(2)}`,
          `收：${Number(c).toFixed(2)}`,
          vol ? `量：${vol.data.value ?? vol.data}` : ''
        ].filter(Boolean).join('<br/>')
      }
    },
    axisPointer: { link: [{ xAxisIndex: 'all' }] },
    grid: [
      { left: 56, right: 20, top: 24, height: '58%' },
      { left: 56, right: 20, top: '76%', height: '16%' }
    ],
    xAxis: [
      {
        type: 'category',
        data: categories,
        boundaryGap: true,
        axisLine: { lineStyle: { color: '#cbd5e1' } },
        axisLabel: { color: '#6b7280', hideOverlap: true },
        min: 'dataMin',
        max: 'dataMax'
      },
      {
        type: 'category',
        gridIndex: 1,
        data: categories,
        boundaryGap: true,
        axisLabel: { show: false },
        axisTick: { show: false },
        axisLine: { show: false }
      }
    ],
    yAxis: [
      {
        scale: true,
        axisLabel: { color: '#6b7280' },
        splitLine: { lineStyle: { color: '#e6eaf2' } }
      },
      {
        scale: true,
        gridIndex: 1,
        splitNumber: 2,
        axisLabel: { show: false },
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { show: false }
      }
    ],
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: 40, end: 100 },
      { type: 'slider', xAxisIndex: [0, 1], start: 40, end: 100, height: 18, bottom: 4 }
    ],
    series: [
      {
        name: 'K线',
        type: 'candlestick',
        data: candle,
        itemStyle: {
          color: '#f05656',
          color0: '#12b886',
          borderColor: '#f05656',
          borderColor0: '#12b886'
        }
      },
      {
        name: '成交量',
        type: 'bar',
        xAxisIndex: 1,
        yAxisIndex: 1,
        data: volumes
      }
    ]
  }, true)
}

function applyKlineBar(bar) {
  if (!bar || !props.stockId) return
  if (bar.stock_id && bar.stock_id !== props.stockId) return
  if (bar.interval && bar.interval !== interval.value) return

  const list = bars.value.slice()
  const key = new Date(bar.open_time).getTime()
  const idx = list.findIndex((b) => new Date(b.open_time).getTime() === key)
  if (idx >= 0) {
    list[idx] = { ...list[idx], ...bar }
  } else {
    list.push(bar)
    list.sort((a, b) => new Date(a.open_time) - new Date(b.open_time))
  }
  bars.value = list
  render()
}

async function reload() {
  if (!props.stockId) {
    bars.value = []
    chart?.clear()
    return
  }
  const data = await http.get(`/stocks/${props.stockId}/kline`, {
    params: { interval: interval.value, limit: interval.value === '1d' ? 120 : 200 }
  })
  bars.value = data || []
  render()
}

function bindStream() {
  disconnectStream?.()
  disconnectStream = null
  if (!props.stockId) {
    live.value = false
    return
  }
  disconnectStream = connectMarketStream({
    stockId: props.stockId,
    onStatus: (s) => {
      live.value = s === 'connected' || s === 'heartbeat'
    },
    onTick: (payload) => {
      if (!payload || payload.stock_id !== props.stockId) return
      const bar = payload.klines?.[interval.value]
      if (bar) applyKlineBar(bar)
    }
  })
}

watch(() => props.stockId, async () => {
  await reload()
  bindStream()
})

onMounted(async () => {
  await reload()
  bindStream()
  window.addEventListener('resize', () => chart?.resize())
})

onUnmounted(() => {
  disconnectStream?.()
  chart?.dispose()
  chart = null
})
</script>

<style scoped>
.kline-wrap {
  position: relative;
}
.kline-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  gap: 12px;
}
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
}
.kline-title {
  font-weight: 700;
  color: #1f2a37;
}
.kline-chart {
  width: 100%;
}
.kline-empty {
  position: absolute;
  inset: 48px 0 0;
  display: grid;
  place-items: center;
  color: #6b7280;
  pointer-events: none;
}
</style>
