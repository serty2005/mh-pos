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
			"ordered_modifiers_by_application_index",
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
		subtotal, err := safeAddNonNegative(result.SubtotalMinor, line.Subtotal, "subtotal overflow")
		if err != nil {
			return CalculationResult{}, err
		}
		result.SubtotalMinor = subtotal
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
	modifiers, err := buildOrderedModifiers(input.Discounts, input.Surcharges)
	if err != nil {
		return CalculationResult{}, err
	}
	if err := applyModifiers(&result, lineIndex, modifiers); err != nil {
		return CalculationResult{}, err
	}
	for i := range result.Lines {
		line := &result.Lines[i]
		taxableBase, err := currentLineAmount(*line)
		if err != nil {
			return CalculationResult{}, err
		}
		line.TaxableBaseMinor = taxableBase
	}
	if err := applyTaxes(&result, input.TaxProfiles, input.TaxRules); err != nil {
		return CalculationResult{}, err
	}
	grand, err := safeSubNonNegative(result.SubtotalMinor, result.DiscountTotalMinor, "subtotal cannot become negative")
	if err != nil {
		return CalculationResult{}, err
	}
	grand, err = safeAddNonNegative(grand, result.SurchargeTotalMinor, "grand total overflow")
	if err != nil {
		return CalculationResult{}, err
	}
	grand, err = safeAddNonNegative(grand, result.TaxAddedTotalMinor, "grand total overflow")
	if err != nil {
		return CalculationResult{}, err
	}
	result.GrandTotalMinor = grand
	if result.GrandTotalMinor < 0 {
		return CalculationResult{}, fmt.Errorf("%w: grand total cannot be negative", shared.ErrInvalid)
	}
	for i := range result.Lines {
		line := &result.Lines[i]
		total, err := currentLineAmount(*line)
		if err != nil {
			return CalculationResult{}, err
		}
		total, err = safeAddNonNegative(total, line.TaxAddedMinor, "line total overflow")
		if err != nil {
			return CalculationResult{}, err
		}
		line.TotalMinor = total
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

func buildOrderedModifiers(discounts []OrderDiscount, surcharges []OrderSurcharge) ([]CalculationModifier, error) {
	modifiers := make([]CalculationModifier, 0, len(discounts)+len(surcharges))
	seen := make(map[int]string, len(discounts)+len(surcharges))
	for i := range discounts {
		discount := discounts[i]
		if err := registerApplicationIndex(seen, discount.ApplicationIndex, string(ModifierTypeDiscount), discount.ID); err != nil {
			return nil, err
		}
		modifiers = append(modifiers, CalculationModifier{Type: ModifierTypeDiscount, ApplicationIndex: discount.ApplicationIndex, Discount: &discount})
	}
	for i := range surcharges {
		surcharge := surcharges[i]
		if err := registerApplicationIndex(seen, surcharge.ApplicationIndex, string(ModifierTypeSurcharge), surcharge.ID); err != nil {
			return nil, err
		}
		modifiers = append(modifiers, CalculationModifier{Type: ModifierTypeSurcharge, ApplicationIndex: surcharge.ApplicationIndex, Surcharge: &surcharge})
	}
	sort.SliceStable(modifiers, func(i, j int) bool {
		return modifiers[i].ApplicationIndex < modifiers[j].ApplicationIndex
	})
	return modifiers, nil
}

func registerApplicationIndex(seen map[int]string, index int, modifierType, id string) error {
	if index <= 0 {
		return fmt.Errorf("%w: application_index must be positive", shared.ErrInvalid)
	}
	if previous, ok := seen[index]; ok {
		return fmt.Errorf("%w: duplicate application_index %d between %s and %s", shared.ErrInvalid, index, previous, modifierType)
	}
	seen[index] = modifierType + ":" + strings.TrimSpace(id)
	return nil
}

func applyModifiers(result *CalculationResult, lineIndex map[string]int, modifiers []CalculationModifier) error {
	for _, modifier := range modifiers {
		switch modifier.Type {
		case ModifierTypeDiscount:
			if modifier.Discount == nil {
				return fmt.Errorf("%w: discount modifier is empty", shared.ErrInvalid)
			}
			if err := applyDiscount(result, lineIndex, *modifier.Discount); err != nil {
				return err
			}
		case ModifierTypeSurcharge:
			if modifier.Surcharge == nil {
				return fmt.Errorf("%w: surcharge modifier is empty", shared.ErrInvalid)
			}
			if err := applySurcharge(result, *modifier.Surcharge); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: unsupported modifier type", shared.ErrInvalid)
		}
	}
	return nil
}

func applyDiscount(result *CalculationResult, lineIndex map[string]int, discount OrderDiscount) error {
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
		target, err := currentLineAmount(*line)
		if err != nil {
			return err
		}
		amount, err := adjustmentAmount(discount.AmountKind, discount.AmountMinor, discount.ValueBasisPoints, target)
		if err != nil {
			return err
		}
		if amount > target {
			return fmt.Errorf("%w: discount cannot exceed target amount", shared.ErrInvalid)
		}
		line.DiscountTotalMinor, err = safeAddNonNegative(line.DiscountTotalMinor, amount, "line discount overflow")
		if err != nil {
			return err
		}
		result.DiscountTotalMinor, err = safeAddNonNegative(result.DiscountTotalMinor, amount, "discount total overflow")
		if err != nil {
			return err
		}
		result.Discounts = append(result.Discounts, DiscountBreakdown{DiscountID: discount.ID, Scope: discount.Scope, ApplicationIndex: discount.ApplicationIndex, OrderLineID: discount.OrderLineID, AmountKind: discount.AmountKind, AmountMinor: amount, Reason: discount.Reason})
	case DiscountScopeOrder:
		bases, target, err := currentLineAmounts(result.Lines)
		if err != nil {
			return err
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
			result.Lines[i].DiscountTotalMinor, err = safeAddNonNegative(result.Lines[i].DiscountTotalMinor, value, "line discount overflow")
			if err != nil {
				return err
			}
		}
		result.DiscountTotalMinor, err = safeAddNonNegative(result.DiscountTotalMinor, amount, "discount total overflow")
		if err != nil {
			return err
		}
		result.Discounts = append(result.Discounts, DiscountBreakdown{DiscountID: discount.ID, Scope: discount.Scope, ApplicationIndex: discount.ApplicationIndex, AmountKind: discount.AmountKind, AmountMinor: amount, Reason: discount.Reason})
	default:
		return fmt.Errorf("%w: unsupported discount scope", shared.ErrInvalid)
	}
	return nil
}

