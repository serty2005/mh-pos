package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateFinancialOperation(ctx context.Context, v *domain.FinancialOperation) error {
	snapshot := v.Snapshot
	if len(snapshot) == 0 {
		snapshot = json.RawMessage(`{}`)
	}
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO financial_operations(id,edge_operation_id,restaurant_id,device_id,shift_id,original_shift_id,check_id,precheck_id,operation_type,operation_kind,status,amount,currency,business_date_local,inventory_disposition,reason,created_by_employee_id,approved_by_employee_id,snapshot,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EdgeOperationID, v.RestaurantID, v.DeviceID, v.ShiftID, v.OriginalShiftID, v.CheckID, v.PrecheckID, string(v.Type), string(v.Kind), string(v.Status), v.Amount, v.Currency, v.BusinessDateLocal, string(v.InventoryDisposition), v.Reason, v.CreatedByEmployeeID, nullableString(v.ApprovedByEmployeeID), string(snapshot), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) CreateFinancialOperationItem(ctx context.Context, v *domain.FinancialOperationItem) error {
	snapshot := v.Snapshot
	if len(snapshot) == 0 {
		snapshot = json.RawMessage(`{}`)
	}
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO financial_operation_items(id,operation_id,scope,order_line_id,payment_id,quantity,amount,currency,tax_amount,snapshot,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OperationID, string(v.Scope), nullableString(v.OrderLineID), nullableString(v.PaymentID), nullableInt64(v.Quantity), v.Amount, v.Currency, v.TaxAmount, string(snapshot), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListFinancialOperations(ctx context.Context, query domain.FinancialOperationListQuery) ([]domain.FinancialOperation, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 9)
	if strings.TrimSpace(query.RestaurantID) != "" {
		where = append(where, "restaurant_id = ?")
		args = append(args, strings.TrimSpace(query.RestaurantID))
	}
	if strings.TrimSpace(query.CheckID) != "" {
		where = append(where, "check_id = ?")
		args = append(args, strings.TrimSpace(query.CheckID))
	}
	if strings.TrimSpace(query.BusinessDateFrom) != "" {
		where = append(where, "business_date_local >= ?")
		args = append(args, strings.TrimSpace(query.BusinessDateFrom))
	}
	if strings.TrimSpace(query.BusinessDateTo) != "" {
		where = append(where, "business_date_local <= ?")
		args = append(args, strings.TrimSpace(query.BusinessDateTo))
	}
	if query.OperationType != "" {
		where = append(where, "operation_type = ?")
		args = append(args, string(query.OperationType))
	}
	if strings.TrimSpace(query.ShiftID) != "" {
		where = append(where, "shift_id = ?")
		args = append(args, strings.TrimSpace(query.ShiftID))
	}
	if strings.TrimSpace(query.OriginalShiftID) != "" {
		where = append(where, "original_shift_id = ?")
		args = append(args, strings.TrimSpace(query.OriginalShiftID))
	}
	limit := query.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	sqlText := financialOperationSelectSQL + ` WHERE ` + strings.Join(where, " AND ") + ` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	return r.queryFinancialOperations(ctx, sqlText, args...)
}

func (r *Repository) ListFinancialOperationsByCheck(ctx context.Context, checkID string) ([]domain.FinancialOperation, error) {
	return r.queryFinancialOperations(ctx, financialOperationSelectSQL+` WHERE check_id = ? ORDER BY created_at, id`, checkID)
}

func (r *Repository) queryFinancialOperations(ctx context.Context, sqlText string, args ...any) ([]domain.FinancialOperation, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, normalizeErr(err)
	}
	var out []domain.FinancialOperation
	for rows.Next() {
		v, err := scanFinancialOperationRows(rows)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		out = append(out, *v)
	}
	if err := normalizeErr(rows.Err()); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, normalizeErr(err)
	}
	for i := range out {
		items, err := r.listFinancialOperationItems(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Items = items
	}
	return out, nil
}

func (r *Repository) SumFinancialOperationAmountByCheck(ctx context.Context, checkID string, typ domain.FinancialOperationType) (int64, error) {
	var amount sql.NullInt64
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM financial_operations WHERE check_id = ? AND operation_type = ?`, checkID, string(typ)).Scan(&amount)
	if err != nil {
		return 0, normalizeErr(err)
	}
	if !amount.Valid {
		return 0, nil
	}
	return amount.Int64, nil
}

func (r *Repository) SumFinancialOperationAmountByPayment(ctx context.Context, paymentID string, typ domain.FinancialOperationType) (int64, error) {
	var amount sql.NullInt64
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COALESCE(SUM(i.amount),0) FROM financial_operation_items i JOIN financial_operations o ON o.id = i.operation_id WHERE i.payment_id = ? AND o.operation_type = ?`, paymentID, string(typ)).Scan(&amount)
	if err != nil {
		return 0, normalizeErr(err)
	}
	if !amount.Valid {
		return 0, nil
	}
	return amount.Int64, nil
}

func (r *Repository) SumFinancialOperationAmountByOrderLine(ctx context.Context, orderLineID string, typ domain.FinancialOperationType) (int64, error) {
	var amount sql.NullInt64
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COALESCE(SUM(i.amount),0) FROM financial_operation_items i JOIN financial_operations o ON o.id = i.operation_id WHERE i.order_line_id = ? AND o.operation_type = ?`, orderLineID, string(typ)).Scan(&amount)
	if err != nil {
		return 0, normalizeErr(err)
	}
	if !amount.Valid {
		return 0, nil
	}
	return amount.Int64, nil
}

