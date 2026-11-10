package ready

import (
	"context"
	"net/http"
	"time"

	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/response"
)

type Handler struct {
	repositoryStatus repository
	readyTimeout     time.Duration
}

func New(repositoryStatus repository, readyTimeout time.Duration) *Handler {
	return &Handler{repositoryStatus: repositoryStatus, readyTimeout: readyTimeout}
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.readyTimeout)
	defer cancel()

	if err := h.repositoryStatus.Ping(ctx); err != nil {
		response.WriteJSON(r.Context(), w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}
