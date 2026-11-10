package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Artem-229/avito-lab/internal/domain/trip"
)

func TestFinish(t *testing.T) {
	at := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	for _, status := range []trip.Status{trip.StatusActive, trip.StatusCompleted, "unknown"} {
		t.Run(string(status), func(t *testing.T) {
			model := trip.Trip{Status: status}
			if status == trip.StatusCompleted {
				previous := at.Add(-time.Hour)
				model.FinishedAt = &previous
			}
			before := model
			err := model.Finish(at)
			if status != trip.StatusActive {
				if !errors.Is(err, trip.ErrTripCompleted) || model != before {
					t.Fatalf("rejected transition changed trip: %+v, error: %v", model, err)
				}
				return
			}
			if err != nil || model.Status != trip.StatusCompleted || model.FinishedAt == nil || !model.FinishedAt.Equal(at) {
				t.Fatalf("unexpected finish result: %+v, error: %v", model, err)
			}
		})
	}
}
