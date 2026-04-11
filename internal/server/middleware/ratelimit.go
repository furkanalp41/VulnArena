package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimit applies a general rate limit based on user ID or remote IP.
func RateLimit(redisClient *redis.Client, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimiter(redisClient, "ratelimit", maxRequests, window)
}

// StrictRateLimit applies a tighter rate limit for sensitive endpoints (auth, API key management).
// Uses the route path as part of the key to isolate limits per endpoint group.
func StrictRateLimit(redisClient *redis.Client, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimiter(redisClient, "strict_rl", maxRequests, window)
}

func rateLimiter(redisClient *redis.Client, prefix string, maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use user ID if authenticated, otherwise IP
			identifier := r.RemoteAddr
			if uid := UserIDFromContext(r.Context()); uid.String() != "00000000-0000-0000-0000-000000000000" {
				identifier = uid.String()
			}

			key := fmt.Sprintf("%s:%s", prefix, identifier)
			ctx := context.Background()

			count, err := redisClient.Incr(ctx, key).Result()
			if err != nil {
				// If Redis is down, allow the request
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				redisClient.Expire(ctx, key, window)
			}

			remaining := int64(maxRequests) - count
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			if count > int64(maxRequests) {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
