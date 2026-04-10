package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

func RateLimit(scope string, maxRequests int64, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if redisClient == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := getRequestIP(r)
			key := fmt.Sprintf("rate_limit:%s:%s", scope, ip)

			count, err := redisClient.Incr(r.Context(), key).Result()
			if err == nil && count == 1 {
				_ = redisClient.Expire(r.Context(), key, window).Err()
			}

			if err == nil && count > maxRequests {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getRequestIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}

	if r.RemoteAddr == "" {
		return "unknown"
	}
	return r.RemoteAddr
}
