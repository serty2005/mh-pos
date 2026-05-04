package ports

import (
	"context"

	"pos-backend/internal/pos/domain/menu"
)

type MenuRepository interface {
	CreateMenuItem(context.Context, *menu.MenuItem) error
	ListMenuItems(context.Context) ([]menu.MenuItem, error)
	GetMenuItem(context.Context, string) (*menu.MenuItem, error)
}
