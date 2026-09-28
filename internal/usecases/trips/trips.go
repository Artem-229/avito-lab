package trips

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Artem-229/avito-lab/internal/entities"
	"github.com/google/uuid"
)

type (
	tripsRepository interface {
		CreateTrip(ctx context.Context, trip *entities.Trip) (*entities.Trip, error)
		GetTrip(ctx context.Context, tripID uuid.UUID) (*entities.Trip, error)
		FinishTrip(ctx context.Context, tripID uuid.UUID) (*entities.Trip, error)
	}

	tripsHistoryRepository interface {
		CreateRecord(ctx context.Context, tripID uuid.UUID, from *entities.TripStatus, to entities.TripStatus, reason string) error
	}

	idempotencyRepository interface {
		Reserve(ctx context.Context, key uuid.UUID, requestHash string, ttl time.Duration) (bool, error)
		Get(ctx context.Context, key uuid.UUID) (string, uuid.UUID, error)
		SetTrip(ctx context.Context, key uuid.UUID, tripID uuid.UUID) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)

type Deps struct {
	TxManager      txManager
	Trips          tripsRepository
	TripsHistory   tripsHistoryRepository
	Idempotency    idempotencyRepository
	IdempotencyTTL time.Duration
}

type Trips struct {
	txManager      txManager
	trips          tripsRepository
	tripsHistory   tripsHistoryRepository
	idempotency    idempotencyRepository
	idempotencyTTL time.Duration
}

func New(deps *Deps) *Trips {
	return &Trips{
		txManager:      deps.TxManager,
		trips:          deps.Trips,
		tripsHistory:   deps.TripsHistory,
		idempotency:    deps.Idempotency,
		idempotencyTTL: deps.IdempotencyTTL,
	}
}

func (t *Trips) CreateTrip(ctx context.Context, trip *entities.Trip, idempotencyKey *uuid.UUID) (*entities.Trip, bool, error) {
	var (
		result  *entities.Trip
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
					return entities.ErrIdempotencyConflict
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

		if err := t.tripsHistory.CreateRecord(ctx, result.ID, nil, entities.TripStatusActive, "new trip"); err != nil {
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

	return result, created, nil
}

func requestHash(trip *entities.Trip) string {
	data := fmt.Sprintf("%s|%s|%v|%v|%v|%v|%d",
		trip.UserID, trip.DriverID,
		trip.Start.Latitude, trip.Start.Longitude,
		trip.End.Latitude, trip.End.Longitude,
		trip.Price,
	)
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func (t *Trips) GetTrip(ctx context.Context, tripID uuid.UUID) (*entities.Trip, error) {
	trip, err := t.trips.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("could not get trip: %w", err)
	}

	return trip, nil
}

func (t *Trips) FinishTrip(ctx context.Context, tripID uuid.UUID) (*entities.Trip, error) {
	var (
		trip *entities.Trip
		err  error
	)

	txErr := t.txManager.Do(ctx, func(ctx context.Context) error {
		trip, err = t.trips.FinishTrip(ctx, tripID)
		if err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}

		from := entities.TripStatusActive
		if err := t.tripsHistory.CreateRecord(ctx, trip.ID, &from, entities.TripStatusCompleted, "trip finished"); err != nil {
			return fmt.Errorf("create finish trip history record: %w", err)
		}

		return nil
	})

	return trip, txErr
}
