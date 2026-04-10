package cache

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and verifies a Redis client using REDIS_URL.
// If REDIS_URL is empty, it falls back to redis://localhost:6379/0.
func NewRedisClient() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}
