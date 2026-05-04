package ports

import (
	"context"

	"pos-backend/internal/pos/domain/shared"
)

type OutboxRepository interface {
	CreateOutboxMessage(context.Context, *shared.OutboxMessage) error
	GetOutboxByCommandID(context.Context, string) (*shared.OutboxMessage, error)
	ListOutbox(context.Context, int) ([]shared.OutboxMessage, error)
	MarkOutboxSent(context.Context, string, string) error
	MarkOutboxFailed(context.Context, string, string, string) error
	CountOutbox(context.Context) (int, error)
}
