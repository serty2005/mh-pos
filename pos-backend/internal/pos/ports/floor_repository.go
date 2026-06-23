package ports

import (
	"context"
	"time"

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
	// GetSystemTable возвращает системный стол для counter sale по restaurantID.
	GetSystemTable(ctx context.Context, restaurantID string) (*floor.Table, error)
	// EnsureSystemFloor идемпотентно создаёт системный зал и стол для ресторана.
	EnsureSystemFloor(ctx context.Context, restaurantID, hallID, tableID string, now time.Time) error
}
