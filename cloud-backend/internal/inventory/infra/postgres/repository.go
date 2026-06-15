package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/inventory/app"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ClaimPending(ctx context.Context, cmd app.ClaimCommand) ([]app.QueuedEvent, error) {
	limit := cmd.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	rows, err := r.pool.Query(ctx, `
WITH picked AS (
  SELECT id
  FROM inventory_event_queue
  WHERE status IN ('pending','failed')
    AND (next_retry_at IS NULL OR next_retry_at <= $1)
    AND NOT EXISTS (
      SELECT 1
      FROM inventory_event_queue q2
      WHERE q2.status = 'processing'
        AND q2.id <> inventory_event_queue.id
        AND q2.restaurant_id = inventory_event_queue.restaurant_id
        AND COALESCE(q2.warehouse_id, '') = COALESCE(inventory_event_queue.warehouse_id, '')
    )
  ORDER BY restaurant_id, COALESCE(warehouse_id, ''), occurred_at, id
  LIMIT $2
  FOR UPDATE SKIP LOCKED
)
UPDATE inventory_event_queue q
SET status = 'processing', locked_at = $1, locked_by = $3, updated_at = $1
FROM picked
WHERE q.id = picked.id
RETURNING q.id,q.receipt_id,q.restaurant_id,COALESCE(q.warehouse_id,''),q.device_id,q.event_id,q.event_type,q.aggregate_id,q.occurred_at,
  (SELECT raw_payload FROM cloud_edge_event_raw_payloads raw WHERE raw.receipt_id = q.receipt_id)`,
		cmd.Now, limit, strings.TrimSpace(cmd.LockedBy))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]app.QueuedEvent, 0, limit)
	for rows.Next() {
		var event app.QueuedEvent
		var eventType string
		var raw []byte
		if err := rows.Scan(&event.ID, &event.ReceiptID, &event.RestaurantID, &event.WarehouseID, &event.DeviceID, &event.EventID, &eventType, &event.AggregateID, &event.OccurredAt, &raw); err != nil {
			return nil, err
		}
		var envelope contracts.SyncEnvelope
		if err := json.Unmarshal(raw, &envelope); err == nil {
			event.Payload = append(json.RawMessage(nil), envelope.Payload...)
		}
		event.EventType = contracts.EventType(eventType)
		out = append(out, event)
	}
	return out, rows.Err()
}

func (r *Repository) BeginProcessingState(ctx context.Context, cmd app.ProcessingStateCommand) (app.ProcessingState, error) {
	now := cmd.Now.UTC()
	_, err := r.pool.Exec(ctx, `
INSERT INTO inventory_document_processing_state(
  id,restaurant_id,source_event_id,source_event_type,source_aggregate_id,status,
  posted_ledger_count,costing_status,needs_recalculation,created_at,updated_at
) VALUES (
  $1,$2,$3,$4,NULLIF($5,''),'accepted',0,'estimated',false,$6,$6
)
ON CONFLICT (restaurant_id, source_event_id, source_event_type) DO NOTHING`,
		strings.TrimSpace(cmd.ID),
		strings.TrimSpace(cmd.RestaurantID),
		strings.TrimSpace(cmd.SourceEventID),
		strings.TrimSpace(cmd.SourceEventType),
		strings.TrimSpace(cmd.SourceAggregateID),
		now,
	)
	if err != nil {
		return app.ProcessingState{}, err
	}
	return r.getProcessingState(ctx, cmd.RestaurantID, cmd.SourceEventID, cmd.SourceEventType)
}

func (r *Repository) CompleteProcessingState(ctx context.Context, cmd app.ProcessingStateCommand) error {
	now := cmd.Now.UTC()
	_, err := r.pool.Exec(ctx, `
UPDATE inventory_document_processing_state
SET stock_document_id = NULLIF($4,''),
    status = $5,
    posted_ledger_count = $6,
    expected_ledger_count = $7,
    costing_status = $8,
    needs_recalculation = $9,
    failure_code = NULL,
    failure_message_key = NULL,
    posted_at = $10,
    updated_at = $10
WHERE restaurant_id = $1 AND source_event_id = $2 AND source_event_type = $3`,
		strings.TrimSpace(cmd.RestaurantID),
		strings.TrimSpace(cmd.SourceEventID),
		strings.TrimSpace(cmd.SourceEventType),
		strings.TrimSpace(cmd.StockDocumentID),
		string(cmd.Status),
		cmd.PostedLedgerCount,
		cmd.ExpectedLedgerCount,
		strings.TrimSpace(cmd.CostingStatus),
		cmd.NeedsRecalculation,
		now,
	)
	return err
}

func (r *Repository) FailProcessingState(ctx context.Context, cmd app.ProcessingStateCommand) error {
	now := cmd.Now.UTC()
	_, err := r.pool.Exec(ctx, `
UPDATE inventory_document_processing_state
SET status = 'failed',
    failure_code = NULLIF($4,''),
    failure_message_key = NULLIF($5,''),
    updated_at = $6
WHERE restaurant_id = $1 AND source_event_id = $2 AND source_event_type = $3`,
		strings.TrimSpace(cmd.RestaurantID),
		strings.TrimSpace(cmd.SourceEventID),
		strings.TrimSpace(cmd.SourceEventType),
		strings.TrimSpace(cmd.FailureCode),
		strings.TrimSpace(cmd.FailureMessageKey),
		now,
	)
	return err
}

