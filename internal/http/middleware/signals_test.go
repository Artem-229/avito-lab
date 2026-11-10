package middleware_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Artem-229/avito-lab/internal/http/middleware"
)

func TestHTTPSpansAndMetrics(t *testing.T) {
	for _, status := range []int{201, 409, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
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
			chain, err := middleware.New(slog.New(slog.NewJSONHandler(io.Discard, nil)), traces.Tracer("test"), meters.Meter("test"), func(*http.Request) string { return "/trips/{id}" })
			if err != nil {
				t.Fatal(err)
			}
			handler := chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if status == 500 {
					panic("test panic")
				}
				w.WriteHeader(status)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/trips/123", nil))
			spans := recorder.Ended()
			if response.Code != status || len(spans) != 1 || spans[0].Name() != "GET /trips/{id}" {
				t.Fatalf("status=%d spans=%v", response.Code, spans)
			}
			attrs := attribute.NewSet(spans[0].Attributes()...)
			if got, _ := attrs.Value("http.response.status_code"); got.AsInt64() != int64(status) {
				t.Fatalf("span status=%v", got)
			}
			if status == 500 {
				if got, _ := attrs.Value("error.type"); spans[0].Status().Code != codes.Error || got.AsString() != "panic" {
					t.Fatalf("panic span=%v", spans[0])
				}
			}
			var data metricdata.ResourceMetrics
			if err := reader.Collect(ctx, &data); err != nil {
				t.Fatal(err)
			}
			foundDuration, foundActive := false, false
			for _, scope := range data.ScopeMetrics {
				for _, measurement := range scope.Metrics {
					switch measurement.Name {
					case "http.server.request.duration":
						foundDuration = true
						histogram, ok := measurement.Data.(metricdata.Histogram[float64])
						if !ok || measurement.Unit != "s" || len(histogram.DataPoints) != 1 || histogram.DataPoints[0].Count != 1 {
							t.Fatalf("HTTP duration=%+v", measurement)
						}
						point := histogram.DataPoints[0]
						if got, _ := point.Attributes.Value("http.route"); got.AsString() != "/trips/{id}" {
							t.Fatalf("route=%v", got)
						}
						if got, _ := point.Attributes.Value("http.response.status_code"); got.AsInt64() != int64(status) {
							t.Fatalf("metric status=%v", got)
						}
					case "http.server.active_requests":
						foundActive = true
						sum, ok := measurement.Data.(metricdata.Sum[int64])
						if !ok || sum.IsMonotonic || len(sum.DataPoints) != 1 || sum.DataPoints[0].Value != 0 {
							t.Fatalf("active=%+v", measurement)
						}
					}
				}
			}
			if !foundDuration || !foundActive {
				t.Fatal("missing HTTP metrics")
			}
		})
	}
}
