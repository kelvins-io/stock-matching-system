package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// MarketStream SSE 实时行情：tick(报价/成交/K线) + heartbeat
func (h *Handler) MarketStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	stockFilter := c.Query("stock_id")
	ch, unsub := h.hub.Subscribe()
	defer unsub()

	// 连接建立提示
	writeSSE(c.Writer, "connected", map[string]any{
		"clients": h.hub.ClientCount(),
		"filter":  stockFilter,
	})

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if err := writeSSE(c.Writer, "heartbeat", map[string]any{"ts": time.Now().UnixMilli()}); err != nil {
				return
			}
		case raw, ok := <-ch:
			if !ok {
				return
			}
			if stockFilter != "" {
				var env struct {
					Type string `json:"type"`
					Data struct {
						StockID string `json:"stock_id"`
					} `json:"data"`
				}
				if err := json.Unmarshal(raw, &env); err == nil && env.Type == "tick" {
					if env.Data.StockID != "" && env.Data.StockID != stockFilter {
						continue
					}
				}
			}
			if _, err := fmt.Fprintf(c.Writer, "event: tick\ndata: %s\n\n", raw); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

func writeSSE(w io.Writer, event string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
