package ports

import (
	"context"

	"pos-backend/internal/pos/domain/shared"
)

type MasterSyncRepository interface {
	UpsertMasterDataSyncState(context.Context, *shared.MasterDataSyncState) error
	GetMasterDataSyncState(context.Context, string, shared.MasterDataStream) (*shared.MasterDataSyncState, error)
	ListMasterDataSyncStates(context.Context, string) ([]shared.MasterDataSyncState, error)
}
