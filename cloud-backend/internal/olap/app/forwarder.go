package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cloud-backend/internal/platform/clock"
)

type InboxEvent struct {
	ID                  string
	ReceiptID           string
	TenantID            string
	RestaurantID        string
	DeviceID            string
	EmployeeID          string
	CommandID           string
	EventID             string
	EdgeEventID         string
	EventType           string
	AggregateType       string
	AggregateID         string
	EnvelopeVersion     string
	OccurredAt          time.Time
	CloudReceivedAt     time.Time
	RawPayload          json.RawMessage
	RawPayloadSHA256Hex string
}

type ClaimCommand struct {
	Limit       int
	LockedBy    string
	Now         time.Time
	StaleBefore time.Time
}

type QueueRepository interface {
	ClaimPending(context.Context, ClaimCommand) ([]InboxEvent, error)
	MarkProcessed(context.Context, []InboxEvent, time.Time) error
	MarkFailed(context.Context, []InboxEvent, string, time.Time, time.Time) error
}

type Exporter interface {
	InsertRawBusinessEvents(context.Context, []InboxEvent, time.Time) error
}

type ForwarderConfig struct {
	WorkerID      string
	BatchSize     int
	RetryDelay    time.Duration
	ProcessingTTL time.Duration
}

type Forwarder struct {
	queue    QueueRepository
	exporter Exporter
	clock    clock.Clock
	config   ForwarderConfig
	logger   *slog.Logger
}

func NewForwarder(queue QueueRepository, exporter Exporter, clock clock.Clock, config ForwarderConfig) *Forwarder {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = fmt.Sprintf("olap-forwarder-%d", time.Now().UnixNano())
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 1000
	}
	if config.BatchSize > 100000 {
		config.BatchSize = 100000
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = time.Minute
	}
	if config.ProcessingTTL <= 0 {
		config.ProcessingTTL = 5 * time.Minute
	}
	return &Forwarder{
		queue:    queue,
		exporter: exporter,
		clock:    clock,
		config:   config,
		logger:   slog.Default(),
	}
}

func (f *Forwarder) RunOnce(ctx context.Context) error {
	if f == nil || f.queue == nil || f.exporter == nil {
		return ErrOLAPUnavailable
	}
	now := f.clock.Now().UTC()
	events, err := f.queue.ClaimPending(ctx, ClaimCommand{
		Limit:       f.config.BatchSize,
		LockedBy:    f.config.WorkerID,
		Now:         now,
		StaleBefore: now.Add(-f.config.ProcessingTTL),
	})
	if err != nil || len(events) == 0 {
		return err
	}
	if err := f.exporter.InsertRawBusinessEvents(ctx, events, now); err != nil {
		nextRetry := now.Add(f.config.RetryDelay)
		if markErr := f.queue.MarkFailed(ctx, events, safeError(err), nextRetry, now); markErr != nil {
			return markErr
		}
		f.logger.WarnContext(ctx, "clickhouse export failed",
			"operation", "olap.forwarder",
			"action", "export_raw_business_events",
			"result", "retry_scheduled",
			"error_code", "CLICKHOUSE_EXPORT_FAILED",
			"batch_size", len(events),
			"next_retry_at", nextRetry,
			"internal_error", safeError(err),
		)
		return nil
	}
	return f.queue.MarkProcessed(ctx, events, now)
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "olap export failed"
	}
	if len(msg) > 500 {
		return msg[:500]
	}
	return msg
}
