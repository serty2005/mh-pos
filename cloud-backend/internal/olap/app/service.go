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

// RawBusinessEventRepository читает ClickHouse event archive без участия transactional command path.
type RawBusinessEventRepository interface {
	ListRawBusinessEvents(context.Context, RawBusinessEventFilter) ([]RawBusinessEvent, error)
}

// Service валидирует OLAP read API и делегирует bounded чтение ClickHouse repository.
type Service struct {
	repo RawBusinessEventRepository
}

// NewService создает read-only OLAP service.
func NewService(repo RawBusinessEventRepository) *Service {
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
