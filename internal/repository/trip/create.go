package trip

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	"github.com/Artem-229/avito-lab/internal/infra/postgres/transaction"
)

func (t *TripsRepository) CreateTrip(ctx context.Context, trip *domain.Trip) (*domain.Trip, error) {
	now := time.Now()
	query, args, err := psql.
		Insert("trips").
		Columns(tripColumns...).
		Values(
			trip.ID, trip.UserID, trip.DriverID,
			trip.Start.Latitude, trip.Start.Longitude,
			trip.End.Latitude, trip.End.Longitude,
			trip.Price, domain.StatusActive,
			now, nil,
			now, now,
		).
		Suffix("RETURNING " + strings.Join(tripColumns, ", ")).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert trip: %w", err)
	}

	created, err := scanTrip(transaction.ExtractTx(ctx, t.pool).QueryRow(ctx, query, args...))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_driver_active_uidx" {
			return nil, domain.ErrDriverBusy
		}
		return nil, fmt.Errorf("create trip: %w", err)
	}

	return created, nil
}
