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
	EventShiftOpened              EventType = "ShiftOpened"
	EventShiftClosed              EventType = "ShiftClosed"
	EventOrderCreated             EventType = "OrderCreated"
	EventOrderLineAdded           EventType = "OrderLineAdded"
	EventOrderLineQuantityChanged EventType = "OrderLineQuantityChanged"
	EventOrderLineVoided          EventType = "OrderLineVoided"
	EventPrecheckIssued           EventType = "PrecheckIssued"
	EventPrecheckReprinted        EventType = "PrecheckReprinted"
	EventPrecheckCancelled        EventType = "PrecheckCancelled"
	EventCheckCreated             EventType = "CheckCreated"
	EventCheckRefunded            EventType = "CheckRefunded"
	EventCheckReprinted           EventType = "CheckReprinted"
	EventPaymentCaptured          EventType = "PaymentCaptured"
	EventPaymentRefunded          EventType = "PaymentRefunded"
	EventCancellationRecorded     EventType = "CancellationRecorded"
	EventRefundRecorded           EventType = "RefundRecorded"
	EventOrderClosed              EventType = "OrderClosed"
	EventCashSessionOpened        EventType = "CashSessionOpened"
	EventCashSessionClosed        EventType = "CashSessionClosed"
	EventCashDrawerEventRecorded  EventType = "CashDrawerEventRecorded"
	EventAuthSessionStarted       EventType = "AuthSessionStarted"
	EventAuthSessionRevoked       EventType = "AuthSessionRevoked"
	EventDeviceRegistered         EventType = "DeviceRegistered"
)

var (
	ErrInvalidEnvelope = errors.New("invalid sync envelope")
	ErrPayloadConflict = errors.New("sync envelope payload conflicts with accepted event")
	ErrNotFound        = errors.New("not found")
)