func (r *Repository) SumFinancialOperationQuantityByOrderLine(ctx context.Context, orderLineID string, typ domain.FinancialOperationType) (int64, error) {
	var quantity sql.NullInt64
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COALESCE(SUM(i.quantity),0) FROM financial_operation_items i JOIN financial_operations o ON o.id = i.operation_id WHERE i.order_line_id = ? AND o.operation_type = ?`, orderLineID, string(typ)).Scan(&quantity)
	if err != nil {
		return 0, normalizeErr(err)
	}
	if !quantity.Valid {
		return 0, nil
	}
	return quantity.Int64, nil
}

func (r *Repository) listFinancialOperationItems(ctx context.Context, operationID string) ([]domain.FinancialOperationItem, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,operation_id,scope,order_line_id,payment_id,quantity,amount,currency,tax_amount,snapshot,created_at FROM financial_operation_items WHERE operation_id = ? ORDER BY created_at, id`, operationID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []domain.FinancialOperationItem
	for rows.Next() {
		v, err := scanFinancialOperationItemRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

const financialOperationSelectSQL = `SELECT id,edge_operation_id,restaurant_id,device_id,shift_id,original_shift_id,check_id,precheck_id,operation_type,operation_kind,status,amount,currency,business_date_local,inventory_disposition,reason,created_by_employee_id,approved_by_employee_id,snapshot,created_at FROM financial_operations`

func scanFinancialOperationRows(row outboxScanner) (*domain.FinancialOperation, error) {
	var v domain.FinancialOperation
	var typ, kind, status, disposition, snapshot, created string
	var approvedBy sql.NullString
	if err := row.Scan(&v.ID, &v.EdgeOperationID, &v.RestaurantID, &v.DeviceID, &v.ShiftID, &v.OriginalShiftID, &v.CheckID, &v.PrecheckID, &typ, &kind, &status, &v.Amount, &v.Currency, &v.BusinessDateLocal, &disposition, &v.Reason, &v.CreatedByEmployeeID, &approvedBy, &snapshot, &created); err != nil {
		return nil, normalizeErr(err)
	}
	v.Type = domain.FinancialOperationType(typ)
	v.Kind = domain.FinancialOperationKind(kind)
	v.Status = domain.FinancialOperationStatus(status)
	v.InventoryDisposition = domain.InventoryDisposition(disposition)
	v.ApprovedByEmployeeID = stringPtr(approvedBy)
	v.Snapshot = json.RawMessage(snapshot)
	v.CreatedAt = parseTime(created)
	return &v, nil
}

func scanFinancialOperationItemRows(row outboxScanner) (*domain.FinancialOperationItem, error) {
	var v domain.FinancialOperationItem
	var scope, snapshot, created string
	var orderLineID, paymentID sql.NullString
	var quantity sql.NullInt64
	if err := row.Scan(&v.ID, &v.OperationID, &scope, &orderLineID, &paymentID, &quantity, &v.Amount, &v.Currency, &v.TaxAmount, &snapshot, &created); err != nil {
		return nil, normalizeErr(err)
	}
	v.Scope = domain.FinancialOperationItemScope(scope)
	v.OrderLineID = stringPtr(orderLineID)
	v.PaymentID = stringPtr(paymentID)
	v.Quantity = int64Ptr(quantity)
	v.Snapshot = json.RawMessage(snapshot)
	v.CreatedAt = parseTime(created)
	return &v, nil
}
