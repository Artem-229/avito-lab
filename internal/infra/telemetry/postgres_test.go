package telemetry

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestDatabaseSignals(t *testing.T) {
	ctx := context.Background()
	recorder := tracetest.NewSpanRecorder()
	traces := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	reader := sdkmetric.NewManualReader()
	meters := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		if err := errors.Join(traces.Shutdown(ctx), meters.Shutdown(ctx)); err != nil {
			t.Error(err)
		}
	})
	tracer, err := NewDBTracer(traces, meters)
	if err != nil {
		t.Fatal(err)
	}
	parentCtx, parent := traces.Tracer("test").Start(ctx, "HTTP")
	queryCtx := tracer.TraceQueryStart(parentCtx, nil, pgx.TraceQueryStartData{SQL: "SELECT id FROM trips WHERE id = $1"})
	tracer.TraceQueryEnd(queryCtx, nil, pgx.TraceQueryEndData{Err: errors.New("database failure")})
	parent.End()
	spans := recorder.Ended()
	if len(spans) != 2 || spans[0].Status().Code != codes.Error || spans[0].Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Fatalf("spans=%v", spans)
	}
	attrs := attribute.NewSet(spans[0].Attributes()...)
	for key, want := range map[attribute.Key]string{"db.operation.name": "SELECT", "db.collection.name": "trips", "error.type": "database_error"} {
		got, ok := attrs.Value(key)
		if !ok || got.AsString() != want {
			t.Fatalf("span attribute %s=%v, want %s", key, got, want)
		}
	}
	var data metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &data); err != nil {
		t.Fatal(err)
	}
	for _, scope := range data.ScopeMetrics {
		for _, measurement := range scope.Metrics {
			if measurement.Name != "db.client.operation.duration" {
				continue
			}
			histogram, ok := measurement.Data.(metricdata.Histogram[float64])
			if !ok || measurement.Unit != "s" || len(histogram.DataPoints) != 1 || histogram.DataPoints[0].Count != 1 {
				t.Fatalf("DB metric=%+v", measurement)
			}
			value, _ := histogram.DataPoints[0].Attributes.Value("db.collection.name")
			if value.AsString() != "trips" {
				t.Fatalf("DB metric collection=%s", value.AsString())
			}
			return
		}
	}
	t.Fatal("DB duration metric missing")
}
