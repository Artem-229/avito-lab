package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func Logging(logger *slog.Logger, route func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writer := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			completed := false
			defer func() {
				status := responseStatus(writer, completed)
				level := slog.LevelInfo
				if status >= 500 {
					level = slog.LevelError
				}
				args := make([]any, 0, 14)
				args = append(args, "service", "trip-service", "method", r.Method, "route", route(r), "status", status, "duration_ms", time.Since(start).Milliseconds())
				args = append(args, traceFields(r.Context())...)
				logger.Log(r.Context(), level, "request handled", args...)
			}()
			next.ServeHTTP(writer, r)
			completed = true
		})
	}
}
