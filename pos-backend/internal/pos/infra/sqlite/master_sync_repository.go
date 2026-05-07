package sqlite

import (
	"context"
	"database/sql"

	"pos-backend/internal/pos/domain"
)

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
