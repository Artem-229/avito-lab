package get

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/Artem-229/avito-lab/internal/http/domainerror"
	"github.com/Artem-229/avito-lab/internal/http/response"
	triphttp "github.com/Artem-229/avito-lab/internal/http/trip"
)

type Handler struct{ usecase tripUsecase }

func New(usecase tripUsecase) *Handler { return &Handler{usecase: usecase} }

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	trip, err := h.usecase.GetTrip(r.Context(), tripId)
	if err != nil {
		domainerror.WriteError(w, r, err)
		return
	}

	response.WriteJSON(r.Context(), w, http.StatusOK, triphttp.ToAPITrip(trip))
}
