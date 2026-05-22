package ports

import (
	"context"

	"pos-backend/internal/pos/domain/storage"
)

// StorageLifecycleRepository управляет безопасным lifecycle локального storage.
type StorageLifecycleRepository interface {
	GetStorageLifecycleStatus(context.Context) (storage.LifecycleStatus, error)
	DryRunStorageRetention(context.Context, string) (storage.RetentionDryRunResult, error)
	BuildStorageArchiveExportPlan(context.Context, string) (storage.ArchiveExportPlan, error)
	BuildStorageArchiveExportScope(context.Context, string) (storage.ArchiveExportScope, error)
	BuildStorageArchiveApplyRuntimeScope(context.Context, string) (storage.ArchiveApplyRuntimeScope, error)
	ApplyStorageArchiveDestructive(context.Context, string) (storage.ArchiveExportCounts, error)
}
