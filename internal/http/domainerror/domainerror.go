package domainerror

import (
	"errors"
	"fmt"
	"net/http"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/logcontext"
	"github.com/Artem-229/avito-lab/internal/http/response"
)

const problemTypeBase = "https://tripgo.example/problems/"

var ErrInvalidRequest = errors.New("invalid request")

type problem struct {
	status int32
	code   string
	slug   string
	title  string
	detail string
}

func WriteParamError(w http.ResponseWriter, r *http.Request, err error) {
	WriteError(w, r, fmt.Errorf("%w: %w", ErrInvalidRequest, err))
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	p := mapError(err)
	if p.status == http.StatusInternalServerError {
		logcontext.Error(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	}

	instance := r.URL.Path
	body := api.Problem{
		Type:     problemTypeBase + p.slug,
		Title:    p.title,
		Status:   p.status,
		Detail:   &p.detail,
		Instance: &instance,
		Code:     p.code,
	}

	response.WriteBody(r.Context(), w, "application/problem+json", int(p.status), body)
}

func mapError(err error) problem {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		return problem{http.StatusBadRequest, "invalid_request", "invalid-request", "Invalid request", validation.Error()}
	case errors.Is(err, ErrInvalidRequest):
		return problem{http.StatusBadRequest, "invalid_request", "invalid-request", "Invalid request", err.Error()}
	case errors.Is(err, domain.ErrTripNotFound):
		return problem{http.StatusNotFound, "trip_not_found", "trip-not-found", "Trip not found", "Trip with this id does not exist"}
	case errors.Is(err, domain.ErrTripCompleted):
		return problem{http.StatusConflict, "trip_completed", "trip-completed", "Trip completed", "Trip is already completed"}
	case errors.Is(err, domain.ErrDriverBusy):
		return problem{http.StatusConflict, "driver_busy", "driver-busy", "Driver busy", "Driver already has an active trip"}
	case errors.Is(err, domain.ErrIdempotencyConflict):
		return problem{http.StatusConflict, "idempotency_conflict", "idempotency-conflict", "Idempotency conflict", "Idempotency key was already used with a different request"}
	default:
		return problem{http.StatusInternalServerError, "internal_error", "internal-error", "Internal error", "Internal server error"}
	}
}
