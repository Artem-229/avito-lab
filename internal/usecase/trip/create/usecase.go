package create

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
)

type Deps struct {
	Logger         *slog.Logger
	Created        metric.Int64Counter
	TxManager      txManager
	Trips          tripsRepository
	TripsHistory   tripsHistoryRepository
	Idempotency    idempotencyRepository
	IdempotencyTTL time.Duration
}

type Usecase struct {
	logger         *slog.Logger
	counter        metric.Int64Counter
	txManager      txManager
	trips          tripsRepository
	tripsHistory   tripsHistoryRepository
	idempotency    idempotencyRepository
	idempotencyTTL time.Duration
}

func New(deps *Deps) *Usecase {
	return &Usecase{
		logger: deps.Logger, counter: deps.Created,
		txManager:      deps.TxManager,
		trips:          deps.Trips,
		tripsHistory:   deps.TripsHistory,
		idempotency:    deps.Idempotency,
		idempotencyTTL: deps.IdempotencyTTL,
	}
}

func (t *Usecase) CreateTrip(ctx context.Context, trip *domain.Trip, idempotencyKey *uuid.UUID) (*domain.Trip, bool, error) {
	if err := trip.Validate(); err != nil {
		return nil, false, err
	}
	var (
		result  *domain.Trip
		created bool
	)

	err := t.txManager.Do(ctx, func(ctx context.Context) error {
		if idempotencyKey != nil {
			hash := requestHash(trip)

			reserved, err := t.idempotency.Reserve(ctx, *idempotencyKey, hash, t.idempotencyTTL)
			if err != nil {
				return fmt.Errorf("reserve idempotency key: %w", err)
			}

			if !reserved {
				storedHash, tripID, err := t.idempotency.Get(ctx, *idempotencyKey)
				if err != nil {
					return fmt.Errorf("get idempotency key: %w", err)
				}
				if storedHash != hash {
					return domain.ErrIdempotencyConflict
				}

				result, err = t.trips.GetTrip(ctx, tripID)
				if err != nil {
					return fmt.Errorf("get trip by idempotency key: %w", err)
				}
				return nil
			}
		}

		trip.ID = uuid.New()

		var err error
		result, err = t.trips.CreateTrip(ctx, trip)
		if err != nil {
			return fmt.Errorf("create trip: %w", err)
		}

		if err := t.tripsHistory.CreateRecord(ctx, result.ID, nil, domain.StatusActive, "new trip"); err != nil {
			return fmt.Errorf("create trip history: %w", err)
		}

		if idempotencyKey != nil {
			if err := t.idempotency.SetTrip(ctx, *idempotencyKey, result.ID); err != nil {
				return fmt.Errorf("set idempotency key trip: %w", err)
			}
		}

		created = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	if created {
		t.recordCreated(ctx, result)
	}
	return result, created, nil
}

func requestHash(trip *domain.Trip) string {
	data := fmt.Sprintf("%s|%s|%v|%v|%v|%v|%d",
		trip.UserID, trip.DriverID,
		trip.Start.Latitude, trip.Start.Longitude,
		trip.End.Latitude, trip.End.Longitude,
		trip.Price,
	)
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func (t *Usecase) recordCreated(ctx context.Context, trip *domain.Trip) {
	if t.counter != nil {
		t.counter.Add(ctx, 1)
	}
	if t.logger != nil {
		t.logger.InfoContext(ctx, "trip created", "trip_id", trip.ID)
	}
}
