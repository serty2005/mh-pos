package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateShift(ctx context.Context, v *domain.Shift) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO shifts(id,restaurant_id,device_id,opened_by_employee_id,closed_by_employee_id,status,business_date_local,opened_at,closed_at,opening_cash_amount,closing_cash_amount,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceID, v.OpenedByEmployeeID, v.ClosedByEmployeeID, string(v.Status), v.BusinessDateLocal, dbTime(v.OpenedAt), nil, v.OpeningCashAmount, nil, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) UpdateShiftClosed(ctx context.Context, v *domain.Shift) error {
	var closedAt any
	if v.ClosedAt != nil {
		closedAt = dbTime(*v.ClosedAt)
	}
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE shifts SET closed_by_employee_id = ?, status = ?, closed_at = ?, closing_cash_amount = ?, updated_at = ? WHERE id = ?`,
		v.ClosedByEmployeeID, string(v.Status), closedAt, v.ClosingCashAmount, dbTime(v.UpdatedAt), v.ID)
	return normalizeErr(err)
}

func (r *Repository) GetShift(ctx context.Context, id string) (*domain.Shift, error) {
	return r.scanShift(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,device_id,opened_by_employee_id,closed_by_employee_id,status,business_date_local,opened_at,closed_at,opening_cash_amount,closing_cash_amount,created_at,updated_at FROM shifts WHERE id = ?`, id))
}

func (r *Repository) GetOpenShiftByDevice(ctx context.Context, deviceID string) (*domain.Shift, error) {
	return r.scanShift(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,device_id,opened_by_employee_id,closed_by_employee_id,status,business_date_local,opened_at,closed_at,opening_cash_amount,closing_cash_amount,created_at,updated_at FROM shifts WHERE device_id = ? AND status = 'open'`, deviceID))
}

func (r *Repository) GetOpenShiftByEmployee(ctx context.Context, restaurantID, employeeID string) (*domain.Shift, error) {
	return r.scanShift(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,device_id,opened_by_employee_id,closed_by_employee_id,status,business_date_local,opened_at,closed_at,opening_cash_amount,closing_cash_amount,created_at,updated_at FROM shifts WHERE restaurant_id = ? AND opened_by_employee_id = ? AND status = 'open'`, restaurantID, employeeID))
}

func (r *Repository) ListRecentShiftsByEmployee(ctx context.Context, restaurantID, employeeID string, limit int) ([]domain.Shift, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,device_id,opened_by_employee_id,closed_by_employee_id,status,business_date_local,opened_at,closed_at,opening_cash_amount,closing_cash_amount,created_at,updated_at FROM shifts WHERE restaurant_id = ? AND opened_by_employee_id = ? ORDER BY opened_at DESC LIMIT ?`, restaurantID, employeeID, limit)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []domain.Shift
	for rows.Next() {
		v, err := scanShiftRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) scanShift(row *sql.Row) (*domain.Shift, error) {
	return scanShiftRows(row)
}

type shiftScanner interface {
	Scan(dest ...any) error
}

func scanShiftRows(row shiftScanner) (*domain.Shift, error) {
	var v domain.Shift
	var status, opened, created, updated string
	var closedBy sql.NullString
	var closedAt sql.NullString
	var closingCash sql.NullInt64
	err := row.Scan(&v.ID, &v.RestaurantID, &v.DeviceID, &v.OpenedByEmployeeID, &closedBy, &status, &v.BusinessDateLocal, &opened, &closedAt, &v.OpeningCashAmount, &closingCash, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	if closedBy.Valid {
		v.ClosedByEmployeeID = &closedBy.String
	}
	if closedAt.Valid {
		t := parseTime(closedAt.String)
		v.ClosedAt = &t
	}
	if closingCash.Valid {
		v.ClosingCashAmount = &closingCash.Int64
	}
	v.Status = domain.ShiftStatus(status)
	v.OpenedAt = parseTime(opened)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) HasOpenOrdersForShift(ctx context.Context, shiftID string) (bool, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM orders WHERE shift_id = ? AND status IN ('open', 'locked')`, shiftID).Scan(&n)
	return n > 0, normalizeErr(err)
}
