package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/olap/app"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetExportStatus(ctx context.Context, stream string, now time.Time) (app.ExportStatus, error) {
	status := app.ExportStatus{Stream: stream}
	checkpointID := "raw_business_events"
	if stream == "stock_moves" {
		checkpointID = "olap_stock_moves"
	}
	if err := r.loadCheckpoint(ctx, checkpointID, now, &status); err != nil {
		return app.ExportStatus{}, err
	}
	switch stream {
	case "raw_business_events":
		err := r.pool.QueryRow(ctx, `
SELECT
  COUNT(*) FILTER (WHERE processed_for_olap = false AND olap_export_status = 'pending'),
  COUNT(*) FILTER (WHERE processed_for_olap = false AND olap_export_status = 'processing'),
  COUNT(*) FILTER (WHERE processed_for_olap = false AND olap_export_status = 'failed')
FROM inbox_events`).Scan(&status.PendingCount, &status.ProcessingCount, &status.FailedCount)
		return status, err
	case "stock_moves":
		if err := r.pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM stock_ledger
WHERE ($1 = '' OR id > $1)`, status.LastCheckpoint).Scan(&status.PendingCount); err != nil {
			return app.ExportStatus{}, err
		}
		if status.LastError != "" {
			status.FailedCount = status.ConsecutiveFailures
		}
		return status, nil
	default:
		return app.ExportStatus{}, app.ErrOLAPUnavailable
	}
}

func (r *Repository) loadCheckpoint(ctx context.Context, checkpointID string, now time.Time, status *app.ExportStatus) error {
	var lastExportedAt *time.Time
	var nextRetryAt *time.Time
	var updatedAt *time.Time
	err := r.pool.QueryRow(ctx, `
SELECT COALESCE(last_exported_inbox_id, ''),
       COALESCE(last_exported_event_id, ''),
       last_exported_at,
       COALESCE(last_error, ''),
       consecutive_failures,
       next_retry_at,
       updated_at
FROM olap_export_checkpoints
WHERE id = $1`, checkpointID).Scan(
		&status.LastCheckpoint,
		&status.LastExportedID,
		&lastExportedAt,
		&status.LastError,
		&status.ConsecutiveFailures,
		&nextRetryAt,
		&updatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	status.LastExportedAt = lastExportedAt
	status.NextRetryAt = nextRetryAt
	status.CheckpointUpdatedAt = updatedAt
	if nextRetryAt != nil && nextRetryAt.After(now) {
		status.RetryBlocked = true
	}
	return nil
}

func (r *Repository) ClaimPending(ctx context.Context, cmd app.ClaimCommand) ([]app.InboxEvent, error) {
	if cmd.Limit <= 0 {
		cmd.Limit = 1000
	}
	rows, err := r.pool.Query(ctx, `
WITH picked AS (
  SELECT id
  FROM inbox_events
  WHERE processed_for_olap = false
    AND (
      olap_export_status IN ('pending','failed')
      OR (olap_export_status = 'processing' AND olap_locked_at < $4)
    )
    AND (olap_next_retry_at IS NULL OR olap_next_retry_at <= $3)
  ORDER BY cloud_received_at, id
  LIMIT $1
  FOR UPDATE SKIP LOCKED
)
UPDATE inbox_events e
SET olap_export_status = 'processing',
    olap_export_attempts = e.olap_export_attempts + 1,
    olap_locked_at = $3,
    olap_locked_by = $2,
    olap_last_error = NULL,
    updated_at = $3
FROM picked
WHERE e.id = picked.id
RETURNING e.id,e.receipt_id,e.tenant_id,e.restaurant_id,e.device_id,e.employee_id,
          e.command_id,e.event_id,e.edge_event_id,e.event_type,e.aggregate_type,e.aggregate_id,
          e.envelope_version,e.occurred_at,e.cloud_received_at,e.raw_payload,e.raw_payload_sha256_hex`,
		cmd.Limit,
		strings.TrimSpace(cmd.LockedBy),
		cmd.Now,
		cmd.StaleBefore,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]app.InboxEvent, 0, cmd.Limit)
	for rows.Next() {
		var event app.InboxEvent
		var raw []byte
		if err := rows.Scan(
			&event.ID,
			&event.ReceiptID,
			&event.TenantID,
			&event.RestaurantID,
			&event.DeviceID,
			&event.EmployeeID,
			&event.CommandID,
			&event.EventID,
			&event.EdgeEventID,
			&event.EventType,
			&event.AggregateType,
			&event.AggregateID,
			&event.EnvelopeVersion,
			&event.OccurredAt,
			&event.CloudReceivedAt,
			&raw,
			&event.RawPayloadSHA256Hex,
		); err != nil {
			return nil, err
		}
		event.RawPayload = json.RawMessage(raw)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *Repository) MarkProcessed(ctx context.Context, events []app.InboxEvent, now time.Time) error {
	if len(events) == 0 {
		return nil
	}
	ids := eventIDs(events)
	if _, err := r.pool.Exec(ctx, `
UPDATE inbox_events
SET processed_for_olap = true,
    olap_export_status = 'processed',
    olap_processed_at = $2,
    olap_locked_at = NULL,
    olap_locked_by = NULL,
    olap_next_retry_at = NULL,
    olap_last_error = NULL,
    updated_at = $2
WHERE id = ANY($1)`, ids, now); err != nil {
		return err
	}
	last := events[len(events)-1]
	_, err := r.pool.Exec(ctx, `
INSERT INTO olap_export_checkpoints(id,worker_id,last_exported_inbox_id,last_exported_event_id,last_exported_at,last_error,consecutive_failures,updated_at)
VALUES ('raw_business_events','', $1, $2, $3, '', 0, $3)
ON CONFLICT (id) DO UPDATE SET
  last_exported_inbox_id = EXCLUDED.last_exported_inbox_id,
  last_exported_event_id = EXCLUDED.last_exported_event_id,
  last_exported_at = EXCLUDED.last_exported_at,
  last_error = '',
  consecutive_failures = 0,
  updated_at = EXCLUDED.updated_at`, last.ID, last.EventID, now)
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, events []app.InboxEvent, reason string, nextRetry, now time.Time) error {
	if len(events) == 0 {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `
UPDATE inbox_events
SET olap_export_status = 'failed',
    olap_next_retry_at = $2,
    olap_locked_at = NULL,
    olap_locked_by = NULL,
    olap_last_error = $3,
    updated_at = $4
WHERE id = ANY($1)`, eventIDs(events), nextRetry, reason, now); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `
INSERT INTO olap_export_checkpoints(id,worker_id,last_error,consecutive_failures,updated_at)
VALUES ('raw_business_events','', $1, 1, $2)
ON CONFLICT (id) DO UPDATE SET
  last_error = EXCLUDED.last_error,
  consecutive_failures = olap_export_checkpoints.consecutive_failures + 1,
  updated_at = EXCLUDED.updated_at`, reason, now)
	return err
}

func (r *Repository) ClaimPendingStockMoves(ctx context.Context, cmd app.ClaimCommand) ([]app.StockMove, error) {
	if cmd.Limit <= 0 {
		cmd.Limit = 1000
	}
	var lastLedgerID string
	var retryBlocked bool
	err := r.pool.QueryRow(ctx, `
SELECT COALESCE(last_exported_inbox_id, ''), COALESCE(next_retry_at > $1, false)
FROM olap_export_checkpoints
WHERE id = 'olap_stock_moves'`, cmd.Now).Scan(&lastLedgerID, &retryBlocked)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if retryBlocked {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT id,restaurant_id,COALESCE(warehouse_id,''),stock_document_id,source_event_id,source_event_type,
       catalog_item_id,COALESCE(order_line_id,''),movement_type,quantity::text,unit_code,
       unit_cost_minor,total_cost_minor,costing_status,occurred_at,business_date_local::text,created_at
FROM stock_ledger
WHERE ($1 = '' OR id > $1)
ORDER BY id
LIMIT $2`, lastLedgerID, cmd.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	moves := make([]app.StockMove, 0, cmd.Limit)
	for rows.Next() {
		var move app.StockMove
		if err := rows.Scan(
			&move.LedgerEntryID,
			&move.RestaurantID,
			&move.WarehouseID,
			&move.StockDocumentID,
			&move.SourceEventID,
			&move.SourceEventType,
			&move.CatalogItemID,
			&move.OrderLineID,
			&move.MovementType,
			&move.Quantity,
			&move.UnitCode,
			&move.UnitCostMinor,
			&move.TotalCostMinor,
			&move.CostingStatus,
			&move.OccurredAt,
			&move.BusinessDateLocal,
			&move.LedgerCreatedAt,
		); err != nil {
			return nil, err
		}
		moves = append(moves, move)
	}
	return moves, rows.Err()
}

