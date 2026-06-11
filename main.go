package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"gateway/internal/config"
	"gateway/internal/config/router"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	defer logger.Sync()

	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	logger.Info("Connected to Redis", zap.String("addr", cfg.RedisAddr))

	// Создаём роутер
	r := router.NewRouter(cfg, logger, rdb)

	// Настраиваем HTTP сервер
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Логируем запуск
	logger.Info("API Gateway starting",
		zap.String("port", cfg.Port),
		zap.String("backend_url", cfg.BackendURL),
		zap.String("streaming_url", cfg.StreamingURL),
	)

	// Запускаем сервер
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Server failed: %v", zap.Error(err))
	}
}
