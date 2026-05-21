package ports

import (
	"context"

	"pos-backend/internal/pos/domain/inventory"
)

type InventoryRepository interface {
	CreateRecipeVersion(context.Context, *inventory.RecipeVersion) error
	ListRecipeVersions(context.Context) ([]inventory.RecipeVersion, error)
	CreateRecipeLine(context.Context, *inventory.RecipeLine) error
	ListRecipeLines(context.Context, string) ([]inventory.RecipeLine, error)
}
