//go:build integration

package transaction_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	"github.com/Artem-229/avito-lab/internal/infra/postgres/transaction"
	triprepo "github.com/Artem-229/avito-lab/internal/repository/trip"
	"github.com/Artem-229/avito-lab/internal/repository/triphistory"
)

func TestTransactionCommitAndRollback(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}
	for _, rollback := range []bool{false, true} {
		name := "commit"
		if rollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			pool, err := pgxpool.New(ctx, url)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(pool.Close)
			id := uuid.New()
			t.Cleanup(func() {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if _, err := pool.Exec(cleanupCtx, `DELETE FROM trips WHERE id=$1`, id); err != nil {
					t.Error(err)
				}
			})
			trips := triprepo.NewTripsRepository(pool)
			history := triphistory.NewTripsHistoryRepository(pool)
			forced := errors.New("failure after both inserts")
			err = transaction.NewTxManager(pool, 5*time.Second).Do(ctx, func(txCtx context.Context) error {
				if _, err := trips.CreateTrip(txCtx, &domain.Trip{ID: id, UserID: uuid.New(), DriverID: uuid.New(), Price: 100}); err != nil {
					return err
				}
				if err := history.CreateRecord(txCtx, id, nil, domain.StatusActive, "new trip"); err != nil {
					return err
				}
				if rollback {
					return forced
				}
				return nil
			})
			if rollback && !errors.Is(err, forced) {
				t.Fatalf("error = %v, want forced failure after both inserts", err)
			}
			if !rollback && err != nil {
				t.Fatal(err)
			}
			var tripCount, historyCount int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM trips WHERE id=$1`, id).Scan(&tripCount); err != nil {
				t.Fatal(err)
			}
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM trip_status_history WHERE trip_id=$1`, id).Scan(&historyCount); err != nil {
				t.Fatal(err)
			}
			want := 1
			if rollback {
				want = 0
			}
			if tripCount != want || historyCount != want {
				t.Fatalf("trips=%d, history=%d; want %d in both tables", tripCount, historyCount, want)
			}
		})
	}
}
