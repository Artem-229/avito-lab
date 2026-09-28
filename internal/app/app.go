package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Artem-229/avito-lab/internal/config"
	"github.com/Artem-229/avito-lab/internal/infra/http/rest"
	"github.com/Artem-229/avito-lab/internal/infra/http/rest/handlers"
)

type App struct {
	logger *slog.Logger
	repos  *Repositories
	server *rest.Server
}

func New(ctx context.Context, cfg *config.Configuration) (*App, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", cfg.Log.Level, err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(logger)

	repos, err := NewRepo(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create repositories: %w", err)
	}

	usecases := NewUsecases(repos, cfg)

	server, err := rest.NewServer(
		rest.Config{
			Addr:              cfg.HTTP.Addr,
			ReadTimeout:       cfg.HTTP.ReadTimeout,
			ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
			WriteTimeout:      cfg.HTTP.WriteTimeout,
			IdleTimeout:       cfg.HTTP.IdleTimeout,
			ShutdownTimeout:   cfg.HTTP.ShutdownTimeout,
		},
		handlers.NewHandlers(repos, cfg.Postgres.QueryTimeout, usecases.Trips),
		logger,
	)
	if err != nil {
		repos.Close()
		return nil, fmt.Errorf("create http server: %w", err)
	}

	return &App{
		logger: logger,
		repos:  repos,
		server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.repos.Close()

	a.logger.Info("app started")

	if err := a.server.Run(ctx); err != nil {
		return err
	}

	a.logger.Info("app stopped")

	return nil
}
