package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"go.uber.org/zap"
)

// NewReverseProxy создает reverse proxy для бэкенда
func NewReverseProxy(targetURL string, log *zap.Logger) *httputil.ReverseProxy {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		log.Fatal("Failed to parse backend URL", zap.Error(err))
	}

	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	proxy.Transport = &http.Transport{
		MaxIdleConns:            100,
		MaxIdleConnsPerHost:     20,
		IdleConnTimeout:         90 * time.Second,
		DisableCompression:      true,
		ResponseHeaderTimeout:   0, // не ограничиваем — streaming-service может транскодировать минуты
		ExpectContinueTimeout:   0, // не ждём 100-continue
	}

	// Изменяем ответ перед отправкой клиенту
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Добавляем security headers
		resp.Header.Set("X-Content-Type-Options", "nosniff")
		resp.Header.Set("X-Frame-Options", "DENY")
		resp.Header.Set("X-XSS-Protection", "1; mode=block")
		return nil
	}

	// Обработка ошибок прокси
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("Proxy error",
			zap.Error(err),
			zap.String("path", r.URL.Path),
		)
		http.Error(w, "Gateway error", http.StatusBadGateway)
	}

	return proxy
}
