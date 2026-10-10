package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kelvins-io/stock-matching-system/internal/cache"
	"github.com/kelvins-io/stock-matching-system/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MatchingEngine A股风格价格优先、时间优先限价撮合
type MatchingEngine struct {
	db     *gorm.DB
	cache  *cache.RedisCache
	market *MarketService
	mu     sync.Mutex
}

func NewMatchingEngine(db *gorm.DB, c *cache.RedisCache, market *MarketService) *MatchingEngine {
	return &MatchingEngine{db: db, cache: c, market: market}
}

type PlaceOrderReq struct {
	StockID  uuid.UUID        `json:"stock_id" binding:"required"`
	Side     models.OrderSide `json:"side" binding:"required"`
	Price    float64          `json:"price" binding:"required,gt=0"`
	Quantity int64            `json:"quantity" binding:"required,gt=0"`
}

func (e *MatchingEngine) PlaceOrder(userID uuid.UUID, req PlaceOrderReq) (*models.Order, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	open, reason, err := e.market.IsOpen()
	if err != nil {
		return nil, err
	}
	if !open {
		return nil, fmt.Errorf("当前休市：%s", reason)
	}

	if req.Side != models.SideBuy && req.Side != models.SideSell {
		return nil, errors.New("无效的买卖方向")
	}
	if req.Quantity%100 != 0 {
		return nil, errors.New("委托数量须为 100 股的整数倍")
	}

	var order models.Order
	err = e.db.Transaction(func(tx *gorm.DB) error {
		var stock models.Stock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&stock, "id = ?", req.StockID).Error; err != nil {
			return errors.New("股票不存在")
		}
		if stock.Status != models.StockActive {
			return errors.New("股票已停牌，无法交易")
		}

		limitUp := stock.LimitUp()
		limitDown := stock.LimitDown()
		if req.Price > limitUp+1e-9 || req.Price < limitDown-1e-9 {
			return fmt.Errorf("委托价格超出涨跌停范围 [%.2f, %.2f]", limitDown, limitUp)
		}

		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&user, "id = ?", userID).Error; err != nil {
			return errors.New("用户不存在")
		}

		order = models.Order{
			UserID:    userID,
			StockID:   stock.ID,
			StockCode: stock.Code,
			StockName: stock.Name,
			Side:      req.Side,
			Price:     req.Price,
			Quantity:  req.Quantity,
			FilledQty: 0,
			Status:    models.OrderPending,
		}

		if req.Side == models.SideBuy {
			need := req.Price * float64(req.Quantity)
			if user.Balance < need {
				return errors.New("可用资金不足")
			}
			user.Balance -= need
			if err := tx.Save(&user).Error; err != nil {
				return err
			}
		} else {
			var holding models.Holding
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ? AND stock_id = ?", userID, stock.ID).
				First(&holding).Error
			if err != nil {
				return errors.New("持仓不足")
			}
			if holding.Available() < req.Quantity {
				return errors.New("可用持仓不足")
			}
			holding.FrozenQty += req.Quantity
			if err := tx.Save(&holding).Error; err != nil {
				return err
			}
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		return e.match(tx, &order, &stock)
	})
	if err != nil {
		return nil, err
	}

	e.invalidateCache(order.StockCode)
	return &order, nil
}

func (e *MatchingEngine) match(tx *gorm.DB, incoming *models.Order, stock *models.Stock) error {
	for incoming.Remaining() > 0 {
		var opposite models.Order
		q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("stock_id = ? AND status IN ? AND side = ? AND id <> ?",
				stock.ID,
				[]models.OrderStatus{models.OrderPending, models.OrderPartial},
				oppositeSide(incoming.Side),
				incoming.ID)

		if incoming.Side == models.SideBuy {
			// 买盘吃卖盘：卖价 <= 买价，卖价从低到高，时间从早到晚
			q = q.Where("price <= ?", incoming.Price).
				Order("price ASC, created_at ASC")
		} else {
			// 卖盘吃买盘：买价 >= 卖价，买价从高到低，时间从早到晚
			q = q.Where("price >= ?", incoming.Price).
				Order("price DESC, created_at ASC")
		}

		if err := q.First(&opposite).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return err
		}

		tradeQty := min64(incoming.Remaining(), opposite.Remaining())
		// 成交价取先挂单价格（对手方价格）
		tradePrice := opposite.Price
		if err := e.executeTrade(tx, stock, incoming, &opposite, tradePrice, tradeQty); err != nil {
			return err
		}
	}

	if err := tx.Save(incoming).Error; err != nil {
		return err
	}
	return tx.Save(stock).Error
}

