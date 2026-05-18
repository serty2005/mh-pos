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
	PricingPolicyID  string                      `json:"pricing_policy_id"`
	Scope            domainpricing.DiscountScope `json:"scope,omitempty"`
	ApplicationIndex int                         `json:"application_index,omitempty"`
	AmountKind       domainpricing.AmountKind    `json:"amount_kind,omitempty"`
	AmountMinor      int64                       `json:"amount_minor,omitempty"`
	ValueBasisPoints int64                       `json:"value_basis_points,omitempty"`
	Reason           string                      `json:"reason,omitempty"`
}

type AddSurchargeCommand struct {
	shared.CommandMeta
	OrderID          string                      `json:"order_id"`
	PricingPolicyID  string                      `json:"pricing_policy_id"`
	Kind             domainpricing.SurchargeKind `json:"kind,omitempty"`
	ApplicationIndex int                         `json:"application_index,omitempty"`
	AmountKind       domainpricing.AmountKind    `json:"amount_kind,omitempty"`
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
	policyID := strings.TrimSpace(cmd.PricingPolicyID)
	if policyID != "" && hasManualDiscountFields(cmd) {
		return nil, fmt.Errorf("%w: cashier discount fields must come from pricing policy", domain.ErrInvalid)
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
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPricingDiscountApply))
		if err != nil {
			return err
		}
		order, err := ensurePricingEditableOrder(ctx, s.repo, cmd.OrderID, cmd.DeviceID)
		if err != nil {
			return err
		}
		policy, err := discountPolicyFromCommand(ctx, s.repo, policyID, cmd, order.RestaurantID, operator)
		if err != nil {
			return err
		}
		if err := ensureApplicationIndexAvailable(ctx, s.repo, order.ID, policy.ApplicationIndex); err != nil {
			return err
		}
		var lineID *string
		if policy.Scope == domainpricing.DiscountScopeLine {
			line, err := s.repo.GetOrderLine(ctx, cmd.OrderLineID)
			if err != nil {
				return err
			}
			if line.OrderID != order.ID || line.Status != domain.OrderLineActive {
				return fmt.Errorf("%w: discount line target is not active in order", domain.ErrConflict)
			}
			lineID = optionalString(cmd.OrderLineID)
		} else if strings.TrimSpace(cmd.OrderLineID) != "" {
			return fmt.Errorf("%w: order-scoped discount policy must not receive order_line_id", domain.ErrInvalid)
		}
		reason := optionalString(cmd.Reason)
		if reason == nil {
			reason = optionalString(policy.Name)
		}
		discount = &domain.OrderDiscount{
			ID:               s.ids.NewID(),
			OrderID:          order.ID,
			OrderLineID:      lineID,
			PricingPolicyID:  optionalString(policyID),
			Scope:            policy.Scope,
			ApplicationIndex: policy.ApplicationIndex,
			AmountKind:       policy.AmountKind,
			AmountMinor:      policy.AmountMinor,
			ValueBasisPoints: policy.ValueBasisPoints,
			Reason:           reason,
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
	policyID := strings.TrimSpace(cmd.PricingPolicyID)
	if policyID != "" && hasManualSurchargeFields(cmd) {
		return nil, fmt.Errorf("%w: cashier surcharge fields must come from pricing policy", domain.ErrInvalid)
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
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionPricingSurchargeApply))
		if err != nil {
			return err
		}
		order, err := ensurePricingEditableOrder(ctx, s.repo, cmd.OrderID, cmd.DeviceID)
		if err != nil {
			return err
		}
		policy, err := surchargePolicyFromCommand(ctx, s.repo, policyID, cmd, order.RestaurantID, operator)
		if err != nil {
			return err
		}
		if policy.Scope != domainpricing.DiscountScopeOrder {
			return fmt.Errorf("%w: surcharge policy must be order-scoped", domain.ErrInvalid)
		}
		if err := ensureApplicationIndexAvailable(ctx, s.repo, order.ID, policy.ApplicationIndex); err != nil {
			return err
		}
		reason := optionalString(cmd.Reason)
		if reason == nil {
			reason = optionalString(policy.Name)
		}
		surcharge = &domain.OrderSurcharge{
			ID:               s.ids.NewID(),
			OrderID:          order.ID,
			PricingPolicyID:  optionalString(policyID),
			Kind:             domain.SurchargeServiceCharge,
			ApplicationIndex: policy.ApplicationIndex,
			AmountKind:       policy.AmountKind,
			AmountMinor:      policy.AmountMinor,
			ValueBasisPoints: policy.ValueBasisPoints,
			Reason:           reason,
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

func (s *Service) ListActivePricingPoliciesAsOperator(ctx context.Context, meta shared.CommandMeta) ([]domainpricing.PricingPolicy, error) {
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionPricingView))
	if err != nil {
		return nil, err
	}
	if operator == nil || operator.Session == nil {
		return nil, fmt.Errorf("%w: operator session is required", domain.ErrInvalid)
	}
	return s.repo.ListActivePricingPolicies(ctx, operator.Session.RestaurantID)
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
		modifiers := make([]domainpricing.LineModifierInput, 0, len(line.Modifiers))
		for _, modifier := range line.Modifiers {
			modifiers = append(modifiers, domainpricing.LineModifierInput{
				ModifierGroupID:  modifier.ModifierGroupID,
				ModifierOptionID: modifier.ModifierOptionID,
				Name:             modifier.Name,
				Quantity:         modifier.Quantity,
				UnitPriceMinor:   modifier.UnitPrice,
				TotalMinor:       modifier.TotalPrice,
			})
		}
		inputLines = append(inputLines, domainpricing.OrderLineInput{
			ID:            line.ID,
			MenuItemID:    line.MenuItemID,
			CatalogItemID: line.CatalogItemID,
			Name:          line.Name,
			Quantity:      line.Quantity,
			UnitPrice:     line.UnitPrice,
			Modifiers:     modifiers,
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

func discountPolicyFromCommand(ctx context.Context, repo ports.Repository, policyID string, cmd AddDiscountCommand, restaurantID string, operator *shared.OperatorContext) (*domainpricing.PricingPolicy, error) {
	if policyID == "" {
		return &domainpricing.PricingPolicy{ID: "", RestaurantID: restaurantID, Kind: domainpricing.PricingPolicyDiscount, Scope: cmd.Scope, AmountKind: cmd.AmountKind, AmountMinor: cmd.AmountMinor, ValueBasisPoints: cmd.ValueBasisPoints, ApplicationIndex: cmd.ApplicationIndex, Active: true}, nil
	}
	policy, err := repo.GetPricingPolicy(ctx, policyID)
	if err != nil {
		return nil, err
	}
	return policy, validatePolicyForOrder(policy, restaurantID, domainpricing.PricingPolicyDiscount, operator)
}

func surchargePolicyFromCommand(ctx context.Context, repo ports.Repository, policyID string, cmd AddSurchargeCommand, restaurantID string, operator *shared.OperatorContext) (*domainpricing.PricingPolicy, error) {
	if policyID == "" {
		return &domainpricing.PricingPolicy{ID: "", RestaurantID: restaurantID, Kind: domainpricing.PricingPolicySurcharge, Scope: domainpricing.DiscountScopeOrder, AmountKind: cmd.AmountKind, AmountMinor: cmd.AmountMinor, ValueBasisPoints: cmd.ValueBasisPoints, ApplicationIndex: cmd.ApplicationIndex, Active: true}, nil
	}
	policy, err := repo.GetPricingPolicy(ctx, policyID)
	if err != nil {
		return nil, err
	}
	return policy, validatePolicyForOrder(policy, restaurantID, domainpricing.PricingPolicySurcharge, operator)
}

func hasManualDiscountFields(cmd AddDiscountCommand) bool {
	return strings.TrimSpace(string(cmd.Scope)) != "" || cmd.ApplicationIndex != 0 || strings.TrimSpace(string(cmd.AmountKind)) != "" || cmd.AmountMinor != 0 || cmd.ValueBasisPoints != 0
}

func hasManualSurchargeFields(cmd AddSurchargeCommand) bool {
	return strings.TrimSpace(string(cmd.Kind)) != "" || cmd.ApplicationIndex != 0 || strings.TrimSpace(string(cmd.AmountKind)) != "" || cmd.AmountMinor != 0 || cmd.ValueBasisPoints != 0
}

func validatePolicyForOrder(policy *domainpricing.PricingPolicy, restaurantID string, expected domainpricing.PricingPolicyKind, operator *shared.OperatorContext) error {
	if policy == nil {
		return fmt.Errorf("%w: pricing policy is required", domain.ErrInvalid)
	}
	if policy.RestaurantID != restaurantID {
		return fmt.Errorf("%w: pricing policy belongs to another restaurant", domain.ErrForbidden)
	}
	if !policy.Active {
		return fmt.Errorf("%w: pricing policy is inactive", domain.ErrConflict)
	}
	if policy.Manual {
		return fmt.Errorf("%w: manual override pricing policy is outside pilot runtime flow", domain.ErrForbidden)
	}
	if policy.Kind != expected {
		return fmt.Errorf("%w: pricing policy kind mismatch", domain.ErrInvalid)
	}
	if policy.ApplicationIndex <= 0 {
		return fmt.Errorf("%w: pricing policy application_index must be positive", domain.ErrInvalid)
	}
	if strings.TrimSpace(policy.RequiresPermission) != "" && !operatorHasPermission(operator, policy.RequiresPermission) {
		return fmt.Errorf("%w: permission %s is required", domain.ErrForbidden, policy.RequiresPermission)
	}
	return nil
}

func operatorHasPermission(operator *shared.OperatorContext, permission string) bool {
	permission = strings.TrimSpace(permission)
	if permission == "" || operator == nil {
		return true
	}
	for _, item := range operator.Permissions {
		if item == permission {
			return true
		}
	}
	return false
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

func ensureApplicationIndexAvailable(ctx context.Context, repo ports.Repository, orderID string, applicationIndex int) error {
	discounts, err := repo.ListOrderDiscounts(ctx, orderID)
	if err != nil {
		return err
	}
	for _, discount := range discounts {
		if discount.ApplicationIndex == applicationIndex {
			return fmt.Errorf("%w: duplicate application_index", domain.ErrInvalid)
		}
	}
	surcharges, err := repo.ListOrderSurcharges(ctx, orderID)
	if err != nil {
		return err
	}
	for _, surcharge := range surcharges {
		if surcharge.ApplicationIndex == applicationIndex {
			return fmt.Errorf("%w: duplicate application_index", domain.ErrInvalid)
		}
	}
	return nil
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
