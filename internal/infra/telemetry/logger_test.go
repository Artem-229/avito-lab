package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestLogCorrelation(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(logHandler{slog.NewJSONHandler(&output, nil)}).With("service", "trip-service")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled,
	})
	logger.InfoContext(trace.ContextWithSpanContext(context.Background(), sc), "trip created")
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["trace_id"] != sc.TraceID().String() || entry["span_id"] != sc.SpanID().String() || entry["service"] != "trip-service" {
		t.Fatalf("log=%v", entry)
	}
}
