package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-backend/internal/olap/app"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
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

func eventIDs(events []app.InboxEvent) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}
