package financial

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/pos/domain/shared"
)

type OperationType string
type OperationKind string
type OperationStatus string
type InventoryDisposition string
type OperationItemScope string

const (
	OperationCancellation OperationType = "cancellation"
	OperationRefund       OperationType = "refund"

	OperationFull    OperationKind = "full"
	OperationPartial OperationKind = "partial"

	OperationRecorded OperationStatus = "recorded"

	InventoryNoStockEffect InventoryDisposition = "no_stock_effect"
	InventoryReturnToStock InventoryDisposition = "return_to_stock"
	InventoryWriteOffWaste InventoryDisposition = "write_off_waste"
	InventoryManualReview  InventoryDisposition = "manual_review"

	ItemWholeCheck    OperationItemScope = "whole_check"
	ItemOrderLine     OperationItemScope = "order_line"
	ItemModifierLine  OperationItemScope = "modifier_line"
	ItemServiceCharge OperationItemScope = "service_charge"
	ItemTip           OperationItemScope = "tip"
	ItemPayment       OperationItemScope = "payment"
)

// Operation фиксирует append-only compensating financial operation без мутации исходного чека или платежа.
type Operation struct {
	ID                   string               `json:"id"`
	EdgeOperationID      string               `json:"edge_operation_id"`
	RestaurantID         string               `json:"restaurant_id"`
	DeviceID             string               `json:"device_id"`
	ShiftID              string               `json:"shift_id"`
	OriginalShiftID      string               `json:"original_shift_id"`
	CheckID              string               `json:"check_id"`
	PrecheckID           string               `json:"precheck_id"`
	Type                 OperationType        `json:"operation_type"`
	Kind                 OperationKind        `json:"operation_kind"`
	Status               OperationStatus      `json:"status"`
	Amount               int64                `json:"amount"`
	Currency             string               `json:"currency"`
	BusinessDateLocal    string               `json:"business_date_local"`
	InventoryDisposition InventoryDisposition `json:"inventory_disposition"`
	Reason               string               `json:"reason"`
	CreatedByEmployeeID  string               `json:"created_by_employee_id"`
	ApprovedByEmployeeID *string              `json:"approved_by_employee_id,omitempty"`
	Snapshot             json.RawMessage      `json:"snapshot,omitempty"`
	Items                []OperationItem      `json:"items,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
}

// OperationListQuery задает bounded read model для append-only ledger reporting.
type OperationListQuery struct {
	RestaurantID       string
	CheckID            string
	BusinessDateFrom   string
	BusinessDateTo     string
	OperationType      OperationType
	ShiftID            string
	OriginalShiftID    string
	Limit              int
	Offset             int
}

// OperationItem описывает часть операции: весь чек, строку, количество строки, modifier/service/tip или payment allocation.
type OperationItem struct {
	ID          string             `json:"id"`
	OperationID string             `json:"operation_id"`
	Scope       OperationItemScope `json:"scope"`
	OrderLineID *string            `json:"order_line_id,omitempty"`
	PaymentID   *string            `json:"payment_id,omitempty"`
	Quantity    *int64             `json:"quantity,omitempty"`
	Amount      int64              `json:"amount"`
	Currency    string             `json:"currency"`
	TaxAmount   int64              `json:"tax_amount"`
	Snapshot    json.RawMessage    `json:"snapshot,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

func (v Operation) Validate() error {
	if strings.TrimSpace(v.ID) == "" || strings.TrimSpace(v.EdgeOperationID) == "" || strings.TrimSpace(v.RestaurantID) == "" || strings.TrimSpace(v.DeviceID) == "" || strings.TrimSpace(v.ShiftID) == "" || strings.TrimSpace(v.OriginalShiftID) == "" || strings.TrimSpace(v.CheckID) == "" || strings.TrimSpace(v.PrecheckID) == "" {
		return fmt.Errorf("%w: financial operation identity is required", shared.ErrInvalid)
	}
	if !ValidOperationType(v.Type) || !ValidOperationKind(v.Kind) || v.Status != OperationRecorded {
		return fmt.Errorf("%w: unsupported financial operation state", shared.ErrInvalid)
	}
	if v.Amount <= 0 || strings.TrimSpace(v.Currency) == "" {
		return fmt.Errorf("%w: positive amount and currency are required", shared.ErrInvalid)
	}
	if !ValidInventoryDisposition(v.InventoryDisposition) {
		return fmt.Errorf("%w: unsupported inventory disposition", shared.ErrInvalid)
	}
	if strings.TrimSpace(v.BusinessDateLocal) == "" || strings.TrimSpace(v.Reason) == "" || strings.TrimSpace(v.CreatedByEmployeeID) == "" || v.CreatedAt.IsZero() {
		return fmt.Errorf("%w: business date, reason, actor and created_at are required", shared.ErrInvalid)
	}
	if len(v.Items) == 0 {
		return fmt.Errorf("%w: financial operation requires at least one item", shared.ErrInvalid)
	}
	return nil
}

func (v OperationItem) Validate(operationID, currency string) error {
	if strings.TrimSpace(v.ID) == "" || strings.TrimSpace(v.OperationID) == "" || strings.TrimSpace(operationID) != strings.TrimSpace(v.OperationID) {
		return fmt.Errorf("%w: operation item identity is required", shared.ErrInvalid)
	}
	if !ValidOperationItemScope(v.Scope) {
		return fmt.Errorf("%w: unsupported operation item scope", shared.ErrInvalid)
	}
	if v.Amount <= 0 || strings.ToUpper(strings.TrimSpace(v.Currency)) != strings.ToUpper(strings.TrimSpace(currency)) {
		return fmt.Errorf("%w: operation item amount/currency is invalid", shared.ErrInvalid)
	}
	if v.TaxAmount < 0 {
		return fmt.Errorf("%w: operation item tax_amount must be non-negative", shared.ErrInvalid)
	}
	if v.Scope == ItemOrderLine && (v.OrderLineID == nil || strings.TrimSpace(*v.OrderLineID) == "") {
		return fmt.Errorf("%w: order_line item requires order_line_id", shared.ErrInvalid)
	}
	if v.Scope == ItemPayment && (v.PaymentID == nil || strings.TrimSpace(*v.PaymentID) == "") {
		return fmt.Errorf("%w: payment item requires payment_id", shared.ErrInvalid)
	}
	if v.Quantity != nil && *v.Quantity <= 0 {
		return fmt.Errorf("%w: operation item quantity must be positive", shared.ErrInvalid)
	}
	return nil
}

func ValidOperationType(v OperationType) bool {
	return v == OperationCancellation || v == OperationRefund
}

func ValidOperationKind(v OperationKind) bool {
	return v == OperationFull || v == OperationPartial
}

func ValidInventoryDisposition(v InventoryDisposition) bool {
	switch v {
	case InventoryNoStockEffect, InventoryReturnToStock, InventoryWriteOffWaste, InventoryManualReview:
		return true
	default:
		return false
	}
}

func ValidOperationItemScope(v OperationItemScope) bool {
	switch v {
	case ItemWholeCheck, ItemOrderLine, ItemModifierLine, ItemServiceCharge, ItemTip, ItemPayment:
		return true
	default:
		return false
	}
}
