package telemetry

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type DBTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

type (
	queryKey  struct{}
	queryInfo struct {
		started               time.Time
		operation, collection string
	}
)

func NewDBTracer(tracer trace.TracerProvider, meter metric.MeterProvider) (*DBTracer, error) {
	duration, err := meter.Meter("trip-service/postgres").Float64Histogram("db.client.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	return &DBTracer{
		Tracer:   otelpgx.NewTracer(otelpgx.WithTracerProvider(tracer), otelpgx.WithMeterProvider(noop.NewMeterProvider()), otelpgx.WithDisableSQLStatementInAttributes()),
		duration: duration,
	}, nil
}

func (t *DBTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	info := queryInfo{started: time.Now()}
	words := strings.Fields(data.SQL)
	if len(words) > 0 {
		info.operation = strings.ToUpper(words[0])
	}
	for _, word := range words {
		switch word = strings.Trim(word, "\";"); word {
		case "trips", "trip_status_history", "idempotency_keys":
			info.collection = word
		}
	}
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("db.operation.name", info.operation), attribute.String("db.collection.name", info.collection))
	return context.WithValue(ctx, queryKey{}, info)
}

func (t *DBTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	info, ok := ctx.Value(queryKey{}).(queryInfo)
	if ok {
		errorType := ""
		if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
			errorType = "database_error"
			span := trace.SpanFromContext(ctx)
			span.SetStatus(codes.Error, "database operation failed")
			span.SetAttributes(attribute.String("error.type", errorType))
		}
		t.duration.Record(ctx, time.Since(info.started).Seconds(), metric.WithAttributes(attribute.String("db.operation.name", info.operation), attribute.String("db.collection.name", info.collection), attribute.String("error.type", errorType)))
	}
	t.Tracer.TraceQueryEnd(ctx, conn, data)
}
