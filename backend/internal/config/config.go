package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisAddr   string
	RedisPass   string
	RedisDB     int
	JWTSecret   string
	JWTExpire   time.Duration
}

func Load() *Config {
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	expireHours, _ := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))

	return &Config{
		Port:        getEnv("PORT", "10020"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://stock:stock123@localhost:5432/stock_matching?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:     redisDB,
		JWTSecret:   getEnv("JWT_SECRET", "stock-matching-jwt-secret-change-me"),
		JWTExpire:   time.Duration(expireHours) * time.Hour,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
