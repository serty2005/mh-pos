package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud-backend/internal/cloudsync/contracts"
)

// ErrOLAPUnavailable означает, что ClickHouse runtime не сконфигурирован или недоступен для текущего Cloud instance.
var ErrOLAPUnavailable = errors.New("olap runtime unavailable")

// RawBusinessEvent описывает безопасную metadata-проекцию ClickHouse raw event без raw payload.
type RawBusinessEvent struct {
	EventID             string    `json:"event_id"`
	TenantID            string    `json:"tenant_id"`
	RestaurantID        string    `json:"restaurant_id"`
	DeviceID            string    `json:"device_id"`
	EmployeeID          string    `json:"employee_id,omitempty"`
	EventType           string    `json:"event_type"`
	OccurredAt          time.Time `json:"occurred_at"`
	CloudReceivedAt     time.Time `json:"cloud_received_at"`
	RawPayloadSHA256Hex string    `json:"raw_payload_sha256_hex"`
}

// RawBusinessEventFilter задает bounded read-only выборку из ClickHouse event archive.
type RawBusinessEventFilter struct {
	RestaurantID string
	EventType    string
	OccurredFrom *time.Time
	OccurredTo   *time.Time
	Limit        int
	Offset       int
}

// StockMove описывает безопасную ClickHouse projection строку складского движения без raw sync payload.
type StockMove struct {
	LedgerEntryID     string    `json:"ledger_entry_id"`
	RestaurantID      string    `json:"restaurant_id"`
	WarehouseID       string    `json:"warehouse_id,omitempty"`
	StockDocumentID   string    `json:"stock_document_id"`
	SourceEventID     string    `json:"source_event_id"`
	SourceEventType   string    `json:"source_event_type"`
	CatalogItemID     string    `json:"catalog_item_id"`
	OrderLineID       string    `json:"order_line_id,omitempty"`
	MovementType      string    `json:"movement_type"`
	Quantity          string    `json:"quantity"`
	UnitCode          string    `json:"unit_code"`
	UnitCostMinor     int64     `json:"unit_cost_minor"`
	TotalCostMinor    int64     `json:"total_cost_minor"`
	CostingStatus     string    `json:"costing_status"`
	OccurredAt        time.Time `json:"occurred_at"`
	BusinessDateLocal string    `json:"business_date_local"`
	LedgerCreatedAt   time.Time `json:"ledger_created_at"`
}

// StockMoveSummary описывает агрегированное чтение складских движений из ClickHouse без raw payload.
type StockMoveSummary struct {
	GroupBy           string     `json:"group_by"`
	GroupKey          string     `json:"group_key"`
	BusinessDateLocal string     `json:"business_date_local,omitempty"`
	CatalogItemID     string     `json:"catalog_item_id,omitempty"`
	WarehouseID       string     `json:"warehouse_id,omitempty"`
	MoveCount         int64      `json:"move_count"`
	InQuantity        string     `json:"in_quantity"`
	OutQuantity       string     `json:"out_quantity"`
	NetQuantity       string     `json:"net_quantity"`
	TotalCostMinor    int64      `json:"total_cost_minor"`
	FirstOccurredAt   *time.Time `json:"first_occurred_at,omitempty"`
	LastOccurredAt    *time.Time `json:"last_occurred_at,omitempty"`
}

// StockMoveFilter задает bounded read-only выборку из ClickHouse stock moves projection.
type StockMoveFilter struct {
	RestaurantID     string
	BusinessDateFrom string
	BusinessDateTo   string
	CatalogItemID    string
	WarehouseID      string
	SourceEventType  string
	Limit            int
	Offset           int
}

// StockMoveSummaryFilter задает bounded агрегированное чтение из ClickHouse stock moves projection.
type StockMoveSummaryFilter struct {
	RestaurantID     string
	BusinessDateFrom string
	BusinessDateTo   string
	CatalogItemID    string
	WarehouseID      string
	SourceEventType  string
	GroupBy          string
	Limit            int
	Offset           int
}

// ExportStatus описывает безопасное состояние async OLAP export без raw payload.
type ExportStatus struct {
	Stream              string     `json:"stream"`
	LastCheckpoint      string     `json:"last_checkpoint,omitempty"`
	LastExportedID      string     `json:"last_exported_id,omitempty"`
	LastExportedAt      *time.Time `json:"last_exported_at,omitempty"`
	PendingCount        int64      `json:"pending_count"`
	ProcessingCount     int64      `json:"processing_count"`
	FailedCount         int64      `json:"failed_count"`
	LastError           string     `json:"last_error,omitempty"`
	ConsecutiveFailures int64      `json:"consecutive_failures"`
	NextRetryAt         *time.Time `json:"next_retry_at,omitempty"`
	RetryBlocked        bool       `json:"retry_blocked"`
	CheckpointUpdatedAt *time.Time `json:"checkpoint_updated_at,omitempty"`
}

// ExportRetryCommand описывает support-only команду снятия OLAP retry/backoff state без записи business rows.
type ExportRetryCommand struct {
	CommandID string `json:"command_id"`
	Stream    string `json:"stream"`
	Mode      string `json:"mode"`
	Reason    string `json:"reason"`
}

