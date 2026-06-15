package contracts

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const EnvelopeVersion = "1"

type EventType string

const (
	EventShiftOpened                EventType = "ShiftOpened"
	EventShiftClosed                EventType = "ShiftClosed"
	EventOrderCreated               EventType = "OrderCreated"
	EventOrderLineAdded             EventType = "OrderLineAdded"
	EventOrderLineQuantityChanged   EventType = "OrderLineQuantityChanged"
	EventOrderLineVoided            EventType = "OrderLineVoided"
	EventPrecheckIssued             EventType = "PrecheckIssued"
	EventPrecheckReprinted          EventType = "PrecheckReprinted"
	EventPrecheckCancelled          EventType = "PrecheckCancelled"
	EventCheckCreated               EventType = "CheckCreated"
	EventCheckRefunded              EventType = "CheckRefunded"
	EventCheckReprinted             EventType = "CheckReprinted"
	EventPaymentCaptured            EventType = "PaymentCaptured"
	EventPaymentRefunded            EventType = "PaymentRefunded"
	EventCancellationRecorded       EventType = "CancellationRecorded"
	EventRefundRecorded             EventType = "RefundRecorded"
	EventCheckClosed                EventType = "CheckClosed"
	EventKitchenTicketStatusChanged EventType = "KitchenTicketStatusChanged"
	EventItemServed                 EventType = "ItemServed"
	EventStockReceiptCaptured       EventType = "StockReceiptCaptured"
	EventInventoryCountCaptured     EventType = "InventoryCountCaptured"
	EventStockWriteOffCaptured      EventType = "StockWriteOffCaptured"
	EventProductionCompleted        EventType = "ProductionCompleted"
	EventStopListUpdated            EventType = "StopListUpdated"
	EventCatalogItemChangeSuggested EventType = "CatalogItemChangeSuggested"
	EventRecipeChangeSuggested      EventType = "RecipeChangeSuggested"
	EventOrderClosed                EventType = "OrderClosed"
	EventCashSessionOpened          EventType = "CashSessionOpened"
	EventCashSessionClosed          EventType = "CashSessionClosed"
	EventCashDrawerEventRecorded    EventType = "CashDrawerEventRecorded"
	EventAuthSessionStarted         EventType = "AuthSessionStarted"
	EventAuthSessionRevoked         EventType = "AuthSessionRevoked"
	EventDeviceRegistered           EventType = "DeviceRegistered"
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

// InventoryLedgerEntry описывает bounded read-only Cloud inventory ledger view.
type InventoryLedgerEntry struct {
	ID                string    `json:"id"`
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
	CreatedAt         time.Time `json:"created_at"`
}

// InventoryStockBalance описывает bounded Cloud-owned materialized balance read model.
type InventoryStockBalance struct {
	RestaurantID       string    `json:"restaurant_id"`
	WarehouseID        string    `json:"warehouse_id,omitempty"`
	CatalogItemID      string    `json:"catalog_item_id"`
	QuantityOnHand     string    `json:"quantity_on_hand"`
	UnitCode           string    `json:"unit_code"`
	CostingStatus      string    `json:"costing_status"`
	NeedsRecalculation bool      `json:"needs_recalculation"`
	LastMovementAt     time.Time `json:"last_movement_at"`
	BusinessDateTo     string    `json:"business_date_to"`
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
	ID                   string                   `json:"id"`
	EdgeOperationID      string                   `json:"edge_operation_id"`
	RestaurantID         string                   `json:"restaurant_id"`
	DeviceID             string                   `json:"device_id"`
	ShiftID              string                   `json:"shift_id"`
	OriginalShiftID      string                   `json:"original_shift_id"`
	CheckID              string                   `json:"check_id"`
	PrecheckID           string                   `json:"precheck_id"`
	OperationType        string                   `json:"operation_type"`
	OperationKind        string                   `json:"operation_kind"`
	Status               string                   `json:"status"`
	Amount               int64                    `json:"amount"`
	Currency             string                   `json:"currency"`
	BusinessDateLocal    string                   `json:"business_date_local"`
	InventoryDisposition string                   `json:"inventory_disposition"`
	Reason               string                   `json:"reason"`
	CreatedByEmployeeID  string                   `json:"created_by_employee_id,omitempty"`
	ApprovedByEmployeeID *string                  `json:"approved_by_employee_id,omitempty"`
	Snapshot             json.RawMessage          `json:"snapshot,omitempty"`
	Items                []FinancialOperationItem `json:"items,omitempty"`
	CreatedAt            time.Time                `json:"created_at"`
}

// FinancialOperationItem принимает текущую форму POS ledger item и уже нормализованные складские поля.
type FinancialOperationItem struct {
	ID                   string          `json:"id,omitempty"`
	OperationID          string          `json:"operation_id,omitempty"`
	Scope                string          `json:"scope,omitempty"`
	OrderLineID          string          `json:"order_line_id,omitempty"`
	PaymentID            string          `json:"payment_id,omitempty"`
	Quantity             json.RawMessage `json:"quantity,omitempty"`
	Amount               int64           `json:"amount,omitempty"`
	Currency             string          `json:"currency,omitempty"`
	TaxAmount            int64           `json:"tax_amount,omitempty"`
	Snapshot             json.RawMessage `json:"snapshot,omitempty"`
	CatalogItemID        string          `json:"catalog_item_id,omitempty"`
	UnitCode             string          `json:"unit_code,omitempty"`
	InventoryDisposition string          `json:"inventory_disposition,omitempty"`
	Reason               string          `json:"reason,omitempty"`
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

type InventoryItem struct {
	OrderLineID          string              `json:"order_line_id,omitempty"`
	CatalogItemID        string              `json:"catalog_item_id"`
	Quantity             string              `json:"quantity"`
	CountedQuantity      string              `json:"counted_quantity,omitempty"`
	UnitCode             string              `json:"unit_code"`
	RequiredForInventory bool                `json:"required_for_inventory,omitempty"`
	UnitCostMinor        int64               `json:"unit_cost_minor,omitempty"`
	Currency             string              `json:"currency,omitempty"`
	Modifiers            []InventoryModifier `json:"modifiers,omitempty"`
}

type InventoryModifier struct {
	OrderLineModifierID string `json:"order_line_modifier_id,omitempty"`
	ModifierGroupID     string `json:"modifier_group_id,omitempty"`
	ModifierOptionID    string `json:"modifier_option_id"`
	Name                string `json:"name,omitempty"`
	Quantity            string `json:"quantity"`
	UnitCode            string `json:"unit_code,omitempty"`
}

type CheckClosed struct {
	CheckID           string          `json:"check_id"`
	OrderID           string          `json:"order_id"`
	PrecheckID        string          `json:"precheck_id"`
	RestaurantID      string          `json:"restaurant_id"`
	BusinessDateLocal string          `json:"business_date_local"`
	ClosedAt          time.Time       `json:"closed_at"`
	Items             []InventoryItem `json:"items"`
}

type ItemServed struct {
	ServedEventID           string    `json:"served_event_id"`
	TicketID                string    `json:"ticket_id"`
	ServeSequence           int       `json:"serve_sequence"`
	SupersedesServedEventID string    `json:"supersedes_served_event_id,omitempty"`
	RestaurantID            string    `json:"restaurant_id,omitempty"`
	OrderID                 string    `json:"order_id"`
	OrderLineID             string    `json:"order_line_id"`
	CatalogItemID           string    `json:"catalog_item_id"`
	Quantity                string    `json:"quantity"`
	UnitCode                string    `json:"unit_code"`
	ServedAt                time.Time `json:"served_at"`
	ChangedByEmployeeID     string    `json:"changed_by_employee_id,omitempty"`
	StationID               string    `json:"station_id,omitempty"`
}

type KitchenTicketStatusChanged struct {
	TicketID    string    `json:"ticket_id"`
	OrderID     string    `json:"order_id"`
	OrderLineID string    `json:"order_line_id"`
	FromStatus  string    `json:"from_status"`
	ToStatus    string    `json:"to_status"`
	ChangedAt   time.Time `json:"changed_at"`
	StationID   string    `json:"station_id,omitempty"`
}

type StockReceiptCaptured struct {
	ReceiptID         string          `json:"receipt_id"`
	RestaurantID      string          `json:"restaurant_id"`
	WarehouseID       string          `json:"warehouse_id,omitempty"`
	ReceivedAt        time.Time       `json:"received_at"`
	BusinessDateLocal string          `json:"business_date_local"`
	SupplierID        string          `json:"supplier_id,omitempty"`
	SupplierName      string          `json:"supplier_name_snapshot,omitempty"`
	DocumentNumber    string          `json:"document_number,omitempty"`
	DocumentDate      string          `json:"document_date,omitempty"`
	Items             []InventoryItem `json:"items"`
}

type InventoryCountCaptured struct {
	CountID           string          `json:"count_id"`
	RestaurantID      string          `json:"restaurant_id"`
	WarehouseID       string          `json:"warehouse_id,omitempty"`
	CountedAt         time.Time       `json:"counted_at"`
	BusinessDateLocal string          `json:"business_date_local"`
	Items             []InventoryItem `json:"items"`
}

type StockWriteOffCaptured struct {
	WriteOffID        string          `json:"write_off_id"`
	RestaurantID      string          `json:"restaurant_id"`
	WarehouseID       string          `json:"warehouse_id,omitempty"`
	ReasonCode        string          `json:"reason_code"`
	ReasonText        string          `json:"reason_text,omitempty"`
	WrittenOffAt      time.Time       `json:"written_off_at"`
	BusinessDateLocal string          `json:"business_date_local"`
	Items             []InventoryItem `json:"items"`
}

type ProductionCompleted struct {
	ProductionID              string    `json:"production_id"`
	RestaurantID              string    `json:"restaurant_id"`
	WarehouseID               string    `json:"warehouse_id,omitempty"`
	SemiFinishedCatalogItemID string    `json:"semi_finished_catalog_item_id"`
	Quantity                  string    `json:"quantity"`
	UnitCode                  string    `json:"unit_code"`
	CompletedAt               time.Time `json:"completed_at"`
	BusinessDateLocal         string    `json:"business_date_local"`
}

type StopListUpdated struct {
	StopListID        string    `json:"stop_list_id"`
	RestaurantID      string    `json:"restaurant_id"`
	WarehouseID       string    `json:"warehouse_id,omitempty"`
	CatalogItemID     string    `json:"catalog_item_id"`
	AvailableQuantity string    `json:"available_quantity,omitempty"`
	Active            bool      `json:"active"`
	ConflictPolicy    string    `json:"conflict_policy,omitempty"`
	Source            string    `json:"source"`
	Reason            string    `json:"reason,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type StopListConflictPolicy string

const (
	StopListConflictPolicyCloudWins                        StopListConflictPolicy = "cloud_wins"
	StopListConflictPolicyEdgeOverlayUntilNextPublication  StopListConflictPolicy = "edge_overlay_until_next_publication"
	StopListConflictPolicyEdgeOverlayRequiresManagerReview StopListConflictPolicy = "edge_overlay_requires_manager_review"
	DefaultStopListConflictPolicy                          StopListConflictPolicy = StopListConflictPolicyEdgeOverlayRequiresManagerReview
)

func NormalizeStopListConflictPolicy(value string) StopListConflictPolicy {
	switch StopListConflictPolicy(strings.TrimSpace(value)) {
	case StopListConflictPolicyCloudWins:
		return StopListConflictPolicyCloudWins
	case StopListConflictPolicyEdgeOverlayUntilNextPublication:
		return StopListConflictPolicyEdgeOverlayUntilNextPublication
	case StopListConflictPolicyEdgeOverlayRequiresManagerReview, "":
		return StopListConflictPolicyEdgeOverlayRequiresManagerReview
	default:
		return ""
	}
}

type CatalogItemChangeSuggested struct {
	SuggestionID      string    `json:"suggestion_id"`
	RestaurantID      string    `json:"restaurant_id"`
	CatalogItemID     string    `json:"catalog_item_id,omitempty"`
	ProposalGroupID   string    `json:"proposal_group_id,omitempty"`
	Action            string    `json:"action"`
	Reason            string    `json:"reason,omitempty"`
	SuggestedAt       time.Time `json:"suggested_at"`
	BusinessDateLocal string    `json:"business_date_local,omitempty"`
}

type RecipeChangeSuggested struct {
	SuggestionID       string    `json:"suggestion_id"`
	RestaurantID       string    `json:"restaurant_id"`
	RecipeVersionID    string    `json:"recipe_version_id,omitempty"`
	OwnerCatalogItemID string    `json:"owner_catalog_item_id,omitempty"`
	ProposalGroupID    string    `json:"proposal_group_id,omitempty"`
	Action             string    `json:"action"`
	Reason             string    `json:"reason,omitempty"`
	SuggestedAt        time.Time `json:"suggested_at"`
	BusinessDateLocal  string    `json:"business_date_local,omitempty"`
}

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
	case EventCheckClosed:
		return validateCheckClosedPayload(v)
	case EventKitchenTicketStatusChanged:
		return validateKitchenTicketStatusChangedPayload(v)
	case EventItemServed:
		return validateItemServedPayload(v)
	case EventStockReceiptCaptured:
		return validateStockReceiptCapturedPayload(v)
	case EventInventoryCountCaptured:
		return validateInventoryCountCapturedPayload(v)
	case EventStockWriteOffCaptured:
		return validateStockWriteOffCapturedPayload(v)
	case EventProductionCompleted:
		return validateProductionCompletedPayload(v)
	case EventStopListUpdated:
		return validateStopListUpdatedPayload(v)
	case EventCatalogItemChangeSuggested:
		return validateCatalogItemChangeSuggestedPayload(v)
	case EventRecipeChangeSuggested:
		return validateRecipeChangeSuggestedPayload(v)
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
	if data.InventoryDisposition != "no_stock_effect" && len(data.Items) == 0 {
		return fmt.Errorf("%w: financial operation stock disposition requires items", ErrInvalidEnvelope)
	}
	for _, item := range data.Items {
		switch strings.TrimSpace(item.Scope) {
		case "", "whole_check", "order_line", "modifier_line", "service_charge", "tip", "payment":
		default:
			return fmt.Errorf("%w: financial operation item scope is invalid", ErrInvalidEnvelope)
		}
		if len(item.Quantity) > 0 && string(item.Quantity) != "null" && !positiveJSONNumber(item.Quantity) {
			return fmt.Errorf("%w: financial operation item quantity must be positive", ErrInvalidEnvelope)
		}
		if item.TaxAmount < 0 {
			return fmt.Errorf("%w: financial operation item tax_amount must be non-negative", ErrInvalidEnvelope)
		}
	}
	return nil
}

func validateCheckClosedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[CheckClosed](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.CheckID) == "" || strings.TrimSpace(data.OrderID) == "" || strings.TrimSpace(data.PrecheckID) == "" || strings.TrimSpace(data.RestaurantID) == "" {
		return fmt.Errorf("%w: check closed ids are required", ErrInvalidEnvelope)
	}
	if err := validateBusinessDate(data.BusinessDateLocal, "check closed business_date_local"); err != nil {
		return err
	}
	if data.ClosedAt.IsZero() || len(data.Items) == 0 {
		return fmt.Errorf("%w: check closed closed_at and items are required", ErrInvalidEnvelope)
	}
	return validateInventoryItems(data.Items, false)
}

func validateItemServedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[ItemServed](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.ServedEventID) == "" || strings.TrimSpace(data.TicketID) == "" || strings.TrimSpace(data.OrderID) == "" || strings.TrimSpace(data.OrderLineID) == "" || strings.TrimSpace(data.CatalogItemID) == "" {
		return fmt.Errorf("%w: item served ids are required", ErrInvalidEnvelope)
	}
	if data.ServeSequence <= 0 {
		return fmt.Errorf("%w: item served serve_sequence must be positive", ErrInvalidEnvelope)
	}
	if !positiveDecimal(data.Quantity) || strings.TrimSpace(data.UnitCode) == "" || data.ServedAt.IsZero() {
		return fmt.Errorf("%w: item served quantity, unit_code and served_at are required", ErrInvalidEnvelope)
	}
	return nil
}

func validateKitchenTicketStatusChangedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[KitchenTicketStatusChanged](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.TicketID) == "" || strings.TrimSpace(data.OrderID) == "" || strings.TrimSpace(data.OrderLineID) == "" {
		return fmt.Errorf("%w: kitchen ticket status ids are required", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(data.FromStatus) == "" || strings.TrimSpace(data.ToStatus) == "" || data.ChangedAt.IsZero() {
		return fmt.Errorf("%w: kitchen ticket statuses and changed_at are required", ErrInvalidEnvelope)
	}
	return nil
}

func validateStockReceiptCapturedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[StockReceiptCaptured](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.ReceiptID) == "" || strings.TrimSpace(data.RestaurantID) == "" || data.ReceivedAt.IsZero() {
		return fmt.Errorf("%w: stock receipt id, restaurant_id and received_at are required", ErrInvalidEnvelope)
	}
	if err := validateBusinessDate(data.BusinessDateLocal, "stock receipt business_date_local"); err != nil {
		return err
	}
	return validateInventoryItems(data.Items, false)
}

func validateInventoryCountCapturedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[InventoryCountCaptured](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.CountID) == "" || strings.TrimSpace(data.RestaurantID) == "" || data.CountedAt.IsZero() {
		return fmt.Errorf("%w: inventory count id, restaurant_id and counted_at are required", ErrInvalidEnvelope)
	}
	if err := validateBusinessDate(data.BusinessDateLocal, "inventory count business_date_local"); err != nil {
		return err
	}
	return validateInventoryItems(data.Items, true)
}

func validateProductionCompletedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[ProductionCompleted](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.ProductionID) == "" || strings.TrimSpace(data.RestaurantID) == "" || strings.TrimSpace(data.SemiFinishedCatalogItemID) == "" {
		return fmt.Errorf("%w: production ids are required", ErrInvalidEnvelope)
	}
	if err := validateBusinessDate(data.BusinessDateLocal, "production business_date_local"); err != nil {
		return err
	}
	if !positiveDecimal(data.Quantity) || strings.TrimSpace(data.UnitCode) == "" || data.CompletedAt.IsZero() {
		return fmt.Errorf("%w: production quantity, unit_code and completed_at are required", ErrInvalidEnvelope)
	}
	return nil
}

func validateStockWriteOffCapturedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[StockWriteOffCaptured](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.WriteOffID) == "" || strings.TrimSpace(data.RestaurantID) == "" || strings.TrimSpace(data.ReasonCode) == "" {
		return fmt.Errorf("%w: stock write-off id, restaurant_id and reason_code are required", ErrInvalidEnvelope)
	}
	if err := validateBusinessDate(data.BusinessDateLocal, "stock write-off business_date_local"); err != nil {
		return err
	}
	if data.WrittenOffAt.IsZero() {
		return fmt.Errorf("%w: stock write-off written_off_at is required", ErrInvalidEnvelope)
	}
	return validateInventoryItems(data.Items, false)
}

func validateStopListUpdatedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[StopListUpdated](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.StopListID) == "" || strings.TrimSpace(data.RestaurantID) == "" || strings.TrimSpace(data.CatalogItemID) == "" || strings.TrimSpace(data.Source) == "" || data.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: stop list id, restaurant_id, catalog_item_id, source and updated_at are required", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(data.AvailableQuantity) != "" && !nonNegativeDecimal(data.AvailableQuantity) {
		return fmt.Errorf("%w: stop list available_quantity is invalid", ErrInvalidEnvelope)
	}
	if NormalizeStopListConflictPolicy(data.ConflictPolicy) == "" {
		return fmt.Errorf("%w: stop list conflict_policy is invalid", ErrInvalidEnvelope)
	}
	return nil
}

func validateCatalogItemChangeSuggestedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[CatalogItemChangeSuggested](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.SuggestionID) == "" || strings.TrimSpace(data.RestaurantID) == "" || strings.TrimSpace(data.Action) == "" || data.SuggestedAt.IsZero() {
		return fmt.Errorf("%w: catalog suggestion id, restaurant_id, action and suggested_at are required", ErrInvalidEnvelope)
	}
	return nil
}

func validateRecipeChangeSuggestedPayload(v SyncEnvelope) error {
	payload, err := decodePayload[RecipeChangeSuggested](v)
	if err != nil {
		return err
	}
	data := payload.Data
	if strings.TrimSpace(data.SuggestionID) == "" || strings.TrimSpace(data.RestaurantID) == "" || strings.TrimSpace(data.Action) == "" || data.SuggestedAt.IsZero() {
		return fmt.Errorf("%w: recipe suggestion id, restaurant_id, action and suggested_at are required", ErrInvalidEnvelope)
	}
	return nil
}

func decodePayload[T any](v SyncEnvelope) (Payload[T], error) {
	var payload Payload[T]
	if err := json.Unmarshal(v.Payload, &payload); err != nil {
		return payload, fmt.Errorf("%w: invalid %s payload: %v", ErrInvalidEnvelope, v.EventType, err)
	}
	if strings.TrimSpace(payload.Origin) == "" {
		return payload, fmt.Errorf("%w: payload.origin is required", ErrInvalidEnvelope)
	}
	return payload, nil
}

func validateInventoryItems(items []InventoryItem, useCountedQuantity bool) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: inventory items are required", ErrInvalidEnvelope)
	}
	for _, item := range items {
		if strings.TrimSpace(item.CatalogItemID) == "" || strings.TrimSpace(item.UnitCode) == "" {
			return fmt.Errorf("%w: inventory item catalog_item_id and unit_code are required", ErrInvalidEnvelope)
		}
		quantity := item.Quantity
		if useCountedQuantity {
			quantity = item.CountedQuantity
		}
		if !positiveDecimal(quantity) {
			return fmt.Errorf("%w: inventory item quantity must be positive", ErrInvalidEnvelope)
		}
		if item.UnitCostMinor < 0 {
			return fmt.Errorf("%w: inventory item unit_cost_minor must be non-negative", ErrInvalidEnvelope)
		}
		if strings.TrimSpace(item.Currency) != "" && !validCurrency(item.Currency) {
			return fmt.Errorf("%w: inventory item currency is invalid", ErrInvalidEnvelope)
		}
	}
	return nil
}

func validateBusinessDate(value, name string) error {
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(value)); err != nil {
		return fmt.Errorf("%w: %s must use YYYY-MM-DD", ErrInvalidEnvelope, name)
	}
	return nil
}

func positiveDecimal(value string) bool {
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && n > 0
}

func positiveJSONNumber(value json.RawMessage) bool {
	var raw any
	if err := json.Unmarshal(value, &raw); err != nil {
		return false
	}
	switch v := raw.(type) {
	case float64:
		return v > 0
	case string:
		return positiveDecimal(v)
	default:
		return false
	}
}

func nonNegativeDecimal(value string) bool {
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && n >= 0
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
	case EventShiftOpened, EventShiftClosed, EventOrderCreated, EventOrderLineAdded, EventOrderLineQuantityChanged, EventOrderLineVoided, EventPrecheckIssued, EventPrecheckReprinted, EventPrecheckCancelled, EventCheckCreated, EventCheckRefunded, EventCheckReprinted, EventPaymentCaptured, EventPaymentRefunded, EventCancellationRecorded, EventRefundRecorded, EventCheckClosed, EventKitchenTicketStatusChanged, EventItemServed, EventStockReceiptCaptured, EventInventoryCountCaptured, EventStockWriteOffCaptured, EventProductionCompleted, EventStopListUpdated, EventCatalogItemChangeSuggested, EventRecipeChangeSuggested, EventOrderClosed, EventCashSessionOpened, EventCashSessionClosed, EventCashDrawerEventRecorded, EventAuthSessionStarted, EventAuthSessionRevoked, EventDeviceRegistered:
		return true
	default:
		return false
	}
}

func IsInventoryRelevantEventType(v EventType) bool {
	switch v {
	case EventCheckClosed, EventItemServed, EventStockReceiptCaptured, EventInventoryCountCaptured, EventStockWriteOffCaptured, EventProductionCompleted, EventStopListUpdated, EventRefundRecorded, EventCancellationRecorded:
		return true
	default:
		return false
	}
}

func ShouldEnqueueInventoryEvent(v EventType, payloadRaw json.RawMessage) bool {
	if !IsInventoryRelevantEventType(v) {
		return false
	}
	switch v {
	case EventRefundRecorded, EventCancellationRecorded:
		var payload Payload[FinancialOperationRecorded]
		if err := json.Unmarshal(payloadRaw, &payload); err != nil {
			return false
		}
		return strings.TrimSpace(payload.Data.InventoryDisposition) != "no_stock_effect"
	default:
		return true
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
