package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/domain/kitchen"
)

func (r *Repository) CreateKitchenTicket(ctx context.Context, v *kitchen.Ticket) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO kitchen_tickets(id,restaurant_id,device_id,shift_id,order_id,order_line_id,table_name,menu_item_id,catalog_item_id,name,quantity,unit_code,station_routing_key,course,comment,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceID, v.ShiftID, v.OrderID, v.OrderLineID, v.TableName, v.MenuItemID, v.CatalogItemID, v.Name, v.Quantity, v.UnitCode, v.StationRoutingKey, nullableString(v.Course), nullableString(v.Comment), string(v.Status), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetKitchenTicket(ctx context.Context, id string) (*kitchen.Ticket, error) {
	return scanKitchenTicket(r.queryer(ctx).QueryRowContext(ctx, kitchenTicketSelectSQL()+` WHERE kt.id = ?`, id))
}

func (r *Repository) ListKitchenTickets(ctx context.Context, query kitchen.TicketListQuery) ([]kitchen.Ticket, error) {
	where := []string{"kt.restaurant_id = ?"}
	args := []any{strings.TrimSpace(query.RestaurantID)}
	if query.Status != "" {
		where = append(where, "kt.status = ?")
		args = append(args, string(query.Status))
	}
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := r.queryer(ctx).QueryContext(ctx, kitchenTicketSelectSQL()+` WHERE `+strings.Join(where, " AND ")+` ORDER BY kt.created_at ASC, kt.id ASC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []kitchen.Ticket
	for rows.Next() {
		v, err := scanKitchenTicketRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) UpdateKitchenTicketStatus(ctx context.Context, id string, status kitchen.TicketStatus, updatedAt string) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE kitchen_tickets SET status = ?, updated_at = ? WHERE id = ?`, string(status), updatedAt, id)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateKitchenTicketEvent(ctx context.Context, v *kitchen.TicketEvent) error {
	actorID := strings.TrimSpace(v.ActorEmployeeID)
	var actorIDPtr *string
	if actorID != "" {
		actorIDPtr = &actorID
	}
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO kitchen_ticket_events(id,ticket_id,order_line_id,from_status,to_status,command_id,actor_employee_id,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.TicketID, v.OrderLineID, string(v.FromStatus), string(v.ToStatus), v.CommandID, nullableString(actorIDPtr), dbTime(v.OccurredAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func kitchenTicketSelectSQL() string {
	return `SELECT kt.id,kt.restaurant_id,kt.device_id,kt.shift_id,kt.order_id,kt.order_line_id,kt.table_name,kt.menu_item_id,kt.catalog_item_id,kt.name,kt.quantity,kt.unit_code,kt.station_routing_key,kt.course,kt.comment,kt.status,kt.created_at,kt.updated_at FROM kitchen_tickets kt`
}

func scanKitchenTicket(row scanner) (*kitchen.Ticket, error) {
	return scanKitchenTicketRows(row)
}

func scanKitchenTicketRows(row scanner) (*kitchen.Ticket, error) {
	var v kitchen.Ticket
	var status, created, updated string
	var course, comment sql.NullString
	if err := row.Scan(&v.ID, &v.RestaurantID, &v.DeviceID, &v.ShiftID, &v.OrderID, &v.OrderLineID, &v.TableName, &v.MenuItemID, &v.CatalogItemID, &v.Name, &v.Quantity, &v.UnitCode, &v.StationRoutingKey, &course, &comment, &status, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.Course = stringPtr(course)
	v.Comment = stringPtr(comment)
	v.Status = kitchen.TicketStatus(status)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}
