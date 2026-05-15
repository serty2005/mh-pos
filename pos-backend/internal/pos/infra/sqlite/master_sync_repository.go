package sqlite

import (
	"context"
	"database/sql"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) UpsertMasterRestaurant(ctx context.Context, v *domain.Restaurant, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO restaurants(id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  name = excluded.name,
  timezone = excluded.timezone,
  currency = excluded.currency,
  business_day_mode = excluded.business_day_mode,
  business_day_boundary_local_time = excluded.business_day_boundary_local_time,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.Name, v.Timezone, v.Currency, string(v.BusinessDayMode), v.BusinessDayBoundaryLocalTime, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterDevice(ctx context.Context, v *domain.Device, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO devices(id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  restaurant_id = excluded.restaurant_id,
  device_code = excluded.device_code,
  name = excluded.name,
  type = excluded.type,
  active = excluded.active,
  registered_at = excluded.registered_at,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.DeviceCode, v.Name, v.Type, boolInt(v.Active), dbTime(v.RegisteredAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterRole(ctx context.Context, v *domain.Role, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO roles(id,name,permissions_json,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  name = excluded.name,
  permissions_json = excluded.permissions_json,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.Name, v.PermissionsJSON, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterEmployee(ctx context.Context, v *domain.Employee, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO employees(id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  restaurant_id = excluded.restaurant_id,
  role_id = excluded.role_id,
  name = excluded.name,
  pin_hash = excluded.pin_hash,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.RoleID, v.Name, v.PINHash, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterHall(ctx context.Context, v *domain.Hall, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO halls(id,restaurant_id,name,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  restaurant_id = excluded.restaurant_id,
  name = excluded.name,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.Name, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterTable(ctx context.Context, v *domain.Table, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO tables(id,restaurant_id,hall_id,name,seats,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  restaurant_id = excluded.restaurant_id,
  hall_id = excluded.hall_id,
  name = excluded.name,
  seats = excluded.seats,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.HallID, v.Name, v.Seats, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterCatalogItem(ctx context.Context, v *domain.CatalogItem, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO catalog_items(id,type,folder_id,name,sku,base_unit,kitchen_type,accounting_category,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type = excluded.type,
  folder_id = excluded.folder_id,
  name = excluded.name,
  sku = excluded.sku,
  base_unit = excluded.base_unit,
  kitchen_type = excluded.kitchen_type,
  accounting_category = excluded.accounting_category,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, string(v.Type), nullableString(v.FolderID), v.Name, v.SKU, v.BaseUnit, v.KitchenType, v.AccountingCategory, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterCatalogFolder(ctx context.Context, v *domain.CatalogFolder, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO catalog_folders(id,restaurant_id,parent_id,name,sort_order,active,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET restaurant_id=excluded.restaurant_id,parent_id=excluded.parent_id,name=excluded.name,sort_order=excluded.sort_order,active=excluded.active,cloud_version=excluded.cloud_version,cloud_updated_at=excluded.cloud_updated_at,cloud_deleted_at=excluded.cloud_deleted_at,last_synced_at=excluded.last_synced_at`,
		v.ID, v.RestaurantID, nullableString(v.ParentID), v.Name, v.SortOrder, boolInt(v.Active), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterFolderParameter(ctx context.Context, v *domain.FolderParameter, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO catalog_folder_parameters(id,restaurant_id,folder_id,parameter_key,value_type,value_json,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET restaurant_id=excluded.restaurant_id,folder_id=excluded.folder_id,parameter_key=excluded.parameter_key,value_type=excluded.value_type,value_json=excluded.value_json,cloud_version=excluded.cloud_version,cloud_updated_at=excluded.cloud_updated_at,cloud_deleted_at=excluded.cloud_deleted_at,last_synced_at=excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.FolderID, v.ParameterKey, v.ValueType, v.ValueJSON, meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterCatalogTag(ctx context.Context, v *domain.CatalogTag, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO catalog_tags(id,restaurant_id,name,code,active,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET restaurant_id=excluded.restaurant_id,name=excluded.name,code=excluded.code,active=excluded.active,cloud_version=excluded.cloud_version,cloud_updated_at=excluded.cloud_updated_at,cloud_deleted_at=excluded.cloud_deleted_at,last_synced_at=excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.Name, v.Code, boolInt(v.Active), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterCatalogItemTag(ctx context.Context, v *domain.CatalogItemTag, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO catalog_item_tags(catalog_item_id,tag_id,restaurant_id,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(catalog_item_id,tag_id) DO UPDATE SET restaurant_id=excluded.restaurant_id,cloud_version=excluded.cloud_version,cloud_updated_at=excluded.cloud_updated_at,cloud_deleted_at=excluded.cloud_deleted_at,last_synced_at=excluded.last_synced_at`,
		v.CatalogItemID, v.TagID, v.RestaurantID, meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterModifierGroup(ctx context.Context, v *domain.ModifierGroup, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO modifier_groups(id,restaurant_id,name,required,min_count,max_count,active,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET restaurant_id=excluded.restaurant_id,name=excluded.name,required=excluded.required,min_count=excluded.min_count,max_count=excluded.max_count,active=excluded.active,cloud_version=excluded.cloud_version,cloud_updated_at=excluded.cloud_updated_at,cloud_deleted_at=excluded.cloud_deleted_at,last_synced_at=excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.Name, boolInt(v.Required), v.MinCount, v.MaxCount, boolInt(v.Active), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterModifierOption(ctx context.Context, v *domain.ModifierOption, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO modifier_options(id,restaurant_id,modifier_group_id,name,price_minor,active,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET restaurant_id=excluded.restaurant_id,modifier_group_id=excluded.modifier_group_id,name=excluded.name,price_minor=excluded.price_minor,active=excluded.active,cloud_version=excluded.cloud_version,cloud_updated_at=excluded.cloud_updated_at,cloud_deleted_at=excluded.cloud_deleted_at,last_synced_at=excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.ModifierGroupID, v.Name, v.PriceMinor, boolInt(v.Active), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterModifierGroupBinding(ctx context.Context, v *domain.ModifierGroupBinding, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO modifier_group_bindings(id,restaurant_id,modifier_group_id,target_type,target_id,sort_order,active,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET restaurant_id=excluded.restaurant_id,modifier_group_id=excluded.modifier_group_id,target_type=excluded.target_type,target_id=excluded.target_id,sort_order=excluded.sort_order,active=excluded.active,cloud_version=excluded.cloud_version,cloud_updated_at=excluded.cloud_updated_at,cloud_deleted_at=excluded.cloud_deleted_at,last_synced_at=excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.ModifierGroupID, string(v.TargetType), v.TargetID, v.SortOrder, boolInt(v.Active), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterMenuItemModifierGroup(ctx context.Context, v *domain.MenuItemModifierGroup, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO menu_item_modifier_groups(menu_item_id,modifier_group_id,sort_order,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT(menu_item_id,modifier_group_id) DO UPDATE SET sort_order=excluded.sort_order,cloud_version=excluded.cloud_version,cloud_updated_at=excluded.cloud_updated_at,cloud_deleted_at=excluded.cloud_deleted_at,last_synced_at=excluded.last_synced_at`,
		v.MenuItemID, v.ModifierGroupID, v.SortOrder, meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterMenuItem(ctx context.Context, v *domain.MenuItem, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO menu_items(id,catalog_item_id,name,price,currency,tax_profile_id,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  catalog_item_id = excluded.catalog_item_id,
  name = excluded.name,
  price = excluded.price,
  currency = excluded.currency,
  tax_profile_id = excluded.tax_profile_id,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.CatalogItemID, v.Name, v.Price, v.Currency, nullableString(v.TaxProfileID), boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterTaxProfile(ctx context.Context, v *domain.TaxProfile, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO tax_profiles(id,name,tax_exempt,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  name = excluded.name,
  tax_exempt = excluded.tax_exempt,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.Name, boolInt(v.TaxExempt), boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterTaxRule(ctx context.Context, v *domain.TaxRule, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO tax_rules(id,tax_profile_id,name,kind,mode,rate_basis_points,amount_minor,compound,priority,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  tax_profile_id = excluded.tax_profile_id,
  name = excluded.name,
  kind = excluded.kind,
  mode = excluded.mode,
  rate_basis_points = excluded.rate_basis_points,
  amount_minor = excluded.amount_minor,
  compound = excluded.compound,
  priority = excluded.priority,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.TaxProfileID, v.Name, string(v.Kind), string(v.Mode), v.RateBasisPoints, v.AmountMinor, boolInt(v.Compound), v.Priority, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterServiceChargeRule(ctx context.Context, v *domain.ServiceChargeRule, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO service_charge_rules(id,restaurant_id,name,kind,amount_kind,amount_minor,value_basis_points,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  restaurant_id = excluded.restaurant_id,
  name = excluded.name,
  kind = excluded.kind,
  amount_kind = excluded.amount_kind,
  amount_minor = excluded.amount_minor,
  value_basis_points = excluded.value_basis_points,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.RestaurantID, v.Name, string(v.Kind), string(v.AmountKind), v.AmountMinor, v.ValueBasisPoints, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterPricingPolicy(ctx context.Context, v *domain.PricingPolicy, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO pricing_policies(id,restaurant_id,kind,name,scope,amount_kind,amount_minor,value_basis_points,application_index,requires_permission,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  restaurant_id = excluded.restaurant_id,
  kind = excluded.kind,
  name = excluded.name,
  scope = excluded.scope,
  amount_kind = excluded.amount_kind,
  amount_minor = excluded.amount_minor,
  value_basis_points = excluded.value_basis_points,
  application_index = excluded.application_index,
  requires_permission = excluded.requires_permission,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.RestaurantID, string(v.Kind), v.Name, string(v.Scope), string(v.AmountKind), v.AmountMinor, v.ValueBasisPoints, v.ApplicationIndex, v.RequiresPermission, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterDataSyncState(ctx context.Context, v *domain.MasterDataSyncState) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO cloud_master_sync_state(id,restaurant_id,node_device_id,stream_name,direction,sync_mode,checkpoint_token,last_cloud_version,last_cloud_updated_at,last_applied_at,status,last_error,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(node_device_id,stream_name) DO UPDATE SET
  restaurant_id = excluded.restaurant_id,
  direction = excluded.direction,
  sync_mode = excluded.sync_mode,
  checkpoint_token = excluded.checkpoint_token,
  last_cloud_version = excluded.last_cloud_version,
  last_cloud_updated_at = excluded.last_cloud_updated_at,
  last_applied_at = excluded.last_applied_at,
  status = excluded.status,
  last_error = excluded.last_error,
  updated_at = excluded.updated_at`,
		v.ID, nullableString(v.RestaurantID), v.NodeDeviceID, string(v.StreamName), string(v.Direction), string(v.SyncMode), nullableString(v.CheckpointToken), v.LastCloudVersion, nullableString(v.LastCloudUpdatedAt), nullableString(v.LastAppliedAt), v.Status, nullableString(v.LastError), v.CreatedAt, v.UpdatedAt)
	return normalizeErr(err)
}

func (r *Repository) GetMasterDataSyncState(ctx context.Context, nodeDeviceID string, stream domain.MasterDataStream) (*domain.MasterDataSyncState, error) {
	return scanMasterDataSyncState(r.queryer(ctx).QueryRowContext(ctx, masterDataSyncStateSelect+` FROM cloud_master_sync_state WHERE node_device_id = ? AND stream_name = ?`, nodeDeviceID, string(stream)))
}

func (r *Repository) ListMasterDataSyncStates(ctx context.Context, nodeDeviceID string) ([]domain.MasterDataSyncState, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, masterDataSyncStateSelect+` FROM cloud_master_sync_state WHERE node_device_id = ? ORDER BY stream_name`, nodeDeviceID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []domain.MasterDataSyncState
	for rows.Next() {
		v, err := scanMasterDataSyncState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

const masterDataSyncStateSelect = `SELECT id,restaurant_id,node_device_id,stream_name,direction,sync_mode,checkpoint_token,last_cloud_version,last_cloud_updated_at,last_applied_at,status,last_error,created_at,updated_at`

type masterDataSyncStateScanner interface {
	Scan(...any) error
}

func scanMasterDataSyncState(row masterDataSyncStateScanner) (*domain.MasterDataSyncState, error) {
	var v domain.MasterDataSyncState
	var restaurantID, checkpointToken, lastCloudUpdatedAt, lastAppliedAt, lastError sql.NullString
	var stream, direction, syncMode string
	if err := row.Scan(&v.ID, &restaurantID, &v.NodeDeviceID, &stream, &direction, &syncMode, &checkpointToken, &v.LastCloudVersion, &lastCloudUpdatedAt, &lastAppliedAt, &v.Status, &lastError, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, normalizeErr(err)
	}
	v.RestaurantID = stringPtr(restaurantID)
	v.StreamName = domain.MasterDataStream(stream)
	v.Direction = domain.SyncDirection(direction)
	v.SyncMode = domain.SyncMode(syncMode)
	v.CheckpointToken = stringPtr(checkpointToken)
	v.LastCloudUpdatedAt = stringPtr(lastCloudUpdatedAt)
	v.LastAppliedAt = stringPtr(lastAppliedAt)
	v.LastError = stringPtr(lastError)
	return &v, nil
}
