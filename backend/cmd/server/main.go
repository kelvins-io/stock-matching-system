package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/stock-matching-system/internal/cache"
	"github.com/kelvins-io/stock-matching-system/internal/config"
	"github.com/kelvins-io/stock-matching-system/internal/database"
	"github.com/kelvins-io/stock-matching-system/internal/handlers"
	"github.com/kelvins-io/stock-matching-system/internal/middleware"
	"github.com/kelvins-io/stock-matching-system/internal/services"
)

func main() {
	cfg := config.Load()

	var dbConnErr error
	db, dbConnErr := database.Connect(cfg.DatabaseURL)
	for i := 0; i < 30 && dbConnErr != nil; i++ {
		log.Printf("waiting for postgres... (%v)", dbConnErr)
		time.Sleep(2 * time.Second)
		db, dbConnErr = database.Connect(cfg.DatabaseURL)
	}
	if dbConnErr != nil {
		log.Fatalf("postgres connect failed: %v", dbConnErr)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("migrate failed: %v", err)
	}
	if err := database.Seed(db); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	var rdb *cache.RedisCache
	var redisErr error
	for i := 0; i < 30; i++ {
		rdb, redisErr = cache.New(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
		if redisErr == nil {
			break
		}
		log.Printf("waiting for redis... (%v)", redisErr)
		time.Sleep(2 * time.Second)
	}
	if redisErr != nil {
		log.Fatalf("redis connect failed: %v", redisErr)
	}

	jwtAuth := middleware.NewJWTAuth(cfg.JWTSecret, cfg.JWTExpire)
	authSvc := services.NewAuthService(db, jwtAuth)
	stockSvc := services.NewStockService(db, rdb)
	marketSvc := services.NewMarketService(db, rdb)
	orderSvc := services.NewOrderService(db)
	matcher := services.NewMatchingEngine(db, rdb, marketSvc)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger(), middleware.CORS())

	h := handlers.New(authSvc, stockSvc, marketSvc, orderSvc, matcher, jwtAuth)
	h.RegisterRoutes(r)

	log.Printf("stock matching API listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
