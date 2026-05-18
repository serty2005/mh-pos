package check

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
)

type FinancialOperationItemCommand struct {
	Scope       domain.FinancialOperationItemScope `json:"scope"`
	OrderLineID string                             `json:"order_line_id,omitempty"`
	PaymentID   string                             `json:"payment_id,omitempty"`
	Quantity    int64                              `json:"quantity,omitempty"`
	Amount      int64                              `json:"amount"`
	Currency    string                             `json:"currency,omitempty"`
	TaxAmount   int64                              `json:"tax_amount,omitempty"`
	Snapshot    json.RawMessage                    `json:"snapshot,omitempty"`
}

type RecordCheckCancellationCommand struct {
	shared.CommandMeta
	CheckID              string                          `json:"check_id"`
	OperationKind        domain.FinancialOperationKind   `json:"operation_kind,omitempty"`
	InventoryDisposition domain.InventoryDisposition     `json:"inventory_disposition,omitempty"`
	Reason               string                          `json:"reason"`
	ApprovedByEmployeeID string                          `json:"approved_by_employee_id,omitempty"`
	Items                []FinancialOperationItemCommand `json:"items,omitempty"`
}

type RecordCheckRefundCommand struct {
	shared.CommandMeta
	CheckID              string                          `json:"check_id"`
	OperationKind        domain.FinancialOperationKind   `json:"operation_kind,omitempty"`
	InventoryDisposition domain.InventoryDisposition     `json:"inventory_disposition,omitempty"`
	Reason               string                          `json:"reason"`
	ApprovedByEmployeeID string                          `json:"approved_by_employee_id,omitempty"`
	Items                []FinancialOperationItemCommand `json:"items,omitempty"`
}

type financialOperationCommand struct {
	shared.CommandMeta
	CheckID              string
	OperationKind        domain.FinancialOperationKind
	InventoryDisposition domain.InventoryDisposition
	Reason               string
	ApprovedByEmployeeID string
	Items                []FinancialOperationItemCommand
}

func (s *Service) RecordCancellation(ctx context.Context, cmd RecordCheckCancellationCommand) (*domain.FinancialOperation, error) {
	return s.recordFinancialOperation(ctx, domain.FinancialOperationCancellation, financialOperationCommand{
		CommandMeta:          cmd.CommandMeta,
		CheckID:              cmd.CheckID,
		OperationKind:        cmd.OperationKind,
		InventoryDisposition: cmd.InventoryDisposition,
		Reason:               cmd.Reason,
		ApprovedByEmployeeID: cmd.ApprovedByEmployeeID,
		Items:                cmd.Items,
	})
}

func (s *Service) RecordRefund(ctx context.Context, cmd RecordCheckRefundCommand) (*domain.FinancialOperation, error) {
	return s.recordFinancialOperation(ctx, domain.FinancialOperationRefund, financialOperationCommand{
		CommandMeta:          cmd.CommandMeta,
		CheckID:              cmd.CheckID,
		OperationKind:        cmd.OperationKind,
		InventoryDisposition: cmd.InventoryDisposition,
		Reason:               cmd.Reason,
		ApprovedByEmployeeID: cmd.ApprovedByEmployeeID,
		Items:                cmd.Items,
	})
}

