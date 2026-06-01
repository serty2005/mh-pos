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

type StockMoveQueueRepository interface {
	ClaimPendingStockMoves(context.Context, ClaimCommand) ([]StockMove, error)
	MarkStockMovesProcessed(context.Context, []StockMove, time.Time) error
	MarkStockMovesFailed(context.Context, []StockMove, string, time.Time, time.Time) error
}

type StockMoveExporter interface {
	InsertStockMoves(context.Context, []StockMove, time.Time) error
}

type BackfillBatch struct {
	RawEvents  []InboxEvent
	StockMoves []StockMove
}

type BackfillQueueRepository interface {
	ClaimBackfillJob(context.Context, string, time.Time) (BackfillJob, bool, error)
	LoadBackfillBatch(context.Context, BackfillJob, int) (BackfillBatch, error)
	MarkBackfillProgress(context.Context, BackfillJob, BackfillBatch, time.Time) error
	MarkBackfillFailed(context.Context, BackfillJob, string, time.Time) error
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

type StockMoveForwarder struct {
	queue    StockMoveQueueRepository
	exporter StockMoveExporter
	clock    clock.Clock
	config   ForwarderConfig
	logger   *slog.Logger
}

type BackfillWorker struct {
	queue    BackfillQueueRepository
	exporter interface {
		Exporter
		StockMoveExporter
	}
	clock  clock.Clock
	config ForwarderConfig
	logger *slog.Logger
}

func NewForwarder(queue QueueRepository, exporter Exporter, clock clock.Clock, config ForwarderConfig) *Forwarder {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = fmt.Sprintf("olap-forwarder-%d", time.Now().UnixNano())
	}
	config = normalizeForwarderConfig(config)
	return &Forwarder{
		queue:    queue,
		exporter: exporter,
		clock:    clock,
		config:   config,
		logger:   slog.Default(),
	}
}

func NewStockMoveForwarder(queue StockMoveQueueRepository, exporter StockMoveExporter, clock clock.Clock, config ForwarderConfig) *StockMoveForwarder {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = fmt.Sprintf("olap-stock-moves-forwarder-%d", time.Now().UnixNano())
	}
	config = normalizeForwarderConfig(config)
	return &StockMoveForwarder{
		queue:    queue,
		exporter: exporter,
		clock:    clock,
		config:   config,
		logger:   slog.Default(),
	}
}

func NewBackfillWorker(queue BackfillQueueRepository, exporter interface {
	Exporter
	StockMoveExporter
}, clock clock.Clock, config ForwarderConfig) *BackfillWorker {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = fmt.Sprintf("olap-backfill-worker-%d", time.Now().UnixNano())
	}
	config = normalizeForwarderConfig(config)
	return &BackfillWorker{
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

func (f *StockMoveForwarder) RunOnce(ctx context.Context) error {
	if f == nil || f.queue == nil || f.exporter == nil {
		return ErrOLAPUnavailable
	}
	now := f.clock.Now().UTC()
	moves, err := f.queue.ClaimPendingStockMoves(ctx, ClaimCommand{
		Limit:       f.config.BatchSize,
		LockedBy:    f.config.WorkerID,
		Now:         now,
		StaleBefore: now.Add(-f.config.ProcessingTTL),
	})
	if err != nil || len(moves) == 0 {
		return err
	}
	if err := f.exporter.InsertStockMoves(ctx, moves, now); err != nil {
		nextRetry := now.Add(f.config.RetryDelay)
		if markErr := f.queue.MarkStockMovesFailed(ctx, moves, safeError(err), nextRetry, now); markErr != nil {
			return markErr
		}
		f.logger.WarnContext(ctx, "clickhouse stock moves export failed",
			"operation", "olap.forwarder",
			"action", "export_stock_moves",
			"result", "retry_scheduled",
			"error_code", "CLICKHOUSE_STOCK_MOVES_EXPORT_FAILED",
			"batch_size", len(moves),
			"next_retry_at", nextRetry,
			"internal_error", safeError(err),
		)
		return nil
	}
	return f.queue.MarkStockMovesProcessed(ctx, moves, now)
}

func (w *BackfillWorker) RunOnce(ctx context.Context) error {
	if w == nil || w.queue == nil || w.exporter == nil {
		return ErrOLAPUnavailable
	}
	now := w.clock.Now().UTC()
	job, ok, err := w.queue.ClaimBackfillJob(ctx, w.config.WorkerID, now)
	if err != nil || !ok {
		return err
	}
	batchSize := job.BatchSize
	if batchSize <= 0 || batchSize > w.config.BatchSize {
		batchSize = w.config.BatchSize
	}
	batch, err := w.queue.LoadBackfillBatch(ctx, job, batchSize)
	if err != nil {
		return w.queue.MarkBackfillFailed(ctx, job, safeError(err), now)
	}
	if len(batch.RawEvents) == 0 && len(batch.StockMoves) == 0 {
		return w.queue.MarkBackfillProgress(ctx, job, batch, now)
	}
	if len(batch.RawEvents) > 0 {
		if err := w.exporter.InsertRawBusinessEvents(ctx, batch.RawEvents, now); err != nil {
			if markErr := w.queue.MarkBackfillFailed(ctx, job, safeError(err), now); markErr != nil {
				return markErr
			}
			w.logBackfillFailure(ctx, job, err)
			return nil
		}
	}
	if len(batch.StockMoves) > 0 {
		if err := w.exporter.InsertStockMoves(ctx, batch.StockMoves, now); err != nil {
			if markErr := w.queue.MarkBackfillFailed(ctx, job, safeError(err), now); markErr != nil {
				return markErr
			}
			w.logBackfillFailure(ctx, job, err)
			return nil
		}
	}
	return w.queue.MarkBackfillProgress(ctx, job, batch, now)
}

func (w *BackfillWorker) logBackfillFailure(ctx context.Context, job BackfillJob, err error) {
	w.logger.WarnContext(ctx, "clickhouse backfill failed",
		"operation", "olap.backfill",
		"action", "export_backfill_batch",
		"result", "retry_required",
		"error_code", "CLICKHOUSE_BACKFILL_FAILED",
		"job_id", job.ID,
		"stream", job.Stream,
		"internal_error", safeError(err),
	)
}

func normalizeForwarderConfig(config ForwarderConfig) ForwarderConfig {
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
	return config
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
