package catalog

import "time"

type CatalogItemType string

const (
	CatalogItemIngredient CatalogItemType = "ingredient"
	CatalogItemDish       CatalogItemType = "dish"
	CatalogItemGood       CatalogItemType = "good"
)

type CatalogItem struct {
	ID        string          `json:"id"`
	Type      CatalogItemType `json:"type"`
	Name      string          `json:"name"`
	SKU       string          `json:"sku"`
	BaseUnit  string          `json:"base_unit"`
	Active    bool            `json:"active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
