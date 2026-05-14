package pricing

import (
	"fmt"
	"sort"
	"strings"

	"pos-backend/internal/pos/domain/shared"
)

const (
	RoundingPolicyHalfUpMinorUnits = "integer_half_up_minor_units_v1"
)

// Calculator реализует единственный порядок расчета totals для order pricing и precheck snapshot.
type Calculator struct{}

func NewCalculator() Calculator {
	return Calculator{}
}

func (Calculator) Calculate(input CalculationInput) (CalculationResult, error) {
	currency := strings.ToUpper(strings.TrimSpace(input.CurrencyCode))
	if currency == "" {
		return CalculationResult{}, fmt.Errorf("%w: currency_code is required", shared.ErrInvalid)
	}
	result := CalculationResult{
		CurrencyCode:   currency,
		RoundingPolicy: RoundingPolicyHalfUpMinorUnits,
		Pipeline: []string{
			"order_lines_subtotal",
			"line_discounts",
			"order_discounts",
			"surcharges",
			"taxable_base",
			"taxes",
			"grand_total",
		},
	}
	lines := append([]OrderLineInput(nil), input.Lines...)
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].ID < lines[j].ID })
	if len(lines) == 0 {
		return result, nil
	}
	lineIndex := make(map[string]int, len(lines))
	for _, line := range lines {
		if err := validateLine(line, currency); err != nil {
			return CalculationResult{}, err
		}
		lineIndex[line.ID] = len(result.Lines)
		result.SubtotalMinor += line.Subtotal
		result.Lines = append(result.Lines, LineBreakdown{
			OrderLineID:    line.ID,
			MenuItemID:     line.MenuItemID,
			CatalogItemID:  line.CatalogItemID,
			Name:           line.Name,
			Quantity:       line.Quantity,
			UnitPriceMinor: line.UnitPrice,
			SubtotalMinor:  line.Subtotal,
			CurrencyCode:   line.CurrencyCode,
			TaxProfileID:   line.TaxProfileID,
		})
	}
	if err := applyDiscounts(&result, lineIndex, input.Discounts); err != nil {
		return CalculationResult{}, err
	}
	if err := applySurcharges(&result, input.Surcharges); err != nil {
		return CalculationResult{}, err
	}
	for i := range result.Lines {
		line := &result.Lines[i]
		line.TaxableBaseMinor = line.SubtotalMinor - line.DiscountTotalMinor + line.SurchargeTotalMinor
		if line.TaxableBaseMinor < 0 {
			return CalculationResult{}, fmt.Errorf("%w: taxable base cannot be negative", shared.ErrInvalid)
		}
	}
	if err := applyTaxes(&result, input.TaxProfiles, input.TaxRules); err != nil {
		return CalculationResult{}, err
	}
	result.GrandTotalMinor = result.SubtotalMinor - result.DiscountTotalMinor + result.SurchargeTotalMinor + result.TaxAddedTotalMinor
	if result.GrandTotalMinor < 0 {
		return CalculationResult{}, fmt.Errorf("%w: grand total cannot be negative", shared.ErrInvalid)
	}
	for i := range result.Lines {
		line := &result.Lines[i]
		line.TotalMinor = line.SubtotalMinor - line.DiscountTotalMinor + line.SurchargeTotalMinor + line.TaxAddedMinor
		if line.TotalMinor < 0 {
			return CalculationResult{}, fmt.Errorf("%w: line total cannot be negative", shared.ErrInvalid)
		}
	}
	return result, nil
}

func validateLine(line OrderLineInput, currency string) error {
	if strings.TrimSpace(line.ID) == "" || strings.TrimSpace(line.MenuItemID) == "" || strings.TrimSpace(line.CatalogItemID) == "" {
		return fmt.Errorf("%w: order line identity is required", shared.ErrInvalid)
	}
	if line.Quantity <= 0 || line.UnitPrice < 0 || line.Subtotal < 0 {
		return fmt.Errorf("%w: order line money values must be non-negative", shared.ErrInvalid)
	}
	if line.UnitPrice*line.Quantity != line.Subtotal {
		return fmt.Errorf("%w: order line subtotal is inconsistent", shared.ErrInvalid)
	}
	if strings.ToUpper(strings.TrimSpace(line.CurrencyCode)) != currency {
		return fmt.Errorf("%w: mixed currencies are not supported in one calculation", shared.ErrInvalid)
	}
	return nil
}

