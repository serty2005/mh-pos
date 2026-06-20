package menu

import "time"

type MenuItem struct {
	ID                        string                  `json:"id"`
	CatalogItemID             string                  `json:"catalog_item_id"`
	CategoryID                string                  `json:"category_id,omitempty"`
	TagID                     string                  `json:"tag_id,omitempty"`
	ItemType                  string                  `json:"item_type,omitempty"`
	Name                      string                  `json:"name"`
	Price                     int64                   `json:"price"`
	Currency                  string                  `json:"currency"`
	TaxProfileID              *string                 `json:"tax_profile_id,omitempty"`
	RuntimeStatus             string                  `json:"runtime_status,omitempty"`
	ModifierGroups            []MenuItemModifierGroup `json:"modifier_groups,omitempty"`
	StopListActive            bool                    `json:"stop_list_active,omitempty"`
	StopListBlocked           bool                    `json:"stop_list_blocked,omitempty"`
	StopListAvailableQuantity *float64                `json:"stop_list_available_quantity,omitempty"`
	Active                    bool                    `json:"active"`
	CreatedAt                 time.Time               `json:"created_at"`
	UpdatedAt                 time.Time               `json:"updated_at"`
}

type MenuItemModifierGroup struct {
	ID           string                   `json:"id"`
	RestaurantID string                   `json:"restaurant_id"`
	Name         string                   `json:"name"`
	Required     bool                     `json:"required"`
	MinCount     int                      `json:"min_count"`
	MaxCount     int                      `json:"max_count"`
	Active       bool                     `json:"active"`
	Options      []MenuItemModifierOption `json:"options,omitempty"`
}

type MenuItemModifierOption struct {
	ID              string `json:"id"`
	RestaurantID    string `json:"restaurant_id"`
	ModifierGroupID string `json:"modifier_group_id"`
	Name            string `json:"name"`
	PriceMinor      int64  `json:"price_minor"`
	Active          bool   `json:"active"`
}
