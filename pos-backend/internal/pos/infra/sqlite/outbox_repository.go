package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
)

type outboxScanner interface {
	Scan(...any) error
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
	v, err := scanOutboxRows(row)
	if err != nil {
		return nil, normalizeErr(err)
	}
	return v, nil
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
