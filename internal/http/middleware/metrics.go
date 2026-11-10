package middleware

import (
	"fmt"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func Metrics(meter metric.Meter, route func(*http.Request) string) (func(http.Handler) http.Handler, error) {
	duration, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("create HTTP duration: %w", err)
	}
	active, err := meter.Int64UpDownCounter("http.server.active_requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, fmt.Errorf("create active HTTP requests: %w", err)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writer := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			attrs := []attribute.KeyValue{attribute.String("http.request.method", r.Method), attribute.String("http.route", route(r))}
			active.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			completed := false
			defer func() {
				active.Add(r.Context(), -1, metric.WithAttributes(attrs...))
				attrs = append(attrs, attribute.Int("http.response.status_code", responseStatus(writer, completed)))
				duration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(attrs...))
			}()
			next.ServeHTTP(writer, r)
			completed = true
		})
	}, nil
}
