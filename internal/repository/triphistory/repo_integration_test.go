//go:build integration

package triphistory_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
	triprepo "github.com/Artem-229/avito-lab/internal/repository/trip"
	"github.com/Artem-229/avito-lab/internal/repository/triphistory"
)

func TestCreateRecord(t *testing.T) {
	ctx := context.Background()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	id := uuid.New()
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM trips WHERE id=$1`, id); err != nil {
			t.Error(err)
		}
	})
	if _, err := triprepo.NewTripsRepository(pool).CreateTrip(ctx, &domain.Trip{ID: id, UserID: uuid.New(), DriverID: uuid.New(), Price: 100}); err != nil {
		t.Fatal(err)
	}
	repo := triphistory.NewTripsHistoryRepository(pool)
	if err := repo.CreateRecord(ctx, id, nil, domain.StatusActive, "new trip"); err != nil {
		t.Fatal(err)
	}
	from := domain.StatusActive
	if err := repo.CreateRecord(ctx, id, &from, domain.StatusCompleted, "trip finished"); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(ctx, `SELECT from_status, to_status FROM trip_status_history WHERE trip_id=$1 ORDER BY id`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var values [][2]string
	for rows.Next() {
		var from *string
		var to string
		if err := rows.Scan(&from, &to); err != nil {
			t.Fatal(err)
		}
		left := ""
		if from != nil {
			left = *from
		}
		values = append(values, [2]string{left, to})
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != [2]string{"", "active"} || values[1] != [2]string{"active", "completed"} {
		t.Fatalf("history = %v", values)
	}
}