func (r *Repository) MarkStockMovesProcessed(ctx context.Context, moves []app.StockMove, now time.Time) error {
	if len(moves) == 0 {
		return nil
	}
	last := moves[len(moves)-1]
	_, err := r.pool.Exec(ctx, `
INSERT INTO olap_export_checkpoints(id,worker_id,last_exported_inbox_id,last_exported_event_id,last_exported_at,last_error,consecutive_failures,next_retry_at,updated_at)
VALUES ('olap_stock_moves','', $1, $2, $3, '', 0, NULL, $3)
ON CONFLICT (id) DO UPDATE SET
  last_exported_inbox_id = EXCLUDED.last_exported_inbox_id,
  last_exported_event_id = EXCLUDED.last_exported_event_id,
  last_exported_at = EXCLUDED.last_exported_at,
  last_error = '',
  consecutive_failures = 0,
  next_retry_at = NULL,
  updated_at = EXCLUDED.updated_at`, last.LedgerEntryID, last.SourceEventID, now)
	return err
}

func (r *Repository) MarkStockMovesFailed(ctx context.Context, _ []app.StockMove, reason string, nextRetry, now time.Time) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO olap_export_checkpoints(id,worker_id,last_error,consecutive_failures,next_retry_at,updated_at)
VALUES ('olap_stock_moves','', $1, 1, $2, $3)
ON CONFLICT (id) DO UPDATE SET
  last_error = EXCLUDED.last_error,
  consecutive_failures = olap_export_checkpoints.consecutive_failures + 1,
	next_retry_at = EXCLUDED.next_retry_at,
	updated_at = EXCLUDED.updated_at`, reason, nextRetry, now)
	return err
}

func (r *Repository) RequestExportRetry(ctx context.Context, cmd app.ExportRetryCommand, now time.Time) (app.ExportRetryResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return app.ExportRetryResult{}, err
	}
	defer tx.Rollback(ctx)

	existing, err := scanExportRetryResult(tx.QueryRow(ctx, `
