package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// AuditLog wraps a handler group to emit structured audit log entries
// for every request that mutates state (POST, PUT, PATCH, DELETE).
func AuditLog(logger *slog.Logger, category string) func(http.Handler) http.Handler {
	audit := logger.With(slog.String("log_type", "audit"), slog.String("category", category))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &wrappedResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			// Only audit state-changing requests
			if r.Method == http.MethodGet || r.Method == http.MethodOptions || r.Method == http.MethodHead {
				return
			}

			attrs := []any{
				slog.String("action", r.Method+" "+r.URL.Path),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote_ip", r.RemoteAddr),
			}

			// Include user identity if available
			if uid := UserIDFromContext(r.Context()); uid.String() != "00000000-0000-0000-0000-000000000000" {
				attrs = append(attrs, slog.String("user_id", uid.String()))
			}

			if role := UserRoleFromContext(r.Context()); role != "" {
				attrs = append(attrs, slog.String("user_role", role))
			}

			if wrapped.statusCode >= 400 {
				audit.Warn("audit_event", attrs...)
			} else {
				audit.Info("audit_event", attrs...)
			}
		})
	}
}
