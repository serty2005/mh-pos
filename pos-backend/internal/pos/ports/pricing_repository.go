package ports

import (
	"context"

	"pos-backend/internal/pos/domain/pricing"
)

type PricingRepository interface {
	CreateOrderDiscount(context.Context, *pricing.OrderDiscount) error
	CreateOrderSurcharge(context.Context, *pricing.OrderSurcharge) error
	ListOrderDiscounts(context.Context, string) ([]pricing.OrderDiscount, error)
	ListOrderSurcharges(context.Context, string) ([]pricing.OrderSurcharge, error)
	ListTaxProfilesByIDs(context.Context, []string) (map[string]pricing.TaxProfile, error)
	ListTaxRulesByProfileIDs(context.Context, []string) (map[string][]pricing.TaxRule, error)
	CreatePrecheckBreakdown(context.Context, string, pricing.CalculationResult) error
}
