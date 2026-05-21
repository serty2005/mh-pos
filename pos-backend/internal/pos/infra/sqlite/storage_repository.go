package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"pos-backend/internal/pos/domain/storage"
)

// GetStorageLifecycleStatus возвращает read-only объемы SQLite runtime и outbox blocking state.
func (r *Repository) GetStorageLifecycleStatus(ctx context.Context) (storage.LifecycleStatus, error) {
	sqliteStats, err := r.sqliteDatabaseStats(ctx)
	if err != nil {
		return storage.LifecycleStatus{}, err
	}
	counts, err := r.storageTableCounts(ctx)
	if err != nil {
		return storage.LifecycleStatus{}, err
	}
	dateRange, err := r.closedOrderBusinessDateRange(ctx)
	if err != nil {
		return storage.LifecycleStatus{}, err
	}
	byDate, err := r.closedOrdersByBusinessDate(ctx)
	if err != nil {
		return storage.LifecycleStatus{}, err
	}
	outbox, err := r.storageOutboxStatusCounts(ctx)
	if err != nil {
		return storage.LifecycleStatus{}, err
	}
	blocking, err := r.blockingOutboxMessages(ctx)
	if err != nil {
		return storage.LifecycleStatus{}, err
	}
	return storage.LifecycleStatus{
		SQLite:                       sqliteStats,
		Tables:                       counts,
		ClosedOrderBusinessDateRange: dateRange,
		ClosedOrdersByBusinessDate:   byDate,
		Outbox:                       outbox,
		BlockingOutboxMessages:       blocking,
	}, nil
}

// DryRunStorageRetention считает candidate rows старше cutoff без записи, удаления или архивации.
func (r *Repository) DryRunStorageRetention(ctx context.Context, cutoffBusinessDateLocal string) (storage.RetentionDryRunResult, error) {
	eligible, err := r.retentionEligibleCounts(ctx, cutoffBusinessDateLocal)
	if err != nil {
		return storage.RetentionDryRunResult{}, err
	}
	counts, err := r.storageTableCounts(ctx)
	if err != nil {
		return storage.RetentionDryRunResult{}, err
	}
	blocking, err := r.blockingOutboxMessages(ctx)
	if err != nil {
		return storage.RetentionDryRunResult{}, err
	}
	blockReasons := []string{}
	activeOrders := counts.OpenOrders + counts.LockedOrders
	if activeOrders > 0 {
		blockReasons = append(blockReasons, "active_orders")
	}
	if counts.OpenShifts > 0 {
		blockReasons = append(blockReasons, "open_shifts")
	}
	if counts.OpenCashSessions > 0 {
		blockReasons = append(blockReasons, "open_cash_sessions")
	}
	if blocking > 0 {
		blockReasons = append(blockReasons, "pending_edge_to_cloud_outbox")
	}
	return storage.RetentionDryRunResult{
		CutoffBusinessDateLocal: cutoffBusinessDateLocal,
		Eligible:                eligible,
		ActiveOrders:            activeOrders,
		OpenShifts:              counts.OpenShifts,
		OpenCashSessions:        counts.OpenCashSessions,
		BlockingOutboxMessages:  blocking,
		BlockReasons:            blockReasons,
	}, nil
}

