package inventory

import "time"

type RecipeVersionStatus string

const (
	RecipeVersionDraft    RecipeVersionStatus = "draft"
	RecipeVersionActive   RecipeVersionStatus = "active"
	RecipeVersionArchived RecipeVersionStatus = "archived"
)

type RecipeVersion struct {
	ID                string              `json:"id"`
	DishCatalogItemID string              `json:"dish_catalog_item_id"`
	Version           int                 `json:"version"`
	Name              string              `json:"name"`
	Status            RecipeVersionStatus `json:"status"`
	YieldQuantity     int64               `json:"yield_quantity"`
	YieldUnit         string              `json:"yield_unit"`
	Active            bool                `json:"active"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type RecipeLine struct {
	ID              string    `json:"id"`
	RecipeVersionID string    `json:"recipe_version_id"`
	CatalogItemID   string    `json:"catalog_item_id"`
	Quantity        int64     `json:"quantity"`
	Unit            string    `json:"unit"`
	LossPercent     int       `json:"loss_percent"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
