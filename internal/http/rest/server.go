package rest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/domainerror"
	"github.com/Artem-229/avito-lab/internal/http/middleware"
)

type Config struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	QueryTimeout      time.Duration
}

type Server struct {
	server *http.Server
	logger *slog.Logger
}

func NewServer(cfg Config, handlers api.ServerInterface, logger *slog.Logger, tracer trace.TracerProvider, meter metric.MeterProvider) (*Server, error) {
	spec, err := api.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	spec.Servers = nil
	openapi3.SchemaErrorDetailsDisabled = true

	validator := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, _ nethttpmiddleware.ErrorHandlerOpts) {
			domainerror.WriteParamError(w, r, err)
		},
	})

	router := chi.NewRouter()
	handler := api.HandlerWithOptions(handlers, api.ChiServerOptions{
		BaseRouter:       router,
		Middlewares:      []api.MiddlewareFunc{validator},
		ErrorHandlerFunc: domainerror.WriteParamError,
	})
	chain, err := middleware.New(logger, tracer.Tracer("trip-service/http"), meter.Meter("trip-service/http"), func(r *http.Request) string {
		ctx := chi.NewRouteContext()
		if router.Match(ctx, r.Method, r.URL.Path) {
			return ctx.RoutePattern()
		}
		return ""
	})
	if err != nil {
		return nil, fmt.Errorf("create HTTP middleware: %w", err)
	}

	return &Server{
		server: &http.Server{
			Addr: cfg.Addr,
			Handler: chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx, cancel := context.WithTimeout(r.Context(), cfg.QueryTimeout)
				defer cancel()
				handler.ServeHTTP(w, r.WithContext(ctx))
			})),
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
		logger: logger,
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.InfoContext(ctx, "http server listening", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.InfoContext(ctx, "http server shutting down")
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.ErrorContext(ctx, "http server shutdown timeout exceeded, forcing close", "err", err)
		return errors.Join(fmt.Errorf("shutdown http server: %w", err), s.server.Close())
	}
	return nil
}
