package ports

import (
	"context"

	"pos-backend/internal/pos/domain/catalog"
	"pos-backend/internal/pos/domain/device"
	"pos-backend/internal/pos/domain/employee"
	"pos-backend/internal/pos/domain/floor"
	"pos-backend/internal/pos/domain/menu"
	"pos-backend/internal/pos/domain/pricing"
	"pos-backend/internal/pos/domain/restaurant"
	"pos-backend/internal/pos/domain/shared"
)

type MasterSyncRepository interface {
	UpsertMasterRestaurant(context.Context, *restaurant.Restaurant, shared.MasterRecordSyncMeta) error
	UpsertMasterDevice(context.Context, *device.Device, shared.MasterRecordSyncMeta) error
	UpsertMasterRole(context.Context, *employee.Role, shared.MasterRecordSyncMeta) error
	UpsertMasterEmployee(context.Context, *employee.Employee, shared.MasterRecordSyncMeta) error
	UpsertMasterHall(context.Context, *floor.Hall, shared.MasterRecordSyncMeta) error
	UpsertMasterTable(context.Context, *floor.Table, shared.MasterRecordSyncMeta) error
	UpsertMasterCatalogItem(context.Context, *catalog.CatalogItem, shared.MasterRecordSyncMeta) error
	UpsertMasterCatalogFolder(context.Context, *catalog.CatalogFolder, shared.MasterRecordSyncMeta) error
	UpsertMasterFolderParameter(context.Context, *catalog.FolderParameter, shared.MasterRecordSyncMeta) error
	UpsertMasterCatalogTag(context.Context, *catalog.CatalogTag, shared.MasterRecordSyncMeta) error
	UpsertMasterCatalogItemTag(context.Context, *catalog.CatalogItemTag, shared.MasterRecordSyncMeta) error
	UpsertMasterModifierGroup(context.Context, *catalog.ModifierGroup, shared.MasterRecordSyncMeta) error
	UpsertMasterModifierOption(context.Context, *catalog.ModifierOption, shared.MasterRecordSyncMeta) error
	UpsertMasterModifierGroupBinding(context.Context, *catalog.ModifierGroupBinding, shared.MasterRecordSyncMeta) error
	UpsertMasterMenuItemModifierGroup(context.Context, *catalog.MenuItemModifierGroup, shared.MasterRecordSyncMeta) error
	UpsertMasterMenuItem(context.Context, *menu.MenuItem, shared.MasterRecordSyncMeta) error
	UpsertMasterTaxProfile(context.Context, *pricing.TaxProfile, shared.MasterRecordSyncMeta) error
	UpsertMasterTaxRule(context.Context, *pricing.TaxRule, shared.MasterRecordSyncMeta) error
	UpsertMasterServiceChargeRule(context.Context, *pricing.ServiceChargeRule, shared.MasterRecordSyncMeta) error
	UpsertMasterPricingPolicy(context.Context, *pricing.PricingPolicy, shared.MasterRecordSyncMeta) error
	UpsertMasterDataSyncState(context.Context, *shared.MasterDataSyncState) error
	GetMasterDataSyncState(context.Context, string, shared.MasterDataStream) (*shared.MasterDataSyncState, error)
	ListMasterDataSyncStates(context.Context, string) ([]shared.MasterDataSyncState, error)
}
