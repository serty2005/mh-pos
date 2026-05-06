package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateLocalEvent(ctx context.Context, v *domain.LocalEvent) error {
	if v.NodeDeviceID == "" {
		v.NodeDeviceID = v.DeviceID
	}
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO local_event_log(id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,node_device_id,client_device_id,shift_id,actor_employee_id,session_id,payload_json,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.EventID, v.CommandID, v.EnvelopeVersion, v.EventType, v.AggregateType, v.AggregateID, nullableString(v.RestaurantID), v.DeviceID, v.NodeDeviceID, nullableString(v.ClientDeviceID), nullableString(v.ShiftID), nullableString(v.ActorEmployeeID), nullableString(v.SessionID), v.PayloadJSON, dbTime(v.OccurredAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListLocalEvents(ctx context.Context, limit int, eventType string) ([]domain.LocalEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	eventType = strings.TrimSpace(eventType)
	query := `SELECT id,event_id,command_id,envelope_version,event_type,aggregate_type,aggregate_id,restaurant_id,device_id,node_device_id,client_device_id,shift_id,actor_employee_id,session_id,payload_json,occurred_at,created_at FROM local_event_log`
	args := []any{}
	if eventType != "" {
		query += ` WHERE event_type = ?`
		args = append(args, eventType)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.queryer(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.LocalEvent
	for rows.Next() {
		v, err := scanLocalEventRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repository) CountLocalEvents(ctx context.Context) (int, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM local_event_log`).Scan(&n)
	return n, normalizeErr(err)
}

type localEventScanner interface {
	Scan(...any) error
}

func scanLocalEventRows(row localEventScanner) (*domain.LocalEvent, error) {
	var v domain.LocalEvent
	var restaurantID, clientDeviceID, shiftID, actorEmployeeID, sessionID sql.NullString
	var occurred, created string
	if err := row.Scan(&v.ID, &v.EventID, &v.CommandID, &v.EnvelopeVersion, &v.EventType, &v.AggregateType, &v.AggregateID, &restaurantID, &v.DeviceID, &v.NodeDeviceID, &clientDeviceID, &shiftID, &actorEmployeeID, &sessionID, &v.PayloadJSON, &occurred, &created); err != nil {
		return nil, err
	}
	v.RestaurantID = stringPtr(restaurantID)
	v.ClientDeviceID = stringPtr(clientDeviceID)
	v.ShiftID = stringPtr(shiftID)
	v.ActorEmployeeID = stringPtr(actorEmployeeID)
	v.SessionID = stringPtr(sessionID)
	v.OccurredAt = parseTime(occurred)
	v.CreatedAt = parseTime(created)
	return &v, nil
}
