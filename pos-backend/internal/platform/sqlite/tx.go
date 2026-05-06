package sqlite

import (
	"context"
	"database/sql"

	txctx "pos-backend/internal/platform/tx"
)

type TxManager struct {
	db *sql.DB
}

type immediateTx struct {
	ctx    context.Context
	conn   *sql.Conn
	closed bool
}

func NewTxManager(db *sql.DB) TxManager {
	return TxManager{db: db}
}

func (m TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := txctx.FromContext(ctx); ok {
		return fn(ctx)
	}
	tx, err := m.beginImmediate(ctx)
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

func (m TxManager) beginImmediate(ctx context.Context) (*immediateTx, error) {
	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &immediateTx{ctx: ctx, conn: conn}, nil
}

func (t *immediateTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.conn.ExecContext(ctx, query, args...)
}

func (t *immediateTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.conn.QueryContext(ctx, query, args...)
}

func (t *immediateTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.conn.QueryRowContext(ctx, query, args...)
}

func (t *immediateTx) Commit() error {
	if t.closed {
		return sql.ErrTxDone
	}
	t.closed = true
	_, err := t.conn.ExecContext(t.ctx, `COMMIT`)
	closeErr := t.conn.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func (t *immediateTx) Rollback() error {
	if t.closed {
		return sql.ErrTxDone
	}
	t.closed = true
	_, err := t.conn.ExecContext(t.ctx, `ROLLBACK`)
	closeErr := t.conn.Close()
	if err != nil {
		return err
	}
	return closeErr
}
