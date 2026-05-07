package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

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
	if err := tx.Commit(ctx); err != nil {
		return contracts.EventAck{}, err
	}
	return ack, nil
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