type SyncEnvelope struct {
	Version         string          `json:"version"`
	EventID         string          `json:"event_id"`
	CommandID       string          `json:"command_id"`
	EventType       EventType       `json:"event_type"`
	AggregateType   string          `json:"aggregate_type"`
	AggregateID     string          `json:"aggregate_id"`
	RestaurantID    *string         `json:"restaurant_id,omitempty"`
	DeviceID        string          `json:"device_id"`
	NodeDeviceID    string          `json:"node_device_id,omitempty"`
	ClientDeviceID  *string         `json:"client_device_id,omitempty"`
	ShiftID         *string         `json:"shift_id,omitempty"`
	ActorEmployeeID *string         `json:"actor_employee_id,omitempty"`
	SessionID       *string         `json:"session_id,omitempty"`
	OccurredAt      time.Time       `json:"occurred_at"`
	Payload         json.RawMessage `json:"payload"`
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
	ID                string          `json:"id"`
	OrderID           string          `json:"order_id"`
	Status            string          `json:"status"`
	Subtotal          int64           `json:"subtotal"`
	DiscountTotal     int64           `json:"discount_total"`
	TaxTotal          int64           `json:"tax_total"`
	Total             int64           `json:"total"`
	PaidTotal         int64           `json:"paid_total"`
	BusinessDateLocal string          `json:"business_date_local"`
	ClosedAt          time.Time       `json:"closed_at"`
	Snapshot          json.RawMessage `json:"snapshot,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type PaymentCaptured struct {
	ID                    string    `json:"id"`
	EdgePaymentID         string    `json:"edge_payment_id"`
	RestaurantID          string    `json:"restaurant_id"`
	DeviceID              string    `json:"device_id"`
	ShiftID               string    `json:"shift_id"`
	PrecheckID            string    `json:"precheck_id"`
	Method                string    `json:"method"`
	Amount                int64     `json:"amount"`
	Currency              string    `json:"currency"`
	Status                string    `json:"status"`
	BusinessDateLocal     string    `json:"business_date_local"`
	ProviderName          *string   `json:"provider_name,omitempty"`
	ProviderTransactionID *string   `json:"provider_transaction_id,omitempty"`
	ProviderReference     *string   `json:"provider_reference,omitempty"`
	FingerprintHash       *string   `json:"fingerprint_hash,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// PaymentRefunded использует payload shape платежа, где status фиксирует возврат.
type PaymentRefunded = PaymentCaptured

// CheckRefunded использует check payload shape для подтвержденного возврата чека.
type CheckRefunded = CheckCreated

type FinancialOperationRecorded struct {
	ID                   string          `json:"id"`
	EdgeOperationID      string          `json:"edge_operation_id"`
	RestaurantID         string          `json:"restaurant_id"`
	DeviceID             string          `json:"device_id"`
	ShiftID              string          `json:"shift_id"`
	OriginalShiftID      string          `json:"original_shift_id"`
	CheckID              string          `json:"check_id"`
	PrecheckID           string          `json:"precheck_id"`
	OperationType        string          `json:"operation_type"`
	OperationKind        string          `json:"operation_kind"`
	Status               string          `json:"status"`
	Amount               int64           `json:"amount"`
	Currency             string          `json:"currency"`
	BusinessDateLocal    string          `json:"business_date_local"`
	InventoryDisposition string          `json:"inventory_disposition"`
	Reason               string          `json:"reason"`
	CreatedByEmployeeID  string          `json:"created_by_employee_id,omitempty"`
	ApprovedByEmployeeID *string         `json:"approved_by_employee_id,omitempty"`
	Snapshot             json.RawMessage `json:"snapshot,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
}

// FinancialOperationProjection описывает Cloud read model ledger operation без чтения mutable POS state.
type FinancialOperationProjection struct {
	OperationID          string          `json:"operation_id"`
	EdgeOperationID      string          `json:"edge_operation_id"`
	EventID              string          `json:"event_id"`
	ReceiptID            string          `json:"receipt_id"`
	RestaurantID         string          `json:"restaurant_id"`
	DeviceID             string          `json:"device_id"`
	NodeDeviceID         *string         `json:"node_device_id,omitempty"`
	ClientDeviceID       *string         `json:"client_device_id,omitempty"`
	ActorEmployeeID      *string         `json:"actor_employee_id,omitempty"`
	SessionID            *string         `json:"session_id,omitempty"`
	ShiftID              string          `json:"shift_id"`
	OriginalShiftID      string          `json:"original_shift_id"`
	CheckID              string          `json:"check_id"`
	PrecheckID           string          `json:"precheck_id"`
	OperationType        string          `json:"operation_type"`
	OperationKind        string          `json:"operation_kind"`
	Amount               int64           `json:"amount"`
	Currency             string          `json:"currency"`
	BusinessDateLocal    string          `json:"business_date_local"`
	InventoryDisposition string          `json:"inventory_disposition"`
	Reason               string          `json:"reason"`
	CreatedByEmployeeID  string          `json:"created_by_employee_id,omitempty"`
	ApprovedByEmployeeID *string         `json:"approved_by_employee_id,omitempty"`
	Snapshot             json.RawMessage `json:"snapshot,omitempty"`
	OperationCreatedAt   time.Time       `json:"operation_created_at"`
	OccurredAt           time.Time       `json:"occurred_at"`
	CloudReceivedAt      time.Time       `json:"cloud_received_at"`
	RawPayloadSHA256Hex  string          `json:"raw_payload_sha256_hex"`
}

type OrderClosed = OrderCreated

type CashSessionOpened struct {
	ID                 string    `json:"id"`
	EdgeCashSessionID  string    `json:"edge_cash_session_id"`
	RestaurantID       string    `json:"restaurant_id"`
	DeviceID           string    `json:"device_id"`
	ShiftID            string    `json:"shift_id"`
	OpenedByEmployeeID string    `json:"opened_by_employee_id"`
	Status             string    `json:"status"`
	OpeningCashAmount  int64     `json:"opening_cash_amount"`
	OpenedAt           time.Time `json:"opened_at"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CashSessionClosed struct {
	ID                 string     `json:"id"`
	EdgeCashSessionID  string     `json:"edge_cash_session_id"`
	RestaurantID       string     `json:"restaurant_id"`
	DeviceID           string     `json:"device_id"`
	ShiftID            string     `json:"shift_id"`
	OpenedByEmployeeID string     `json:"opened_by_employee_id"`
	ClosedByEmployeeID *string    `json:"closed_by_employee_id,omitempty"`
	Status             string     `json:"status"`
	OpeningCashAmount  int64      `json:"opening_cash_amount"`
	ClosingCashAmount  *int64     `json:"closing_cash_amount,omitempty"`
	OpenedAt           time.Time  `json:"opened_at"`
	ClosedAt           *time.Time `json:"closed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type CashDrawerEventRecorded struct {
	ID                    string    `json:"id"`
	EdgeCashDrawerEventID string    `json:"edge_cash_drawer_event_id"`
	CashSessionID         string    `json:"cash_session_id"`
	RestaurantID          string    `json:"restaurant_id"`
	DeviceID              string    `json:"device_id"`
	ShiftID               string    `json:"shift_id"`
	CreatedByEmployeeID   string    `json:"created_by_employee_id"`
	EventType             string    `json:"event_type"`
	Amount                int64     `json:"amount"`
	Reason                *string   `json:"reason,omitempty"`
	Note                  *string   `json:"note,omitempty"`
	OccurredAt            time.Time `json:"occurred_at"`
	CreatedAt             time.Time `json:"created_at"`
}

type ReprintDocument struct {
	DocumentType    string          `json:"document_type"`
	SourceID        string          `json:"source_id"`
	CopyMarker      string          `json:"copy_marker"`
	ActorEmployeeID string          `json:"actor_employee_id,omitempty"`
	ReprintedAt     time.Time       `json:"reprinted_at"`
	Snapshot        json.RawMessage `json:"snapshot"`
}

// EdgeEventView описывает безопасную строку журнала входящих Edge events для Cloud UI.
type EdgeEventView struct {
	CloudReceiptID      string    `json:"cloud_receipt_id"`
	IdempotencyKey      string    `json:"idempotency_key"`
	RestaurantID        string    `json:"restaurant_id"`
	DeviceID            string    `json:"device_id"`
	CommandID           string    `json:"command_id"`
	EventID             string    `json:"event_id"`
	EdgeEventID         string    `json:"edge_event_id"`
	EventType           string    `json:"event_type"`
	AggregateType       string    `json:"aggregate_type"`
	AggregateID         string    `json:"aggregate_id"`
	EnvelopeVersion     string    `json:"envelope_version"`
	OccurredAt          time.Time `json:"occurred_at"`
	CloudReceivedAt     time.Time `json:"cloud_received_at"`
	RawPayloadSHA256Hex string    `json:"raw_payload_sha256_hex"`
}

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
	if !isUUIDv7(strings.TrimSpace(v.EventID)) {
		return fmt.Errorf("%w: event_id must be uuidv7", ErrInvalidEnvelope)
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
	case EventOrderLineQuantityChanged, EventOrderLineVoided, EventPrecheckIssued, EventPrecheckCancelled, EventAuthSessionStarted, EventAuthSessionRevoked, EventDeviceRegistered:
		return validateOperationalPayload(v)
	case EventPrecheckReprinted, EventCheckReprinted:
		return validatePayload[ReprintDocument](v)
	case EventCheckCreated:
		return validatePayload[CheckCreated](v)
	case EventCheckRefunded:
		return validatePayload[CheckRefunded](v)
	case EventPaymentCaptured:
		return validatePayload[PaymentCaptured](v)
	case EventPaymentRefunded:
		return validatePayload[PaymentRefunded](v)
	case EventCancellationRecorded, EventRefundRecorded:
		return validateFinancialOperationRecordedPayload(v)
	case EventOrderClosed:
		return validatePayload[OrderClosed](v)
	case EventCashSessionOpened:
		return validatePayload[CashSessionOpened](v)
	case EventCashSessionClosed:
		return validatePayload[CashSessionClosed](v)
	case EventCashDrawerEventRecorded:
		return validatePayload[CashDrawerEventRecorded](v)
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

func validateOperationalPayload(v SyncEnvelope) error {
	var payload Payload[map[string]any]
	if err := json.Unmarshal(v.Payload, &payload); err != nil {
		return fmt.Errorf("%w: invalid %s payload: %v", ErrInvalidEnvelope, v.EventType, err)
	}
	if strings.TrimSpace(payload.Origin) == "" {
		return fmt.Errorf("%w: payload.origin is required", ErrInvalidEnvelope)
	}
	if len(payload.Data) == 0 {
		return fmt.Errorf("%w: payload.data is required", ErrInvalidEnvelope)
	}
	return nil
}

func validateFinancialOperationRecordedPayload(v SyncEnvelope) error {
	var payload Payload[FinancialOperationRecorded]
	if err := json.Unmarshal(v.Payload, &payload); err != nil {
		return fmt.Errorf("%w: invalid %s payload: %v", ErrInvalidEnvelope, v.EventType, err)
	}
	if strings.TrimSpace(payload.Origin) == "" {
		return fmt.Errorf("%w: payload.origin is required", ErrInvalidEnvelope)
	}
	data := payload.Data
	if strings.TrimSpace(data.ID) == "" ||
		strings.TrimSpace(data.EdgeOperationID) == "" ||
		strings.TrimSpace(data.RestaurantID) == "" ||
		strings.TrimSpace(data.DeviceID) == "" ||
		strings.TrimSpace(data.CheckID) == "" ||
		strings.TrimSpace(data.PrecheckID) == "" ||
		strings.TrimSpace(data.OriginalShiftID) == "" ||
		strings.TrimSpace(data.ShiftID) == "" {
		return fmt.Errorf("%w: financial operation id, edge_operation_id, restaurant_id, device_id, check_id, precheck_id, original_shift_id and shift_id are required", ErrInvalidEnvelope)
	}
	if v.RestaurantID == nil || strings.TrimSpace(*v.RestaurantID) != strings.TrimSpace(data.RestaurantID) {
		return fmt.Errorf("%w: financial operation restaurant_id must match envelope restaurant_id", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(v.DeviceID) != strings.TrimSpace(data.DeviceID) {
		return fmt.Errorf("%w: financial operation device_id must match envelope device_id", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(v.AggregateType) != "FinancialOperation" || strings.TrimSpace(v.AggregateID) != strings.TrimSpace(data.ID) {
		return fmt.Errorf("%w: financial operation aggregate must match payload id", ErrInvalidEnvelope)
	}
	if v.ShiftID == nil || strings.TrimSpace(*v.ShiftID) != strings.TrimSpace(data.ShiftID) {
		return fmt.Errorf("%w: financial operation shift_id must match envelope shift_id", ErrInvalidEnvelope)
	}
	expectedType := "refund"
	if v.EventType == EventCancellationRecorded {
		expectedType = "cancellation"
	}
	if strings.TrimSpace(data.OperationType) != expectedType {
		return fmt.Errorf("%w: financial operation type does not match event_type", ErrInvalidEnvelope)
	}
	if data.OperationKind != "full" && data.OperationKind != "partial" {
		return fmt.Errorf("%w: financial operation kind is invalid", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(data.Status) != "recorded" {
		return fmt.Errorf("%w: financial operation status must be recorded", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(data.Reason) == "" {
		return fmt.Errorf("%w: financial operation reason is required", ErrInvalidEnvelope)
	}
	if data.Amount <= 0 || !validCurrency(data.Currency) {
		return fmt.Errorf("%w: financial operation amount and currency are required", ErrInvalidEnvelope)
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(data.BusinessDateLocal)); err != nil {
		return fmt.Errorf("%w: financial operation business_date_local must use YYYY-MM-DD", ErrInvalidEnvelope)
	}
	switch data.InventoryDisposition {
	case "no_stock_effect", "return_to_stock", "write_off_waste", "manual_review":
	default:
		return fmt.Errorf("%w: financial operation inventory_disposition is invalid", ErrInvalidEnvelope)
	}
	if len(data.Snapshot) == 0 || string(data.Snapshot) == "null" || !json.Valid(data.Snapshot) {
		return fmt.Errorf("%w: financial operation snapshot is required", ErrInvalidEnvelope)
	}
	if data.CreatedAt.IsZero() {
		return fmt.Errorf("%w: financial operation created_at is required", ErrInvalidEnvelope)
	}
	return nil
}

func validCurrency(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) != 3 {
		return false
	}
	for _, r := range v {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func IsKnownEventType(v EventType) bool {
	switch v {
	case EventShiftOpened, EventShiftClosed, EventOrderCreated, EventOrderLineAdded, EventOrderLineQuantityChanged, EventOrderLineVoided, EventPrecheckIssued, EventPrecheckReprinted, EventPrecheckCancelled, EventCheckCreated, EventCheckRefunded, EventCheckReprinted, EventPaymentCaptured, EventPaymentRefunded, EventCancellationRecorded, EventRefundRecorded, EventOrderClosed, EventCashSessionOpened, EventCashSessionClosed, EventCashDrawerEventRecorded, EventAuthSessionStarted, EventAuthSessionRevoked, EventDeviceRegistered:
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
