package services

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/kelvins-io/stock-matching-system/internal/cache"
	"github.com/kelvins-io/stock-matching-system/internal/models"
	"gorm.io/gorm"
)

type MarketService struct {
	db    *gorm.DB
	cache *cache.RedisCache
}

func NewMarketService(db *gorm.DB, c *cache.RedisCache) *MarketService {
	return &MarketService{db: db, cache: c}
}

func (s *MarketService) GetRule() (*models.MarketRule, error) {
	var rule models.MarketRule
	if err := s.db.First(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

type UpdateMarketRuleReq struct {
	OpenAM      string `json:"open_am"`
	CloseAM     string `json:"close_am"`
	OpenPM      string `json:"open_pm"`
	ClosePM     string `json:"close_pm"`
	TradingDays string `json:"trading_days"`
	ForceClosed bool   `json:"force_closed"`
	ForceOpen   bool   `json:"force_open"`
}

func (s *MarketService) UpdateRule(req UpdateMarketRuleReq) (*models.MarketRule, error) {
	rule, err := s.GetRule()
	if err != nil {
		return nil, err
	}
	if req.OpenAM != "" {
		rule.OpenAM = req.OpenAM
	}
	if req.CloseAM != "" {
		rule.CloseAM = req.CloseAM
	}
	if req.OpenPM != "" {
		rule.OpenPM = req.OpenPM
	}
	if req.ClosePM != "" {
		rule.ClosePM = req.ClosePM
	}
	if req.TradingDays != "" {
		rule.TradingDays = req.TradingDays
	}
	rule.ForceClosed = req.ForceClosed
	rule.ForceOpen = req.ForceOpen
	if err := s.db.Save(rule).Error; err != nil {
		return nil, err
	}
	_ = s.cache.Delete(context.Background(), cache.MarketStatusKey())
	return rule, nil
}

type MarketStatus struct {
	IsOpen      bool   `json:"is_open"`
	Reason      string `json:"reason"`
	ServerTime  string `json:"server_time"`
	OpenAM      string `json:"open_am"`
	CloseAM     string `json:"close_am"`
	OpenPM      string `json:"open_pm"`
	ClosePM     string `json:"close_pm"`
	TradingDays string `json:"trading_days"`
	ForceClosed bool   `json:"force_closed"`
	ForceOpen   bool   `json:"force_open"`
}

func (s *MarketService) Status() (*MarketStatus, error) {
	ctx := context.Background()
	var cached MarketStatus
	if err := s.cache.GetJSON(ctx, cache.MarketStatusKey(), &cached); err == nil {
		cached.ServerTime = time.Now().Format("2006-01-02 15:04:05")
		return &cached, nil
	}

	rule, err := s.GetRule()
	if err != nil {
		return nil, err
	}
	status := s.computeStatus(rule, time.Now())
	_ = s.cache.SetJSON(ctx, cache.MarketStatusKey(), status, 5*time.Second)
	return status, nil
}

func (s *MarketService) IsOpen() (bool, string, error) {
	st, err := s.Status()
	if err != nil {
		return false, "", err
	}
	return st.IsOpen, st.Reason, nil
}

func (s *MarketService) computeStatus(rule *models.MarketRule, now time.Time) *MarketStatus {
	st := &MarketStatus{
		ServerTime:  now.Format("2006-01-02 15:04:05"),
		OpenAM:      rule.OpenAM,
		CloseAM:     rule.CloseAM,
		OpenPM:      rule.OpenPM,
		ClosePM:     rule.ClosePM,
		TradingDays: rule.TradingDays,
		ForceClosed: rule.ForceClosed,
		ForceOpen:   rule.ForceOpen,
	}

	if rule.ForceClosed {
		st.IsOpen = false
		st.Reason = "管理员强制休市"
		return st
	}
	if rule.ForceOpen {
		st.IsOpen = true
		st.Reason = "管理员强制开市"
		return st
	}

	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	allowed := false
	for _, d := range strings.Split(rule.TradingDays, ",") {
		if strings.TrimSpace(d) == strconv.Itoa(weekday) {
			allowed = true
			break
		}
	}
	if !allowed {
		st.IsOpen = false
		st.Reason = "非交易日"
		return st
	}

	hm := now.Format("15:04")
	if (hm >= rule.OpenAM && hm < rule.CloseAM) || (hm >= rule.OpenPM && hm < rule.ClosePM) {
		st.IsOpen = true
		st.Reason = "交易时段内"
		return st
	}
	st.IsOpen = false
	st.Reason = "非交易时段"
	return st
}
