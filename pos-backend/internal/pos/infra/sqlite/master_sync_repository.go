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
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO catalog_items(id,type,name,sku,base_unit,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type = excluded.type,
  name = excluded.name,
  sku = excluded.sku,
  base_unit = excluded.base_unit,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, string(v.Type), v.Name, v.SKU, v.BaseUnit, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterMenuItem(ctx context.Context, v *domain.MenuItem, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO menu_items(id,catalog_item_id,name,price,currency,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  catalog_item_id = excluded.catalog_item_id,
  name = excluded.name,
  price = excluded.price,
  currency = excluded.currency,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.CatalogItemID, v.Name, v.Price, v.Currency, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
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