func applyDiscounts(result *CalculationResult, lineIndex map[string]int, discounts []OrderDiscount) error {
	items := append([]OrderDiscount(nil), discounts...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	for _, discount := range items {
		switch discount.Scope {
		case DiscountScopeLine:
			if discount.OrderLineID == nil || strings.TrimSpace(*discount.OrderLineID) == "" {
				return fmt.Errorf("%w: line discount requires order_line_id", shared.ErrInvalid)
			}
			idx, ok := lineIndex[*discount.OrderLineID]
			if !ok {
				return fmt.Errorf("%w: line discount target is not in order", shared.ErrInvalid)
			}
			line := &result.Lines[idx]
			target := line.SubtotalMinor - line.DiscountTotalMinor
			amount, err := adjustmentAmount(discount.AmountKind, discount.AmountMinor, discount.ValueBasisPoints, target)
			if err != nil {
				return err
			}
			if amount > target {
				return fmt.Errorf("%w: discount cannot exceed target amount", shared.ErrInvalid)
			}
			line.DiscountTotalMinor += amount
			result.DiscountTotalMinor += amount
			result.Discounts = append(result.Discounts, DiscountBreakdown{DiscountID: discount.ID, Scope: discount.Scope, OrderLineID: discount.OrderLineID, AmountKind: discount.AmountKind, AmountMinor: amount, Reason: discount.Reason})
		case DiscountScopeOrder:
			var bases []int64
			var target int64
			for _, line := range result.Lines {
				base := line.SubtotalMinor - line.DiscountTotalMinor
				bases = append(bases, base)
				target += base
			}
			amount, err := adjustmentAmount(discount.AmountKind, discount.AmountMinor, discount.ValueBasisPoints, target)
			if err != nil {
				return err
			}
			if amount > target {
				return fmt.Errorf("%w: discount cannot exceed target amount", shared.ErrInvalid)
			}
			allocated := allocateProportionally(amount, bases)
			for i, value := range allocated {
				result.Lines[i].DiscountTotalMinor += value
			}
			result.DiscountTotalMinor += amount
			result.Discounts = append(result.Discounts, DiscountBreakdown{DiscountID: discount.ID, Scope: discount.Scope, AmountKind: discount.AmountKind, AmountMinor: amount, Reason: discount.Reason})
		default:
			return fmt.Errorf("%w: unsupported discount scope", shared.ErrInvalid)
		}
	}
	return nil
}

func applySurcharges(result *CalculationResult, surcharges []OrderSurcharge) error {
	items := append([]OrderSurcharge(nil), surcharges...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	for _, surcharge := range items {
		var bases []int64
		var target int64
		for _, line := range result.Lines {
			base := line.SubtotalMinor - line.DiscountTotalMinor + line.SurchargeTotalMinor
			bases = append(bases, base)
			target += base
		}
		amount, err := adjustmentAmount(surcharge.AmountKind, surcharge.AmountMinor, surcharge.ValueBasisPoints, target)
		if err != nil {
			return err
		}
		allocated := allocateProportionally(amount, bases)
		for i, value := range allocated {
			result.Lines[i].SurchargeTotalMinor += value
		}
		result.SurchargeTotalMinor += amount
		result.Surcharges = append(result.Surcharges, SurchargeBreakdown{SurchargeID: surcharge.ID, Kind: surcharge.Kind, AmountKind: surcharge.AmountKind, AmountMinor: amount, Reason: surcharge.Reason})
	}
	return nil
}

func adjustmentAmount(kind AmountKind, amountMinor, basisPoints, target int64) (int64, error) {
	if target < 0 {
		return 0, fmt.Errorf("%w: adjustment target cannot be negative", shared.ErrInvalid)
	}
	switch kind {
	case AmountFixed:
		if amountMinor < 0 {
			return 0, fmt.Errorf("%w: fixed adjustment amount must be non-negative", shared.ErrInvalid)
		}
		return amountMinor, nil
	case AmountPercentage:
		return percentOf(target, basisPoints)
	default:
		return 0, fmt.Errorf("%w: unsupported adjustment amount kind", shared.ErrInvalid)
	}
}

func applyTaxes(result *CalculationResult, profiles map[string]TaxProfile, rulesByProfile map[string][]TaxRule) error {
	for i := range result.Lines {
		line := &result.Lines[i]
		if line.TaxProfileID == nil || strings.TrimSpace(*line.TaxProfileID) == "" {
			continue
		}
		profileID := strings.TrimSpace(*line.TaxProfileID)
		profile, ok := profiles[profileID]
		if !ok || !profile.Active || profile.TaxExempt {
			continue
		}
		rules := append([]TaxRule(nil), rulesByProfile[profileID]...)
		sort.SliceStable(rules, func(i, j int) bool {
			if rules[i].Priority == rules[j].Priority {
				return rules[i].ID < rules[j].ID
			}
			return rules[i].Priority < rules[j].Priority
		})
		var previousExclusiveTax int64
		for _, rule := range rules {
			if !rule.Active {
				continue
			}
			base := line.TaxableBaseMinor
			if rule.Compound {
				base += previousExclusiveTax
			}
			amount, err := taxAmount(rule, base)
			if err != nil {
				return err
			}
			line.TaxTotalMinor += amount
			result.TaxTotalMinor += amount
			if rule.Mode == TaxModeExclusive {
				line.TaxAddedMinor += amount
				result.TaxAddedTotalMinor += amount
			}
			if rule.Mode == TaxModeExclusive {
				previousExclusiveTax += amount
			}
			result.Taxes = append(result.Taxes, TaxComponentBreakdown{
				OrderLineID:      line.OrderLineID,
				TaxProfileID:     profileID,
				TaxRuleID:        rule.ID,
				Name:             rule.Name,
				Kind:             rule.Kind,
				Mode:             rule.Mode,
				RateBasisPoints:  rule.RateBasisPoints,
				TaxableBaseMinor: base,
				TaxAmountMinor:   amount,
				Compound:         rule.Compound,
				Priority:         rule.Priority,
			})
		}
	}
	return nil
}

func taxAmount(rule TaxRule, base int64) (int64, error) {
	if base < 0 {
		return 0, fmt.Errorf("%w: tax base cannot be negative", shared.ErrInvalid)
	}
	switch rule.Kind {
	case TaxRuleFixed:
		if rule.AmountMinor < 0 {
			return 0, fmt.Errorf("%w: fixed tax amount must be non-negative", shared.ErrInvalid)
		}
		return rule.AmountMinor, nil
	case TaxRulePercentage:
		switch rule.Mode {
		case TaxModeInclusive:
			return inclusiveTaxOf(base, rule.RateBasisPoints)
		case TaxModeExclusive:
			return percentOf(base, rule.RateBasisPoints)
		default:
			return 0, fmt.Errorf("%w: unsupported tax mode", shared.ErrInvalid)
		}
	default:
		return 0, fmt.Errorf("%w: unsupported tax rule kind", shared.ErrInvalid)
	}
}
