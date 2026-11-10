package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	"github.com/Artem-229/avito-lab/internal/usecase/trip/get"
)

func TestGetTrip(t *testing.T) {
	for _, wantErr := range []error{nil, domain.ErrTripNotFound, errors.New("database unavailable")} {
		t.Run(map[bool]string{true: "success", false: "error"}[wantErr == nil], func(t *testing.T) {
			repo := NewMocktripsRepository(gomock.NewController(t))
			ctx := context.Background()
			trip := &domain.Trip{ID: uuid.New()}
			var want *domain.Trip
			if wantErr == nil {
				want = trip
			}
			repo.EXPECT().GetTrip(ctx, trip.ID).Return(want, wantErr)
			got, err := get.New(&get.Deps{Trips: repo}).GetTrip(ctx, trip.ID)
			if !errors.Is(err, wantErr) || got != want {
				t.Fatalf("got (%v, %v), want (%v, %v)", got, err, want, wantErr)
			}
		})
	}
}
