package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type logHandler struct{ slog.Handler }

func NewLogger(level slog.Level) *slog.Logger {
	return slog.New(logHandler{slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})}).With("service", "trip-service")
}

func (h logHandler) Handle(ctx context.Context, record slog.Record) error {
	sc := trace.SpanContextFromContext(ctx)
	hasTrace := false
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == "trace_id" {
			hasTrace = true
		}
		return true
	})
	if sc.IsValid() && !hasTrace {
		record.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	return h.Handler.Handle(ctx, record)
}

func (h logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return logHandler{h.Handler.WithAttrs(attrs)}
}
func (h logHandler) WithGroup(name string) slog.Handler { return logHandler{h.Handler.WithGroup(name)} }
