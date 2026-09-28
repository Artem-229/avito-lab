package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}

func (t *TripsRepository) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ExtractTx(ctx); ok {
		return fn(ctx)
	}

	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(p)
		}
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil {
			return errors.Join(err, fmt.Errorf("rollback transaction: %w", rbErr))
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func ExtractTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}
