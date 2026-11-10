package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/Artem-229/avito-lab/internal/http/domainerror"
	"github.com/Artem-229/avito-lab/internal/http/logcontext"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*r = *r.WithContext(logcontext.WithLogger(r.Context(), logger))
			writer := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				if value := recover(); value != nil {
					logger.ErrorContext(r.Context(), "request panic", "panic", value, "stack", string(debug.Stack()))
					if writer.Status() == 0 {
						domainerror.WriteError(writer, r.WithContext(logcontext.WithLogger(r.Context(), nil)), errors.New("handler panic"))
					}
				}
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
