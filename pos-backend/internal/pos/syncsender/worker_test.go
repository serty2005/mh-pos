package syncsender

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"pos-backend/internal/pos/app"
	"pos-backend/internal/pos/domain"
)

type fakeOutboxService struct {
	claimed       []domain.OutboxMessage
	sent          []string
	retryable     []string
	suspended     map[string]string
	releasedLocks []string
	exchangeState SyncExchangeState
	applied       []CloudPackage
	applyErr      error
}

func (f *fakeOutboxService) ClaimPendingOutbox(context.Context, app.ClaimPendingOutboxCommand) ([]domain.OutboxMessage, error) {
	return append([]domain.OutboxMessage(nil), f.claimed...), nil
}

func (f *fakeOutboxService) ReclaimStaleProcessingOutbox(context.Context, app.ReclaimStaleOutboxCommand) (int, error) {
	return 0, nil
}

func (f *fakeOutboxService) ReleaseProcessingOutbox(_ context.Context, lockedBy string) (int, error) {
	f.releasedLocks = append(f.releasedLocks, lockedBy)
	return 1, nil
}

func (f *fakeOutboxService) MarkOutboxSent(_ context.Context, id string) error {
	f.sent = append(f.sent, id)
	return nil
}

func (f *fakeOutboxService) MarkOutboxRetryableFailure(_ context.Context, id, _ string) error {
	f.retryable = append(f.retryable, id)
	return nil
}

func (f *fakeOutboxService) SuspendOutboxMessage(_ context.Context, id, reason string) error {
	if f.suspended == nil {
		f.suspended = map[string]string{}
	}
	f.suspended[id] = reason
	return nil
}

func (f *fakeOutboxService) RefreshSyncExchangeState(context.Context) error {
	return nil
}

func (f *fakeOutboxService) GetSyncExchangeState(context.Context) (SyncExchangeState, error) {
	if f.exchangeState.NodeDeviceID == "" {
		f.exchangeState = SyncExchangeState{
			NodeDeviceID: "node-1",
			RestaurantID: "restaurant-1",
			AuthToken:    "node-token",
			Streams:      []SyncExchangeStreamRequest{{StreamName: "catalog", LastCloudVersion: 1, CheckpointToken: "catalog:1"}},
		}
	}
	return f.exchangeState, nil
}

func (f *fakeOutboxService) ApplySyncExchangeCloudPackages(_ context.Context, packages []CloudPackage) error {
	f.applied = append(f.applied, packages...)
	return f.applyErr
}

type fakeSender struct {
	err error
}

func (s fakeSender) Send(context.Context, domain.OutboxMessage) error {
	return s.err
}

type fakeBatchSender struct {
	results []BatchSendResult
	err     error
}

type fakeExchangeSender struct {
	requests  []SyncExchangeRequest
	response  SyncExchangeResponse
	err       error
	sendCalls int
}

func (s *fakeExchangeSender) Send(context.Context, domain.OutboxMessage) error {
	s.sendCalls++
	return nil
}

func (s *fakeExchangeSender) Exchange(_ context.Context, req SyncExchangeRequest) (SyncExchangeResponse, error) {
	s.requests = append(s.requests, req)
	if s.err != nil {
		return SyncExchangeResponse{}, s.err
	}
	return s.response, nil
}

func (s fakeBatchSender) Send(context.Context, domain.OutboxMessage) error {
	return nil
}

func (s fakeBatchSender) SendBatch(context.Context, []domain.OutboxMessage) ([]BatchSendResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]BatchSendResult(nil), s.results...), nil
}

func TestRunOnceSuspendsWrongDirectionMessageAndContinues(t *testing.T) {
	service := &fakeOutboxService{claimed: []domain.OutboxMessage{
		{ID: "outbox-config", SequenceNo: 1, Origin: domain.OriginSystemSeed, CommandType: "RestaurantCreated"},
		{ID: "outbox-order", SequenceNo: 2, Origin: domain.OriginEdgeDevice, CommandType: "OrderCreated", PayloadJSON: `{}`},
	}}
	worker := NewWorker(service, fakeSender{}, Config{WorkerID: "worker-test", PollInterval: time.Hour}, nil)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if service.suspended["outbox-config"] == "" {
		t.Fatal("expected wrong-direction message to be suspended")
	}
	if len(service.sent) != 1 || service.sent[0] != "outbox-order" {
		t.Fatalf("expected operational message to be sent, got %v", service.sent)
	}
}

