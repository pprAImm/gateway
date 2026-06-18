package router

import (
	"github.com/pprAImm/gateway/internal/config"
	"github.com/pprAImm/gateway/internal/config/middleware"

	"net/http"
	"strings"

	"github.com/pprAImm/gateway/internal/config/proxy"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// NewRouter создает маршрутизатор API Gateway
func NewRouter(cfg *config.Config, log *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Глобальные middleware
	r.Use(chimiddleware.Recoverer)           // Встроенный recovery
	r.Use(chimiddleware.RealIP)              // Определяем реальный IP
	r.Use(chimiddleware.CleanPath)           // Очищаем путь
	r.Use(chimiddleware.StripSlashes)        // Убираем слеши
	r.Use(middleware.LoggingMiddleware(log)) // Логирование запросов

	// CORS — разрешаем запросы с любых источников (фронтенд на любом порту/домене)
	r.Use(corsMiddleware)

	// Rate limiting (глобальный)
	r.Use(middleware.RateLimitMiddleware(cfg.RateLimitRequests, int(cfg.RateLimitWindow.Seconds())))

	backendProxy := proxy.NewReverseProxy(cfg.BackendURL, log)
	streamingProxy := proxy.NewReverseProxy(cfg.StreamingURL, log)

	// Health checks (не проксируются)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// 1. API запросы (core-backend) - категории, сериалы, избранное, пользователи
	r.HandleFunc("/api/*", func(w http.ResponseWriter, r *http.Request) {
		// Strip the /api prefix before proxying to backend
		origPath := r.URL.Path
		r2 := r.Clone(r.Context())
		r2.URL.Path = strings.TrimPrefix(origPath, "/api")

		if userID := r2.Header.Get("X-User-Id"); userID != "" {
			log.Debug("Proxying to core-backend with user_id",
				zap.String("path", r2.URL.Path),
				zap.String("method", r2.Method),
				zap.String("x-user-id", userID),
			)
		} else {
			log.Debug("Proxying to core-backend (public)",
				zap.String("path", r2.URL.Path),
				zap.String("method", r2.Method),
			)
		}
		backendProxy.ServeHTTP(w, r2)
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

	// 3. Загрузка видео (streaming-service)
	r.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		streamingProxy.ServeHTTP(w, r)
	})

	// 4. Загруженные файлы (обложки) — прокси в core-backend без изменения пути
	r.HandleFunc("/uploads/*", func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Proxying to core-backend for uploads",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
		)
		backendProxy.ServeHTTP(w, r)
	})

	// 5. Статика фронтенда — раздаём через gateway (всё на одном порту, без CORS)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Страница подтверждения email
	r.Get("/verify", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./frontend/verify.html")
	})

	// Корень → центральная страница
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./frontend/central.html")
	})

	// Все остальные пути — отдача фронтенда
	r.NotFound(http.FileServer(http.Dir("./frontend")).ServeHTTP)

	return r
}

// corsMiddleware разрешает CORS-запросы с любого источника
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-User-Id")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
