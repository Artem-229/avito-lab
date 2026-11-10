package middleware_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/Artem-229/avito-lab/internal/http/middleware"
)

func TestRequestLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name               string
		code               int
		panicValue         bool
		path, route, level string
	}{
		{"success", 201, false, "/trips/123", "/trips/{id}", "INFO"},
		{"client error", 409, false, "/trips/123", "/trips/{id}", "INFO"},
		{"server error", 503, false, "/trips/123", "/trips/{id}", "ERROR"},
		{"panic", 500, true, "/trips/123", "/trips/{id}", "ERROR"},
		{"not found", 404, false, "/missing/123", "", "INFO"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			router := chi.NewRouter()
			router.Get("/trips/{id}", func(w http.ResponseWriter, _ *http.Request) {
				if tc.panicValue {
					panic("private panic detail")
				}
				w.WriteHeader(tc.code)
			})
			chain, err := middleware.New(logger, tracenoop.NewTracerProvider().Tracer("test"), noop.NewMeterProvider().Meter("test"), func(r *http.Request) string {
				ctx := chi.NewRouteContext()
				if router.Match(ctx, r.Method, r.URL.Path) {
					return ctx.RoutePattern()
				}
				return ""
			})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
			rec := httptest.NewRecorder()
			chain(router).ServeHTTP(rec, req)
			if rec.Code != tc.code {
				t.Fatalf("status=%d, want %d", rec.Code, tc.code)
			}
			if tc.panicValue {
				var body struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Code != "internal_error" || strings.Contains(rec.Body.String(), "private") {
					t.Fatalf("panic response=%s", rec.Body.String())
				}
			}
			decoder := json.NewDecoder(&output)
			accessLogs, panicLogs := 0, 0
			for {
				var entry map[string]any
				err := decoder.Decode(&entry)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				if entry["msg"] == "request panic" {
					panicLogs++
					if entry["stack"] == "" || entry["stack"] == nil {
						t.Fatal("missing panic stack")
					}
				}
				if entry["msg"] != "request handled" {
					continue
				}
				accessLogs++
				if entry["status"] != float64(tc.code) || entry["route"] != tc.route || entry["level"] != tc.level {
					t.Fatalf("access log=%v", entry)
				}
				if entry["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
					t.Fatalf("trace correlation=%v", entry)
				}
			}
			if accessLogs != 1 {
				t.Fatalf("access logs=%d, want 1", accessLogs)
			}
			if tc.panicValue && panicLogs != 1 {
				t.Fatalf("panic logs=%d, want 1", panicLogs)
			}
		})
	}
}

func TestRecoverAfterResponseStarted(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	handler := middleware.Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		if _, err := io.WriteString(w, "partial response"); err != nil {
			t.Fatal(err)
		}
		panic("failure after headers were sent")
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusAccepted || rec.Body.String() != "partial response" {
		t.Fatalf("committed response was overwritten: status=%d, body=%q", rec.Code, rec.Body.String())
	}
}