func (r *Repository) getProcessingState(ctx context.Context, restaurantID, sourceEventID, sourceEventType string) (app.ProcessingState, error) {
	var state app.ProcessingState
	var expected sql.NullInt64
	var postedAt *time.Time
	err := r.pool.QueryRow(ctx, `
SELECT id,restaurant_id,source_event_id,source_event_type,COALESCE(source_aggregate_id,''),COALESCE(stock_document_id,''),
       status,posted_ledger_count,expected_ledger_count,costing_status,needs_recalculation,
       COALESCE(failure_code,''),COALESCE(failure_message_key,''),created_at,updated_at,posted_at
FROM inventory_document_processing_state
WHERE restaurant_id = $1 AND source_event_id = $2 AND source_event_type = $3`,
		strings.TrimSpace(restaurantID), strings.TrimSpace(sourceEventID), strings.TrimSpace(sourceEventType),
	).Scan(
		&state.ID,
		&state.RestaurantID,
		&state.SourceEventID,
		&state.SourceEventType,
		&state.SourceAggregateID,
		&state.StockDocumentID,
		&state.Status,
		&state.PostedLedgerCount,
		&expected,
		&state.CostingStatus,
		&state.NeedsRecalculation,
		&state.FailureCode,
		&state.FailureMessageKey,
		&state.CreatedAt,
		&state.UpdatedAt,
		&postedAt,
	)
	if expected.Valid {
		v := int(expected.Int64)
		state.ExpectedLedgerCount = &v
	}
	state.PostedAt = postedAt
	return state, err
}

func (r *Repository) CreateStockDocument(ctx context.Context, document app.StockDocument) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if document.ProcessingState != nil {
		if done, err := ensureProcessingStateForDocument(ctx, tx, *document.ProcessingState, document); err != nil || done {
			if err != nil {
				return err
			}
			return tx.Commit(ctx)
		}
	}

	var existing string
	err = tx.QueryRow(ctx, `
SELECT id FROM stock_documents
WHERE source_event_id = $1 AND source_event_type = $2
LIMIT 1
FOR UPDATE`, document.SourceEventID, document.SourceEventType).Scan(&existing)
	if err == nil {
		if document.ProcessingState != nil {
			if err := updateProcessingStatePosted(ctx, tx, *document.ProcessingState, existing, ledgerCountForExistingDocument(ctx, tx, existing), document.Ledger, document.CreatedAt); err != nil {
				return err
			}
		}
		return tx.Commit(ctx)
	}
	if err != nil && !errorsIsNoRows(err) {
		return err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO stock_documents(id,restaurant_id,warehouse_id,document_type,source_event_id,source_event_type,business_date_local,occurred_at,created_at)
VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9)`,
		document.ID,
		document.RestaurantID,
		document.WarehouseID,
		string(document.Type),
		document.SourceEventID,
		document.SourceEventType,
		document.BusinessDateLocal,
		document.OccurredAt,
		document.CreatedAt,
	); err != nil {
		return err
	}
	for _, entry := range document.Ledger {
		if _, err := tx.Exec(ctx, `
