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
	// Инициализация логгера
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

	// Создаём роутер с поддержкой прокси и статики
	r := router.NewRouter(cfg, logger)

	// Настраиваем HTTP сервер
	// ReadTimeout не задан — upload handler требует неограниченного времени
	// на чтение тела POST с видеофайлом и на ожидание ответа от streaming-service
	// (транскодинг может занимать минуты).
	// ReadHeaderTimeout: 10s — защита от slowloris (клиент должен прислать заголовки за 10с).
	// WriteTimeout не задан — ответ от бэкенда может идти долго.
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Логируем запуск
	logger.Info("API Gateway starting",
		zap.String("port", cfg.Port),
		zap.String("backend_url", cfg.BackendURL),
		zap.String("streaming_url", cfg.StreamingURL),
	)

	// Запускаем сервер
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}
