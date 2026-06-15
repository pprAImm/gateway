package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	BackendURL        string
	RedisAddr         string
	RedisPassword     string
	StreamingURL      string
	RedisDB           int
	RateLimitRequests int
	RateLimitWindow   time.Duration
	LogLevel          string
}

// getEnv - вспомогательная функция (должна быть определена ДО использования)
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func Load() (*Config, error) {
	// Загружаем .env файл (игнорируем ошибку, если файла нет)
	_ = godotenv.Load()

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	rateLimitRequests, _ := strconv.Atoi(getEnv("RATE_LIMIT_REQUESTS", "100"))
	rateLimitWindow, _ := strconv.ParseInt(getEnv("RATE_LIMIT_WINDOW", "60"), 10, 64)

	return &Config{
		Port:              getEnv("PORT", "8081"),
		BackendURL:        getEnv("BACKEND_URL", "http://localhost:8080"),
		StreamingURL:      getEnv("STREAMING_URL", "http://localhost:8082"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           redisDB,
		RateLimitRequests: rateLimitRequests,
		RateLimitWindow:   time.Duration(rateLimitWindow) * time.Second,
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}, nil
}
