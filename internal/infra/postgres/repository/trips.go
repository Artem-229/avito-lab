package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Artem-229/avito-lab/internal/entities"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var tripColumns = []string{
	"id", "user_id", "driver_id",
	"start_latitude", "start_longitude",
	"end_latitude", "end_longitude",
	"price", "status",
	"started_at", "finished_at",
	"created_at", "updated_at",
}

type TripsRepository struct {
	pool *pgxpool.Pool
}

func NewTripsRepository(pool *pgxpool.Pool) *TripsRepository {
	return &TripsRepository{
		pool: pool,
	}
}

func (t *TripsRepository) CreateTrip(ctx context.Context, trip *entities.Trip) (*entities.Trip, error) {
	now := time.Now()
	query, args, err := psql.
		Insert("trips").
		Columns(tripColumns...).
		Values(
			trip.ID, trip.UserID, trip.DriverID,
			trip.Start.Latitude, trip.Start.Longitude,
			trip.End.Latitude, trip.End.Longitude,
			trip.Price, entities.TripStatusActive,
			now, nil,
			now, now,
		).
		Suffix("RETURNING " + strings.Join(tripColumns, ", ")).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert trip: %w", err)
	}

	created, err := scanTrip(extractTx(ctx, t.pool).QueryRow(ctx, query, args...))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_driver_active_uidx" {
			return nil, entities.ErrDriverBusy
		}
		return nil, fmt.Errorf("create trip: %w", err)
	}

	return created, nil
}

func (t *TripsRepository) GetTrip(ctx context.Context, tripID uuid.UUID) (*entities.Trip, error) {
	query, args, err := psql.
		Select(tripColumns...).
		From("trips").
		Where(sq.Eq{"id": tripID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select trip: %w", err)
	}

	row := extractTx(ctx, t.pool).QueryRow(ctx, query, args...)

	trip, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, entities.ErrTripNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}

func (t *TripsRepository) FinishTrip(ctx context.Context, tripID uuid.UUID) (*entities.Trip, error) {
	now := time.Now()

	query, args, err := psql.
		Update("trips").
		Set("status", entities.TripStatusCompleted).
		Set("finished_at", now).
		Set("updated_at", now).
		Where(sq.Eq{"id": tripID, "status": entities.TripStatusActive}).
		Suffix("RETURNING " + strings.Join(tripColumns, ", ")).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update finish trip: %w", err)
	}

	row := extractTx(ctx, t.pool).QueryRow(ctx, query, args...)

	trip, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := t.GetTrip(ctx, tripID); err != nil {
			return nil, err
		}
		return nil, entities.ErrTripCompleted
	}
	if err != nil {
		return nil, fmt.Errorf("finish trip: %w", err)
	}

	return trip, nil
}

func scanTrip(row pgx.Row) (*entities.Trip, error) {
	var trip entities.Trip

	err := row.Scan(
		&trip.ID, &trip.UserID, &trip.DriverID,
		&trip.Start.Latitude, &trip.Start.Longitude,
		&trip.End.Latitude, &trip.End.Longitude,
		&trip.Price, &trip.Status,
		&trip.StartedAt, &trip.FinishedAt,
		&trip.CreatedAt, &trip.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}