SELECT command_id,stream,mode,reason,accepted,checkpoint_before,retry_requested_at,pending_count,failed_count
FROM olap_export_retry_commands
WHERE command_id = $1
FOR UPDATE`, cmd.CommandID))
	if err == nil {
		if existing.Stream != cmd.Stream || existing.Mode != cmd.Mode || existing.reason != cmd.Reason {
			return app.ExportRetryResult{}, contracts.ErrPayloadConflict
		}
		existing.AlreadyProcessed = true
		if err := tx.Commit(ctx); err != nil {
			return app.ExportRetryResult{}, err
		}
		return existing.ExportRetryResult, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return app.ExportRetryResult{}, err
	}

	checkpointID := checkpointIDForStream(cmd.Stream)
	checkpointBefore, err := lockCheckpoint(ctx, tx, checkpointID)
	if err != nil {
		return app.ExportRetryResult{}, err
	}
	if err := applyExportRetry(ctx, tx, cmd, checkpointID, now); err != nil {
		return app.ExportRetryResult{}, err
	}
	pendingCount, failedCount, err := exportCounters(ctx, tx, cmd.Stream, checkpointBefore)
	if err != nil {
		return app.ExportRetryResult{}, err
	}
	result := app.ExportRetryResult{
		CommandID:        cmd.CommandID,
		Stream:           cmd.Stream,
		Mode:             cmd.Mode,
		Accepted:         true,
		CheckpointBefore: checkpointBefore,
		RetryRequestedAt: now,
		PendingCount:     pendingCount,
		FailedCount:      failedCount,
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO olap_export_retry_commands(
  command_id,stream,mode,reason,accepted,checkpoint_before,retry_requested_at,pending_count,failed_count,created_at
) VALUES ($1,$2,$3,$4,true,$5,$6,$7,$8,$6)`,
		cmd.CommandID, cmd.Stream, cmd.Mode, cmd.Reason, checkpointBefore, now, pendingCount, failedCount); err != nil {
		return app.ExportRetryResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return app.ExportRetryResult{}, err
	}
	return result, nil
}

