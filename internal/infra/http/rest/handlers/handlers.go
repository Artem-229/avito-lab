package handlers

import (
	"context"
	"net/http"

	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/usecases/trips"
	"github.com/google/uuid"
)

type Repository interface {
	Ping(ctx context.Context) error
}

type Handlers struct {
	repositoryStatus Repository
	trips            *trips.Trips
}

func NewHandlers(repository Repository, tripsUsecase *trips.Trips) *Handlers {
	return &Handlers{
		repositoryStatus: repository,
		trips:            tripsUsecase,
	}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.repositoryStatus.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {

}

func (h *Handlers) GetTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {

}

func (h *Handlers) FinishTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {

}
