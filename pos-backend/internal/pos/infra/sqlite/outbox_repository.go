package sqlite

import (
	"context"
	"database/sql"
	"sort"

	"pos-backend/internal/pos/domain"
)

type outboxScanner interface {
	Scan(...any) error
}

func (r *Repository) CreateOutboxMessage(ctx context.Context, v *domain.OutboxMessage) error {
	sequenceNo := any(v.SequenceNo)
	if v.SequenceNo <= 0 {
		sequenceNo = nil
	}
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO pos_sync_outbox(id,command_id,sequence_no,origin,restaurant_id,device_id,actor_employee_id,session_id,aggregate_type,aggregate_id,command_type,payload_json,status,attempts,next_retry_at,locked_at,locked_by,sent_at,last_error,created_at,updated_at) VALUES (?,?,COALESCE(?,(SELECT COALESCE(MAX(sequence_no),0) + 1 FROM pos_sync_outbox)),?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.CommandID, sequenceNo, string(v.Origin), nullableString(v.RestaurantID), v.DeviceID, nullableString(v.ActorEmployeeID), nullableString(v.SessionID), v.AggregateType, v.AggregateID, v.CommandType, v.PayloadJSON, string(v.Status), v.Attempts, nullableTime(v.NextRetryAt), nullableTime(v.LockedAt), nullableString(v.LockedBy), nullableTime(v.SentAt), nullableString(v.LastError), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetOutboxByCommandID(ctx context.Context, commandID string) (*domain.OutboxMessage, error) {
	return r.scanOutbox(r.queryer(ctx).QueryRowContext(ctx, outboxSelectColumns+` FROM pos_sync_outbox WHERE command_id = ? ORDER BY sequence_no LIMIT 1`, commandID))
}

func (r *Repository) ListOutbox(ctx context.Context, limit int) ([]domain.OutboxMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.queryer(ctx).QueryContext(ctx, outboxSelectColumns+` FROM pos_sync_outbox ORDER BY sequence_no LIMIT ?`, limit)
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

func (r *Repository) GetSyncStatus(ctx context.Context) (domain.SyncStatus, error) {
	var status domain.SyncStatus
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT status, COUNT(1) FROM pos_sync_outbox GROUP BY status`)
	if err != nil {
		return status, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return status, err
		}
		status.Total += count
		switch domain.OutboxStatus(name) {
		case domain.OutboxPending:
			status.Pending = count
		case domain.OutboxProcessing:
			status.Processing = count
		case domain.OutboxSent:
			status.Sent = count
		case domain.OutboxFailed:
			status.Failed = count
		case domain.OutboxSuspended:
			status.Suspended = count
		}
	}
	if err := rows.Err(); err != nil {
		return status, err
	}
	var oldest sql.NullInt64
	if err := r.queryer(ctx).QueryRowContext(ctx, `SELECT MIN(sequence_no) FROM pos_sync_outbox WHERE status = 'pending'`).Scan(&oldest); err != nil {
		return status, normalizeErr(err)
	}
	status.OldestPendingSequenceNo = int64Ptr(oldest)
	return status, nil
}

func (r *Repository) RetryFailedOutbox(ctx context.Context, updatedAt string) (int, error) {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE pos_sync_outbox SET status = 'pending', attempts = 0, next_retry_at = NULL, locked_at = NULL, locked_by = NULL, last_error = NULL, updated_at = ? WHERE status IN ('failed','suspended')`, updatedAt)
	if err != nil {
		return 0, normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *Repository) ClaimPendingOutbox(ctx context.Context, limit int, lockedBy, now string) ([]domain.OutboxMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.queryer(ctx).QueryContext(ctx, `WITH candidates AS (
  SELECT id FROM pos_sync_outbox
  WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= ?)
  ORDER BY sequence_no
  LIMIT ?
)
UPDATE pos_sync_outbox
SET status = 'processing', locked_at = ?, locked_by = ?, updated_at = ?
WHERE id IN (SELECT id FROM candidates)
RETURNING id,command_id,sequence_no,origin,restaurant_id,device_id,actor_employee_id,session_id,aggregate_type,aggregate_id,command_type,payload_json,status,attempts,next_retry_at,locked_at,locked_by,sent_at,last_error,created_at,updated_at`, now, limit, now, lockedBy, now)
	if err != nil {
		return nil, normalizeErr(err)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].SequenceNo < out[j].SequenceNo
	})
	return out, nil
}

func (r *Repository) ReclaimStaleProcessingOutbox(ctx context.Context, staleBefore, updatedAt string) (int, error) {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE pos_sync_outbox SET status = 'pending', locked_at = NULL, locked_by = NULL, updated_at = ? WHERE status = 'processing' AND locked_at IS NOT NULL AND locked_at <= ?`, updatedAt, staleBefore)
	if err != nil {
		return 0, normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *Repository) scanOutbox(row *sql.Row) (*domain.OutboxMessage, error) {
	v, err := scanOutboxRows(row)
	if err != nil {
		return nil, normalizeErr(err)
	}
	return v, nil
}

const outboxSelectColumns = `SELECT id,command_id,sequence_no,origin,restaurant_id,device_id,actor_employee_id,session_id,aggregate_type,aggregate_id,command_type,payload_json,status,attempts,next_retry_at,locked_at,locked_by,sent_at,last_error,created_at,updated_at`

func scanOutboxRows(row outboxScanner) (*domain.OutboxMessage, error) {
	var v domain.OutboxMessage
	var restaurantID, actorEmployeeID, sessionID, nextRetryAt, lockedAt, lockedBy, sentAt, lastErr sql.NullString
	var status, created, updated string
	var origin string
	if err := row.Scan(&v.ID, &v.CommandID, &v.SequenceNo, &origin, &restaurantID, &v.DeviceID, &actorEmployeeID, &sessionID, &v.AggregateType, &v.AggregateID, &v.CommandType, &v.PayloadJSON, &status, &v.Attempts, &nextRetryAt, &lockedAt, &lockedBy, &sentAt, &lastErr, &created, &updated); err != nil {
		return nil, err
	}
	v.Origin = domain.CommandOrigin(origin)
	v.RestaurantID = stringPtr(restaurantID)
	v.ActorEmployeeID = stringPtr(actorEmployeeID)
	v.SessionID = stringPtr(sessionID)
	v.NextRetryAt = timePtr(nextRetryAt)
	v.LockedAt = timePtr(lockedAt)
	v.LockedBy = stringPtr(lockedBy)
	v.SentAt = timePtr(sentAt)
	v.LastError = stringPtr(lastErr)
	v.Status = domain.OutboxStatus(status)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) MarkOutboxSent(ctx context.Context, id, updatedAt string) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE pos_sync_outbox SET status = 'sent', locked_at = NULL, locked_by = NULL, next_retry_at = NULL, sent_at = ?, updated_at = ? WHERE id = ?`, updatedAt, updatedAt, id)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) MarkOutboxFailed(ctx context.Context, id, errorText string, nextRetryAt *string, updatedAt string, maxAttempts int) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE pos_sync_outbox SET status = CASE WHEN attempts + 1 > ? THEN 'suspended' ELSE 'failed' END, attempts = attempts + 1, next_retry_at = CASE WHEN attempts + 1 > ? THEN NULL ELSE ? END, locked_at = NULL, locked_by = NULL, last_error = ?, updated_at = ? WHERE id = ?`, maxAttempts, maxAttempts, nullableString(nextRetryAt), errorText, updatedAt, id)
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
