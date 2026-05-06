package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	txctx "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/domain"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (r *Repository) execer(ctx context.Context) execer {
	if tx, ok := txctx.FromContext(ctx); ok {
		return tx
	}
	return r.db
}

func (r *Repository) queryer(ctx context.Context) queryer {
	if tx, ok := txctx.FromContext(ctx); ok {
		return tx
	}
	return r.db
}

func dbTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullableString(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return dbTime(*v)
}

func stringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func int64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func normalizeErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique constraint") {
		return fmt.Errorf("%w: %v", domain.ErrDuplicate, err)
	}
	if strings.Contains(msg, "foreign key constraint") || strings.Contains(msg, "check constraint") {
		return fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	return err
}
