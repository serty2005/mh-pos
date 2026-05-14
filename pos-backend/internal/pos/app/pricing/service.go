package pricing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	domainpricing "pos-backend/internal/pos/domain/pricing"
	"pos-backend/internal/pos/ports"
)

type Service struct {
	repo  ports.Repository
	tx    txmanager.Manager
	ids   idgen.Generator
	clock clock.Clock
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock}
}

type AddDiscountCommand struct {
	shared.CommandMeta
	OrderID          string                      `json:"order_id"`
	OrderLineID      string                      `json:"order_line_id,omitempty"`
	Scope            domainpricing.DiscountScope `json:"scope"`
	AmountKind       domainpricing.AmountKind    `json:"amount_kind"`
	AmountMinor      int64                       `json:"amount_minor,omitempty"`
	ValueBasisPoints int64                       `json:"value_basis_points,omitempty"`
	Reason           string                      `json:"reason,omitempty"`
}

type AddSurchargeCommand struct {
	shared.CommandMeta
	OrderID          string                      `json:"order_id"`
	Kind             domainpricing.SurchargeKind `json:"kind"`
	AmountKind       domainpricing.AmountKind    `json:"amount_kind"`
	AmountMinor      int64                       `json:"amount_minor,omitempty"`
	ValueBasisPoints int64                       `json:"value_basis_points,omitempty"`
	Reason           string                      `json:"reason,omitempty"`
}

func (s *Service) AddDiscount(ctx context.Context, cmd AddDiscountCommand) (*domain.OrderDiscount, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" {
		return nil, fmt.Errorf("%w: order_id is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var discount *domain.OrderDiscount
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPricingDiscountApply)); err != nil {
			return err
		}
		order, err := ensurePricingEditableOrder(ctx, s.repo, cmd.OrderID, cmd.DeviceID)
		if err != nil {
			return err
		}
		var lineID *string
		if cmd.Scope == domainpricing.DiscountScopeLine {
			line, err := s.repo.GetOrderLine(ctx, cmd.OrderLineID)
			if err != nil {
				return err
			}
			if line.OrderID != order.ID || line.Status != domain.OrderLineActive {
				return fmt.Errorf("%w: discount line target is not active in order", domain.ErrConflict)
			}
			lineID = optionalString(cmd.OrderLineID)
		}
		discount = &domain.OrderDiscount{
			ID:               s.ids.NewID(),
			OrderID:          order.ID,
			OrderLineID:      lineID,
			Scope:            cmd.Scope,
			AmountKind:       cmd.AmountKind,
			AmountMinor:      cmd.AmountMinor,
			ValueBasisPoints: cmd.ValueBasisPoints,
			Reason:           optionalString(cmd.Reason),
			CreatedAt:        now,
		}
		if err := s.repo.CreateOrderDiscount(ctx, discount); err != nil {
			return err
		}
		if _, err := s.CalculateOrderPricing(ctx, order.ID); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Order", order.ID, "OrderDiscountApplied", discount)
	})
	return discount, err
}

func (s *Service) AddSurcharge(ctx context.Context, cmd AddSurchargeCommand) (*domain.OrderSurcharge, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.OrderID) == "" {
		return nil, fmt.Errorf("%w: order_id is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var surcharge *domain.OrderSurcharge
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPricingSurchargeApply)); err != nil {
			return err
		}
		order, err := ensurePricingEditableOrder(ctx, s.repo, cmd.OrderID, cmd.DeviceID)
		if err != nil {
			return err
		}
		surcharge = &domain.OrderSurcharge{
			ID:               s.ids.NewID(),
			OrderID:          order.ID,
			Kind:             cmd.Kind,
			AmountKind:       cmd.AmountKind,
			AmountMinor:      cmd.AmountMinor,
			ValueBasisPoints: cmd.ValueBasisPoints,
			Reason:           optionalString(cmd.Reason),
			CreatedAt:        now,
		}
		if err := s.repo.CreateOrderSurcharge(ctx, surcharge); err != nil {
			return err
		}
		if _, err := s.CalculateOrderPricing(ctx, order.ID); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, order.RestaurantID, order.ShiftID, "Order", order.ID, "OrderSurchargeApplied", surcharge)
	})
	return surcharge, err
}

func (s *Service) GetOrderPricingAsOperator(ctx context.Context, orderID string, meta shared.CommandMeta) (*domain.CalculationResult, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPricingView)); err != nil {
		return nil, err
	}
	result, err := s.CalculateOrderPricing(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) CalculateOrderPricing(ctx context.Context, orderID string) (domain.CalculationResult, error) {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return domain.CalculationResult{}, err
	}
	restaurant, err := s.repo.GetRestaurant(ctx, order.RestaurantID)
	if err != nil {
		return domain.CalculationResult{}, err
	}
	lines, err := s.repo.ListOrderLines(ctx, order.ID)
	if err != nil {
		return domain.CalculationResult{}, err
	}
	discounts, err := s.repo.ListOrderDiscounts(ctx, order.ID)
	if err != nil {
		return domain.CalculationResult{}, err
	}
	surcharges, err := s.repo.ListOrderSurcharges(ctx, order.ID)
	if err != nil {
		return domain.CalculationResult{}, err
	}
	var inputLines []domainpricing.OrderLineInput
	profileSeen := map[string]struct{}{}
	for _, line := range lines {
		if line.Status != domain.OrderLineActive {
			continue
		}
		currency := strings.TrimSpace(line.CurrencyCode)
		if currency == "" {
			currency = restaurant.Currency
		}
		if line.TaxProfileID != nil && strings.TrimSpace(*line.TaxProfileID) != "" {
			profileSeen[strings.TrimSpace(*line.TaxProfileID)] = struct{}{}
		}
		inputLines = append(inputLines, domainpricing.OrderLineInput{
			ID:            line.ID,
			MenuItemID:    line.MenuItemID,
			CatalogItemID: line.CatalogItemID,
			Name:          line.Name,
			Quantity:      line.Quantity,
			UnitPrice:     line.UnitPrice,
			Subtotal:      line.TotalPrice,
			CurrencyCode:  currency,
			TaxProfileID:  line.TaxProfileID,
		})
	}
	profileIDs := make([]string, 0, len(profileSeen))
	for id := range profileSeen {
		profileIDs = append(profileIDs, id)
	}
	profiles, err := s.repo.ListTaxProfilesByIDs(ctx, profileIDs)
	if err != nil {
		return domain.CalculationResult{}, err
	}
	rules, err := s.repo.ListTaxRulesByProfileIDs(ctx, profileIDs)
	if err != nil {
		return domain.CalculationResult{}, err
	}
	return domainpricing.NewCalculator().Calculate(domainpricing.CalculationInput{
		CurrencyCode: restaurant.Currency,
		Lines:        inputLines,
		Discounts:    discounts,
		Surcharges:   surcharges,
		TaxProfiles:  profiles,
		TaxRules:     rules,
	})
}

func ensurePricingEditableOrder(ctx context.Context, repo ports.Repository, orderID, deviceID string) (*domain.Order, error) {
	order, err := repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.DeviceID != deviceID {
		return nil, fmt.Errorf("%w: order device does not match command device", domain.ErrConflict)
	}
	if order.Status != domain.OrderOpen {
		return nil, fmt.Errorf("%w: cannot change pricing for non-open order", domain.ErrConflict)
	}
	if _, err := repo.GetActivePrecheckByOrder(ctx, order.ID); err == nil {
		return nil, fmt.Errorf("%w: cannot change pricing with active precheck", domain.ErrConflict)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	return order, nil
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
