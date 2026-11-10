//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package create

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
)

type tripUsecase interface {
	CreateTrip(ctx context.Context, trip *domain.Trip, idempotencyKey *uuid.UUID) (*domain.Trip, bool, error)
}