INSERT INTO stock_ledger(
  id,restaurant_id,warehouse_id,stock_document_id,source_event_id,source_event_type,catalog_item_id,order_line_id,
  movement_type,quantity,unit_code,unit_cost_minor,total_cost_minor,costing_status,occurred_at,business_date_local,created_at
) VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,NULLIF($8,''),$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
			entry.ID,
			entry.RestaurantID,
			entry.WarehouseID,
			document.ID,
			entry.SourceEventID,
			entry.SourceEventType,
			entry.CatalogItemID,
			entry.OrderLineID,
			string(entry.MovementType),
			entry.Quantity,
			entry.UnitCode,
			entry.UnitCostMinor,
			entry.TotalCostMinor,
			entry.CostingStatus,
			entry.OccurredAt,
			entry.BusinessDateLocal,
			entry.CreatedAt,
		); err != nil {
			return err
		}
		if err := upsertStockBalance(ctx, tx, entry); err != nil {
			return err
		}
	}
	if document.ProcessingState != nil {
		if err := updateProcessingStatePosted(ctx, tx, *document.ProcessingState, document.ID, len(document.Ledger), document.Ledger, document.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) CreateRecalculationJob(ctx context.Context, cmd app.RecalculationTriggerCommand) error {
	directItems := make(map[string]bool)
	for _, entry := range cmd.Ledger {
		if item := strings.TrimSpace(entry.CatalogItemID); item != "" {
			directItems[item] = true
		}
	}
	if len(directItems) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	affectedItems, edges, err := recipeDependencyClosure(ctx, tx, strings.TrimSpace(cmd.RestaurantID), directItems)
	if err != nil {
		return err
	}
	itemIDs := mapKeys(affectedItems)
	rows, err := tx.Query(ctx, `
SELECT catalog_item_id,COALESCE(warehouse_id,''),unit_code,MIN(business_date_local)::text,MAX(business_date_local)::text,COUNT(1)
FROM stock_ledger
WHERE restaurant_id = $1
  AND occurred_at > $2
  AND catalog_item_id = ANY($3)
GROUP BY catalog_item_id,COALESCE(warehouse_id,''),unit_code
ORDER BY catalog_item_id,COALESCE(warehouse_id,''),unit_code`,
		strings.TrimSpace(cmd.RestaurantID),
		cmd.OccurredAt.UTC(),
		itemIDs,
	)
	if err != nil {
		return err
	}
	type affectedRange struct {
		catalogItemID string
		warehouseID   string
		unitCode      string
		from          string
		to            string
		count         int
	}
	ranges := make([]affectedRange, 0)
	totalSteps := 0
	warehouses := map[string]bool{}
	affectedCatalog := map[string]bool{}
	businessDateTo := strings.TrimSpace(cmd.BusinessDateFrom)
	for rows.Next() {
		var item affectedRange
		if err := rows.Scan(&item.catalogItemID, &item.warehouseID, &item.unitCode, &item.from, &item.to, &item.count); err != nil {
			rows.Close()
			return err
		}
		ranges = append(ranges, item)
		totalSteps += item.count
		warehouses[item.warehouseID] = true
		affectedCatalog[item.catalogItemID] = true
		if businessDateTo == "" || item.to > businessDateTo {
			businessDateTo = item.to
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if totalSteps == 0 {
		return tx.Commit(ctx)
	}
	now := cmd.Now.UTC()
	if businessDateTo == "" {
		businessDateTo = strings.TrimSpace(cmd.BusinessDateFrom)
	}
	var insertedID string
	err = tx.QueryRow(ctx, `
INSERT INTO stock_recalculation_jobs(
  id,restaurant_id,source_document_id,trigger_type,trigger_event_id,trigger_command_id,status,
  business_date_from,business_date_to,affected_catalog_item_count,affected_warehouse_count,total_steps,completed_steps,created_at,updated_at
) VALUES (
  $1,$2,NULLIF($3,''),$4,NULLIF($5,''),NULLIF($6,''),'queued',
  $7,$8,$9,$10,$11,0,$12,$12
)
ON CONFLICT DO NOTHING
RETURNING id`,
		strings.TrimSpace(cmd.ID),
		strings.TrimSpace(cmd.RestaurantID),
		strings.TrimSpace(cmd.SourceDocumentID),
		strings.TrimSpace(cmd.TriggerType),
		strings.TrimSpace(cmd.TriggerEventID),
		strings.TrimSpace(cmd.TriggerCommandID),
		strings.TrimSpace(cmd.BusinessDateFrom),
		businessDateTo,
		len(affectedCatalog),
		len(warehouses),
		totalSteps,
		now,
	).Scan(&insertedID)
	if errorsIsNoRows(err) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	for _, item := range ranges {
		if _, err := tx.Exec(ctx, `
INSERT INTO stock_recalculation_job_items(job_id,catalog_item_id,warehouse_id,unit_code,business_date_from,business_date_to,created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT DO NOTHING`,
			insertedID, item.catalogItemID, item.warehouseID, item.unitCode, item.from, item.to, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
UPDATE stock_ledger
SET costing_status = 'needs_recalculation'
WHERE restaurant_id = $1
  AND catalog_item_id = $2
  AND COALESCE(warehouse_id,'') = $3
  AND unit_code = $4
  AND business_date_local BETWEEN $5::date AND $6::date
  AND costing_status <> 'failed'`,
			strings.TrimSpace(cmd.RestaurantID), item.catalogItemID, item.warehouseID, item.unitCode, item.from, item.to); err != nil {
			return err
		}
		if err := refreshStockBalanceCosting(ctx, tx, strings.TrimSpace(cmd.RestaurantID), item.warehouseID, item.catalogItemID, item.unitCode, now); err != nil {
			return err
		}
	}
	for i, edge := range edges {
		if _, err := tx.Exec(ctx, `
INSERT INTO stock_recalculation_edges(job_id,dependency_catalog_item_id,dependent_catalog_item_id,edge_type,sort_order,created_at)
VALUES ($1,$2,$3,'recipe',$4,$5)
ON CONFLICT DO NOTHING`,
			insertedID, edge.from, edge.to, i, now); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

type recipeEdge struct {
	from string
	to   string
}

func recipeDependencyClosure(ctx context.Context, tx pgx.Tx, restaurantID string, direct map[string]bool) (map[string]bool, []recipeEdge, error) {
	rows, err := tx.Query(ctx, `
SELECT rl.component_catalog_item_id, rv.owner_catalog_item_id
FROM cloud_recipe_versions rv
JOIN cloud_recipe_lines rl ON rl.recipe_version_id = rv.id
WHERE rv.restaurant_id = $1 AND rv.status = 'active'
UNION
SELECT component_catalog_item_id, recipe_owner_catalog_item_id
FROM cloud_recipe_items
WHERE restaurant_id = $1
ORDER BY 1,2`, restaurantID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	graph := map[string][]string{}
	for rows.Next() {
		var from, to string
		if err := rows.Scan(&from, &to); err != nil {
			return nil, nil, err
		}
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		if from != "" && to != "" {
			graph[from] = append(graph[from], to)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	affected := map[string]bool{}
	edges := make([]recipeEdge, 0)
	var visit func(string)
	visit = func(item string) {
		if affected[item] {
			return
		}
		affected[item] = true
		for _, dependent := range graph[item] {
			edges = append(edges, recipeEdge{from: item, to: dependent})
			visit(dependent)
		}
	}
	for item := range direct {
		visit(item)
	}
	return affected, edges, nil
}

func mapKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return out
}

func (r *Repository) ClaimRecalculationJob(ctx context.Context, cmd app.RecalculationClaimCommand) (app.RecalculationJob, bool, error) {
	var job app.RecalculationJob
	var status string
	err := r.pool.QueryRow(ctx, `
WITH picked AS (
  SELECT id
  FROM stock_recalculation_jobs
  WHERE status = 'queued'
  ORDER BY created_at, id
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
UPDATE stock_recalculation_jobs j
SET status = 'running',
    started_at = COALESCE(started_at, $1),
    updated_at = $1,
    failure_code = NULL,
    failure_message_key = NULL
FROM picked
WHERE j.id = picked.id
RETURNING j.id,j.restaurant_id,j.trigger_type,COALESCE(j.trigger_event_id,''),COALESCE(j.trigger_command_id,''),j.status,j.total_steps,j.completed_steps`,
		cmd.Now.UTC(),
	).Scan(&job.ID, &job.RestaurantID, &job.TriggerType, &job.TriggerEventID, &job.TriggerCommandID, &status, &job.TotalSteps, &job.CompletedSteps)
	if errorsIsNoRows(err) {
		return app.RecalculationJob{}, false, nil
	}
	if err != nil {
		return app.RecalculationJob{}, false, err
	}
	job.Status = app.RecalculationJobStatus(status)
	return job, true, nil
}

func (r *Repository) ValidateRecalculationDAG(ctx context.Context, jobID string) error {
	rows, err := r.pool.Query(ctx, `
SELECT dependency_catalog_item_id,dependent_catalog_item_id
FROM stock_recalculation_edges
WHERE job_id = $1
ORDER BY sort_order,dependency_catalog_item_id,dependent_catalog_item_id`, strings.TrimSpace(jobID))
	if err != nil {
		return err
	}
	defer rows.Close()
	graph := map[string][]string{}
	for rows.Next() {
		var from, to string
		if err := rows.Scan(&from, &to); err != nil {
			return err
		}
		graph[from] = append(graph[from], to)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) bool
	visit = func(node string) bool {
		if visiting[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visiting[node] = true
		for _, next := range graph[node] {
			if visit(next) {
				return true
			}
		}
		delete(visiting, node)
		visited[node] = true
		return false
	}
	for node := range graph {
		if visit(node) {
			return fmt.Errorf("recipe dependency cycle")
		}
	}
	return nil
}

func (r *Repository) ListRecalculationLedgerRows(ctx context.Context, jobID string) ([]app.RecalculationLedgerRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT l.id,l.restaurant_id,COALESCE(l.warehouse_id,''),l.catalog_item_id,l.movement_type,l.quantity::text,l.unit_code,l.unit_cost_minor,l.occurred_at
FROM stock_ledger l
JOIN stock_recalculation_job_items i
  ON i.job_id = $1
 AND i.catalog_item_id = l.catalog_item_id
 AND i.warehouse_id = COALESCE(l.warehouse_id,'')
 AND i.unit_code = l.unit_code
 AND l.business_date_local BETWEEN i.business_date_from AND i.business_date_to
JOIN stock_recalculation_jobs j ON j.id = i.job_id AND j.restaurant_id = l.restaurant_id
WHERE j.id = $1
ORDER BY l.business_date_local ASC,l.occurred_at ASC,l.id ASC`, strings.TrimSpace(jobID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]app.RecalculationLedgerRow, 0)
	for rows.Next() {
		var row app.RecalculationLedgerRow
		var movement string
		if err := rows.Scan(&row.ID, &row.RestaurantID, &row.WarehouseID, &row.CatalogItemID, &movement, &row.Quantity, &row.UnitCode, &row.UnitCostMinor, &row.OccurredAt); err != nil {
			return nil, err
		}
		row.MovementType = app.MovementType(movement)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) LatestCostBasis(ctx context.Context, q app.CostBasisQuery) (int64, bool, error) {
	var cost int64
	err := r.pool.QueryRow(ctx, `
SELECT unit_cost_minor
FROM stock_ledger
WHERE restaurant_id = $1
  AND COALESCE(warehouse_id,'') = $2
  AND catalog_item_id = $3
  AND unit_code = $4
  AND movement_type = 'IN'
  AND unit_cost_minor > 0
  AND (occurred_at < $5 OR (occurred_at = $5 AND id < $6))
ORDER BY occurred_at DESC,id DESC
LIMIT 1`,
		strings.TrimSpace(q.RestaurantID),
		strings.TrimSpace(q.WarehouseID),
		strings.TrimSpace(q.CatalogItemID),
		strings.TrimSpace(q.UnitCode),
		q.OccurredAt.UTC(),
		strings.TrimSpace(q.LedgerID),
	).Scan(&cost)
	if errorsIsNoRows(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return cost, true, nil
}

func (r *Repository) UpdateRecalculationLedgerRow(ctx context.Context, update app.RecalculationLedgerUpdate) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var restaurantID, warehouseID, catalogItemID, unitCode string
	if err := tx.QueryRow(ctx, `
UPDATE stock_ledger
SET unit_cost_minor = $2,
    total_cost_minor = $3,
    costing_status = $4
WHERE id = $1
RETURNING restaurant_id,COALESCE(warehouse_id,''),catalog_item_id,unit_code`,
		strings.TrimSpace(update.LedgerID),
		update.UnitCostMinor,
		update.TotalCostMinor,
		strings.TrimSpace(update.CostingStatus),
	).Scan(&restaurantID, &warehouseID, &catalogItemID, &unitCode); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
UPDATE stock_recalculation_jobs
SET completed_steps = $2, updated_at = $3
WHERE id = $1`,
		strings.TrimSpace(update.JobID), update.CompletedSteps, update.Now.UTC()); err != nil {
		return err
	}
	if err := refreshStockBalanceCosting(ctx, tx, restaurantID, warehouseID, catalogItemID, unitCode, update.Now.UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) CompleteRecalculationJob(ctx context.Context, progress app.RecalculationJobProgress) error {
	_, err := r.pool.Exec(ctx, `
UPDATE stock_recalculation_jobs
SET status = 'completed',
    total_steps = $2,
    completed_steps = $3,
    failure_code = NULL,
    failure_message_key = NULL,
    finished_at = $4,
    updated_at = $4
WHERE id = $1`,
		strings.TrimSpace(progress.JobID), progress.TotalSteps, progress.CompletedSteps, progress.Now.UTC())
	return err
}

func (r *Repository) FailRecalculationJob(ctx context.Context, failure app.RecalculationJobFailure) error {
	_, err := r.pool.Exec(ctx, `
UPDATE stock_recalculation_jobs
SET status = 'failed',
    completed_steps = $2,
    failure_code = NULLIF($3,''),
    failure_message_key = NULLIF($4,''),
    finished_at = $5,
    updated_at = $5
WHERE id = $1`,
		strings.TrimSpace(failure.JobID),
		failure.CompletedSteps,
		strings.TrimSpace(failure.FailureCode),
		strings.TrimSpace(failure.FailureMessageKey),
		failure.Now.UTC())
	return err
}

func refreshStockBalanceCosting(ctx context.Context, tx pgx.Tx, restaurantID, warehouseID, catalogItemID, unitCode string, now time.Time) error {
	_, err := tx.Exec(ctx, `
WITH aggregate AS (
  SELECT
    CASE
      WHEN BOOL_OR(costing_status = 'failed') THEN 'failed'
      WHEN BOOL_OR(costing_status = 'needs_recalculation') THEN 'needs_recalculation'
      WHEN BOOL_OR(costing_status = 'estimated') THEN 'estimated'
      WHEN (ARRAY_AGG(costing_status ORDER BY occurred_at DESC, id DESC))[1] = 'recalculated' THEN 'recalculated'
      ELSE 'final'
    END AS costing_status
  FROM stock_ledger
  WHERE restaurant_id = $1
    AND COALESCE(warehouse_id,'') = $2
    AND catalog_item_id = $3
    AND unit_code = $4
)
UPDATE inventory_stock_balances
SET costing_status = (SELECT costing_status FROM aggregate),
    needs_recalculation = (SELECT costing_status IN ('needs_recalculation','estimated') FROM aggregate),
    updated_at = $5
WHERE restaurant_id = $1
  AND warehouse_id = $2
  AND catalog_item_id = $3
  AND unit_code = $4`,
		strings.TrimSpace(restaurantID),
		strings.TrimSpace(warehouseID),
		strings.TrimSpace(catalogItemID),
		strings.TrimSpace(unitCode),
		now.UTC(),
	)
	return err
}

func ensureProcessingStateForDocument(ctx context.Context, tx pgx.Tx, cmd app.ProcessingStateCommand, document app.StockDocument) (bool, error) {
	now := cmd.Now.UTC()
	if now.IsZero() {
		now = document.CreatedAt.UTC()
	}
	_, err := tx.Exec(ctx, `
INSERT INTO inventory_document_processing_state(
  id,restaurant_id,source_event_id,source_event_type,source_aggregate_id,status,
  posted_ledger_count,costing_status,needs_recalculation,created_at,updated_at
) VALUES (
  $1,$2,$3,$4,NULLIF($5,''),'accepted',0,'estimated',false,$6,$6
)
ON CONFLICT (restaurant_id, source_event_id, source_event_type) DO NOTHING`,
		strings.TrimSpace(cmd.ID),
		strings.TrimSpace(cmd.RestaurantID),
		strings.TrimSpace(cmd.SourceEventID),
		strings.TrimSpace(cmd.SourceEventType),
		strings.TrimSpace(cmd.SourceAggregateID),
		now,
	)
	if err != nil {
		return false, err
	}
	var status string
	err = tx.QueryRow(ctx, `
SELECT status
FROM inventory_document_processing_state
WHERE restaurant_id = $1 AND source_event_id = $2 AND source_event_type = $3
FOR UPDATE`,
		strings.TrimSpace(cmd.RestaurantID),
		strings.TrimSpace(cmd.SourceEventID),
		strings.TrimSpace(cmd.SourceEventType),
	).Scan(&status)
	if err != nil {
		return false, err
	}
	switch app.ProcessingStatus(status) {
	case app.ProcessingStatusPosted, app.ProcessingStatusPartiallyPosted, app.ProcessingStatusFailed:
		return true, nil
	default:
		return false, nil
	}
}

func ledgerCountForExistingDocument(ctx context.Context, tx pgx.Tx, documentID string) int {
	var count int
	_ = tx.QueryRow(ctx, `SELECT COUNT(1) FROM stock_ledger WHERE stock_document_id = $1`, strings.TrimSpace(documentID)).Scan(&count)
	return count
}

func updateProcessingStatePosted(ctx context.Context, tx pgx.Tx, cmd app.ProcessingStateCommand, documentID string, postedCount int, ledger []app.StockLedgerEntry, now time.Time) error {
	expectedCount := len(ledger)
	status := app.ProcessingStatusPosted
	if postedCount < expectedCount {
		status = app.ProcessingStatusPartiallyPosted
	}
	costingStatus, needsRecalculation := aggregateCostingStatus(ledger)
	_, err := tx.Exec(ctx, `
UPDATE inventory_document_processing_state
SET stock_document_id = $4,
    status = $5,
    posted_ledger_count = $6,
    expected_ledger_count = $7,
    costing_status = $8,
    needs_recalculation = $9,
    failure_code = NULL,
    failure_message_key = NULL,
    posted_at = $10,
    updated_at = $10
WHERE restaurant_id = $1 AND source_event_id = $2 AND source_event_type = $3`,
		strings.TrimSpace(cmd.RestaurantID),
		strings.TrimSpace(cmd.SourceEventID),
		strings.TrimSpace(cmd.SourceEventType),
		strings.TrimSpace(documentID),
		string(status),
		postedCount,
		expectedCount,
		costingStatus,
		needsRecalculation,
		now.UTC(),
	)
	return err
}

func aggregateCostingStatus(ledger []app.StockLedgerEntry) (string, bool) {
	status := "final"
	for _, entry := range ledger {
		switch strings.TrimSpace(entry.CostingStatus) {
		case "failed":
			return "failed", false
		case "needs_recalculation":
			status = "needs_recalculation"
		case "estimated":
			if status != "needs_recalculation" {
				status = "estimated"
			}
		case "recalculated":
			if status == "final" {
				status = "recalculated"
			}
		}
	}
	return status, status == "needs_recalculation" || status == "estimated"
}

func upsertStockBalance(ctx context.Context, tx pgx.Tx, entry app.StockLedgerEntry) error {
	_, err := tx.Exec(ctx, `
WITH aggregate AS (
  SELECT
    CASE
      WHEN BOOL_OR(costing_status = 'failed') THEN 'failed'
      WHEN BOOL_OR(costing_status = 'needs_recalculation') THEN 'needs_recalculation'
      WHEN BOOL_OR(costing_status = 'estimated') THEN 'estimated'
      WHEN (ARRAY_AGG(costing_status ORDER BY occurred_at DESC, id DESC))[1] = 'recalculated' THEN 'recalculated'
      ELSE 'final'
    END AS costing_status
  FROM stock_ledger
  WHERE restaurant_id = $1
    AND COALESCE(warehouse_id, '') = $2
    AND catalog_item_id = $3
    AND unit_code = $4
)
INSERT INTO inventory_stock_balances(
  restaurant_id,warehouse_id,catalog_item_id,unit_code,quantity_on_hand,last_movement_at,last_ledger_entry_id,
  costing_status,needs_recalculation,created_at,updated_at
) VALUES (
  $1,$2,$3,$4,
  CASE WHEN $5 = 'IN' THEN $6::numeric ELSE -$6::numeric END,
  $7,$8,
  (SELECT costing_status FROM aggregate),
  (SELECT costing_status IN ('needs_recalculation','estimated') FROM aggregate),
  $9,$9
)
ON CONFLICT (restaurant_id, warehouse_id, catalog_item_id, unit_code) DO UPDATE SET
  quantity_on_hand = inventory_stock_balances.quantity_on_hand + EXCLUDED.quantity_on_hand,
  last_movement_at = EXCLUDED.last_movement_at,
  last_ledger_entry_id = EXCLUDED.last_ledger_entry_id,
  costing_status = EXCLUDED.costing_status,
  needs_recalculation = EXCLUDED.needs_recalculation,
  updated_at = EXCLUDED.updated_at`,
		strings.TrimSpace(entry.RestaurantID),
		strings.TrimSpace(entry.WarehouseID),
		strings.TrimSpace(entry.CatalogItemID),
		strings.TrimSpace(entry.UnitCode),
		string(entry.MovementType),
		strings.TrimSpace(entry.Quantity),
		entry.OccurredAt,
		strings.TrimSpace(entry.ID),
		entry.CreatedAt,
	)
	return err
}

func (r *Repository) ApplyStopListUpdate(ctx context.Context, cmd app.StopListProjectionCommand) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	action := stopListProjectionAction(cmd.ConflictPolicy)
	var quantity any
	if strings.TrimSpace(cmd.AvailableQuantity) != "" {
		quantity = strings.TrimSpace(cmd.AvailableQuantity)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO cloud_projection_stop_list_updates(
  source_event_id,queue_id,restaurant_id,device_id,stop_list_id,warehouse_id,catalog_item_id,
  available_quantity,active,conflict_policy,source,reason,projection_action,updated_at,occurred_at,projected_at
) VALUES (
  $1,$2,$3,$4,$5,NULLIF($6,''),$7,
  $8,$9,$10,$11,NULLIF($12,''),$13,$14,$15,$16
)
ON CONFLICT (source_event_id) DO NOTHING`,
		strings.TrimSpace(cmd.SourceEventID),
		strings.TrimSpace(cmd.QueueID),
		strings.TrimSpace(cmd.RestaurantID),
		strings.TrimSpace(cmd.DeviceID),
		strings.TrimSpace(cmd.StopListID),
		strings.TrimSpace(cmd.WarehouseID),
		strings.TrimSpace(cmd.CatalogItemID),
		quantity,
		cmd.Active,
		string(cmd.ConflictPolicy),
		strings.TrimSpace(cmd.Source),
		strings.TrimSpace(cmd.Reason),
		action,
		cmd.UpdatedAt,
		cmd.OccurredAt,
		cmd.ProjectedAt,
	); err != nil {
		return err
	}
	if cmd.ConflictPolicy == contracts.StopListConflictPolicyEdgeOverlayUntilNextPublication {
		if _, err := tx.Exec(ctx, `
INSERT INTO stop_lists(id,restaurant_id,catalog_item_id,available_quantity,source,reason,active,cloud_version,updated_at)
VALUES ($1,$2,$3,$4,'edge_overlay',NULLIF($5,''),$6,NULL,$7)
ON CONFLICT (restaurant_id, catalog_item_id) DO UPDATE SET
  id = EXCLUDED.id,
  available_quantity = EXCLUDED.available_quantity,
  source = EXCLUDED.source,
  reason = EXCLUDED.reason,
  active = EXCLUDED.active,
  cloud_version = NULL,
  updated_at = EXCLUDED.updated_at`,
			strings.TrimSpace(cmd.StopListID),
			strings.TrimSpace(cmd.RestaurantID),
			strings.TrimSpace(cmd.CatalogItemID),
			quantity,
			strings.TrimSpace(cmd.Reason),
			cmd.Active,
			cmd.UpdatedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func stopListProjectionAction(policy contracts.StopListConflictPolicy) string {
	switch policy {
	case contracts.StopListConflictPolicyCloudWins:
		return "ignored_cloud_wins"
	case contracts.StopListConflictPolicyEdgeOverlayUntilNextPublication:
		return "applied_edge_overlay"
	default:
		return "requires_manager_review"
	}
}

func (r *Repository) MarkProcessed(ctx context.Context, queueID string, now time.Time) error {
	_, err := r.pool.Exec(ctx, `
UPDATE inventory_event_queue
SET status = 'processed', processed_at = $2, locked_at = NULL, locked_by = NULL, last_error = NULL, updated_at = $2
WHERE id = $1`, queueID, now)
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, queueID, reason string, now time.Time) error {
	_, err := r.pool.Exec(ctx, `
UPDATE inventory_event_queue
SET status = 'failed', attempts = attempts + 1, locked_at = NULL, locked_by = NULL, last_error = $2, updated_at = $3
WHERE id = $1`, queueID, reason, now)
	return err
}

func errorsIsNoRows(err error) bool {
	return err == pgx.ErrNoRows
}

func (r *Repository) ListActiveRecipeLines(ctx context.Context, restaurantID, catalogItemID string) ([]app.RecipeLine, error) {
	rows, err := r.pool.Query(ctx, `
SELECT rl.component_catalog_item_id, rl.quantity::text, rl.unit
FROM cloud_recipe_versions rv
JOIN cloud_recipe_lines rl ON rl.recipe_version_id = rv.id
WHERE rv.restaurant_id = $1
  AND rv.owner_catalog_item_id = $2
  AND rv.status = 'active'
ORDER BY rl.sort_order, rl.id`, strings.TrimSpace(restaurantID), strings.TrimSpace(catalogItemID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lines := make([]app.RecipeLine, 0)
	for rows.Next() {
		var line app.RecipeLine
		if err := rows.Scan(&line.ComponentCatalogItemID, &line.Quantity, &line.UnitCode); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil || len(lines) > 0 {
		return lines, err
	}
	return r.listLegacyRecipeItems(ctx, restaurantID, catalogItemID)
}

func (r *Repository) listLegacyRecipeItems(ctx context.Context, restaurantID, catalogItemID string) ([]app.RecipeLine, error) {
	rows, err := r.pool.Query(ctx, `
SELECT component_catalog_item_id, quantity::text, unit
FROM cloud_recipe_items
WHERE restaurant_id = $1 AND recipe_owner_catalog_item_id = $2
ORDER BY component_catalog_item_id`, strings.TrimSpace(restaurantID), strings.TrimSpace(catalogItemID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lines := make([]app.RecipeLine, 0)
	for rows.Next() {
		var line app.RecipeLine
		if err := rows.Scan(&line.ComponentCatalogItemID, &line.Quantity, &line.UnitCode); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	return lines, rows.Err()
}

func (r *Repository) ListModifierOptionLinks(ctx context.Context, restaurantID string, optionIDs []string) (map[string]string, error) {
	if len(optionIDs) == 0 {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(optionIDs))
	hasLinkedCatalogItemID, err := r.columnExists(ctx, "cloud_modifier_options", "linked_catalog_item_id")
	if err != nil {
		return nil, err
	}
	if !hasLinkedCatalogItemID {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, COALESCE(linked_catalog_item_id,'')
FROM cloud_modifier_options
WHERE restaurant_id = $1
  AND id = ANY($2)`, strings.TrimSpace(restaurantID), optionIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var linked string
		if err := rows.Scan(&id, &linked); err != nil {
			return nil, err
		}
		out[id] = strings.TrimSpace(linked)
	}
	return out, rows.Err()
}

func (r *Repository) columnExists(ctx context.Context, tableName, columnName string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.columns
  WHERE table_schema = 'public'
    AND table_name = $1
    AND column_name = $2
)`, tableName, columnName).Scan(&exists)
	return exists, err
}

func (r *Repository) ListServedOrderLineQuantities(ctx context.Context, restaurantID string, orderLineIDs []string) (map[string]string, error) {
	if len(orderLineIDs) == 0 {
		return map[string]string{}, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT order_line_id,
       SUM(CASE
         WHEN source_event_type = $3 AND movement_type = 'OUT' THEN quantity
         WHEN source_event_type = $4 AND movement_type = 'IN' THEN -quantity
         ELSE 0
       END)::text
FROM stock_ledger
WHERE restaurant_id = $1
  AND order_line_id = ANY($2)
  AND source_event_type IN ($3, $4)
GROUP BY order_line_id`, strings.TrimSpace(restaurantID), orderLineIDs, string(contracts.EventItemServed), app.SourceEventItemServedCompensation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string, len(orderLineIDs))
	for rows.Next() {
		var orderLineID string
		var quantity string
		if err := rows.Scan(&orderLineID, &quantity); err != nil {
			return nil, err
		}
		out[strings.TrimSpace(orderLineID)] = strings.TrimSpace(quantity)
	}
	return out, rows.Err()
}

func (r *Repository) GetCurrentQuantity(ctx context.Context, restaurantID, warehouseID, catalogItemID, unitCode string) (string, error) {
	var quantity string
	err := r.pool.QueryRow(ctx, `
SELECT quantity_on_hand::text
FROM inventory_stock_balances
WHERE restaurant_id = $1
  AND warehouse_id = $2
  AND catalog_item_id = $3
  AND unit_code = $4`,
		strings.TrimSpace(restaurantID),
		strings.TrimSpace(warehouseID),
		strings.TrimSpace(catalogItemID),
		strings.TrimSpace(unitCode),
	).Scan(&quantity)
	if errorsIsNoRows(err) {
		return "0.000", nil
	}
	return strings.TrimSpace(quantity), err
}

func (r *Repository) HasSupersedingServedEvent(ctx context.Context, restaurantID, orderLineID, servedEventID string) (bool, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	orderLineID = strings.TrimSpace(orderLineID)
	servedEventID = strings.TrimSpace(servedEventID)
	if restaurantID == "" || orderLineID == "" || servedEventID == "" {
		return false, nil
	}
	var exists bool
	err := r.pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM inbox_events
  WHERE restaurant_id = $1
    AND event_type = $2
    AND raw_payload->'payload'->'data'->>'order_line_id' = $3
    AND raw_payload->'payload'->'data'->>'supersedes_served_event_id' = $4
)`, restaurantID, string(contracts.EventItemServed), orderLineID, servedEventID).Scan(&exists)
	return exists, err
}
