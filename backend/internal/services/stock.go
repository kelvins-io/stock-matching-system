package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kelvins-io/stock-matching-system/internal/cache"
	"github.com/kelvins-io/stock-matching-system/internal/models"
	"gorm.io/gorm"
)

type StockService struct {
	db    *gorm.DB
	cache *cache.RedisCache
}

func NewStockService(db *gorm.DB, c *cache.RedisCache) *StockService {
	return &StockService{db: db, cache: c}
}

type IssueStockReq struct {
	Code          string  `json:"code" binding:"required"`
	Name          string  `json:"name" binding:"required"`
	TotalShares   int64   `json:"total_shares" binding:"required,gt=0"`
	IssuePrice    float64 `json:"issue_price" binding:"required,gt=0"`
	PriceLimitPct float64 `json:"price_limit_pct"`
}

type UpdateStockReq struct {
	Name          string              `json:"name"`
	PriceLimitPct *float64            `json:"price_limit_pct"`
	Status        models.StockStatus  `json:"status"`
}

func (s *StockService) Issue(req IssueStockReq) (*models.Stock, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	limit := req.PriceLimitPct
	if limit <= 0 {
		limit = 10
	}
	stock := models.Stock{
		Code:          code,
		Name:          strings.TrimSpace(req.Name),
		TotalShares:   req.TotalShares,
		PriceLimitPct: limit,
		IssuePrice:    req.IssuePrice,
		PrevClose:     req.IssuePrice,
		CurrentPrice:  req.IssuePrice,
		HighPrice:     req.IssuePrice,
		LowPrice:      req.IssuePrice,
		Status:        models.StockActive,
	}
	if err := s.db.Create(&stock).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return nil, errors.New("股票代码已存在")
		}
		return nil, err
	}
	return &stock, nil
}

func (s *StockService) Update(id uuid.UUID, req UpdateStockReq) (*models.Stock, error) {
	var stock models.Stock
	if err := s.db.First(&stock, "id = ?", id).Error; err != nil {
		return nil, errors.New("股票不存在")
	}
	if req.Name != "" {
		stock.Name = req.Name
	}
	if req.PriceLimitPct != nil {
		if *req.PriceLimitPct <= 0 || *req.PriceLimitPct > 100 {
			return nil, errors.New("涨跌幅限制需在 0~100 之间")
		}
		stock.PriceLimitPct = *req.PriceLimitPct
	}
	if req.Status == models.StockActive || req.Status == models.StockSuspended {
		stock.Status = req.Status
	}
	if err := s.db.Save(&stock).Error; err != nil {
		return nil, err
	}
	_ = s.cache.Delete(context.Background(), cache.StockQuoteKey(stock.Code))
	return &stock, nil
}

func (s *StockService) List() ([]models.Stock, error) {
	var stocks []models.Stock
	err := s.db.Order("code asc").Find(&stocks).Error
	return stocks, err
}

func (s *StockService) GetByID(id uuid.UUID) (*models.Stock, error) {
	var stock models.Stock
	if err := s.db.First(&stock, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &stock, nil
}

func (s *StockService) GetByCode(code string) (*models.Stock, error) {
	var stock models.Stock
	if err := s.db.First(&stock, "code = ?", strings.ToUpper(code)).Error; err != nil {
		return nil, err
	}
	return &stock, nil
}

type QuoteView struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	CurrentPrice  float64   `json:"current_price"`
	PrevClose     float64   `json:"prev_close"`
	HighPrice     float64   `json:"high_price"`
	LowPrice      float64   `json:"low_price"`
	Volume        int64     `json:"volume"`
	Amount        float64   `json:"amount"`
	ChangePct     float64   `json:"change_pct"`
	LimitUp       float64   `json:"limit_up"`
	LimitDown     float64   `json:"limit_down"`
	PriceLimitPct float64   `json:"price_limit_pct"`
	Status        string    `json:"status"`
}

