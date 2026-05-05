package ports

import (
	"context"

	"pos-backend/internal/pos/domain/shared"
)

type LocalEventRepository interface {
	CreateLocalEvent(context.Context, *shared.LocalEvent) error
	ListLocalEvents(context.Context, int, string) ([]shared.LocalEvent, error)
	CountLocalEvents(context.Context) (int, error)
}