func applySurcharge(result *CalculationResult, surcharge OrderSurcharge) error {
	bases, target, err := currentLineAmounts(result.Lines)
	if err != nil {
		return err
	}
	amount, err := adjustmentAmount(surcharge.AmountKind, surcharge.AmountMinor, surcharge.ValueBasisPoints, target)
	if err != nil {
		return err
	}
	allocated := allocateProportionally(amount, bases)
	for i, value := range allocated {
		result.Lines[i].SurchargeTotalMinor, err = safeAddNonNegative(result.Lines[i].SurchargeTotalMinor, value, "line surcharge overflow")
		if err != nil {
			return err
		}
	}
	result.SurchargeTotalMinor, err = safeAddNonNegative(result.SurchargeTotalMinor, amount, "surcharge total overflow")
	if err != nil {
		return err
	}
	result.Surcharges = append(result.Surcharges, SurchargeBreakdown{SurchargeID: surcharge.ID, Kind: surcharge.Kind, ApplicationIndex: surcharge.ApplicationIndex, AmountKind: surcharge.AmountKind, AmountMinor: amount, Reason: surcharge.Reason})
	return nil
}

func currentLineAmounts(lines []LineBreakdown) ([]int64, int64, error) {
	bases := make([]int64, 0, len(lines))
	var target int64
	for _, line := range lines {
		base, err := currentLineAmount(line)
		if err != nil {
			return nil, 0, err
		}
		bases = append(bases, base)
		target, err = safeAddNonNegative(target, base, "modifier target overflow")
		if err != nil {
			return nil, 0, err
		}
	}
	return bases, target, nil
}

func currentLineAmount(line LineBreakdown) (int64, error) {
	current, err := safeSubNonNegative(line.SubtotalMinor, line.DiscountTotalMinor, "subtotal cannot become negative")
	if err != nil {
		return 0, err
	}
	return safeAddNonNegative(current, line.SurchargeTotalMinor, "line modifier target overflow")
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

const maxInt64 = int64(1<<63 - 1)

func safeAddNonNegative(a, b int64, message string) (int64, error) {
	if a < 0 || b < 0 {
		return 0, fmt.Errorf("%w: money values must be non-negative", shared.ErrInvalid)
	}
	if a > maxInt64-b {
		return 0, fmt.Errorf("%w: %s", shared.ErrInvalid, message)
	}
	return a + b, nil
}

func safeSubNonNegative(a, b int64, message string) (int64, error) {
	if a < 0 || b < 0 {
		return 0, fmt.Errorf("%w: money values must be non-negative", shared.ErrInvalid)
	}
	if b > a {
		return 0, fmt.Errorf("%w: %s", shared.ErrInvalid, message)
	}
	return a - b, nil
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
				var err error
				base, err = safeAddNonNegative(base, previousExclusiveTax, "compound tax base overflow")
				if err != nil {
					return err
				}
			}
			amount, err := taxAmount(rule, base)
			if err != nil {
				return err
			}
			line.TaxTotalMinor, err = safeAddNonNegative(line.TaxTotalMinor, amount, "line tax total overflow")
			if err != nil {
				return err
			}
			result.TaxTotalMinor, err = safeAddNonNegative(result.TaxTotalMinor, amount, "tax total overflow")
			if err != nil {
				return err
			}
			if rule.Mode == TaxModeExclusive {
				line.TaxAddedMinor, err = safeAddNonNegative(line.TaxAddedMinor, amount, "line tax added overflow")
				if err != nil {
					return err
				}
				result.TaxAddedTotalMinor, err = safeAddNonNegative(result.TaxAddedTotalMinor, amount, "tax added total overflow")
				if err != nil {
					return err
				}
			}
			if rule.Mode == TaxModeExclusive {
				previousExclusiveTax, err = safeAddNonNegative(previousExclusiveTax, amount, "compound tax accumulator overflow")
				if err != nil {
					return err
				}
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
