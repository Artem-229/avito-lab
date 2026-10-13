package app

import (
	"github.com/Artem-229/avito-lab/internal/config"
	"github.com/Artem-229/avito-lab/internal/usecases/trips"
)

type Usecases struct {
	Trips *trips.Trips
}

func NewUsecases(repos *Repositories, cfg *config.Configuration) *Usecases {
	return &Usecases{
		Trips: trips.New(&trips.Deps{
			TxManager:      repos.TxManager,
			Trips:          repos.Trips,
			TripsHistory:   repos.TripsHistory,
			Idempotency:    repos.Idempotency,
			IdempotencyTTL: cfg.Idempotency.KeyTTL,
		}),
	}
}
