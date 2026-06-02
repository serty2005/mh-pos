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

func TestBackfillWorkerReturnsUnavailableWhenDependenciesAreMissing(t *testing.T) {
	tests := []struct {
		name   string
		worker *app.BackfillWorker
	}{
		{name: "nil worker", worker: nil},
		{name: "missing queue", worker: app.NewBackfillWorker(nil, &backfillExporter{}, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})},
		{name: "missing exporter", worker: app.NewBackfillWorker(&backfillQueueRepo{}, nil, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.worker.RunOnce(context.Background()); !errors.Is(err, app.ErrOLAPUnavailable) {
				t.Fatalf("expected ErrOLAPUnavailable, got %v", err)
			}
		})
	}
}

func TestBackfillWorkerSkipsExportWhenNoJobClaimed(t *testing.T) {
	repo := &backfillQueueRepo{claimOK: false}
	exporter := &backfillExporter{}
	worker := app.NewBackfillWorker(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repo.workerID != "worker-1" {
		t.Fatalf("expected worker id worker-1, got %q", repo.workerID)
	}
	if repo.loadCalls != 0 {
		t.Fatalf("did not expect batch load, got %d calls", repo.loadCalls)
	}
	if len(exporter.rawEvents) != 0 || len(exporter.stockMoves) != 0 {
		t.Fatalf("did not expect exports, got raw=%+v stock=%+v", exporter.rawEvents, exporter.stockMoves)
	}
	if len(repo.progressCalls) != 0 || len(repo.failedCalls) != 0 {
		t.Fatalf("did not expect markers, got progress=%+v failed=%+v", repo.progressCalls, repo.failedCalls)
	}
}

func TestBackfillWorkerReturnsClaimError(t *testing.T) {
	claimErr := errors.New("claim failed")
	repo := &backfillQueueRepo{claimErr: claimErr}
	worker := app.NewBackfillWorker(repo, &backfillExporter{}, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); !errors.Is(err, claimErr) {
		t.Fatalf("expected claim error, got %v", err)
	}
}

func TestBackfillWorkerMarksFailedWhenLoadBatchFails(t *testing.T) {
	loadErr := errors.New("postgres load failed")
	repo := &backfillQueueRepo{
		claimOK: true,
		job:     app.BackfillJob{ID: "job-1", Stream: "raw_business_events"},
		loadErr: loadErr,
	}
	exporter := &backfillExporter{}
	worker := app.NewBackfillWorker(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.failedCalls) != 1 || repo.failedCalls[0].reason != "postgres load failed" {
		t.Fatalf("expected failed marker with safe reason, got %+v", repo.failedCalls)
	}
	if len(repo.progressCalls) != 0 {
		t.Fatalf("did not expect progress marker, got %+v", repo.progressCalls)
	}
	if len(exporter.rawEvents) != 0 || len(exporter.stockMoves) != 0 {
		t.Fatalf("did not expect exports, got raw=%+v stock=%+v", exporter.rawEvents, exporter.stockMoves)
	}
}

func TestBackfillWorkerMarksProgressForEmptyBatch(t *testing.T) {
	recorder := &backfillRecorder{}
	repo := &backfillQueueRepo{
		recorder: recorder,
		claimOK:  true,
		job:      app.BackfillJob{ID: "job-1", Stream: "raw_business_events"},
	}
	exporter := &backfillExporter{recorder: recorder}
	worker := app.NewBackfillWorker(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.progressCalls) != 1 {
		t.Fatalf("expected progress marker, got %+v", repo.progressCalls)
	}
	if len(repo.failedCalls) != 0 {
		t.Fatalf("did not expect failed marker, got %+v", repo.failedCalls)
	}
	requireBackfillCalls(t, recorder.calls, []string{"claim", "load", "progress"})
}

func TestBackfillWorkerExportsRawEventsAndMarksProgress(t *testing.T) {
	recorder := &backfillRecorder{}
	repo := &backfillQueueRepo{
		recorder: recorder,
		claimOK:  true,
		job:      app.BackfillJob{ID: "job-1", Stream: "raw_business_events"},
		batch:    app.BackfillBatch{RawEvents: []app.InboxEvent{{ID: "inbox-1", EventID: "event-1"}}},
	}
	exporter := &backfillExporter{recorder: recorder}
	worker := app.NewBackfillWorker(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(exporter.rawEvents) != 1 || exporter.rawEvents[0].ID != "inbox-1" {
		t.Fatalf("expected raw export, got %+v", exporter.rawEvents)
	}
	if len(repo.progressCalls) != 1 {
		t.Fatalf("expected progress marker, got %+v", repo.progressCalls)
	}
	requireBackfillCalls(t, recorder.calls, []string{"claim", "load", "export_raw", "progress"})
}

func TestBackfillWorkerExportsStockMovesAndMarksProgress(t *testing.T) {
	recorder := &backfillRecorder{}
	repo := &backfillQueueRepo{
		recorder: recorder,
		claimOK:  true,
		job:      app.BackfillJob{ID: "job-1", Stream: "stock_moves"},
		batch:    app.BackfillBatch{StockMoves: []app.StockMove{{LedgerEntryID: "ledger-1", SourceEventID: "event-1"}}},
	}
	exporter := &backfillExporter{recorder: recorder}
	worker := app.NewBackfillWorker(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(exporter.stockMoves) != 1 || exporter.stockMoves[0].LedgerEntryID != "ledger-1" {
		t.Fatalf("expected stock move export, got %+v", exporter.stockMoves)
	}
	if len(repo.progressCalls) != 1 {
		t.Fatalf("expected progress marker, got %+v", repo.progressCalls)
	}
	requireBackfillCalls(t, recorder.calls, []string{"claim", "load", "export_stock", "progress"})
}

func TestBackfillWorkerMarksFailedWhenExportFails(t *testing.T) {
	tests := []struct {
		name         string
		batch        app.BackfillBatch
		rawErr       error
		stockErr     error
		wantReason   string
		wantCalls    []string
		wantRaw      int
		wantStock    int
		wantFailed   int
		wantProgress int
	}{
		{
			name:         "raw events",
			batch:        app.BackfillBatch{RawEvents: []app.InboxEvent{{ID: "inbox-1", EventID: "event-1"}}},
			rawErr:       errors.New(" clickhouse raw down "),
			wantReason:   "clickhouse raw down",
			wantCalls:    []string{"claim", "load", "export_raw", "failed"},
			wantRaw:      1,
			wantFailed:   1,
			wantProgress: 0,
		},
		{
			name:         "stock moves",
			batch:        app.BackfillBatch{StockMoves: []app.StockMove{{LedgerEntryID: "ledger-1", SourceEventID: "event-1"}}},
			stockErr:     errors.New("clickhouse stock down"),
			wantReason:   "clickhouse stock down",
			wantCalls:    []string{"claim", "load", "export_stock", "failed"},
			wantStock:    1,
			wantFailed:   1,
			wantProgress: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &backfillRecorder{}
			repo := &backfillQueueRepo{
				recorder: recorder,
				claimOK:  true,
				job:      app.BackfillJob{ID: "job-1", Stream: "raw_business_events"},
				batch:    tt.batch,
			}
			exporter := &backfillExporter{recorder: recorder, rawErr: tt.rawErr, stockErr: tt.stockErr}
			worker := app.NewBackfillWorker(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

			if err := worker.RunOnce(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(exporter.rawEvents) != tt.wantRaw || len(exporter.stockMoves) != tt.wantStock {
				t.Fatalf("unexpected exports, got raw=%+v stock=%+v", exporter.rawEvents, exporter.stockMoves)
			}
			if len(repo.failedCalls) != tt.wantFailed || repo.failedCalls[0].reason != tt.wantReason {
				t.Fatalf("expected failed marker with reason %q, got %+v", tt.wantReason, repo.failedCalls)
			}
			if len(repo.progressCalls) != tt.wantProgress {
				t.Fatalf("unexpected progress markers, got %+v", repo.progressCalls)
			}
			requireBackfillCalls(t, recorder.calls, tt.wantCalls)
		})
	}
}

func TestBackfillWorkerReturnsMarkFailedErrorAfterExportFailure(t *testing.T) {
	markErr := errors.New("mark failed")
	tests := []struct {
		name       string
		batch      app.BackfillBatch
		rawErr     error
		stockErr   error
		wantReason string
	}{
		{
			name:       "raw events",
			batch:      app.BackfillBatch{RawEvents: []app.InboxEvent{{ID: "inbox-1", EventID: "event-1"}}},
			rawErr:     errors.New("clickhouse raw down"),
			wantReason: "clickhouse raw down",
		},
		{
			name:       "stock moves",
			batch:      app.BackfillBatch{StockMoves: []app.StockMove{{LedgerEntryID: "ledger-1", SourceEventID: "event-1"}}},
			stockErr:   errors.New("clickhouse stock down"),
			wantReason: "clickhouse stock down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &backfillQueueRepo{
				claimOK:       true,
				job:           app.BackfillJob{ID: "job-1", Stream: "raw_business_events"},
				batch:         tt.batch,
				markFailedErr: markErr,
			}
			exporter := &backfillExporter{rawErr: tt.rawErr, stockErr: tt.stockErr}
			worker := app.NewBackfillWorker(repo, exporter, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

			if err := worker.RunOnce(context.Background()); !errors.Is(err, markErr) {
				t.Fatalf("expected mark failed error, got %v", err)
			}
			if len(repo.failedCalls) != 1 || repo.failedCalls[0].reason != tt.wantReason {
				t.Fatalf("expected failed marker with export reason %q, got %+v", tt.wantReason, repo.failedCalls)
			}
		})
	}
}

func TestBackfillWorkerNormalizesJobBatchSize(t *testing.T) {
	tests := []struct {
		name      string
		batchSize int
		wantLimit int
	}{
		{name: "zero uses config", batchSize: 0, wantLimit: 10},
		{name: "negative uses config", batchSize: -5, wantLimit: 10},
		{name: "oversized uses config", batchSize: 50, wantLimit: 10},
		{name: "bounded job size is kept", batchSize: 7, wantLimit: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &backfillQueueRepo{
				claimOK: true,
				job:     app.BackfillJob{ID: "job-1", Stream: "raw_business_events", BatchSize: tt.batchSize},
			}
			worker := app.NewBackfillWorker(repo, &backfillExporter{}, fixedClock{}, app.ForwarderConfig{WorkerID: "worker-1", BatchSize: 10})

			if err := worker.RunOnce(context.Background()); err != nil {
				t.Fatal(err)
			}
			if repo.loadLimit != tt.wantLimit {
				t.Fatalf("expected load limit %d, got %d", tt.wantLimit, repo.loadLimit)
			}
		})
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

type backfillRecorder struct {
	calls []string
}

func (r *backfillRecorder) add(call string) {
	if r != nil {
		r.calls = append(r.calls, call)
	}
}

type backfillQueueRepo struct {
	recorder      *backfillRecorder
	job           app.BackfillJob
	claimOK       bool
	claimErr      error
	batch         app.BackfillBatch
	loadErr       error
	markFailedErr error
	workerID      string
	loadLimit     int
	loadCalls     int
	progressCalls []backfillProgressCall
	failedCalls   []backfillFailedCall
}

type backfillProgressCall struct {
	job   app.BackfillJob
	batch app.BackfillBatch
	now   time.Time
}

type backfillFailedCall struct {
	job    app.BackfillJob
	reason string
	now    time.Time
}

func (q *backfillQueueRepo) ClaimBackfillJob(_ context.Context, workerID string, _ time.Time) (app.BackfillJob, bool, error) {
	q.recorder.add("claim")
	q.workerID = workerID
	return q.job, q.claimOK, q.claimErr
}

func (q *backfillQueueRepo) LoadBackfillBatch(_ context.Context, job app.BackfillJob, limit int) (app.BackfillBatch, error) {
	q.recorder.add("load")
	q.job = job
	q.loadLimit = limit
	q.loadCalls++
	return q.batch, q.loadErr
}

func (q *backfillQueueRepo) MarkBackfillProgress(_ context.Context, job app.BackfillJob, batch app.BackfillBatch, now time.Time) error {
	q.recorder.add("progress")
	q.progressCalls = append(q.progressCalls, backfillProgressCall{job: job, batch: batch, now: now})
	return nil
}

func (q *backfillQueueRepo) MarkBackfillFailed(_ context.Context, job app.BackfillJob, reason string, now time.Time) error {
	q.recorder.add("failed")
	q.failedCalls = append(q.failedCalls, backfillFailedCall{job: job, reason: reason, now: now})
	return q.markFailedErr
}

type backfillExporter struct {
	recorder   *backfillRecorder
	rawEvents  []app.InboxEvent
	stockMoves []app.StockMove
	rawErr     error
	stockErr   error
}

func (e *backfillExporter) InsertRawBusinessEvents(_ context.Context, events []app.InboxEvent, _ time.Time) error {
	e.recorder.add("export_raw")
	e.rawEvents = append(e.rawEvents, events...)
	return e.rawErr
}

func (e *backfillExporter) InsertStockMoves(_ context.Context, moves []app.StockMove, _ time.Time) error {
	e.recorder.add("export_stock")
	e.stockMoves = append(e.stockMoves, moves...)
	return e.stockErr
}

func requireBackfillCalls(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected calls %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected calls %v, got %v", want, got)
		}
	}
}

var fixedNow = time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return fixedNow
}
