package sqlite

import (
	"context"
	"database/sql"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreatePrecheck(ctx context.Context, v *domain.Precheck) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO prechecks(id,order_id,status,subtotal,discount_total,tax_total,total,created_at,issued_at,closed_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderID, string(v.Status), v.Subtotal, v.DiscountTotal, v.TaxTotal, v.Total, dbTime(v.CreatedAt), dbTime(v.IssuedAt), nullableTime(v.ClosedAt))
	return normalizeErr(err)
}

func (r *Repository) GetPrecheck(ctx context.Context, id string) (*domain.Precheck, error) {
	return r.scanPrecheck(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,order_id,status,subtotal,discount_total,tax_total,total,created_at,issued_at,closed_at FROM prechecks WHERE id = ?`, id))
}

func (r *Repository) GetActivePrecheckByOrder(ctx context.Context, orderID string) (*domain.Precheck, error) {
	return r.scanPrecheck(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,order_id,status,subtotal,discount_total,tax_total,total,created_at,issued_at,closed_at FROM prechecks WHERE order_id = ? AND status = 'issued'`, orderID))
}

func (r *Repository) scanPrecheck(row *sql.Row) (*domain.Precheck, error) {
	var v domain.Precheck
	var status, created, issued string
	var closed sql.NullString
	err := row.Scan(&v.ID, &v.OrderID, &status, &v.Subtotal, &v.DiscountTotal, &v.TaxTotal, &v.Total, &created, &issued, &closed)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.PrecheckStatus(status)
	v.CreatedAt = parseTime(created)
	v.IssuedAt = parseTime(issued)
	if closed.Valid {
		t := parseTime(closed.String)
		v.ClosedAt = &t
	}
	return &v, nil
}
