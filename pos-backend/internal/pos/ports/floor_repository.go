package ports

import (
	"context"

	"pos-backend/internal/pos/domain/floor"
)

type FloorRepository interface {
	CreateHall(context.Context, *floor.Hall) error
	GetHall(context.Context, string) (*floor.Hall, error)
	ListHalls(context.Context, string) ([]floor.Hall, error)
	ArchiveHall(context.Context, string, string) error
	CreateTable(context.Context, *floor.Table) error
	GetTable(context.Context, string) (*floor.Table, error)
	ListTables(context.Context, string, string) ([]floor.Table, error)
	ArchiveTable(context.Context, string, string) error
	GetRestaurantSection(context.Context, string) (*floor.RestaurantSection, error)
	ListRestaurantSections(context.Context, string) ([]floor.RestaurantSection, error)
	GetSalesPoint(context.Context, string) (*floor.SalesPoint, error)
	ListSalesPoints(context.Context, string) ([]floor.SalesPoint, error)
}
