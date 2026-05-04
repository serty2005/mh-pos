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

func (r *Repository) CreateRecipeVersion(ctx context.Context, v *domain.RecipeVersion) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.DishCatalogItemID, v.Version, v.Name, string(v.Status), v.YieldQuantity, v.YieldUnit, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRecipeVersions(ctx context.Context) ([]domain.RecipeVersion, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at FROM recipe_versions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecipeVersion
	for rows.Next() {
		var v domain.RecipeVersion
		var status, created, updated string
		var active int
		if err := rows.Scan(&v.ID, &v.DishCatalogItemID, &v.Version, &v.Name, &status, &v.YieldQuantity, &v.YieldUnit, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Status = domain.RecipeVersionStatus(status)
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateRecipeLine(ctx context.Context, v *domain.RecipeLine) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.RecipeVersionID, v.CatalogItemID, v.Quantity, v.Unit, v.LossPercent, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRecipeLines(ctx context.Context, recipeVersionID string) ([]domain.RecipeLine, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at FROM recipe_lines WHERE recipe_version_id = ? ORDER BY created_at`, recipeVersionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecipeLine
	for rows.Next() {
		var v domain.RecipeLine
		var created, updated string
		if err := rows.Scan(&v.ID, &v.RecipeVersionID, &v.CatalogItemID, &v.Quantity, &v.Unit, &v.LossPercent, &created, &updated); err != nil {
			return nil, err
		}
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePurchaseReceipt(ctx context.Context, v *domain.PurchaseReceipt) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO purchase_receipts(id,restaurant_id,device_id,supplier_name,document_number,status,received_at,total_amount,currency,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceID, v.SupplierName, v.DocumentNumber, string(v.Status), dbTime(v.ReceivedAt), v.TotalAmount, v.Currency, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListPurchaseReceipts(ctx context.Context) ([]domain.PurchaseReceipt, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,device_id,supplier_name,document_number,status,received_at,total_amount,currency,created_at,updated_at FROM purchase_receipts ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PurchaseReceipt
	for rows.Next() {
		var v domain.PurchaseReceipt
		var status, received, created, updated string
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.DeviceID, &v.SupplierName, &v.DocumentNumber, &status, &received, &v.TotalAmount, &v.Currency, &created, &updated); err != nil {
			return nil, err
		}
		v.Status = domain.PurchaseReceiptStatus(status)
		v.ReceivedAt = parseTime(received)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePurchaseReceiptLine(ctx context.Context, v *domain.PurchaseReceiptLine) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO purchase_receipt_lines(id,purchase_receipt_id,catalog_item_id,quantity,unit,unit_cost,total_cost,currency,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.PurchaseReceiptID, v.CatalogItemID, v.Quantity, v.Unit, v.UnitCost, v.TotalCost, v.Currency, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListPurchaseReceiptLines(ctx context.Context, purchaseReceiptID string) ([]domain.PurchaseReceiptLine, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,purchase_receipt_id,catalog_item_id,quantity,unit,unit_cost,total_cost,currency,created_at,updated_at FROM purchase_receipt_lines WHERE purchase_receipt_id = ? ORDER BY created_at`, purchaseReceiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PurchaseReceiptLine
	for rows.Next() {
		var v domain.PurchaseReceiptLine
		var created, updated string
		if err := rows.Scan(&v.ID, &v.PurchaseReceiptID, &v.CatalogItemID, &v.Quantity, &v.Unit, &v.UnitCost, &v.TotalCost, &v.Currency, &created, &updated); err != nil {
			return nil, err
		}
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateStockDocument(ctx context.Context, v *domain.StockDocument) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO stock_documents(id,restaurant_id,device_id,document_type,source_type,source_id,status,occurred_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceID, string(v.Type), nullableString(v.SourceType), nullableString(v.SourceID), string(v.Status), dbTime(v.OccurredAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListStockDocuments(ctx context.Context) ([]domain.StockDocument, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,device_id,document_type,source_type,source_id,status,occurred_at,created_at,updated_at FROM stock_documents ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StockDocument
	for rows.Next() {
		var v domain.StockDocument
		var typ, status, occurred, created, updated string
		var sourceType, sourceID sql.NullString
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.DeviceID, &typ, &sourceType, &sourceID, &status, &occurred, &created, &updated); err != nil {
			return nil, err
		}
		v.Type = domain.StockDocumentType(typ)
		v.SourceType = stringPtr(sourceType)
		v.SourceID = stringPtr(sourceID)
		v.Status = domain.StockDocumentStatus(status)
		v.OccurredAt = parseTime(occurred)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateStockMove(ctx context.Context, v *domain.StockMove) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO stock_moves(id,stock_document_id,catalog_item_id,location_id,movement_type,quantity,unit,unit_cost,total_cost,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.StockDocumentID, v.CatalogItemID, nullableString(v.LocationID), string(v.Type), v.Quantity, v.Unit, nullableInt64(v.UnitCost), nullableInt64(v.TotalCost), dbTime(v.OccurredAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListStockMoves(ctx context.Context, stockDocumentID string) ([]domain.StockMove, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,stock_document_id,catalog_item_id,location_id,movement_type,quantity,unit,unit_cost,total_cost,occurred_at,created_at FROM stock_moves WHERE stock_document_id = ? ORDER BY created_at`, stockDocumentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StockMove
	for rows.Next() {
		var v domain.StockMove
		var location sql.NullString
		var unitCost, totalCost sql.NullInt64
		var typ, occurred, created string
		if err := rows.Scan(&v.ID, &v.StockDocumentID, &v.CatalogItemID, &location, &typ, &v.Quantity, &v.Unit, &unitCost, &totalCost, &occurred, &created); err != nil {
			return nil, err
		}
		v.LocationID = stringPtr(location)
		v.Type = domain.StockMoveType(typ)
		v.UnitCost = int64Ptr(unitCost)
		v.TotalCost = int64Ptr(totalCost)
		v.OccurredAt = parseTime(occurred)
		v.CreatedAt = parseTime(created)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertStockBalance(ctx context.Context, v *domain.StockBalance) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO stock_balances(id,catalog_item_id,location_id,quantity,unit,updated_at) VALUES (?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET catalog_item_id = excluded.catalog_item_id, location_id = excluded.location_id, quantity = excluded.quantity, unit = excluded.unit, updated_at = excluded.updated_at`,
		v.ID, v.CatalogItemID, nullableString(v.LocationID), v.Quantity, v.Unit, dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListStockBalances(ctx context.Context) ([]domain.StockBalance, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,catalog_item_id,location_id,quantity,unit,updated_at FROM stock_balances ORDER BY updated_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.StockBalance
	for rows.Next() {
		var v domain.StockBalance
		var location sql.NullString
		var updated string
		if err := rows.Scan(&v.ID, &v.CatalogItemID, &location, &v.Quantity, &v.Unit, &updated); err != nil {
			return nil, err
		}
		v.LocationID = stringPtr(location)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateItemCost(ctx context.Context, v *domain.ItemCost) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO item_costs(id,catalog_item_id,cost_type,amount,currency,source_type,source_id,effective_at,created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.CatalogItemID, string(v.Type), v.Amount, v.Currency, nullableString(v.SourceType), nullableString(v.SourceID), dbTime(v.EffectiveAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetLastItemCost(ctx context.Context, catalogItemID string) (*domain.ItemCost, error) {
	return r.scanItemCost(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,catalog_item_id,cost_type,amount,currency,source_type,source_id,effective_at,created_at FROM item_costs WHERE catalog_item_id = ? AND cost_type = 'last_purchase' ORDER BY effective_at DESC, created_at DESC LIMIT 1`, catalogItemID))
}

func (r *Repository) ListItemCosts(ctx context.Context, catalogItemID string) ([]domain.ItemCost, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,catalog_item_id,cost_type,amount,currency,source_type,source_id,effective_at,created_at FROM item_costs WHERE catalog_item_id = ? ORDER BY effective_at`, catalogItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ItemCost
	for rows.Next() {
		v, err := scanItemCostRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repository) scanItemCost(row *sql.Row) (*domain.ItemCost, error) {
	v, err := scanItemCostRows(row)
	if err != nil {
		return nil, normalizeErr(err)
	}
	return v, nil
}

func scanItemCostRows(row outboxScanner) (*domain.ItemCost, error) {
	var v domain.ItemCost
	var typ, effective, created string
	var sourceType, sourceID sql.NullString
	if err := row.Scan(&v.ID, &v.CatalogItemID, &typ, &v.Amount, &v.Currency, &sourceType, &sourceID, &effective, &created); err != nil {
		return nil, err
	}
	v.Type = domain.ItemCostType(typ)
	v.SourceType = stringPtr(sourceType)
	v.SourceID = stringPtr(sourceID)
	v.EffectiveAt = parseTime(effective)
	v.CreatedAt = parseTime(created)
	return &v, nil
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
