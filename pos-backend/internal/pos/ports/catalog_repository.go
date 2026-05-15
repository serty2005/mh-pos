package ports

import (
	"context"

	"pos-backend/internal/pos/domain/catalog"
)

type CatalogRepository interface {
	CreateCatalogItem(context.Context, *catalog.CatalogItem) error
	ListCatalogItems(context.Context) ([]catalog.CatalogItem, error)
	GetCatalogItem(context.Context, string) (*catalog.CatalogItem, error)
	ListModifierGroupsForMenuItem(context.Context, string) ([]catalog.ModifierGroup, error)
	ListModifierOptionsByGroupIDs(context.Context, []string) (map[string][]catalog.ModifierOption, error)
	CatalogItemInUse(context.Context, string) (bool, error)
}
