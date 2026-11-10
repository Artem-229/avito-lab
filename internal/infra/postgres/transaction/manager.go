package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const IsolationLevel = pgx.ReadCommitted

type executor interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type txKey struct{}

type TxManager struct {
	pool            *pgxpool.Pool
	rollbackTimeout time.Duration
}

func NewTxManager(pool *pgxpool.Pool, rollbackTimeout time.Duration) *TxManager {
	return &TxManager{
		rollbackTimeout: rollbackTimeout,
		pool:            pool,
	}
}

func (t *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: IsolationLevel})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = t.rollback(ctx, tx)
			panic(p)
		}
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		if rbErr := t.rollback(ctx, tx); rbErr != nil {
			return errors.Join(err, fmt.Errorf("rollback transaction: %w", rbErr))
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func ExtractTx(ctx context.Context, pool *pgxpool.Pool) executor {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

func (t *TxManager) rollback(ctx context.Context, tx pgx.Tx) error {
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), t.rollbackTimeout)
	defer cancel()
	return tx.Rollback(rollbackCtx)
}
