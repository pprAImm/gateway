package main

import (
	"log"
	"net/http"
	"time"

	"github.com/pprAImm/gateway/internal/config"
	"github.com/pprAImm/gateway/internal/config/router"
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

	// Создаём роутер без Redis-зависимости
	r := router.NewRouter(cfg, logger)

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
