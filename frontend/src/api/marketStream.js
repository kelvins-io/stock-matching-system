/**
 * 行情 SSE：event = connected | heartbeat | tick
 * tick.data = { stock_id, quote, trades?, klines? }
 */
export function connectMarketStream({ stockId = '', onTick, onStatus } = {}) {
  const base = import.meta.env.VITE_API_BASE || '/api/v1'
  const qs = stockId ? `?stock_id=${encodeURIComponent(stockId)}` : ''
  const url = `${base}/market/stream${qs}`
  const es = new EventSource(url)

  es.addEventListener('connected', () => {
    onStatus?.('connected')
  })
  es.addEventListener('heartbeat', () => {
    onStatus?.('heartbeat')
  })
  es.addEventListener('tick', (ev) => {
    try {
      const msg = JSON.parse(ev.data)
      const payload = msg?.data || msg
      onTick?.(payload)
    } catch (_) {
      /* ignore malformed */
    }
  })
  es.onerror = () => {
    onStatus?.('error')
  }

  return () => {
    es.close()
  }
}
