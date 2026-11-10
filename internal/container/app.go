package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/Artem-229/avito-lab/internal/config"
	"github.com/Artem-229/avito-lab/internal/http/rest"
	"github.com/Artem-229/avito-lab/internal/infra/telemetry"
)

type App struct {
	logger          *slog.Logger
	repos           *Repositories
	server          *rest.Server
	telemetry       *telemetry.Providers
	shutdownTimeout time.Duration
}

func New(ctx context.Context, cfg *config.Configuration) (*App, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", cfg.Log.Level, err)
	}
	logger := telemetry.NewLogger(lvl)
	providers, err := telemetry.New(ctx)
	if err != nil {
		return nil, err
	}
	ready := false
	defer func() {
		if !ready {
			shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.ShutdownTimeout)
			defer cancel()
			if err := providers.Shutdown(shutdownCtx); err != nil {
				logger.Error("shutdown telemetry", "err", err)
			}
		}
	}()

	repos, err := NewRepo(ctx, cfg, providers)
	if err != nil {
		return nil, fmt.Errorf("create repositories: %w", err)
	}

	meter := providers.Meter.Meter("trip-service/business")
	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		repos.Close()
		return nil, err
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		repos.Close()
		return nil, err
	}
	active, err := meter.Int64ObservableGauge("tripgo.trips.active")
	if err != nil {
		repos.Close()
		return nil, err
	}
	_, err = meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		queryCtx, cancel := context.WithTimeout(ctx, cfg.Postgres.QueryTimeout)
		defer cancel()
		count, err := repos.Trips.CountActive(queryCtx)
		if err != nil {
			return err
		}
		observer.ObserveInt64(active, count)
		return nil
	}, active)
	if err != nil {
		repos.Close()
		return nil, err
	}
	usecases := NewUsecases(repos, cfg, logger, created, completed)

	server, err := rest.NewServer(
		rest.Config{
			Addr:              cfg.HTTP.Addr,
			ReadTimeout:       cfg.HTTP.ReadTimeout,
			ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
			WriteTimeout:      cfg.HTTP.WriteTimeout,
			IdleTimeout:       cfg.HTTP.IdleTimeout,
			QueryTimeout:      cfg.Postgres.QueryTimeout,
		},
		newHandlers(repos, cfg.Postgres.QueryTimeout, usecases),
		logger, providers.Tracer, providers.Meter,
	)
	if err != nil {
		repos.Close()
		return nil, fmt.Errorf("create http server: %w", err)
	}

	ready = true
	return &App{
		logger:    logger,
		telemetry: providers, shutdownTimeout: cfg.HTTP.ShutdownTimeout,
		repos:  repos,
		server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.InfoContext(ctx, "container started")
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- a.server.Run(ctx) }()
	var runErr error
	select {
	case runErr = <-serverErrors:
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.shutdownTimeout)
	defer cancel()
	serverErr := a.server.Shutdown(shutdownCtx)
	telemetryErr := a.telemetry.Shutdown(shutdownCtx)
	closed := make(chan struct{})
	go func() { a.repos.Close(); close(closed) }()
	select {
	case <-closed:
	case <-shutdownCtx.Done():
		a.logger.ErrorContext(shutdownCtx, "shutdown timeout exceeded", "err", shutdownCtx.Err())
		return errors.Join(runErr, serverErr, telemetryErr, shutdownCtx.Err())
	}
	a.logger.InfoContext(shutdownCtx, "container stopped")
	return errors.Join(runErr, serverErr, telemetryErr)
}
