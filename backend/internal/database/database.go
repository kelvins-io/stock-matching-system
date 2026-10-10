package database

import (
	"log"
	"time"

	"github.com/kelvins-io/stock-matching-system/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Stock{},
		&models.MarketRule{},
		&models.Order{},
		&models.Trade{},
		&models.Holding{},
	)
}

func Seed(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		admin := models.User{
			Username:     "admin",
			PasswordHash: string(hash),
			Role:         models.RoleAdmin,
			Balance:      0,
		}
		if err := db.Create(&admin).Error; err != nil {
			return err
		}
		log.Println("seeded admin user: admin / admin123")
	}

	var ruleCount int64
	db.Model(&models.MarketRule{}).Count(&ruleCount)
	if ruleCount == 0 {
		rule := models.MarketRule{
			OpenAM:      "09:30",
			CloseAM:     "11:30",
			OpenPM:      "13:00",
			ClosePM:     "15:00",
			TradingDays: "1,2,3,4,5",
			ForceClosed: false,
			ForceOpen:   true, // 演示环境默认强制开市，便于测试撮合
		}
		if err := db.Create(&rule).Error; err != nil {
			return err
		}
		log.Println("seeded default market rules (force_open=true for demo)")
	}

	var stockCount int64
	db.Model(&models.Stock{}).Count(&stockCount)
	if stockCount == 0 {
		demo := []models.Stock{
			{Code: "600000", Name: "浦发银行", TotalShares: 1000000000, PriceLimitPct: 10, IssuePrice: 10.50, PrevClose: 10.50, CurrentPrice: 10.50, HighPrice: 10.50, LowPrice: 10.50, Status: models.StockActive},
			{Code: "000001", Name: "平安银行", TotalShares: 800000000, PriceLimitPct: 10, IssuePrice: 12.80, PrevClose: 12.80, CurrentPrice: 12.80, HighPrice: 12.80, LowPrice: 12.80, Status: models.StockActive},
			{Code: "600519", Name: "贵州茅台", TotalShares: 125000000, PriceLimitPct: 10, IssuePrice: 1680.00, PrevClose: 1680.00, CurrentPrice: 1680.00, HighPrice: 1680.00, LowPrice: 1680.00, Status: models.StockActive},
		}
		for i := range demo {
			if err := db.Create(&demo[i]).Error; err != nil {
				return err
			}
		}
		log.Println("seeded demo stocks")
	}

	// 演示卖方账户：持有部分股票，便于撮合买卖测试
	var traderCount int64
	db.Model(&models.User{}).Where("username = ?", "trader").Count(&traderCount)
	if traderCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("trader123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		trader := models.User{
			Username:     "trader",
			PasswordHash: string(hash),
			Role:         models.RoleUser,
			Balance:      1000000,
		}
		if err := db.Create(&trader).Error; err != nil {
			return err
		}
		var stocks []models.Stock
		if err := db.Find(&stocks).Error; err != nil {
			return err
		}
		for _, st := range stocks {
			h := models.Holding{
				UserID:    trader.ID,
				StockID:   st.ID,
				StockCode: st.Code,
				StockName: st.Name,
				Quantity:  10000,
				AvgCost:   st.IssuePrice,
			}
			if err := db.Create(&h).Error; err != nil {
				return err
			}
		}
		log.Println("seeded demo trader: trader / trader123 (with holdings)")
	}

	return nil
}