func TestRunOnceSuspendsCloudToEdgeLocalOnlyAndUnsupportedRows(t *testing.T) {
	service := &fakeOutboxService{claimed: []domain.OutboxMessage{
		{ID: "outbox-cloud", SequenceNo: 1, Origin: domain.OriginCloudSync, SyncDirection: domain.SyncDirectionCloudToEdge, CommandType: "RestaurantCreated"},
		{ID: "outbox-local", SequenceNo: 2, Origin: domain.OriginEdgeDevice, SyncDirection: domain.SyncDirectionLocalOnly, CommandType: "LocalDiagnosticRecorded"},
		{ID: "outbox-wrong-direction", SequenceNo: 3, Origin: domain.OriginEdgeDevice, SyncDirection: domain.SyncDirectionCloudToEdge, CommandType: "OrderCreated"},
		{ID: "outbox-order", SequenceNo: 4, Origin: domain.OriginEdgeDevice, SyncDirection: domain.SyncDirectionEdgeToCloud, CommandType: "OrderCreated", PayloadJSON: `{}`},
	}}
	worker := NewWorker(service, fakeSender{}, Config{WorkerID: "worker-test", PollInterval: time.Hour}, nil)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"outbox-cloud", "outbox-local", "outbox-wrong-direction"} {
		if service.suspended[id] == "" {
			t.Fatalf("expected %s to be suspended, got suspended=%v", id, service.suspended)
		}
	}
	if len(service.sent) != 1 || service.sent[0] != "outbox-order" {
		t.Fatalf("expected only operational Edge->Cloud message to be sent, got %v", service.sent)
	}
}

func TestRunOnceMarksRetryableFailureAndReleasesRemainingBatch(t *testing.T) {
	service := &fakeOutboxService{claimed: []domain.OutboxMessage{
		{ID: "outbox-order", SequenceNo: 1, Origin: domain.OriginEdgeDevice, CommandType: "OrderCreated", PayloadJSON: `{}`},
		{ID: "outbox-payment", SequenceNo: 2, Origin: domain.OriginEdgeDevice, CommandType: "PaymentCaptured", PayloadJSON: `{}`},
	}}
	worker := NewWorker(service, fakeSender{err: errors.New("network down")}, Config{WorkerID: "worker-test", PollInterval: time.Hour}, nil)

	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected retryable send error")
	}
	if len(service.retryable) != 1 || service.retryable[0] != "outbox-order" {
		t.Fatalf("expected first message to be marked retryable, got %v", service.retryable)
	}
	if len(service.releasedLocks) != 1 || service.releasedLocks[0] != "worker-test" {
		t.Fatalf("expected remaining batch locks to be released, got %v", service.releasedLocks)
	}
}

func TestDirectionFoundationKeepsDeviceRegisteredOperational(t *testing.T) {
	direction := domain.DirectionForOutbox(domain.OriginEdgeDevice, "Device", "DeviceRegistered")
	if direction != domain.SyncDirectionEdgeToCloud {
		t.Fatalf("expected DeviceRegistered to stay edge_to_cloud, got %s", direction)
	}
}

func TestDirectionKeepsRefundRecordedOperational(t *testing.T) {
	direction := domain.DirectionForOutbox(domain.OriginEdgeDevice, "FinancialOperation", "RefundRecorded")
	if direction != domain.SyncDirectionEdgeToCloud {
		t.Fatalf("expected RefundRecorded to stay edge_to_cloud, got %s", direction)
	}
}

