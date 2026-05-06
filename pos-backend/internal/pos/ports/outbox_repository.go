package ports

import (
	"context"

	"pos-backend/internal/pos/domain/shared"
)

type OutboxRepository interface {
	CreateOutboxMessage(context.Context, *shared.OutboxMessage) error
	GetOutboxByCommandID(context.Context, string) (*shared.OutboxMessage, error)
	ListOutbox(context.Context, int) ([]shared.OutboxMessage, error)
	GetSyncStatus(context.Context) (shared.SyncStatus, error)
	RetryFailedOutbox(context.Context, string) (int, error)
	ClaimPendingOutbox(context.Context, int, string, string) ([]shared.OutboxMessage, error)
	ReclaimStaleProcessingOutbox(context.Context, string, string) (int, error)
	MarkOutboxSent(context.Context, string, string) error
	MarkOutboxFailed(context.Context, string, string, *string, string, int) error
	CountOutbox(context.Context) (int, error)
}
