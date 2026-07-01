package precheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	domainprecheck "pos-backend/internal/pos/domain/precheck"
	domainpricing "pos-backend/internal/pos/domain/pricing"
	"pos-backend/internal/pos/ports"
)

type Service struct {
	repo    ports.Repository
	tx      txmanager.Manager
	ids     idgen.Generator
	clock   clock.Clock
	pricing pricingCalculator
	prints  printEnqueuer
}

type pricingCalculator interface {
	CalculateOrderPricing(context.Context, string) (domainpricing.CalculationResult, error)
}

type printEnqueuer interface {
	EnqueuePrecheck(context.Context, PrecheckPrintInput) error
}

type PrecheckPrintInput struct {
	Meta         shared.CommandMeta
	RestaurantID string
	PrecheckID   string
	SectionID    string
	Now          time.Time
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock, pricing pricingCalculator) *Service {
	return NewServiceWithOptions(repo, tx, ids, clock, pricing, nil)
}

func NewServiceWithOptions(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock, pricing pricingCalculator, prints printEnqueuer) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock, pricing: pricing, prints: prints}
}

type IssuePrecheckCommand struct {
	shared.CommandMeta
	OrderID string `json:"order_id"`
}

type CancelPrecheckCommand struct {
	shared.CommandMeta
	PrecheckID string `json:"precheck_id"`
	// ManagerEmployeeID сохранен для старых клиентов; новые клиенты передают только manager_pin.
	ManagerEmployeeID  string `json:"manager_employee_id,omitempty"`
	ManagerPIN         string `json:"manager_pin"`
	CancellationReason string `json:"cancellation_reason"`
}

type ReprintPrecheckCommand struct {
	shared.CommandMeta
	PrecheckID string `json:"precheck_id"`
}

func (s *Service) GetPrecheck(ctx context.Context, id string) (*domain.Precheck, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: precheck id is required", domain.ErrInvalid)
	}
	return s.repo.GetPrecheck(ctx, id)
}

// GetPrecheckAsOperator загружает precheck details для аутентифицированных операторских сценариев с проверкой RBAC.
func (s *Service) GetPrecheckAsOperator(ctx context.Context, id string, meta shared.CommandMeta) (*domain.Precheck, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrecheckView)); err != nil {
		return nil, err
	}
	return s.GetPrecheck(ctx, id)
}

func (s *Service) ListPrechecksByOrder(ctx context.Context, orderID string) ([]domain.Precheck, error) {
	if strings.TrimSpace(orderID) == "" {
		return nil, fmt.Errorf("%w: order_id is required", domain.ErrInvalid)
	}
	return s.repo.ListPrechecksByOrder(ctx, orderID)
}

// ListPrechecksByOrderAsOperator возвращает precheck history заказа для аутентифицированных операторских сценариев.
func (s *Service) ListPrechecksByOrderAsOperator(ctx context.Context, orderID string, meta shared.CommandMeta) ([]domain.Precheck, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPrecheckView)); err != nil {
		return nil, err
	}
	return s.ListPrechecksByOrder(ctx, orderID)
}

