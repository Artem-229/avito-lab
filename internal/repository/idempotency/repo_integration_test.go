//go:build integration

package idempotency_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	"github.com/Artem-229/avito-lab/internal/repository/idempotency"
	triprepo "github.com/Artem-229/avito-lab/internal/repository/trip"
)

func TestReserveAndGet(t *testing.T) {
	ctx := context.Background()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	key, id := uuid.New(), uuid.New()
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM idempotency_keys WHERE key=$1`, key); err != nil {
			t.Error(err)
		}
		if _, err := pool.Exec(context.Background(), `DELETE FROM trips WHERE id=$1`, id); err != nil {
			t.Error(err)
		}
	})
	if _, err := triprepo.NewTripsRepository(pool).CreateTrip(ctx, &domain.Trip{ID: id, UserID: uuid.New(), DriverID: uuid.New(), Price: 100}); err != nil {
		t.Fatal(err)
	}
	repo := idempotency.NewIdempotencyRepository(pool)
	first, err := repo.Reserve(ctx, key, "hash", time.Hour)
	if err != nil || !first {
		t.Fatalf("first reserve = %v, %v", first, err)
	}
	second, err := repo.Reserve(ctx, key, "hash", time.Hour)
	if err != nil || second {
		t.Fatalf("second reserve = %v, %v", second, err)
	}
	if err := repo.SetTrip(ctx, key, id); err != nil {
		t.Fatal(err)
	}
	hash, gotID, err := repo.Get(ctx, key)
	if err != nil || hash != "hash" || gotID != id {
		t.Fatalf("Get = %q, %s, %v", hash, gotID, err)
	}
}
