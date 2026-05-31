package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
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
	if strings.TrimSpace(query.Station) != "" {
		where = append(where, "kt.station_routing_key = ?")
		args = append(args, strings.TrimSpace(query.Station))
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

func (r *Repository) ListKitchenOrderQueueTickets(ctx context.Context, query kitchen.OrderQueueQuery) ([]kitchen.OrderTicket, error) {
	where := []string{"kt.restaurant_id = ?"}
	args := []any{strings.TrimSpace(query.RestaurantID)}
	if strings.TrimSpace(query.Station) != "" {
		where = append(where, "kt.station_routing_key = ?")
		args = append(args, strings.TrimSpace(query.Station))
	}
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT kt.id,kt.restaurant_id,kt.device_id,kt.shift_id,kt.order_id,kt.order_line_id,kt.table_name,kt.menu_item_id,kt.catalog_item_id,kt.name,kt.quantity,kt.unit_code,kt.station_routing_key,kt.course,kt.comment,kt.status,kt.created_at,kt.updated_at,o.edge_order_id FROM kitchen_tickets kt JOIN orders o ON o.id = kt.order_id WHERE `+strings.Join(where, " AND ")+` ORDER BY kt.created_at ASC, kt.id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []kitchen.OrderTicket
	for rows.Next() {
		v, err := scanKitchenOrderTicketRows(rows)
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

func (r *Repository) UpdateKitchenTicketLineDetails(ctx context.Context, orderLineID string, course, comment *string, updatedAt string) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE kitchen_tickets SET course = ?, comment = ?, updated_at = ? WHERE order_line_id = ?`, nullableString(course), nullableString(comment), updatedAt, strings.TrimSpace(orderLineID))
	return normalizeErr(err)
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

func (r *Repository) GetKitchenTicketEventByCommandID(ctx context.Context, commandID string) (*kitchen.TicketEvent, error) {
	return scanKitchenTicketEvent(r.queryer(ctx).QueryRowContext(ctx, kitchenTicketEventSelectSQL()+` WHERE command_id = ? ORDER BY created_at LIMIT 1`, strings.TrimSpace(commandID)))
}

func (r *Repository) GetLatestKitchenServedEvent(ctx context.Context, ticketID string) (*kitchen.TicketEvent, error) {
	return scanKitchenTicketEvent(r.queryer(ctx).QueryRowContext(ctx, kitchenTicketEventSelectSQL()+` WHERE ticket_id = ? AND to_status = ? ORDER BY occurred_at DESC, created_at DESC, id DESC LIMIT 1`, strings.TrimSpace(ticketID), string(kitchen.TicketServed)))
}

func (r *Repository) CountKitchenServedEvents(ctx context.Context, ticketID string) (int, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM kitchen_ticket_events WHERE ticket_id = ? AND to_status = ?`, strings.TrimSpace(ticketID), string(kitchen.TicketServed)).Scan(&n)
	return n, normalizeErr(err)
}

func (r *Repository) CreateKitchenProposal(ctx context.Context, v *kitchen.Proposal) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO kitchen_proposals(id,restaurant_id,proposal_group_id,kind,status,action,owner_catalog_item_id,owner_catalog_suggestion_id,recipe_version_id,payload_json,outbox_command_id,outbox_event_type,created_by_employee_id,created_at,updated_at,cloud_version,cloud_updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.ProposalGroupID, string(v.Kind), string(v.Status), v.Action, v.OwnerCatalogItemID, v.OwnerCatalogSuggestionID, v.RecipeVersionID, string(v.Payload), v.OutboxCommandID, v.OutboxEventType, v.CreatedByEmployeeID, dbTime(v.CreatedAt), dbTime(v.UpdatedAt), nullableInt64(v.CloudVersion), nullableString(v.CloudUpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetKitchenProposalByCommandID(ctx context.Context, commandID string) (*kitchen.Proposal, error) {
	return scanKitchenProposal(r.queryer(ctx).QueryRowContext(ctx, kitchenProposalSelectSQL()+` WHERE outbox_command_id = ? ORDER BY created_at LIMIT 1`, strings.TrimSpace(commandID)))
}

func (r *Repository) ListKitchenProposals(ctx context.Context, query kitchen.ProposalListQuery) ([]kitchen.Proposal, error) {
	where := []string{"restaurant_id = ?"}
	args := []any{strings.TrimSpace(query.RestaurantID)}
	if query.Kind != "" {
		where = append(where, "kind = ?")
		args = append(args, string(query.Kind))
	}
	if query.Status != "" {
		where = append(where, "status = ?")
		args = append(args, string(query.Status))
	}
	if strings.TrimSpace(query.OwnerCatalogItemID) != "" {
		where = append(where, "owner_catalog_item_id = ?")
		args = append(args, strings.TrimSpace(query.OwnerCatalogItemID))
	}
	if strings.TrimSpace(query.RecipeVersionID) != "" {
		where = append(where, "recipe_version_id = ?")
		args = append(args, strings.TrimSpace(query.RecipeVersionID))
	}
	if strings.TrimSpace(query.OutboxEventType) != "" {
		where = append(where, "outbox_event_type = ?")
		args = append(args, strings.TrimSpace(query.OutboxEventType))
	}
	if strings.TrimSpace(query.OutboxCommandID) != "" {
		where = append(where, "outbox_command_id = ?")
		args = append(args, strings.TrimSpace(query.OutboxCommandID))
	}
	if !query.IncludeTerminal && query.Status == "" {
		where = append(where, "status IN ('draft','pending_sync','synced','changes_requested','failed')")
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
	rows, err := r.queryer(ctx).QueryContext(ctx, kitchenProposalSelectSQL()+` WHERE `+strings.Join(where, " AND ")+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []kitchen.Proposal
	for rows.Next() {
		v, err := scanKitchenProposalRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) ApplyKitchenProposalFeedback(ctx context.Context, kind kitchen.ProposalKind, suggestionID string, status kitchen.ProposalStatus, cloudVersion int64, cloudUpdatedAt, updatedAt string) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE kitchen_proposals SET status = ?, cloud_version = ?, cloud_updated_at = ?, updated_at = ? WHERE id = ? AND kind = ?`,
		string(status), cloudVersion, cloudUpdatedAt, updatedAt, strings.TrimSpace(suggestionID), string(kind))
	return normalizeErr(err)
}

func kitchenTicketSelectSQL() string {
	return `SELECT kt.id,kt.restaurant_id,kt.device_id,kt.shift_id,kt.order_id,kt.order_line_id,kt.table_name,kt.menu_item_id,kt.catalog_item_id,kt.name,kt.quantity,kt.unit_code,kt.station_routing_key,kt.course,kt.comment,kt.status,kt.created_at,kt.updated_at FROM kitchen_tickets kt`
}

func kitchenTicketEventSelectSQL() string {
	return `SELECT id,ticket_id,order_line_id,from_status,to_status,command_id,actor_employee_id,occurred_at,created_at FROM kitchen_ticket_events`
}

func kitchenProposalSelectSQL() string {
	return `SELECT id,restaurant_id,proposal_group_id,kind,status,action,owner_catalog_item_id,owner_catalog_suggestion_id,recipe_version_id,payload_json,outbox_command_id,outbox_event_type,created_by_employee_id,created_at,updated_at,cloud_version,cloud_updated_at FROM kitchen_proposals`
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

func scanKitchenOrderTicketRows(row scanner) (*kitchen.OrderTicket, error) {
	var v kitchen.OrderTicket
	var status, created, updated string
	var course, comment sql.NullString
	if err := row.Scan(&v.ID, &v.RestaurantID, &v.DeviceID, &v.ShiftID, &v.OrderID, &v.OrderLineID, &v.TableName, &v.MenuItemID, &v.CatalogItemID, &v.Name, &v.Quantity, &v.UnitCode, &v.StationRoutingKey, &course, &comment, &status, &created, &updated, &v.EdgeOrderID); err != nil {
		return nil, normalizeErr(err)
	}
	v.Course = stringPtr(course)
	v.Comment = stringPtr(comment)
	v.Status = kitchen.TicketStatus(status)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func scanKitchenTicketEvent(row scanner) (*kitchen.TicketEvent, error) {
	var v kitchen.TicketEvent
	var fromStatus, toStatus, occurred, created string
	var actorID sql.NullString
	if err := row.Scan(&v.ID, &v.TicketID, &v.OrderLineID, &fromStatus, &toStatus, &v.CommandID, &actorID, &occurred, &created); err != nil {
		return nil, normalizeErr(err)
	}
	v.FromStatus = kitchen.TicketStatus(fromStatus)
	v.ToStatus = kitchen.TicketStatus(toStatus)
	if actorID.Valid {
		v.ActorEmployeeID = actorID.String
	}
	v.OccurredAt = parseTime(occurred)
	v.CreatedAt = parseTime(created)
	return &v, nil
}

func scanKitchenProposal(row scanner) (*kitchen.Proposal, error) {
	return scanKitchenProposalRows(row)
}

func scanKitchenProposalRows(row scanner) (*kitchen.Proposal, error) {
	var v kitchen.Proposal
	var kind, status, payload, created, updated string
	var cloudVersion sql.NullInt64
	var cloudUpdatedAt sql.NullString
	if err := row.Scan(&v.ID, &v.RestaurantID, &v.ProposalGroupID, &kind, &status, &v.Action, &v.OwnerCatalogItemID, &v.OwnerCatalogSuggestionID, &v.RecipeVersionID, &payload, &v.OutboxCommandID, &v.OutboxEventType, &v.CreatedByEmployeeID, &created, &updated, &cloudVersion, &cloudUpdatedAt); err != nil {
		return nil, normalizeErr(err)
	}
	v.Kind = kitchen.ProposalKind(kind)
	v.Status = kitchen.ProposalStatus(status)
	if payload != "" {
		v.Payload = json.RawMessage(payload)
	}
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	v.CloudVersion = nullInt64Ptr(cloudVersion)
	v.CloudUpdatedAt = nullStringPtr(cloudUpdatedAt)
	return &v, nil
}
