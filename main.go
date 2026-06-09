package main

import (
	"log"
	"net/http"
	"time"

	"gateway/internal/config"
	"gateway/internal/config/router"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Создаём роутер
	r := router.NewRouter(cfg, nil, nil)

	// Настраиваем HTTP сервер
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Логируем запуск
	log.Printf("API Gateway starting on port %s", cfg.Port)
	log.Printf("Backend URL: %s", cfg.BackendURL)
	log.Printf("Health check: http://localhost:%s/health", cfg.Port)

	// Запускаем сервер
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
