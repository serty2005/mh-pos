package sqlite

import (
	"context"
	"database/sql"

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
	if blocking > 0 {
		blockReasons = append(blockReasons, "pending_edge_to_cloud_outbox")
	}
	return storage.RetentionDryRunResult{
		CutoffBusinessDateLocal: cutoffBusinessDateLocal,
		Eligible:                eligible,
		ActiveOrders:            counts.OpenOrders + counts.LockedOrders,
		OpenShifts:              counts.OpenShifts,
		OpenCashSessions:        counts.OpenCashSessions,
		BlockingOutboxMessages:  blocking,
		BlockReasons:            blockReasons,
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
  (SELECT COUNT(1) FROM pos_sync_outbox),
  (SELECT COUNT(1) FROM stock_documents),
  (SELECT COUNT(1) FROM stock_moves)`).
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
			&counts.StockDocuments,
			&counts.StockMoves,
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

const eligibleOrdersSQL = `SELECT o.id FROM orders o JOIN checks c ON c.order_id = o.id WHERE o.status = 'closed' AND c.business_date_local < ?`
