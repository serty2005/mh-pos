package clickhouse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud-backend/internal/olap/app"
)

const (
	salesKitchenSaleEventTypesSQL    = "('CheckCreated','CheckClosed','PaymentCaptured','CancellationRecorded','RefundRecorded')"
	salesKitchenKitchenEventTypesSQL = "('KitchenTicketStatusChanged','ItemServed','StockReceiptCaptured','InventoryCountCaptured','StockWriteOffCaptured','ProductionCompleted','CatalogItemChangeSuggested','RecipeChangeSuggested','StopListUpdated')"
)

type Config struct {
	URL      string
	Database string
	Username string
	Password string
}

type Repository struct {
	endpoint string
	database string
	username string
	password string
	client   *http.Client
}

func NewRepository(config Config) *Repository {
	database := strings.TrimSpace(config.Database)
	if database == "" {
		database = "mh_pos_cloud"
	}
	return &Repository{
		endpoint: strings.TrimRight(strings.TrimSpace(config.URL), "/"),
		database: database,
		username: strings.TrimSpace(config.Username),
		password: config.Password,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (r *Repository) Migrate(ctx context.Context) error {
	if r == nil || r.endpoint == "" {
		return app.ErrOLAPUnavailable
	}
	if err := r.exec(ctx, fmt.Sprintf(`CREATE DATABASE IF NOT EXISTS %s`, ident(r.database))); err != nil {
		return err
	}
	if err := r.exec(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.raw_business_events (
  event_id String,
  tenant_id String,
  restaurant_id String,
  device_id String,
  employee_id String,
  event_type LowCardinality(String),
  occurred_at DateTime64(3, 'UTC'),
  cloud_received_at DateTime64(3, 'UTC'),
  raw_payload_sha256_hex String,
  payload String,
  exported_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(exported_at)
PARTITION BY toYYYYMM(occurred_at)
ORDER BY (tenant_id, event_type, event_id)`, ident(r.database))); err != nil {
		return err
	}
	return r.exec(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.olap_stock_moves (
  ledger_entry_id String,
  restaurant_id String,
  warehouse_id String,
  stock_document_id String,
  source_event_id String,
  source_event_type LowCardinality(String),
  catalog_item_id String,
  order_line_id String,
  movement_type LowCardinality(String),
  quantity String,
  unit_code LowCardinality(String),
  unit_cost_minor Int64,
  total_cost_minor Int64,
  costing_status LowCardinality(String),
  occurred_at DateTime64(3, 'UTC'),
  business_date_local Date,
  ledger_created_at DateTime64(3, 'UTC'),
  exported_at DateTime64(3, 'UTC')
) ENGINE = ReplacingMergeTree(exported_at)
PARTITION BY toYYYYMM(occurred_at)
ORDER BY (restaurant_id, business_date_local, catalog_item_id, warehouse_id, ledger_entry_id)`, ident(r.database)))
}

func (r *Repository) InsertRawBusinessEvents(ctx context.Context, events []app.InboxEvent, exportedAt time.Time) error {
	if len(events) == 0 {
		return nil
	}
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("INSERT INTO %s.raw_business_events FORMAT JSONEachRow\n", ident(r.database)))
	enc := json.NewEncoder(&body)
	for _, event := range events {
		row := map[string]string{
			"event_id":               event.EventID,
			"tenant_id":              event.TenantID,
			"restaurant_id":          event.RestaurantID,
			"device_id":              event.DeviceID,
			"employee_id":            event.EmployeeID,
			"event_type":             event.EventType,
			"occurred_at":            chTime(event.OccurredAt),
			"cloud_received_at":      chTime(event.CloudReceivedAt),
			"raw_payload_sha256_hex": event.RawPayloadSHA256Hex,
			"payload":                string(event.RawPayload),
			"exported_at":            chTime(exportedAt),
		}
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return r.post(ctx, body.String())
}

func (r *Repository) InsertStockMoves(ctx context.Context, moves []app.StockMove, exportedAt time.Time) error {
	if len(moves) == 0 {
		return nil
	}
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("INSERT INTO %s.olap_stock_moves FORMAT JSONEachRow\n", ident(r.database)))
	enc := json.NewEncoder(&body)
	for _, move := range moves {
		row := map[string]any{
			"ledger_entry_id":     move.LedgerEntryID,
			"restaurant_id":       move.RestaurantID,
			"warehouse_id":        move.WarehouseID,
			"stock_document_id":   move.StockDocumentID,
			"source_event_id":     move.SourceEventID,
			"source_event_type":   move.SourceEventType,
			"catalog_item_id":     move.CatalogItemID,
			"order_line_id":       move.OrderLineID,
			"movement_type":       move.MovementType,
			"quantity":            move.Quantity,
			"unit_code":           move.UnitCode,
			"unit_cost_minor":     move.UnitCostMinor,
			"total_cost_minor":    move.TotalCostMinor,
			"costing_status":      move.CostingStatus,
			"occurred_at":         chTime(move.OccurredAt),
			"business_date_local": move.BusinessDateLocal,
			"ledger_created_at":   chTime(move.LedgerCreatedAt),
			"exported_at":         chTime(exportedAt),
		}
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return r.post(ctx, body.String())
}

func (r *Repository) ListRawBusinessEvents(ctx context.Context, filter app.RawBusinessEventFilter) ([]app.RawBusinessEvent, error) {
	query := strings.Builder{}
	query.WriteString("SELECT event_id,tenant_id,restaurant_id,device_id,employee_id,event_type,occurred_at,cloud_received_at,raw_payload_sha256_hex FROM ")
	query.WriteString(ident(r.database))
	query.WriteString(".raw_business_events FINAL WHERE 1=1")
	if filter.RestaurantID != "" {
		query.WriteString(" AND restaurant_id = ")
		query.WriteString(quote(filter.RestaurantID))
	}
	if filter.EventType != "" {
		query.WriteString(" AND event_type = ")
		query.WriteString(quote(filter.EventType))
	}
	if filter.OccurredFrom != nil {
		query.WriteString(" AND occurred_at >= parseDateTime64BestEffort(")
		query.WriteString(quote(filter.OccurredFrom.UTC().Format(time.RFC3339Nano)))
		query.WriteString(", 3, 'UTC')")
	}
	if filter.OccurredTo != nil {
		query.WriteString(" AND occurred_at <= parseDateTime64BestEffort(")
		query.WriteString(quote(filter.OccurredTo.UTC().Format(time.RFC3339Nano)))
		query.WriteString(", 3, 'UTC')")
	}
	query.WriteString(fmt.Sprintf(" ORDER BY occurred_at DESC, event_id DESC LIMIT %d OFFSET %d FORMAT JSONEachRow", filter.Limit, filter.Offset))

	respBody, err := r.query(ctx, query.String())
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	rows := make([]app.RawBusinessEvent, 0, filter.Limit)
	scanner := bufio.NewScanner(respBody)
	for scanner.Scan() {
		var row struct {
			EventID             string `json:"event_id"`
			TenantID            string `json:"tenant_id"`
			RestaurantID        string `json:"restaurant_id"`
			DeviceID            string `json:"device_id"`
			EmployeeID          string `json:"employee_id"`
			EventType           string `json:"event_type"`
			OccurredAt          string `json:"occurred_at"`
			CloudReceivedAt     string `json:"cloud_received_at"`
			RawPayloadSHA256Hex string `json:"raw_payload_sha256_hex"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		occurredAt, err := parseCHTime(row.OccurredAt)
		if err != nil {
			return nil, err
		}
		receivedAt, err := parseCHTime(row.CloudReceivedAt)
		if err != nil {
			return nil, err
		}
		rows = append(rows, app.RawBusinessEvent{
			EventID:             row.EventID,
			TenantID:            row.TenantID,
			RestaurantID:        row.RestaurantID,
			DeviceID:            row.DeviceID,
			EmployeeID:          row.EmployeeID,
			EventType:           row.EventType,
			OccurredAt:          occurredAt,
			CloudReceivedAt:     receivedAt,
			RawPayloadSHA256Hex: row.RawPayloadSHA256Hex,
		})
	}
	return rows, scanner.Err()
}

func (r *Repository) ListStockMoves(ctx context.Context, filter app.StockMoveFilter) ([]app.StockMove, error) {
	query := strings.Builder{}
	query.WriteString("SELECT ledger_entry_id,restaurant_id,warehouse_id,stock_document_id,source_event_id,source_event_type,catalog_item_id,order_line_id,movement_type,quantity,unit_code,unit_cost_minor,total_cost_minor,costing_status,occurred_at,toString(business_date_local) AS business_date_local,ledger_created_at FROM ")
	query.WriteString(ident(r.database))
	query.WriteString(".olap_stock_moves FINAL WHERE 1=1")
	if filter.RestaurantID != "" {
		query.WriteString(" AND restaurant_id = ")
		query.WriteString(quote(filter.RestaurantID))
	}
	if filter.BusinessDateFrom != "" {
		query.WriteString(" AND business_date_local >= toDate(")
		query.WriteString(quote(filter.BusinessDateFrom))
		query.WriteString(")")
	}
	if filter.BusinessDateTo != "" {
		query.WriteString(" AND business_date_local <= toDate(")
		query.WriteString(quote(filter.BusinessDateTo))
		query.WriteString(")")
	}
	if filter.CatalogItemID != "" {
		query.WriteString(" AND catalog_item_id = ")
		query.WriteString(quote(filter.CatalogItemID))
	}
	if filter.WarehouseID != "" {
		query.WriteString(" AND warehouse_id = ")
		query.WriteString(quote(filter.WarehouseID))
	}
	if filter.SourceEventType != "" {
		query.WriteString(" AND source_event_type = ")
		query.WriteString(quote(filter.SourceEventType))
	}
	query.WriteString(fmt.Sprintf(" ORDER BY business_date_local DESC, occurred_at DESC, ledger_entry_id DESC LIMIT %d OFFSET %d FORMAT JSONEachRow", filter.Limit, filter.Offset))

	respBody, err := r.query(ctx, query.String())
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	rows := make([]app.StockMove, 0, filter.Limit)
	scanner := bufio.NewScanner(respBody)
	for scanner.Scan() {
		var row struct {
			LedgerEntryID     string  `json:"ledger_entry_id"`
			RestaurantID      string  `json:"restaurant_id"`
			WarehouseID       string  `json:"warehouse_id"`
			StockDocumentID   string  `json:"stock_document_id"`
			SourceEventID     string  `json:"source_event_id"`
			SourceEventType   string  `json:"source_event_type"`
			CatalogItemID     string  `json:"catalog_item_id"`
			OrderLineID       string  `json:"order_line_id"`
			MovementType      string  `json:"movement_type"`
			Quantity          string  `json:"quantity"`
			UnitCode          string  `json:"unit_code"`
			UnitCostMinor     chInt64 `json:"unit_cost_minor"`
			TotalCostMinor    chInt64 `json:"total_cost_minor"`
			CostingStatus     string  `json:"costing_status"`
			OccurredAt        string  `json:"occurred_at"`
			BusinessDateLocal string  `json:"business_date_local"`
			LedgerCreatedAt   string  `json:"ledger_created_at"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		occurredAt, err := parseCHTime(row.OccurredAt)
		if err != nil {
			return nil, err
		}
		createdAt, err := parseCHTime(row.LedgerCreatedAt)
		if err != nil {
			return nil, err
		}
		rows = append(rows, app.StockMove{
			LedgerEntryID:     row.LedgerEntryID,
			RestaurantID:      row.RestaurantID,
			WarehouseID:       row.WarehouseID,
			StockDocumentID:   row.StockDocumentID,
			SourceEventID:     row.SourceEventID,
			SourceEventType:   row.SourceEventType,
			CatalogItemID:     row.CatalogItemID,
			OrderLineID:       row.OrderLineID,
			MovementType:      row.MovementType,
			Quantity:          row.Quantity,
			UnitCode:          row.UnitCode,
			UnitCostMinor:     int64(row.UnitCostMinor),
			TotalCostMinor:    int64(row.TotalCostMinor),
			CostingStatus:     row.CostingStatus,
			OccurredAt:        occurredAt,
			BusinessDateLocal: row.BusinessDateLocal,
			LedgerCreatedAt:   createdAt,
		})
	}
	return rows, scanner.Err()
}

func (r *Repository) ListStockMoveSummary(ctx context.Context, filter app.StockMoveSummaryFilter) ([]app.StockMoveSummary, error) {
	groupKeyExpr := "toString(business_date_local)"
	businessDateExpr := "toString(business_date_local)"
	catalogItemExpr := "''"
	warehouseExpr := "''"
	groupByExpr := "business_date_local"
	orderExpr := "business_date_local DESC, group_key DESC"
	switch filter.GroupBy {
	case "catalog_item":
		groupKeyExpr = "catalog_item_id"
		businessDateExpr = "''"
		catalogItemExpr = "catalog_item_id"
		groupByExpr = "catalog_item_id"
		orderExpr = "catalog_item_id ASC"
	case "warehouse":
		groupKeyExpr = "warehouse_id"
		businessDateExpr = "''"
		warehouseExpr = "warehouse_id"
		groupByExpr = "warehouse_id"
		orderExpr = "warehouse_id ASC"
	}

	query := strings.Builder{}
	query.WriteString("SELECT ")
	query.WriteString(quote(filter.GroupBy))
	query.WriteString(" AS group_by, ")
	query.WriteString(groupKeyExpr)
	query.WriteString(" AS group_key, ")
	query.WriteString(businessDateExpr)
	query.WriteString(" AS business_date_local, ")
	query.WriteString(catalogItemExpr)
	query.WriteString(" AS catalog_item_id, ")
	query.WriteString(warehouseExpr)
	query.WriteString(" AS warehouse_id, count() AS move_count, ")
	query.WriteString("toString(sumIf(toDecimal64OrZero(quantity, 3), movement_type = 'IN')) AS in_quantity, ")
	query.WriteString("toString(sumIf(toDecimal64OrZero(quantity, 3), movement_type = 'OUT')) AS out_quantity, ")
	query.WriteString("toString(sumIf(toDecimal64OrZero(quantity, 3), movement_type = 'IN') - sumIf(toDecimal64OrZero(quantity, 3), movement_type = 'OUT')) AS net_quantity, ")
	query.WriteString("sum(total_cost_minor) AS total_cost_minor, min(occurred_at) AS first_occurred_at, max(occurred_at) AS last_occurred_at FROM ")
	query.WriteString(ident(r.database))
	query.WriteString(".olap_stock_moves FINAL WHERE 1=1")
	if filter.RestaurantID != "" {
		query.WriteString(" AND restaurant_id = ")
		query.WriteString(quote(filter.RestaurantID))
	}
	if filter.BusinessDateFrom != "" {
		query.WriteString(" AND business_date_local >= toDate(")
		query.WriteString(quote(filter.BusinessDateFrom))
		query.WriteString(")")
	}
	if filter.BusinessDateTo != "" {
		query.WriteString(" AND business_date_local <= toDate(")
		query.WriteString(quote(filter.BusinessDateTo))
		query.WriteString(")")
	}
	if filter.CatalogItemID != "" {
		query.WriteString(" AND catalog_item_id = ")
		query.WriteString(quote(filter.CatalogItemID))
	}
	if filter.WarehouseID != "" {
		query.WriteString(" AND warehouse_id = ")
		query.WriteString(quote(filter.WarehouseID))
	}
	if filter.SourceEventType != "" {
		query.WriteString(" AND source_event_type = ")
		query.WriteString(quote(filter.SourceEventType))
	}
	query.WriteString(" GROUP BY ")
	query.WriteString(groupByExpr)
	query.WriteString(" ORDER BY ")
	query.WriteString(orderExpr)
	query.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d FORMAT JSONEachRow", filter.Limit, filter.Offset))

	respBody, err := r.query(ctx, query.String())
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	rows := make([]app.StockMoveSummary, 0, filter.Limit)
	scanner := bufio.NewScanner(respBody)
	for scanner.Scan() {
		var row struct {
			GroupBy           string  `json:"group_by"`
			GroupKey          string  `json:"group_key"`
			BusinessDateLocal string  `json:"business_date_local"`
			CatalogItemID     string  `json:"catalog_item_id"`
			WarehouseID       string  `json:"warehouse_id"`
			MoveCount         chInt64 `json:"move_count"`
			InQuantity        string  `json:"in_quantity"`
			OutQuantity       string  `json:"out_quantity"`
			NetQuantity       string  `json:"net_quantity"`
			TotalCostMinor    chInt64 `json:"total_cost_minor"`
			FirstOccurredAt   string  `json:"first_occurred_at"`
			LastOccurredAt    string  `json:"last_occurred_at"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		firstOccurredAt, err := parseCHTime(row.FirstOccurredAt)
		if err != nil {
			return nil, err
		}
		lastOccurredAt, err := parseCHTime(row.LastOccurredAt)
		if err != nil {
			return nil, err
		}
		rows = append(rows, app.StockMoveSummary{
			GroupBy:           row.GroupBy,
			GroupKey:          row.GroupKey,
			BusinessDateLocal: row.BusinessDateLocal,
			CatalogItemID:     row.CatalogItemID,
			WarehouseID:       row.WarehouseID,
			MoveCount:         int64(row.MoveCount),
			InQuantity:        row.InQuantity,
			OutQuantity:       row.OutQuantity,
			NetQuantity:       row.NetQuantity,
			TotalCostMinor:    int64(row.TotalCostMinor),
			FirstOccurredAt:   &firstOccurredAt,
			LastOccurredAt:    &lastOccurredAt,
		})
	}
	return rows, scanner.Err()
}

func (r *Repository) ListSalesKitchenSummary(ctx context.Context, filter app.SalesKitchenSummaryFilter) ([]app.SalesKitchenSummary, error) {
	parts := make([]string, 0, 2)
	if filter.GroupBy != "catalog_item" {
		parts = append(parts, r.salesKitchenRawSummaryQuery(filter))
	}
	parts = append(parts, r.salesKitchenStockSummaryQuery(filter))

	orderExpr := "group_key ASC"
	if filter.GroupBy == "business_date" {
		orderExpr = "group_key DESC"
	}

	query := strings.Builder{}
	query.WriteString("SELECT group_by, group_key, max(business_date_local) AS business_date_local, ")
	query.WriteString("max(event_type) AS event_type, max(source_event_type) AS source_event_type, max(catalog_item_id) AS catalog_item_id, ")
	query.WriteString("sum(event_count) AS event_count, sum(stock_move_count) AS stock_move_count, ")
	query.WriteString("sum(sale_event_count) AS sale_event_count, sum(kitchen_event_count) AS kitchen_event_count, ")
	query.WriteString("toString(sum(stock_in_quantity)) AS in_quantity, toString(sum(stock_out_quantity)) AS out_quantity, ")
	query.WriteString("toString(sum(stock_in_quantity) - sum(stock_out_quantity)) AS net_quantity, sum(total_cost_minor) AS total_cost_minor, ")
	query.WriteString("min(first_occurred_at) AS first_occurred_at, max(last_occurred_at) AS last_occurred_at FROM (")
	query.WriteString(strings.Join(parts, " UNION ALL "))
	query.WriteString(") GROUP BY group_by, group_key ORDER BY ")
	query.WriteString(orderExpr)
	query.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d FORMAT JSONEachRow", filter.Limit, filter.Offset))

	respBody, err := r.query(ctx, query.String())
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	rows := make([]app.SalesKitchenSummary, 0, filter.Limit)
	scanner := bufio.NewScanner(respBody)
	for scanner.Scan() {
		var row struct {
			GroupBy           string  `json:"group_by"`
			GroupKey          string  `json:"group_key"`
			BusinessDateLocal string  `json:"business_date_local"`
			EventType         string  `json:"event_type"`
			SourceEventType   string  `json:"source_event_type"`
			CatalogItemID     string  `json:"catalog_item_id"`
			EventCount        chInt64 `json:"event_count"`
			StockMoveCount    chInt64 `json:"stock_move_count"`
			SaleEventCount    chInt64 `json:"sale_event_count"`
			KitchenEventCount chInt64 `json:"kitchen_event_count"`
			OutQuantity       string  `json:"out_quantity"`
			InQuantity        string  `json:"in_quantity"`
			NetQuantity       string  `json:"net_quantity"`
			TotalCostMinor    chInt64 `json:"total_cost_minor"`
			FirstOccurredAt   string  `json:"first_occurred_at"`
			LastOccurredAt    string  `json:"last_occurred_at"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		firstOccurredAt, err := parseOptionalCHTime(row.FirstOccurredAt)
		if err != nil {
			return nil, err
		}
		lastOccurredAt, err := parseOptionalCHTime(row.LastOccurredAt)
		if err != nil {
			return nil, err
		}
		rows = append(rows, app.SalesKitchenSummary{
			GroupBy:           row.GroupBy,
			GroupKey:          row.GroupKey,
			BusinessDateLocal: row.BusinessDateLocal,
			EventType:         row.EventType,
			SourceEventType:   row.SourceEventType,
			CatalogItemID:     row.CatalogItemID,
			EventCount:        int64(row.EventCount),
			StockMoveCount:    int64(row.StockMoveCount),
			SaleEventCount:    int64(row.SaleEventCount),
			KitchenEventCount: int64(row.KitchenEventCount),
			OutQuantity:       row.OutQuantity,
			InQuantity:        row.InQuantity,
			NetQuantity:       row.NetQuantity,
			TotalCostMinor:    int64(row.TotalCostMinor),
			FirstOccurredAt:   firstOccurredAt,
			LastOccurredAt:    lastOccurredAt,
		})
	}
	return rows, scanner.Err()
}

func (r *Repository) ListKitchenTimingSummary(ctx context.Context, filter app.KitchenTimingSummaryFilter) ([]app.KitchenTimingSummary, error) {
	groupKeyExpr := "business_date_local"
	businessDateExpr := "business_date_local"
	stationExpr := "''"
	groupByExpr := "business_date_local"
	orderExpr := "business_date_local DESC, group_key DESC"
	if filter.GroupBy == "station" {
		groupKeyExpr = "station_id"
		businessDateExpr = "''"
		stationExpr = "station_id"
		groupByExpr = "station_id"
		orderExpr = "station_id ASC"
	}

	query := strings.Builder{}
	query.WriteString("SELECT ")
	query.WriteString(quote(filter.GroupBy))
	query.WriteString(" AS group_by, ")
	query.WriteString(groupKeyExpr)
	query.WriteString(" AS group_key, ")
	query.WriteString(businessDateExpr)
	query.WriteString(" AS business_date_local, ")
	query.WriteString(stationExpr)
	query.WriteString(" AS station_id, count() AS ticket_count, ")
	query.WriteString("countIf(accepted_at IS NOT NULL) AS accepted_count, countIf(started_at IS NOT NULL) AS started_count, ")
	query.WriteString("countIf(ready_at IS NOT NULL) AS ready_count, countIf(served_at IS NOT NULL) AS served_count, ")
	query.WriteString("if(isNaN(avgIf(dateDiff('second', accepted_at, ready_at), accepted_at IS NOT NULL AND ready_at IS NOT NULL AND ready_at >= accepted_at)), 0, toInt64(avgIf(dateDiff('second', accepted_at, ready_at), accepted_at IS NOT NULL AND ready_at IS NOT NULL AND ready_at >= accepted_at))) AS avg_accept_to_ready_seconds, ")
	query.WriteString("if(isNaN(avgIf(dateDiff('second', started_at, ready_at), started_at IS NOT NULL AND ready_at IS NOT NULL AND ready_at >= started_at)), 0, toInt64(avgIf(dateDiff('second', started_at, ready_at), started_at IS NOT NULL AND ready_at IS NOT NULL AND ready_at >= started_at))) AS avg_start_to_ready_seconds, ")
	query.WriteString("if(isNaN(avgIf(dateDiff('second', ready_at, served_at), ready_at IS NOT NULL AND served_at IS NOT NULL AND served_at >= ready_at)), 0, toInt64(avgIf(dateDiff('second', ready_at, served_at), ready_at IS NOT NULL AND served_at IS NOT NULL AND served_at >= ready_at))) AS avg_ready_to_served_seconds, ")
	query.WriteString("min(first_status_changed_at) AS first_status_changed_at, max(last_status_changed_at) AS last_status_changed_at FROM (")
	query.WriteString(r.kitchenTimingPerTicketQuery(filter))
	query.WriteString(") GROUP BY ")
	query.WriteString(groupByExpr)
	query.WriteString(" ORDER BY ")
	query.WriteString(orderExpr)
	query.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d FORMAT JSONEachRow", filter.Limit, filter.Offset))

	respBody, err := r.query(ctx, query.String())
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	rows := make([]app.KitchenTimingSummary, 0, filter.Limit)
	scanner := bufio.NewScanner(respBody)
	for scanner.Scan() {
		var row struct {
			GroupBy                 string  `json:"group_by"`
			GroupKey                string  `json:"group_key"`
			BusinessDateLocal       string  `json:"business_date_local"`
			StationID               string  `json:"station_id"`
			TicketCount             chInt64 `json:"ticket_count"`
			AcceptedCount           chInt64 `json:"accepted_count"`
			StartedCount            chInt64 `json:"started_count"`
			ReadyCount              chInt64 `json:"ready_count"`
			ServedCount             chInt64 `json:"served_count"`
			AvgAcceptToReadySeconds chInt64 `json:"avg_accept_to_ready_seconds"`
			AvgStartToReadySeconds  chInt64 `json:"avg_start_to_ready_seconds"`
			AvgReadyToServedSeconds chInt64 `json:"avg_ready_to_served_seconds"`
			FirstStatusChangedAt    string  `json:"first_status_changed_at"`
			LastStatusChangedAt     string  `json:"last_status_changed_at"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, err
		}
		firstStatusChangedAt, err := parseOptionalCHTime(row.FirstStatusChangedAt)
		if err != nil {
			return nil, err
		}
		lastStatusChangedAt, err := parseOptionalCHTime(row.LastStatusChangedAt)
		if err != nil {
			return nil, err
		}
		rows = append(rows, app.KitchenTimingSummary{
			GroupBy:                 row.GroupBy,
			GroupKey:                row.GroupKey,
			BusinessDateLocal:       row.BusinessDateLocal,
			StationID:               row.StationID,
			TicketCount:             int64(row.TicketCount),
			AcceptedCount:           int64(row.AcceptedCount),
			StartedCount:            int64(row.StartedCount),
			ReadyCount:              int64(row.ReadyCount),
			ServedCount:             int64(row.ServedCount),
			AvgAcceptToReadySeconds: int64(row.AvgAcceptToReadySeconds),
			AvgStartToReadySeconds:  int64(row.AvgStartToReadySeconds),
			AvgReadyToServedSeconds: int64(row.AvgReadyToServedSeconds),
			FirstStatusChangedAt:    firstStatusChangedAt,
			LastStatusChangedAt:     lastStatusChangedAt,
		})
	}
	return rows, scanner.Err()
}

func (r *Repository) kitchenTimingPerTicketQuery(filter app.KitchenTimingSummaryFilter) string {
	query := strings.Builder{}
	query.WriteString("SELECT toString(toDate(occurred_at)) AS business_date_local, ")
	query.WriteString("if(station_id = '', 'unknown', station_id) AS station_id, order_line_id, ")
	query.WriteString("nullIf(minIf(occurred_at, to_status = 'accepted'), toDateTime64(0, 3, 'UTC')) AS accepted_at, ")
	query.WriteString("nullIf(minIf(occurred_at, to_status = 'in_progress'), toDateTime64(0, 3, 'UTC')) AS started_at, ")
	query.WriteString("nullIf(minIf(occurred_at, to_status = 'ready'), toDateTime64(0, 3, 'UTC')) AS ready_at, ")
	query.WriteString("nullIf(minIf(occurred_at, to_status = 'served' OR event_type = 'ItemServed'), toDateTime64(0, 3, 'UTC')) AS served_at, ")
	query.WriteString("min(occurred_at) AS first_status_changed_at, max(occurred_at) AS last_status_changed_at FROM (")
	query.WriteString("SELECT event_type, occurred_at, ")
	query.WriteString("JSONExtractString(payload, 'payload', 'data', 'order_line_id') AS order_line_id, ")
	query.WriteString("JSONExtractString(payload, 'payload', 'data', 'station_id') AS station_id, ")
	query.WriteString("JSONExtractString(payload, 'payload', 'data', 'to_status') AS to_status FROM ")
	query.WriteString(ident(r.database))
	query.WriteString(".raw_business_events FINAL WHERE event_type IN ('KitchenTicketStatusChanged','ItemServed')")
	if filter.RestaurantID != "" {
		query.WriteString(" AND restaurant_id = ")
		query.WriteString(quote(filter.RestaurantID))
	}
	if filter.BusinessDateFrom != "" {
		query.WriteString(" AND toDate(occurred_at) >= toDate(")
		query.WriteString(quote(filter.BusinessDateFrom))
		query.WriteString(")")
	}
	if filter.BusinessDateTo != "" {
		query.WriteString(" AND toDate(occurred_at) <= toDate(")
		query.WriteString(quote(filter.BusinessDateTo))
		query.WriteString(")")
	}
	query.WriteString(") WHERE order_line_id != ''")
	if filter.StationID != "" {
		query.WriteString(" AND station_id = ")
		query.WriteString(quote(filter.StationID))
	}
	query.WriteString(" GROUP BY business_date_local, station_id, order_line_id")
	return query.String()
}

func (r *Repository) salesKitchenRawSummaryQuery(filter app.SalesKitchenSummaryFilter) string {
	groupKeyExpr, businessDateExpr, eventTypeExpr, sourceEventTypeExpr, groupByExpr := salesKitchenRawGroupExpressions(filter.GroupBy)
	query := strings.Builder{}
	query.WriteString("SELECT ")
	query.WriteString(quote(filter.GroupBy))
	query.WriteString(" AS group_by, ")
	query.WriteString(groupKeyExpr)
	query.WriteString(" AS group_key, ")
	query.WriteString(businessDateExpr)
	query.WriteString(" AS business_date_local, ")
	query.WriteString(eventTypeExpr)
	query.WriteString(" AS event_type, ")
	query.WriteString(sourceEventTypeExpr)
	query.WriteString(" AS source_event_type, '' AS catalog_item_id, ")
	query.WriteString("toInt64(count()) AS event_count, toInt64(0) AS stock_move_count, ")
	query.WriteString("toInt64(countIf(event_type IN ")
	query.WriteString(salesKitchenSaleEventTypesSQL)
	query.WriteString(")) AS sale_event_count, toInt64(countIf(event_type IN ")
	query.WriteString(salesKitchenKitchenEventTypesSQL)
	query.WriteString(")) AS kitchen_event_count, ")
	query.WriteString("toDecimal64(0, 3) AS stock_in_quantity, toDecimal64(0, 3) AS stock_out_quantity, toInt64(0) AS total_cost_minor, ")
	query.WriteString("min(occurred_at) AS first_occurred_at, max(occurred_at) AS last_occurred_at FROM ")
	query.WriteString(ident(r.database))
	query.WriteString(".raw_business_events FINAL WHERE 1=1")
	appendSalesKitchenRawFilters(&query, filter)
	query.WriteString(" GROUP BY ")
	query.WriteString(groupByExpr)
	return query.String()
}

func (r *Repository) salesKitchenStockSummaryQuery(filter app.SalesKitchenSummaryFilter) string {
	groupKeyExpr, businessDateExpr, eventTypeExpr, sourceEventTypeExpr, catalogItemExpr, groupByExpr := salesKitchenStockGroupExpressions(filter.GroupBy)
	query := strings.Builder{}
	query.WriteString("SELECT ")
	query.WriteString(quote(filter.GroupBy))
	query.WriteString(" AS group_by, ")
	query.WriteString(groupKeyExpr)
	query.WriteString(" AS group_key, ")
	query.WriteString(businessDateExpr)
	query.WriteString(" AS business_date_local, ")
	query.WriteString(eventTypeExpr)
	query.WriteString(" AS event_type, ")
	query.WriteString(sourceEventTypeExpr)
	query.WriteString(" AS source_event_type, ")
	query.WriteString(catalogItemExpr)
	query.WriteString(" AS catalog_item_id, ")
	query.WriteString("toInt64(0) AS event_count, toInt64(count()) AS stock_move_count, toInt64(0) AS sale_event_count, toInt64(0) AS kitchen_event_count, ")
	query.WriteString("sumIf(toDecimal64OrZero(quantity, 3), movement_type = 'IN') AS stock_in_quantity, ")
	query.WriteString("sumIf(toDecimal64OrZero(quantity, 3), movement_type = 'OUT') AS stock_out_quantity, ")
	query.WriteString("sum(total_cost_minor) AS total_cost_minor, min(occurred_at) AS first_occurred_at, max(occurred_at) AS last_occurred_at FROM ")
	query.WriteString(ident(r.database))
	query.WriteString(".olap_stock_moves FINAL WHERE 1=1")
	appendSalesKitchenStockFilters(&query, filter)
	query.WriteString(" GROUP BY ")
	query.WriteString(groupByExpr)
	return query.String()
}

func salesKitchenRawGroupExpressions(groupBy string) (groupKeyExpr, businessDateExpr, eventTypeExpr, sourceEventTypeExpr, groupByExpr string) {
	switch groupBy {
	case "event_type":
		return "event_type", "''", "event_type", "''", "event_type"
	case "source_event_type":
		return "event_type", "''", "''", "event_type", "event_type"
	default:
		return "toString(toDate(occurred_at))", "toString(toDate(occurred_at))", "''", "''", "toDate(occurred_at)"
	}
}

func salesKitchenStockGroupExpressions(groupBy string) (groupKeyExpr, businessDateExpr, eventTypeExpr, sourceEventTypeExpr, catalogItemExpr, groupByExpr string) {
	switch groupBy {
	case "event_type":
		return "source_event_type", "''", "source_event_type", "''", "''", "source_event_type"
	case "source_event_type":
		return "source_event_type", "''", "''", "source_event_type", "''", "source_event_type"
	case "catalog_item":
		return "catalog_item_id", "''", "''", "''", "catalog_item_id", "catalog_item_id"
	default:
		return "toString(business_date_local)", "toString(business_date_local)", "''", "''", "''", "business_date_local"
	}
}

func appendSalesKitchenRawFilters(query *strings.Builder, filter app.SalesKitchenSummaryFilter) {
	if filter.RestaurantID != "" {
		query.WriteString(" AND restaurant_id = ")
		query.WriteString(quote(filter.RestaurantID))
	}
	if filter.BusinessDateFrom != "" {
		query.WriteString(" AND toDate(occurred_at) >= toDate(")
		query.WriteString(quote(filter.BusinessDateFrom))
		query.WriteString(")")
	}
	if filter.BusinessDateTo != "" {
		query.WriteString(" AND toDate(occurred_at) <= toDate(")
		query.WriteString(quote(filter.BusinessDateTo))
		query.WriteString(")")
	}
}

func appendSalesKitchenStockFilters(query *strings.Builder, filter app.SalesKitchenSummaryFilter) {
	if filter.RestaurantID != "" {
		query.WriteString(" AND restaurant_id = ")
		query.WriteString(quote(filter.RestaurantID))
	}
	if filter.BusinessDateFrom != "" {
		query.WriteString(" AND business_date_local >= toDate(")
		query.WriteString(quote(filter.BusinessDateFrom))
		query.WriteString(")")
	}
	if filter.BusinessDateTo != "" {
		query.WriteString(" AND business_date_local <= toDate(")
		query.WriteString(quote(filter.BusinessDateTo))
		query.WriteString(")")
	}
}

func (r *Repository) exec(ctx context.Context, query string) error {
	return r.post(ctx, query)
}

func (r *Repository) query(ctx context.Context, query string) (io.ReadCloser, error) {
	return r.do(ctx, query)
}

func (r *Repository) post(ctx context.Context, query string) error {
	body, err := r.do(ctx, query)
	if err != nil {
		return err
	}
	defer body.Close()
	_, _ = io.Copy(io.Discard, body)
	return nil
}

func (r *Repository) do(ctx context.Context, query string) (io.ReadCloser, error) {
	if r == nil || r.endpoint == "" {
		return nil, app.ErrOLAPUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, strings.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	if r.username != "" {
		req.SetBasicAuth(r.username, r.password)
	}
	q := req.URL.Query()
	q.Set("database", r.database)
	req.URL.RawQuery = q.Encode()
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("clickhouse http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return resp.Body, nil
}

func ident(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "default"
	}
	return b.String()
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `'`, `\'`) + "'"
}

func chTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05.000")
}

func parseCHTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
	} {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid ClickHouse timestamp %q", value)
}

func parseOptionalCHTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := parseCHTime(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

type chInt64 int64

func (v *chInt64) UnmarshalJSON(raw []byte) error {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		*v = 0
		return nil
	}
	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return err
		}
		value = strings.TrimSpace(unquoted)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	*v = chInt64(parsed)
	return nil
}
