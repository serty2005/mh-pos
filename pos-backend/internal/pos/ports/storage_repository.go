package ports

import (
	"context"

	"pos-backend/internal/pos/domain/storage"
)

// StorageLifecycleRepository читает storage lifecycle метрики без изменения runtime tables.
type StorageLifecycleRepository interface {
	GetStorageLifecycleStatus(context.Context) (storage.LifecycleStatus, error)
	DryRunStorageRetention(context.Context, string) (storage.RetentionDryRunResult, error)
}
