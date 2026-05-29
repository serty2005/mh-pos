package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud-backend/internal/olap/app"
)

func TestForwarderExportsClaimedInboxEventsAndMarksProcessed(t *testing.T) {
	repo := &queueRepo{events: []app.InboxEvent{{ID: "inbox-1", EventID: "event-1"}}}
	exporter := &exporter{}
	worker := app.NewForwarder(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(exporter.events) != 1 || exporter.events[0].ID != "inbox-1" {
		t.Fatalf("expected export batch, got %+v", exporter.events)
	}
	if len(repo.processed) != 1 || repo.processed[0].ID != "inbox-1" {
		t.Fatalf("expected processed marker, got %+v", repo.processed)
	}
	if len(repo.failed) != 0 {
		t.Fatalf("did not expect retry marker, got %+v", repo.failed)
	}
}

func TestForwarderSchedulesRetryWhenClickHouseExportFails(t *testing.T) {
	repo := &queueRepo{events: []app.InboxEvent{{ID: "inbox-1", EventID: "event-1"}}}
	exporter := &exporter{err: errors.New("clickhouse down")}
	worker := app.NewForwarder(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10, RetryDelay: 2 * time.Minute})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.processed) != 0 {
		t.Fatalf("did not expect processed marker, got %+v", repo.processed)
	}
	if len(repo.failed) != 1 || repo.failed[0].ID != "inbox-1" {
		t.Fatalf("expected retry marker, got %+v", repo.failed)
	}
	wantRetry := fixedNow.Add(2 * time.Minute)
	if !repo.nextRetry.Equal(wantRetry) {
		t.Fatalf("expected retry at %s, got %s", wantRetry, repo.nextRetry)
	}
}

func TestStockMoveForwarderExportsClaimedMovesAndMarksProcessed(t *testing.T) {
	repo := &stockMoveQueueRepo{moves: []app.StockMove{{LedgerEntryID: "ledger-1", SourceEventID: "event-1"}}}
	exporter := &stockMoveExporter{}
	worker := app.NewStockMoveForwarder(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(exporter.moves) != 1 || exporter.moves[0].LedgerEntryID != "ledger-1" {
		t.Fatalf("expected stock move export batch, got %+v", exporter.moves)
	}
	if len(repo.processed) != 1 || repo.processed[0].LedgerEntryID != "ledger-1" {
		t.Fatalf("expected processed stock move marker, got %+v", repo.processed)
	}
	if len(repo.failed) != 0 {
		t.Fatalf("did not expect retry marker, got %+v", repo.failed)
	}
}

func TestStockMoveForwarderSchedulesRetryWhenClickHouseExportFails(t *testing.T) {
	repo := &stockMoveQueueRepo{moves: []app.StockMove{{LedgerEntryID: "ledger-1", SourceEventID: "event-1"}}}
	exporter := &stockMoveExporter{err: errors.New("clickhouse down")}
	worker := app.NewStockMoveForwarder(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10, RetryDelay: 2 * time.Minute})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.processed) != 0 {
		t.Fatalf("did not expect processed marker, got %+v", repo.processed)
	}
	if len(repo.failed) != 1 || repo.failed[0].LedgerEntryID != "ledger-1" {
		t.Fatalf("expected retry marker, got %+v", repo.failed)
	}
	wantRetry := fixedNow.Add(2 * time.Minute)
	if !repo.nextRetry.Equal(wantRetry) {
		t.Fatalf("expected retry at %s, got %s", wantRetry, repo.nextRetry)
	}
}

type queueRepo struct {
	events    []app.InboxEvent
	processed []app.InboxEvent
	failed    []app.InboxEvent
	nextRetry time.Time
}

func (q *queueRepo) ClaimPending(_ context.Context, cmd app.ClaimCommand) ([]app.InboxEvent, error) {
	if cmd.LockedBy != "worker-1" {
		return nil, errors.New("unexpected worker id")
	}
	out := append([]app.InboxEvent(nil), q.events...)
	q.events = nil
	return out, nil
}

func (q *queueRepo) MarkProcessed(_ context.Context, events []app.InboxEvent, _ time.Time) error {
	q.processed = append(q.processed, events...)
	return nil
}

func (q *queueRepo) MarkFailed(_ context.Context, events []app.InboxEvent, _ string, nextRetry, _ time.Time) error {
	q.failed = append(q.failed, events...)
	q.nextRetry = nextRetry
	return nil
}

type exporter struct {
	events []app.InboxEvent
	err    error
}

func (e *exporter) InsertRawBusinessEvents(_ context.Context, events []app.InboxEvent, _ time.Time) error {
	e.events = append(e.events, events...)
	return e.err
}

type stockMoveQueueRepo struct {
	moves     []app.StockMove
	processed []app.StockMove
	failed    []app.StockMove
	nextRetry time.Time
}

func (q *stockMoveQueueRepo) ClaimPendingStockMoves(_ context.Context, cmd app.ClaimCommand) ([]app.StockMove, error) {
	if cmd.LockedBy != "worker-1" {
		return nil, errors.New("unexpected worker id")
	}
	out := append([]app.StockMove(nil), q.moves...)
	q.moves = nil
	return out, nil
}

func (q *stockMoveQueueRepo) MarkStockMovesProcessed(_ context.Context, moves []app.StockMove, _ time.Time) error {
	q.processed = append(q.processed, moves...)
	return nil
}

func (q *stockMoveQueueRepo) MarkStockMovesFailed(_ context.Context, moves []app.StockMove, _ string, nextRetry, _ time.Time) error {
	q.failed = append(q.failed, moves...)
	q.nextRetry = nextRetry
	return nil
}

type stockMoveExporter struct {
	moves []app.StockMove
	err   error
}

func (e *stockMoveExporter) InsertStockMoves(_ context.Context, moves []app.StockMove, _ time.Time) error {
	e.moves = append(e.moves, moves...)
	return e.err
}

var fixedNow = time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return fixedNow
}