// BuildStorageArchiveExportPlan строит deterministic manifest-only scope без записи archive files.
func (r *Repository) BuildStorageArchiveExportPlan(ctx context.Context, cutoffBusinessDateLocal string) (storage.ArchiveExportPlan, error) {
	eligible, err := r.retentionEligibleCounts(ctx, cutoffBusinessDateLocal)
	if err != nil {
		return storage.ArchiveExportPlan{}, err
	}
	dateRange, err := r.archivePlanBusinessDateRange(ctx, cutoffBusinessDateLocal)
	if err != nil {
		return storage.ArchiveExportPlan{}, err
	}
	restaurantID, err := r.archivePlanRestaurantID(ctx, cutoffBusinessDateLocal)
	if err != nil {
		return storage.ArchiveExportPlan{}, err
	}
	counts, err := r.storageTableCounts(ctx)
	if err != nil {
		return storage.ArchiveExportPlan{}, err
	}
	blocking, err := r.blockingOutboxMessages(ctx)
	if err != nil {
		return storage.ArchiveExportPlan{}, err
	}
	activeOrders := counts.OpenOrders + counts.LockedOrders
	blockReasons := []string{"dry_run_only_no_archive_policy"}
	if activeOrders > 0 {
		blockReasons = append(blockReasons, "active_orders")
	}
	if counts.OpenShifts > 0 {
		blockReasons = append(blockReasons, "open_shifts")
	}
	if counts.OpenCashSessions > 0 {
		blockReasons = append(blockReasons, "open_cash_sessions")
	}
	if blocking > 0 {
		blockReasons = append(blockReasons, "pending_edge_to_cloud_outbox")
	}
	return storage.ArchiveExportPlan{
		CutoffBusinessDateLocal:   cutoffBusinessDateLocal,
		Mode:                      "manifest_only",
		DestructiveApplySupported: false,
		Blocked:                   true,
		BlockReasons:              blockReasons,
		ArchiveSet:                eligible,
		Protected: storage.ArchivePlanProtectedFlags{
			FinancialLedgerProtected:    true,
			ImmutableSnapshotsProtected: true,
			LocalEventsProtected:        true,
			OutboxProtected:             true,
		},
		ActiveOrders:           activeOrders,
		OpenShifts:             counts.OpenShifts,
		OpenCashSessions:       counts.OpenCashSessions,
		BlockingOutboxMessages: blocking,
		Manifest: storage.ArchivePlanManifest{
			FormatVersion:           "storage-archive-manifest-v1",
			RestaurantID:            restaurantID,
			BusinessDateRange:       dateRange,
			CutoffBusinessDateLocal: cutoffBusinessDateLocal,
			Tables:                  archivePlanTableManifest(eligible),
		},
	}, nil
}

