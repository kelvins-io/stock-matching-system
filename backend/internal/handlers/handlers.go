package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kelvins-io/stock-matching-system/internal/middleware"
	"github.com/kelvins-io/stock-matching-system/internal/models"
	"github.com/kelvins-io/stock-matching-system/internal/services"
)

type Handler struct {
	auth    *services.AuthService
	stock   *services.StockService
	market  *services.MarketService
	order   *services.OrderService
	matcher *services.MatchingEngine
	kline   *services.KlineService
	hub     *services.MarketHub
	jwt     *middleware.JWTAuth
}

func New(
	auth *services.AuthService,
	stock *services.StockService,
	market *services.MarketService,
	order *services.OrderService,
	matcher *services.MatchingEngine,
	jwt *middleware.JWTAuth,
	kline *services.KlineService,
	hub *services.MarketHub,
) *Handler {
	return &Handler{auth: auth, stock: stock, market: market, order: order, matcher: matcher, jwt: jwt, kline: kline, hub: hub}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/auth/register", h.Register)
		api.POST("/auth/login", h.Login)

		api.GET("/market/status", h.MarketStatus)
		api.GET("/stocks", h.ListStocks)
		api.GET("/stocks/quotes", h.Quotes)
		api.GET("/market/summary", h.DailySummary)
		api.GET("/trades/recent", h.RecentTrades)
		api.GET("/stocks/:id/orderbook", h.OrderBook)
		api.GET("/stocks/:id/kline", h.Klines)
		api.GET("/market/stream", h.MarketStream)

		auth := api.Group("")
		auth.Use(h.jwt.Required())
		{
			auth.GET("/auth/me", h.Me)
			auth.POST("/orders", h.PlaceOrder)
			auth.DELETE("/orders/:id", h.CancelOrder)
			auth.GET("/orders", h.MyOrders)
			auth.GET("/trades", h.MyTrades)
			auth.GET("/holdings", h.MyHoldings)

			admin := auth.Group("/admin")
			admin.Use(middleware.AdminRequired())
			{
				admin.POST("/stocks", h.IssueStock)
				admin.PUT("/stocks/:id", h.UpdateStock)
				admin.GET("/market/rules", h.GetMarketRule)
				admin.PUT("/market/rules", h.UpdateMarketRule)
			}
		}
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req services.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：" + err.Error()})
		return
	}
	resp, err := h.auth.Register(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Login(c *gin.Context) {
	var req services.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	resp, err := h.auth.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Me(c *gin.Context) {
	uid, _ := c.Get("user_id")
	user, err := h.auth.GetUser(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) IssueStock(c *gin.Context) {
	var req services.IssueStockReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：" + err.Error()})
		return
	}
	stock, err := h.stock.Issue(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *Handler) UpdateStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}
	var req services.UpdateStockReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	stock, err := h.stock.Update(id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *Handler) ListStocks(c *gin.Context) {
	stocks, err := h.stock.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stocks)
}

func (h *Handler) Quotes(c *gin.Context) {
	quotes, err := h.stock.Quotes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, quotes)
}

func (h *Handler) DailySummary(c *gin.Context) {
	summary, err := h.stock.DailySummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *Handler) MarketStatus(c *gin.Context) {
	st, err := h.market.Status()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, st)
}

func (h *Handler) GetMarketRule(c *gin.Context) {
	rule, err := h.market.GetRule()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func (h *Handler) UpdateMarketRule(c *gin.Context) {
	var req services.UpdateMarketRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	rule, err := h.market.UpdateRule(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func (h *Handler) PlaceOrder(c *gin.Context) {
	uid := c.MustGet("user_id").(uuid.UUID)
	var req services.PlaceOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：" + err.Error()})
		return
	}
	order, err := h.matcher.PlaceOrder(uid, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *Handler) CancelOrder(c *gin.Context) {
	uid := c.MustGet("user_id").(uuid.UUID)
	oid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效订单 ID"})
		return
	}
	order, err := h.matcher.CancelOrder(uid, oid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *Handler) MyOrders(c *gin.Context) {
	uid := c.MustGet("user_id").(uuid.UUID)
	orders, err := h.order.ListByUser(uid, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *Handler) MyTrades(c *gin.Context) {
	uid := c.MustGet("user_id").(uuid.UUID)
	trades, err := h.order.ListTradesByUser(uid, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, trades)
}

func (h *Handler) MyHoldings(c *gin.Context) {
	uid := c.MustGet("user_id").(uuid.UUID)
	holdings, err := h.order.ListHoldings(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, holdings)
}

func (h *Handler) RecentTrades(c *gin.Context) {
	var stockID *uuid.UUID
	if sid := c.Query("stock_id"); sid != "" {
		id, err := uuid.Parse(sid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效股票 ID"})
			return
		}
		stockID = &id
	}
	trades, err := h.order.RecentTrades(stockID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, trades)
}

func (h *Handler) OrderBook(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}
	book, err := h.order.OrderBook(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, book)
}

func (h *Handler) Klines(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}
	interval := models.KlineInterval(c.DefaultQuery("interval", "1d"))
	limit := 120
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	bars, err := h.kline.List(id, interval, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bars)
}
