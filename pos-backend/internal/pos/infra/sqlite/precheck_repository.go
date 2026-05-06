package sqlite

import (
	"context"
	"database/sql"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreatePrecheck(ctx context.Context, v *domain.Precheck) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO prechecks(id,order_id,status,version,supersedes_precheck_id,subtotal,discount_total,tax_total,total,paid_total,created_at,issued_at,closed_at,cancelled_by_employee_id,cancellation_reason) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderID, string(v.Status), v.Version, nullableString(v.SupersedesPrecheckID), v.Subtotal, v.DiscountTotal, v.TaxTotal, v.Total, v.PaidTotal, dbTime(v.CreatedAt), dbTime(v.IssuedAt), nullableTime(v.ClosedAt), nullableString(v.CancelledByEmployeeID), nullableString(v.CancellationReason))
	return normalizeErr(err)
}

func (r *Repository) GetPrecheck(ctx context.Context, id string) (*domain.Precheck, error) {
	return r.scanPrecheck(r.queryer(ctx).QueryRowContext(ctx, precheckSelectSQL+` WHERE id = ?`, id))
}

func (r *Repository) GetActivePrecheckByOrder(ctx context.Context, orderID string) (*domain.Precheck, error) {
	return r.scanPrecheck(r.queryer(ctx).QueryRowContext(ctx, precheckSelectSQL+` WHERE order_id = ? AND status = 'issued'`, orderID))
}

func (r *Repository) ListPrechecksByOrder(ctx context.Context, orderID string) ([]domain.Precheck, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, precheckSelectSQL+` WHERE order_id = ? ORDER BY version, created_at`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Precheck
	for rows.Next() {
		v, err := scanPrecheckFields(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repository) UpdatePrecheckLifecycle(ctx context.Context, v *domain.Precheck) error {
	result, err := r.execer(ctx).ExecContext(ctx, `UPDATE prechecks SET status = ?, closed_at = ?, cancelled_by_employee_id = ?, cancellation_reason = ? WHERE id = ? AND status = 'issued' AND paid_total = 0`,
		string(v.Status), nullableTime(v.ClosedAt), nullableString(v.CancelledByEmployeeID), nullableString(v.CancellationReason), v.ID)
	if err != nil {
		return normalizeErr(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrConflict
	}
	return nil
}

func (r *Repository) scanPrecheck(row *sql.Row) (*domain.Precheck, error) {
	v, err := scanPrecheckFields(row)
	if err != nil {
		return nil, normalizeErr(err)
	}
	return v, nil
}

const precheckSelectSQL = `SELECT id,order_id,status,version,supersedes_precheck_id,subtotal,discount_total,tax_total,total,paid_total,created_at,issued_at,closed_at,cancelled_by_employee_id,cancellation_reason FROM prechecks`

type precheckScanner interface {
	Scan(dest ...any) error
}

func scanPrecheckFields(scanner precheckScanner) (*domain.Precheck, error) {
	var v domain.Precheck
	var status, created, issued string
	var supersedes, closed, cancelledBy, reason sql.NullString
	err := scanner.Scan(&v.ID, &v.OrderID, &status, &v.Version, &supersedes, &v.Subtotal, &v.DiscountTotal, &v.TaxTotal, &v.Total, &v.PaidTotal, &created, &issued, &closed, &cancelledBy, &reason)
	if err != nil {
		return nil, err
	}
	v.Status = domain.PrecheckStatus(status)
	if supersedes.Valid {
		value := supersedes.String
		v.SupersedesPrecheckID = &value
	}
	v.CreatedAt = parseTime(created)
	v.IssuedAt = parseTime(issued)
	if closed.Valid {
		t := parseTime(closed.String)
		v.ClosedAt = &t
	}
	if cancelledBy.Valid {
		value := cancelledBy.String
		v.CancelledByEmployeeID = &value
	}
	if reason.Valid {
		value := reason.String
		v.CancellationReason = &value
	}
	return &v, nil
}
