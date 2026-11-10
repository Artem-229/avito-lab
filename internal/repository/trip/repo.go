package trip

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var tripColumns = []string{
	"id", "user_id", "driver_id",
	"start_latitude", "start_longitude",
	"end_latitude", "end_longitude",
	"price", "status",
	"started_at", "finished_at",
	"created_at", "updated_at",
}

type TripsRepository struct {
	pool *pgxpool.Pool
}

func NewTripsRepository(pool *pgxpool.Pool) *TripsRepository {
	return &TripsRepository{
		pool: pool,
	}
}
