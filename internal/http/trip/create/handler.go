package create

import (
	"encoding/json"
	"fmt"
	"net/http"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/domainerror"
	"github.com/Artem-229/avito-lab/internal/http/response"
	triphttp "github.com/Artem-229/avito-lab/internal/http/trip"
)

type Handler struct{ usecase tripUsecase }

func New(usecase tripUsecase) *Handler { return &Handler{usecase: usecase} }

const maxBodyBytes = 1 << 20

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var req api.CreateTripJSONRequestBody
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		domainerror.WriteError(w, r, fmt.Errorf("%w: decode body: %w", domainerror.ErrInvalidRequest, err))
		return
	}
	if dec.More() {
		domainerror.WriteError(w, r, fmt.Errorf("%w: unexpected data after JSON body", domainerror.ErrInvalidRequest))
		return
	}

	if err := validateTripData(req); err != nil {
		domainerror.WriteError(w, r, err)
		return
	}

	trip, created, err := h.usecase.CreateTrip(r.Context(), &domain.Trip{
		UserID:   req.UserId,
		DriverID: req.DriverId,
		Start:    domain.Point{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		End:      domain.Point{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
		Price:    req.Price,
	}, params.IdempotencyKey)
	if err != nil {
		domainerror.WriteError(w, r, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	response.WriteJSON(r.Context(), w, status, triphttp.ToAPITrip(trip))
}
