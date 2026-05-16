package order

import "time"

type OrderLineStatus string

const (
	OrderLineActive    OrderLineStatus = "active"
	OrderLineCancelled OrderLineStatus = "cancelled"
	OrderLineVoided    OrderLineStatus = "voided"
)

type OrderLine struct {
	ID            string          `json:"id"`
	OrderID       string          `json:"order_id"`
	MenuItemID    string          `json:"menu_item_id"`
	CatalogItemID string          `json:"catalog_item_id"`
	Name          string          `json:"name"`
	Quantity      int64           `json:"quantity"`
	UnitPrice     int64           `json:"unit_price"`
	TotalPrice    int64           `json:"total_price"`
	CurrencyCode  string          `json:"currency_code"`
	TaxProfileID  *string         `json:"tax_profile_id,omitempty"`
	Course        *string         `json:"course,omitempty"`
	Comment       *string         `json:"comment,omitempty"`
	Modifiers     []LineModifier  `json:"modifiers,omitempty"`
	Status        OrderLineStatus `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type LineModifier struct {
	ID               string `json:"id"`
	OrderLineID      string `json:"order_line_id"`
	ModifierGroupID  string `json:"modifier_group_id"`
	ModifierOptionID string `json:"modifier_option_id"`
	Name             string `json:"name"`
	Quantity         int64  `json:"quantity"`
	UnitPrice        int64  `json:"unit_price"`
	TotalPrice       int64  `json:"total_price"`
}
