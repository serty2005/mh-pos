package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/domain/receipt"
)

const printJobColumns = `id,restaurant_id,document_type,source_kind,source_id,status,attempts,max_attempts,printer_class,last_error,next_attempt_at,locked_by,locked_at,printed_at,created_at,updated_at`

// EnqueuePrintJob вставляет задачу печати идемпотентно по document_type/source_id.
func (r *Repository) EnqueuePrintJob(ctx context.Context, v *receipt.PrintJob) error {
	if err := v.ValidateForCreate(); err != nil {
		return err
	}
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT OR IGNORE INTO print_jobs(`+printJobColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, string(v.DocumentType), v.SourceKind, v.SourceID, string(v.Status), v.Attempts, v.MaxAttempts, v.PrinterClass,
		nullableString(v.LastError), nullableTime(v.NextAttemptAt), nullableString(v.LockedBy), nullableTime(v.LockedAt), nullableTime(v.PrintedAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetPrintJob(ctx context.Context, id string) (*receipt.PrintJob, error) {
	return scanPrintJob(r.queryer(ctx).QueryRowContext(ctx, `SELECT `+printJobColumns+` FROM print_jobs WHERE id = ?`, strings.TrimSpace(id)))
}

func (r *Repository) ListPrintJobs(ctx context.Context, q receipt.PrintJobListQuery) ([]receipt.PrintJob, error) {
	limit := q.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := `SELECT ` + printJobColumns + ` FROM print_jobs WHERE 1=1`
	args := make([]any, 0, 4)
	if strings.TrimSpace(q.RestaurantID) != "" {
		query += ` AND restaurant_id = ?`
		args = append(args, strings.TrimSpace(q.RestaurantID))
	}
	if q.Status != "" {
		query += ` AND status = ?`
		args = append(args, string(q.Status))
	}
	if q.DocumentType != "" {
		query += ` AND document_type = ?`
		args = append(args, string(q.DocumentType))
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.queryer(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []receipt.PrintJob
	for rows.Next() {
		v, err := scanPrintJobRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

// ClaimDuePrintJob атомарно переводит ближайшую due pending job в processing.
func (r *Repository) ClaimDuePrintJob(ctx context.Context, workerID string, now time.Time) (*receipt.PrintJob, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer tx.Rollback()
	var id string
	err = tx.QueryRowContext(ctx, `SELECT id FROM print_jobs
WHERE status = 'pending' AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
ORDER BY created_at ASC, id ASC LIMIT 1`, dbTime(now)).Scan(&id)
	if err != nil {
		return nil, normalizeErr(err)
	}
	res, err := tx.ExecContext(ctx, `UPDATE print_jobs
SET status = 'processing', locked_by = ?, locked_at = ?, updated_at = ?
WHERE id = ? AND status = 'pending'`, strings.TrimSpace(workerID), dbTime(now), dbTime(now), id)
	if err != nil {
		return nil, normalizeErr(err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return nil, domain.ErrNotFound
	}
	job, err := scanPrintJob(tx.QueryRowContext(ctx, `SELECT `+printJobColumns+` FROM print_jobs WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, normalizeErr(err)
	}
	return job, nil
}

func (r *Repository) MarkPrintJobSucceeded(ctx context.Context, id string, attempts int, now time.Time) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE print_jobs
SET status = 'succeeded', attempts = ?, last_error = NULL, next_attempt_at = NULL, locked_by = NULL, locked_at = NULL, printed_at = ?, updated_at = ?
WHERE id = ?`, attempts, dbTime(now), dbTime(now), strings.TrimSpace(id))
	return normalizeErr(err)
}

func (r *Repository) MarkPrintJobFailedAttempt(ctx context.Context, id string, attempts int, status receipt.PrintJobStatus, nextAttemptAt *time.Time, lastError string, now time.Time) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE print_jobs
SET status = ?, attempts = ?, last_error = ?, next_attempt_at = ?, locked_by = NULL, locked_at = NULL, updated_at = ?
WHERE id = ?`, string(status), attempts, safePrintError(lastError), nullableTime(nextAttemptAt), dbTime(now), strings.TrimSpace(id))
	return normalizeErr(err)
}

func (r *Repository) ResetPrintJobForRetry(ctx context.Context, id string, now time.Time) (*receipt.PrintJob, error) {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE print_jobs
SET status = 'pending', attempts = 0, last_error = NULL, next_attempt_at = ?, locked_by = NULL, locked_at = NULL, updated_at = ?
WHERE id = ? AND status IN ('pending','failed')`, dbTime(now), dbTime(now), strings.TrimSpace(id))
	if err != nil {
		return nil, normalizeErr(err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		if _, getErr := r.GetPrintJob(ctx, id); getErr == nil {
			return nil, domain.ErrConflict
		}
		return nil, domain.ErrNotFound
	}
	return r.GetPrintJob(ctx, id)
}

func scanPrintJob(row scanner) (*receipt.PrintJob, error) {
	return scanPrintJobRows(row)
}

func scanPrintJobRows(row scanner) (*receipt.PrintJob, error) {
	var v receipt.PrintJob
	var documentType, status, created, updated string
	var lastError, nextAttemptAt, lockedBy, lockedAt, printedAt sql.NullString
	if err := row.Scan(&v.ID, &v.RestaurantID, &documentType, &v.SourceKind, &v.SourceID, &status, &v.Attempts, &v.MaxAttempts,
		&v.PrinterClass, &lastError, &nextAttemptAt, &lockedBy, &lockedAt, &printedAt, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.DocumentType = receipt.DocumentType(documentType)
	v.Status = receipt.PrintJobStatus(status)
	v.LastError = stringPtr(lastError)
	v.NextAttemptAt = timePtr(nextAttemptAt)
	v.LockedBy = stringPtr(lockedBy)
	v.LockedAt = timePtr(lockedAt)
	v.PrintedAt = timePtr(printedAt)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func safePrintError(value string) string {
	value = strings.TrimSpace(value)
	if isSafePrintErrorCode(value) {
		return value
	}
	return "PRINT_DELIVERY_FAILED"
}

func isSafePrintErrorCode(value string) bool {
	if !strings.HasPrefix(value, "PRINT_") {
		return false
	}
	if len(value) > 500 {
		return false
	}
	for _, r := range value {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
