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

// RawBusinessEventRepository читает ClickHouse event archive без участия transactional command path.
type RawBusinessEventRepository interface {
	ListRawBusinessEvents(context.Context, RawBusinessEventFilter) ([]RawBusinessEvent, error)
}

// StockMoveRepository читает ClickHouse projection складских движений без raw payload.
type StockMoveRepository interface {
	ListStockMoves(context.Context, StockMoveFilter) ([]StockMove, error)
}

type Repository interface {
	RawBusinessEventRepository
	StockMoveRepository
}

// Service валидирует OLAP read API и делегирует bounded чтение ClickHouse repository.
type Service struct {
	repo Repository
}

// NewService создает read-only OLAP service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

func validateBusinessDate(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("%w: %s must be YYYY-MM-DD", contracts.ErrInvalidEnvelope, name)
	}
	return nil
}
