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
	CreatePurchaseReceipt(context.Context, *inventory.PurchaseReceipt) error
	ListPurchaseReceipts(context.Context) ([]inventory.PurchaseReceipt, error)
	CreatePurchaseReceiptLine(context.Context, *inventory.PurchaseReceiptLine) error
	ListPurchaseReceiptLines(context.Context, string) ([]inventory.PurchaseReceiptLine, error)
	CreateStockDocument(context.Context, *inventory.StockDocument) error
	ListStockDocuments(context.Context) ([]inventory.StockDocument, error)
	CreateStockMove(context.Context, *inventory.StockMove) error
	ListStockMoves(context.Context, string) ([]inventory.StockMove, error)
	UpsertStockBalance(context.Context, *inventory.StockBalance) error
	ListStockBalances(context.Context) ([]inventory.StockBalance, error)
	CreateItemCost(context.Context, *inventory.ItemCost) error
	GetLastItemCost(context.Context, string) (*inventory.ItemCost, error)
	ListItemCosts(context.Context, string) ([]inventory.ItemCost, error)
}