func TestRunOnceWritesNormalizedTelemetryFields(t *testing.T) {
	clientID := "client-device-123456"
	sessionID := "session-123456"
	actorID := "employee-123456"
	service := &fakeOutboxService{claimed: []domain.OutboxMessage{
		{
			ID:              "outbox-order",
			SequenceNo:      1,
			Origin:          domain.OriginEdgeDevice,
			SyncDirection:   domain.SyncDirectionEdgeToCloud,
			CommandType:     "OrderCreated",
			NodeDeviceID:    "node-device-123456",
			ClientDeviceID:  &clientID,
			SessionID:       &sessionID,
			ActorEmployeeID: &actorID,
		},
	}}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.Level(-8)}))
	worker := NewWorker(service, fakeSender{}, Config{WorkerID: "worker-test", PollInterval: time.Hour}, logger)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	raw := logs.String()
	for _, want := range []string{
		`"operation":"sync.sender"`,
		`"action":"message.ack"`,
		`"result":"success"`,
		`"error_code":""`,
		`"node_device_id":"node-dev..."`,
		`"client_device_id":"client-d..."`,
		`"session_id":"session-..."`,
		`"actor_employee_id":"employee..."`,
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("expected telemetry field %q in logs: %s", want, raw)
		}
	}
	if strings.Contains(raw, "pin") || strings.Contains(raw, "manager_pin") {
		t.Fatalf("expected no sensitive auth fields in logs, got: %s", raw)
	}
}

func TestRunOnceProcessesBatchItemLevelAck(t *testing.T) {
	service := &fakeOutboxService{claimed: []domain.OutboxMessage{
		{ID: "outbox-1", SequenceNo: 1, Origin: domain.OriginEdgeDevice, SyncDirection: domain.SyncDirectionEdgeToCloud, CommandType: "OrderCreated", PayloadJSON: `{"event_id":"e1"}`},
		{ID: "outbox-2", SequenceNo: 2, Origin: domain.OriginEdgeDevice, SyncDirection: domain.SyncDirectionEdgeToCloud, CommandType: "OrderCreated", PayloadJSON: `{"event_id":"e2"}`},
		{ID: "outbox-3", SequenceNo: 3, Origin: domain.OriginEdgeDevice, SyncDirection: domain.SyncDirectionEdgeToCloud, CommandType: "OrderCreated", PayloadJSON: `{"event_id":"e3"}`},
	}}
	worker := NewWorker(service, fakeBatchSender{
		results: []BatchSendResult{
			{OutboxID: "outbox-1", Status: BatchSendAccepted},
			{OutboxID: "outbox-2", Status: BatchSendRejected, Reason: "bad envelope"},
			{OutboxID: "outbox-3", Status: BatchSendRetryable, Reason: "cloud temporary"},
		},
	}, Config{WorkerID: "worker-test", PollInterval: time.Hour}, nil)

	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected retryable batch result error")
	}
	if len(service.sent) != 1 || service.sent[0] != "outbox-1" {
		t.Fatalf("expected one sent item, got sent=%v", service.sent)
	}
	if service.suspended["outbox-2"] == "" {
		t.Fatalf("expected outbox-2 suspended, suspended=%v", service.suspended)
	}
	if len(service.retryable) != 1 || service.retryable[0] != "outbox-3" {
		t.Fatalf("expected outbox-3 retryable, got retryable=%v", service.retryable)
	}
}

func TestRunOnceUsesExchangeAppliesPackageThenMarksAck(t *testing.T) {
	service := &fakeOutboxService{claimed: []domain.OutboxMessage{
		{ID: "outbox-1", SequenceNo: 1, Origin: domain.OriginEdgeDevice, SyncDirection: domain.SyncDirectionEdgeToCloud, CommandType: "OrderCreated", PayloadJSON: `{"event_id":"event-1"}`},
	}}
	sender := &fakeExchangeSender{response: SyncExchangeResponse{
		Status:   SyncExchangeStatusAccepted,
		EdgeAcks: []BatchSendResult{{OutboxID: "outbox-1", Status: BatchSendAccepted}},
		CloudPackages: []CloudPackage{{
			StreamName:      "catalog",
			SyncMode:        "incremental",
			CloudVersion:    2,
			CheckpointToken: "catalog:2",
			PayloadJSON:     json.RawMessage(`{"catalog_items":[{"id":"cat-1","name":"Tea"}]}`),
		}},
	}}
	worker := NewWorker(service, sender, Config{WorkerID: "worker-test", PollInterval: time.Hour}, nil)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sender.requests) != 1 || len(sender.requests[0].EdgeEvents) != 1 || sender.requests[0].EdgeEvents[0].ClientItemID != "outbox-1" {
		t.Fatalf("expected one exchange request with outbox payload, got %+v", sender.requests)
	}
	if len(sender.requests[0].Streams) != 1 || sender.requests[0].Streams[0].LastCloudVersion != 1 {
		t.Fatalf("expected local stream checkpoint in request, got %+v", sender.requests[0].Streams)
	}
	if len(service.applied) != 1 || service.applied[0].CloudVersion != 2 {
		t.Fatalf("expected cloud package applied before ack, got %+v", service.applied)
	}
	if len(service.sent) != 1 || service.sent[0] != "outbox-1" {
		t.Fatalf("expected outbox marked sent after apply, got %+v", service.sent)
	}
	if sender.sendCalls != 0 {
		t.Fatalf("legacy Send must not be used when exchange is available")
	}
}