// BuildStorageArchiveExportScope собирает export-only граф закрытых заказов без мутации runtime tables.
func (r *Repository) BuildStorageArchiveExportScope(ctx context.Context, cutoffBusinessDateLocal string) (storage.ArchiveExportScope, error) {
	source, err := r.storageArchiveSourceMetadata(ctx)
	if err != nil {
		return storage.ArchiveExportScope{}, err
	}
	dateRange, err := r.archiveBusinessDateRange(ctx, cutoffBusinessDateLocal)
	if err != nil {
		return storage.ArchiveExportScope{}, err
	}
	blocking, err := r.blockingOutboxMessagesForArchiveScope(ctx, cutoffBusinessDateLocal)
	if err != nil {
		return storage.ArchiveExportScope{}, err
	}
	blockReasons := []string{
		"destructive_apply_not_supported",
		"financial_ledger_protected",
		"immutable_snapshots_protected",
	}
	if blocking > 0 {
		blockReasons = append(blockReasons, "pending_edge_to_cloud_outbox_for_archive_scope")
	}

	scope := storage.ArchiveExportScope{
		BusinessDateRange: dateRange,
		Source:            source,
		BlockReasons:      blockReasons,
		Blocked:           true,
	}
	tables := []archiveTableQuery{
		{name: "orders", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT o.* FROM orders o JOIN eligible_orders eo ON eo.id = o.id ORDER BY o.closed_at, o.id`},
		{name: "order_lines", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT ol.* FROM order_lines ol JOIN eligible_orders eo ON eo.id = ol.order_id ORDER BY ol.order_id, ol.id`},
		{name: "order_line_modifiers", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT olm.* FROM order_line_modifiers olm JOIN order_lines ol ON ol.id = olm.order_line_id JOIN eligible_orders eo ON eo.id = ol.order_id ORDER BY ol.order_id, olm.id`},
		{name: "order_line_discounts", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT old.* FROM order_line_discounts old JOIN eligible_orders eo ON eo.id = old.order_id ORDER BY old.order_id, old.application_index, old.id`},
		{name: "order_surcharges", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT os.* FROM order_surcharges os JOIN eligible_orders eo ON eo.id = os.order_id ORDER BY os.order_id, os.application_index, os.id`},
		{name: "prechecks", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT p.* FROM prechecks p JOIN eligible_orders eo ON eo.id = p.order_id ORDER BY p.order_id, p.version, p.id`},
		{name: "precheck_lines", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT pl.* FROM precheck_lines pl JOIN prechecks p ON p.id = pl.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id ORDER BY pl.precheck_id, pl.id`},
		{name: "precheck_line_modifiers", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT plm.* FROM precheck_line_modifiers plm JOIN prechecks p ON p.id = plm.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id ORDER BY plm.precheck_id, plm.id`},
		{name: "precheck_discounts", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT pd.* FROM precheck_discounts pd JOIN prechecks p ON p.id = pd.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id ORDER BY pd.precheck_id, pd.application_index, pd.id`},
		{name: "precheck_surcharges", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT ps.* FROM precheck_surcharges ps JOIN prechecks p ON p.id = ps.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id ORDER BY ps.precheck_id, ps.application_index, ps.id`},
		{name: "precheck_taxes", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT pt.* FROM precheck_taxes pt JOIN prechecks p ON p.id = pt.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id ORDER BY pt.precheck_id, pt.order_line_id, pt.id`},
		{name: "payments", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT pay.* FROM payments pay JOIN prechecks p ON p.id = pay.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id ORDER BY pay.precheck_id, pay.created_at, pay.id`},
		{name: "payment_attempts", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT pa.* FROM payment_attempts pa JOIN payments pay ON pay.id = pa.payment_id JOIN prechecks p ON p.id = pay.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id ORDER BY pa.payment_id, pa.attempt_no, pa.id`},
		{name: "checks", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT c.* FROM checks c JOIN eligible_orders eo ON eo.id = c.order_id ORDER BY c.business_date_local, c.closed_at, c.id`},
		{name: "financial_operations", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT fo.* FROM financial_operations fo JOIN checks c ON c.id = fo.check_id JOIN eligible_orders eo ON eo.id = c.order_id ORDER BY fo.created_at, fo.id`},
		{name: "financial_operation_items", query: `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT foi.* FROM financial_operation_items foi JOIN financial_operations fo ON fo.id = foi.operation_id JOIN checks c ON c.id = fo.check_id JOIN eligible_orders eo ON eo.id = c.order_id ORDER BY foi.operation_id, foi.created_at, foi.id`},
		{name: "local_event_log_summary", query: archiveLocalEventsSQL},
		{name: "pos_sync_outbox_summary", query: archiveOutboxSQL},
	}
	for _, table := range tables {
		rows, err := r.archiveRows(ctx, table.name, table.query, cutoffBusinessDateLocal)
		if err != nil {
			return storage.ArchiveExportScope{}, err
		}
		scope.Rows = append(scope.Rows, rows...)
		addArchiveCount(&scope.Counts, table.name, len(rows))
	}
	scope.Counts.BlockingOutboxMessages = blocking
	scope.Counts.ArchivedRows = len(scope.Rows)
	return scope, nil
}

// BuildStorageArchiveApplyRuntimeScope считает текущий eligible scope для apply-plan без чтения payload.
func (r *Repository) BuildStorageArchiveApplyRuntimeScope(ctx context.Context, cutoffBusinessDateLocal string) (storage.ArchiveApplyRuntimeScope, error) {
	counts, err := r.archiveEligibleCounts(ctx, cutoffBusinessDateLocal)
	if err != nil {
		return storage.ArchiveApplyRuntimeScope{}, err
	}
	tableCounts, err := r.storageTableCounts(ctx)
	if err != nil {
		return storage.ArchiveApplyRuntimeScope{}, err
	}
	blocking, err := r.blockingOutboxMessages(ctx)
	if err != nil {
		return storage.ArchiveApplyRuntimeScope{}, err
	}
	return storage.ArchiveApplyRuntimeScope{
		Counts:                 counts,
		ActiveOrders:           tableCounts.OpenOrders + tableCounts.LockedOrders,
		OpenShifts:             tableCounts.OpenShifts,
		OpenCashSessions:       tableCounts.OpenCashSessions,
		BlockingOutboxMessages: blocking,
	}, nil
}

func (r *Repository) sqliteDatabaseStats(ctx context.Context) (storage.SQLiteDatabaseStats, error) {
	var stats storage.SQLiteDatabaseStats
	if err := r.queryer(ctx).QueryRowContext(ctx, `PRAGMA page_count`).Scan(&stats.PageCount); err != nil {
		return stats, normalizeErr(err)
	}
	if err := r.queryer(ctx).QueryRowContext(ctx, `PRAGMA page_size`).Scan(&stats.PageSizeBytes); err != nil {
		return stats, normalizeErr(err)
	}
	if err := r.queryer(ctx).QueryRowContext(ctx, `PRAGMA freelist_count`).Scan(&stats.FreelistCount); err != nil {
		return stats, normalizeErr(err)
	}
	if err := r.queryer(ctx).QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&stats.JournalMode); err != nil {
		return stats, normalizeErr(err)
	}
	stats.EstimatedSizeBytes = stats.PageCount * stats.PageSizeBytes
	stats.FreelistBytes = stats.FreelistCount * stats.PageSizeBytes
	return stats, nil
}

func (r *Repository) storageTableCounts(ctx context.Context) (storage.TableCounts, error) {
	var counts storage.TableCounts
	if err := r.queryer(ctx).QueryRowContext(ctx, `
SELECT
  COUNT(1),
  COALESCE(SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END),0),
  COALESCE(SUM(CASE WHEN status = 'locked' THEN 1 ELSE 0 END),0),
  COALESCE(SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END),0),
  COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END),0)