func (r *Repository) ListBackfillJobs(ctx context.Context, filter app.BackfillJobFilter) ([]app.BackfillJob, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id,command_id,stream,status,requested_from,requested_to,checkpoint_cursor,batch_size,
       total_rows,processed_rows,last_error,cancel_requested,reason,requested_by,
       created_at,started_at,completed_at,updated_at
FROM olap_backfill_jobs
WHERE ($1 = '' OR stream = $1)
  AND ($2 = '' OR status = $2)
ORDER BY created_at DESC, id DESC
LIMIT $3 OFFSET $4`, filter.Stream, filter.Status, filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]app.BackfillJob, 0, filter.Limit)
	for rows.Next() {
		job, err := scanBackfillJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (r *Repository) GetBackfillJob(ctx context.Context, id string) (app.BackfillJob, error) {
	job, err := scanBackfillJob(r.pool.QueryRow(ctx, `
SELECT id,command_id,stream,status,requested_from,requested_to,checkpoint_cursor,batch_size,
       total_rows,processed_rows,last_error,cancel_requested,reason,requested_by,
       created_at,started_at,completed_at,updated_at
FROM olap_backfill_jobs
WHERE id = $1`, strings.TrimSpace(id)))
	if errors.Is(err, pgx.ErrNoRows) {
		return app.BackfillJob{}, contracts.ErrNotFound
	}
	return job, err
}

func (r *Repository) CreateBackfillJob(ctx context.Context, cmd app.BackfillCreateCommand, now time.Time) (app.BackfillJob, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return app.BackfillJob{}, err
	}
	defer tx.Rollback(ctx)

	existing, err := scanBackfillJob(tx.QueryRow(ctx, `
SELECT id,command_id,stream,status,requested_from,requested_to,checkpoint_cursor,batch_size,
       total_rows,processed_rows,last_error,cancel_requested,reason,requested_by,
       created_at,started_at,completed_at,updated_at
FROM olap_backfill_jobs
WHERE command_id = $1
FOR UPDATE`, cmd.CommandID))
	if err == nil {
		if existing.Stream != cmd.Stream || existing.Reason != cmd.Reason {
			return app.BackfillJob{}, contracts.ErrPayloadConflict
		}
		existing.AlreadyProcessed = true
		if err := tx.Commit(ctx); err != nil {
			return app.BackfillJob{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return app.BackfillJob{}, err
	}

	totalRows, err := countBackfillRows(ctx, tx, cmd.Stream, cmd.RequestedFrom, cmd.RequestedTo)
	if err != nil {
		return app.BackfillJob{}, err
	}
	jobID := cmd.CommandID
	job, err := scanBackfillJob(tx.QueryRow(ctx, `
INSERT INTO olap_backfill_jobs(
  id,command_id,stream,status,requested_from,requested_to,batch_size,total_rows,
  processed_rows,reason,requested_by,created_at,updated_at
) VALUES ($1,$2,$3,'queued',$4,$5,$6,$7,0,$8,$9,$10,$10)
RETURNING id,command_id,stream,status,requested_from,requested_to,checkpoint_cursor,batch_size,
          total_rows,processed_rows,last_error,cancel_requested,reason,requested_by,
          created_at,started_at,completed_at,updated_at`,
		jobID, cmd.CommandID, cmd.Stream, cmd.RequestedFrom, cmd.RequestedTo, cmd.BatchSize, totalRows, cmd.Reason, cmd.RequestedBy, now))
	if err != nil {
		return app.BackfillJob{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO olap_operator_audit_events(id,command_id,action,stream,job_id,requested_by,reason,created_at)
VALUES ($1,$2,'create_backfill_job',$3,$4,$5,$6,$7)`,
		cmd.CommandID+":create", cmd.CommandID, cmd.Stream, job.ID, cmd.RequestedBy, cmd.Reason, now); err != nil {
		return app.BackfillJob{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return app.BackfillJob{}, err
	}
	return job, nil
}

