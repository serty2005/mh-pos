package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/domain/receipt"
)

const printJobColumns = `id,restaurant_id,document_type,scope_id,source_kind,source_id,status,attempts,max_attempts,printer_class,last_error,next_attempt_at,locked_by,locked_at,printed_at,created_at,updated_at`
const printJobTargetColumns = `id,print_job_id,restaurant_id,printer_id,scope_type,scope_id,status,attempts,max_attempts,is_required,last_error,next_attempt_at,locked_by,locked_at,printed_at,created_at,updated_at`
const printRouteColumns = `id,restaurant_id,document_type,scope_type,scope_id,printer_id,is_required,sort_order,origin,is_active,created_at,updated_at`

// EnqueuePrintJob вставляет задачу печати идемпотентно по document_type/source_id.
func (r *Repository) EnqueuePrintJob(ctx context.Context, v *receipt.PrintJob) error {
	if err := v.ValidateForCreate(); err != nil {
		return err
	}
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT OR IGNORE INTO print_jobs(`+printJobColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, string(v.DocumentType), nullableString(v.ScopeID), v.SourceKind, v.SourceID, string(v.Status), v.Attempts, v.MaxAttempts, v.PrinterClass,
		nullableString(v.LastError), nullableTime(v.NextAttemptAt), nullableString(v.LockedBy), nullableTime(v.LockedAt), nullableTime(v.PrintedAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) EnqueuePrintJobWithTargets(ctx context.Context, v *receipt.PrintJob, targets []receipt.PrintJobTarget) error {
	if err := v.ValidateForCreate(); err != nil {
		return err
	}
	exec := r.execer(ctx)
	query := r.queryer(ctx)
	if _, err := exec.ExecContext(ctx, `INSERT OR IGNORE INTO print_jobs(`+printJobColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, string(v.DocumentType), nullableString(v.ScopeID), v.SourceKind, v.SourceID, string(v.Status), v.Attempts, v.MaxAttempts, v.PrinterClass,
		nullableString(v.LastError), nullableTime(v.NextAttemptAt), nullableString(v.LockedBy), nullableTime(v.LockedAt), nullableTime(v.PrintedAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt)); err != nil {
		return normalizeErr(err)
	}
	var jobID string
	if err := query.QueryRowContext(ctx, `SELECT id FROM print_jobs WHERE document_type = ? AND source_id = ?`, string(v.DocumentType), v.SourceID).Scan(&jobID); err != nil {
		return normalizeErr(err)
	}
	for i := range targets {
		t := targets[i]
		if t.PrintJobID == "" {
			t.PrintJobID = jobID
		}
		_, err := exec.ExecContext(ctx, `INSERT OR IGNORE INTO print_job_targets(`+printJobTargetColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			t.ID, t.PrintJobID, t.RestaurantID, t.PrinterID, t.ScopeType, nullableString(t.ScopeID), string(t.Status), t.Attempts, t.MaxAttempts, boolInt(t.IsRequired),
			nullableString(t.LastError), nullableTime(t.NextAttemptAt), nullableString(t.LockedBy), nullableTime(t.LockedAt), nullableTime(t.PrintedAt), dbTime(t.CreatedAt), dbTime(t.UpdatedAt))
		if err != nil {
			return normalizeErr(err)
		}
	}
	if len(targets) > 0 {
		if err := aggregatePrintJobStatus(ctx, exec, jobID, dbTime(v.UpdatedAt)); err != nil {
			return err
		}
	}
	return nil
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

func (r *Repository) ListPrintRoutes(ctx context.Context, restaurantID string) ([]receipt.PrintRoute, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT `+printRouteColumns+` FROM print_routes WHERE restaurant_id = ? ORDER BY document_type, scope_type, COALESCE(scope_id,''), sort_order, id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	out := []receipt.PrintRoute{}
	for rows.Next() {
		v, err := scanPrintRoute(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) CreatePrintRoute(ctx context.Context, v receipt.PrintRoute, auditID, actorEmployeeID string, now time.Time) (*receipt.PrintRoute, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer tx.Rollback()
	stored, err := scanPrintRoute(tx.QueryRowContext(ctx, `INSERT INTO print_routes(
id,restaurant_id,document_type,scope_type,scope_id,printer_id,is_required,sort_order,origin,is_active,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
RETURNING `+printRouteColumns,
		v.ID, v.RestaurantID, string(v.DocumentType), v.ScopeType, nullableString(v.ScopeID), v.PrinterID, boolInt(v.IsRequired), v.SortOrder, v.Origin, boolInt(v.IsActive), dbTime(now), dbTime(now)))
	if err != nil {
		return nil, normalizeErr(err)
	}
	if err := insertPrintRouteAudit(ctx, tx, auditID, stored.RestaurantID, actorEmployeeID, "create", stored, nil, stored, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, normalizeErr(err)
	}
	return stored, nil
}

func (r *Repository) UpdatePrintRoute(ctx context.Context, v receipt.PrintRoute, auditID, actorEmployeeID string, now time.Time) (*receipt.PrintRoute, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer tx.Rollback()
	before, err := scanPrintRoute(tx.QueryRowContext(ctx, `SELECT `+printRouteColumns+` FROM print_routes WHERE id = ?`, strings.TrimSpace(v.ID)))
	if err != nil {
		return nil, err
	}
	stored, err := scanPrintRoute(tx.QueryRowContext(ctx, `UPDATE print_routes
SET document_type = ?, scope_type = ?, scope_id = ?, printer_id = ?, is_required = ?, sort_order = ?, origin = ?, is_active = ?, updated_at = ?
WHERE id = ?
RETURNING `+printRouteColumns,
		string(v.DocumentType), v.ScopeType, nullableString(v.ScopeID), v.PrinterID, boolInt(v.IsRequired), v.SortOrder, v.Origin, boolInt(v.IsActive), dbTime(now), strings.TrimSpace(v.ID)))
	if err != nil {
		return nil, normalizeErr(err)
	}
	if err := insertPrintRouteAudit(ctx, tx, auditID, stored.RestaurantID, actorEmployeeID, "update", stored, before, stored, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, normalizeErr(err)
	}
	return stored, nil
}

func (r *Repository) DeactivatePrintRoute(ctx context.Context, id, auditID, actorEmployeeID string, now time.Time) (*receipt.PrintRoute, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer tx.Rollback()
	before, err := scanPrintRoute(tx.QueryRowContext(ctx, `SELECT `+printRouteColumns+` FROM print_routes WHERE id = ?`, strings.TrimSpace(id)))
	if err != nil {
		return nil, err
	}
	stored, err := scanPrintRoute(tx.QueryRowContext(ctx, `UPDATE print_routes
SET is_active = 0, updated_at = ?
WHERE id = ?
RETURNING `+printRouteColumns, dbTime(now), strings.TrimSpace(id)))
	if err != nil {
		return nil, normalizeErr(err)
	}
	if err := insertPrintRouteAudit(ctx, tx, auditID, stored.RestaurantID, actorEmployeeID, "delete", stored, before, nil, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, normalizeErr(err)
	}
	return stored, nil
}

func (r *Repository) ListActivePrintRoutes(ctx context.Context, restaurantID string, documentType receipt.DocumentType, scopeType string, scopeID *string) ([]receipt.PrintRoute, error) {
	query := `SELECT ` + printRouteColumns + ` FROM print_routes WHERE restaurant_id = ? AND document_type = ? AND scope_type = ? AND is_active = 1`
	args := []any{strings.TrimSpace(restaurantID), string(documentType), strings.TrimSpace(scopeType)}
	if scopeID == nil {
		query += ` AND scope_id IS NULL`
	} else {
		query += ` AND scope_id = ?`
		args = append(args, strings.TrimSpace(*scopeID))
	}
	query += ` ORDER BY sort_order ASC, created_at ASC, id ASC`
	rows, err := r.queryer(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	out := []receipt.PrintRoute{}
	for rows.Next() {
		v, err := scanPrintRoute(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) ListPrintJobTargets(ctx context.Context, jobID string) ([]receipt.PrintJobTarget, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT `+printJobTargetColumns+` FROM print_job_targets WHERE print_job_id = ? ORDER BY created_at ASC, id ASC`, strings.TrimSpace(jobID))
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	out := []receipt.PrintJobTarget{}
	for rows.Next() {
		v, err := scanPrintJobTarget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) ListPrintJobTargetsForCheck(ctx context.Context, checkID string) ([]receipt.PrintJobTarget, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT `+prefixPrintJobTargetColumns("t")+`
FROM print_job_targets t
JOIN print_jobs j ON j.id = t.print_job_id
WHERE (j.document_type = 'check_nonfiscal' AND j.source_kind = 'check' AND j.source_id = ?)
   OR (j.document_type = 'ticket' AND j.source_kind = 'ticket' AND j.source_id IN (
        SELECT tu.id FROM ticket_units tu WHERE tu.check_id = ?
      ))
ORDER BY j.created_at ASC, t.created_at ASC, t.id ASC`, strings.TrimSpace(checkID), strings.TrimSpace(checkID))
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	out := []receipt.PrintJobTarget{}
	for rows.Next() {
		v, err := scanPrintJobTarget(rows)
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

func (r *Repository) ClaimDuePrintJobTarget(ctx context.Context, workerID string, now time.Time) (*receipt.PrintJob, *receipt.PrintJobTarget, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, normalizeErr(err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE print_job_targets
SET status = 'processing', locked_by = ?, locked_at = ?, updated_at = ?
WHERE id = (
  SELECT t.id FROM print_job_targets t
  WHERE t.status = 'pending' AND (t.next_attempt_at IS NULL OR t.next_attempt_at <= ?)
    AND NOT EXISTS (
      SELECT 1 FROM print_job_targets c
      WHERE c.printer_id = t.printer_id AND c.status = 'processing'
    )
  ORDER BY t.printer_id, t.created_at
  LIMIT 1
)`, strings.TrimSpace(workerID), dbTime(now), dbTime(now), dbTime(now))
	if err != nil {
		return nil, nil, normalizeErr(err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return nil, nil, domain.ErrNotFound
	}
	target, err := scanPrintJobTarget(tx.QueryRowContext(ctx, `SELECT `+printJobTargetColumns+` FROM print_job_targets WHERE locked_by = ? AND locked_at = ? ORDER BY updated_at DESC LIMIT 1`, strings.TrimSpace(workerID), dbTime(now)))
	if err != nil {
		return nil, nil, err
	}
	job, err := scanPrintJob(tx.QueryRowContext(ctx, `SELECT `+printJobColumns+` FROM print_jobs WHERE id = ?`, target.PrintJobID))
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, normalizeErr(err)
	}
	return job, target, nil
}

func (r *Repository) MarkPrintJobSucceeded(ctx context.Context, id string, attempts int, now time.Time) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE print_jobs
SET status = 'succeeded', attempts = ?, last_error = NULL, next_attempt_at = NULL, locked_by = NULL, locked_at = NULL, printed_at = ?, updated_at = ?
WHERE id = ?`, attempts, dbTime(now), dbTime(now), strings.TrimSpace(id))
	return normalizeErr(err)
}

func (r *Repository) MarkPrintJobTargetSucceeded(ctx context.Context, id string, attempts int, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return normalizeErr(err)
	}
	defer tx.Rollback()
	var jobID string
	if err := tx.QueryRowContext(ctx, `SELECT print_job_id FROM print_job_targets WHERE id = ?`, strings.TrimSpace(id)).Scan(&jobID); err != nil {
		return normalizeErr(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE print_job_targets
SET status = 'succeeded', attempts = ?, last_error = NULL, next_attempt_at = NULL, locked_by = NULL, locked_at = NULL, printed_at = ?, updated_at = ?
WHERE id = ?`, attempts, dbTime(now), dbTime(now), strings.TrimSpace(id)); err != nil {
		return normalizeErr(err)
	}
	if err := aggregatePrintJobStatus(ctx, tx, jobID, dbTime(now)); err != nil {
		return err
	}
	return normalizeErr(tx.Commit())
}

func (r *Repository) MarkPrintJobFailedAttempt(ctx context.Context, id string, attempts int, status receipt.PrintJobStatus, nextAttemptAt *time.Time, lastError string, now time.Time) error {
	_, err := r.execer(ctx).ExecContext(ctx, `UPDATE print_jobs
SET status = ?, attempts = ?, last_error = ?, next_attempt_at = ?, locked_by = NULL, locked_at = NULL, updated_at = ?
WHERE id = ?`, string(status), attempts, safePrintError(lastError), nullableTime(nextAttemptAt), dbTime(now), strings.TrimSpace(id))
	return normalizeErr(err)
}

func (r *Repository) MarkPrintJobTargetFailedAttempt(ctx context.Context, id string, attempts int, status receipt.PrintJobStatus, nextAttemptAt *time.Time, lastError string, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return normalizeErr(err)
	}
	defer tx.Rollback()
	var jobID string
	if err := tx.QueryRowContext(ctx, `SELECT print_job_id FROM print_job_targets WHERE id = ?`, strings.TrimSpace(id)).Scan(&jobID); err != nil {
		return normalizeErr(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE print_job_targets
SET status = ?, attempts = ?, last_error = ?, next_attempt_at = ?, locked_by = NULL, locked_at = NULL, updated_at = ?
WHERE id = ?`, string(status), attempts, safePrintError(lastError), nullableTime(nextAttemptAt), dbTime(now), strings.TrimSpace(id)); err != nil {
		return normalizeErr(err)
	}
	if err := aggregatePrintJobStatus(ctx, tx, jobID, dbTime(now)); err != nil {
		return err
	}
	return normalizeErr(tx.Commit())
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

func (r *Repository) ResetPrintJobForRetryWithTargets(ctx context.Context, id string, targets []receipt.PrintJobTarget, now time.Time) (*receipt.PrintJob, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE print_jobs
SET status = 'pending', attempts = 0, last_error = NULL, next_attempt_at = NULL, locked_by = NULL, locked_at = NULL, printed_at = NULL, updated_at = ?
WHERE id = ? AND status IN ('pending','failed','succeeded')`, dbTime(now), strings.TrimSpace(id))
	if err != nil {
		return nil, normalizeErr(err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		if _, getErr := r.GetPrintJob(ctx, id); getErr == nil {
			return nil, domain.ErrConflict
		}
		return nil, domain.ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM print_job_targets WHERE print_job_id = ?`, strings.TrimSpace(id)); err != nil {
		return nil, normalizeErr(err)
	}
	for i := range targets {
		t := targets[i]
		t.PrintJobID = strings.TrimSpace(id)
		if _, err := tx.ExecContext(ctx, `INSERT INTO print_job_targets(`+printJobTargetColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			t.ID, t.PrintJobID, t.RestaurantID, t.PrinterID, t.ScopeType, nullableString(t.ScopeID), string(t.Status), t.Attempts, t.MaxAttempts, boolInt(t.IsRequired),
			nullableString(t.LastError), nullableTime(t.NextAttemptAt), nullableString(t.LockedBy), nullableTime(t.LockedAt), nullableTime(t.PrintedAt), dbTime(t.CreatedAt), dbTime(t.UpdatedAt)); err != nil {
			return nil, normalizeErr(err)
		}
	}
	if len(targets) > 0 {
		if err := aggregatePrintJobStatus(ctx, tx, id, dbTime(now)); err != nil {
			return nil, err
		}
	}
	job, err := scanPrintJob(tx.QueryRowContext(ctx, `SELECT `+printJobColumns+` FROM print_jobs WHERE id = ?`, strings.TrimSpace(id)))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, normalizeErr(err)
	}
	return job, nil
}

func (r *Repository) RetryPrintJobTarget(ctx context.Context, jobID, targetID string, now time.Time) (*receipt.PrintJob, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE print_job_targets
SET status = 'pending', attempts = 0, last_error = NULL, next_attempt_at = NULL, locked_by = NULL, locked_at = NULL, printed_at = NULL, updated_at = ?
WHERE id = ? AND print_job_id = ? AND status IN ('pending','failed','processing')`, dbTime(now), strings.TrimSpace(targetID), strings.TrimSpace(jobID))
	if err != nil {
		return nil, normalizeErr(err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		if _, getErr := scanPrintJobTarget(tx.QueryRowContext(ctx, `SELECT `+printJobTargetColumns+` FROM print_job_targets WHERE id = ? AND print_job_id = ?`, strings.TrimSpace(targetID), strings.TrimSpace(jobID))); getErr == nil {
			return nil, domain.ErrConflict
		}
		return nil, domain.ErrNotFound
	}
	if err := aggregatePrintJobStatus(ctx, tx, strings.TrimSpace(jobID), dbTime(now)); err != nil {
		return nil, err
	}
	job, err := scanPrintJob(tx.QueryRowContext(ctx, `SELECT `+printJobColumns+` FROM print_jobs WHERE id = ?`, strings.TrimSpace(jobID)))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, normalizeErr(err)
	}
	return job, nil
}

func insertPrintRouteAudit(ctx context.Context, exec execer, id, restaurantID, actorEmployeeID, action string, route, before, after *receipt.PrintRoute, now time.Time) error {
	var beforeRaw, afterRaw any
	if before != nil {
		body, err := json.Marshal(before)
		if err != nil {
			return normalizeErr(err)
		}
		beforeRaw = string(body)
	}
	if after != nil {
		body, err := json.Marshal(after)
		if err != nil {
			return normalizeErr(err)
		}
		afterRaw = string(body)
	}
	_, err := exec.ExecContext(ctx, `INSERT INTO printer_route_override_audit(
id,restaurant_id,actor_employee_id,action,route_id,scope_type,scope_id,document_type,before_json,after_json,outbox_command_id,occurred_at,created_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		strings.TrimSpace(id), strings.TrimSpace(restaurantID), nullableStringValue(actorEmployeeID), action, route.ID, route.ScopeType, nullableString(route.ScopeID), string(route.DocumentType), beforeRaw, afterRaw, nil, dbTime(now), dbTime(now))
	return normalizeErr(err)
}

func scanPrintJob(row scanner) (*receipt.PrintJob, error) {
	return scanPrintJobRows(row)
}

func scanPrintJobRows(row scanner) (*receipt.PrintJob, error) {
	var v receipt.PrintJob
	var documentType, status, created, updated string
	var lastError, nextAttemptAt, lockedBy, lockedAt, printedAt sql.NullString
	var scopeID sql.NullString
	if err := row.Scan(&v.ID, &v.RestaurantID, &documentType, &scopeID, &v.SourceKind, &v.SourceID, &status, &v.Attempts, &v.MaxAttempts,
		&v.PrinterClass, &lastError, &nextAttemptAt, &lockedBy, &lockedAt, &printedAt, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.DocumentType = receipt.DocumentType(documentType)
	v.ScopeID = stringPtr(scopeID)
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

func scanPrintRoute(row scanner) (*receipt.PrintRoute, error) {
	var v receipt.PrintRoute
	var documentType, scopeType, origin, created, updated string
	var scopeID sql.NullString
	var required, active int
	if err := row.Scan(&v.ID, &v.RestaurantID, &documentType, &scopeType, &scopeID, &v.PrinterID, &required, &v.SortOrder, &origin, &active, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.DocumentType = receipt.DocumentType(documentType)
	v.ScopeType = scopeType
	v.ScopeID = stringPtr(scopeID)
	v.IsRequired = required == 1
	v.Origin = origin
	v.IsActive = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func scanPrintJobTarget(row scanner) (*receipt.PrintJobTarget, error) {
	var v receipt.PrintJobTarget
	var scopeID, lastError, nextAttemptAt, lockedBy, lockedAt, printedAt sql.NullString
	var status, created, updated string
	var required int
	if err := row.Scan(&v.ID, &v.PrintJobID, &v.RestaurantID, &v.PrinterID, &v.ScopeType, &scopeID, &status, &v.Attempts, &v.MaxAttempts, &required,
		&lastError, &nextAttemptAt, &lockedBy, &lockedAt, &printedAt, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.ScopeID = stringPtr(scopeID)
	v.Status = receipt.PrintJobStatus(status)
	v.IsRequired = required == 1
	v.LastError = stringPtr(lastError)
	v.NextAttemptAt = timePtr(nextAttemptAt)
	v.LockedBy = stringPtr(lockedBy)
	v.LockedAt = timePtr(lockedAt)
	v.PrintedAt = timePtr(printedAt)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func aggregatePrintJobStatus(ctx context.Context, exec execer, jobID, updatedAt string) error {
	_, err := exec.ExecContext(ctx, `UPDATE print_jobs
SET status = CASE
    WHEN EXISTS (
      SELECT 1 FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1 AND t.status = 'failed' AND t.attempts >= t.max_attempts
    ) THEN 'failed'
    WHEN EXISTS (
      SELECT 1 FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1
    ) AND NOT EXISTS (
      SELECT 1 FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1 AND t.status <> 'succeeded'
    ) THEN 'succeeded'
    ELSE 'pending'
  END,
  attempts = CASE
    WHEN EXISTS (
      SELECT 1 FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1 AND t.status = 'failed' AND t.attempts >= t.max_attempts
    ) THEN (
      SELECT t.attempts FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1 AND t.status = 'failed'
      ORDER BY t.updated_at DESC LIMIT 1
    )
    WHEN EXISTS (
      SELECT 1 FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1
    ) AND NOT EXISTS (
      SELECT 1 FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1 AND t.status <> 'succeeded'
    ) THEN (
      SELECT MAX(t.attempts) FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1
    )
    ELSE attempts
  END,
  last_error = (
    SELECT t.last_error FROM print_job_targets t
    WHERE t.print_job_id = print_jobs.id AND t.is_required = 1 AND t.status = 'failed'
    ORDER BY t.updated_at DESC LIMIT 1
  ),
  next_attempt_at = (
    SELECT MIN(t.next_attempt_at) FROM print_job_targets t
    WHERE t.print_job_id = print_jobs.id AND t.status = 'pending'
  ),
  locked_by = NULL,
  locked_at = NULL,
  printed_at = CASE
    WHEN EXISTS (
      SELECT 1 FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1
    ) AND NOT EXISTS (
      SELECT 1 FROM print_job_targets t
      WHERE t.print_job_id = print_jobs.id AND t.is_required = 1 AND t.status <> 'succeeded'
    ) THEN ?
    ELSE printed_at
  END,
  updated_at = ?
WHERE id = ?`, updatedAt, updatedAt, strings.TrimSpace(jobID))
	return normalizeErr(err)
}

func safePrintError(value string) string {
	value = strings.TrimSpace(value)
	if isSafePrintErrorCode(value) {
		return value
	}
	return "PRINT_DELIVERY_FAILED"
}

func prefixPrintJobTargetColumns(alias string) string {
	parts := strings.Split(printJobTargetColumns, ",")
	for i, part := range parts {
		parts[i] = alias + "." + strings.TrimSpace(part)
	}
	return strings.Join(parts, ",")
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
