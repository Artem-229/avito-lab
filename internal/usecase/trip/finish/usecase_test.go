package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	"github.com/Artem-229/avito-lab/internal/usecase/trip/finish"
)

func TestFinishTrip(t *testing.T) {
	dbErr := errors.New("database unavailable")
	for _, name := range []string{"success", "not found", "completed", "concurrent finish", "history failure", "transaction failure"} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripsRepository(ctrl)
			history := NewMocktripsHistoryRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			uc := finish.New(&finish.Deps{Trips: repo, TripsHistory: history, TxManager: tx})
			ctx := context.Background()
			trip := &domain.Trip{ID: uuid.New(), Status: domain.StatusActive}
			var wantErr error
			if name == "transaction failure" {
				wantErr = dbErr
				tx.EXPECT().Do(ctx, gomock.Any()).Return(dbErr)
			} else {
				tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
				switch name {
				case "not found":
					wantErr = domain.ErrTripNotFound
					repo.EXPECT().GetTrip(ctx, trip.ID).Return(nil, wantErr)
				case "completed":
					trip.Status = domain.StatusCompleted
					wantErr = domain.ErrTripCompleted
					repo.EXPECT().GetTrip(ctx, trip.ID).Return(trip, nil)
				default:
					get := repo.EXPECT().GetTrip(ctx, trip.ID).Return(trip, nil)
					var updateErr error
					if name == "concurrent finish" {
						updateErr = domain.ErrTripCompleted
						wantErr = updateErr
					}
					update := repo.EXPECT().FinishTrip(ctx, trip.ID, gomock.Any()).After(get).DoAndReturn(func(_ context.Context, _ uuid.UUID, at time.Time) (*domain.Trip, error) {
						if trip.Status != domain.StatusCompleted || trip.FinishedAt == nil || !trip.FinishedAt.Equal(at) || at.IsZero() {
							t.Fatalf("repository called before domain transition: %+v, at %v", trip, at)
						}
						return trip, updateErr
					})
					if updateErr == nil {
						if name == "history failure" {
							wantErr = dbErr
						}
						from := domain.StatusActive
						history.EXPECT().CreateRecord(ctx, trip.ID, &from, domain.StatusCompleted, "trip finished").After(update).Return(wantErr)
					}
				}
			}
			got, err := uc.FinishTrip(ctx, trip.ID)
			if !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want %v", err, wantErr)
			}
			if wantErr == nil && (got != trip || got.Status != domain.StatusCompleted) {
				t.Fatalf("unexpected result: %+v", got)
			}
		})
	}
}
