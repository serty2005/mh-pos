package ports

import (
	"context"

	"pos-backend/internal/pos/domain/shared"
)

type LocalEventRepository interface {
	CreateLocalEvent(context.Context, *shared.LocalEvent) error
	CountLocalEvents(context.Context) (int, error)
}
