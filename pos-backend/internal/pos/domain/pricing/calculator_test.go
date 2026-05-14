package pricing

import (
	"reflect"
	"testing"
	"time"
)

func TestCalculatorTotalsPipeline(t *testing.T) {
	now := time.Date(2026, 5, 4, 20, 0, 0, 0, time.UTC)
	lineID := "line-1"
	profileID := "tax-profile-1"
	tests := []struct {
		name    string
		input   CalculationInput
		want    CalculationResult
		wantErr bool
	}{
		{
			name: "line discount before exclusive tax",
			input: baseInput([]OrderDiscount{
				{ID: "discount-1", OrderID: "order-1", OrderLineID: &lineID, Scope: DiscountScopeLine, AmountKind: AmountPercentage, ValueBasisPoints: 1000, CreatedAt: now},
			}, nil, []TaxRule{{ID: "tax-1", TaxProfileID: profileID, Name: "VAT 10", Kind: TaxRulePercentage, Mode: TaxModeExclusive, RateBasisPoints: 1000, Active: true}}),
			want: resultTotals(1000, 100, 0, 90, 990),
		},
		{
			name: "order discount and surcharge are allocated before tax",
			input: baseInput([]OrderDiscount{
				{ID: "discount-1", OrderID: "order-1", Scope: DiscountScopeOrder, AmountKind: AmountFixed, AmountMinor: 100, CreatedAt: now},
			}, []OrderSurcharge{
				{ID: "surcharge-1", OrderID: "order-1", Kind: SurchargeServiceCharge, AmountKind: AmountPercentage, ValueBasisPoints: 1000, CreatedAt: now},
			}, []TaxRule{{ID: "tax-1", TaxProfileID: profileID, Name: "VAT 10", Kind: TaxRulePercentage, Mode: TaxModeExclusive, RateBasisPoints: 1000, Active: true}}),
			want: resultTotals(1000, 100, 90, 99, 1089),
		},
		{
			name:  "inclusive tax extracts tax from taxable base",
			input: baseInput(nil, nil, []TaxRule{{ID: "tax-1", TaxProfileID: profileID, Name: "VAT 20 inclusive", Kind: TaxRulePercentage, Mode: TaxModeInclusive, RateBasisPoints: 2000, Active: true}}),
			want:  resultTotals(1000, 0, 0, 167, 1000),
		},
		{
			name: "compound tax uses previous exclusive component",
			input: baseInput(nil, nil, []TaxRule{
				{ID: "tax-1", TaxProfileID: profileID, Name: "Tax 10", Kind: TaxRulePercentage, Mode: TaxModeExclusive, RateBasisPoints: 1000, Priority: 1, Active: true},
				{ID: "tax-2", TaxProfileID: profileID, Name: "Compound 10", Kind: TaxRulePercentage, Mode: TaxModeExclusive, RateBasisPoints: 1000, Compound: true, Priority: 2, Active: true},
			}),
			want: resultTotals(1000, 0, 0, 210, 1210),
		},
		{
			name:  "fixed tax component is supported",
			input: baseInput(nil, nil, []TaxRule{{ID: "tax-1", TaxProfileID: profileID, Name: "Bottle fee", Kind: TaxRuleFixed, Mode: TaxModeExclusive, AmountMinor: 25, Active: true}}),
			want:  resultTotals(1000, 0, 0, 25, 1025),
		},
		{
			name: "over discount rejected",
			input: baseInput([]OrderDiscount{
				{ID: "discount-1", OrderID: "order-1", Scope: DiscountScopeOrder, AmountKind: AmountFixed, AmountMinor: 1001, CreatedAt: now},
			}, nil, nil),
			wantErr: true,
		},
		{
			name: "rounding is stable half up",
			input: CalculationInput{
				CurrencyCode: "RUB",
				Lines: []OrderLineInput{
					{ID: "line-1", MenuItemID: "menu-1", CatalogItemID: "catalog-1", Name: "A", Quantity: 1, UnitPrice: 1, Subtotal: 1, CurrencyCode: "RUB"},
					{ID: "line-2", MenuItemID: "menu-2", CatalogItemID: "catalog-2", Name: "B", Quantity: 1, UnitPrice: 1, Subtotal: 1, CurrencyCode: "RUB"},
				},
				Discounts: []OrderDiscount{{ID: "discount-1", OrderID: "order-1", Scope: DiscountScopeOrder, AmountKind: AmountFixed, AmountMinor: 1}},
			},
			want: resultTotals(2, 1, 0, 0, 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCalculator().Calculate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.SubtotalMinor != tt.want.SubtotalMinor ||
				got.DiscountTotalMinor != tt.want.DiscountTotalMinor ||
				got.SurchargeTotalMinor != tt.want.SurchargeTotalMinor ||
				got.TaxTotalMinor != tt.want.TaxTotalMinor ||
				got.GrandTotalMinor != tt.want.GrandTotalMinor {
				t.Fatalf("unexpected totals: %+v", got)
			}
			gotAgain, err := NewCalculator().Calculate(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, gotAgain) {
				t.Fatalf("calculation is not deterministic:\n%+v\n%+v", got, gotAgain)
			}
		})
	}
}

func baseInput(discounts []OrderDiscount, surcharges []OrderSurcharge, rules []TaxRule) CalculationInput {
	profileID := "tax-profile-1"
	return CalculationInput{
		CurrencyCode: "RUB",
		Lines: []OrderLineInput{{
			ID:            "line-1",
			MenuItemID:    "menu-1",
			CatalogItemID: "catalog-1",
			Name:          "Soup",
			Quantity:      1,
			UnitPrice:     1000,
			Subtotal:      1000,
			CurrencyCode:  "RUB",
			TaxProfileID:  &profileID,
		}},
		Discounts:  discounts,
		Surcharges: surcharges,
		TaxProfiles: map[string]TaxProfile{
			profileID: {ID: profileID, Name: "VAT", Active: true},
		},
		TaxRules: map[string][]TaxRule{
			profileID: rules,
		},
	}
}

func resultTotals(subtotal, discount, surcharge, tax, grand int64) CalculationResult {
	return CalculationResult{
		SubtotalMinor:       subtotal,
		DiscountTotalMinor:  discount,
		SurchargeTotalMinor: surcharge,
		TaxTotalMinor:       tax,
		GrandTotalMinor:     grand,
	}
}
