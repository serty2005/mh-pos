package sqlite

import (
	"context"
	"database/sql"

	"pos-backend/internal/pos/domain/ticket"
)

const ticketUnitColumns = `id,ticket_number,restaurant_id,device_id,cash_session_id,shift_id,check_id,order_id,order_line_id,catalog_item_id,menu_item_id,name,sale_date_local,timezone,validity_mode,validity_date_local,cash_shift_sequence,qr_payload,print_status,snapshot,created_at,updated_at`

// CreateTicketUnit вставляет выпущенную ticket unit. UNIQUE(order_line_id) и
// UNIQUE(cash_session_id, cash_shift_sequence) защищают от дублей при replay.
func (r *Repository) CreateTicketUnit(ctx context.Context, v *ticket.TicketUnit) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO ticket_units(`+ticketUnitColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.TicketNumber, v.RestaurantID, v.DeviceID, v.CashSessionID, v.ShiftID, v.CheckID, v.OrderID, v.OrderLineID, v.CatalogItemID, v.MenuItemID,
		v.Name, v.SaleDateLocal, v.Timezone, string(v.ValidityMode), nullableStringValue(v.ValidityDateLocal), v.CashShiftSequence, v.QRPayload, v.PrintStatus, string(v.Snapshot), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetTicketUnit(ctx context.Context, id string) (*ticket.TicketUnit, error) {
	return scanTicketUnit(r.queryer(ctx).QueryRowContext(ctx, `SELECT `+ticketUnitColumns+` FROM ticket_units WHERE id = ?`, id))
}

func (r *Repository) GetTicketUnitByOrderLine(ctx context.Context, orderLineID string) (*ticket.TicketUnit, error) {
	return scanTicketUnit(r.queryer(ctx).QueryRowContext(ctx, `SELECT `+ticketUnitColumns+` FROM ticket_units WHERE order_line_id = ?`, orderLineID))
}

func (r *Repository) ListTicketUnitsByCheck(ctx context.Context, checkID string) ([]ticket.TicketUnit, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT `+ticketUnitColumns+` FROM ticket_units WHERE check_id = ? ORDER BY cash_shift_sequence ASC, id ASC`, checkID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []ticket.TicketUnit
	for rows.Next() {
		v, err := scanTicketUnitRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

// NextTicketCashShiftSequence считает следующий порядковый номер билета внутри кассовой смены.
// Вызывается внутри транзакции CapturePayment, поэтому single-writer SQLite гарантирует монотонность.
func (r *Repository) NextTicketCashShiftSequence(ctx context.Context, cashSessionID string) (int64, error) {
	var next int64
	if err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COALESCE(MAX(cash_shift_sequence), 0) + 1 FROM ticket_units WHERE cash_session_id = ?`, cashSessionID).Scan(&next); err != nil {
		return 0, normalizeErr(err)
	}
	return next, nil
}

func scanTicketUnit(row scanner) (*ticket.TicketUnit, error) {
	return scanTicketUnitRows(row)
}

func scanTicketUnitRows(row scanner) (*ticket.TicketUnit, error) {
	var v ticket.TicketUnit
	var validityMode, snapshot, created, updated string
	var validityDate sql.NullString
	if err := row.Scan(&v.ID, &v.TicketNumber, &v.RestaurantID, &v.DeviceID, &v.CashSessionID, &v.ShiftID, &v.CheckID, &v.OrderID, &v.OrderLineID, &v.CatalogItemID, &v.MenuItemID,
		&v.Name, &v.SaleDateLocal, &v.Timezone, &validityMode, &validityDate, &v.CashShiftSequence, &v.QRPayload, &v.PrintStatus, &snapshot, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.ValidityMode = ticket.ValidityMode(validityMode)
	if validityDate.Valid {
		v.ValidityDateLocal = validityDate.String
	}
	v.Snapshot = []byte(snapshot)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}
