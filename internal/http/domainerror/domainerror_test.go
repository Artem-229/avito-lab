package domainerror_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/domainerror"
)

func TestWriteError(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{&domain.ValidationError{Field: "price", Message: "must be >= 0"}, 400, "invalid_request"},
		{domain.ErrTripNotFound, 404, "trip_not_found"},
		{domain.ErrTripCompleted, 409, "trip_completed"},
		{domain.ErrDriverBusy, 409, "driver_busy"},
		{domain.ErrIdempotencyConflict, 409, "idempotency_conflict"},
		{domainerror.ErrInvalidRequest, 400, "invalid_request"},
		{errors.New("secret database password"), 500, "internal_error"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/api/v1/trips/test", nil)
			domainerror.WriteError(w, r, fmt.Errorf("operation failed: %w", tc.err))
			var body api.Problem
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != tc.status || int(body.Status) != tc.status || body.Code != tc.code || w.Header().Get("Content-Type") != "application/problem+json" || body.Instance == nil || *body.Instance != r.URL.Path {
				t.Fatalf("unexpected response: %d %+v", w.Code, body)
			}
			if tc.status == 500 && strings.Contains(w.Body.String(), "secret") {
				t.Fatal("internal error leaked into response")
			}
		})
	}
}
