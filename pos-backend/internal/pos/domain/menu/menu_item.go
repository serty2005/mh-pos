package menu

import "time"

type MenuItem struct {
	ID            string    `json:"id"`
	CatalogItemID string    `json:"catalog_item_id"`
	Name          string    `json:"name"`
	Price         int64     `json:"price"`
	Currency      string    `json:"currency"`
	TaxProfileID  *string   `json:"tax_profile_id,omitempty"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