FROM orders`).Scan(&counts.Orders, &counts.OpenOrders, &counts.LockedOrders, &counts.ClosedOrders, &counts.CancelledOrders); err != nil {
		return counts, normalizeErr(err)
	}
	if err := r.queryer(ctx).QueryRowContext(ctx, `
SELECT
  (SELECT COUNT(1) FROM order_lines),
  (SELECT COUNT(1) FROM order_line_modifiers),
  (SELECT COUNT(1) FROM prechecks),
  (SELECT COUNT(1) FROM precheck_lines),
  (SELECT COUNT(1) FROM precheck_line_modifiers),
  (SELECT COUNT(1) FROM precheck_discounts),
  (SELECT COUNT(1) FROM precheck_surcharges),
  (SELECT COUNT(1) FROM precheck_taxes),
  (SELECT COUNT(1) FROM payments),
  (SELECT COUNT(1) FROM payment_attempts),
  (SELECT COUNT(1) FROM checks),
  (SELECT COUNT(1) FROM financial_operations),
  (SELECT COUNT(1) FROM financial_operation_items),
  (SELECT COUNT(1) FROM shifts),
  (SELECT COUNT(1) FROM shifts WHERE status = 'open'),
  (SELECT COUNT(1) FROM cash_sessions),
  (SELECT COUNT(1) FROM cash_sessions WHERE status = 'open'),
  (SELECT COUNT(1) FROM local_event_log),
  (SELECT COUNT(1) FROM pos_sync_outbox)`).
		Scan(
			&counts.OrderLines,
			&counts.OrderLineModifiers,
			&counts.Prechecks,
			&counts.PrecheckLines,
			&counts.PrecheckLineModifiers,
			&counts.PrecheckDiscounts,
			&counts.PrecheckSurcharges,
			&counts.PrecheckTaxes,
			&counts.Payments,
			&counts.PaymentAttempts,
			&counts.Checks,
			&counts.FinancialOperations,
			&counts.FinancialOperationItems,
			&counts.Shifts,
			&counts.OpenShifts,
			&counts.CashSessions,
			&counts.OpenCashSessions,
			&counts.LocalEvents,
			&counts.OutboxMessages,
		); err != nil {
		return counts, normalizeErr(err)
	}
	return counts, nil
}

func (r *Repository) closedOrderBusinessDateRange(ctx context.Context) (storage.BusinessDateRange, error) {
	var oldest, newest sql.NullString
	if err := r.queryer(ctx).QueryRowContext(ctx, `SELECT MIN(c.business_date_local), MAX(c.business_date_local) FROM orders o JOIN checks c ON c.order_id = o.id WHERE o.status = 'closed'`).Scan(&oldest, &newest); err != nil {
		return storage.BusinessDateRange{}, normalizeErr(err)
	}
	return storage.BusinessDateRange{Oldest: oldest.String, Newest: newest.String}, nil
}

func (r *Repository) closedOrdersByBusinessDate(ctx context.Context) ([]storage.ClosedOrdersBusinessDateCount, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `
SELECT c.business_date_local, COUNT(DISTINCT o.id), COUNT(DISTINCT c.id), COALESCE(SUM(c.total),0), MIN(c.closed_at), MAX(c.closed_at)
FROM orders o
JOIN checks c ON c.order_id = o.id
WHERE o.status = 'closed'
GROUP BY c.business_date_local
ORDER BY c.business_date_local DESC`)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []storage.ClosedOrdersBusinessDateCount
	for rows.Next() {
		var item storage.ClosedOrdersBusinessDateCount
		var firstClosedAt, lastClosedAt sql.NullString
		if err := rows.Scan(&item.BusinessDateLocal, &item.Orders, &item.Checks, &item.TotalAmount, &firstClosedAt, &lastClosedAt); err != nil {
			return nil, normalizeErr(err)
		}
		item.FirstClosedAt = firstClosedAt.String
		item.LastClosedAt = lastClosedAt.String
		out = append(out, item)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) storageOutboxStatusCounts(ctx context.Context) ([]storage.OutboxStatusCount, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT status, sync_direction, COUNT(1) FROM pos_sync_outbox GROUP BY status, sync_direction ORDER BY sync_direction, status`)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []storage.OutboxStatusCount
	for rows.Next() {
		var item storage.OutboxStatusCount
		if err := rows.Scan(&item.Status, &item.SyncDirection, &item.Count); err != nil {
			return nil, normalizeErr(err)
		}
		out = append(out, item)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) blockingOutboxMessages(ctx context.Context) (int, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM pos_sync_outbox WHERE sync_direction = 'edge_to_cloud' AND status <> 'sent'`).Scan(&n)
	return n, normalizeErr(err)
}

func (r *Repository) retentionEligibleCounts(ctx context.Context, cutoffBusinessDateLocal string) (storage.RetentionEligibleCounts, error) {
	var counts storage.RetentionEligibleCounts
	scans := []struct {
		target *int
		query  string
	}{
		{&counts.ClosedOrders, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM eligible_orders`},
		{&counts.Checks, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM checks c JOIN eligible_orders eo ON eo.id = c.order_id`},
		{&counts.Prechecks, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM prechecks p JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckLines, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM precheck_lines pl JOIN prechecks p ON p.id = pl.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckLineModifiers, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM precheck_line_modifiers plm JOIN prechecks p ON p.id = plm.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckDiscounts, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM precheck_discounts pd JOIN prechecks p ON p.id = pd.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckSurcharges, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM precheck_surcharges ps JOIN prechecks p ON p.id = ps.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckTaxes, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM precheck_taxes pt JOIN prechecks p ON p.id = pt.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.Payments, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM payments pay JOIN prechecks p ON p.id = pay.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PaymentAttempts, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM payment_attempts pa JOIN payments pay ON pay.id = pa.payment_id JOIN prechecks p ON p.id = pay.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.OrderLines, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM order_lines ol JOIN eligible_orders eo ON eo.id = ol.order_id`},
		{&counts.OrderLineModifiers, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM order_line_modifiers olm JOIN order_lines ol ON ol.id = olm.order_line_id JOIN eligible_orders eo ON eo.id = ol.order_id`},
		{&counts.FinancialOperations, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM financial_operations fo JOIN checks c ON c.id = fo.check_id JOIN eligible_orders eo ON eo.id = c.order_id`},
		{&counts.FinancialOperationItems, `WITH eligible_orders AS (` + eligibleOrdersSQL + `) SELECT COUNT(1) FROM financial_operation_items foi JOIN financial_operations fo ON fo.id = foi.operation_id JOIN checks c ON c.id = fo.check_id JOIN eligible_orders eo ON eo.id = c.order_id`},
	}
	for _, scan := range scans {
		if err := r.queryer(ctx).QueryRowContext(ctx, scan.query, cutoffBusinessDateLocal).Scan(scan.target); err != nil {
			return counts, normalizeErr(err)
		}
	}
	return counts, nil
}

