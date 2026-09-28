package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Artem-229/avito-lab/internal/entities"
	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/usecases/trips"
	"github.com/google/uuid"
)

const maxBodyBytes = 1 << 20

type Repository interface {
	Ping(ctx context.Context) error
}

type Handlers struct {
	repositoryStatus Repository
	readyTimeout     time.Duration
	trips            *trips.Trips
}

func NewHandlers(repository Repository, readyTimeout time.Duration, tripsUsecase *trips.Trips) *Handlers {
	return &Handlers{
		repositoryStatus: repository,
		readyTimeout:     readyTimeout,
		trips:            tripsUsecase,
	}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.readyTimeout)
	defer cancel()

	if err := h.repositoryStatus.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var req api.CreateTripJSONRequestBody
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, r, fmt.Errorf("%w: decode body: %w", errInvalidRequest, err))
		return
	}
	if dec.More() {
		writeError(w, r, fmt.Errorf("%w: unexpected data after JSON body", errInvalidRequest))
		return
	}

	if err := validateTripData(req); err != nil {
		writeError(w, r, err)
		return
	}

	trip, created, err := h.trips.CreateTrip(r.Context(), &entities.Trip{
		UserID:   req.UserId,
		DriverID: req.DriverId,
		Start:    entities.Point{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		End:      entities.Point{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
		Price:    req.Price,
	}, params.IdempotencyKey)
	if err != nil {
		writeError(w, r, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	writeJSON(w, status, toAPITrip(trip))
}

func (h *Handlers) GetTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	trip, err := h.trips.GetTrip(r.Context(), tripId)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(trip))
}

func (h *Handlers) FinishTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	trip, err := h.trips.FinishTrip(r.Context(), tripId)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(trip))
}

func validateTripData(req api.TripData) error {
	if req.UserId == uuid.Nil {
		return fmt.Errorf("%w: user_id is required", errInvalidRequest)
	}
	if req.DriverId == uuid.Nil {
		return fmt.Errorf("%w: driver_id is required", errInvalidRequest)
	}
	if req.Price < 0 {
		return fmt.Errorf("%w: price must be >= 0", errInvalidRequest)
	}
	if err := validateCoordinates("start_point", req.StartPoint); err != nil {
		return err
	}
	if err := validateCoordinates("end_point", req.EndPoint); err != nil {
		return err
	}
	return nil
}

func validateCoordinates(field string, c api.Coordinates) error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return fmt.Errorf("%w: %s.latitude must be in [-90, 90]", errInvalidRequest, field)
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return fmt.Errorf("%w: %s.longitude must be in [-180, 180]", errInvalidRequest, field)
	}
	return nil
}

func toAPITrip(t *entities.Trip) api.Trip {
	return api.Trip{
		Id:         t.ID,
		UserId:     t.UserID,
		DriverId:   t.DriverID,
		StartPoint: api.Coordinates{Latitude: t.Start.Latitude, Longitude: t.Start.Longitude},
		EndPoint:   api.Coordinates{Latitude: t.End.Latitude, Longitude: t.End.Longitude},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}
