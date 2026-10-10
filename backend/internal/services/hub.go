package services

import (
	"encoding/json"
	"sync"
	"time"
)

// MarketHub 内存广播，供 SSE 推行情/成交/K 线
type MarketHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func NewMarketHub() *MarketHub {
	return &MarketHub{clients: make(map[chan []byte]struct{})}
}

func (h *MarketHub) Subscribe() (ch chan []byte, unsubscribe func()) {
	ch = make(chan []byte, 32)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	unsubscribe = func() {
		h.mu.Lock()
		if _, ok := h.clients[ch]; ok {
			delete(h.clients, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
	return ch, unsubscribe
}

func (h *MarketHub) Publish(eventType string, payload any) {
	envelope := map[string]any{
		"type": eventType,
		"ts":   time.Now().UnixMilli(),
		"data": payload,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- raw:
		default:
			// 慢客户端丢弃，避免阻塞撮合
		}
	}
}

func (h *MarketHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
