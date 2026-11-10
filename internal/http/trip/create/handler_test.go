package create_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	api "github.com/Artem-229/avito-lab/internal/generated"
	"github.com/Artem-229/avito-lab/internal/http/trip/create"
)

func TestCreateTripHandler(t *testing.T) {
	userID, driverID, tripID := uuid.New(), uuid.New(), uuid.New()
	validBody := `{"user_id":"` + userID.String() + `","driver_id":"` + driverID.String() + `","start_point":{"latitude":59.9,"longitude":30.3},"end_point":{"latitude":59.8,"longitude":30.2},"price":1450}`
	dbErr := errors.New("secret database details")

	tests := []struct {
		name       string
		body       string
		usecaseErr error
		wantStatus int
		wantCode   string
	}{
		{name: "created", body: validBody, wantStatus: http.StatusCreated},
		{name: "driver busy", body: validBody, usecaseErr: domain.ErrDriverBusy, wantStatus: http.StatusConflict, wantCode: "driver_busy"},
		{name: "invalid JSON", body: `{"user_id":`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "internal error", body: validBody, usecaseErr: dbErr, wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewMocktripUsecase(gomock.NewController(t))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			if tt.name != "invalid JSON" {
				uc.EXPECT().CreateTrip(req.Context(), gomock.Any(), (*uuid.UUID)(nil)).DoAndReturn(
					func(_ context.Context, got *domain.Trip, _ *uuid.UUID) (*domain.Trip, bool, error) {
						if got.UserID != userID || got.DriverID != driverID || got.Price != 1450 ||
							got.Start.Latitude != 59.9 || got.Start.Longitude != 30.3 ||
							got.End.Latitude != 59.8 || got.End.Longitude != 30.2 {
							t.Fatalf("unexpected trip passed to usecase: %+v", got)
						}
						if tt.usecaseErr != nil {
							return nil, false, tt.usecaseErr
						}
						return &domain.Trip{ID: tripID, UserID: userID, DriverID: driverID}, true, nil
					},
				)
			}

			create.New(uc).CreateTrip(rec, req, api.CreateTripParams{})
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.name == "created" {
				if got, want := rec.Header().Get("Location"), "/api/v1/trips/"+tripID.String(); got != want {
					t.Fatalf("Location = %q, want %q", got, want)
				}
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
			if tt.name == "internal error" && strings.Contains(rec.Body.String(), "secret") {
				t.Fatal("internal error leaked into response")
			}
		})
	}
}
