package check

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
)

// Значения зеркалят CHECK (status IN ('active','voided')) на ticket_units — статус
// хранится как plain string в domain.TicketUnit, отдельного enum-алиаса на уровне
// domain package нет.
const (
	ticketStatusActive = "active"
	ticketStatusVoided = "voided"
)

// CancelUnconfirmedOrderCommand отменяет заказ, чек которого не получил
// подтверждения печати (check.PrintConfirmedAt == nil). Требует manager PIN —
// это не рутинный кассирский флоу, а осознанное решение отказаться от продажи,
// потому что чек/билет не был напечатан после нескольких попыток.
type CancelUnconfirmedOrderCommand struct {
	shared.CommandMeta
	OrderID    string `json:"order_id"`
	ManagerPIN string `json:"manager_pin"`
	Reason     string `json:"reason"`
}

// CancelUnconfirmedOrder транзакционно: пишет полноценную компенсирующую финансовую
// операцию (cancellation — та же cash-session shift и business date, что закрывали
// чек; refund в терминах системы предназначен для другого business date и здесь
// неприменим по ensureFinancialBoundary), void все выпущенные билеты чека и
// soft-cancel сам заказ. Каждое из трёх действий пишет отдельное outbox-событие для
// истории/аналитики — они не переиспользуют существующие Payment/Check события.
// Оплата/check/ticket issuance при этом не переигрываются: это компенсация уже
// свершившегося факта, а не откат транзакции.
func (s *Service) CancelUnconfirmedOrder(ctx context.Context, cmd CancelUnconfirmedOrderCommand) (*domain.Order, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" || strings.TrimSpace(cmd.ManagerPIN) == "" || strings.TrimSpace(cmd.Reason) == "" {
		return nil, fmt.Errorf("%w: order_id, manager_pin and reason are required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var result *domain.Order
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionCheckView))
		if err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, strings.TrimSpace(cmd.OrderID))
		if err != nil {
			return err
		}
		if order.RestaurantID != operator.Employee.RestaurantID {
			return fmt.Errorf("%w: order is outside operator restaurant", domain.ErrForbidden)
		}
		if order.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: order device does not match command device", domain.ErrConflict)
		}
		if order.Status == domain.OrderCancelled {
			return fmt.Errorf("%w: order is already cancelled", domain.ErrConflict)
		}
		checkRow, err := s.repo.GetCheckByOrder(ctx, order.ID)
		if err != nil {
			return err
		}
		if checkRow.PrintConfirmedAt != nil {
			return fmt.Errorf("%w: check print is already confirmed, cancel-unconfirmed is not applicable", domain.ErrConflict)
		}
		manager, err := s.resolveCancelUnconfirmedApprover(ctx, order.RestaurantID, cmd.ManagerPIN)
		if err != nil {
			return err
		}
		cashSession, err := s.repo.GetOpenCashSessionByDevice(ctx, cmd.DeviceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("%w: cancel-unconfirmed requires an open cash session", domain.ErrConflict)
			}
			return err
		}
		if cashSession.RestaurantID != order.RestaurantID {
			return fmt.Errorf("%w: cash session restaurant does not match order restaurant", domain.ErrConflict)
		}
		restaurant, err := s.repo.GetRestaurant(ctx, order.RestaurantID)
		if err != nil {
			return err
		}
		businessDate, err := shared.BusinessDateLocal(*restaurant, now)
		if err != nil {
			return err
		}
		cashShift, err := s.repo.GetShift(ctx, cashSession.ShiftID)
		if err != nil {
			return err
		}
		if err := ensureFinancialBoundary(domain.FinancialOperationCancellation, cashShift, cashSession, checkRow.BusinessDateLocal, businessDate); err != nil {
			return err
		}
		precheckID, err := precheckIDForCheck(checkRow)
		if err != nil {
			return err
		}
		if precheckID == "" {
			precheckID, err = s.latestPrecheckIDForOrder(ctx, order.ID)
			if err != nil {
				return err
			}
		}
		refunded, err := s.repo.SumFinancialOperationAmountByCheck(ctx, checkRow.ID, domain.FinancialOperationRefund)
		if err != nil {
			return err
		}
		cancelled, err := s.repo.SumFinancialOperationAmountByCheck(ctx, checkRow.ID, domain.FinancialOperationCancellation)
		if err != nil {
			return err
		}
		remaining := checkRow.Total - refunded - cancelled
		if remaining <= 0 {
			return fmt.Errorf("%w: check already has full compensating operations", domain.ErrConflict)
		}
		operationID := s.ids.NewID()
		item := domain.FinancialOperationItem{
			ID:          s.ids.NewID(),
			OperationID: operationID,
			Scope:       domain.FinancialItemWholeCheck,
			Amount:      remaining,
			Currency:    checkRow.CurrencyCode,
			CreatedAt:   now,
		}
		snapshot, err := s.operationItemSnapshot(ctx, domain.FinancialOperationCancellation, item, nil, checkRow, order, precheckID)
		if err != nil {
			return err
		}
		item.Snapshot = snapshot
		if err := item.Validate(operationID, checkRow.CurrencyCode); err != nil {
			return err
		}
		approvedBy, err := s.approvedByEmployee(ctx, manager.ID, manager.ID, order.RestaurantID)
		if err != nil {
			return err
		}
		operation := &domain.FinancialOperation{
			ID:                   operationID,
			EdgeOperationID:      s.ids.NewID(),
			RestaurantID:         order.RestaurantID,
			DeviceID:             order.DeviceID,
			ShiftID:              cashSession.ShiftID,
			OriginalShiftID:      order.ShiftID,
			CheckID:              checkRow.ID,
			PrecheckID:           precheckID,
			Type:                 domain.FinancialOperationCancellation,
			Kind:                 domain.FinancialOperationFull,
			Status:               domain.FinancialOperationRecorded,
			Amount:               remaining,
			Currency:             checkRow.CurrencyCode,
			BusinessDateLocal:    businessDate,
			InventoryDisposition: domain.InventoryNoStockEffect,
			Reason:               strings.TrimSpace(cmd.Reason),
			CreatedByEmployeeID:  operator.Employee.ID,
			ApprovedByEmployeeID: approvedBy,
			Items:                []domain.FinancialOperationItem{item},
			CreatedAt:            now,
		}
		opSnapshot, err := buildFinancialOperationSnapshot(operation, checkRow, operation.Items, now)
		if err != nil {
			return err
		}
		operation.Snapshot = opSnapshot
		if err := operation.Validate(); err != nil {
			return err
		}
		if err := s.repo.CreateFinancialOperation(ctx, operation); err != nil {
			return err
		}
		if err := s.repo.CreateFinancialOperationItem(ctx, &operation.Items[0]); err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, cashSession.ShiftID, "FinancialOperation", operation.ID, "CheckPrintUnconfirmedRefunded", operation); err != nil {
			return err
		}

		tickets, err := s.repo.ListTicketUnitsByCheck(ctx, checkRow.ID)
		if err != nil {
			return err
		}
		activeTicketIDs := make([]string, 0, len(tickets))
		for i := range tickets {
			if tickets[i].Status == ticketStatusActive {
				activeTicketIDs = append(activeTicketIDs, tickets[i].ID)
			}
		}
		if len(activeTicketIDs) > 0 {
			if err := s.repo.VoidTicketUnitsByCheck(ctx, checkRow.ID, now); err != nil {
				return err
			}
			for i := range tickets {
				if tickets[i].Status != ticketStatusActive {
					continue
				}
				voided := tickets[i]
				voided.Status = ticketStatusVoided
				voided.UpdatedAt = now
				if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, cashSession.ShiftID, "TicketUnit", voided.ID, "TicketVoided", voided); err != nil {
					return err
				}
			}
		}

		audit := &domain.ManagerOverrideAudit{
			ID:                s.ids.NewID(),
			CommandID:         cmd.CommandID,
			RestaurantID:      order.RestaurantID,
			DeviceID:          order.DeviceID,
			NodeDeviceID:      order.DeviceID,
			ClientDeviceID:    shared.OptionalID(cmd.ClientDeviceID),
			ShiftID:           order.ShiftID,
			OrderID:           order.ID,
			PrecheckID:        precheckID,
			ManagerEmployeeID: manager.ID,
			ActorEmployeeID:   shared.OptionalID(operator.Employee.ID),
			SessionID:         shared.OptionalID(cmd.SessionID),
			Action:            "cancel_unconfirmed_order",
			Reason:            strings.TrimSpace(cmd.Reason),
			OccurredAt:        now,
			CreatedAt:         now,
		}
		if err := s.repo.CreateManagerOverrideAudit(ctx, audit); err != nil {
			return err
		}

		order.Status = domain.OrderCancelled
		order.CancelledAt = &now
		order.UpdatedAt = now
		if err := s.repo.UpdateOrderCancelled(ctx, order); err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Order", order.ID, "OrderCancelled", order); err != nil {
			return err
		}
		result = order
		return nil
	})
	return result, err
}

// resolveCancelUnconfirmedApprover ищет активного сотрудника ресторана с правом
// pos.order.cancel_unconfirmed, чей PIN совпадает с введённым — по образцу
// resolveManagerOverrideByPIN в app/precheck.
func (s *Service) resolveCancelUnconfirmedApprover(ctx context.Context, restaurantID, pin string) (*domain.Employee, error) {
	employees, err := s.repo.ListEmployeesByRestaurant(ctx, restaurantID)
	if err != nil {
		return nil, err
	}
	for i := range employees {
		employee := employees[i]
		if !employee.Active {
			continue
		}
		role, err := s.repo.GetRole(ctx, employee.RoleID)
		if err != nil {
			return nil, err
		}
		if !role.Active || !shared.HasPermission(role.PermissionsJSON, string(shared.PermissionOrderCancelUnconfirmed)) {
			continue
		}
		if err := shared.VerifyPIN(employee.PINHash, pin); err == nil {
			return &employee, nil
		}
	}
	return nil, fmt.Errorf("%w: manager override permission is required", domain.ErrForbidden)
}
