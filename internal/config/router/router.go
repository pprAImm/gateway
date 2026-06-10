package router

import (
	"github.com/pprAImm/gateway/internal/config"
	"github.com/pprAImm/gateway/internal/config/auth"
	"github.com/pprAImm/gateway/internal/config/middleware"

	"net/http"

	"github.com/pprAImm/gateway/internal/config/proxy"

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

	backendProxy := proxy.NewReverseProxy(cfg.BackendURL, log)
	streamingProxy := proxy.NewReverseProxy(cfg.StreamingURL, log)

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

	// 1. API запросы (core-backend) - категории, сериалы, избранное, пользователи
	r.HandleFunc("/api/*", func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Proxying to core-backend",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
		)
		backendProxy.ServeHTTP(w, r)
	})

	// 2. Стриминг запросы (streaming-service) - видео, HLS плейлисты
	r.HandleFunc("/stream/*", func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Proxying to streaming-service",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
		)
		streamingProxy.ServeHTTP(w, r)
	})

	r.HandleFunc("/videos/*", func(w http.ResponseWriter, r *http.Request) {
		streamingProxy.ServeHTTP(w, r)
	})

	// Опционально: статика для фронтенда (если фронт через gateway)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	return r
}
