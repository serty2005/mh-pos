package sqlite

import (
	"context"
	"database/sql"

	txctx "pos-backend/internal/platform/tx"
)

type TxManager struct {
	db *sql.DB
}

func NewTxManager(db *sql.DB) TxManager {
	return TxManager{db: db}
}

func (m TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := txctx.FromContext(ctx); ok {
		return fn(ctx)
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txCtx := txctx.ContextWithTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
