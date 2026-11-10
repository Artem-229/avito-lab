package middleware

import (
	"errors"
	"net/http"
	"strconv"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tracer trace.Tracer, route func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parent := propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			ctx, span := tracer.Start(parent, r.Method+" "+route(r), trace.WithSpanKind(trace.SpanKindServer))
			writer := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			completed := false
			defer func() {
				defer span.End()
				status := responseStatus(writer, completed)
				span.SetAttributes(attribute.String("http.request.method", r.Method), attribute.String("http.route", route(r)), attribute.Int("http.response.status_code", status))
				if !completed {
					span.RecordError(errors.New("handler panic"))
					span.SetStatus(codes.Error, "handler panic")
					span.SetAttributes(attribute.String("error.type", "panic"))
				} else if status >= 500 {
					span.SetStatus(codes.Error, http.StatusText(status))
					span.SetAttributes(attribute.String("error.type", strconv.Itoa(status)))
				}
			}()
			*r = *r.WithContext(ctx)
			next.ServeHTTP(writer, r)
			completed = true
		})
	}
}
