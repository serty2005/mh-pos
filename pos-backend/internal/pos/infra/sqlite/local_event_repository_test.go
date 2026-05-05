package sqlite_test

import (
	"context"
	"testing"
	"time"

	"pos-backend/internal/pos/domain"
	possqlite "pos-backend/internal/pos/infra/sqlite"
)

func TestListLocalEventsOrdersByCreatedAtAndIDDescending(t *testing.T) {
	db, ctx := newSchemaDB(t)
	repo := possqlite.NewRepository(db)
	base := time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)

	insertLocalEvent(t, repo, ctx, domain.LocalEvent{ID: "local-001", EventID: "event-001", EventType: "OrderCreated", CreatedAt: base, OccurredAt: base})
	insertLocalEvent(t, repo, ctx, domain.LocalEvent{ID: "local-003", EventID: "event-003", EventType: "PaymentCaptured", CreatedAt: base.Add(time.Minute), OccurredAt: base.Add(time.Minute)})
	insertLocalEvent(t, repo, ctx, domain.LocalEvent{ID: "local-002", EventID: "event-002", EventType: "OrderCreated", CreatedAt: base.Add(time.Minute), OccurredAt: base.Add(time.Minute)})

	events, err := repo.ListLocalEvents(ctx, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("expected %d events, got %d", want, got)
	}
	if events[0].ID != "local-003" || events[1].ID != "local-002" {
		t.Fatalf("unexpected order: got %s, %s", events[0].ID, events[1].ID)
	}
}

func TestListLocalEventsFiltersByEventType(t *testing.T) {
	db, ctx := newSchemaDB(t)
	repo := possqlite.NewRepository(db)
	base := time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)

	insertLocalEvent(t, repo, ctx, domain.LocalEvent{ID: "local-001", EventID: "event-001", EventType: "OrderCreated", CreatedAt: base, OccurredAt: base})
	insertLocalEvent(t, repo, ctx, domain.LocalEvent{ID: "local-002", EventID: "event-002", EventType: "PaymentCaptured", CreatedAt: base.Add(time.Minute), OccurredAt: base.Add(time.Minute)})
	insertLocalEvent(t, repo, ctx, domain.LocalEvent{ID: "local-003", EventID: "event-003", EventType: "OrderCreated", CreatedAt: base.Add(2 * time.Minute), OccurredAt: base.Add(2 * time.Minute)})

	events, err := repo.ListLocalEvents(ctx, 50, "OrderCreated")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("expected %d events, got %d", want, got)
	}
	for _, event := range events {
		if event.EventType != "OrderCreated" {
			t.Fatalf("expected only OrderCreated events, got %s", event.EventType)
		}
	}
	if events[0].ID != "local-003" || events[1].ID != "local-001" {
		t.Fatalf("unexpected filtered order: got %s, %s", events[0].ID, events[1].ID)
	}
}

func insertLocalEvent(t *testing.T, repo *possqlite.Repository, ctx context.Context, event domain.LocalEvent) {
	t.Helper()
	if event.EnvelopeVersion == "" {
		event.EnvelopeVersion = domain.SyncEnvelopeVersion
	}
	if event.CommandID == "" {
		event.CommandID = "cmd-" + event.ID
	}
	if event.AggregateType == "" {
		event.AggregateType = "Order"
	}
	if event.AggregateID == "" {
		event.AggregateID = event.ID
	}
	if event.DeviceID == "" {
		event.DeviceID = "device-1"
	}
	if event.PayloadJSON == "" {
		event.PayloadJSON = `{}`
	}
	if err := repo.CreateLocalEvent(ctx, &event); err != nil {
		t.Fatal(err)
	}
}
