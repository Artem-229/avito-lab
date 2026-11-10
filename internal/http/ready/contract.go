package ready

import (
	"context"
)

type repository interface {
	Ping(ctx context.Context) error
}