func (r *Repository) archiveEligibleCounts(ctx context.Context, cutoffBusinessDateLocal string) (storage.ArchiveExportCounts, error) {
	var counts storage.ArchiveExportCounts
	scans := []struct {
		target *int
		query  string
	}{
		{&counts.ClosedOrders, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM eligible_orders`},
		{&counts.OrderLines, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM order_lines ol JOIN eligible_orders eo ON eo.id = ol.order_id`},
		{&counts.OrderLineModifiers, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM order_line_modifiers olm JOIN order_lines ol ON ol.id = olm.order_line_id JOIN eligible_orders eo ON eo.id = ol.order_id`},
		{&counts.OrderLineDiscounts, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM order_line_discounts old JOIN eligible_orders eo ON eo.id = old.order_id`},
		{&counts.OrderSurcharges, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM order_surcharges os JOIN eligible_orders eo ON eo.id = os.order_id`},
		{&counts.Prechecks, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM prechecks p JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckLines, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM precheck_lines pl JOIN prechecks p ON p.id = pl.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckLineModifiers, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM precheck_line_modifiers plm JOIN prechecks p ON p.id = plm.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckDiscounts, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM precheck_discounts pd JOIN prechecks p ON p.id = pd.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckSurcharges, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM precheck_surcharges ps JOIN prechecks p ON p.id = ps.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PrecheckTaxes, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM precheck_taxes pt JOIN prechecks p ON p.id = pt.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.Payments, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM payments pay JOIN prechecks p ON p.id = pay.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.PaymentAttempts, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM payment_attempts pa JOIN payments pay ON pay.id = pa.payment_id JOIN prechecks p ON p.id = pay.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id`},
		{&counts.Checks, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM checks c JOIN eligible_orders eo ON eo.id = c.order_id`},
		{&counts.FinancialOperations, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM financial_operations fo JOIN checks c ON c.id = fo.check_id JOIN eligible_orders eo ON eo.id = c.order_id`},
		{&counts.FinancialOperationItems, `WITH eligible_orders AS (` + eligibleOrdersForArchiveSQL + `) SELECT COUNT(1) FROM financial_operation_items foi JOIN financial_operations fo ON fo.id = foi.operation_id JOIN checks c ON c.id = fo.check_id JOIN eligible_orders eo ON eo.id = c.order_id`},
		{&counts.LocalEventReferences, `SELECT COUNT(1) FROM (` + archiveLocalEventsSQL + `)`},
		{&counts.OutboxMessageReferences, `SELECT COUNT(1) FROM (` + archiveOutboxSQL + `)`},
	}
	for _, scan := range scans {
		if err := r.queryer(ctx).QueryRowContext(ctx, scan.query, cutoffBusinessDateLocal).Scan(scan.target); err != nil {
			return counts, normalizeErr(err)
		}
	}
	blocking, err := r.blockingOutboxMessagesForArchiveScope(ctx, cutoffBusinessDateLocal)
	if err != nil {
		return counts, err
	}
	counts.BlockingOutboxMessages = blocking
	counts.ArchivedRows = archiveCountsTotalRows(counts)
	return counts, nil
}

