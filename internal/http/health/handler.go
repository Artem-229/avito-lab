package health

import (
	"net/http"

	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/response"
)

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(r.Context(), w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
