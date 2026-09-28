package app

import (
	"github.com/Artem-229/avito-lab/internal/usecases/trips"
)

type Usecases struct {
	Trips *trips.Trips
}

func NewUsecases(repos *Repositories) *Usecases {
	return &Usecases{
		Trips: trips.New(&trips.Deps{
			TxManager:    repos.Trips,
			Trips:        repos.Trips,
			TripsHistory: repos.TripsHistory,
		}),
	}
}