// ExportRetryResult возвращает безопасный результат control-команды без raw payload.
type ExportRetryResult struct {
	CommandID          string    `json:"command_id"`
	Stream             string    `json:"stream"`
	Mode               string    `json:"mode"`
	Accepted           bool      `json:"accepted"`
	CheckpointBefore   string    `json:"checkpoint_before,omitempty"`
	RetryRequestedAt   time.Time `json:"retry_requested_at"`
	PendingCount       int64     `json:"pending_count"`
	FailedCount        int64     `json:"failed_count"`
	AlreadyProcessed   bool      `json:"already_processed,omitempty"`
}

// RawBusinessEventRepository читает ClickHouse event archive без участия transactional command path.
type RawBusinessEventRepository interface {
	ListRawBusinessEvents(context.Context, RawBusinessEventFilter) ([]RawBusinessEvent, error)
}

// StockMoveRepository читает ClickHouse projection складских движений без raw payload.
type StockMoveRepository interface {
	ListStockMoves(context.Context, StockMoveFilter) ([]StockMove, error)
}

// StockMoveSummaryRepository читает агрегированные складские показатели из ClickHouse read model.
type StockMoveSummaryRepository interface {
	ListStockMoveSummary(context.Context, StockMoveSummaryFilter) ([]StockMoveSummary, error)
}

// ExportStatusRepository читает PostgreSQL checkpoint/retry состояние OLAP export.
type ExportStatusRepository interface {
	GetExportStatus(context.Context, string, time.Time) (ExportStatus, error)
}

// ExportRetryRepository применяет support-only retry/backfill control state в PostgreSQL.
type ExportRetryRepository interface {
	RequestExportRetry(context.Context, ExportRetryCommand, time.Time) (ExportRetryResult, error)
}

type Repository interface {
	RawBusinessEventRepository
	StockMoveRepository
	StockMoveSummaryRepository
}

// Service валидирует OLAP read API и делегирует bounded чтение ClickHouse repository.
type Service struct {
	repo             Repository
	exportStatusRepo ExportStatusRepository
	exportRetryRepo  ExportRetryRepository
}

// NewService создает read-only OLAP service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// NewServiceWithExportStatus создает OLAP service с read-only observability по PostgreSQL checkpoint state.
func NewServiceWithExportStatus(repo Repository, exportStatusRepo ExportStatusRepository) *Service {
	return &Service{repo: repo, exportStatusRepo: exportStatusRepo}
}

// NewServiceWithControls создает OLAP service с read-only observability и support-only retry controls.
func NewServiceWithControls(repo Repository, exportStatusRepo ExportStatusRepository, exportRetryRepo ExportRetryRepository) *Service {
	return &Service{repo: repo, exportStatusRepo: exportStatusRepo, exportRetryRepo: exportRetryRepo}
}

// ListRawBusinessEvents возвращает bounded metadata view без раскрытия raw payload.
func (s *Service) ListRawBusinessEvents(ctx context.Context, filter RawBusinessEventFilter) ([]RawBusinessEvent, error) {
	if s == nil || s.repo == nil {
		return nil, ErrOLAPUnavailable
	}
	filter.RestaurantID = strings.TrimSpace(filter.RestaurantID)
	filter.EventType = strings.TrimSpace(filter.EventType)
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		return nil, fmt.Errorf("%w: offset must be non-negative", contracts.ErrInvalidEnvelope)
	}
	if filter.OccurredFrom != nil && filter.OccurredTo != nil && filter.OccurredFrom.After(*filter.OccurredTo) {
		return nil, fmt.Errorf("%w: occurred_from must be before occurred_to", contracts.ErrInvalidEnvelope)
	}
	return s.repo.ListRawBusinessEvents(ctx, filter)
}

// ListStockMoves возвращает bounded stock movement projection без раскрытия raw sync payload.
func (s *Service) ListStockMoves(ctx context.Context, filter StockMoveFilter) ([]StockMove, error) {
	if s == nil || s.repo == nil {
		return nil, ErrOLAPUnavailable
	}
	filter.RestaurantID = strings.TrimSpace(filter.RestaurantID)
	filter.BusinessDateFrom = strings.TrimSpace(filter.BusinessDateFrom)
	filter.BusinessDateTo = strings.TrimSpace(filter.BusinessDateTo)
	filter.CatalogItemID = strings.TrimSpace(filter.CatalogItemID)
	filter.WarehouseID = strings.TrimSpace(filter.WarehouseID)
	filter.SourceEventType = strings.TrimSpace(filter.SourceEventType)
	if err := validateBusinessDate(filter.BusinessDateFrom, "business_date_from"); err != nil {
		return nil, err
	}
	if err := validateBusinessDate(filter.BusinessDateTo, "business_date_to"); err != nil {
		return nil, err
	}
	if filter.BusinessDateFrom != "" && filter.BusinessDateTo != "" && filter.BusinessDateFrom > filter.BusinessDateTo {
		return nil, fmt.Errorf("%w: business_date_from must be before business_date_to", contracts.ErrInvalidEnvelope)
	}
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		return nil, fmt.Errorf("%w: offset must be non-negative", contracts.ErrInvalidEnvelope)
	}
	return s.repo.ListStockMoves(ctx, filter)
}

