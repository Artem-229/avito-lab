package middleware

import (
	"context"
	"log/slog"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func New(logger *slog.Logger, tracer trace.Tracer, meter metric.Meter, route func(*http.Request) string) (func(http.Handler) http.Handler, error) {
	metrics, err := Metrics(meter, route)
	if err != nil {
		return nil, err
	}
	return func(handler http.Handler) http.Handler {
		handler = metrics(handler)
		handler = Logging(logger, route)(handler)
		handler = Tracing(tracer, route)(handler)
		handler = Recover(logger)(handler)
		return handler
	}, nil
}

func responseStatus(w chimiddleware.WrapResponseWriter, completed bool) int {
	if status := w.Status(); status != 0 {
		return status
	}
	if !completed {
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

func traceFields(ctx context.Context) []any {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}
	return []any{"trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String()}
}