func TestRunOnceDoesNotMarkAckWhenExchangePackageApplyFails(t *testing.T) {
	service := &fakeOutboxService{
		claimed:  []domain.OutboxMessage{{ID: "outbox-1", SequenceNo: 1, Origin: domain.OriginEdgeDevice, SyncDirection: domain.SyncDirectionEdgeToCloud, CommandType: "OrderCreated", PayloadJSON: `{"event_id":"event-1"}`}},
		applyErr: errors.New("apply failed"),
	}
	sender := &fakeExchangeSender{response: SyncExchangeResponse{
		Status:        SyncExchangeStatusAccepted,
		EdgeAcks:      []BatchSendResult{{OutboxID: "outbox-1", Status: BatchSendAccepted}},
		CloudPackages: []CloudPackage{{StreamName: "catalog", SyncMode: "incremental", CloudVersion: 2, PayloadJSON: json.RawMessage(`{"catalog_items":[{"id":"cat-1"}]}`)}},
	}}
	worker := NewWorker(service, sender, Config{WorkerID: "worker-test", PollInterval: time.Hour}, nil)

	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected apply failure")
	}
	if len(service.sent) != 0 {
		t.Fatalf("outbox must not be marked sent when package apply fails, got %+v", service.sent)
	}
	if len(service.retryable) != 1 || service.retryable[0] != "outbox-1" {
		t.Fatalf("expected claimed item marked retryable for safe idempotent replay, got %+v", service.retryable)
	}
}

func TestRunOnceThrottlesEmptyExchangePulls(t *testing.T) {
	service := &fakeOutboxService{}
	sender := &fakeExchangeSender{response: SyncExchangeResponse{Status: SyncExchangeStatusAccepted}}
	worker := NewWorker(service, sender, Config{WorkerID: "worker-test", PollInterval: time.Hour, CloudPullInterval: time.Hour}, nil)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sender.requests) != 1 {
		t.Fatalf("expected one empty exchange pull inside throttle window, got %d", len(sender.requests))
	}
	if len(sender.requests[0].EdgeEvents) != 0 {
		t.Fatalf("expected throttled request to be cloud pull only, got %+v", sender.requests[0].EdgeEvents)
	}
}

func TestNextPollDelayWithoutJitterUsesBaseInterval(t *testing.T) {
	worker := NewWorker(&fakeOutboxService{}, fakeSender{}, Config{PollInterval: 30 * time.Second, PollJitter: 0}, nil)
	if got := worker.nextPollDelay(); got != 30*time.Second {
		t.Fatalf("expected exact interval, got %s", got)
	}
}

func TestNextPollDelayWithJitterStaysWithinBounds(t *testing.T) {
	worker := NewWorker(&fakeOutboxService{}, fakeSender{}, Config{PollInterval: 30 * time.Second, PollJitter: 3 * time.Second}, nil)
	for i := 0; i < 100; i++ {
		got := worker.nextPollDelay()
		if got < 30*time.Second || got > 33*time.Second {
			t.Fatalf("expected delay in [30s,33s], got %s", got)
		}
	}
}
