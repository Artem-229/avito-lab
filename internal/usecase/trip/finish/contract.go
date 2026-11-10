//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
)

type tripsRepository interface {
	FinishTrip(ctx context.Context, tripID uuid.UUID, finishedAt time.Time) (*domain.Trip, error)
	GetTrip(ctx context.Context, tripID uuid.UUID) (*domain.Trip, error)
}
type tripsHistoryRepository interface {
	CreateRecord(ctx context.Context, tripID uuid.UUID, from *domain.Status, to domain.Status, reason string) error
}
type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