func (s *Service) IssuePrecheck(ctx context.Context, cmd IssuePrecheckCommand) (*domain.Precheck, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" {
		return nil, fmt.Errorf("%w: order_id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var precheck *domain.Precheck
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPrecheckIssue)); err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, cmd.OrderID)
		if err != nil {
			return err
		}
		if order.Status != domain.OrderOpen {
			return fmt.Errorf("%w: cannot issue precheck for closed order", domain.ErrConflict)
		}
		if order.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: precheck device does not match order device", domain.ErrConflict)
		}
		if _, err := s.repo.GetActivePrecheckByOrder(ctx, order.ID); err == nil {
			return fmt.Errorf("%w: order already has active precheck", domain.ErrConflict)
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		existing, err := s.repo.ListPrechecksByOrder(ctx, order.ID)
		if err != nil {
			return err
		}
		version := 1
		var supersedesPrecheckID *string
		for _, item := range existing {
			if item.Version >= version {
				version = item.Version + 1
				id := item.ID
				supersedesPrecheckID = &id
			}
		}
		calculation, err := s.pricing.CalculateOrderPricing(ctx, order.ID)
		if err != nil {
			return err
		}
		precheck, err = domainprecheck.NewIssuedVersion(s.ids.NewID(), order.ID, version, supersedesPrecheckID, calculation.CurrencyCode, calculation.SubtotalMinor, calculation.DiscountTotalMinor, calculation.SurchargeTotalMinor, calculation.TaxTotalMinor, calculation.GrandTotalMinor, now)
		if err != nil {
			return err
		}
		snapshot, err := buildPrecheckSnapshot(order, precheck, calculation, now)
		if err != nil {
			return err
		}
		precheck.Snapshot = snapshot
		if err := s.repo.CreatePrecheck(ctx, precheck); err != nil {
			return err
		}
		if err := s.repo.CreatePrecheckBreakdown(ctx, precheck.ID, calculation); err != nil {
			return err
		}
		order.Status = domain.OrderLocked
		order.UpdatedAt = now
		if err := s.repo.UpdateOrderLocked(ctx, order); err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Precheck", precheck.ID, "PrecheckIssued", precheck); err != nil {
			return err
		}
		if s.prints != nil {
			table, err := s.repo.GetTable(ctx, order.TableID)
			if err != nil {
				return err
			}
			return s.prints.EnqueuePrecheck(ctx, PrecheckPrintInput{
				Meta:         cmd.CommandMeta,
				RestaurantID: order.RestaurantID,
				PrecheckID:   precheck.ID,
				SectionID:    table.SectionID,
				Now:          now,
			})
		}
		return nil
	})
	return precheck, err
}

func (s *Service) ReprintPrecheck(ctx context.Context, cmd ReprintPrecheckCommand) (*domain.ReprintDocument, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.PrecheckID) == "" {
		return nil, fmt.Errorf("%w: precheck_id is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var document *domain.ReprintDocument
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPrecheckReprint)); err != nil {
			return err
		}
		precheck, err := s.repo.GetPrecheck(ctx, cmd.PrecheckID)
		if err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, precheck.OrderID)
		if err != nil {
			return err
		}
		if order.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: precheck device does not match command device", domain.ErrConflict)
		}
		if len(precheck.Snapshot) == 0 || !json.Valid(precheck.Snapshot) {
			return fmt.Errorf("%w: precheck snapshot is not available", domain.ErrConflict)
		}
		document = domain.NewReprintDocument("precheck", precheck.ID, precheck.Snapshot, cmd.ActorEmployeeID, now)
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Precheck", precheck.ID, "PrecheckReprinted", document)
	})
	return document, err
}