func (s *Service) recordFinancialOperation(ctx context.Context, typ domain.FinancialOperationType, cmd financialOperationCommand) (*domain.FinancialOperation, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.CheckID) == "" || strings.TrimSpace(cmd.Reason) == "" {
		return nil, fmt.Errorf("%w: check_id and reason are required", domain.ErrInvalid)
	}
	if cmd.InventoryDisposition == "" {
		cmd.InventoryDisposition = domain.InventoryNoStockEffect
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var operation *domain.FinancialOperation
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(requiredFinancialOperationPermission(typ)))
		if err != nil {
			return err
		}
		check, err := s.repo.GetCheck(ctx, cmd.CheckID)
		if err != nil {
			return err
		}
		if check.Status != domain.CheckPaid && check.Status != domain.CheckRefunded && check.Status != domain.CheckVoided {
			return fmt.Errorf("%w: financial operation requires finalized check", domain.ErrConflict)
		}
		order, err := s.repo.GetOrder(ctx, check.OrderID)
		if err != nil {
			return err
		}
		if order.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: check device does not match command device", domain.ErrConflict)
		}
		cashSession, err := s.repo.GetOpenCashSessionByDevice(ctx, cmd.DeviceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: financial operation requires open cash session", domain.ErrConflict)
			}
			return err
		}
		if cashSession.RestaurantID != order.RestaurantID {
			return fmt.Errorf("%w: cash session restaurant does not match check restaurant", domain.ErrConflict)
		}
		originalShift, err := s.repo.GetShift(ctx, order.ShiftID)
		if err != nil {
			return err
		}
		restaurant, err := s.repo.GetRestaurant(ctx, order.RestaurantID)
		if err != nil {
			return err
		}
		businessDate, err := shared.BusinessDateLocal(*restaurant, now)
		if err != nil {
			return err
		}
		if err := ensureFinancialBoundary(typ, originalShift, cashSession, check.BusinessDateLocal, businessDate); err != nil {
			return err
		}
		precheckID, err := precheckIDForCheck(check)
		if err != nil {
			return err
		}
		if precheckID == "" {
			precheckID, err = s.latestPrecheckIDForOrder(ctx, order.ID)
			if err != nil {
				return err
			}
		}
		refunded, err := s.repo.SumFinancialOperationAmountByCheck(ctx, check.ID, domain.FinancialOperationRefund)
		if err != nil {
			return err
		}
		cancelled, err := s.repo.SumFinancialOperationAmountByCheck(ctx, check.ID, domain.FinancialOperationCancellation)
		if err != nil {
			return err
		}
		remainingCompensable := check.Total - refunded - cancelled
		if remainingCompensable <= 0 {
			return fmt.Errorf("%w: check already has full compensating operations", domain.ErrConflict)
		}
		operationID := s.ids.NewID()
		items, amount, err := s.buildFinancialOperationItems(ctx, operationID, typ, cmd.Items, check, order, precheckID, remainingCompensable, now)
		if err != nil {
			return err
		}
		if amount > remainingCompensable {
			return fmt.Errorf("%w: financial operation exceeds remaining check amount", domain.ErrConflict)
		}
		kind, err := normalizeOperationKind(cmd.OperationKind, amount, remainingCompensable)
		if err != nil {
			return err
		}
		approvedBy, err := s.approvedByEmployee(ctx, cmd.ApprovedByEmployeeID, operator.Employee.ID, order.RestaurantID)
		if err != nil {
			return err
		}
		operation = &domain.FinancialOperation{
			ID:                   operationID,
			EdgeOperationID:      s.ids.NewID(),
			RestaurantID:         order.RestaurantID,
			DeviceID:             order.DeviceID,
			ShiftID:              cashSession.ShiftID,
			OriginalShiftID:      order.ShiftID,
			CheckID:              check.ID,
			PrecheckID:           precheckID,
			Type:                 typ,
			Kind:                 kind,
			Status:               domain.FinancialOperationRecorded,
			Amount:               amount,
			Currency:             check.CurrencyCode,
			BusinessDateLocal:    businessDate,
			InventoryDisposition: cmd.InventoryDisposition,
			Reason:               strings.TrimSpace(cmd.Reason),
			CreatedByEmployeeID:  operator.Employee.ID,
			ApprovedByEmployeeID: approvedBy,
			Items:                items,
			CreatedAt:            now,
		}
		snapshot, err := buildFinancialOperationSnapshot(operation, check, items, now)
		if err != nil {
			return err
		}
		operation.Snapshot = snapshot
		if err := operation.Validate(); err != nil {
			return err
		}
		for i := range operation.Items {
			if err := operation.Items[i].Validate(operation.ID, operation.Currency); err != nil {
				return err
			}
		}
		if err := s.repo.CreateFinancialOperation(ctx, operation); err != nil {
			return err
		}
		for i := range operation.Items {
			if err := s.repo.CreateFinancialOperationItem(ctx, &operation.Items[i]); err != nil {
				return err
			}
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, cashSession.ShiftID, "FinancialOperation", operation.ID, financialOperationEventType(typ), operation)
	})
	return operation, err
}

