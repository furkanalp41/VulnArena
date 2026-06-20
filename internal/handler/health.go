package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "vulnarena-api",
	})
}

// Readiness returns a deep readiness handler that verifies connectivity to
// Postgres and Redis. It responds 503 if either dependency is unreachable,
// with a JSON body indicating which dependency is down.
func Readiness(pool *pgxpool.Pool, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		deps := map[string]string{
			"postgres": "ok",
			"redis":    "ok",
		}
		ready := true

		if pool == nil || pool.Ping(ctx) != nil {
			deps["postgres"] = "down"
			ready = false
		}

		if redisClient == nil || redisClient.Ping(ctx).Err() != nil {
			deps["redis"] = "down"
			ready = false
		}

		status := http.StatusOK
		state := "ready"
		if !ready {
			status = http.StatusServiceUnavailable
			state = "not_ready"
		}

		writeJSON(w, status, map[string]any{
			"status":       state,
			"dependencies": deps,
		})
	}
}
