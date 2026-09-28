package repository

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdempotencyRepository struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{
		pool: pool,
	}
}

func (r *IdempotencyRepository) Reserve(ctx context.Context, key uuid.UUID, requestHash string, ttl time.Duration) (bool, error) {
	query, args, err := psql.
		Delete("idempotency_keys").
		Where(sq.Eq{"key": key}).
		Where(sq.Lt{"created_at": time.Now().Add(-ttl)}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build delete expired idempotency key: %w", err)
	}

	if _, err = extractTx(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return false, fmt.Errorf("delete expired idempotency key: %w", err)
	}

	query, args, err = psql.
		Insert("idempotency_keys").
		Columns("key", "request_hash").
		Values(key, requestHash).
		Suffix("ON CONFLICT (key) DO NOTHING").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build insert idempotency key: %w", err)
	}

	tag, err := extractTx(ctx, r.pool).Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("insert idempotency key: %w", err)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *IdempotencyRepository) Get(ctx context.Context, key uuid.UUID) (string, uuid.UUID, error) {
	query, args, err := psql.
		Select("request_hash", "trip_id").
		From("idempotency_keys").
		Where(sq.Eq{"key": key}).
		ToSql()
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("build select idempotency key: %w", err)
	}

	var (
		requestHash string
		tripID      uuid.UUID
	)
	if err := extractTx(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&requestHash, &tripID); err != nil {
		return "", uuid.Nil, fmt.Errorf("get idempotency key: %w", err)
	}

	return requestHash, tripID, nil
}

func (r *IdempotencyRepository) SetTrip(ctx context.Context, key uuid.UUID, tripID uuid.UUID) error {
	query, args, err := psql.
		Update("idempotency_keys").
		Set("trip_id", tripID).
		Where(sq.Eq{"key": key}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update idempotency key: %w", err)
	}

	if _, err = extractTx(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("set idempotency key trip: %w", err)
	}

	return nil
}
