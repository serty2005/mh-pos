package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/domain/order"
)

type scanner interface {
	Scan(dest ...any) error
}

func (r *Repository) CreateOrder(ctx context.Context, v *domain.Order) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO orders(id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,closed_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EdgeOrderID, v.RestaurantID, v.DeviceID, v.ShiftID, string(v.Status), v.TableID, v.TableName, v.GuestCount, dbTime(v.OpenedAt), nil, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return r.scanOrder(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,closed_at,created_at,updated_at FROM orders WHERE id = ?`, id))
}

func (r *Repository) GetActiveOrderByDeviceAndTable(ctx context.Context, deviceID, tableID string) (*domain.Order, error) {
	return r.scanOrder(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,edge_order_id,restaurant_id,device_id,shift_id,status,table_id,table_name,guest_count,opened_at,closed_at,created_at,updated_at FROM orders WHERE device_id = ? AND table_id = ? AND status IN ('open','locked') ORDER BY opened_at DESC LIMIT 1`, deviceID, tableID))
}

func (r *Repository) ListActiveOrdersByRestaurantAndHall(ctx context.Context, restaurantID, hallID string) ([]domain.Order, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT o.id,o.edge_order_id,o.restaurant_id,o.device_id,o.shift_id,o.status,o.table_id,o.table_name,o.guest_count,o.opened_at,o.closed_at,o.created_at,o.updated_at FROM orders o JOIN tables t ON t.id = o.table_id WHERE o.restaurant_id = ? AND t.restaurant_id = ? AND t.hall_id = ? AND o.status IN ('open','locked') ORDER BY o.opened_at`, restaurantID, restaurantID, hallID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Order
	for rows.Next() {
		v, err := scanOrderRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) scanOrder(row scanner) (*domain.Order, error) {
	return scanOrderRows(row)
}

func scanOrderRows(row scanner) (*domain.Order, error) {
	var v domain.Order
	var status, opened, created, updated string
	var closed sql.NullString
	if err := row.Scan(&v.ID, &v.EdgeOrderID, &v.RestaurantID, &v.DeviceID, &v.ShiftID, &status, &v.TableID, &v.TableName, &v.GuestCount, &opened, &closed, &created, &updated); err != nil {
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
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO order_lines(id,order_id,menu_item_id,catalog_item_id,category_id,tag_id,name,quantity,unit_price,total_price,currency_code,tax_profile_id,course,comment,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderID, v.MenuItemID, v.CatalogItemID, nullableStringValue(v.CategoryID), nullableStringValue(v.TagID), v.Name, v.Quantity, v.UnitPrice, v.TotalPrice, v.CurrencyCode, nullableString(v.TaxProfileID), nullableString(v.Course), nullableString(v.Comment), string(v.Status), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) CreateOrderLineModifier(ctx context.Context, v *domain.LineModifier) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO order_line_modifiers(id,order_line_id,modifier_group_id,modifier_option_id,name,quantity,unit_price,total_price) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.OrderLineID, v.ModifierGroupID, v.ModifierOptionID, v.Name, v.Quantity, v.UnitPrice, v.TotalPrice)
	return normalizeErr(err)
}

// ReplaceOrderLineModifiers заменяет selected modifiers редактируемой строки внутри transaction boundary вызывающего сервиса.
func (r *Repository) ReplaceOrderLineModifiers(ctx context.Context, lineID string, modifiers []domain.LineModifier) error {
	if _, err := r.execer(ctx).ExecContext(ctx, `DELETE FROM order_line_modifiers WHERE order_line_id = ?`, lineID); err != nil {
		return normalizeErr(err)
	}
	for i := range modifiers {
		if err := r.CreateOrderLineModifier(ctx, &modifiers[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) GetOrderLine(ctx context.Context, id string) (*domain.OrderLine, error) {
	var v domain.OrderLine
	var status, created, updated string
	var taxProfileID sql.NullString
	var course, comment sql.NullString
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,order_id,menu_item_id,catalog_item_id,COALESCE(category_id,''),COALESCE(tag_id,''),name,quantity,unit_price,total_price,currency_code,tax_profile_id,course,comment,status,created_at,updated_at FROM order_lines WHERE id = ?`, id).
		Scan(&v.ID, &v.OrderID, &v.MenuItemID, &v.CatalogItemID, &v.CategoryID, &v.TagID, &v.Name, &v.Quantity, &v.UnitPrice, &v.TotalPrice, &v.CurrencyCode, &taxProfileID, &course, &comment, &status, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.OrderLineStatus(status)
	v.TaxProfileID = stringPtr(taxProfileID)
	v.Course = stringPtr(course)
	v.Comment = stringPtr(comment)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	modifiers, err := r.listOrderLineModifiers(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	v.Modifiers = modifiers
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

func (r *Repository) UpdateOrderLineDetails(ctx context.Context, v *domain.OrderLine) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE order_lines SET course = ?, comment = ?, updated_at = ? WHERE id = ?`,
		nullableString(v.Course), nullableString(v.Comment), dbTime(v.UpdatedAt), v.ID)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListClosedOrders(ctx context.Context, query order.ClosedOrderListQuery) ([]order.OrderSummary, error) {
	where := []string{"o.status = 'closed'"}
	args := make([]any, 0, 10)
	if strings.TrimSpace(query.RestaurantID) != "" {
		where = append(where, "o.restaurant_id = ?")
		args = append(args, strings.TrimSpace(query.RestaurantID))
	}
	if strings.TrimSpace(query.BusinessDateLocal) != "" {
		where = append(where, "c.business_date_local = ?")
		args = append(args, strings.TrimSpace(query.BusinessDateLocal))
	}
	if strings.TrimSpace(query.FromBusinessDateLocal) != "" {
		where = append(where, "c.business_date_local >= ?")
		args = append(args, strings.TrimSpace(query.FromBusinessDateLocal))
	}
	if strings.TrimSpace(query.ToBusinessDateLocal) != "" {
		where = append(where, "c.business_date_local <= ?")
		args = append(args, strings.TrimSpace(query.ToBusinessDateLocal))
	}
	if strings.TrimSpace(query.ShiftID) != "" {
		where = append(where, "o.shift_id = ?")
		args = append(args, strings.TrimSpace(query.ShiftID))
	}
	if strings.TrimSpace(query.DeviceID) != "" {
		where = append(where, "o.device_id = ?")
		args = append(args, strings.TrimSpace(query.DeviceID))
	}
	if strings.TrimSpace(query.CheckID) != "" {
		where = append(where, "c.id = ?")
		args = append(args, strings.TrimSpace(query.CheckID))
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	sqlText := `SELECT o.id, o.table_name, o.opened_at, o.closed_at, c.id, c.status, c.currency_code, c.subtotal, c.discount_total, c.surcharge_total, c.tax_total, c.total, c.paid_total, c.remaining_total, c.business_date_local, c.closed_at, c.created_at, c.updated_at, (SELECT pr.id FROM prechecks pr WHERE pr.order_id = o.id ORDER BY pr.version DESC, pr.created_at DESC LIMIT 1), o.status FROM orders o LEFT JOIN checks c ON o.id = c.order_id WHERE ` + strings.Join(where, " AND ") + ` ORDER BY COALESCE(c.closed_at, o.closed_at) DESC, o.id DESC LIMIT ? OFFSET ?`
	rows, err := r.queryer(ctx).QueryContext(ctx, sqlText, args...)
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
		var currencyCode sql.NullString
		var subtotal, discountTotal, surchargeTotal, taxTotal, total, paidTotal, remainingTotal sql.NullInt64
		var businessDateLocal, checkClosedAt, checkCreatedAt, checkUpdatedAt sql.NullString
		var precheckID sql.NullString
		if err := rows.Scan(&v.ID, &v.TableName, &opened, &closed, &checkID, &checkStatus, &currencyCode, &subtotal, &discountTotal, &surchargeTotal, &taxTotal, &total, &paidTotal, &remainingTotal, &businessDateLocal, &checkClosedAt, &checkCreatedAt, &checkUpdatedAt, &precheckID, &status); err != nil {
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
				CurrencyCode:      currencyCode.String,
				Subtotal:          subtotal.Int64,
				DiscountTotal:     discountTotal.Int64,
				SurchargeTotal:    surchargeTotal.Int64,
				TaxTotal:          taxTotal.Int64,
				Total:             total.Int64,
				PaidTotal:         paidTotal.Int64,
				RemainingTotal:    remainingTotal.Int64,
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
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,order_id,menu_item_id,catalog_item_id,COALESCE(category_id,''),COALESCE(tag_id,''),name,quantity,unit_price,total_price,currency_code,tax_profile_id,course,comment,status,created_at,updated_at FROM order_lines WHERE order_id = ? ORDER BY created_at`, orderID)
	if err != nil {
		return nil, err
	}
	var out []domain.OrderLine
	for rows.Next() {
		var v domain.OrderLine
		var status, created, updated string
		var taxProfileID sql.NullString
		var course, comment sql.NullString
		if err := rows.Scan(&v.ID, &v.OrderID, &v.MenuItemID, &v.CatalogItemID, &v.CategoryID, &v.TagID, &v.Name, &v.Quantity, &v.UnitPrice, &v.TotalPrice, &v.CurrencyCode, &taxProfileID, &course, &comment, &status, &created, &updated); err != nil {
			return nil, err
		}
		v.Status = domain.OrderLineStatus(status)
		v.TaxProfileID = stringPtr(taxProfileID)
		v.Course = stringPtr(course)
		v.Comment = stringPtr(comment)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		modifiers, err := r.listOrderLineModifiers(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Modifiers = modifiers
	}
	return out, nil
}

func (r *Repository) listOrderLineModifiers(ctx context.Context, lineID string) ([]domain.LineModifier, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,order_line_id,modifier_group_id,modifier_option_id,name,quantity,unit_price,total_price FROM order_line_modifiers WHERE order_line_id = ? ORDER BY id`, lineID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []domain.LineModifier
	for rows.Next() {
		var v domain.LineModifier
		if err := rows.Scan(&v.ID, &v.OrderLineID, &v.ModifierGroupID, &v.ModifierOptionID, &v.Name, &v.Quantity, &v.UnitPrice, &v.TotalPrice); err != nil {
			return nil, normalizeErr(err)
		}
		out = append(out, v)
	}
	return out, normalizeErr(rows.Err())
}
