package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type OrderSide string

const (
	SideBuy  OrderSide = "buy"
	SideSell OrderSide = "sell"
)

type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderPartial   OrderStatus = "partial"
	OrderFilled    OrderStatus = "filled"
	OrderCancelled OrderStatus = "cancelled"
	OrderRejected  OrderStatus = "rejected"
)

type StockStatus string

const (
	StockActive   StockStatus = "active"
	StockSuspended StockStatus = "suspended"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         Role      `gorm:"size:16;not null;default:user" json:"role"`
	Balance      float64   `gorm:"type:numeric(18,2);not null;default:1000000" json:"balance"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type Stock struct {
	ID            uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	Code          string      `gorm:"uniqueIndex;size:16;not null" json:"code"`
	Name          string      `gorm:"size:64;not null" json:"name"`
	TotalShares   int64       `gorm:"not null" json:"total_shares"`
	PriceLimitPct float64     `gorm:"type:numeric(6,2);not null;default:10" json:"price_limit_pct"`
	IssuePrice    float64     `gorm:"type:numeric(18,4);not null" json:"issue_price"`
	PrevClose     float64     `gorm:"type:numeric(18,4);not null" json:"prev_close"`
	CurrentPrice  float64     `gorm:"type:numeric(18,4);not null" json:"current_price"`
	HighPrice     float64     `gorm:"type:numeric(18,4);not null;default:0" json:"high_price"`
	LowPrice      float64     `gorm:"type:numeric(18,4);not null;default:0" json:"low_price"`
	Volume        int64       `gorm:"not null;default:0" json:"volume"`
	Amount        float64     `gorm:"type:numeric(20,2);not null;default:0" json:"amount"`
	Status        StockStatus `gorm:"size:16;not null;default:active" json:"status"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

func (s *Stock) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (s *Stock) LimitUp() float64 {
	return roundPrice(s.PrevClose * (1 + s.PriceLimitPct/100))
}

func (s *Stock) LimitDown() float64 {
	return roundPrice(s.PrevClose * (1 - s.PriceLimitPct/100))
}

func (s *Stock) ChangePct() float64 {
	if s.PrevClose == 0 {
		return 0
	}
	return (s.CurrentPrice - s.PrevClose) / s.PrevClose * 100
}

func roundPrice(p float64) float64 {
	return float64(int(p*100+0.5)) / 100
}

// MarketRule 开市闭市规则（A股风格：上午/下午两个时段）
type MarketRule struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OpenAM       string    `gorm:"size:8;not null;default:09:30" json:"open_am"`
	CloseAM      string    `gorm:"size:8;not null;default:11:30" json:"close_am"`
	OpenPM       string    `gorm:"size:8;not null;default:13:00" json:"open_pm"`
	ClosePM      string    `gorm:"size:8;not null;default:15:00" json:"close_pm"`
	TradingDays  string    `gorm:"size:32;not null;default:1,2,3,4,5" json:"trading_days"` // 1=周一 ... 7=周日
	ForceClosed  bool      `gorm:"not null;default:false" json:"force_closed"`
	ForceOpen    bool      `gorm:"not null;default:false" json:"force_open"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (m *MarketRule) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type Order struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID   `gorm:"type:uuid;index;not null" json:"user_id"`
	StockID     uuid.UUID   `gorm:"type:uuid;index;not null" json:"stock_id"`
	StockCode   string      `gorm:"size:16;not null" json:"stock_code"`
	StockName   string      `gorm:"size:64;not null" json:"stock_name"`
	Side        OrderSide   `gorm:"size:8;not null" json:"side"`
	Price       float64     `gorm:"type:numeric(18,4);not null" json:"price"`
	Quantity    int64       `gorm:"not null" json:"quantity"`
	FilledQty   int64       `gorm:"not null;default:0" json:"filled_qty"`
	Status      OrderStatus `gorm:"size:16;not null;default:pending;index" json:"status"`
	RejectReason string     `gorm:"size:255" json:"reject_reason,omitempty"`
	CreatedAt   time.Time   `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

func (o *Order) Remaining() int64 {
	return o.Quantity - o.FilledQty
}

type Trade struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	StockID     uuid.UUID `gorm:"type:uuid;index;not null" json:"stock_id"`
	StockCode   string    `gorm:"size:16;not null" json:"stock_code"`
	BuyOrderID  uuid.UUID `gorm:"type:uuid;not null" json:"buy_order_id"`
	SellOrderID uuid.UUID `gorm:"type:uuid;not null" json:"sell_order_id"`
	BuyerID     uuid.UUID `gorm:"type:uuid;index;not null" json:"buyer_id"`
	SellerID    uuid.UUID `gorm:"type:uuid;index;not null" json:"seller_id"`
	Price       float64   `gorm:"type:numeric(18,4);not null" json:"price"`
	Quantity    int64     `gorm:"not null" json:"quantity"`
	Amount      float64   `gorm:"type:numeric(20,2);not null" json:"amount"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

func (t *Trade) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type Holding struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_user_stock;not null" json:"user_id"`
	StockID   uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_user_stock;not null" json:"stock_id"`
	StockCode string    `gorm:"size:16;not null" json:"stock_code"`
	StockName string    `gorm:"size:64;not null" json:"stock_name"`
	Quantity  int64     `gorm:"not null;default:0" json:"quantity"`
	FrozenQty int64     `gorm:"not null;default:0" json:"frozen_qty"`
	AvgCost   float64   `gorm:"type:numeric(18,4);not null;default:0" json:"avg_cost"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *Holding) BeforeCreate(tx *gorm.DB) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return nil
}

func (h *Holding) Available() int64 {
	return h.Quantity - h.FrozenQty
}
