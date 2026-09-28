package rest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/go-chi/chi/v5"
)

type Config struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type Server struct {
	server          *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

func NewServer(cfg Config, handlers api.ServerInterface, logger *slog.Logger) *Server {
	router := chi.NewRouter()

	return &Server{
		server: &http.Server{
			Addr:              cfg.Addr,
			Handler:           api.HandlerFromMux(handlers, router),
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("http server listening", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	s.logger.Info("http server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("http server shutdown timeout exceeded, forcing close", "err", err)
		_ = s.server.Close()
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}
