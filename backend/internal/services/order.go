package services

import (
	"github.com/google/uuid"
	"github.com/kelvins-io/stock-matching-system/internal/models"
	"gorm.io/gorm"
)

type OrderService struct {
	db *gorm.DB
}

func NewOrderService(db *gorm.DB) *OrderService {
	return &OrderService{db: db}
}

func (s *OrderService) ListByUser(userID uuid.UUID, limit int) ([]models.Order, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var orders []models.Order
	err := s.db.Where("user_id = ?", userID).
		Order("created_at desc").
		Limit(limit).
		Find(&orders).Error
	return orders, err
}

func (s *OrderService) ListTradesByUser(userID uuid.UUID, limit int) ([]models.Trade, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var trades []models.Trade
	err := s.db.Where("buyer_id = ? OR seller_id = ?", userID, userID).
		Order("created_at desc").
		Limit(limit).
		Find(&trades).Error
	return trades, err
}

func (s *OrderService) RecentTrades(stockID *uuid.UUID, limit int) ([]models.Trade, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.db.Model(&models.Trade{}).Order("created_at desc").Limit(limit)
	if stockID != nil {
		q = q.Where("stock_id = ?", *stockID)
	}
	var trades []models.Trade
	err := q.Find(&trades).Error
	return trades, err
}

func (s *OrderService) ListHoldings(userID uuid.UUID) ([]models.Holding, error) {
	var holdings []models.Holding
	err := s.db.Where("user_id = ? AND quantity > 0", userID).
		Order("stock_code asc").
		Find(&holdings).Error
	return holdings, err
}

// DepthLevel 五档盘口
type DepthLevel struct {
	Price    float64 `json:"price"`
	Quantity int64   `json:"quantity"`
}

type OrderBook struct {
	StockCode string       `json:"stock_code"`
	Bids      []DepthLevel `json:"bids"`
	Asks      []DepthLevel `json:"asks"`
}

func (s *OrderService) OrderBook(stockID uuid.UUID) (*OrderBook, error) {
	var stock models.Stock
	if err := s.db.First(&stock, "id = ?", stockID).Error; err != nil {
		return nil, err
	}

	type agg struct {
		Price    float64
		Quantity int64
	}

	var bids, asks []agg
	s.db.Raw(`
		SELECT price, SUM(quantity - filled_qty) AS quantity
		FROM orders
		WHERE stock_id = ? AND side = 'buy' AND status IN ('pending','partial')
		GROUP BY price ORDER BY price DESC LIMIT 5
	`, stockID).Scan(&bids)

	s.db.Raw(`
		SELECT price, SUM(quantity - filled_qty) AS quantity
		FROM orders
		WHERE stock_id = ? AND side = 'sell' AND status IN ('pending','partial')
		GROUP BY price ORDER BY price ASC LIMIT 5
	`, stockID).Scan(&asks)

	book := &OrderBook{StockCode: stock.Code, Bids: []DepthLevel{}, Asks: []DepthLevel{}}
	for _, b := range bids {
		book.Bids = append(book.Bids, DepthLevel{Price: b.Price, Quantity: b.Quantity})
	}
	for _, a := range asks {
		book.Asks = append(book.Asks, DepthLevel{Price: a.Price, Quantity: a.Quantity})
	}
	return book, nil
}
