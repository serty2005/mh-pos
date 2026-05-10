package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/domain/order"
)

func (r *Repository) CreateOrder(ctx context.Context, v *domain.Order) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,closed_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EdgeOrderID, v.RestaurantID, v.DeviceID, v.ShiftID, string(v.Status), v.TableID, v.TableName, v.GuestCount, dbTime(v.OpenedAt), nil, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	var v domain.Order
	var status, opened, created, updated string
	var closed sql.NullString
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,closed_at,created_at,updated_at FROM orders WHERE id = ?`, id).
		Scan(&v.ID, &v.EdgeOrderID, &v.RestaurantID, &v.DeviceID, &v.ShiftID, &status, &v.TableID, &v.TableName, &v.GuestCount, &opened, &closed, &created, &updated)
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

func (r *Repository) GetActiveOrderByDeviceAndTable(ctx context.Context, deviceID, tableID string) (*domain.Order, error) {
	var v domain.Order
	var status, opened, created, updated string
	var closed sql.NullString
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,closed_at,created_at,updated_at FROM orders WHERE device_id = ? AND table_id = ? AND status IN ('open','locked') ORDER BY opened_at DESC LIMIT 1`, deviceID, tableID).
		Scan(&v.ID, &v.EdgeOrderID, &v.RestaurantID, &v.DeviceID, &v.ShiftID, &status, &v.TableID, &v.TableName, &v.GuestCount, &opened, &closed, &created, &updated)
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

func (r *Repository) UpdateOrderOpen(ctx context.Context, v *domain.Order) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE orders SET status = ?, updated_at = ? WHERE id = ?`, string(v.Status), dbTime(v.UpdatedAt), v.ID)
	return normalizeErr(err)
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

func (r *Repository) GetOrderLine(ctx context.Context, id string) (*domain.OrderLine, error) {
	var v domain.OrderLine
	var status, created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,order_id,menu_item_id,catalog_item_id,name,quantity,unit_price,total_price,status,created_at,updated_at FROM order_lines WHERE id = ?`, id).
		Scan(&v.ID, &v.OrderID, &v.MenuItemID, &v.CatalogItemID, &v.Name, &v.Quantity, &v.UnitPrice, &v.TotalPrice, &status, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.OrderLineStatus(status)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) UpdateOrderLine(ctx context.Context, v *domain.OrderLine) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE order_lines SET quantity = ?, total_price = ?, status = ?, updated_at = ? WHERE id = ?`,
		v.Quantity, v.TotalPrice, string(v.Status), dbTime(v.UpdatedAt), v.ID)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListClosedOrders(ctx context.Context, limit int) ([]order.OrderSummary, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT o.id, o.table_name, o.opened_at, o.closed_at, c.id, c.status, c.subtotal, c.discount_total, c.tax_total, c.total, c.paid_total, c.business_date_local, c.closed_at, c.created_at, c.updated_at, (SELECT pr.id FROM prechecks pr WHERE pr.order_id = o.id ORDER BY pr.version DESC, pr.created_at DESC LIMIT 1), o.status FROM orders o LEFT JOIN checks c ON o.id = c.order_id WHERE o.status = 'closed' ORDER BY o.closed_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}

	type closedOrderRow struct {
		summary    order.OrderSummary
		precheckID string
	}

	var scanned []closedOrderRow
	for rows.Next() {
		var v order.OrderSummary
		var status, opened string
		var closed sql.NullString
		var checkID sql.NullString
		var checkStatus sql.NullString
		var subtotal, discountTotal, taxTotal, total, paidTotal sql.NullInt64
		var businessDateLocal, checkClosedAt, checkCreatedAt, checkUpdatedAt sql.NullString
		var precheckID sql.NullString
		if err := rows.Scan(&v.ID, &v.TableName, &opened, &closed, &checkID, &checkStatus, &subtotal, &discountTotal, &taxTotal, &total, &paidTotal, &businessDateLocal, &checkClosedAt, &checkCreatedAt, &checkUpdatedAt, &precheckID, &status); err != nil {
			return nil, err
		}
		v.Status = domain.OrderStatus(status)
		v.OpenedAt = parseTime(opened)
		if closed.Valid {
			t := parseTime(closed.String)
			v.ClosedAt = &t
		}
		if total.Valid {
			v.Total = total.Int64
		}
		if checkID.Valid {
			v.Check = &domain.Check{
				ID:                checkID.String,
				OrderID:           v.ID,
				Status:            domain.CheckStatus(checkStatus.String),
				Subtotal:          subtotal.Int64,
				DiscountTotal:     discountTotal.Int64,
				TaxTotal:          taxTotal.Int64,
				Total:             total.Int64,
				PaidTotal:         paidTotal.Int64,
				BusinessDateLocal: businessDateLocal.String,
				ClosedAt:          parseTime(checkClosedAt.String),
				CreatedAt:         parseTime(checkCreatedAt.String),
				UpdatedAt:         parseTime(checkUpdatedAt.String),
			}
		}
		row := closedOrderRow{summary: v}
		if precheckID.Valid {
			row.precheckID = precheckID.String
		}
		scanned = append(scanned, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	out := make([]order.OrderSummary, 0, len(scanned))
	for _, row := range scanned {
		v := row.summary
		if v.Check != nil && row.precheckID != "" {
			// Платежи нужны экрану закрытых заказов для операции возврата.
			payments, err := r.ListPaymentsByPrecheck(ctx, row.precheckID)
			if err != nil {
				return nil, err
			}
			v.Check.Payments = payments
		}
		out = append(out, v)
	}
	return out, nil
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
