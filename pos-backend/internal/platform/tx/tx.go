package tx

import (
	"context"
	"database/sql"
)

type Manager interface {
	WithinTx(ctx context.Context, fn func(context.Context) error) error
}

type txKey struct{}

type Runner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func ContextWithTx(ctx context.Context, tx Runner) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func FromContext(ctx context.Context) (Runner, bool) {
	tx, ok := ctx.Value(txKey{}).(Runner)
	return tx, ok
}
