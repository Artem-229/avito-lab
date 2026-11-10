package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Providers struct {
	Tracer *sdktrace.TracerProvider
	Meter  *sdkmetric.MeterProvider
}

func New(ctx context.Context) (*Providers, error) {
	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", "trip-service"),
		attribute.String("service.version", "homework-02"),
		attribute.String("service.namespace", "tripgo"),
		attribute.String("deployment.environment", "local"),
	), resource.WithFromEnv())
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}
	var traceExporter sdktrace.SpanExporter
	var metricExporter sdkmetric.Exporter
	switch protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); protocol {
	case "grpc":
		traceExporter, err = otlptracegrpc.New(ctx)
	case "", "http/protobuf":
		traceExporter, err = otlptracehttp.New(ctx)
	default:
		return nil, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL %q", protocol)
	}
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}
	if os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL") == "grpc" {
		metricExporter, err = otlpmetricgrpc.New(ctx)
	} else {
		metricExporter, err = otlpmetrichttp.New(ctx)
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("metric exporter: %w", err), traceExporter.Shutdown(ctx))
	}
	return &Providers{
		Tracer: sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(traceExporter)),
		Meter:  sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter))),
	}, nil
}

func (p *Providers) Shutdown(ctx context.Context) error {
	return errors.Join(p.Meter.Shutdown(ctx), p.Tracer.Shutdown(ctx))
}
