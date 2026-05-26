package postgres

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/platform/idgen"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ReceiveEdgeEvent(ctx context.Context, receipt app.EdgeEventReceipt) (contracts.EventAck, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return contracts.EventAck{}, err
	}
	defer tx.Rollback(ctx)

	existing, err := scanAck(tx.QueryRow(ctx, `
SELECT id,idempotency_key,command_id,event_id,edge_event_id,envelope_version,cloud_received_at,raw_payload_sha256_hex
FROM cloud_edge_event_receipts
WHERE idempotency_key = $1
FOR UPDATE`, receipt.IdempotencyKey))
	if err == nil {
		if existing.RawPayloadSHA256Hex != receipt.RawPayloadSHA256 {
			return contracts.EventAck{}, contracts.ErrPayloadConflict
		}
		if err := enqueueExistingInboxEvent(ctx, tx, existing.CloudReceiptID); err != nil {
			return contracts.EventAck{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return contracts.EventAck{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return contracts.EventAck{}, err
	}

	receiptID, err := newID()
	if err != nil {
		return contracts.EventAck{}, err
	}
	ack := contracts.EventAck{
		Status:              "accepted",
		IdempotencyKey:      receipt.IdempotencyKey,
		CloudReceiptID:      receiptID,
		CommandID:           receipt.Envelope.CommandID,
		EventID:             receipt.Envelope.EventID,
		EdgeEventID:         contracts.EdgeEventID(receipt.Envelope),
		EnvelopeVersion:     receipt.Envelope.Version,
		CloudReceivedAt:     receipt.CloudReceivedAt,
		RawPayloadSHA256Hex: receipt.RawPayloadSHA256,
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO cloud_edge_event_receipts(
  id,idempotency_key,restaurant_id,device_id,command_id,event_id,edge_event_id,
  event_type,aggregate_type,aggregate_id,envelope_version,occurred_at,cloud_received_at,raw_payload_sha256_hex
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		ack.CloudReceiptID,
		ack.IdempotencyKey,
		*receipt.Envelope.RestaurantID,
		receipt.Envelope.DeviceID,
		receipt.Envelope.CommandID,
		receipt.Envelope.EventID,
		ack.EdgeEventID,
		string(receipt.Envelope.EventType),
		receipt.Envelope.AggregateType,
		receipt.Envelope.AggregateID,
		receipt.Envelope.Version,
		receipt.Envelope.OccurredAt,
		receipt.CloudReceivedAt,
		receipt.RawPayloadSHA256,
	); err != nil {
		return contracts.EventAck{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO cloud_edge_event_raw_payloads(receipt_id, raw_payload, raw_payload_sha256_hex, created_at)
VALUES ($1,$2::jsonb,$3,$4)`,
		ack.CloudReceiptID,
		string(receipt.RawPayload),
		receipt.RawPayloadSHA256,
		receipt.CloudReceivedAt,
	); err != nil {
		return contracts.EventAck{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO inbox_events(
  id,receipt_id,idempotency_key,tenant_id,restaurant_id,device_id,employee_id,
  command_id,event_id,edge_event_id,event_type,aggregate_type,aggregate_id,envelope_version,
  occurred_at,cloud_received_at,raw_payload,raw_payload_sha256_hex,processed_for_olap,olap_export_status,created_at,updated_at
) VALUES (
  $1,$1,$2,$3,$3,$4,$5,
  $6,$7,$8,$9,$10,$11,$12,
  $13,$14,$15::jsonb,$16,false,'pending',$14,$14
)
ON CONFLICT (id) DO NOTHING`,
		ack.CloudReceiptID,
		ack.IdempotencyKey,
		*receipt.Envelope.RestaurantID,
		receipt.Envelope.DeviceID,
		trimStringPtr(receipt.Envelope.ActorEmployeeID),
		receipt.Envelope.CommandID,
		receipt.Envelope.EventID,
		ack.EdgeEventID,
		string(receipt.Envelope.EventType),
		receipt.Envelope.AggregateType,
		receipt.Envelope.AggregateID,
		receipt.Envelope.Version,
		receipt.Envelope.OccurredAt,
		receipt.CloudReceivedAt,
		string(receipt.RawPayload),
		receipt.RawPayloadSHA256,
	); err != nil {
		return contracts.EventAck{}, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO cloud_operational_events(
  id,receipt_id,idempotency_key,restaurant_id,device_id,command_id,event_id,edge_event_id,
  event_type,aggregate_type,aggregate_id,envelope_version,occurred_at,cloud_received_at,raw_payload_sha256_hex
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		ack.CloudReceiptID,
		ack.CloudReceiptID,
		ack.IdempotencyKey,
		*receipt.Envelope.RestaurantID,
		receipt.Envelope.DeviceID,
		receipt.Envelope.CommandID,
		receipt.Envelope.EventID,
		ack.EdgeEventID,
		string(receipt.Envelope.EventType),
		receipt.Envelope.AggregateType,
		receipt.Envelope.AggregateID,
		receipt.Envelope.Version,
		receipt.Envelope.OccurredAt,
		receipt.CloudReceivedAt,
		receipt.RawPayloadSHA256,
	); err != nil {
		return contracts.EventAck{}, err
	}
	if err := r.applyEventProjections(ctx, tx, receipt, ack.CloudReceiptID); err != nil {
		return contracts.EventAck{}, err
	}
	if contracts.IsInventoryRelevantEventType(receipt.Envelope.EventType) {
		if err := enqueueInventoryEvent(ctx, tx, receipt, ack.CloudReceiptID); err != nil {
			return contracts.EventAck{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return contracts.EventAck{}, err
	}
	return ack, nil
}

func (r *Repository) RecordProblemEdgeEvent(ctx context.Context, item app.ProblemEdgeEvent) error {
	raw := strings.TrimSpace(string(item.RawPayload))
	if raw == "" {
		raw = "{}"
	}
	errorCode := strings.TrimSpace(item.ErrorCode)
	if errorCode == "" {
		errorCode = "UNKNOWN"
	}
	errorMessage := strings.TrimSpace(item.ErrorMessage)
	if errorMessage == "" {
		errorMessage = "sync problem item"
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(item.Direction),
		strings.TrimSpace(item.NodeDeviceID),
		strings.TrimSpace(item.RestaurantID),
		strings.TrimSpace(item.ClientItemID),
		item.RawPayloadSHA256,
		errorCode,
		createdAt.Format(time.RFC3339Nano),
	}, "|")))
	id := "sync-problem-" + hex.EncodeToString(sum[:])
	_, err := r.pool.Exec(ctx, `
INSERT INTO cloud_sync_problem_events(
  id,direction,node_device_id,restaurant_id,client_item_id,error_code,error_message,raw_payload,raw_payload_sha256_hex,created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id,
		strings.TrimSpace(item.Direction),
		nullableText(item.NodeDeviceID),
		nullableText(item.RestaurantID),
		nullableText(item.ClientItemID),
		errorCode,
		errorMessage,
		raw,
		item.RawPayloadSHA256,
		createdAt,
	)
	return err
}

// ListEdgeEvents возвращает безопасный журнал входящих Edge events без raw payload.
func (r *Repository) ListEdgeEvents(ctx context.Context, filter app.EdgeEventListFilter) ([]contracts.EdgeEventView, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
SELECT id,idempotency_key,restaurant_id,device_id,command_id,event_id,edge_event_id,
       event_type,aggregate_type,aggregate_id,envelope_version,occurred_at,cloud_received_at,raw_payload_sha256_hex
FROM cloud_edge_event_receipts
WHERE ($1 = '' OR restaurant_id = $1)
  AND ($2 = '' OR device_id = $2)
  AND ($3 = '' OR event_type = $3)
ORDER BY cloud_received_at DESC, id DESC
LIMIT $4`, filter.RestaurantID, filter.DeviceID, filter.EventType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]contracts.EdgeEventView, 0, limit)
	for rows.Next() {
		var view contracts.EdgeEventView
		if err := rows.Scan(
			&view.CloudReceiptID,
			&view.IdempotencyKey,
			&view.RestaurantID,
			&view.DeviceID,
			&view.CommandID,
			&view.EventID,
			&view.EdgeEventID,
			&view.EventType,
			&view.AggregateType,
			&view.AggregateID,
			&view.EnvelopeVersion,
			&view.OccurredAt,
			&view.CloudReceivedAt,
			&view.RawPayloadSHA256Hex,
		); err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	return out, rows.Err()
}

// ListFinancialOperations читает detailed financial operation projection без raw payload.
func (r *Repository) ListFinancialOperations(ctx context.Context, filter app.FinancialOperationProjectionFilter) ([]contracts.FinancialOperationProjection, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err := r.pool.Query(ctx, `
SELECT operation_id,edge_operation_id,event_id,receipt_id,restaurant_id,device_id,
       node_device_id,client_device_id,actor_employee_id,session_id,
       shift_id,original_shift_id,check_id,precheck_id,operation_type,operation_kind,
       amount,currency,business_date_local,inventory_disposition,reason,
       COALESCE(created_by_employee_id,''),approved_by_employee_id,snapshot_json,
       operation_created_at,occurred_at,cloud_received_at,raw_payload_sha256_hex
FROM cloud_projection_financial_operations
WHERE ($1 = '' OR restaurant_id = $1)
  AND ($2 = '' OR business_date_local >= $2)
  AND ($3 = '' OR business_date_local <= $3)
  AND ($4 = '' OR operation_type = $4)
  AND ($5 = '' OR shift_id = $5)
  AND ($6 = '' OR original_shift_id = $6)
  AND ($7 = '' OR check_id = $7)
ORDER BY operation_created_at DESC, operation_id DESC
LIMIT $8 OFFSET $9`,
		filter.RestaurantID,
		filter.BusinessDateFrom,
		filter.BusinessDateTo,
		filter.OperationType,
		filter.ShiftID,
		filter.OriginalShiftID,
		filter.CheckID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]contracts.FinancialOperationProjection, 0, limit)
	for rows.Next() {
		var v contracts.FinancialOperationProjection
		if err := rows.Scan(
			&v.OperationID,
			&v.EdgeOperationID,
			&v.EventID,
			&v.ReceiptID,
			&v.RestaurantID,
			&v.DeviceID,
			&v.NodeDeviceID,
			&v.ClientDeviceID,
			&v.ActorEmployeeID,
			&v.SessionID,
			&v.ShiftID,
			&v.OriginalShiftID,
			&v.CheckID,
			&v.PrecheckID,
			&v.OperationType,
			&v.OperationKind,
			&v.Amount,
			&v.Currency,
			&v.BusinessDateLocal,
			&v.InventoryDisposition,
			&v.Reason,
			&v.CreatedByEmployeeID,
			&v.ApprovedByEmployeeID,
			&v.Snapshot,
			&v.OperationCreatedAt,
			&v.OccurredAt,
			&v.CloudReceivedAt,
			&v.RawPayloadSHA256Hex,
		); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ListInventoryLedger читает Cloud-owned stock ledger без raw event payload.
func (r *Repository) ListInventoryLedger(ctx context.Context, filter app.InventoryLedgerFilter) ([]contracts.InventoryLedgerEntry, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err := r.pool.Query(ctx, `
SELECT id,restaurant_id,stock_document_id,source_event_id,source_event_type,catalog_item_id,
       COALESCE(order_line_id,''),movement_type,quantity::text,unit_code,unit_cost_minor,total_cost_minor,
       costing_status,occurred_at,business_date_local::text,created_at
FROM stock_ledger
WHERE ($1 = '' OR restaurant_id = $1)
  AND ($2 = '' OR source_event_type = $2)
  AND ($3 = '' OR source_event_id = $3)
  AND ($4 = '' OR order_line_id = $4)
  AND ($5 = '' OR catalog_item_id = $5)
ORDER BY occurred_at DESC, id DESC
LIMIT $6 OFFSET $7`,
		filter.RestaurantID,
		filter.SourceEventType,
		filter.SourceEventID,
		filter.OrderLineID,
		filter.CatalogItemID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]contracts.InventoryLedgerEntry, 0, limit)
	for rows.Next() {
		var v contracts.InventoryLedgerEntry
		if err := rows.Scan(
			&v.ID,
			&v.RestaurantID,
			&v.StockDocumentID,
			&v.SourceEventID,
			&v.SourceEventType,
			&v.CatalogItemID,
			&v.OrderLineID,
			&v.MovementType,
			&v.Quantity,
			&v.UnitCode,
			&v.UnitCostMinor,
			&v.TotalCostMinor,
			&v.CostingStatus,
			&v.OccurredAt,
			&v.BusinessDateLocal,
			&v.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertMasterDataPackage(ctx context.Context, v contracts.MasterDataPackage) (contracts.MasterDataPackage, error) {
	nodeDeviceID := strings.TrimSpace(v.NodeDeviceID)
	payload := bytesTrimSpace(v.PayloadJSON)
	var stored contracts.MasterDataPackage
	var cloudUpdatedAt *time.Time
	err := r.pool.QueryRow(ctx, `
INSERT INTO cloud_master_data_packages(
  stream_name,node_device_id,restaurant_id,sync_mode,full_snapshot_reason,cloud_version,checkpoint_token,cloud_updated_at,payload_json,created_at,updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11)
ON CONFLICT (stream_name,node_device_id) DO UPDATE SET
  restaurant_id = EXCLUDED.restaurant_id,
  sync_mode = EXCLUDED.sync_mode,
  full_snapshot_reason = EXCLUDED.full_snapshot_reason,
  cloud_version = EXCLUDED.cloud_version,
  checkpoint_token = EXCLUDED.checkpoint_token,
  cloud_updated_at = EXCLUDED.cloud_updated_at,
  payload_json = EXCLUDED.payload_json,
  updated_at = EXCLUDED.updated_at
RETURNING stream_name,node_device_id,restaurant_id,sync_mode,full_snapshot_reason,cloud_version,checkpoint_token,cloud_updated_at,payload_json,created_at,updated_at`,
		v.StreamName,
		nodeDeviceID,
		nullableText(v.RestaurantID),
		v.SyncMode,
		v.FullSnapshotReason,
		v.CloudVersion,
		nullableText(v.CheckpointToken),
		v.CloudUpdatedAt,
		string(payload),
		v.CreatedAt,
		v.UpdatedAt,
	).Scan(
		&stored.StreamName,
		&stored.NodeDeviceID,
		&stored.RestaurantID,
		&stored.SyncMode,
		&stored.FullSnapshotReason,
		&stored.CloudVersion,
		&stored.CheckpointToken,
		&cloudUpdatedAt,
		&stored.PayloadJSON,
		&stored.CreatedAt,
		&stored.UpdatedAt,
	)
	if err != nil {
		return contracts.MasterDataPackage{}, err
	}
	stored.CloudUpdatedAt = cloudUpdatedAt
	return stored, nil
}

func (r *Repository) GetMasterDataPackage(ctx context.Context, streamName, nodeDeviceID string) (contracts.MasterDataPackage, error) {
	streamName = strings.TrimSpace(streamName)
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	var out contracts.MasterDataPackage
	var cloudUpdatedAt *time.Time
	err := r.pool.QueryRow(ctx, `
SELECT stream_name,node_device_id,COALESCE(restaurant_id,''),sync_mode,full_snapshot_reason,cloud_version,COALESCE(checkpoint_token,''),cloud_updated_at,payload_json,created_at,updated_at
FROM cloud_master_data_packages
WHERE stream_name = $1 AND node_device_id IN ($2, '')
ORDER BY CASE WHEN node_device_id = $2 THEN 0 ELSE 1 END
LIMIT 1`, streamName, nodeDeviceID).Scan(
		&out.StreamName,
		&out.NodeDeviceID,
		&out.RestaurantID,
		&out.SyncMode,
		&out.FullSnapshotReason,
		&out.CloudVersion,
		&out.CheckpointToken,
		&cloudUpdatedAt,
		&out.PayloadJSON,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return contracts.MasterDataPackage{}, contracts.ErrNotFound
	}
	if err != nil {
		return contracts.MasterDataPackage{}, err
	}
	out.CloudUpdatedAt = cloudUpdatedAt
	return out, nil
}

func (r *Repository) AuthenticateNodeToken(ctx context.Context, nodeDeviceID, restaurantID, token string) error {
	var storedRestaurantID, status, credentialsHash string
	err := r.pool.QueryRow(ctx, `
SELECT COALESCE(restaurant_id,''), status, COALESCE(credentials_hash,'')
FROM cloud_edge_nodes
WHERE node_device_id = $1`, strings.TrimSpace(nodeDeviceID)).Scan(&storedRestaurantID, &status, &credentialsHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return contracts.ErrSyncUnauthorized
	}
	if err != nil {
		return err
	}
	if status != "assigned" || strings.TrimSpace(credentialsHash) == "" {
		return contracts.ErrSyncUnauthorized
	}
	if strings.TrimSpace(storedRestaurantID) != strings.TrimSpace(restaurantID) {
		return contracts.ErrSyncForbidden
	}
	if subtle.ConstantTimeCompare([]byte(credentialsHash), []byte(secretHash(token))) != 1 {
		return contracts.ErrSyncUnauthorized
	}
	return nil
}

func (r *Repository) applyEventProjections(ctx context.Context, tx pgx.Tx, receipt app.EdgeEventReceipt, receiptID string) error {
	if _, err := tx.Exec(ctx, `
INSERT INTO cloud_projection_event_type_stats(
  restaurant_id,device_id,event_type,event_count,first_occurred_at,last_occurred_at,last_cloud_received_at,last_event_id,last_command_id,updated_at
) VALUES ($1,$2,$3,1,$4,$4,$5,$6,$7,$5)
ON CONFLICT (restaurant_id,device_id,event_type) DO UPDATE SET
  event_count = cloud_projection_event_type_stats.event_count + 1,
  first_occurred_at = LEAST(cloud_projection_event_type_stats.first_occurred_at, EXCLUDED.first_occurred_at),
  last_occurred_at = GREATEST(cloud_projection_event_type_stats.last_occurred_at, EXCLUDED.last_occurred_at),
  last_cloud_received_at = EXCLUDED.last_cloud_received_at,
  last_event_id = EXCLUDED.last_event_id,
  last_command_id = EXCLUDED.last_command_id,
  updated_at = EXCLUDED.updated_at`,
		*receipt.Envelope.RestaurantID,
		receipt.Envelope.DeviceID,
		string(receipt.Envelope.EventType),
		receipt.Envelope.OccurredAt,
		receipt.CloudReceivedAt,
		receipt.Envelope.EventID,
		receipt.Envelope.CommandID,
	); err != nil {
		return err
	}
	if err := applyFinancialOperationProjection(ctx, tx, receipt, receiptID); err != nil {
		return err
	}
	if receipt.Envelope.ShiftID == nil || strings.TrimSpace(*receipt.Envelope.ShiftID) == "" {
		return nil
	}
	shiftID := strings.TrimSpace(*receipt.Envelope.ShiftID)
	switch receipt.Envelope.EventType {
	case contracts.EventPaymentCaptured:
		amount, err := paymentAmount(receipt.Envelope.Payload)
		if err != nil {
			return err
		}
		return upsertShiftFinanceProjection(ctx, tx, receipt, shiftID, 1, amount, 0, 0, 0, 0, 0, 0)
	case contracts.EventPaymentRefunded:
		amount, err := paymentRefundAmount(receipt.Envelope.Payload)
		if err != nil {
			return err
		}
		return upsertShiftFinanceProjection(ctx, tx, receipt, shiftID, 0, 0, 1, amount, 0, 0, 0, 0)
	case contracts.EventCheckCreated:
		total, err := checkTotal(receipt.Envelope.Payload)
		if err != nil {
			return err
		}
		return upsertShiftFinanceProjection(ctx, tx, receipt, shiftID, 0, 0, 0, 0, 1, total, 0, 0)
	case contracts.EventCheckRefunded:
		total, err := checkRefundedPaidTotal(receipt.Envelope.Payload)
		if err != nil {
			return err
		}
		return upsertShiftFinanceProjection(ctx, tx, receipt, shiftID, 0, 0, 0, 0, 0, 0, 1, total)
	case contracts.EventRefundRecorded:
		total, err := financialOperationAmount(receipt.Envelope.Payload)
		if err != nil {
			return err
		}
		return upsertShiftFinanceProjection(ctx, tx, receipt, shiftID, 0, 0, 0, 0, 0, 0, 1, total)
	default:
		return nil
	}
}

func applyFinancialOperationProjection(ctx context.Context, tx pgx.Tx, receipt app.EdgeEventReceipt, receiptID string) error {
	if receipt.Envelope.EventType != contracts.EventCancellationRecorded && receipt.Envelope.EventType != contracts.EventRefundRecorded {
		return nil
	}
	var payload contracts.Payload[contracts.FinancialOperationRecorded]
	if err := json.Unmarshal(receipt.Envelope.Payload, &payload); err != nil {
		return fmt.Errorf("%w: invalid financial operation payload", contracts.ErrInvalidEnvelope)
	}
	data := payload.Data
	_, err := tx.Exec(ctx, `
INSERT INTO cloud_projection_financial_operations(
  operation_id,edge_operation_id,event_id,receipt_id,restaurant_id,device_id,
  node_device_id,client_device_id,actor_employee_id,session_id,
  shift_id,original_shift_id,check_id,precheck_id,operation_type,operation_kind,
  amount,currency,business_date_local,inventory_disposition,reason,
  created_by_employee_id,approved_by_employee_id,snapshot_json,
  operation_created_at,occurred_at,cloud_received_at,raw_payload_sha256_hex,created_at
) VALUES (
  $1,$2,$3,$4,$5,$6,
  $7,$8,$9,$10,
  $11,$12,$13,$14,$15,$16,
  $17,$18,$19,$20,$21,
  $22,$23,$24::jsonb,
  $25,$26,$27,$28,$27
)
ON CONFLICT (operation_id) DO NOTHING`,
		strings.TrimSpace(data.ID),
		strings.TrimSpace(data.EdgeOperationID),
		strings.TrimSpace(receipt.Envelope.EventID),
		receiptID,
		strings.TrimSpace(data.RestaurantID),
		strings.TrimSpace(data.DeviceID),
		nullableText(receipt.Envelope.NodeDeviceID),
		nullableStringPtr(receipt.Envelope.ClientDeviceID),
		nullableStringPtr(receipt.Envelope.ActorEmployeeID),
		nullableStringPtr(receipt.Envelope.SessionID),
		strings.TrimSpace(data.ShiftID),
		strings.TrimSpace(data.OriginalShiftID),
		strings.TrimSpace(data.CheckID),
		strings.TrimSpace(data.PrecheckID),
		strings.TrimSpace(data.OperationType),
		strings.TrimSpace(data.OperationKind),
		data.Amount,
		strings.TrimSpace(data.Currency),
		strings.TrimSpace(data.BusinessDateLocal),
		strings.TrimSpace(data.InventoryDisposition),
		strings.TrimSpace(data.Reason),
		nullableText(data.CreatedByEmployeeID),
		nullableStringPtr(data.ApprovedByEmployeeID),
		string(data.Snapshot),
		data.CreatedAt,
		receipt.Envelope.OccurredAt,
		receipt.CloudReceivedAt,
		receipt.RawPayloadSHA256,
	)
	return err
}

func enqueueInventoryEvent(ctx context.Context, tx pgx.Tx, receipt app.EdgeEventReceipt, receiptID string) error {
	_, err := tx.Exec(ctx, `
INSERT INTO inventory_event_queue(
  id,receipt_id,restaurant_id,device_id,event_id,event_type,aggregate_type,aggregate_id,status,attempts,occurred_at,created_at,updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending',0,$9,$10,$10)
ON CONFLICT (receipt_id) DO NOTHING`,
		receiptID,
		receiptID,
		strings.TrimSpace(*receipt.Envelope.RestaurantID),
		strings.TrimSpace(receipt.Envelope.DeviceID),
		strings.TrimSpace(receipt.Envelope.EventID),
		string(receipt.Envelope.EventType),
		strings.TrimSpace(receipt.Envelope.AggregateType),
		strings.TrimSpace(receipt.Envelope.AggregateID),
		receipt.Envelope.OccurredAt,
		receipt.CloudReceivedAt,
	)
	return err
}

func enqueueExistingInboxEvent(ctx context.Context, tx pgx.Tx, receiptID string) error {
	_, err := tx.Exec(ctx, `
INSERT INTO inbox_events(
  id,receipt_id,idempotency_key,tenant_id,restaurant_id,device_id,employee_id,
  command_id,event_id,edge_event_id,event_type,aggregate_type,aggregate_id,envelope_version,
  occurred_at,cloud_received_at,raw_payload,raw_payload_sha256_hex,processed_for_olap,olap_export_status,created_at,updated_at
)
SELECT
  r.id,r.id,r.idempotency_key,r.restaurant_id,r.restaurant_id,r.device_id,'',
  r.command_id,r.event_id,r.edge_event_id,r.event_type,r.aggregate_type,r.aggregate_id,r.envelope_version,
  r.occurred_at,r.cloud_received_at,p.raw_payload,p.raw_payload_sha256_hex,false,'pending',r.cloud_received_at,r.cloud_received_at
FROM cloud_edge_event_receipts r
JOIN cloud_edge_event_raw_payloads p ON p.receipt_id = r.id
WHERE r.id = $1
ON CONFLICT (id) DO NOTHING`, receiptID)
	return err
}

func upsertShiftFinanceProjection(ctx context.Context, tx pgx.Tx, receipt app.EdgeEventReceipt, shiftID string, paymentCount, paymentTotal, paymentRefundCount, paymentRefundTotal, checkCount, checkTotal, checkRefundCount, checkRefundTotal int64) error {
	_, err := tx.Exec(ctx, `
INSERT INTO cloud_projection_shift_finance(
  restaurant_id,device_id,shift_id,payments_captured_count,payments_captured_total,payments_refunded_count,payments_refunded_total,checks_created_count,checks_total_amount,checks_refunded_count,checks_refunded_total,last_event_id,last_command_id,last_occurred_at,last_cloud_received_at,updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$15)
ON CONFLICT (restaurant_id,device_id,shift_id) DO UPDATE SET
  payments_captured_count = cloud_projection_shift_finance.payments_captured_count + EXCLUDED.payments_captured_count,
  payments_captured_total = cloud_projection_shift_finance.payments_captured_total + EXCLUDED.payments_captured_total,
  payments_refunded_count = cloud_projection_shift_finance.payments_refunded_count + EXCLUDED.payments_refunded_count,
  payments_refunded_total = cloud_projection_shift_finance.payments_refunded_total + EXCLUDED.payments_refunded_total,
  checks_created_count = cloud_projection_shift_finance.checks_created_count + EXCLUDED.checks_created_count,
  checks_total_amount = cloud_projection_shift_finance.checks_total_amount + EXCLUDED.checks_total_amount,
  checks_refunded_count = cloud_projection_shift_finance.checks_refunded_count + EXCLUDED.checks_refunded_count,
  checks_refunded_total = cloud_projection_shift_finance.checks_refunded_total + EXCLUDED.checks_refunded_total,
  last_event_id = EXCLUDED.last_event_id,
  last_command_id = EXCLUDED.last_command_id,
  last_occurred_at = GREATEST(cloud_projection_shift_finance.last_occurred_at, EXCLUDED.last_occurred_at),
  last_cloud_received_at = EXCLUDED.last_cloud_received_at,
  updated_at = EXCLUDED.updated_at`,
		*receipt.Envelope.RestaurantID,
		receipt.Envelope.DeviceID,
		shiftID,
		paymentCount,
		paymentTotal,
		paymentRefundCount,
		paymentRefundTotal,
		checkCount,
		checkTotal,
		checkRefundCount,
		checkRefundTotal,
		receipt.Envelope.EventID,
		receipt.Envelope.CommandID,
		receipt.Envelope.OccurredAt,
		receipt.CloudReceivedAt,
	)
	return err
}

func paymentAmount(payloadRaw json.RawMessage) (int64, error) {
	var payload contracts.Payload[contracts.PaymentCaptured]
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return 0, fmt.Errorf("%w: invalid PaymentCaptured payload", contracts.ErrInvalidEnvelope)
	}
	return payload.Data.Amount, nil
}

func paymentRefundAmount(payloadRaw json.RawMessage) (int64, error) {
	var payload contracts.Payload[contracts.PaymentRefunded]
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return 0, fmt.Errorf("%w: invalid PaymentRefunded payload", contracts.ErrInvalidEnvelope)
	}
	return payload.Data.Amount, nil
}

func checkTotal(payloadRaw json.RawMessage) (int64, error) {
	var payload contracts.Payload[contracts.CheckCreated]
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return 0, fmt.Errorf("%w: invalid CheckCreated payload", contracts.ErrInvalidEnvelope)
	}
	return payload.Data.Total, nil
}

func checkRefundedPaidTotal(payloadRaw json.RawMessage) (int64, error) {
	var payload contracts.Payload[contracts.CheckRefunded]
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return 0, fmt.Errorf("%w: invalid CheckRefunded payload", contracts.ErrInvalidEnvelope)
	}
	return payload.Data.PaidTotal, nil
}

func financialOperationAmount(payloadRaw json.RawMessage) (int64, error) {
	var payload contracts.Payload[contracts.FinancialOperationRecorded]
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return 0, fmt.Errorf("%w: invalid financial operation payload", contracts.ErrInvalidEnvelope)
	}
	return payload.Data.Amount, nil
}

func bytesTrimSpace(v []byte) []byte {
	return []byte(strings.TrimSpace(string(v)))
}

func nullableText(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
}

func nullableStringPtr(v *string) any {
	if v == nil {
		return nil
	}
	return nullableText(*v)
}

func trimStringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func scanAck(row pgx.Row) (contracts.EventAck, error) {
	var ack contracts.EventAck
	if err := row.Scan(
		&ack.CloudReceiptID,
		&ack.IdempotencyKey,
		&ack.CommandID,
		&ack.EventID,
		&ack.EdgeEventID,
		&ack.EnvelopeVersion,
		&ack.CloudReceivedAt,
		&ack.RawPayloadSHA256Hex,
	); err != nil {
		return contracts.EventAck{}, err
	}
	ack.Status = "accepted"
	return ack, nil
}

func newID() (string, error) {
	return idgen.NewV7()
}

func secretHash(v string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(v)))
	return "sha256:" + hex.EncodeToString(sum[:])
}
