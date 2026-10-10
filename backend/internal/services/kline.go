package services

import (
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/kelvins-io/stock-matching-system/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type KlineService struct {
	db *gorm.DB
}

func NewKlineService(db *gorm.DB) *KlineService {
	return &KlineService{db: db}
}

var supportedIntervals = map[models.KlineInterval]bool{
	models.Kline1m:  true,
	models.Kline5m:  true,
	models.Kline15m: true,
	models.Kline1d:  true,
}

// LatestBars 返回各周期最新一根 K 线
func (s *KlineService) LatestBars(stockID uuid.UUID) map[string]*models.Kline {
	out := make(map[string]*models.Kline)
	for _, iv := range []models.KlineInterval{models.Kline1m, models.Kline5m, models.Kline15m, models.Kline1d} {
		var bar models.Kline
		err := s.db.Where("stock_id = ? AND interval = ?", stockID, iv).
			Order("open_time desc").
			First(&bar).Error
		if err == nil {
			cp := bar
			out[string(iv)] = &cp
		}
	}
	return out
}

func (s *KlineService) List(stockID uuid.UUID, interval models.KlineInterval, limit int) ([]models.Kline, error) {
	if !supportedIntervals[interval] {
		return nil, errors.New("不支持的 K 线周期，可选：1m/5m/15m/1d")
	}
	if limit <= 0 || limit > 500 {
		limit = 120
	}

	if interval == models.Kline1d {
		var cnt int64
		s.db.Model(&models.Kline{}).Where("stock_id = ? AND interval = ?", stockID, interval).Count(&cnt)
		if cnt == 0 {
			var stock models.Stock
			if err := s.db.First(&stock, "id = ?", stockID).Error; err == nil {
				_ = s.SeedDailyHistory(&stock, 60)
			}
		}
	}

	var bars []models.Kline
	err := s.db.Where("stock_id = ? AND interval = ?", stockID, interval).
		Order("open_time desc").
		Limit(limit).
		Find(&bars).Error
	if err != nil {
		return nil, err
	}
	// 按时间升序返回，便于前端绘图
	for i, j := 0, len(bars)-1; i < j; i, j = i+1, j-1 {
		bars[i], bars[j] = bars[j], bars[i]
	}
	return bars, nil
}

// ApplyTrade 成交后更新各周期当前蜡烛（同事务）
func (s *KlineService) ApplyTrade(tx *gorm.DB, stockID uuid.UUID, price float64, qty int64, amount float64, at time.Time) error {
	intervals := []models.KlineInterval{models.Kline1m, models.Kline5m, models.Kline15m, models.Kline1d}
	for _, iv := range intervals {
		if err := s.upsertBar(tx, stockID, iv, price, qty, amount, at); err != nil {
			return err
		}
	}
	return nil
}

func (s *KlineService) upsertBar(tx *gorm.DB, stockID uuid.UUID, interval models.KlineInterval, price float64, qty int64, amount float64, at time.Time) error {
	openTime := truncateInterval(at, interval)
	var bar models.Kline
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("stock_id = ? AND interval = ? AND open_time = ?", stockID, interval, openTime).
		First(&bar).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		bar = models.Kline{
			StockID:  stockID,
			Interval: interval,
			OpenTime: openTime,
			Open:     price,
			High:     price,
			Low:      price,
			Close:    price,
			Volume:   qty,
			Amount:   amount,
		}
		return tx.Create(&bar).Error
	}
	if err != nil {
		return err
	}
	if price > bar.High {
		bar.High = price
	}
	if price < bar.Low {
		bar.Low = price
	}
	bar.Close = price
	bar.Volume += qty
	bar.Amount += amount
	return tx.Save(&bar).Error
}

func truncateInterval(t time.Time, interval models.KlineInterval) time.Time {
	loc := t.Location()
	switch interval {
	case models.Kline1d:
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc)
	case models.Kline1m:
		return t.Truncate(time.Minute)
	case models.Kline5m:
		return t.Truncate(5 * time.Minute)
	case models.Kline15m:
		return t.Truncate(15 * time.Minute)
	default:
		return t.Truncate(time.Minute)
	}
}

// SeedMissingDaily 为缺少日 K 的股票补全演示数据
func (s *KlineService) SeedMissingDaily(days int) error {
	var stocks []models.Stock
	if err := s.db.Find(&stocks).Error; err != nil {
		return err
	}
	for i := range stocks {
		if err := s.SeedDailyHistory(&stocks[i], days); err != nil {
			return err
		}
	}
	return nil
}

// SeedDailyHistory 生成演示用日 K（随机游走，终点贴近现价）
func (s *KlineService) SeedDailyHistory(stock *models.Stock, days int) error {
	if stock == nil || days <= 0 {
		return nil
	}
	var exist int64
	s.db.Model(&models.Kline{}).
		Where("stock_id = ? AND interval = ?", stock.ID, models.Kline1d).
		Count(&exist)
	if exist > 0 {
		return nil
	}

	seed := int64(0)
	for _, c := range stock.Code {
		seed = seed*31 + int64(c)
	}
	rng := rand.New(rand.NewSource(seed + 20261010))

	target := stock.CurrentPrice
	if target <= 0 {
		target = stock.IssuePrice
	}

	// 收集交易日
	dates := make([]time.Time, 0, days)
	for d := 1; len(dates) < days; d++ {
		t := time.Now().AddDate(0, 0, -d)
		if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
			continue
		}
		y, m, day := t.Date()
		dates = append(dates, time.Date(y, m, day, 0, 0, 0, 0, t.Location()))
	}
	// dates[0] 最近，反转为旧->新
	for i, j := 0, len(dates)-1; i < j; i, j = i+1, j-1 {
		dates[i], dates[j] = dates[j], dates[i]
	}

	closes := make([]float64, days)
	closes[days-1] = target
	for i := days - 2; i >= 0; i-- {
		chg := (rng.Float64()*2 - 1) * 0.025
		prev := closes[i+1] / (1 + chg)
		if prev < 0.01 {
			prev = 0.01
		}
		closes[i] = round4(prev)
	}

	bars := make([]models.Kline, 0, days)
	for i, openTime := range dates {
		open := closes[i]
		if i > 0 {
			open = closes[i-1]
		}
		closePx := closes[i]
		high := math.Max(open, closePx) * (1 + rng.Float64()*0.012)
		low := math.Min(open, closePx) * (1 - rng.Float64()*0.012)
		if low < 0.01 {
			low = 0.01
		}
		vol := int64(100000 + rng.Intn(900000))
		amt := (open + closePx) / 2 * float64(vol)
		bars = append(bars, models.Kline{
			StockID:  stock.ID,
			Interval: models.Kline1d,
			OpenTime: openTime,
			Open:     round4(open),
			High:     round4(high),
			Low:      round4(low),
			Close:    round4(closePx),
			Volume:   vol,
			Amount:   round2(amt),
		})
	}
	return s.db.CreateInBatches(bars, 50).Error
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
