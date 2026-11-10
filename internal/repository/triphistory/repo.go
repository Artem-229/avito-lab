package triphistory

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	"github.com/Artem-229/avito-lab/internal/infra/postgres/transaction"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

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
	from *domain.Status,
	to domain.Status,
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

	_, err = transaction.ExtractTx(ctx, t.pool).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("create trip status history: %w", err)
	}

	return nil
}
