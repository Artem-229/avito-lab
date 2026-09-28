package app

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Artem-229/avito-lab/internal/config"
	"github.com/Artem-229/avito-lab/internal/infra/postgres/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repositories struct {
	pool *pgxpool.Pool

	Trips        *repository.TripsRepository
	TripsHistory *repository.TripsHistoryRepository
}

func NewRepo(ctx context.Context, cfg *config.Configuration) (*Repositories, error) {
	connCfg, err := pgxpool.ParseConfig(cfg.Postgres.URL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	connCfg.MaxConns = cfg.Postgres.MaxConns
	connCfg.MinConns = cfg.Postgres.MinConns
	connCfg.MaxConnLifetime = cfg.Postgres.MaxConnLifetime
	connCfg.ConnConfig.ConnectTimeout = cfg.Postgres.ConnectTimeout
	connCfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(cfg.Postgres.QueryTimeout.Milliseconds(), 10)

	pool, err := pgxpool.NewWithConfig(ctx, connCfg)
	if err != nil {
		return nil, fmt.Errorf("create pgxpool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Repositories{
		pool:         pool,
		Trips:        repository.NewTripsRepository(pool),
		TripsHistory: repository.NewTripsHistoryRepository(pool),
	}, nil
}

func (r *Repositories) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Repositories) Close() {
	r.pool.Close()
}
