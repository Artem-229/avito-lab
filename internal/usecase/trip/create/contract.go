//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
)

type tripsRepository interface {
	CreateTrip(ctx context.Context, trip *domain.Trip) (*domain.Trip, error)
	GetTrip(ctx context.Context, tripID uuid.UUID) (*domain.Trip, error)
}
type tripsHistoryRepository interface {
	CreateRecord(ctx context.Context, tripID uuid.UUID, from *domain.Status, to domain.Status, reason string) error
}
type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
type idempotencyRepository interface {
	Reserve(ctx context.Context, key uuid.UUID, requestHash string, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key uuid.UUID) (string, uuid.UUID, error)
	SetTrip(ctx context.Context, key, tripID uuid.UUID) error
}