func (s *Service) buildFinancialOperationItems(ctx context.Context, operationID string, typ domain.FinancialOperationType, inputs []FinancialOperationItemCommand, check *domain.Check, order *domain.Order, precheckID string, defaultAmount int64, now time.Time) ([]domain.FinancialOperationItem, int64, error) {
	if len(inputs) == 0 {
		inputs = []FinancialOperationItemCommand{{
			Scope:    domain.FinancialItemWholeCheck,
			Amount:   defaultAmount,
			Currency: check.CurrencyCode,
			Snapshot: check.Snapshot,
		}}
	}
	items := make([]domain.FinancialOperationItem, 0, len(inputs))
	var total int64
	for _, input := range inputs {
		scope := input.Scope
		if scope == "" {
			scope = domain.FinancialItemWholeCheck
		}
		currency := strings.ToUpper(strings.TrimSpace(input.Currency))
		if currency == "" {
			currency = check.CurrencyCode
		}
		item := domain.FinancialOperationItem{
			ID:          s.ids.NewID(),
			OperationID: operationID,
			Scope:       scope,
			Amount:      input.Amount,
			Currency:    currency,
			TaxAmount:   input.TaxAmount,
			CreatedAt:   now,
		}
		if input.Quantity > 0 {
			item.Quantity = &input.Quantity
		}
		if value := strings.TrimSpace(input.OrderLineID); value != "" {
			item.OrderLineID = &value
		}
		if value := strings.TrimSpace(input.PaymentID); value != "" {
			item.PaymentID = &value
		}
		snapshot, err := s.operationItemSnapshot(ctx, typ, item, input.Snapshot, check, order, precheckID)
		if err != nil {
			return nil, 0, err
		}
		item.Snapshot = snapshot
		if err := item.Validate(operationID, check.CurrencyCode); err != nil {
			return nil, 0, err
		}
		total += item.Amount
		if total < 0 {
			return nil, 0, fmt.Errorf("%w: financial operation total overflow", domain.ErrInvalid)
		}
		items = append(items, item)
	}
	if total <= 0 {
		return nil, 0, fmt.Errorf("%w: financial operation amount must be positive", domain.ErrInvalid)
	}
	return items, total, nil
}

func (s *Service) operationItemSnapshot(ctx context.Context, typ domain.FinancialOperationType, item domain.FinancialOperationItem, raw json.RawMessage, check *domain.Check, order *domain.Order, precheckID string) (json.RawMessage, error) {
	if len(raw) > 0 {
		if !json.Valid(raw) {
			return nil, fmt.Errorf("%w: operation item snapshot must be valid json", domain.ErrInvalid)
		}
		return raw, nil
	}
	switch item.Scope {
	case domain.FinancialItemWholeCheck:
		return check.Snapshot, nil
	case domain.FinancialItemOrderLine:
		line, err := s.repo.GetOrderLine(ctx, deref(item.OrderLineID))
		if err != nil {
			return nil, err
		}
		if line.OrderID != order.ID {
			return nil, fmt.Errorf("%w: operation line does not belong to check order", domain.ErrConflict)
		}
		maxAmount := maxLineItemAmount(line, item.Quantity)
		if item.Amount > maxAmount {
			return nil, fmt.Errorf("%w: operation line amount exceeds selected line amount", domain.ErrConflict)
		}
		alreadyAmount, err := s.sumFinancialOperationAmountByOrderLine(ctx, line.ID)
		if err != nil {
			return nil, err
		}
		if alreadyAmount+item.Amount > line.TotalPrice {
			return nil, fmt.Errorf("%w: operation line amount exceeds remaining line amount", domain.ErrConflict)
		}
		if item.Quantity != nil && *item.Quantity > line.Quantity {
			return nil, fmt.Errorf("%w: operation line quantity exceeds original line quantity", domain.ErrConflict)
		}
		if item.Quantity != nil {
			already, err := s.sumFinancialOperationQuantityByOrderLine(ctx, line.ID)
			if err != nil {
				return nil, err
			}
			if already+*item.Quantity > line.Quantity {
				return nil, fmt.Errorf("%w: operation line quantity exceeds remaining line quantity", domain.ErrConflict)
			}
		}
		return json.Marshal(line)
	case domain.FinancialItemPayment:
		payment, err := s.repo.GetPayment(ctx, deref(item.PaymentID))
		if err != nil {
			return nil, err
		}
		if payment.PrecheckID != precheckID || payment.Status != domain.PaymentCaptured || payment.Currency != check.CurrencyCode {
			return nil, fmt.Errorf("%w: operation payment allocation is not captured for check", domain.ErrConflict)
		}
		already, err := s.sumFinancialOperationAmountByPayment(ctx, payment.ID)
		if err != nil {
			return nil, err
		}
		if already+item.Amount > payment.Amount {
			return nil, fmt.Errorf("%w: operation exceeds payment captured amount", domain.ErrConflict)
		}
		return json.Marshal(payment)
	case domain.FinancialItemModifierLine, domain.FinancialItemServiceCharge, domain.FinancialItemTip:
		return nil, fmt.Errorf("%w: explicit snapshot is required for %s operation item", domain.ErrInvalid, item.Scope)
	default:
		return nil, fmt.Errorf("%w: unsupported operation item scope", domain.ErrInvalid)
	}
}

