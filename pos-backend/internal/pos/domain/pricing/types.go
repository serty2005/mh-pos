package pricing

import "time"

type AmountKind string
type DiscountScope string
type ModifierType string
type SurchargeKind string
type TaxMode string
type TaxRuleKind string

const (
	AmountPercentage AmountKind = "percentage"
	AmountFixed      AmountKind = "fixed"

	DiscountScopeLine  DiscountScope = "line"
	DiscountScopeOrder DiscountScope = "order"

	ModifierTypeDiscount  ModifierType = "discount"
	ModifierTypeSurcharge ModifierType = "surcharge"

	SurchargeServiceCharge SurchargeKind = "service_charge"
	SurchargePB1ServiceFee SurchargeKind = "pb1_service_fee"
	SurchargeManual        SurchargeKind = "manual"

	TaxModeExclusive TaxMode = "exclusive"
	TaxModeInclusive TaxMode = "inclusive"

	TaxRulePercentage TaxRuleKind = "percentage"
	TaxRuleFixed      TaxRuleKind = "fixed"
)

type OrderLineInput struct {
	ID            string
	MenuItemID    string
	CatalogItemID string
	Name          string
	Quantity      int64
	UnitPrice     int64
	Subtotal      int64
	CurrencyCode  string
	TaxProfileID  *string
}

type OrderDiscount struct {
	ID               string        `json:"id"`
	OrderID          string        `json:"order_id"`
	OrderLineID      *string       `json:"order_line_id,omitempty"`
	Scope            DiscountScope `json:"scope"`
	ApplicationIndex int           `json:"application_index"`
	AmountKind       AmountKind    `json:"amount_kind"`
	AmountMinor      int64         `json:"amount_minor,omitempty"`
	ValueBasisPoints int64         `json:"value_basis_points,omitempty"`
	Reason           *string       `json:"reason,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
}

type OrderSurcharge struct {
	ID               string        `json:"id"`
	OrderID          string        `json:"order_id"`
	Kind             SurchargeKind `json:"kind"`
	ApplicationIndex int           `json:"application_index"`
	AmountKind       AmountKind    `json:"amount_kind"`
	AmountMinor      int64         `json:"amount_minor,omitempty"`
	ValueBasisPoints int64         `json:"value_basis_points,omitempty"`
	Reason           *string       `json:"reason,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
}

type TaxProfile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TaxExempt bool      `json:"tax_exempt"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TaxRule struct {
	ID              string      `json:"id"`
	TaxProfileID    string      `json:"tax_profile_id"`
	Name            string      `json:"name"`
	Kind            TaxRuleKind `json:"kind"`
	Mode            TaxMode     `json:"mode"`
	RateBasisPoints int64       `json:"rate_basis_points,omitempty"`
	AmountMinor     int64       `json:"amount_minor,omitempty"`
	Compound        bool        `json:"compound"`
	Priority        int         `json:"priority"`
	Active          bool        `json:"active"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type CalculationInput struct {
	CurrencyCode string
	Lines        []OrderLineInput
	Discounts    []OrderDiscount
	Surcharges   []OrderSurcharge
	TaxProfiles  map[string]TaxProfile
	TaxRules     map[string][]TaxRule
}

// CalculationModifier фиксирует единое ordered-пространство скидок и надбавок.
type CalculationModifier struct {
	Type             ModifierType
	ApplicationIndex int
	Discount         *OrderDiscount
	Surcharge        *OrderSurcharge
}

type Discount = OrderDiscount
type Surcharge = OrderSurcharge
type TaxComponent = TaxComponentBreakdown
type CalculationSnapshot = CalculationResult

type CalculationResult struct {
	CurrencyCode        string                  `json:"currency_code"`
	SubtotalMinor       int64                   `json:"subtotal_minor"`
	DiscountTotalMinor  int64                   `json:"discount_total_minor"`
	SurchargeTotalMinor int64                   `json:"surcharge_total_minor"`
	TaxTotalMinor       int64                   `json:"tax_total_minor"`
	TaxAddedTotalMinor  int64                   `json:"tax_added_total_minor"`
	GrandTotalMinor     int64                   `json:"grand_total_minor"`
	Lines               []LineBreakdown         `json:"lines"`
	Discounts           []DiscountBreakdown     `json:"discounts"`
	Surcharges          []SurchargeBreakdown    `json:"surcharges"`
	Taxes               []TaxComponentBreakdown `json:"taxes"`
	RoundingPolicy      string                  `json:"rounding_policy"`
	Pipeline            []string                `json:"pipeline"`
}

type LineBreakdown struct {
	OrderLineID         string  `json:"order_line_id"`
	MenuItemID          string  `json:"menu_item_id"`
	CatalogItemID       string  `json:"catalog_item_id"`
	Name                string  `json:"name"`
	Quantity            int64   `json:"quantity"`
	UnitPriceMinor      int64   `json:"unit_price_minor"`
	SubtotalMinor       int64   `json:"subtotal_minor"`
	DiscountTotalMinor  int64   `json:"discount_total_minor"`
	SurchargeTotalMinor int64   `json:"surcharge_total_minor"`
	TaxableBaseMinor    int64   `json:"taxable_base_minor"`
	TaxTotalMinor       int64   `json:"tax_total_minor"`
	TaxAddedMinor       int64   `json:"tax_added_minor"`
	TotalMinor          int64   `json:"total_minor"`
	CurrencyCode        string  `json:"currency_code"`
	TaxProfileID        *string `json:"tax_profile_id,omitempty"`
}

type DiscountBreakdown struct {
	DiscountID       string        `json:"discount_id"`
	Scope            DiscountScope `json:"scope"`
	ApplicationIndex int           `json:"application_index"`
	OrderLineID      *string       `json:"order_line_id,omitempty"`
	AmountKind       AmountKind    `json:"amount_kind"`
	AmountMinor      int64         `json:"amount_minor"`
	Reason           *string       `json:"reason,omitempty"`
}

type SurchargeBreakdown struct {
	SurchargeID      string        `json:"surcharge_id"`
	Kind             SurchargeKind `json:"kind"`
	ApplicationIndex int           `json:"application_index"`
	AmountKind       AmountKind    `json:"amount_kind"`
	AmountMinor      int64         `json:"amount_minor"`
	Reason           *string       `json:"reason,omitempty"`
}

type TaxComponentBreakdown struct {
	OrderLineID      string      `json:"order_line_id"`
	TaxProfileID     string      `json:"tax_profile_id"`
	TaxRuleID        string      `json:"tax_rule_id"`
	Name             string      `json:"name"`
	Kind             TaxRuleKind `json:"kind"`
	Mode             TaxMode     `json:"mode"`
	RateBasisPoints  int64       `json:"rate_basis_points,omitempty"`
	TaxableBaseMinor int64       `json:"taxable_base_minor"`
	TaxAmountMinor   int64       `json:"tax_amount_minor"`
	Compound         bool        `json:"compound"`
	Priority         int         `json:"priority"`
}
