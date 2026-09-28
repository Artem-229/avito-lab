package repository

import (
	"context"
	"fmt"

	"github.com/Artem-229/avito-lab/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripsHistoryRepository struct {
	pool *pgxpool.Pool
}

func NewTripsHistoryRepository(pool *pgxpool.Pool) *TripsHistoryRepository {
	return &TripsHistoryRepository{
		pool: pool,
	}
}

func (t *TripsHistoryRepository) CreateRecord(
	ctx context.Context,
	tripID uuid.UUID,
	from *entities.TripStatus,
	to entities.TripStatus,
	reason string,
) error {
	query, args, err := psql.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, from, to, reason).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip status history: %w", err)
	}

	if tx, ok := ExtractTx(ctx); ok {
		_, err = tx.Exec(ctx, query, args...)
	} else {
		_, err = t.pool.Exec(ctx, query, args...)
	}
	if err != nil {
		return fmt.Errorf("create trip status history: %w", err)
	}

	return nil
}
