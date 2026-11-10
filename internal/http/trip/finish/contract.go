//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package finish

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/Artem-229/avito-lab/internal/domain/trip"
)

type tripUsecase interface {
	FinishTrip(ctx context.Context, tripID uuid.UUID) (*domain.Trip, error)
}
