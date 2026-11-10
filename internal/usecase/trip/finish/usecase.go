package finish

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
)

type Deps struct {
	Logger       *slog.Logger
	Completed    metric.Int64Counter
	TxManager    txManager
	Trips        tripsRepository
	TripsHistory tripsHistoryRepository
}

type Usecase struct {
	logger       *slog.Logger
	counter      metric.Int64Counter
	txManager    txManager
	trips        tripsRepository
	tripsHistory tripsHistoryRepository
}

func New(deps *Deps) *Usecase {
	return &Usecase{
		logger: deps.Logger, counter: deps.Completed,
		txManager:    deps.TxManager,
		trips:        deps.Trips,
		tripsHistory: deps.TripsHistory,
	}
}

func (t *Usecase) FinishTrip(ctx context.Context, tripID uuid.UUID) (*domain.Trip, error) {
	var (
		trip *domain.Trip
		err  error
	)

	txErr := t.txManager.Do(ctx, func(ctx context.Context) error {
		trip, err = t.trips.GetTrip(ctx, tripID)
		if err != nil {
			return fmt.Errorf("get trip: %w", err)
		}

		finishedAt := time.Now()
		if err := trip.Finish(finishedAt); err != nil {
			return fmt.Errorf("trip finish: %w", err)
		}

		trip, err = t.trips.FinishTrip(ctx, trip.ID, finishedAt)
		if err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}

		from := domain.StatusActive
		if err := t.tripsHistory.CreateRecord(ctx, trip.ID, &from, domain.StatusCompleted, "trip finished"); err != nil {
			return fmt.Errorf("create finish trip history record: %w", err)
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}
	if t.counter != nil {
		t.counter.Add(ctx, 1)
	}
	if t.logger != nil {
		t.logger.InfoContext(ctx, "trip completed", "trip_id", trip.ID)
	}
	return trip, nil
}
