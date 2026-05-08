package sqlite

import (
	"context"
	"database/sql"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateCashSession(ctx context.Context, v *domain.CashSession) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO cash_sessions(id,edge_cash_session_id,restaurant_id,device_id,shift_id,opened_by_employee_id,closed_by_employee_id,status,business_date_local,opening_cash_amount,closing_cash_amount,opened_at,closed_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EdgeCashSessionID, v.RestaurantID, v.DeviceID, v.ShiftID, v.OpenedByEmployeeID, nullableString(v.ClosedByEmployeeID), string(v.Status), v.BusinessDateLocal, v.OpeningCashAmount, nullableInt64(v.ClosingCashAmount), dbTime(v.OpenedAt), nil, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) UpdateCashSessionClosed(ctx context.Context, v *domain.CashSession) error {
	var closedAt any
	if v.ClosedAt != nil {
		closedAt = dbTime(*v.ClosedAt)
	}
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE cash_sessions SET closed_by_employee_id = ?, status = ?, closing_cash_amount = ?, closed_at = ?, updated_at = ? WHERE id = ?`,
		nullableString(v.ClosedByEmployeeID), string(v.Status), nullableInt64(v.ClosingCashAmount), closedAt, dbTime(v.UpdatedAt), v.ID)
	return normalizeErr(err)
}

func (r *Repository) GetCashSession(ctx context.Context, id string) (*domain.CashSession, error) {
	return r.scanCashSession(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,edge_cash_session_id,restaurant_id,device_id,shift_id,opened_by_employee_id,closed_by_employee_id,status,business_date_local,opening_cash_amount,closing_cash_amount,opened_at,closed_at,created_at,updated_at FROM cash_sessions WHERE id = ?`, id))
}

func (r *Repository) GetOpenCashSessionByDevice(ctx context.Context, deviceID string) (*domain.CashSession, error) {
	return r.scanCashSession(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,edge_cash_session_id,restaurant_id,device_id,shift_id,opened_by_employee_id,closed_by_employee_id,status,business_date_local,opening_cash_amount,closing_cash_amount,opened_at,closed_at,created_at,updated_at FROM cash_sessions WHERE device_id = ? AND status = 'open'`, deviceID))
}

func (r *Repository) scanCashSession(row *sql.Row) (*domain.CashSession, error) {
	var v domain.CashSession
	var status, opened, created, updated string
	var closedBy sql.NullString
	var closingCash sql.NullInt64
	var closedAt sql.NullString
	err := row.Scan(&v.ID, &v.EdgeCashSessionID, &v.RestaurantID, &v.DeviceID, &v.ShiftID, &v.OpenedByEmployeeID, &closedBy, &status, &v.BusinessDateLocal, &v.OpeningCashAmount, &closingCash, &opened, &closedAt, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.ClosedByEmployeeID = stringPtr(closedBy)
	v.ClosingCashAmount = int64Ptr(closingCash)
	if closedAt.Valid {
		t := parseTime(closedAt.String)
		v.ClosedAt = &t
	}
	v.Status = domain.CashSessionStatus(status)
	v.OpenedAt = parseTime(opened)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) CreateCashDrawerEvent(ctx context.Context, v *domain.CashDrawerEvent) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO cash_drawer_events(id,edge_cash_drawer_event_id,cash_session_id,restaurant_id,device_id,shift_id,created_by_employee_id,event_type,amount,reason,note,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EdgeCashDrawerEventID, v.CashSessionID, v.RestaurantID, v.DeviceID, v.ShiftID, v.CreatedByEmployeeID, string(v.EventType), v.Amount, nullableString(v.Reason), nullableString(v.Note), dbTime(v.OccurredAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}
