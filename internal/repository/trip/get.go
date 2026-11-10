package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	"github.com/Artem-229/avito-lab/internal/infra/postgres/transaction"
)

func (t *TripsRepository) GetTrip(ctx context.Context, tripID uuid.UUID) (*domain.Trip, error) {
	query, args, err := psql.
		Select(tripColumns...).
		From("trips").
		Where(sq.Eq{"id": tripID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select trip: %w", err)
	}

	row := transaction.ExtractTx(ctx, t.pool).QueryRow(ctx, query, args...)

	trip, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTripNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}
