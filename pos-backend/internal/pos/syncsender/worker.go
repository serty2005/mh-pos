package syncsender

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	platformlog "pos-backend/internal/platform/logging"
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
	RefreshSyncExchangeState(context.Context) error
	GetSyncExchangeState(context.Context) (SyncExchangeState, error)
	ApplySyncExchangeCloudPackages(context.Context, []CloudPackage) error
}

type Sender interface {
	Send(context.Context, domain.OutboxMessage) error
}

type BatchSender interface {
	SendBatch(context.Context, []domain.OutboxMessage) ([]BatchSendResult, error)
}

type ExchangeSender interface {
	Exchange(context.Context, SyncExchangeRequest) (SyncExchangeResponse, error)
}

type BatchSendResult struct {
	OutboxID string
	Status   BatchSendStatus
	Reason   string
}

type BatchSendStatus string

const (
	BatchSendAccepted  BatchSendStatus = "accepted"
	BatchSendRejected  BatchSendStatus = "rejected"
	BatchSendRetryable BatchSendStatus = "retryable"
)

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
	PollJitter   time.Duration
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
	if config.PollJitter < 0 {
		config.PollJitter = 0
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
	platformlog.Log(ctx, w.logger, slog.LevelInfo, "POS sync sender started", platformlog.Event{
		Operation: "sync.sender",
		Action:    "worker.start",
		Result:    "success",
	}, "worker_id", w.config.WorkerID, "batch_size", w.config.BatchSize)
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if released, err := w.service.ReleaseProcessingOutbox(releaseCtx, w.config.WorkerID); err != nil {
			platformlog.Log(releaseCtx, w.logger, slog.LevelWarn, "POS sync sender failed to release locks during shutdown", platformlog.Event{
				Operation: "sync.sender",
				Action:    "worker.release_shutdown_locks",
				Result:    "rejected",
				ErrorCode: "LOCK_RELEASE_FAILED",
			}, "worker_id", w.config.WorkerID, "error", err)
		} else if released > 0 {
			platformlog.Log(releaseCtx, w.logger, slog.LevelInfo, "POS sync sender released processing locks during shutdown", platformlog.Event{
				Operation: "sync.sender",
				Action:    "worker.release_shutdown_locks",
				Result:    "success",
			}, "worker_id", w.config.WorkerID, "released", released)
		}
		platformlog.Log(releaseCtx, w.logger, slog.LevelInfo, "POS sync sender stopped", platformlog.Event{
			Operation: "sync.sender",
			Action:    "worker.stop",
			Result:    "success",
		}, "worker_id", w.config.WorkerID)
	}()

	for {
		if err := w.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			platformlog.Log(ctx, w.logger, slog.LevelWarn, "POS sync sender iteration failed", platformlog.Event{
				Operation: "sync.sender",
				Action:    "iteration.run_once",
				Result:    "rejected",
				ErrorCode: "ITERATION_FAILED",
			}, "worker_id", w.config.WorkerID, "error", err)
		}
		waitFor := w.nextPollDelay()
		select {
		case <-ctx.Done():
			return
		case <-time.After(waitFor):
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	platformlog.Log(ctx, w.logger, platformlog.LevelTrace, "POS sync sender reclaim step", platformlog.Event{
		Operation: "sync.sender",
		Action:    "reclaim.stale_processing",
		Result:    "attempt",
	}, "worker_id", w.config.WorkerID, "reclaim_after_ms", w.config.ReclaimAfter.Milliseconds())
	if _, err := w.service.ReclaimStaleProcessingOutbox(ctx, app.ReclaimStaleOutboxCommand{
		StaleBefore: time.Now().Add(-w.config.ReclaimAfter),
	}); err != nil {
		return fmt.Errorf("reclaim stale outbox: %w", err)
	}
	platformlog.Log(ctx, w.logger, platformlog.LevelTrace, "POS sync sender claim step", platformlog.Event{
		Operation: "sync.sender",
		Action:    "claim.pending_batch",
		Result:    "attempt",
	}, "worker_id", w.config.WorkerID, "batch_size", w.config.BatchSize)
	messages, err := w.service.ClaimPendingOutbox(ctx, app.ClaimPendingOutboxCommand{
		Limit:    w.config.BatchSize,
		LockedBy: w.config.WorkerID,
	})
	if err != nil {
		return fmt.Errorf("claim pending outbox: %w", err)
	}
	platformlog.Log(ctx, w.logger, slog.LevelDebug, "POS sync sender claimed pending batch", platformlog.Event{
		Operation: "sync.sender",
		Action:    "claim.pending_batch",
		Result:    "success",
	}, "worker_id", w.config.WorkerID, "claimed_count", len(messages))
	sendable := make([]domain.OutboxMessage, 0, len(messages))
	for _, msg := range messages {
		platformlog.Log(ctx, w.logger, platformlog.LevelTrace, "POS sync sender process message", platformlog.Event{
			Operation:       "sync.sender",
			Action:          "message.process",
			Result:          "attempt",
			NodeDeviceID:    msg.NodeDeviceID,
			ClientDeviceID:  derefOptional(msg.ClientDeviceID),
			SessionID:       derefOptional(msg.SessionID),
			ActorEmployeeID: derefOptional(msg.ActorEmployeeID),
		}, "worker_id", w.config.WorkerID, "outbox_id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType)
		if err := ctx.Err(); err != nil {
			_, _ = w.service.ReleaseProcessingOutbox(context.Background(), w.config.WorkerID)
			return err
		}
		if reason := blockedDirectionReason(msg); reason != "" {
			if err := w.service.SuspendOutboxMessage(ctx, msg.ID, reason); err != nil {
				return fmt.Errorf("suspend wrong-direction outbox %s: %w", msg.ID, err)
			}
			platformlog.Log(ctx, w.logger, slog.LevelWarn, "POS sync sender suspended outbox message", platformlog.Event{
				Operation:       "sync.sender",
				Action:          "message.suspend",
				Result:          "rejected",
				ErrorCode:       "SYNC_DIRECTION_BLOCKED",
				NodeDeviceID:    msg.NodeDeviceID,
				ClientDeviceID:  derefOptional(msg.ClientDeviceID),
				SessionID:       derefOptional(msg.SessionID),
				ActorEmployeeID: derefOptional(msg.ActorEmployeeID),
			}, "outbox_id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType, "reason", reason)
			continue
		}
		sendable = append(sendable, msg)
	}
	if exchangeSender, ok := w.sender.(ExchangeSender); ok {
		if err := w.service.RefreshSyncExchangeState(ctx); err != nil {
			platformlog.Log(ctx, w.logger, slog.LevelWarn, "POS sync sender provisioning refresh failed", platformlog.Event{
				Operation: "sync.sender",
				Action:    "exchange.refresh_state",
				Result:    "rejected",
				ErrorCode: "SYNC_EXCHANGE_STATE_REFRESH_FAILED",
			}, "worker_id", w.config.WorkerID, "error", err)
		}
		state, err := w.service.GetSyncExchangeState(ctx)
		if err != nil {
			return fmt.Errorf("get sync exchange state: %w", err)
		}
		if state.NodeDeviceID != "" && state.AuthToken != "" {
			return w.sendExchange(ctx, exchangeSender, sendable, state)
		}
	}
	if len(sendable) == 0 {
		return nil
	}
	if batchSender, ok := w.sender.(BatchSender); ok && len(sendable) > 1 {
		return w.sendBatch(ctx, batchSender, sendable)
	}
	for _, msg := range sendable {
		platformlog.Log(ctx, w.logger, platformlog.LevelTrace, "POS sync sender send attempt", platformlog.Event{
			Operation:       "sync.sender",
			Action:          "message.send",
			Result:          "attempt",
			NodeDeviceID:    msg.NodeDeviceID,
			ClientDeviceID:  derefOptional(msg.ClientDeviceID),
			SessionID:       derefOptional(msg.SessionID),
			ActorEmployeeID: derefOptional(msg.ActorEmployeeID),
		}, "outbox_id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType)
		sendCtx, cancel := context.WithTimeout(ctx, w.config.SendTimeout)
		sendErr := w.sender.Send(sendCtx, msg)
		cancel()
		if sendErr == nil {
			if err := w.service.MarkOutboxSent(ctx, msg.ID); err != nil {
				return fmt.Errorf("mark outbox sent %s: %w", msg.ID, err)
			}
			platformlog.Log(ctx, w.logger, slog.LevelInfo, "POS sync sender delivered outbox message", platformlog.Event{
				Operation:       "sync.sender",
				Action:          "message.ack",
				Result:          "success",
				NodeDeviceID:    msg.NodeDeviceID,
				ClientDeviceID:  derefOptional(msg.ClientDeviceID),
				SessionID:       derefOptional(msg.SessionID),
				ActorEmployeeID: derefOptional(msg.ActorEmployeeID),
			}, "outbox_id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType)
			continue
		}
		var nonRetryable NonRetryableError
		if errors.As(sendErr, &nonRetryable) {
			if err := w.service.SuspendOutboxMessage(ctx, msg.ID, nonRetryable.Reason); err != nil {
				return fmt.Errorf("suspend non-retryable outbox %s: %w", msg.ID, err)
			}
			platformlog.Log(ctx, w.logger, slog.LevelWarn, "POS sync sender suspended non-retryable outbox message", platformlog.Event{
				Operation:       "sync.sender",
				Action:          "message.suspend",
				Result:          "rejected",
				ErrorCode:       "SEND_NON_RETRYABLE",
				NodeDeviceID:    msg.NodeDeviceID,
				ClientDeviceID:  derefOptional(msg.ClientDeviceID),
				SessionID:       derefOptional(msg.SessionID),
				ActorEmployeeID: derefOptional(msg.ActorEmployeeID),
			}, "outbox_id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType, "error", nonRetryable.Reason)
			continue
		}
		platformlog.Log(ctx, w.logger, platformlog.LevelTrace, "POS sync sender retry decision", platformlog.Event{
			Operation:       "sync.sender",
			Action:          "message.retryable_failure",
			Result:          "attempt",
			ErrorCode:       "SEND_RETRYABLE",
			NodeDeviceID:    msg.NodeDeviceID,
			ClientDeviceID:  derefOptional(msg.ClientDeviceID),
			SessionID:       derefOptional(msg.SessionID),
			ActorEmployeeID: derefOptional(msg.ActorEmployeeID),
		}, "outbox_id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType, "error", sendErr.Error())
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

func (w *Worker) sendBatch(ctx context.Context, sender BatchSender, messages []domain.OutboxMessage) error {
	platformlog.Log(ctx, w.logger, platformlog.LevelTrace, "POS sync sender batch send attempt", platformlog.Event{
		Operation: "sync.sender",
		Action:    "batch.send",
		Result:    "attempt",
	}, "worker_id", w.config.WorkerID, "batch_count", len(messages))
	sendCtx, cancel := context.WithTimeout(ctx, w.config.SendTimeout)
	results, err := sender.SendBatch(sendCtx, messages)
	cancel()
	if err != nil {
		for _, msg := range messages {
			markErr := w.service.MarkOutboxRetryableFailure(ctx, msg.ID, err.Error())
			if markErr != nil {
				return fmt.Errorf("mark batch retryable outbox failure %s: %w", msg.ID, markErr)
			}
		}
		return fmt.Errorf("batch send failure: %w", err)
	}
	return w.processBatchResults(ctx, messages, results)
}

func (w *Worker) sendExchange(ctx context.Context, sender ExchangeSender, messages []domain.OutboxMessage, state SyncExchangeState) error {
	events := make([]SyncExchangeEdgeEvent, 0, len(messages))
	for _, msg := range messages {
		events = append(events, SyncExchangeEdgeEvent{
			ClientItemID: msg.ID,
			Payload:      json.RawMessage(msg.PayloadJSON),
		})
	}
	req := SyncExchangeRequest{
		ProtocolVersion: SyncExchangeProtocolVersion,
		NodeDeviceID:    state.NodeDeviceID,
		RestaurantID:    state.RestaurantID,
		AuthToken:       state.AuthToken,
		EdgeEvents:      events,
		Streams:         append([]SyncExchangeStreamRequest(nil), state.Streams...),
	}
	platformlog.Log(ctx, w.logger, platformlog.LevelTrace, "POS sync sender exchange attempt", platformlog.Event{
		Operation: "sync.sender",
		Action:    "exchange.send",
		Result:    "attempt",
	}, "worker_id", w.config.WorkerID, "batch_count", len(messages), "stream_count", len(state.Streams))
	sendCtx, cancel := context.WithTimeout(ctx, w.config.SendTimeout)
	resp, err := sender.Exchange(sendCtx, req)
	cancel()
	if err != nil {
		for _, msg := range messages {
			if markErr := w.service.MarkOutboxRetryableFailure(ctx, msg.ID, err.Error()); markErr != nil {
				return fmt.Errorf("mark exchange retryable outbox failure %s: %w", msg.ID, markErr)
			}
		}
		return fmt.Errorf("exchange send failure: %w", err)
	}
	if len(resp.CloudPackages) > 0 {
		if err := w.service.ApplySyncExchangeCloudPackages(ctx, resp.CloudPackages); err != nil {
			for _, msg := range messages {
				if markErr := w.service.MarkOutboxRetryableFailure(ctx, msg.ID, err.Error()); markErr != nil {
					return fmt.Errorf("mark exchange apply retryable outbox failure %s: %w", msg.ID, markErr)
				}
			}
			return fmt.Errorf("apply exchange cloud packages: %w", err)
		}
	}
	return w.processBatchResults(ctx, messages, resp.EdgeAcks)
}

func (w *Worker) processBatchResults(ctx context.Context, messages []domain.OutboxMessage, results []BatchSendResult) error {
	resultByID := make(map[string]BatchSendResult, len(results))
	for _, item := range results {
		if strings.TrimSpace(item.OutboxID) == "" {
			continue
		}
		resultByID[item.OutboxID] = item
	}
	var hadRetryable bool
	for _, msg := range messages {
		item, ok := resultByID[msg.ID]
		if !ok {
			hadRetryable = true
			if markErr := w.service.MarkOutboxRetryableFailure(ctx, msg.ID, "cloud batch ack did not include outbox item result"); markErr != nil {
				return fmt.Errorf("mark missing batch result retryable %s: %w", msg.ID, markErr)
			}
			continue
		}
		switch item.Status {
		case BatchSendAccepted:
			if err := w.service.MarkOutboxSent(ctx, msg.ID); err != nil {
				return fmt.Errorf("mark outbox sent %s: %w", msg.ID, err)
			}
			platformlog.Log(ctx, w.logger, slog.LevelInfo, "POS sync sender delivered outbox message", platformlog.Event{
				Operation:       "sync.sender",
				Action:          "message.ack",
				Result:          "success",
				NodeDeviceID:    msg.NodeDeviceID,
				ClientDeviceID:  derefOptional(msg.ClientDeviceID),
				SessionID:       derefOptional(msg.SessionID),
				ActorEmployeeID: derefOptional(msg.ActorEmployeeID),
			}, "outbox_id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType)
		case BatchSendRejected:
			reason := strings.TrimSpace(item.Reason)
			if reason == "" {
				reason = "cloud rejected sync event item"
			}
			if err := w.service.SuspendOutboxMessage(ctx, msg.ID, reason); err != nil {
				return fmt.Errorf("suspend non-retryable outbox %s: %w", msg.ID, err)
			}
			platformlog.Log(ctx, w.logger, slog.LevelWarn, "POS sync sender suspended non-retryable outbox message", platformlog.Event{
				Operation:       "sync.sender",
				Action:          "message.suspend",
				Result:          "rejected",
				ErrorCode:       "SEND_NON_RETRYABLE",
				NodeDeviceID:    msg.NodeDeviceID,
				ClientDeviceID:  derefOptional(msg.ClientDeviceID),
				SessionID:       derefOptional(msg.SessionID),
				ActorEmployeeID: derefOptional(msg.ActorEmployeeID),
			}, "outbox_id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType, "error", reason)
		case BatchSendRetryable:
			hadRetryable = true
			reason := strings.TrimSpace(item.Reason)
			if reason == "" {
				reason = "cloud retryable sync event item failure"
			}
			if err := w.service.MarkOutboxRetryableFailure(ctx, msg.ID, reason); err != nil {
				return fmt.Errorf("mark retryable outbox failure %s: %w", msg.ID, err)
			}
			platformlog.Log(ctx, w.logger, platformlog.LevelTrace, "POS sync sender retry decision", platformlog.Event{
				Operation:       "sync.sender",
				Action:          "message.retryable_failure",
				Result:          "attempt",
				ErrorCode:       "SEND_RETRYABLE",
				NodeDeviceID:    msg.NodeDeviceID,
				ClientDeviceID:  derefOptional(msg.ClientDeviceID),
				SessionID:       derefOptional(msg.SessionID),
				ActorEmployeeID: derefOptional(msg.ActorEmployeeID),
			}, "outbox_id", msg.ID, "sequence_no", msg.SequenceNo, "event_type", msg.CommandType, "error", reason)
		default:
			hadRetryable = true
			if err := w.service.MarkOutboxRetryableFailure(ctx, msg.ID, "cloud returned unknown batch item status"); err != nil {
				return fmt.Errorf("mark unknown batch status retryable %s: %w", msg.ID, err)
			}
		}
	}
	if hadRetryable {
		return fmt.Errorf("batch send returned retryable item failures")
	}
	return nil
}

func blockedDirectionReason(msg domain.OutboxMessage) string {
	if msg.Origin != domain.OriginEdgeDevice {
		return fmt.Sprintf("sync direction blocked: origin %q is not Edge runtime origin", msg.Origin)
	}
	if msg.SyncDirection != "" && msg.SyncDirection != domain.SyncDirectionEdgeToCloud {
		return fmt.Sprintf("sync direction blocked: outbox row direction is %q", msg.SyncDirection)
	}
	if !domain.IsEdgeToCloudOperationalEvent(msg.CommandType) {
		return fmt.Sprintf("sync direction blocked: %s is Cloud-managed/configuration or unsupported Edge->Cloud event", msg.CommandType)
	}
	return ""
}

func derefOptional(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func (w *Worker) nextPollDelay() time.Duration {
	if w.config.PollJitter <= 0 {
		return w.config.PollInterval
	}
	return w.config.PollInterval + time.Duration(rand.Int63n(w.config.PollJitter.Nanoseconds()+1))
}
