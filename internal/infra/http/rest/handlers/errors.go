package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Artem-229/avito-lab/internal/entities"
	api "github.com/Artem-229/avito-lab/internal/generated"
)

const problemTypeBase = "https://tripgo.example/problems/"

var errInvalidRequest = errors.New("invalid request")

type problem struct {
	status int
	code   string
	slug   string
	title  string
	detail string
}

func WriteParamError(w http.ResponseWriter, r *http.Request, err error) {
	writeError(w, r, fmt.Errorf("%w: %w", errInvalidRequest, err))
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	p := mapError(err)
	if p.status == http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	}

	instance := r.URL.Path
	body := api.Problem{
		Type:     problemTypeBase + p.slug,
		Title:    p.title,
		Status:   int32(p.status),
		Detail:   &p.detail,
		Instance: &instance,
		Code:     p.code,
	}

	writeBody(w, "application/problem+json", p.status, body)
}

func mapError(err error) problem {
	switch {
	case errors.Is(err, errInvalidRequest):
		return problem{http.StatusBadRequest, "invalid_request", "invalid-request", "Invalid request", err.Error()}
	case errors.Is(err, entities.ErrTripNotFound):
		return problem{http.StatusNotFound, "trip_not_found", "trip-not-found", "Trip not found", "Trip with this id does not exist"}
	case errors.Is(err, entities.ErrTripCompleted):
		return problem{http.StatusConflict, "trip_completed", "trip-completed", "Trip completed", "Trip is already completed"}
	case errors.Is(err, entities.ErrDriverBusy):
		return problem{http.StatusConflict, "driver_busy", "driver-busy", "Driver busy", "Driver already has an active trip"}
	case errors.Is(err, entities.ErrIdempotencyConflict):
		return problem{http.StatusConflict, "idempotency_conflict", "idempotency-conflict", "Idempotency conflict", "Idempotency key was already used with a different request"}
	default:
		return problem{http.StatusInternalServerError, "internal_error", "internal-error", "Internal error", "Internal server error"}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	writeBody(w, "application/json", status, v)
}

func writeBody(w http.ResponseWriter, contentType string, status int, v any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write response body", "err", err)
	}
}
