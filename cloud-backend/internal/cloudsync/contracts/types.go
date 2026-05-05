package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const EnvelopeVersion = "1"

type EventType string

const (
	EventShiftOpened     EventType = "ShiftOpened"
	EventShiftClosed     EventType = "ShiftClosed"
	EventOrderCreated    EventType = "OrderCreated"
	EventOrderLineAdded  EventType = "OrderLineAdded"
	EventCheckCreated    EventType = "CheckCreated"
	EventPaymentCaptured EventType = "PaymentCaptured"
	EventOrderClosed     EventType = "OrderClosed"
)

var (
	ErrInvalidEnvelope = errors.New("invalid sync envelope")
	ErrPayloadConflict = errors.New("sync envelope payload conflicts with accepted event")
)

type SyncEnvelope struct {
	Version       string          `json:"version"`
	EventID       string          `json:"event_id"`
	CommandID     string          `json:"command_id"`
	EventType     EventType       `json:"event_type"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	RestaurantID  *string         `json:"restaurant_id,omitempty"`
	DeviceID      string          `json:"device_id"`
	ShiftID       *string         `json:"shift_id,omitempty"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
}

type Payload[T any] struct {
	Origin string `json:"origin"`
	Data   T      `json:"data"`
}

type ShiftOpened struct {
	ID                 string    `json:"id"`
	RestaurantID       string    `json:"restaurant_id"`
	DeviceID           string    `json:"device_id"`
	OpenedByEmployeeID string    `json:"opened_by_employee_id"`
	Status             string    `json:"status"`
	OpenedAt           time.Time `json:"opened_at"`
	OpeningCashAmount  int64     `json:"opening_cash_amount"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ShiftClosed struct {
	ID                 string     `json:"id"`
	RestaurantID       string     `json:"restaurant_id"`
	DeviceID           string     `json:"device_id"`
	OpenedByEmployeeID string     `json:"opened_by_employee_id"`
	ClosedByEmployeeID *string    `json:"closed_by_employee_id,omitempty"`
	Status             string     `json:"status"`
	OpenedAt           time.Time  `json:"opened_at"`
	ClosedAt           *time.Time `json:"closed_at,omitempty"`
	OpeningCashAmount  int64      `json:"opening_cash_amount"`
	ClosingCashAmount  *int64     `json:"closing_cash_amount,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type OrderCreated struct {
	ID           string     `json:"id"`
	EdgeOrderID  string     `json:"edge_order_id"`
	RestaurantID string     `json:"restaurant_id"`
	DeviceID     string     `json:"device_id"`
	ShiftID      string     `json:"shift_id"`
	Status       string     `json:"status"`
	TableName    string     `json:"table_name"`
	GuestCount   int        `json:"guest_count"`
	OpenedAt     time.Time  `json:"opened_at"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type OrderLineAdded struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	MenuItemID    string    `json:"menu_item_id"`
	CatalogItemID string    `json:"catalog_item_id"`
	Name          string    `json:"name"`
	Quantity      int64     `json:"quantity"`
	UnitPrice     int64     `json:"unit_price"`
	TotalPrice    int64     `json:"total_price"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CheckCreated struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	Status        string    `json:"status"`
	Subtotal      int64     `json:"subtotal"`
	DiscountTotal int64     `json:"discount_total"`
	TaxTotal      int64     `json:"tax_total"`
	Total         int64     `json:"total"`
	PaidTotal     int64     `json:"paid_total"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PaymentCaptured struct {
	ID        string    `json:"id"`
	CheckID   string    `json:"check_id"`
	Method    string    `json:"method"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrderClosed = OrderCreated

type EventAck struct {
	Status              string    `json:"status"`
	IdempotencyKey      string    `json:"idempotency_key"`
	CloudReceiptID      string    `json:"cloud_receipt_id"`
	CommandID           string    `json:"command_id"`
	EventID             string    `json:"event_id"`
	EdgeEventID         string    `json:"edge_event_id"`
	EnvelopeVersion     string    `json:"envelope_version"`
	CloudReceivedAt     time.Time `json:"cloud_received_at"`
	RawPayloadSHA256Hex string    `json:"raw_payload_sha256_hex"`
}

func ValidateEnvelope(v SyncEnvelope) error {
	if strings.TrimSpace(v.Version) != EnvelopeVersion {
		return fmt.Errorf("%w: version must be %s", ErrInvalidEnvelope, EnvelopeVersion)
	}
	if strings.TrimSpace(v.EventID) == "" || strings.TrimSpace(v.CommandID) == "" {
		return fmt.Errorf("%w: event_id and command_id are required", ErrInvalidEnvelope)
	}
	if !IsKnownEventType(v.EventType) {
		return fmt.Errorf("%w: unsupported event_type %q", ErrInvalidEnvelope, v.EventType)
	}
	if strings.TrimSpace(v.AggregateType) == "" || strings.TrimSpace(v.AggregateID) == "" {
		return fmt.Errorf("%w: aggregate_type and aggregate_id are required", ErrInvalidEnvelope)
	}
	if v.RestaurantID == nil || strings.TrimSpace(*v.RestaurantID) == "" {
		return fmt.Errorf("%w: restaurant_id is required", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(v.DeviceID) == "" {
		return fmt.Errorf("%w: device_id is required", ErrInvalidEnvelope)
	}
	if v.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurred_at is required", ErrInvalidEnvelope)
	}
	if len(v.Payload) == 0 || string(v.Payload) == "null" {
		return fmt.Errorf("%w: payload is required", ErrInvalidEnvelope)
	}
	if err := ValidateEventPayload(v); err != nil {
		return err
	}
	return nil
}

func ValidateEventPayload(v SyncEnvelope) error {
	switch v.EventType {
	case EventShiftOpened:
		return validatePayload[ShiftOpened](v)
	case EventShiftClosed:
		return validatePayload[ShiftClosed](v)
	case EventOrderCreated:
		return validatePayload[OrderCreated](v)
	case EventOrderLineAdded:
		return validatePayload[OrderLineAdded](v)
	case EventCheckCreated:
		return validatePayload[CheckCreated](v)
	case EventPaymentCaptured:
		return validatePayload[PaymentCaptured](v)
	case EventOrderClosed:
		return validatePayload[OrderClosed](v)
	default:
		return fmt.Errorf("%w: unsupported event_type %q", ErrInvalidEnvelope, v.EventType)
	}
}

func validatePayload[T any](v SyncEnvelope) error {
	var payload Payload[T]
	if err := json.Unmarshal(v.Payload, &payload); err != nil {
		return fmt.Errorf("%w: invalid %s payload: %v", ErrInvalidEnvelope, v.EventType, err)
	}
	if strings.TrimSpace(payload.Origin) == "" {
		return fmt.Errorf("%w: payload.origin is required", ErrInvalidEnvelope)
	}
	data, err := json.Marshal(payload.Data)
	if err != nil {
		return err
	}
	if string(data) == "null" || string(data) == "{}" {
		return fmt.Errorf("%w: payload.data is required", ErrInvalidEnvelope)
	}
	return nil
}

func IsKnownEventType(v EventType) bool {
	switch v {
	case EventShiftOpened, EventShiftClosed, EventOrderCreated, EventOrderLineAdded, EventCheckCreated, EventPaymentCaptured, EventOrderClosed:
		return true
	default:
		return false
	}
}

func EdgeEventID(v SyncEnvelope) string {
	return strings.TrimSpace(v.EventID)
}

func IdempotencyKey(v SyncEnvelope) (string, error) {
	if err := ValidateEnvelope(v); err != nil {
		return "", err
	}
	return strings.Join([]string{
		strings.TrimSpace(*v.RestaurantID),
		strings.TrimSpace(v.DeviceID),
		EdgeEventID(v),
	}, ":"), nil
}
