package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleware возвращает middleware для ограничения частоты запросов
func RateLimitMiddleware(rdb *redis.Client, requests int, windowSeconds int) func(http.Handler) http.Handler {
	// Преобразуем секунды в time.Duration
	window := time.Duration(windowSeconds) * time.Second
	return httprate.LimitByIP(requests, window)
}

// RateLimitByPath создает разные лимиты для разных путей
func RateLimitByPath(requests int, windowSeconds int) func(http.Handler) http.Handler {
	window := time.Duration(windowSeconds) * time.Second
	return httprate.LimitByIP(requests, window)
}
