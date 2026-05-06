package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
)

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

func (r *Repository) UpdateOrderLocked(ctx context.Context, v *domain.Order) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE orders SET status = ?, updated_at = ? WHERE id = ?`, string(v.Status), dbTime(v.UpdatedAt), v.ID)
	return normalizeErr(err)
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