func (r *Repository) CancelBackfillJob(ctx context.Context, cmd app.BackfillCancelCommand, now time.Time) (app.BackfillJob, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return app.BackfillJob{}, err
	}
	defer tx.Rollback(ctx)

	job, err := scanBackfillJob(tx.QueryRow(ctx, `
SELECT id,command_id,stream,status,requested_from,requested_to,checkpoint_cursor,batch_size,
       total_rows,processed_rows,last_error,cancel_requested,reason,requested_by,
       created_at,started_at,completed_at,updated_at
FROM olap_backfill_jobs
WHERE id = $1
FOR UPDATE`, cmd.JobID))
	if errors.Is(err, pgx.ErrNoRows) {
		return app.BackfillJob{}, contracts.ErrNotFound
	}
	if err != nil {
		return app.BackfillJob{}, err
	}
	if job.Status == "queued" || job.Status == "running" {
		job, err = scanBackfillJob(tx.QueryRow(ctx, `
UPDATE olap_backfill_jobs
SET status = 'cancelled',
    cancel_requested = true,
    completed_at = COALESCE(completed_at, $2),
    updated_at = $2
WHERE id = $1
RETURNING id,command_id,stream,status,requested_from,requested_to,checkpoint_cursor,batch_size,
          total_rows,processed_rows,last_error,cancel_requested,reason,requested_by,
          created_at,started_at,completed_at,updated_at`, cmd.JobID, now))
		if err != nil {
			return app.BackfillJob{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO olap_operator_audit_events(id,command_id,action,stream,job_id,requested_by,reason,created_at)
VALUES ($1,$2,'cancel_backfill_job',$3,$4,$5,$6,$7)
ON CONFLICT (id) DO NOTHING`,
		cmd.CommandID+":cancel", cmd.CommandID, job.Stream, job.ID, cmd.RequestedBy, cmd.Reason, now); err != nil {
		return app.BackfillJob{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return app.BackfillJob{}, err
	}
	return job, nil
}

func (r *Repository) ClaimBackfillJob(ctx context.Context, workerID string, now time.Time) (app.BackfillJob, bool, error) {
	job, err := scanBackfillJob(r.pool.QueryRow(ctx, `
WITH picked AS (
  SELECT id
  FROM olap_backfill_jobs
  WHERE status IN ('queued','running')
    AND cancel_requested = false
  ORDER BY created_at, id
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
UPDATE olap_backfill_jobs j
SET status = 'running',
    started_at = COALESCE(j.started_at, $2),
    locked_by = $1,
    locked_at = $2,
    updated_at = $2
FROM picked
WHERE j.id = picked.id
RETURNING j.id,j.command_id,j.stream,j.status,j.requested_from,j.requested_to,j.checkpoint_cursor,j.batch_size,
          j.total_rows,j.processed_rows,j.last_error,j.cancel_requested,j.reason,j.requested_by,
          j.created_at,j.started_at,j.completed_at,j.updated_at`, strings.TrimSpace(workerID), now))
	if errors.Is(err, pgx.ErrNoRows) {
		return app.BackfillJob{}, false, nil
	}
	return job, err == nil, err
}

func (r *Repository) LoadBackfillBatch(ctx context.Context, job app.BackfillJob, limit int) (app.BackfillBatch, error) {
	if limit <= 0 {
		limit = 1000
	}
	switch job.Stream {
	case "raw_business_events":
		events, err := r.loadRawBackfillBatch(ctx, job, limit)
		return app.BackfillBatch{RawEvents: events}, err
	case "stock_moves":
		moves, err := r.loadStockMoveBackfillBatch(ctx, job, limit)
		return app.BackfillBatch{StockMoves: moves}, err
	default:
		return app.BackfillBatch{}, app.ErrOLAPUnavailable
	}
}

func (r *Repository) MarkBackfillProgress(ctx context.Context, job app.BackfillJob, batch app.BackfillBatch, now time.Time) error {
	delta := len(batch.RawEvents) + len(batch.StockMoves)
	cursor := job.CheckpointCursor
	if len(batch.RawEvents) > 0 {
		cursor = batch.RawEvents[len(batch.RawEvents)-1].ID
	}
	if len(batch.StockMoves) > 0 {
		cursor = batch.StockMoves[len(batch.StockMoves)-1].LedgerEntryID
	}
	status := "running"
	var completedAt *time.Time
	if delta == 0 {
		status = "completed"
		completedAt = &now
	}
	_, err := r.pool.Exec(ctx, `
UPDATE olap_backfill_jobs
SET status = $2,
    checkpoint_cursor = $3,
    processed_rows = processed_rows + $4,
    last_error = '',
    locked_by = '',
    locked_at = NULL,
    completed_at = COALESCE(completed_at, $5),
    updated_at = $6
WHERE id = $1`, job.ID, status, cursor, delta, completedAt, now)
	return err
}

func (r *Repository) MarkBackfillFailed(ctx context.Context, job app.BackfillJob, reason string, now time.Time) error {
	_, err := r.pool.Exec(ctx, `
UPDATE olap_backfill_jobs
SET status = 'failed',
    last_error = $2,
    locked_by = '',
    locked_at = NULL,
    updated_at = $3
WHERE id = $1`, job.ID, reason, now)
	return err
}

func scanBackfillJob(row pgx.Row) (app.BackfillJob, error) {
	var job app.BackfillJob
	err := row.Scan(
		&job.ID,
		&job.CommandID,
		&job.Stream,
		&job.Status,
		&job.RequestedFrom,
		&job.RequestedTo,
		&job.CheckpointCursor,
		&job.BatchSize,
		&job.TotalRows,
		&job.ProcessedRows,
		&job.LastError,
		&job.CancelRequested,
		&job.Reason,
		&job.RequestedBy,
		&job.CreatedAt,
		&job.StartedAt,
		&job.CompletedAt,
		&job.UpdatedAt,
	)
	return job, err
}

func (r *Repository) loadRawBackfillBatch(ctx context.Context, job app.BackfillJob, limit int) ([]app.InboxEvent, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id,receipt_id,tenant_id,restaurant_id,device_id,employee_id,
       command_id,event_id,edge_event_id,event_type,aggregate_type,aggregate_id,
       envelope_version,occurred_at,cloud_received_at,raw_payload,raw_payload_sha256_hex
FROM inbox_events
WHERE ($1 = '' OR id > $1)
  AND ($2::timestamptz IS NULL OR occurred_at >= $2)
  AND ($3::timestamptz IS NULL OR occurred_at <= $3)
ORDER BY id
LIMIT $4`, job.CheckpointCursor, job.RequestedFrom, job.RequestedTo, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]app.InboxEvent, 0, limit)
	for rows.Next() {
		var event app.InboxEvent
		var raw []byte
		if err := rows.Scan(
			&event.ID,
			&event.ReceiptID,
			&event.TenantID,
			&event.RestaurantID,
			&event.DeviceID,
			&event.EmployeeID,
			&event.CommandID,
			&event.EventID,
			&event.EdgeEventID,
			&event.EventType,
			&event.AggregateType,
			&event.AggregateID,
			&event.EnvelopeVersion,
			&event.OccurredAt,
			&event.CloudReceivedAt,
			&raw,
			&event.RawPayloadSHA256Hex,
		); err != nil {
			return nil, err
		}
		event.RawPayload = json.RawMessage(raw)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *Repository) loadStockMoveBackfillBatch(ctx context.Context, job app.BackfillJob, limit int) ([]app.StockMove, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id,restaurant_id,COALESCE(warehouse_id,''),stock_document_id,source_event_id,source_event_type,
       catalog_item_id,COALESCE(order_line_id,''),movement_type,quantity::text,unit_code,
       unit_cost_minor,total_cost_minor,costing_status,occurred_at,business_date_local::text,created_at
FROM stock_ledger
WHERE ($1 = '' OR id > $1)
  AND ($2::timestamptz IS NULL OR occurred_at >= $2)
  AND ($3::timestamptz IS NULL OR occurred_at <= $3)
ORDER BY id
LIMIT $4`, job.CheckpointCursor, job.RequestedFrom, job.RequestedTo, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	moves := make([]app.StockMove, 0, limit)
	for rows.Next() {
		var move app.StockMove
		if err := rows.Scan(
			&move.LedgerEntryID,
			&move.RestaurantID,
			&move.WarehouseID,
			&move.StockDocumentID,
			&move.SourceEventID,
			&move.SourceEventType,
			&move.CatalogItemID,
			&move.OrderLineID,
			&move.MovementType,
			&move.Quantity,
			&move.UnitCode,
			&move.UnitCostMinor,
			&move.TotalCostMinor,
			&move.CostingStatus,
			&move.OccurredAt,
			&move.BusinessDateLocal,
			&move.LedgerCreatedAt,
		); err != nil {
			return nil, err
		}
		moves = append(moves, move)
	}
	return moves, rows.Err()
}

func countBackfillRows(ctx context.Context, tx pgx.Tx, stream string, from, to *time.Time) (int64, error) {
	var count int64
	switch stream {
	case "raw_business_events":
		err := tx.QueryRow(ctx, `
SELECT COUNT(*)
FROM inbox_events
WHERE ($1::timestamptz IS NULL OR occurred_at >= $1)
  AND ($2::timestamptz IS NULL OR occurred_at <= $2)`, from, to).Scan(&count)
		return count, err
	case "stock_moves":
		err := tx.QueryRow(ctx, `
SELECT COUNT(*)
FROM stock_ledger
WHERE ($1::timestamptz IS NULL OR occurred_at >= $1)
  AND ($2::timestamptz IS NULL OR occurred_at <= $2)`, from, to).Scan(&count)
		return count, err
	default:
		return 0, app.ErrOLAPUnavailable
	}
}

type storedExportRetryResult struct {
	app.ExportRetryResult
	reason string
}

func scanExportRetryResult(row pgx.Row) (storedExportRetryResult, error) {
	var out storedExportRetryResult
	err := row.Scan(
		&out.CommandID,
		&out.Stream,
		&out.Mode,
		&out.reason,
		&out.Accepted,
		&out.CheckpointBefore,
		&out.RetryRequestedAt,
		&out.PendingCount,
		&out.FailedCount,
	)
	return out, err
}

func checkpointIDForStream(stream string) string {
	if stream == "stock_moves" {
		return "olap_stock_moves"
	}
	return "raw_business_events"
}

func lockCheckpoint(ctx context.Context, tx pgx.Tx, checkpointID string) (string, error) {
	var checkpoint string
	err := tx.QueryRow(ctx, `
SELECT COALESCE(last_exported_inbox_id, '')
FROM olap_export_checkpoints
WHERE id = $1
FOR UPDATE`, checkpointID).Scan(&checkpoint)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return checkpoint, err
}

func applyExportRetry(ctx context.Context, tx pgx.Tx, cmd app.ExportRetryCommand, checkpointID string, now time.Time) error {
	if cmd.Stream == "raw_business_events" {
		statuses := []string{"failed"}
		if cmd.Mode == "resume_from_checkpoint" {
			statuses = []string{"failed", "processing"}
		}
		if _, err := tx.Exec(ctx, `
UPDATE inbox_events
SET olap_export_status = 'pending',
    olap_next_retry_at = NULL,
    olap_locked_at = NULL,
    olap_locked_by = NULL,
    olap_last_error = NULL,
    updated_at = $2
WHERE processed_for_olap = false
  AND olap_export_status = ANY($1)`, statuses, now); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `
INSERT INTO olap_export_checkpoints(id,worker_id,last_error,consecutive_failures,next_retry_at,updated_at)
VALUES ($1,'','',0,NULL,$2)
ON CONFLICT (id) DO UPDATE SET
  last_error = '',
  consecutive_failures = 0,
  next_retry_at = NULL,
  updated_at = EXCLUDED.updated_at`, checkpointID, now)
	return err
}

func exportCounters(ctx context.Context, tx pgx.Tx, stream, checkpointBefore string) (int64, int64, error) {
	var pendingCount, failedCount int64
	switch stream {
	case "raw_business_events":
		err := tx.QueryRow(ctx, `
SELECT
  COUNT(*) FILTER (WHERE processed_for_olap = false AND olap_export_status = 'pending'),
  COUNT(*) FILTER (WHERE processed_for_olap = false AND olap_export_status = 'failed')
FROM inbox_events`).Scan(&pendingCount, &failedCount)
		return pendingCount, failedCount, err
	case "stock_moves":
		err := tx.QueryRow(ctx, `
SELECT COUNT(*)
FROM stock_ledger
WHERE ($1 = '' OR id > $1)`, checkpointBefore).Scan(&pendingCount)
		return pendingCount, 0, err
	default:
		return 0, 0, app.ErrOLAPUnavailable
	}
}

func eventIDs(events []app.InboxEvent) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}
