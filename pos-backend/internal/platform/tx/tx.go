package tx

import (
	"context"
	"database/sql"
)

type Manager interface {
	WithinTx(ctx context.Context, fn func(context.Context) error) error
}

type txKey struct{}

func ContextWithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func FromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	return tx, ok
}
