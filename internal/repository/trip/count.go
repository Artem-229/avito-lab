package trip

import (
	"context"
	"fmt"

	"github.com/Artem-229/avito-lab/internal/infra/postgres/transaction"
)

func (t *TripsRepository) CountActive(ctx context.Context) (int64, error) {
	var count int64
	if err := transaction.ExtractTx(ctx, t.pool).QueryRow(ctx, "SELECT count(*) FROM trips WHERE status = 'active'").Scan(&count); err != nil {
		return 0, fmt.Errorf("count active trips: %w", err)
	}
	return count, nil
}