func (s *Service) sumFinancialOperationAmountByPayment(ctx context.Context, paymentID string) (int64, error) {
	refunded, err := s.repo.SumFinancialOperationAmountByPayment(ctx, paymentID, domain.FinancialOperationRefund)
	if err != nil {
		return 0, err
	}
	cancelled, err := s.repo.SumFinancialOperationAmountByPayment(ctx, paymentID, domain.FinancialOperationCancellation)
	if err != nil {
		return 0, err
	}
	return refunded + cancelled, nil
}

func (s *Service) sumFinancialOperationAmountByOrderLine(ctx context.Context, orderLineID string) (int64, error) {
	refunded, err := s.repo.SumFinancialOperationAmountByOrderLine(ctx, orderLineID, domain.FinancialOperationRefund)
	if err != nil {
		return 0, err
	}
	cancelled, err := s.repo.SumFinancialOperationAmountByOrderLine(ctx, orderLineID, domain.FinancialOperationCancellation)
	if err != nil {
		return 0, err
	}
	return refunded + cancelled, nil
}

func (s *Service) sumFinancialOperationQuantityByOrderLine(ctx context.Context, orderLineID string) (int64, error) {
	refunded, err := s.repo.SumFinancialOperationQuantityByOrderLine(ctx, orderLineID, domain.FinancialOperationRefund)
	if err != nil {
		return 0, err
	}
	cancelled, err := s.repo.SumFinancialOperationQuantityByOrderLine(ctx, orderLineID, domain.FinancialOperationCancellation)
	if err != nil {
		return 0, err
	}
	return refunded + cancelled, nil
}

func maxLineItemAmount(line *domain.OrderLine, quantity *int64) int64 {
	if quantity == nil || line.Quantity <= 0 || *quantity >= line.Quantity {
		return line.TotalPrice
	}
	return (line.TotalPrice*(*quantity) + line.Quantity/2) / line.Quantity
}

func ensureFinancialBoundary(typ domain.FinancialOperationType, originalShift *domain.Shift, cashSession *domain.CashSession, checkBusinessDate, currentBusinessDate string) error {
	switch typ {
	case domain.FinancialOperationCancellation:
		if originalShift.Status != domain.ShiftOpen || cashSession.ShiftID != originalShift.ID || checkBusinessDate != currentBusinessDate {
			return fmt.Errorf("%w: cancellation requires original open shift and same business day", domain.ErrConflict)
		}
	case domain.FinancialOperationRefund:
		if originalShift.Status != domain.ShiftClosed && checkBusinessDate == currentBusinessDate {
			return fmt.Errorf("%w: refund requires closed original shift or later business day", domain.ErrConflict)
		}
	default:
		return fmt.Errorf("%w: unsupported financial operation type", domain.ErrInvalid)
	}
	return nil
}

