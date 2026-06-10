package router

import (
	"gateway/internal/config"
	"gateway/internal/config/auth"
	"gateway/internal/config/middleware"

	"gateway/internal/config/proxy"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NewRouter создает маршрутизатор API Gateway
func NewRouter(cfg *config.Config, log *zap.Logger, rdb *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	// Глобальные middleware
	r.Use(chimiddleware.Recoverer)           // Встроенный recovery
	r.Use(chimiddleware.RealIP)              // Определяем реальный IP
	r.Use(chimiddleware.CleanPath)           // Очищаем путь
	r.Use(chimiddleware.StripSlashes)        // Убираем слеши
	r.Use(middleware.LoggingMiddleware(log)) // Логирование запросов

	// Rate limiting (глобальный)
	r.Use(middleware.RateLimitMiddleware(rdb, cfg.RateLimitRequests, int(cfg.RateLimitWindow.Seconds())))

	// JWT авторизация
	r.Use(auth.Middleware(cfg.JWTSecret))

	// Создаем reverse proxy для бэкенда
	backendProxy := proxy.NewReverseProxy(cfg.BackendURL, log)

	// Health checks (не проксируются)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Проверяем подключение к Redis и бэкенду
		if err := rdb.Ping(r.Context()).Err(); err != nil {
			log.Error("Redis not ready", zap.Error(err))
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// Все API запросы проксируются в бэкенд
	r.HandleFunc("/api/*", func(w http.ResponseWriter, r *http.Request) {
		// Удаляем префикс /api при проксировании (если нужно)
		// backendProxy.ServeHTTP(w, r) - если бэкенд ожидает /api/*
		backendProxy.ServeHTTP(w, r)
	})

	// Опционально: статика для фронтенда (если фронт через gateway)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	return r
}