func archiveCountsTotalRows(counts storage.ArchiveExportCounts) int {
	return counts.ClosedOrders +
		counts.OrderLines +
		counts.OrderLineModifiers +
		counts.OrderLineDiscounts +
		counts.OrderSurcharges +
		counts.Prechecks +
		counts.PrecheckLines +
		counts.PrecheckLineModifiers +
		counts.PrecheckDiscounts +
		counts.PrecheckSurcharges +
		counts.PrecheckTaxes +
		counts.Payments +
		counts.PaymentAttempts +
		counts.Checks +
		counts.FinancialOperations +
		counts.FinancialOperationItems +
		counts.LocalEventReferences +
		counts.OutboxMessageReferences
}

const eligibleOrdersSQL = `SELECT o.id FROM orders o JOIN checks c ON c.order_id = o.id WHERE o.status = 'closed' AND c.business_date_local < ?`

const eligibleOrdersForArchiveSQL = eligibleOrdersSQL

const eligibleArchiveRefsSQL = `
eligible_orders AS (` + eligibleOrdersForArchiveSQL + `),
eligible_refs AS (
  SELECT 'Order' AS aggregate_type, eo.id AS aggregate_id FROM eligible_orders eo
  UNION ALL SELECT 'Check', c.id FROM checks c JOIN eligible_orders eo ON eo.id = c.order_id
  UNION ALL SELECT 'Precheck', p.id FROM prechecks p JOIN eligible_orders eo ON eo.id = p.order_id
  UNION ALL SELECT 'Payment', pay.id FROM payments pay JOIN prechecks p ON p.id = pay.precheck_id JOIN eligible_orders eo ON eo.id = p.order_id
  UNION ALL SELECT 'FinancialOperation', fo.id FROM financial_operations fo JOIN checks c ON c.id = fo.check_id JOIN eligible_orders eo ON eo.id = c.order_id
)`

const archiveLocalEventsSQL = `WITH ` + eligibleArchiveRefsSQL + `
SELECT
  l.id, l.event_id, l.command_id, l.envelope_version, l.event_type, l.aggregate_type, l.aggregate_id,
  l.restaurant_id, l.device_id, l.node_device_id, l.client_device_id, l.shift_id, l.actor_employee_id,
  l.session_id, l.occurred_at, l.created_at, 'summary_without_payload' AS payload_policy,
  LENGTH(l.payload_json) AS payload_size_bytes
FROM local_event_log l
JOIN eligible_refs r ON r.aggregate_type = l.aggregate_type AND r.aggregate_id = l.aggregate_id
ORDER BY l.created_at, l.id`

const archiveOutboxSQL = `WITH ` + eligibleArchiveRefsSQL + `
SELECT
  o.id, o.command_id, o.sequence_no, o.origin, o.restaurant_id, o.device_id, o.node_device_id,
  o.client_device_id, o.actor_employee_id, o.session_id, o.aggregate_type, o.aggregate_id,
  o.command_type, o.sync_direction, o.status, o.attempts, o.next_retry_at, o.locked_at,
  o.locked_by, o.sent_at, o.last_error, o.created_at, o.updated_at,
  'summary_without_payload' AS payload_policy, LENGTH(o.payload_json) AS payload_size_bytes
FROM pos_sync_outbox o
JOIN eligible_refs r ON r.aggregate_type = o.aggregate_type AND r.aggregate_id = o.aggregate_id
ORDER BY o.sequence_no, o.id`

