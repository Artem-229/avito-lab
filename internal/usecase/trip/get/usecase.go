package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
)

type (
	Deps    struct{ Trips tripsRepository }
	Usecase struct{ trips tripsRepository }
)

func New(deps *Deps) *Usecase { return &Usecase{trips: deps.Trips} }

func (t *Usecase) GetTrip(ctx context.Context, tripID uuid.UUID) (*domain.Trip, error) {
	trip, err := t.trips.GetTrip(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("could not get trip: %w", err)
	}

	return trip, nil
}
