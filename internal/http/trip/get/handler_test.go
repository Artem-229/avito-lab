package get_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/trip/get"
)

func TestGetTripHandler(t *testing.T) {
	tripID := uuid.New()
	tests := []struct {
		name       string
		trip       *domain.Trip
		usecaseErr error
		wantStatus int
		wantCode   string
	}{
		{name: "success", trip: &domain.Trip{ID: tripID}, wantStatus: http.StatusOK},
		{name: "not found", usecaseErr: domain.ErrTripNotFound, wantStatus: http.StatusNotFound, wantCode: "trip_not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewMocktripUsecase(gomock.NewController(t))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
			rec := httptest.NewRecorder()
			uc.EXPECT().GetTrip(req.Context(), tripID).Return(tt.trip, tt.usecaseErr)

			get.New(uc).GetTrip(rec, req, tripID)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCode == "" {
				var got api.Trip
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got.Id != tripID {
					t.Fatalf("trip id = %v, want %v", got.Id, tripID)
				}
				return
			}
			var got api.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got.Code != tt.wantCode {
				t.Fatalf("problem code = %q, want %q", got.Code, tt.wantCode)
			}
		})
	}
}