type archiveTableQuery struct {
	name  string
	query string
}

func (r *Repository) archiveRows(ctx context.Context, table, query, cutoffBusinessDateLocal string) ([]storage.ArchiveExportRow, error) {
	maps, err := r.queryRowsAsMaps(ctx, query, cutoffBusinessDateLocal)
	if err != nil {
		return nil, err
	}
	out := make([]storage.ArchiveExportRow, 0, len(maps))
	for _, row := range maps {
		out = append(out, storage.ArchiveExportRow{Table: table, Row: row})
	}
	return out, nil
}

func (r *Repository) queryRowsAsMaps(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, normalizeErr(err)
	}
	var out []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, normalizeErr(err)
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = normalizeArchiveValue(values[i])
		}
		out = append(out, row)
	}
	return out, normalizeErr(rows.Err())
}

func normalizeArchiveValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	default:
		return v
	}
}

func (r *Repository) archiveBusinessDateRange(ctx context.Context, cutoffBusinessDateLocal string) (storage.BusinessDateRange, error) {
	var oldest, newest sql.NullString
	if err := r.queryer(ctx).QueryRowContext(ctx, `
SELECT MIN(c.business_date_local), MAX(c.business_date_local)
FROM orders o
JOIN checks c ON c.order_id = o.id
WHERE o.status = 'closed' AND c.business_date_local < ?`, cutoffBusinessDateLocal).Scan(&oldest, &newest); err != nil {
		return storage.BusinessDateRange{}, normalizeErr(err)
	}
	return storage.BusinessDateRange{Oldest: oldest.String, Newest: newest.String}, nil
}

func (r *Repository) archivePlanBusinessDateRange(ctx context.Context, cutoffBusinessDateLocal string) (storage.BusinessDateRange, error) {
	var oldest, newest sql.NullString
	if err := r.queryer(ctx).QueryRowContext(ctx, `
SELECT MIN(c.business_date_local), MAX(c.business_date_local)
FROM orders o
JOIN checks c ON c.order_id = o.id
WHERE o.status = 'closed' AND c.business_date_local < ?`, cutoffBusinessDateLocal).Scan(&oldest, &newest); err != nil {
		return storage.BusinessDateRange{}, normalizeErr(err)
	}
	return storage.BusinessDateRange{Oldest: oldest.String, Newest: newest.String}, nil
}

func (r *Repository) archivePlanRestaurantID(ctx context.Context, cutoffBusinessDateLocal string) (string, error) {
	var restaurantID sql.NullString
	if err := r.queryer(ctx).QueryRowContext(ctx, `
SELECT MIN(o.restaurant_id)
FROM orders o
JOIN checks c ON c.order_id = o.id
WHERE o.status = 'closed' AND c.business_date_local < ?`, cutoffBusinessDateLocal).Scan(&restaurantID); err != nil {
		return "", normalizeErr(err)
	}
	if restaurantID.Valid && restaurantID.String != "" {
		return restaurantID.String, nil
	}
	if err := r.queryer(ctx).QueryRowContext(ctx, `SELECT MIN(id) FROM restaurants WHERE active = 1`).Scan(&restaurantID); err != nil {
		return "", normalizeErr(err)
	}
	return restaurantID.String, nil
}

func (r *Repository) blockingOutboxMessagesForArchiveScope(ctx context.Context, cutoffBusinessDateLocal string) (int, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `WITH `+eligibleArchiveRefsSQL+`
SELECT COUNT(1)
FROM pos_sync_outbox o
JOIN eligible_refs r ON r.aggregate_type = o.aggregate_type AND r.aggregate_id = o.aggregate_id
WHERE o.sync_direction = 'edge_to_cloud' AND o.status <> 'sent'`, cutoffBusinessDateLocal).Scan(&n)
	return n, normalizeErr(err)
}