func normalizeOperationKind(kind domain.FinancialOperationKind, amount, remaining int64) (domain.FinancialOperationKind, error) {
	if kind == "" {
		if amount == remaining {
			return domain.FinancialOperationFull, nil
		}
		return domain.FinancialOperationPartial, nil
	}
	switch kind {
	case domain.FinancialOperationFull:
		if amount != remaining {
			return "", fmt.Errorf("%w: full operation must cover remaining check amount", domain.ErrInvalid)
		}
	case domain.FinancialOperationPartial:
		if amount >= remaining {
			return "", fmt.Errorf("%w: partial operation must be below remaining check amount", domain.ErrInvalid)
		}
	default:
		return "", fmt.Errorf("%w: unsupported operation kind", domain.ErrInvalid)
	}
	return kind, nil
}

func (s *Service) approvedByEmployee(ctx context.Context, explicit, fallback, restaurantID string) (*string, error) {
	id := strings.TrimSpace(explicit)
	if id == "" {
		id = strings.TrimSpace(fallback)
	}
	if id == "" {
		return nil, nil
	}
	employee, err := s.repo.GetEmployee(ctx, id)
	if err != nil {
		return nil, err
	}
	if !employee.Active || employee.RestaurantID != restaurantID {
		return nil, fmt.Errorf("%w: approved_by employee is not active for restaurant", domain.ErrForbidden)
	}
	return &id, nil
}

func (s *Service) latestPrecheckIDForOrder(ctx context.Context, orderID string) (string, error) {
	prechecks, err := s.repo.ListPrechecksByOrder(ctx, orderID)
	if err != nil {
		return "", err
	}
	if len(prechecks) == 0 {
		return "", fmt.Errorf("%w: check precheck is not available", domain.ErrConflict)
	}
	return prechecks[len(prechecks)-1].ID, nil
}

func precheckIDForCheck(check *domain.Check) (string, error) {
	if len(check.Snapshot) == 0 {
		return "", nil
	}
	var body struct {
		PrecheckID string `json:"precheck_id"`
	}
	if err := json.Unmarshal(check.Snapshot, &body); err != nil {
		return "", fmt.Errorf("%w: check snapshot cannot be parsed", domain.ErrConflict)
	}
	return strings.TrimSpace(body.PrecheckID), nil
}

func buildFinancialOperationSnapshot(operation *domain.FinancialOperation, check *domain.Check, items []domain.FinancialOperationItem, now time.Time) (json.RawMessage, error) {
	body, err := json.Marshal(struct {
		DocumentType      string                          `json:"document_type"`
		OperationID       string                          `json:"operation_id"`
		OperationType     domain.FinancialOperationType   `json:"operation_type"`
		OperationKind     domain.FinancialOperationKind   `json:"operation_kind"`
		CheckID           string                          `json:"check_id"`
		Amount            int64                           `json:"amount"`
		Currency          string                          `json:"currency"`
		BusinessDateLocal string                          `json:"business_date_local"`
		CreatedAt         string                          `json:"created_at"`
		CheckSnapshot     json.RawMessage                 `json:"check_snapshot"`
		Items             []domain.FinancialOperationItem `json:"items"`
	}{
		DocumentType:      "financial_operation",
		OperationID:       operation.ID,
		OperationType:     operation.Type,
		OperationKind:     operation.Kind,
		CheckID:           operation.CheckID,
		Amount:            operation.Amount,
		Currency:          operation.Currency,
		BusinessDateLocal: operation.BusinessDateLocal,
		CreatedAt:         shared.DBTime(now),
		CheckSnapshot:     check.Snapshot,
		Items:             items,
	})
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func requiredFinancialOperationPermission(typ domain.FinancialOperationType) shared.PermissionID {
	if typ == domain.FinancialOperationCancellation {
		return shared.PermissionPrecheckCancel
	}
	return shared.PermissionPaymentRefund
}

func financialOperationEventType(typ domain.FinancialOperationType) string {
	if typ == domain.FinancialOperationCancellation {
		return "CancellationRecorded"
	}
	return "RefundRecorded"
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
