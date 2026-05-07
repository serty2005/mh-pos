package syncsender

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"pos-backend/internal/pos/app"
	"pos-backend/internal/pos/domain"
)

type OutboxService interface {
	ClaimPendingOutbox(context.Context, app.ClaimPendingOutboxCommand) ([]domain.OutboxMessage, error)
	ReclaimStaleProcessingOutbox(context.Context, app.ReclaimStaleOutboxCommand) (int, error)
	ReleaseProcessingOutbox(context.Context, string) (int, error)
	MarkOutboxSent(context.Context, string) error
	MarkOutboxRetryableFailure(context.Context, string, string) error
	SuspendOutboxMessage(context.Context, string, string) error
}

type Sender interface {
	Send(context.Context, domain.OutboxMessage) error
}

type NonRetryableError struct {
	Reason string
}

func (e NonRetryableError) Error() string {
	return e.Reason
}

type Config struct {
	WorkerID     string
	BatchSize    int
	PollInterval time.Duration
	ReclaimAfter time.Duration
	SendTimeout  time.Duration
}

type Worker struct {
	service OutboxService
	sender  Sender
	config  Config
	logger  *slog.Logger
}

func NewWorker(service OutboxService, sender Sender, config Config, logger *slog.Logger) *Worker {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = fmt.Sprintf("pos-sync-sender-%d", time.Now().UnixNano())
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 25
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 2 * time.Second
	}
	if config.ReclaimAfter <= 0 {
		config.ReclaimAfter = 5 * time.Minute
	}
	if config.SendTimeout <= 0 {
		config.SendTimeout = 10 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{service: service, sender: sender, config: config, logger: logger}
}

func (w *Worker) Run(ctx context.Context) {
	w.logger.Info("POS sync sender started", "worker_id", w.config.WorkerID, "batch_size", w.config.BatchSize)
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if released, err := w.service.ReleaseProcessingOutbox(releaseCtx, w.config.WorkerID); err != nil {
			w.logger.Warn("POS sync sender failed to release locks during shutdown", "worker_id", w.config.WorkerID, "error", err)
		} else if released > 0 {
			w.logger.Info("POS sync sender released processing locks during shutdown", "worker_id", w.config.WorkerID, "released", released)
		}
		w.logger.Info("POS sync sender stopped", "worker_id", w.config.WorkerID)
	}()

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Warn("POS sync sender iteration failed", "worker_id", w.config.WorkerID, "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	if _, err := w.service.ReclaimStaleProcessingOutbox(ctx, app.ReclaimStaleOutboxCommand{
		StaleBefore: time.Now().Add(-w.config.ReclaimAfter),
	}); err != nil {
		return fmt.Errorf("reclaim stale outbox: %w", err)
	}
	messages, err := w.service.ClaimPendingOutbox(ctx, app.ClaimPendingOutboxCommand{
		Limit:    w.config.BatchSize,
		LockedBy: w.config.WorkerID,
	})
	if err != nil {
		return fmt.Errorf("claim pending outbox: %w", err)
	}
	for _, msg := range messages {
		if err := ctx.Err(); err != nil {
			_, _ = w.service.ReleaseProcessingOutbox(context.Background(), w.config.WorkerID)
			return err
		}
		if reason := blockedDirectionReason(msg); reason != "" {
			if err := w.service.SuspendOutboxMessage(ctx, msg.ID, reason); err != nil {
				return fmt.Errorf("suspend wrong-direction outbox %s: %w", msg.ID, err)
			}
			w.logger.Info("POS sync sender suspended outbox message", "id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType, "reason", reason)
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, w.config.SendTimeout)
		sendErr := w.sender.Send(sendCtx, msg)
		cancel()
		if sendErr == nil {
			if err := w.service.MarkOutboxSent(ctx, msg.ID); err != nil {
				return fmt.Errorf("mark outbox sent %s: %w", msg.ID, err)
			}
			w.logger.Info("POS sync sender delivered outbox message", "id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType)
			continue
		}
		var nonRetryable NonRetryableError
		if errors.As(sendErr, &nonRetryable) {
			if err := w.service.SuspendOutboxMessage(ctx, msg.ID, nonRetryable.Reason); err != nil {
				return fmt.Errorf("suspend non-retryable outbox %s: %w", msg.ID, err)
			}
			w.logger.Warn("POS sync sender suspended non-retryable outbox message", "id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType, "error", nonRetryable.Reason)
			continue
		}
		if err := w.service.MarkOutboxRetryableFailure(ctx, msg.ID, sendErr.Error()); err != nil {
			return fmt.Errorf("mark retryable outbox failure %s: %w", msg.ID, err)
		}
		if _, err := w.service.ReleaseProcessingOutbox(ctx, w.config.WorkerID); err != nil {
			return fmt.Errorf("release remaining processing outbox after failure: %w", err)
		}
		return fmt.Errorf("retryable send failure for outbox %s: %w", msg.ID, sendErr)
	}
	return nil
}

func blockedDirectionReason(msg domain.OutboxMessage) string {
	if msg.Origin != domain.OriginEdgeDevice {
		return fmt.Sprintf("sync direction blocked: origin %q is not Edge runtime origin", msg.Origin)
	}
	if !isEdgeToCloudOperationalEvent(msg.CommandType) {
		return fmt.Sprintf("sync direction blocked: %s is Cloud-managed/configuration or unsupported Edge->Cloud event", msg.CommandType)
	}
	return ""
}

func isEdgeToCloudOperationalEvent(eventType string) bool {
	switch eventType {
	case "ShiftOpened",
		"ShiftClosed",
		"CashSessionOpened",
		"CashSessionClosed",
		"CashDrawerEventRecorded",
		"OrderCreated",
		"OrderLineAdded",
		"OrderLineQuantityChanged",
		"OrderLineVoided",
		"PrecheckIssued",
		"PrecheckCancelled",
		"PaymentCaptured",
		"CheckCreated",
		"OrderClosed",
		"AuthSessionStarted",
		"AuthSessionRevoked",
		"DeviceRegistered":
		return true
	default:
		return false
	}
}