// ListStockMoveSummary возвращает bounded агрегат складских движений из ClickHouse.
func (s *Service) ListStockMoveSummary(ctx context.Context, filter StockMoveSummaryFilter) ([]StockMoveSummary, error) {
	if s == nil || s.repo == nil {
		return nil, ErrOLAPUnavailable
	}
	filter.RestaurantID = strings.TrimSpace(filter.RestaurantID)
	filter.BusinessDateFrom = strings.TrimSpace(filter.BusinessDateFrom)
	filter.BusinessDateTo = strings.TrimSpace(filter.BusinessDateTo)
	filter.CatalogItemID = strings.TrimSpace(filter.CatalogItemID)
	filter.WarehouseID = strings.TrimSpace(filter.WarehouseID)
	filter.SourceEventType = strings.TrimSpace(filter.SourceEventType)
	filter.GroupBy = strings.TrimSpace(filter.GroupBy)
	if filter.GroupBy == "" {
		filter.GroupBy = "business_date"
	}
	switch filter.GroupBy {
	case "business_date", "catalog_item", "warehouse":
	default:
		return nil, fmt.Errorf("%w: group_by must be business_date, catalog_item or warehouse", contracts.ErrInvalidEnvelope)
	}
	if err := validateBusinessDate(filter.BusinessDateFrom, "business_date_from"); err != nil {
		return nil, err
	}
	if err := validateBusinessDate(filter.BusinessDateTo, "business_date_to"); err != nil {
		return nil, err
	}
	if filter.BusinessDateFrom != "" && filter.BusinessDateTo != "" && filter.BusinessDateFrom > filter.BusinessDateTo {
		return nil, fmt.Errorf("%w: business_date_from must be before business_date_to", contracts.ErrInvalidEnvelope)
	}
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		return nil, fmt.Errorf("%w: offset must be non-negative", contracts.ErrInvalidEnvelope)
	}
	return s.repo.ListStockMoveSummary(ctx, filter)
}

// GetExportStatus возвращает bounded operator-facing состояние async OLAP export.
func (s *Service) GetExportStatus(ctx context.Context, stream string) (ExportStatus, error) {
	if s == nil || s.exportStatusRepo == nil {
		return ExportStatus{}, ErrOLAPUnavailable
	}
	stream = strings.TrimSpace(stream)
	switch stream {
	case "raw_business_events", "stock_moves":
	default:
		return ExportStatus{}, fmt.Errorf("%w: stream must be raw_business_events or stock_moves", contracts.ErrInvalidEnvelope)
	}
	return s.exportStatusRepo.GetExportStatus(ctx, stream, time.Now().UTC())
}

// RequestExportRetry снимает retry/backoff state для async OLAP export без синхронной записи в ClickHouse.
func (s *Service) RequestExportRetry(ctx context.Context, cmd ExportRetryCommand) (ExportRetryResult, error) {
	if s == nil || s.exportRetryRepo == nil {
		return ExportRetryResult{}, ErrOLAPUnavailable
	}
	cmd.CommandID = strings.TrimSpace(cmd.CommandID)
	cmd.Stream = strings.TrimSpace(cmd.Stream)
	cmd.Mode = strings.TrimSpace(cmd.Mode)
	cmd.Reason = strings.TrimSpace(cmd.Reason)
	if !isUUIDv7(cmd.CommandID) {
		return ExportRetryResult{}, fmt.Errorf("%w: command_id must be uuidv7", contracts.ErrInvalidEnvelope)
	}
	switch cmd.Stream {
	case "raw_business_events", "stock_moves":
	default:
		return ExportRetryResult{}, fmt.Errorf("%w: stream must be raw_business_events or stock_moves", contracts.ErrInvalidEnvelope)
	}
	switch cmd.Mode {
	case "retry_failed", "resume_from_checkpoint":
	default:
		return ExportRetryResult{}, fmt.Errorf("%w: mode must be retry_failed or resume_from_checkpoint", contracts.ErrInvalidEnvelope)
	}
	if cmd.Reason == "" {
		return ExportRetryResult{}, fmt.Errorf("%w: reason is required", contracts.ErrInvalidEnvelope)
	}
	if len(cmd.Reason) > 500 {
		return ExportRetryResult{}, fmt.Errorf("%w: reason must be 500 characters or less", contracts.ErrInvalidEnvelope)
	}
	return s.exportRetryRepo.RequestExportRetry(ctx, cmd, time.Now().UTC())
}

func validateBusinessDate(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("%w: %s must be YYYY-MM-DD", contracts.ErrInvalidEnvelope, name)
	}
	return nil
}

func isUUIDv7(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, r := range v {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
				return false
			}
		}
	}
	if v[14] != '7' {
		return false
	}
	variant := v[19]
	return variant == '8' || variant == '9' || variant == 'a' || variant == 'A' || variant == 'b' || variant == 'B'
}
