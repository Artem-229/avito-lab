package trips

import (
	"context"

	"github.com/Artem-229/avito-lab/internal/entities"
	"github.com/google/uuid"
)

type (
	tripsRepository interface {
		CreateTrip(ctx context.Context, trip *entities.Trip) error
		GetTrip(ctx context.Context, tripID uuid.UUID) (*entities.Trip, error)
		FinishTrip(ctx context.Context, tripID uuid.UUID) (*entities.Trip, error)
	}

	tripsHistoryRepository interface {
		CreateRecord(ctx context.Context, tripID uuid.UUID, from *entities.TripStatus, to entities.TripStatus, reason string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)

type Deps struct {
	TxManager    txManager
	Trips        tripsRepository
	TripsHistory tripsHistoryRepository
}

type Trips struct {
	txManager    txManager
	trips        tripsRepository
	tripsHistory tripsHistoryRepository
}

func New(deps *Deps) *Trips {
	return &Trips{
		txManager:    deps.TxManager,
		trips:        deps.Trips,
		tripsHistory: deps.TripsHistory,
	}
}