func (s *StockService) Quotes() ([]QuoteView, error) {
	stocks, err := s.List()
	if err != nil {
		return nil, err
	}
	views := make([]QuoteView, 0, len(stocks))
	for _, st := range stocks {
		views = append(views, QuoteView{
			ID:            st.ID,
			Code:          st.Code,
			Name:          st.Name,
			CurrentPrice:  st.CurrentPrice,
			PrevClose:     st.PrevClose,
			HighPrice:     st.HighPrice,
			LowPrice:      st.LowPrice,
			Volume:        st.Volume,
			Amount:        st.Amount,
			ChangePct:     st.ChangePct(),
			LimitUp:       st.LimitUp(),
			LimitDown:     st.LimitDown(),
			PriceLimitPct: st.PriceLimitPct,
			Status:        string(st.Status),
		})
	}
	return views, nil
}

type DailySummary struct {
	Date         string      `json:"date"`
	TotalVolume  int64       `json:"total_volume"`
	TotalAmount  float64     `json:"total_amount"`
	TradeCount   int64       `json:"trade_count"`
	StockCount   int64       `json:"stock_count"`
	UpCount      int64       `json:"up_count"`
	DownCount    int64       `json:"down_count"`
	FlatCount    int64       `json:"flat_count"`
	HourlyVolume []HourPoint `json:"hourly_volume"`
	TopGainers   []QuoteView `json:"top_gainers"`
	TopLosers    []QuoteView `json:"top_losers"`
	TopVolume    []QuoteView `json:"top_volume"`
}

type HourPoint struct {
	Hour   string  `json:"hour"`
	Volume int64   `json:"volume"`
	Amount float64 `json:"amount"`
}

func (s *StockService) DailySummary() (*DailySummary, error) {
	ctx := context.Background()
	today := time.Now().Format("2006-01-02")
	var cached DailySummary
	if err := s.cache.GetJSON(ctx, cache.DailySummaryKey(today), &cached); err == nil {
		return &cached, nil
	}

	quotes, err := s.Quotes()
	if err != nil {
		return nil, err
	}

	summary := &DailySummary{
		Date:       today,
		StockCount: int64(len(quotes)),
	}
	for _, q := range quotes {
		summary.TotalVolume += q.Volume
		summary.TotalAmount += q.Amount
		switch {
		case q.ChangePct > 0.0001:
			summary.UpCount++
		case q.ChangePct < -0.0001:
			summary.DownCount++
		default:
			summary.FlatCount++
		}
	}

	start := time.Now().Truncate(24 * time.Hour)
	var tradeCount int64
	s.db.Model(&models.Trade{}).Where("created_at >= ?", start).Count(&tradeCount)
	summary.TradeCount = tradeCount

	type row struct {
		Hour   string
		Volume int64
		Amount float64
	}
	var rows []row
	s.db.Raw(`
		SELECT to_char(created_at, 'HH24:00') AS hour,
		       COALESCE(SUM(quantity),0) AS volume,
		       COALESCE(SUM(amount),0) AS amount
		FROM trades
		WHERE created_at >= ?
		GROUP BY 1
		ORDER BY 1
	`, start).Scan(&rows)
	summary.HourlyVolume = make([]HourPoint, 0, len(rows))
	for _, r := range rows {
		summary.HourlyVolume = append(summary.HourlyVolume, HourPoint{
			Hour: r.Hour, Volume: r.Volume, Amount: r.Amount,
		})
	}

	sorted := append([]QuoteView{}, quotes...)
	summary.TopGainers = topN(sorted, 5, func(a, b QuoteView) bool { return a.ChangePct > b.ChangePct })
	summary.TopLosers = topN(sorted, 5, func(a, b QuoteView) bool { return a.ChangePct < b.ChangePct })
	summary.TopVolume = topN(sorted, 5, func(a, b QuoteView) bool { return a.Volume > b.Volume })

	_ = s.cache.SetJSON(ctx, cache.DailySummaryKey(today), summary, 3*time.Second)
	return summary, nil
}

func topN(items []QuoteView, n int, less func(a, b QuoteView) bool) []QuoteView {
	cp := append([]QuoteView{}, items...)
	for i := 0; i < len(cp); i++ {
		for j := i + 1; j < len(cp); j++ {
			if less(cp[j], cp[i]) {
				cp[i], cp[j] = cp[j], cp[i]
			}
		}
	}
	if len(cp) > n {
		cp = cp[:n]
	}
	return cp
}
