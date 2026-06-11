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
	r.Use(chimiddleware.RealIP)               // определение реального IP
	r.Use(chimiddleware.CleanPath)            // очистка путя
	r.Use(chimiddleware.StripSlashes)         // убираем слеши
	r.Use(middleware.RecoveryMiddleware(log)) //восстановление
	r.Use(middleware.LoggingMiddleware(log))  // логирование запросов

	// Rate limiting (глобальный)
	r.Use(middleware.RateLimitMiddleware(rdb, cfg.RateLimitRequests, int(cfg.RateLimitWindow.Seconds())))

	r.Use(auth.Middleware(rdb))

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
		if userID := r.Header.Get("X-User-Id"); userID != "" {
			log.Debug("Proxying to core-backend with user_id",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
				zap.String("x-user-id", userID),
			)
		} else {
			log.Debug("Proxying to core-backend (public)",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
			)
		}
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
