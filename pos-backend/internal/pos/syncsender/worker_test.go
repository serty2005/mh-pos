package syncsender

import (
	"context"
	"errors"
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

type fakeSender struct {
	err error
}

func (s fakeSender) Send(context.Context, domain.OutboxMessage) error {
	return s.err
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
