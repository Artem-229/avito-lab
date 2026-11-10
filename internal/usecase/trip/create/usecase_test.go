package create_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	"github.com/Artem-229/avito-lab/internal/usecase/trip/create"
)

func TestCreateTrip(t *testing.T) {
	const (
		idempotencyConflict = "idempotency conflict"
		idempotentCreate    = "idempotent create"
	)
	dbErr := errors.New("database unavailable")
	for _, name := range []string{"success", "driver busy", "repository failure", "history failure", idempotentCreate, "idempotent replay", idempotencyConflict} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMocktripsRepository(ctrl)
			history := NewMocktripsHistoryRepository(ctrl)
			idempotency := NewMockidempotencyRepository(ctrl)
			tx := NewMocktxManager(ctrl)
			ctx := context.Background()
			tx.EXPECT().Do(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
			trip := &domain.Trip{UserID: uuid.New(), DriverID: uuid.New(), Price: 100}
			var key *uuid.UUID
			var wantErr error
			wantCreated := true
			if name == idempotentCreate || name == "idempotent replay" || name == idempotencyConflict {
				id := uuid.New()
				key = &id
				var hash string
				reserve := idempotency.EXPECT().Reserve(ctx, id, gomock.Any(), time.Hour).DoAndReturn(func(_ context.Context, _ uuid.UUID, value string, _ time.Duration) (bool, error) {
					hash = value
					return name == idempotentCreate, nil
				})
				if name != idempotentCreate {
					wantCreated = false
					trip.ID = uuid.New()
					idempotency.EXPECT().Get(ctx, id).After(reserve).DoAndReturn(func(context.Context, uuid.UUID) (string, uuid.UUID, error) {
						if name == idempotencyConflict {
							return "different request", trip.ID, nil
						}
						return hash, trip.ID, nil
					})
					if name == idempotencyConflict {
						wantErr = domain.ErrIdempotencyConflict
					} else {
						repo.EXPECT().GetTrip(ctx, trip.ID).Return(trip, nil)
					}
				}
			}
			if wantCreated {
				var createErr error
				if name == "driver busy" {
					createErr = domain.ErrDriverBusy
				}
				if name == "repository failure" {
					createErr = dbErr
				}
				wantErr = createErr
				insert := repo.EXPECT().CreateTrip(ctx, trip).DoAndReturn(func(_ context.Context, input *domain.Trip) (*domain.Trip, error) {
					if input.ID == uuid.Nil {
						t.Fatal("trip ID was not generated")
					}
					if createErr != nil {
						return nil, createErr
					}
					return input, nil
				})
				if createErr == nil {
					if name == "history failure" {
						wantErr = dbErr
					}
					record := history.EXPECT().CreateRecord(ctx, gomock.Any(), nil, domain.StatusActive, "new trip").After(insert).DoAndReturn(func(_ context.Context, id uuid.UUID, _ *domain.Status, _ domain.Status, _ string) error {
						if id != trip.ID {
							t.Fatalf("history trip ID = %v, want %v", id, trip.ID)
						}
						return wantErr
					})
					if key != nil {
						idempotency.EXPECT().SetTrip(ctx, *key, gomock.Any()).After(record).DoAndReturn(func(_ context.Context, _, id uuid.UUID) error {
							if id != trip.ID {
								t.Fatal("wrong idempotency trip ID")
							}
							return nil
						})
					}
				}
			}
			uc := create.New(&create.Deps{Trips: repo, TripsHistory: history, Idempotency: idempotency, TxManager: tx, IdempotencyTTL: time.Hour})
			got, created, err := uc.CreateTrip(ctx, trip, key)
			if !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want %v", err, wantErr)
			}
			if wantErr != nil {
				if got != nil || created {
					t.Fatalf("failed create returned (%v, %v)", got, created)
				}
				return
			}
			if got != trip || created != wantCreated {
				t.Fatalf("got (%v, %v), want (%v, %v)", got, created, trip, wantCreated)
			}
		})
	}
}