func (r *Repository) storageArchiveSourceMetadata(ctx context.Context) (storage.ArchiveSourceMetadata, error) {
	sqliteStats, err := r.sqliteDatabaseStats(ctx)
	if err != nil {
		return storage.ArchiveSourceMetadata{}, err
	}
	var sourceNodeDeviceID, sourceDeviceCode, sourceDeviceName, sourceDeviceType, sourcePairedAt sql.NullString
	err = r.queryer(ctx).QueryRowContext(ctx, `
SELECT e.node_device_id, d.device_code, d.name, d.type, e.paired_at
FROM edge_node_identity e
LEFT JOIN devices d ON d.id = e.node_device_id
WHERE e.id = 'local'`).Scan(&sourceNodeDeviceID, &sourceDeviceCode, &sourceDeviceName, &sourceDeviceType, &sourcePairedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return storage.ArchiveSourceMetadata{}, normalizeErr(err)
	}
	runtimeRows, err := r.queryMetadataRows(ctx, `SELECT module_name, module_version, schema_version, checksum_sha256, status, applied_at, updated_at FROM db_runtime_versions ORDER BY module_name`)
	if err != nil {
		return storage.ArchiveSourceMetadata{}, err
	}
	schemaRows, err := r.queryMetadataRows(ctx, `SELECT version, checksum_sha256, status, applied_at FROM schema_migrations ORDER BY version`)
	if err != nil {
		return storage.ArchiveSourceMetadata{}, err
	}
	return storage.ArchiveSourceMetadata{
		SQLite:             sqliteStats,
		SourceNodeDeviceID: sourceNodeDeviceID.String,
		SourceDeviceCode:   sourceDeviceCode.String,
		SourceDeviceName:   sourceDeviceName.String,
		SourceDeviceType:   sourceDeviceType.String,
		SourcePairedAt:     sourcePairedAt.String,
		RuntimeVersions:    runtimeRows,
		SchemaMigrations:   schemaRows,
	}, nil
}

func (r *Repository) queryMetadataRows(ctx context.Context, query string) ([]storage.ArchiveMetadataRow, error) {
	rows, err := r.queryRowsAsMaps(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("storage archive metadata: %w", err)
	}
	out := make([]storage.ArchiveMetadataRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, storage.ArchiveMetadataRow(row))
	}
	return out, nil
}

func addArchiveCount(counts *storage.ArchiveExportCounts, table string, n int) {
	switch table {
	case "orders":
		counts.ClosedOrders += n
	case "order_lines":
		counts.OrderLines += n
	case "order_line_modifiers":
		counts.OrderLineModifiers += n
	case "order_line_discounts":
		counts.OrderLineDiscounts += n
	case "order_surcharges":
		counts.OrderSurcharges += n
	case "prechecks":
		counts.Prechecks += n
	case "precheck_lines":
		counts.PrecheckLines += n
	case "precheck_line_modifiers":
		counts.PrecheckLineModifiers += n
	case "precheck_discounts":
		counts.PrecheckDiscounts += n
	case "precheck_surcharges":
		counts.PrecheckSurcharges += n
	case "precheck_taxes":
		counts.PrecheckTaxes += n
	case "payments":
		counts.Payments += n
	case "payment_attempts":
		counts.PaymentAttempts += n
	case "checks":
		counts.Checks += n
	case "financial_operations":
		counts.FinancialOperations += n
	case "financial_operation_items":
		counts.FinancialOperationItems += n
	case "local_event_log_summary":
		counts.LocalEventReferences += n
	case "pos_sync_outbox_summary":
		counts.OutboxMessageReferences += n
	}
}

func archivePlanTableManifest(counts storage.RetentionEligibleCounts) []storage.ArchivePlanTableManifest {
	tables := []struct {
		name string
		rows int
	}{
		{name: "orders", rows: counts.ClosedOrders},
		{name: "order_lines", rows: counts.OrderLines},
		{name: "order_line_modifiers", rows: counts.OrderLineModifiers},
		{name: "prechecks", rows: counts.Prechecks},
		{name: "precheck_lines", rows: counts.PrecheckLines},
		{name: "precheck_line_modifiers", rows: counts.PrecheckLineModifiers},
		{name: "precheck_discounts", rows: counts.PrecheckDiscounts},
		{name: "precheck_surcharges", rows: counts.PrecheckSurcharges},
		{name: "precheck_taxes", rows: counts.PrecheckTaxes},
		{name: "payments", rows: counts.Payments},
		{name: "payment_attempts", rows: counts.PaymentAttempts},
		{name: "checks", rows: counts.Checks},
		{name: "financial_operations", rows: counts.FinancialOperations},
		{name: "financial_operation_items", rows: counts.FinancialOperationItems},
	}
	out := make([]storage.ArchivePlanTableManifest, 0, len(tables))
	for _, table := range tables {
		out = append(out, storage.ArchivePlanTableManifest{Name: table.name, Rows: table.rows, KeyField: "id"})
	}
	return out
}
