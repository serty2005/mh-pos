package ports

import (
	"context"

	"pos-backend/internal/pos/domain/inventory"
	"pos-backend/internal/pos/domain/shared"
)

type InventoryRepository interface {
	CreateRecipeVersion(context.Context, *inventory.RecipeVersion) error
	ListRecipeVersions(context.Context) ([]inventory.RecipeVersion, error)
	GetActiveRecipeVersionByCatalogItem(context.Context, string) (*inventory.RecipeVersion, error)
	CreateRecipeLine(context.Context, *inventory.RecipeLine) error
	ListRecipeLines(context.Context, string) ([]inventory.RecipeLine, error)
	UpsertMasterRecipeVersion(context.Context, *inventory.RecipeVersion, shared.MasterRecordSyncMeta) error
	UpsertMasterRecipeLine(context.Context, *inventory.RecipeLine, shared.MasterRecordSyncMeta) error
	UpsertMasterStopListEntry(context.Context, *inventory.StopListEntry, shared.MasterRecordSyncMeta) error
	UpsertMasterWarehouseReference(context.Context, *inventory.WarehouseReference, shared.MasterRecordSyncMeta) error
	GetBlockingStopListEntry(context.Context, string, string) (*inventory.StopListEntry, error)
	GetWarehouseReference(context.Context, string, string) (*inventory.WarehouseReference, error)
	GetDefaultWarehouseReference(context.Context, string) (*inventory.WarehouseReference, error)
}
