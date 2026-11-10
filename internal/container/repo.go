package container

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Artem-229/avito-lab/internal/config"
	"github.com/Artem-229/avito-lab/internal/infra/postgres/transaction"
	"github.com/Artem-229/avito-lab/internal/infra/telemetry"
	"github.com/Artem-229/avito-lab/internal/repository/idempotency"
	"github.com/Artem-229/avito-lab/internal/repository/trip"
	"github.com/Artem-229/avito-lab/internal/repository/triphistory"
)

type Repositories struct {
	pool *pgxpool.Pool

	TxManager    *transaction.TxManager
	Trips        *trip.TripsRepository
	TripsHistory *triphistory.TripsHistoryRepository
	Idempotency  *idempotency.IdempotencyRepository
}

func NewRepo(ctx context.Context, cfg *config.Configuration, providers *telemetry.Providers) (*Repositories, error) {
	connCfg, err := pgxpool.ParseConfig(cfg.Postgres.URL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	tracer, err := telemetry.NewDBTracer(providers.Tracer, providers.Meter)
	if err != nil {
		return nil, err
	}
	connCfg.ConnConfig.Tracer = tracer
	connCfg.MaxConns = cfg.Postgres.MaxConns
	connCfg.MinConns = cfg.Postgres.MinConns
	connCfg.MaxConnLifetime = cfg.Postgres.MaxConnLifetime
	connCfg.ConnConfig.ConnectTimeout = cfg.Postgres.ConnectTimeout
	connCfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(cfg.Postgres.QueryTimeout.Milliseconds(), 10)

	pool, err := pgxpool.NewWithConfig(ctx, connCfg)
	if err != nil {
		return nil, fmt.Errorf("create pgxpool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.Postgres.QueryTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Repositories{
		pool:         pool,
		TxManager:    transaction.NewTxManager(pool, cfg.Postgres.QueryTimeout),
		Trips:        trip.NewTripsRepository(pool),
		TripsHistory: triphistory.NewTripsHistoryRepository(pool),
		Idempotency:  idempotency.NewIdempotencyRepository(pool),
	}, nil
}

func (r *Repositories) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Repositories) Close() {
	r.pool.Close()
}
