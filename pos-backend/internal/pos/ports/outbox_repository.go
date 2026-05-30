package ports

import (
	"context"

	"pos-backend/internal/pos/domain/shared"
)

type OutboxRepository interface {
	CreateOutboxMessage(context.Context, *shared.OutboxMessage) error
	GetOutboxByID(context.Context, string) (*shared.OutboxMessage, error)
	GetOutboxByCommandID(context.Context, string) (*shared.OutboxMessage, error)
	ListOutbox(context.Context, int) ([]shared.OutboxMessage, error)
	ListOutboxByCommandType(context.Context, string, int) ([]shared.OutboxMessage, error)
	GetSyncStatus(context.Context) (shared.SyncStatus, error)
	RetryFailedOutbox(context.Context, string) (int, error)
	ClaimPendingOutbox(context.Context, int, string, string) ([]shared.OutboxMessage, error)
	ReleaseProcessingOutbox(context.Context, string, string) (int, error)
	ReclaimStaleProcessingOutbox(context.Context, string, string) (int, error)
	MarkOutboxSent(context.Context, string, string) error
	MarkOutboxFailed(context.Context, string, string, *string, string, int) error
	MarkOutboxRetryableFailure(context.Context, string, string, *string, string, int) error
	SuspendOutboxMessage(context.Context, string, string, string) error
	CountOutbox(context.Context) (int, error)
}
