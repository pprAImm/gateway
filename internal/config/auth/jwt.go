package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

// ValidateJWT проверяет JWT токен и возвращает user_id
func ValidateJWT(tokenString string, secret []byte) (int, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserID, nil
	}

	return 0, errors.New("invalid token")
}

// ExtractToken извлекает токен из заголовка Authorization
func ExtractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// Middleware проверяет JWT и добавляет user_id в контекст
func Middleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Пропускаем публичные эндпоинты
			if isPublicRoute(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			token := ExtractToken(r)
			if token == "" {
				http.Error(w, "Missing authorization token", http.StatusUnauthorized)
				return
			}

			userID, err := ValidateJWT(token, secret)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Добавляем user_id в контекст и заголовок для бэкенда
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			r = r.WithContext(ctx)
			r.Header.Set("X-User-Id", string(rune(userID)))

			next.ServeHTTP(w, r)
		})
	}
}

// isPublicRoute определяет, какие пути не требуют авторизации
func isPublicRoute(path string) bool {
	publicPaths := []string{
		"/health",
		"/ready",
		"/api/auth/login",
		"/api/auth/register",
		"/api/categories",
		"/api/series",
	}
	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}
	// Проверка на динамические пути: /api/series/123, /api/episodes/456
	if strings.HasPrefix(path, "/api/series/") && len(strings.Split(path, "/")) == 4 {
		return true
	}
	if strings.HasPrefix(path, "/api/episodes/") && len(strings.Split(path, "/")) == 4 {
		return true
	}
	return false
}
