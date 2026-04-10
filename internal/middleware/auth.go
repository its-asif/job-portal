package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/its-asif/job-portal/internal/auth"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const userClaimsContextKey contextKey = "userClaims"

var redisClient *redis.Client

func SetRedisClient(client *redis.Client) {
	redisClient = client
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if authorization == "" {
			respondUnauthorized(w, "missing authorization header")
			return
		}

		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			respondUnauthorized(w, "invalid authorization header")
			return
		}

		token := strings.TrimSpace(parts[1])
		if token == "" {
			respondUnauthorized(w, "missing bearer token")
			return
		}

		if redisClient != nil {
			blacklisted, err := redisClient.Exists(r.Context(), "blacklist:"+token).Result()
			if err == nil && blacklisted > 0 {
				respondUnauthorized(w, "token has been revoked")
				return
			}
		}

		claims, err := auth.ValidateToken(token)
		if err != nil {
			respondUnauthorized(w, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(userClaimsContextKey).(*auth.Claims)
	if !ok || claims == nil {
		return nil, false
	}
	return claims, true
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaimsFromContext(r.Context())
			if !ok {
				respondUnauthorized(w, "unauthorized")
				return
			}

			if !strings.EqualFold(claims.Role, role) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
