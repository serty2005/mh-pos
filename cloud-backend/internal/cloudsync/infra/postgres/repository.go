package postgres

import (
	"context"
	"crypto/rand"
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
	if err := r.applyEventProjections(ctx, tx, receipt); err != nil {
		return contracts.EventAck{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return contracts.EventAck{}, err
	}
	return ack, nil
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

func (r *Repository) applyEventProjections(ctx context.Context, tx pgx.Tx, receipt app.EdgeEventReceipt) error {
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
	default:
		return nil
	}
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

func bytesTrimSpace(v []byte) []byte {
	return []byte(strings.TrimSpace(string(v)))
}

func nullableText(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return strings.TrimSpace(v)
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
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	), nil
}