func (s *Service) CancelPrecheck(ctx context.Context, cmd CancelPrecheckCommand) (*domain.Precheck, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.PrecheckID) == "" || strings.TrimSpace(cmd.ManagerPIN) == "" || strings.TrimSpace(cmd.CancellationReason) == "" {
		return nil, fmt.Errorf("%w: precheck_id, manager_pin and cancellation_reason are required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var precheck *domain.Precheck
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPrecheckCancelRequest)); err != nil {
			return err
		}
		var err error
		precheck, err = s.repo.GetPrecheck(ctx, cmd.PrecheckID)
		if err != nil {
			return err
		}
		order, err := s.repo.GetOrder(ctx, precheck.OrderID)
		if err != nil {
			return err
		}
		manager, err := s.resolveManagerOverrideByPIN(ctx, order.RestaurantID, cmd.ManagerPIN)
		if err != nil {
			return err
		}
		if err := precheck.Cancel(now, manager.ID, cmd.CancellationReason); err != nil {
			return err
		}
		if order.Status != domain.OrderLocked {
			return fmt.Errorf("%w: precheck cancellation expects locked order", domain.ErrConflict)
		}
		if order.DeviceID != cmd.DeviceID {
			return fmt.Errorf("%w: precheck device does not match order device", domain.ErrConflict)
		}
		active, err := s.repo.GetActivePrecheckByOrder(ctx, order.ID)
		if err != nil {
			return err
		}
		if active.ID != precheck.ID {
			return fmt.Errorf("%w: precheck is not active for order", domain.ErrConflict)
		}
		if err := s.repo.UpdatePrecheckLifecycle(ctx, precheck); err != nil {
			return err
		}
		order.Status = domain.OrderOpen
		order.UpdatedAt = now
		if err := s.repo.UpdateOrderOpen(ctx, order); err != nil {
			return err
		}
		actorEmployeeID := cmd.ActorEmployeeID
		if strings.TrimSpace(actorEmployeeID) == "" {
			actorEmployeeID = manager.ID
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
			PrecheckID:        precheck.ID,
			ManagerEmployeeID: manager.ID,
			ActorEmployeeID:   shared.OptionalID(actorEmployeeID),
			SessionID:         shared.OptionalID(cmd.SessionID),
			Action:            "cancel_precheck",
			Reason:            strings.TrimSpace(cmd.CancellationReason),
			OccurredAt:        now,
			CreatedAt:         now,
		}
		if err := s.repo.CreateManagerOverrideAudit(ctx, audit); err != nil {
			return err
		}
		eventMeta := cmd.CommandMeta
		eventMeta.ActorEmployeeID = actorEmployeeID
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, eventMeta, order.RestaurantID, order.ShiftID, "Precheck", precheck.ID, "PrecheckCancelled", precheck)
	})
	return precheck, err
}

func (s *Service) resolveManagerOverrideByPIN(ctx context.Context, restaurantID, pin string) (*domain.Employee, error) {
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
		if !role.Active || !shared.HasPermission(role.PermissionsJSON, string(shared.PermissionPrecheckCancel)) {
			continue
		}
		if err := shared.VerifyPIN(employee.PINHash, pin); err == nil {
			return &employee, nil
		}
	}
	return nil, fmt.Errorf("%w: manager override permission is required", domain.ErrForbidden)
}

type precheckSnapshot struct {
	DocumentType   string                          `json:"document_type"`
	PrecheckID     string                          `json:"precheck_id"`
	OrderID        string                          `json:"order_id"`
	TableID        string                          `json:"table_id"`
	TableName      string                          `json:"table_name"`
	Version        int                             `json:"version"`
	Subtotal       int64                           `json:"subtotal"`
	DiscountTotal  int64                           `json:"discount_total"`
	TaxTotal       int64                           `json:"tax_total"`
	Total          int64                           `json:"total"`
	PaidTotal      int64                           `json:"paid_total"`
	RemainingTotal int64                           `json:"remaining_total"`
	CurrencyCode   string                          `json:"currency_code"`
	SurchargeTotal int64                           `json:"surcharge_total"`
	Breakdown      domainpricing.CalculationResult `json:"breakdown"`
	IssuedAt       string                          `json:"issued_at"`
	Lines          []domainpricing.LineBreakdown   `json:"lines"`
}

func buildPrecheckSnapshot(order *domain.Order, precheck *domain.Precheck, calculation domainpricing.CalculationResult, now time.Time) (json.RawMessage, error) {
	snapshot := precheckSnapshot{
		DocumentType:   "precheck",
		PrecheckID:     precheck.ID,
		OrderID:        order.ID,
		TableID:        order.TableID,
		TableName:      order.TableName,
		Version:        precheck.Version,
		CurrencyCode:   precheck.CurrencyCode,
		Subtotal:       precheck.Subtotal,
		DiscountTotal:  precheck.DiscountTotal,
		SurchargeTotal: precheck.SurchargeTotal,
		TaxTotal:       precheck.TaxTotal,
		Total:          precheck.Total,
		PaidTotal:      precheck.PaidTotal,
		RemainingTotal: precheck.RemainingTotal,
		Breakdown:      calculation,
		IssuedAt:       shared.DBTime(now),
		Lines:          calculation.Lines,
	}
	body, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}
