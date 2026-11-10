package container

import (
	"log/slog"

	"go.opentelemetry.io/otel/metric"

	"github.com/Artem-229/avito-lab/internal/config"
	"github.com/Artem-229/avito-lab/internal/usecase/trip/create"
	"github.com/Artem-229/avito-lab/internal/usecase/trip/finish"
	"github.com/Artem-229/avito-lab/internal/usecase/trip/get"
)

type Usecases struct {
	CreateTrip *create.Usecase
	GetTrip    *get.Usecase
	FinishTrip *finish.Usecase
}

func NewUsecases(repos *Repositories, cfg *config.Configuration, logger *slog.Logger, created, completed metric.Int64Counter) *Usecases {
	return &Usecases{
		CreateTrip: create.New(&create.Deps{Logger: logger, Created: created, TxManager: repos.TxManager, Trips: repos.Trips, TripsHistory: repos.TripsHistory, Idempotency: repos.Idempotency, IdempotencyTTL: cfg.Idempotency.KeyTTL}),
		GetTrip:    get.New(&get.Deps{Trips: repos.Trips}),
		FinishTrip: finish.New(&finish.Deps{Logger: logger, Completed: completed, TxManager: repos.TxManager, Trips: repos.Trips, TripsHistory: repos.TripsHistory}),
	}
}
