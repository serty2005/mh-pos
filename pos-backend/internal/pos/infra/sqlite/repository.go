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

func stringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
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

func (r *Repository) CreateRestaurant(ctx context.Context, v *domain.Restaurant) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO restaurants(id,name,timezone,currency,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		v.ID, v.Name, v.Timezone, v.Currency, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRestaurants(ctx context.Context) ([]domain.Restaurant, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,name,timezone,currency,active,created_at,updated_at FROM restaurants ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Restaurant
	for rows.Next() {
		var v domain.Restaurant
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.Name, &v.Timezone, &v.Currency, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateDevice(ctx context.Context, v *domain.Device) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO devices(id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceCode, v.Name, v.Type, boolInt(v.Active), dbTime(v.RegisteredAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListDevices(ctx context.Context) ([]domain.Device, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at FROM devices ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Device
	for rows.Next() {
		var v domain.Device
		var active int
		var registered, created, updated string
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.DeviceCode, &v.Name, &v.Type, &active, &registered, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.RegisteredAt = parseTime(registered)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateRole(ctx context.Context, v *domain.Role) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO roles(id,name,permissions_json,active,created_at,updated_at) VALUES (?,?,?,?,?,?)`,
		v.ID, v.Name, v.PermissionsJSON, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,name,permissions_json,active,created_at,updated_at FROM roles ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Role
	for rows.Next() {
		var v domain.Role
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.Name, &v.PermissionsJSON, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateEmployee(ctx context.Context, v *domain.Employee) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO employees(id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.RoleID, v.Name, v.PINHash, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListEmployees(ctx context.Context) ([]domain.Employee, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at FROM employees ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Employee
	for rows.Next() {
		var v domain.Employee
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.RoleID, &v.Name, &v.PINHash, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) ArchiveEmployee(ctx context.Context, id, updatedAt string) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE employees SET active = 0, updated_at = ? WHERE id = ?`, updatedAt, id)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateCatalogItem(ctx context.Context, v *domain.CatalogItem) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO catalog_items(id,type,name,sku,base_unit,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, string(v.Type), v.Name, v.SKU, v.BaseUnit, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListCatalogItems(ctx context.Context) ([]domain.CatalogItem, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,type,name,sku,base_unit,active,created_at,updated_at FROM catalog_items ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CatalogItem
	for rows.Next() {
		var v domain.CatalogItem
		var typ string
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &typ, &v.Name, &v.SKU, &v.BaseUnit, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Type = domain.CatalogItemType(typ)
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) GetCatalogItem(ctx context.Context, id string) (*domain.CatalogItem, error) {
	var v domain.CatalogItem
	var typ string
	var active int
	var created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,type,name,sku,base_unit,active,created_at,updated_at FROM catalog_items WHERE id = ?`, id).
		Scan(&v.ID, &typ, &v.Name, &v.SKU, &v.BaseUnit, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Type = domain.CatalogItemType(typ)
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) CatalogItemInUse(ctx context.Context, id string) (bool, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM menu_items WHERE catalog_item_id = ?`, id).Scan(&n)
	return n > 0, normalizeErr(err)
}

func (r *Repository) CreateMenuItem(ctx context.Context, v *domain.MenuItem) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO menu_items(id,catalog_item_id,name,price,currency,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.CatalogItemID, v.Name, v.Price, v.Currency, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListMenuItems(ctx context.Context) ([]domain.MenuItem, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,catalog_item_id,name,price,currency,active,created_at,updated_at FROM menu_items ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MenuItem
	for rows.Next() {
		var v domain.MenuItem
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.CatalogItemID, &v.Name, &v.Price, &v.Currency, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) GetMenuItem(ctx context.Context, id string) (*domain.MenuItem, error) {
	var v domain.MenuItem
	var active int
	var created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,catalog_item_id,name,price,currency,active,created_at,updated_at FROM menu_items WHERE id = ?`, id).
		Scan(&v.ID, &v.CatalogItemID, &v.Name, &v.Price, &v.Currency, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) CreateShift(ctx context.Context, v *domain.Shift) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO shifts(id,restaurant_id,device_id,opened_by_employee_id,closed_by_employee_id,status,opened_at,closed_at,opening_cash_amount,closing_cash_amount,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceID, v.OpenedByEmployeeID, v.ClosedByEmployeeID, string(v.Status), dbTime(v.OpenedAt), nil, v.OpeningCashAmount, nil, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
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
	return r.scanShift(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,device_id,opened_by_employee_id,closed_by_employee_id,status,opened_at,closed_at,opening_cash_amount,closing_cash_amount,created_at,updated_at FROM shifts WHERE id = ?`, id))
}

func (r *Repository) GetOpenShiftByDevice(ctx context.Context, deviceID string) (*domain.Shift, error) {
	return r.scanShift(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,device_id,opened_by_employee_id,closed_by_employee_id,status,opened_at,closed_at,opening_cash_amount,closing_cash_amount,created_at,updated_at FROM shifts WHERE device_id = ? AND status = 'open'`, deviceID))
}

func (r *Repository) scanShift(row *sql.Row) (*domain.Shift, error) {
	var v domain.Shift
	var status, opened, created, updated string
	var closedBy sql.NullString
	var closedAt sql.NullString
	var closingCash sql.NullInt64
	err := row.Scan(&v.ID, &v.RestaurantID, &v.DeviceID, &v.OpenedByEmployeeID, &closedBy, &status, &opened, &closedAt, &v.OpeningCashAmount, &closingCash, &created, &updated)
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
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM orders WHERE shift_id = ? AND status = 'open'`, shiftID).Scan(&n)
	return n > 0, normalizeErr(err)
}

func (r *Repository) CreateOrder(ctx context.Context, v *domain.Order) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_name,guest_count,opened_at,closed_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EdgeOrderID, v.RestaurantID, v.DeviceID, v.ShiftID, string(v.Status), v.TableName, v.GuestCount, dbTime(v.OpenedAt), nil, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	var v domain.Order
	var status, opened, created, updated string
	var closed sql.NullString
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,edge_order_id,restaurant_id,device_id,shift_id,status,table_name,guest_count,opened_at,closed_at,created_at,updated_at FROM orders WHERE id = ?`, id).
		Scan(&v.ID, &v.EdgeOrderID, &v.RestaurantID, &v.DeviceID, &v.ShiftID, &status, &v.TableName, &v.GuestCount, &opened, &closed, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.OrderStatus(status)
	v.OpenedAt = parseTime(opened)
	if closed.Valid {
		t := parseTime(closed.String)
		v.ClosedAt = &t
	}
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) UpdateOrderClosed(ctx context.Context, v *domain.Order) error {
	var closedAt any
	if v.ClosedAt != nil {
		closedAt = dbTime(*v.ClosedAt)
	}
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE orders SET status = ?, closed_at = ?, updated_at = ? WHERE id = ?`, string(v.Status), closedAt, dbTime(v.UpdatedAt), v.ID)
	return normalizeErr(err)
}

func (r *Repository) CreateOrderLine(ctx context.Context, v *domain.OrderLine) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO order_lines(id,order_id,menu_item_id,catalog_item_id,name,quantity,unit_price,total_price,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderID, v.MenuItemID, v.CatalogItemID, v.Name, v.Quantity, v.UnitPrice, v.TotalPrice, string(v.Status), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListOrderLines(ctx context.Context, orderID string) ([]domain.OrderLine, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,order_id,menu_item_id,catalog_item_id,name,quantity,unit_price,total_price,status,created_at,updated_at FROM order_lines WHERE order_id = ? ORDER BY created_at`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OrderLine
	for rows.Next() {
		var v domain.OrderLine
		var status, created, updated string
		if err := rows.Scan(&v.ID, &v.OrderID, &v.MenuItemID, &v.CatalogItemID, &v.Name, &v.Quantity, &v.UnitPrice, &v.TotalPrice, &status, &created, &updated); err != nil {
			return nil, err
		}
		v.Status = domain.OrderLineStatus(status)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCheck(ctx context.Context, v *domain.Check) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO checks(id,order_id,status,subtotal,discount_total,tax_total,total,paid_total,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderID, string(v.Status), v.Subtotal, v.DiscountTotal, v.TaxTotal, v.Total, v.PaidTotal, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetCheck(ctx context.Context, id string) (*domain.Check, error) {
	return r.scanCheck(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,order_id,status,subtotal,discount_total,tax_total,total,paid_total,created_at,updated_at FROM checks WHERE id = ?`, id))
}

func (r *Repository) GetCheckByOrder(ctx context.Context, orderID string) (*domain.Check, error) {
	return r.scanCheck(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,order_id,status,subtotal,discount_total,tax_total,total,paid_total,created_at,updated_at FROM checks WHERE order_id = ?`, orderID))
}

func (r *Repository) scanCheck(row *sql.Row) (*domain.Check, error) {
	var v domain.Check
	var status, created, updated string
	err := row.Scan(&v.ID, &v.OrderID, &status, &v.Subtotal, &v.DiscountTotal, &v.TaxTotal, &v.Total, &v.PaidTotal, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.CheckStatus(status)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) UpdateCheckPaidTotal(ctx context.Context, v *domain.Check) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE checks SET status = ?, paid_total = ?, updated_at = ? WHERE id = ?`,
		string(v.Status), v.PaidTotal, dbTime(v.UpdatedAt), v.ID)
	return normalizeErr(err)
}

func (r *Repository) CreatePayment(ctx context.Context, v *domain.Payment) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO payments(id,check_id,method,amount,currency,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.CheckID, string(v.Method), v.Amount, v.Currency, string(v.Status), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) CreateOutboxMessage(ctx context.Context, v *domain.OutboxMessage) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO pos_sync_outbox(id,command_id,origin,restaurant_id,device_id,aggregate_type,aggregate_id,command_type,payload_json,status,attempts,last_error,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.CommandID, string(v.Origin), nullableString(v.RestaurantID), v.DeviceID, v.AggregateType, v.AggregateID, v.CommandType, v.PayloadJSON, string(v.Status), v.Attempts, v.LastError, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetOutboxByCommandID(ctx context.Context, commandID string) (*domain.OutboxMessage, error) {
	return r.scanOutbox(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,command_id,origin,restaurant_id,device_id,aggregate_type,aggregate_id,command_type,payload_json,status,attempts,last_error,created_at,updated_at FROM pos_sync_outbox WHERE command_id = ?`, commandID))
}

func (r *Repository) ListOutbox(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,command_id,origin,restaurant_id,device_id,aggregate_type,aggregate_id,command_type,payload_json,status,attempts,last_error,created_at,updated_at FROM pos_sync_outbox ORDER BY created_at LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.OutboxMessage
	for rows.Next() {
		v, err := scanOutboxRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repository) scanOutbox(row *sql.Row) (*domain.OutboxMessage, error) {
	var v domain.OutboxMessage
	var restaurantID, lastErr sql.NullString
	var status, created, updated string
	var origin string
	err := row.Scan(&v.ID, &v.CommandID, &origin, &restaurantID, &v.DeviceID, &v.AggregateType, &v.AggregateID, &v.CommandType, &v.PayloadJSON, &status, &v.Attempts, &lastErr, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Origin = domain.CommandOrigin(origin)
	v.RestaurantID = stringPtr(restaurantID)
	v.LastError = stringPtr(lastErr)
	v.Status = domain.OutboxStatus(status)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

type outboxScanner interface {
	Scan(...any) error
}

func scanOutboxRows(row outboxScanner) (*domain.OutboxMessage, error) {
	var v domain.OutboxMessage
	var restaurantID, lastErr sql.NullString
	var status, created, updated string
	var origin string
	if err := row.Scan(&v.ID, &v.CommandID, &origin, &restaurantID, &v.DeviceID, &v.AggregateType, &v.AggregateID, &v.CommandType, &v.PayloadJSON, &status, &v.Attempts, &lastErr, &created, &updated); err != nil {
		return nil, err
	}
	v.Origin = domain.CommandOrigin(origin)
	v.RestaurantID = stringPtr(restaurantID)
	v.LastError = stringPtr(lastErr)
	v.Status = domain.OutboxStatus(status)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) MarkOutboxSent(ctx context.Context, id, updatedAt string) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE pos_sync_outbox SET status = 'sent', updated_at = ? WHERE id = ?`, updatedAt, id)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) MarkOutboxFailed(ctx context.Context, id, errorText, updatedAt string) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE pos_sync_outbox SET status = 'failed', attempts = attempts + 1, last_error = ?, updated_at = ? WHERE id = ?`, errorText, updatedAt, id)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) CountOutbox(ctx context.Context) (int, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM pos_sync_outbox`).Scan(&n)
	return n, normalizeErr(err)
}