func (e *MatchingEngine) executeTrade(
	tx *gorm.DB,
	stock *models.Stock,
	a, b *models.Order,
	price float64,
	qty int64,
) error {
	buy, sell := a, b
	if a.Side == models.SideSell {
		buy, sell = b, a
	}

	amount := price * float64(qty)
	trade := models.Trade{
		StockID:     stock.ID,
		StockCode:   stock.Code,
		BuyOrderID:  buy.ID,
		SellOrderID: sell.ID,
		BuyerID:     buy.UserID,
		SellerID:    sell.UserID,
		Price:       price,
		Quantity:    qty,
		Amount:      amount,
	}
	if err := tx.Create(&trade).Error; err != nil {
		return err
	}

	buy.FilledQty += qty
	sell.FilledQty += qty
	buy.Status = orderStatus(buy)
	sell.Status = orderStatus(sell)
	if err := tx.Save(buy).Error; err != nil {
		return err
	}
	if err := tx.Save(sell).Error; err != nil {
		return err
	}

	// 买方：解冻未用差价退回，增加持仓
	if err := e.settleBuyer(tx, buy, stock, price, qty); err != nil {
		return err
	}
	// 卖方：解冻持仓，增加资金
	if err := e.settleSeller(tx, sell, stock, price, qty); err != nil {
		return err
	}

	stock.CurrentPrice = price
	if stock.HighPrice == 0 || price > stock.HighPrice {
		stock.HighPrice = price
	}
	if stock.LowPrice == 0 || price < stock.LowPrice {
		stock.LowPrice = price
	}
	stock.Volume += qty
	stock.Amount += amount
	return nil
}

func (e *MatchingEngine) settleBuyer(tx *gorm.DB, buy *models.Order, stock *models.Stock, price float64, qty int64) error {
	var buyer models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&buyer, "id = ?", buy.UserID).Error; err != nil {
		return err
	}
	// 下单时按委托价冻结，成交价可能更低，退回差价
	refund := (buy.Price - price) * float64(qty)
	if refund > 0 {
		buyer.Balance += refund
		if err := tx.Save(&buyer).Error; err != nil {
			return err
		}
	}

	var holding models.Holding
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND stock_id = ?", buy.UserID, stock.ID).
		First(&holding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		holding = models.Holding{
			UserID:    buy.UserID,
			StockID:   stock.ID,
			StockCode: stock.Code,
			StockName: stock.Name,
			Quantity:  qty,
			AvgCost:   price,
		}
		return tx.Create(&holding).Error
	}
	if err != nil {
		return err
	}
	totalCost := holding.AvgCost*float64(holding.Quantity) + price*float64(qty)
	holding.Quantity += qty
	holding.AvgCost = totalCost / float64(holding.Quantity)
	return tx.Save(&holding).Error
}

func (e *MatchingEngine) settleSeller(tx *gorm.DB, sell *models.Order, stock *models.Stock, price float64, qty int64) error {
	var seller models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&seller, "id = ?", sell.UserID).Error; err != nil {
		return err
	}
	seller.Balance += price * float64(qty)
	if err := tx.Save(&seller).Error; err != nil {
		return err
	}

	var holding models.Holding
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND stock_id = ?", sell.UserID, stock.ID).
		First(&holding).Error; err != nil {
		return err
	}
	holding.FrozenQty -= qty
	holding.Quantity -= qty
	if holding.FrozenQty < 0 {
		holding.FrozenQty = 0
	}
	if holding.Quantity < 0 {
		return errors.New("持仓结算异常")
	}
	return tx.Save(&holding).Error
}

func (e *MatchingEngine) CancelOrder(userID uuid.UUID, orderID uuid.UUID) (*models.Order, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var order models.Order
	err := e.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&order, "id = ?", orderID).Error; err != nil {
			return errors.New("订单不存在")
		}
		if order.UserID != userID {
			return errors.New("无权撤销该订单")
		}
		if order.Status != models.OrderPending && order.Status != models.OrderPartial {
			return errors.New("订单当前状态不可撤销")
		}

		remain := order.Remaining()
		if order.Side == models.SideBuy {
			var user models.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&user, "id = ?", userID).Error; err != nil {
				return err
			}
			user.Balance += order.Price * float64(remain)
			if err := tx.Save(&user).Error; err != nil {
				return err
			}
		} else {
			var holding models.Holding
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ? AND stock_id = ?", userID, order.StockID).
				First(&holding).Error; err != nil {
				return err
			}
			holding.FrozenQty -= remain
			if holding.FrozenQty < 0 {
				holding.FrozenQty = 0
			}
			if err := tx.Save(&holding).Error; err != nil {
				return err
			}
		}

		order.Status = models.OrderCancelled
		return tx.Save(&order).Error
	})
	if err != nil {
		return nil, err
	}
	e.invalidateCache(order.StockCode)
	return &order, nil
}

func (e *MatchingEngine) invalidateCache(code string) {
	ctx := context.Background()
	today := time.Now().Format("2006-01-02")
	_ = e.cache.Delete(ctx, cache.StockQuoteKey(code), cache.DailySummaryKey(today))
}

func oppositeSide(side models.OrderSide) models.OrderSide {
	if side == models.SideBuy {
		return models.SideSell
	}
	return models.SideBuy
}

func orderStatus(o *models.Order) models.OrderStatus {
	if o.FilledQty <= 0 {
		return models.OrderPending
	}
	if o.FilledQty >= o.Quantity {
		return models.OrderFilled
	}
	return models.OrderPartial
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
