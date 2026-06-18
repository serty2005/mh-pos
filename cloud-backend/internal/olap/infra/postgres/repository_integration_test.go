package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-backend/internal/cloudsync/contracts"
	cloudsyncpg "cloud-backend/internal/cloudsync/infra/postgres"
	"cloud-backend/internal/olap/app"
	platformpg "cloud-backend/internal/platform/postgres"
)

const (
	olapSecretMarker   = "OLAP_SECRET_MARKER"
	operatorReasonText = "operator reason should stay internal"
)

var olapBaseTime = time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

func TestRepositoryExportStatusCheckpointDefaultsAndSafeJSON(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openOLAPRepositoryWithBaseline(t, ctx)
	defer cleanup()

	raw, err := repo.GetExportStatus(ctx, "raw_business_events", olapBaseTime)
	if err != nil {
		t.Fatal(err)
	}
	if raw.Stream != "raw_business_events" || raw.LastCheckpoint != "" || raw.PendingCount != 0 || raw.ProcessingCount != 0 || raw.FailedCount != 0 || raw.RetryBlocked {
		t.Fatalf("unexpected empty raw status: %+v", raw)
	}
	assertPublicJSONDoesNotContain(t, raw, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)

	stock, err := repo.GetExportStatus(ctx, "stock_moves", olapBaseTime)
	if err != nil {
		t.Fatal(err)
	}
	if stock.Stream != "stock_moves" || stock.LastCheckpoint != "" || stock.PendingCount != 0 || stock.FailedCount != 0 || stock.RetryBlocked {
		t.Fatalf("unexpected empty stock status: %+v", stock)
	}

	lastExportedAt := olapBaseTime.Add(-time.Hour)
	nextRetry := olapBaseTime.Add(time.Hour)
	if _, err := pool.Exec(ctx, `
INSERT INTO olap_export_checkpoints(
  id,worker_id,last_exported_inbox_id,last_exported_event_id,last_exported_at,last_error,
  consecutive_failures,next_retry_at,updated_at
) VALUES ('raw_business_events','worker-1','inbox-010','event-010',$1,'clickhouse timeout',3,$2,$3)`,
		lastExportedAt, nextRetry, olapBaseTime.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-011", EventID: "event-011", Status: "pending", ReceivedAt: olapBaseTime.Add(time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-012", EventID: "event-012", Status: "processing", ReceivedAt: olapBaseTime.Add(2 * time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-013", EventID: "event-013", Status: "failed", ReceivedAt: olapBaseTime.Add(3 * time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-014", EventID: "event-014", Status: "processed", Processed: true, ReceivedAt: olapBaseTime.Add(4 * time.Minute)})

	raw, err = repo.GetExportStatus(ctx, "raw_business_events", olapBaseTime)
	if err != nil {
		t.Fatal(err)
	}
	if raw.LastCheckpoint != "inbox-010" || raw.LastExportedID != "event-010" || raw.LastExportedAt == nil || !raw.LastExportedAt.Equal(lastExportedAt) {
		t.Fatalf("unexpected checkpoint status: %+v", raw)
	}
	if raw.LastError != "clickhouse timeout" || raw.ConsecutiveFailures != 3 || raw.NextRetryAt == nil || !raw.NextRetryAt.Equal(nextRetry) || !raw.RetryBlocked {
		t.Fatalf("unexpected retry status: %+v", raw)
	}
	if raw.PendingCount != 1 || raw.ProcessingCount != 1 || raw.FailedCount != 1 {
		t.Fatalf("unexpected raw counters: %+v", raw)
	}
	assertPublicJSONDoesNotContain(t, raw, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)

	insertStockMove(t, ctx, pool, stockMoveFixture{ID: "ledger-001", SourceEventID: "stock-event-001", OccurredAt: olapBaseTime})
	insertStockMove(t, ctx, pool, stockMoveFixture{ID: "ledger-002", SourceEventID: "stock-event-002", OccurredAt: olapBaseTime.Add(time.Minute)})
	if _, err := pool.Exec(ctx, `
INSERT INTO olap_export_checkpoints(
  id,worker_id,last_exported_inbox_id,last_exported_event_id,last_exported_at,last_error,
  consecutive_failures,next_retry_at,updated_at
) VALUES ('olap_stock_moves','worker-1','ledger-001','stock-event-001',$1,'stock export timeout',2,$2,$3)`,
		lastExportedAt, nextRetry, olapBaseTime.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	stock, err = repo.GetExportStatus(ctx, "stock_moves", olapBaseTime)
	if err != nil {
		t.Fatal(err)
	}
	if stock.LastCheckpoint != "ledger-001" || stock.LastExportedID != "stock-event-001" || stock.PendingCount != 1 || stock.FailedCount != 2 || !stock.RetryBlocked {
		t.Fatalf("unexpected stock checkpoint status: %+v", stock)
	}
	assertPublicJSONDoesNotContain(t, stock, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)
}

func TestRepositoryRawBusinessEventsClaimProcessedAndFailed(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openOLAPRepositoryWithBaseline(t, ctx)
	defer cleanup()

	lockedAt := olapBaseTime.Add(-time.Minute)
	staleLockedAt := olapBaseTime.Add(-20 * time.Minute)
	futureRetry := olapBaseTime.Add(time.Hour)
	pastRetry := olapBaseTime.Add(-time.Minute)
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-001", EventID: "event-001", Status: "pending", ReceivedAt: olapBaseTime.Add(time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-002", EventID: "event-002", Status: "processed", Processed: true, ReceivedAt: olapBaseTime.Add(2 * time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-003", EventID: "event-003", Status: "processing", LockedBy: "worker-other", LockedAt: &lockedAt, ReceivedAt: olapBaseTime.Add(3 * time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-004", EventID: "event-004", Status: "failed", NextRetryAt: &futureRetry, ReceivedAt: olapBaseTime.Add(4 * time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-005", EventID: "event-005", Status: "failed", NextRetryAt: &pastRetry, ReceivedAt: olapBaseTime.Add(5 * time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-006", EventID: "event-006", Status: "processing", LockedBy: "worker-stale", LockedAt: &staleLockedAt, ReceivedAt: olapBaseTime.Add(6 * time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-007", EventID: "event-007", Status: "pending", ReceivedAt: olapBaseTime.Add(7 * time.Minute)})

	claimed, err := repo.ClaimPending(ctx, app.ClaimCommand{
		Limit:       2,
		LockedBy:    "worker-1",
		Now:         olapBaseTime,
		StaleBefore: olapBaseTime.Add(-5 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertInboxEventIDs(t, claimed, []string{"inbox-001", "inbox-005"})
	if string(claimed[0].RawPayload) == "" || !strings.Contains(string(claimed[0].RawPayload), olapSecretMarker) {
		t.Fatalf("worker claim should keep internal raw payload contract: %+v", claimed[0])
	}
	assertInboxLock(t, ctx, pool, "inbox-001", "processing", "worker-1", olapBaseTime, 1)
	assertInboxLock(t, ctx, pool, "inbox-005", "processing", "worker-1", olapBaseTime, 1)

	nextClaim, err := repo.ClaimPending(ctx, app.ClaimCommand{
		Limit:       10,
		LockedBy:    "worker-2",
		Now:         olapBaseTime,
		StaleBefore: olapBaseTime.Add(-5 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertInboxEventIDs(t, nextClaim, []string{"inbox-006", "inbox-007"})

	processedAt := olapBaseTime.Add(time.Minute)
	if err := repo.MarkProcessed(ctx, claimed, processedAt); err != nil {
		t.Fatal(err)
	}
	assertInboxProcessed(t, ctx, pool, "inbox-001")
	assertInboxProcessed(t, ctx, pool, "inbox-005")
	checkpoint := readCheckpoint(t, ctx, pool, "raw_business_events")
	if checkpoint.lastCheckpoint != "inbox-005" || checkpoint.lastExportedID != "event-005" || checkpoint.lastError != "" || checkpoint.consecutiveFailures != 0 || checkpoint.nextRetryAt != nil {
		t.Fatalf("unexpected processed checkpoint: %+v", checkpoint)
	}

	nextRetry := olapBaseTime.Add(30 * time.Minute)
	if err := repo.MarkFailed(ctx, nextClaim, "safe clickhouse timeout", nextRetry, olapBaseTime.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	assertInboxFailed(t, ctx, pool, "inbox-006", nextRetry, "safe clickhouse timeout")
	assertInboxFailed(t, ctx, pool, "inbox-007", nextRetry, "safe clickhouse timeout")
	checkpoint = readCheckpoint(t, ctx, pool, "raw_business_events")
	if checkpoint.lastError != "safe clickhouse timeout" || checkpoint.consecutiveFailures != 1 {
		t.Fatalf("unexpected failed checkpoint: %+v", checkpoint)
	}
	status, err := repo.GetExportStatus(ctx, "raw_business_events", olapBaseTime)
	if err != nil {
		t.Fatal(err)
	}
	assertPublicJSONDoesNotContain(t, status, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)
}

func TestRepositoryStockMovesClaimProcessedFailedAndRetry(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openOLAPRepositoryWithBaseline(t, ctx)
	defer cleanup()

	insertStockMove(t, ctx, pool, stockMoveFixture{ID: "ledger-001", SourceEventID: "stock-event-001", BusinessDateLocal: "2026-06-01", OccurredAt: olapBaseTime})
	insertStockMove(t, ctx, pool, stockMoveFixture{ID: "ledger-002", SourceEventID: "stock-event-002", BusinessDateLocal: "2026-06-02", OccurredAt: olapBaseTime.Add(time.Minute)})
	insertStockMove(t, ctx, pool, stockMoveFixture{ID: "ledger-003", SourceEventID: "stock-event-003", BusinessDateLocal: "2026-06-03", OccurredAt: olapBaseTime.Add(2 * time.Minute)})
	if _, err := pool.Exec(ctx, `
INSERT INTO olap_export_checkpoints(id,worker_id,last_exported_inbox_id,last_exported_event_id,last_exported_at,updated_at)
VALUES ('olap_stock_moves','','ledger-001','stock-event-001',$1,$1)`, olapBaseTime.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	moves, err := repo.ClaimPendingStockMoves(ctx, app.ClaimCommand{Limit: 1, LockedBy: "stock-worker", Now: olapBaseTime})
	if err != nil {
		t.Fatal(err)
	}
	assertStockMoveIDs(t, moves, []string{"ledger-002"})
	if moves[0].BusinessDateLocal != "2026-06-02" || moves[0].Quantity != "2.000" || moves[0].UnitCostMinor != 120 || moves[0].TotalCostMinor != 240 {
		t.Fatalf("unexpected stock move fields: %+v", moves[0])
	}

	processedAt := olapBaseTime.Add(5 * time.Minute)
	if err := repo.MarkStockMovesProcessed(ctx, moves, processedAt); err != nil {
		t.Fatal(err)
	}
	checkpoint := readCheckpoint(t, ctx, pool, "olap_stock_moves")
	if checkpoint.lastCheckpoint != "ledger-002" || checkpoint.lastExportedID != "stock-event-002" || checkpoint.lastError != "" || checkpoint.nextRetryAt != nil {
		t.Fatalf("unexpected stock processed checkpoint: %+v", checkpoint)
	}

	next, err := repo.ClaimPendingStockMoves(ctx, app.ClaimCommand{Limit: 10, Now: olapBaseTime.Add(6 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	assertStockMoveIDs(t, next, []string{"ledger-003"})
	nextRetry := olapBaseTime.Add(time.Hour)
	if err := repo.MarkStockMovesFailed(ctx, next, "safe stock timeout", nextRetry, olapBaseTime.Add(7*time.Minute)); err != nil {
		t.Fatal(err)
	}
	blocked, err := repo.ClaimPendingStockMoves(ctx, app.ClaimCommand{Limit: 10, Now: olapBaseTime.Add(8 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if len(blocked) != 0 {
		t.Fatalf("expected checkpoint retry block, got %+v", blocked)
	}
	checkpoint = readCheckpoint(t, ctx, pool, "olap_stock_moves")
	if checkpoint.lastCheckpoint != "ledger-002" || checkpoint.lastError != "safe stock timeout" || checkpoint.consecutiveFailures != 1 || checkpoint.nextRetryAt == nil || !checkpoint.nextRetryAt.Equal(nextRetry) {
		t.Fatalf("unexpected stock failed checkpoint: %+v", checkpoint)
	}

	result, err := repo.RequestExportRetry(ctx, app.ExportRetryCommand{
		CommandID: "018f0000-0000-7000-8000-000000000301",
		Stream:    "stock_moves",
		Mode:      "retry_failed",
		Reason:    operatorReasonText,
	}, olapBaseTime.Add(9*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if result.CheckpointBefore != "ledger-002" || result.PendingCount != 1 || result.FailedCount != 0 {
		t.Fatalf("unexpected stock retry result: %+v", result)
	}
	assertPublicJSONDoesNotContain(t, result, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)
	checkpoint = readCheckpoint(t, ctx, pool, "olap_stock_moves")
	if checkpoint.lastCheckpoint != "ledger-002" || checkpoint.lastError != "" || checkpoint.consecutiveFailures != 0 || checkpoint.nextRetryAt != nil {
		t.Fatalf("retry should clear stock backoff without moving checkpoint: %+v", checkpoint)
	}
}

func TestRepositoryExportRetryCommandIdempotencyAndConflict(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openOLAPRepositoryWithBaseline(t, ctx)
	defer cleanup()

	lockedAt := olapBaseTime
	nextRetry := olapBaseTime.Add(time.Hour)
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-101", EventID: "event-101", Status: "failed", NextRetryAt: &nextRetry, LastError: "safe failure", ReceivedAt: olapBaseTime})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-102", EventID: "event-102", Status: "failed", NextRetryAt: &nextRetry, LastError: "safe failure", ReceivedAt: olapBaseTime.Add(time.Minute)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-103", EventID: "event-103", Status: "processing", LockedBy: "worker-1", LockedAt: &lockedAt, ReceivedAt: olapBaseTime.Add(2 * time.Minute)})
	if _, err := pool.Exec(ctx, `
INSERT INTO olap_export_checkpoints(id,worker_id,last_exported_inbox_id,last_exported_event_id,last_exported_at,last_error,consecutive_failures,next_retry_at,updated_at)
VALUES ('raw_business_events','','inbox-100','event-100',$1,'safe failure',2,$2,$1)`, olapBaseTime.Add(-time.Minute), nextRetry); err != nil {
		t.Fatal(err)
	}

	cmd := app.ExportRetryCommand{
		CommandID: "018f0000-0000-7000-8000-000000000401",
		Stream:    "raw_business_events",
		Mode:      "retry_failed",
		Reason:    operatorReasonText,
	}
	first, err := repo.RequestExportRetry(ctx, cmd, olapBaseTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !first.Accepted || first.AlreadyProcessed || first.CheckpointBefore != "inbox-100" || first.PendingCount != 2 || first.FailedCount != 0 {
		t.Fatalf("unexpected retry result: %+v", first)
	}
	assertInboxRetryCleared(t, ctx, pool, "inbox-101", "pending")
	assertInboxRetryCleared(t, ctx, pool, "inbox-102", "pending")
	assertInboxLock(t, ctx, pool, "inbox-103", "processing", "worker-1", lockedAt, 0)
	assertExportRetryCommandCount(t, ctx, pool, cmd.CommandID, 1)
	assertPublicJSONDoesNotContain(t, first, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)

	replay, err := repo.RequestExportRetry(ctx, cmd, olapBaseTime.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !replay.AlreadyProcessed || replay.PendingCount != first.PendingCount || replay.FailedCount != first.FailedCount || !replay.RetryRequestedAt.Equal(first.RetryRequestedAt) {
		t.Fatalf("expected stable replay result: first=%+v replay=%+v", first, replay)
	}
	assertExportRetryCommandCount(t, ctx, pool, cmd.CommandID, 1)

	conflict := cmd
	conflict.Reason = "different operator reason"
	if _, err := repo.RequestExportRetry(ctx, conflict, olapBaseTime.Add(3*time.Minute)); !errors.Is(err, contracts.ErrPayloadConflict) {
		t.Fatalf("expected retry command conflict, got %v", err)
	}

	resume := app.ExportRetryCommand{
		CommandID: "018f0000-0000-7000-8000-000000000402",
		Stream:    "raw_business_events",
		Mode:      "resume_from_checkpoint",
		Reason:    operatorReasonText,
	}
	resumeResult, err := repo.RequestExportRetry(ctx, resume, olapBaseTime.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if resumeResult.CheckpointBefore != "inbox-100" {
		t.Fatalf("resume must report checkpoint before without resetting it: %+v", resumeResult)
	}
	assertInboxRetryCleared(t, ctx, pool, "inbox-103", "pending")
	checkpoint := readCheckpoint(t, ctx, pool, "raw_business_events")
	if checkpoint.lastCheckpoint != "inbox-100" || checkpoint.lastExportedID != "event-100" || checkpoint.lastError != "" || checkpoint.consecutiveFailures != 0 || checkpoint.nextRetryAt != nil {
		t.Fatalf("resume should clear retry state without resetting checkpoint cursor: %+v", checkpoint)
	}
}

func TestRepositoryBackfillCreateCancelIdempotencyAndSafeResults(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openOLAPRepositoryWithBaseline(t, ctx)
	defer cleanup()

	from := olapBaseTime.Add(-time.Hour)
	to := olapBaseTime.Add(time.Hour)
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-201", EventID: "event-201", Status: "pending", ReceivedAt: olapBaseTime.Add(-2 * time.Hour), OccurredAt: olapBaseTime.Add(-2 * time.Hour)})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-202", EventID: "event-202", Status: "pending", ReceivedAt: olapBaseTime, OccurredAt: olapBaseTime})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-203", EventID: "event-203", Status: "pending", ReceivedAt: olapBaseTime.Add(30 * time.Minute), OccurredAt: olapBaseTime.Add(30 * time.Minute)})
	insertStockMove(t, ctx, pool, stockMoveFixture{ID: "ledger-201", SourceEventID: "stock-event-201", OccurredAt: olapBaseTime})
	insertStockMove(t, ctx, pool, stockMoveFixture{ID: "ledger-202", SourceEventID: "stock-event-202", OccurredAt: olapBaseTime.Add(2 * time.Hour)})

	create := app.BackfillCreateCommand{
		CommandID:     "018f0000-0000-7000-8000-000000000501",
		Stream:        "raw_business_events",
		RequestedFrom: &from,
		RequestedTo:   &to,
		BatchSize:     2,
		Reason:        operatorReasonText,
		RequestedBy:   "support-1",
	}
	job, err := repo.CreateBackfillJob(ctx, create, olapBaseTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if job.ID != create.CommandID || job.Status != "queued" || job.TotalRows != 2 || job.BatchSize != 2 || job.AlreadyProcessed {
		t.Fatalf("unexpected created raw backfill job: %+v", job)
	}
	assertAuditEvent(t, ctx, pool, create.CommandID+":create", "create_backfill_job", job.ID, create.Reason)
	assertPublicJSONDoesNotContain(t, job, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)

	replay, err := repo.CreateBackfillJob(ctx, create, olapBaseTime.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !replay.AlreadyProcessed || replay.ID != job.ID || replay.TotalRows != job.TotalRows {
		t.Fatalf("expected idempotent create replay: first=%+v replay=%+v", job, replay)
	}
	conflict := create
	conflict.BatchSize = 3
	if _, err := repo.CreateBackfillJob(ctx, conflict, olapBaseTime.Add(3*time.Minute)); !errors.Is(err, contracts.ErrPayloadConflict) {
		t.Fatalf("expected conflicting backfill create, got %v", err)
	}

	stockJob, err := repo.CreateBackfillJob(ctx, app.BackfillCreateCommand{
		CommandID:     "018f0000-0000-7000-8000-000000000502",
		Stream:        "stock_moves",
		RequestedFrom: &from,
		RequestedTo:   &to,
		BatchSize:     100,
		Reason:        operatorReasonText,
		RequestedBy:   "support-1",
	}, olapBaseTime.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if stockJob.Stream != "stock_moves" || stockJob.TotalRows != 1 {
		t.Fatalf("unexpected stock backfill job: %+v", stockJob)
	}
	assertPublicJSONDoesNotContain(t, stockJob, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)

	cancelled, err := repo.CancelBackfillJob(ctx, app.BackfillCancelCommand{
		JobID:       job.ID,
		CommandID:   "018f0000-0000-7000-8000-000000000601",
		Reason:      operatorReasonText,
		RequestedBy: "support-2",
	}, olapBaseTime.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != "cancelled" || !cancelled.CancelRequested || cancelled.CompletedAt == nil {
		t.Fatalf("unexpected cancelled job: %+v", cancelled)
	}
	assertAuditEvent(t, ctx, pool, "018f0000-0000-7000-8000-000000000601:cancel", "cancel_backfill_job", job.ID, operatorReasonText)
	assertPublicJSONDoesNotContain(t, cancelled, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)

	replayedCancel, err := repo.CancelBackfillJob(ctx, app.BackfillCancelCommand{
		JobID:       job.ID,
		CommandID:   "018f0000-0000-7000-8000-000000000601",
		Reason:      operatorReasonText,
		RequestedBy: "support-2",
	}, olapBaseTime.Add(6*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if replayedCancel.Status != "cancelled" || replayedCancel.CompletedAt == nil || !replayedCancel.CompletedAt.Equal(*cancelled.CompletedAt) {
		t.Fatalf("expected idempotent cancel replay: first=%+v replay=%+v", cancelled, replayedCancel)
	}

	if _, err := pool.Exec(ctx, `UPDATE olap_backfill_jobs SET status = 'completed', completed_at = $2, updated_at = $2 WHERE id = $1`, stockJob.ID, olapBaseTime.Add(7*time.Minute)); err != nil {
		t.Fatal(err)
	}
	completed, err := repo.CancelBackfillJob(ctx, app.BackfillCancelCommand{
		JobID:       stockJob.ID,
		CommandID:   "018f0000-0000-7000-8000-000000000602",
		Reason:      operatorReasonText,
		RequestedBy: "support-2",
	}, olapBaseTime.Add(8*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "completed" || completed.CancelRequested {
		t.Fatalf("completed job must not be cancelled: %+v", completed)
	}
	assertPublicJSONDoesNotContain(t, completed, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)
}

func TestRepositoryBackfillClaimLoadProgressAndFailure(t *testing.T) {
	ctx := t.Context()
	pool, repo, cleanup := openOLAPRepositoryWithBaseline(t, ctx)
	defer cleanup()

	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-301", EventID: "event-301", Status: "pending", ReceivedAt: olapBaseTime, OccurredAt: olapBaseTime})
	insertInboxEvent(t, ctx, pool, inboxFixture{ID: "inbox-302", EventID: "event-302", Status: "pending", ReceivedAt: olapBaseTime.Add(time.Minute), OccurredAt: olapBaseTime.Add(time.Minute)})
	insertStockMove(t, ctx, pool, stockMoveFixture{ID: "ledger-301", SourceEventID: "stock-event-301", OccurredAt: olapBaseTime})
	insertStockMove(t, ctx, pool, stockMoveFixture{ID: "ledger-302", SourceEventID: "stock-event-302", OccurredAt: olapBaseTime.Add(time.Minute)})

	rawJob, err := repo.CreateBackfillJob(ctx, app.BackfillCreateCommand{
		CommandID:   "018f0000-0000-7000-8000-000000000701",
		Stream:      "raw_business_events",
		BatchSize:   1,
		Reason:      operatorReasonText,
		RequestedBy: "support-1",
	}, olapBaseTime)
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := repo.ClaimBackfillJob(ctx, "worker-1", olapBaseTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || claimed.ID != rawJob.ID || claimed.Status != "running" || claimed.StartedAt == nil {
		t.Fatalf("unexpected claimed job: ok=%v job=%+v", ok, claimed)
	}
	again, ok, err := repo.ClaimBackfillJob(ctx, "worker-2", olapBaseTime.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if ok || again.ID != "" {
		t.Fatalf("second worker must not claim running job: ok=%v job=%+v", ok, again)
	}

	batch, err := repo.LoadBackfillBatch(ctx, claimed, 1)
	if err != nil {
		t.Fatal(err)
	}
	assertInboxEventIDs(t, batch.RawEvents, []string{"inbox-301"})
	if !strings.Contains(string(batch.RawEvents[0].RawPayload), olapSecretMarker) {
		t.Fatalf("raw backfill worker batch should keep internal raw payload: %+v", batch.RawEvents[0])
	}
	if err := repo.MarkBackfillProgress(ctx, claimed, batch, olapBaseTime.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	stored := readBackfillJob(t, ctx, repo, rawJob.ID)
	if stored.Status != "running" || stored.ProcessedRows != 1 || stored.CheckpointCursor != "inbox-301" {
		t.Fatalf("unexpected raw progress: %+v", stored)
	}
	assertPublicJSONDoesNotContain(t, stored, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)

	secondBatch, err := repo.LoadBackfillBatch(ctx, stored, 1)
	if err != nil {
		t.Fatal(err)
	}
	assertInboxEventIDs(t, secondBatch.RawEvents, []string{"inbox-302"})
	if err := repo.MarkBackfillProgress(ctx, stored, secondBatch, olapBaseTime.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	stored = readBackfillJob(t, ctx, repo, rawJob.ID)
	if stored.Status != "running" || stored.ProcessedRows != 2 || stored.CheckpointCursor != "inbox-302" {
		t.Fatalf("unexpected second raw progress: %+v", stored)
	}

	empty, err := repo.LoadBackfillBatch(ctx, stored, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.RawEvents) != 0 {
		t.Fatalf("expected empty terminal raw batch, got %+v", empty)
	}
	if err := repo.MarkBackfillProgress(ctx, stored, empty, olapBaseTime.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	completed := readBackfillJob(t, ctx, repo, rawJob.ID)
	if completed.Status != "completed" || completed.CompletedAt == nil {
		t.Fatalf("expected completed raw job after terminal empty batch: %+v", completed)
	}

	stockJob, err := repo.CreateBackfillJob(ctx, app.BackfillCreateCommand{
		CommandID:   "018f0000-0000-7000-8000-000000000702",
		Stream:      "stock_moves",
		BatchSize:   2,
		Reason:      operatorReasonText,
		RequestedBy: "support-1",
	}, olapBaseTime.Add(6*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	stockBatch, err := repo.LoadBackfillBatch(ctx, stockJob, 1)
	if err != nil {
		t.Fatal(err)
	}
	assertStockMoveIDs(t, stockBatch.StockMoves, []string{"ledger-301"})
	if stockBatch.StockMoves[0].SourceEventID != "stock-event-301" || stockBatch.StockMoves[0].BusinessDateLocal == "" {
		t.Fatalf("unexpected stock backfill batch: %+v", stockBatch.StockMoves[0])
	}

	claimedStock, ok, err := repo.ClaimBackfillJob(ctx, "worker-stock", olapBaseTime.Add(7*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !ok || claimedStock.ID != stockJob.ID {
		t.Fatalf("expected stock job claim, ok=%v job=%+v", ok, claimedStock)
	}
	if err := repo.MarkBackfillFailed(ctx, claimedStock, "safe backfill failure", olapBaseTime.Add(8*time.Minute)); err != nil {
		t.Fatal(err)
	}
	failed := readBackfillJob(t, ctx, repo, stockJob.ID)
	if failed.Status != "failed" || failed.LastError != "safe backfill failure" {
		t.Fatalf("unexpected failed backfill job: %+v", failed)
	}
	assertPublicJSONDoesNotContain(t, failed, "raw_payload", "payload_json", olapSecretMarker, operatorReasonText)
}

type inboxFixture struct {
	ID          string
	EventID     string
	Status      string
	Processed   bool
	LockedBy    string
	LockedAt    *time.Time
	NextRetryAt *time.Time
	LastError   string
	OccurredAt  time.Time
	ReceivedAt  time.Time
}

func insertInboxEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, f inboxFixture) {
	t.Helper()
	if f.Status == "" {
		f.Status = "pending"
	}
	if f.EventID == "" {
		f.EventID = f.ID + "-event"
	}
	if f.OccurredAt.IsZero() {
		f.OccurredAt = olapBaseTime
	}
	if f.ReceivedAt.IsZero() {
		f.ReceivedAt = f.OccurredAt
	}
	raw := []byte(`{"payload":{"secret":"` + olapSecretMarker + `","event_id":"` + f.EventID + `"}}`)
	sha := sha256Hex(raw)
	if _, err := pool.Exec(ctx, `
INSERT INTO cloud_edge_event_receipts(
  id,idempotency_key,restaurant_id,device_id,command_id,event_id,edge_event_id,
  event_type,aggregate_type,aggregate_id,envelope_version,occurred_at,cloud_received_at,
  raw_payload_sha256_hex,created_at
) VALUES ($1,$2,'restaurant-1','device-1',$3,$4,$5,'OrderCreated','Order',$6,'1',$7,$8,$9,$8)`,
		f.ID,
		"idempotency-"+f.ID,
		"command-"+f.ID,
		f.EventID,
		"edge-"+f.EventID,
		"aggregate-"+f.ID,
		f.OccurredAt,
		f.ReceivedAt,
		sha,
	); err != nil {
		t.Fatalf("insert receipt %s: %v", f.ID, err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO inbox_events(
  id,receipt_id,idempotency_key,tenant_id,restaurant_id,device_id,employee_id,
  command_id,event_id,edge_event_id,event_type,aggregate_type,aggregate_id,envelope_version,
  occurred_at,cloud_received_at,raw_payload,raw_payload_sha256_hex,processed_for_olap,
  olap_export_status,olap_next_retry_at,olap_locked_at,olap_locked_by,olap_last_error,created_at,updated_at
) VALUES (
  $1,$1,$2,'tenant-1','restaurant-1','device-1','employee-1',
  $3,$4,$5,'OrderCreated','Order',$6,'1',
  $7,$8,$9::jsonb,$10,$11,$12,$13,$14,$15,$16,$8,$8
)`,
		f.ID,
		"idempotency-"+f.ID,
		"command-"+f.ID,
		f.EventID,
		"edge-"+f.EventID,
		"aggregate-"+f.ID,
		f.OccurredAt,
		f.ReceivedAt,
		string(raw),
		sha,
		f.Processed,
		f.Status,
		f.NextRetryAt,
		f.LockedAt,
		f.LockedBy,
		f.LastError,
	); err != nil {
		t.Fatalf("insert inbox event %s: %v", f.ID, err)
	}
}

type stockMoveFixture struct {
	ID                string
	SourceEventID     string
	BusinessDateLocal string
	OccurredAt        time.Time
}

func insertStockMove(t *testing.T, ctx context.Context, pool *pgxpool.Pool, f stockMoveFixture) {
	t.Helper()
	if f.BusinessDateLocal == "" {
		f.BusinessDateLocal = "2026-06-01"
	}
	if f.OccurredAt.IsZero() {
		f.OccurredAt = olapBaseTime
	}
	docID := "doc-" + f.ID
	if _, err := pool.Exec(ctx, `
INSERT INTO stock_documents(
  id,restaurant_id,warehouse_id,document_type,source_event_id,source_event_type,business_date_local,occurred_at,created_at
) VALUES ($1,'restaurant-1','warehouse-1','PURCHASE',$2,'StockReceiptCaptured',$3,$4,$4)`,
		docID, f.SourceEventID, f.BusinessDateLocal, f.OccurredAt); err != nil {
		t.Fatalf("insert stock document %s: %v", docID, err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO stock_ledger(
  id,restaurant_id,warehouse_id,stock_document_id,source_event_id,source_event_type,
  catalog_item_id,order_line_id,movement_type,quantity,unit_code,unit_cost_minor,
  total_cost_minor,costing_status,occurred_at,business_date_local,created_at
) VALUES ($1,'restaurant-1','warehouse-1',$2,$3,'StockReceiptCaptured','catalog-1',NULL,'IN',2.000,'kg',120,240,'final',$4,$5,$4)`,
		f.ID, docID, f.SourceEventID, f.OccurredAt, f.BusinessDateLocal); err != nil {
		t.Fatalf("insert stock ledger %s: %v", f.ID, err)
	}
}

type checkpointState struct {
	lastCheckpoint      string
	lastExportedID      string
	lastError           string
	consecutiveFailures int64
	nextRetryAt         *time.Time
}

func readCheckpoint(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string) checkpointState {
	t.Helper()
	var state checkpointState
	if err := pool.QueryRow(ctx, `
SELECT last_exported_inbox_id,last_exported_event_id,last_error,consecutive_failures,next_retry_at
FROM olap_export_checkpoints
WHERE id = $1`, id).Scan(&state.lastCheckpoint, &state.lastExportedID, &state.lastError, &state.consecutiveFailures, &state.nextRetryAt); err != nil {
		t.Fatalf("read checkpoint %s: %v", id, err)
	}
	return state
}

func readBackfillJob(t *testing.T, ctx context.Context, repo *Repository, id string) app.BackfillJob {
	t.Helper()
	job, err := repo.GetBackfillJob(ctx, id)
	if err != nil {
		t.Fatalf("read backfill job %s: %v", id, err)
	}
	return job
}

func assertInboxEventIDs(t *testing.T, events []app.InboxEvent, want []string) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("unexpected event count: got %+v want %v", events, want)
	}
	for i := range want {
		if events[i].ID != want[i] {
			t.Fatalf("unexpected event at %d: got %+v want %s", i, events[i], want[i])
		}
	}
}

func assertStockMoveIDs(t *testing.T, moves []app.StockMove, want []string) {
	t.Helper()
	if len(moves) != len(want) {
		t.Fatalf("unexpected stock move count: got %+v want %v", moves, want)
	}
	for i := range want {
		if moves[i].LedgerEntryID != want[i] {
			t.Fatalf("unexpected stock move at %d: got %+v want %s", i, moves[i], want[i])
		}
	}
}

func assertInboxLock(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, wantStatus, wantWorker string, wantLockedAt time.Time, wantAttempts int64) {
	t.Helper()
	var status, worker string
	var lockedAt *time.Time
	var attempts int64
	if err := pool.QueryRow(ctx, `
SELECT olap_export_status,COALESCE(olap_locked_by,''),olap_locked_at,olap_export_attempts
FROM inbox_events
WHERE id = $1`, id).Scan(&status, &worker, &lockedAt, &attempts); err != nil {
		t.Fatalf("read inbox lock %s: %v", id, err)
	}
	if status != wantStatus || worker != wantWorker || lockedAt == nil || !lockedAt.Equal(wantLockedAt) || attempts != wantAttempts {
		t.Fatalf("unexpected lock for %s: status=%s worker=%s locked_at=%v attempts=%d", id, status, worker, lockedAt, attempts)
	}
}

func assertInboxProcessed(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string) {
	t.Helper()
	var processed bool
	var status string
	var lockedAt, nextRetryAt *time.Time
	var lockedBy, lastError string
	if err := pool.QueryRow(ctx, `
SELECT processed_for_olap,olap_export_status,olap_locked_at,COALESCE(olap_locked_by,''),olap_next_retry_at,COALESCE(olap_last_error,'')
FROM inbox_events
WHERE id = $1`, id).Scan(&processed, &status, &lockedAt, &lockedBy, &nextRetryAt, &lastError); err != nil {
		t.Fatalf("read processed inbox %s: %v", id, err)
	}
	if !processed || status != "processed" || lockedAt != nil || lockedBy != "" || nextRetryAt != nil || lastError != "" {
		t.Fatalf("unexpected processed state for %s: processed=%v status=%s locked_at=%v locked_by=%q retry=%v error=%q", id, processed, status, lockedAt, lockedBy, nextRetryAt, lastError)
	}
}

func assertInboxFailed(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string, wantNextRetry time.Time, wantError string) {
	t.Helper()
	var processed bool
	var status, lastError, lockedBy string
	var nextRetryAt, lockedAt *time.Time
	if err := pool.QueryRow(ctx, `
SELECT processed_for_olap,olap_export_status,olap_next_retry_at,olap_locked_at,COALESCE(olap_locked_by,''),COALESCE(olap_last_error,'')
FROM inbox_events
WHERE id = $1`, id).Scan(&processed, &status, &nextRetryAt, &lockedAt, &lockedBy, &lastError); err != nil {
		t.Fatalf("read failed inbox %s: %v", id, err)
	}
	if processed || status != "failed" || nextRetryAt == nil || !nextRetryAt.Equal(wantNextRetry) || lockedAt != nil || lockedBy != "" || lastError != wantError {
		t.Fatalf("unexpected failed state for %s: processed=%v status=%s retry=%v locked_at=%v locked_by=%q error=%q", id, processed, status, nextRetryAt, lockedAt, lockedBy, lastError)
	}
}

func assertInboxRetryCleared(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, wantStatus string) {
	t.Helper()
	var status, lockedBy, lastError string
	var nextRetryAt, lockedAt *time.Time
	if err := pool.QueryRow(ctx, `
SELECT olap_export_status,olap_next_retry_at,olap_locked_at,COALESCE(olap_locked_by,''),COALESCE(olap_last_error,'')
FROM inbox_events
WHERE id = $1`, id).Scan(&status, &nextRetryAt, &lockedAt, &lockedBy, &lastError); err != nil {
		t.Fatalf("read retry-cleared inbox %s: %v", id, err)
	}
	if status != wantStatus || nextRetryAt != nil || lockedAt != nil || lockedBy != "" || lastError != "" {
		t.Fatalf("unexpected retry-cleared state for %s: status=%s retry=%v locked_at=%v locked_by=%q error=%q", id, status, nextRetryAt, lockedAt, lockedBy, lastError)
	}
}

func assertExportRetryCommandCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, commandID string, want int64) {
	t.Helper()
	var got int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM olap_export_retry_commands WHERE command_id = $1`, commandID).Scan(&got); err != nil {
		t.Fatalf("count retry command %s: %v", commandID, err)
	}
	if got != want {
		t.Fatalf("unexpected retry command count: got %d want %d", got, want)
	}
}

func assertAuditEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, action, jobID, reason string) {
	t.Helper()
	var gotAction, gotJobID, gotReason string
	if err := pool.QueryRow(ctx, `
SELECT action,job_id,reason
FROM olap_operator_audit_events
WHERE id = $1`, id).Scan(&gotAction, &gotJobID, &gotReason); err != nil {
		t.Fatalf("read audit event %s: %v", id, err)
	}
	if gotAction != action || gotJobID != jobID || gotReason != reason {
		t.Fatalf("unexpected audit event %s: action=%q job=%q reason=%q", id, gotAction, gotJobID, gotReason)
	}
}

func assertPublicJSONDoesNotContain(t *testing.T, value any, forbidden ...string) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, marker := range forbidden {
		if strings.Contains(body, marker) {
			t.Fatalf("public JSON exposed forbidden marker %q: %s", marker, body)
		}
	}
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func openOLAPRepositoryWithBaseline(t *testing.T, ctx context.Context) (*pgxpool.Pool, *Repository, func()) {
	t.Helper()
	pool := openOLAPPostgresIntegrationPool(t)
	resetOLAPPublicSchema(t, ctx, pool)
	if err := platformpg.MigrateDirWithPolicy(ctx, pool, actualOLAPMigrationsDir(), platformpg.MigrationOptions{
		ModuleName:         "cloud-backend",
		ModuleVersion:      "0.1.0",
		BackupDir:          t.TempDir(),
		SchemaRequirements: cloudsyncpg.RequiredSchema(),
	}); err != nil {
		t.Fatalf("postgres baseline migration failed: %v", err)
	}
	return pool, NewRepository(pool), func() {
		resetOLAPPublicSchema(t, context.Background(), pool)
	}
}

func actualOLAPMigrationsDir() string {
	return filepath.Join("..", "..", "..", "..", "migrations", "postgres")
}

func openOLAPPostgresIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("CLOUD_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("CLOUD_POSTGRES_TEST_DSN is not set")
	}
	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	lockOLAPPostgresIntegration(t, t.Context(), pool)
	return pool
}

func resetOLAPPublicSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset public schema: %v", err)
	}
}

func lockOLAPPostgresIntegration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock(72905101)`); err != nil {
		t.Fatalf("lock postgres integration db: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `SELECT pg_advisory_unlock(72905101)`); err != nil {
			t.Logf("unlock postgres integration db: %v", err)
		}
	})
}
