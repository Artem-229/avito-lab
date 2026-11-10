package trip

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	"github.com/Artem-229/avito-lab/internal/infra/postgres/transaction"
)

func (t *TripsRepository) FinishTrip(ctx context.Context, tripID uuid.UUID, finishedAt time.Time) (*domain.Trip, error) {
	query, args, err := psql.
		Update("trips").
		Set("status", domain.StatusCompleted).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{"id": tripID, "status": domain.StatusActive}).
		Suffix("RETURNING " + strings.Join(tripColumns, ", ")).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update finish trip: %w", err)
	}

	row := transaction.ExtractTx(ctx, t.pool).QueryRow(ctx, query, args...)

	trip, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := t.GetTrip(ctx, tripID); err != nil {
			return nil, err
		}
		return nil, domain.ErrTripCompleted
	}
	if err != nil {
		return nil, fmt.Errorf("finish trip: %w", err)
	}

	return trip, nil
}
