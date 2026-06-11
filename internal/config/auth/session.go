package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// Middleware проверяет сессию по cookie и добавляет user_id в контекст
func Middleware(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Пропускаем публичные эндпоинты
			if isPublicRoute(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Извлекаем session_id из cookie
			sessionCookie, err := r.Cookie("session_id")
			if err != nil {
				zap.L().Warn("No session cookie", zap.String("path", r.URL.Path))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Получаем user_id из Redis по session_id
			ctx := context.Background()
			userIDStr, err := rdb.Get(ctx, "session:"+sessionCookie.Value).Result()
			if err != nil {
				if err == redis.Nil {
					zap.L().Warn("Invalid session", zap.String("session_id", sessionCookie.Value))
					http.Error(w, "Invalid session", http.StatusUnauthorized)
					return
				}
				zap.L().Error("Redis error", zap.Error(err))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Конвертируем userID в int (если нужно)
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				zap.L().Error("Invalid user_id format in Redis", zap.String("user_id", userIDStr))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Добавляем user_id в контекст для дальнейшего использования
			ctx = context.WithValue(r.Context(), UserIDKey, userID)
			r = r.WithContext(ctx)

			// Прокидываем user_id в заголовок для бэкенда
			r.Header.Set("X-User-Id", userIDStr)

			next.ServeHTTP(w, r)
		})
	}
}

// isPublicRoute определяет, какие пути не требуют авторизации (с поддержкой префиксов)
func isPublicRoute(path string) bool {
	// Точные совпадения
	publicExact := []string{
		"/health",
		"/ready",
		"/api/auth/login",
		"/api/auth/register",
		"/api/auth/logout",
	}

	for _, p := range publicExact {
		if path == p {
			return true
		}
	}

	// Префиксные совпадения (для вложенных маршрутов)
	publicPrefixes := []string{
		"/api/categories/",
		"/api/series/",
		"/api/episodes/",
		"/api/series/search",
	}

	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
