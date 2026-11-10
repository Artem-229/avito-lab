//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	triprepo "github.com/Artem-229/avito-lab/internal/repository/trip"
)

func TestTripRepositories(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	trips := triprepo.NewTripsRepository(pool)
	tripID, driverID := uuid.New(), uuid.New()
	defer cleanupTrip(t, pool, tripID)

	created, err := trips.CreateTrip(ctx, validTrip(tripID, driverID))
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != tripID || created.Status != domain.StatusActive {
		t.Fatalf("created trip = %+v", created)
	}

	got, err := trips.GetTrip(ctx, tripID)
	if err != nil || got.ID != tripID || got.DriverID != driverID {
		t.Fatalf("GetTrip() = (%+v, %v)", got, err)
	}
	if _, err := trips.GetTrip(ctx, uuid.New()); !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("GetTrip(missing) error = %v, want ErrTripNotFound", err)
	}
	if _, err := trips.CreateTrip(ctx, validTrip(uuid.New(), driverID)); !errors.Is(err, domain.ErrDriverBusy) {
		t.Fatalf("CreateTrip(occupied driver) error = %v, want ErrDriverBusy", err)
	}

	finishedAt := time.Now().UTC().Truncate(time.Microsecond)
	finished, err := trips.FinishTrip(ctx, tripID, finishedAt)
	if err != nil || finished.Status != domain.StatusCompleted || finished.FinishedAt == nil || !finished.FinishedAt.Equal(finishedAt) {
		t.Fatalf("FinishTrip() = (%+v, %v)", finished, err)
	}
	var storedStatus string
	var storedFinishedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT status, finished_at FROM trips WHERE id = $1`, tripID).Scan(&storedStatus, &storedFinishedAt); err != nil {
		t.Fatal(err)
	}
	if storedStatus != string(domain.StatusCompleted) || !storedFinishedAt.Equal(finishedAt) {
		t.Fatalf("stored trip = (%q, %v), want (completed, %v)", storedStatus, storedFinishedAt, finishedAt)
	}
	if _, err := trips.FinishTrip(ctx, tripID, time.Now()); !errors.Is(err, domain.ErrTripCompleted) {
		t.Fatalf("FinishTrip(completed) error = %v, want ErrTripCompleted", err)
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL is required; run make test-integration")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	return pool
}

func validTrip(id, driverID uuid.UUID) *domain.Trip {
	return &domain.Trip{
		ID: id, UserID: uuid.New(), DriverID: driverID,
		Start: domain.Point{Latitude: 59.9, Longitude: 30.3},
		End:   domain.Point{Latitude: 59.8, Longitude: 30.2}, Price: 1450,
	}
}

func cleanupTrip(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM trips WHERE id = $1`, id); err != nil {
			t.Errorf("cleanup trip %s: %v", id, err)
		}
	})
}
