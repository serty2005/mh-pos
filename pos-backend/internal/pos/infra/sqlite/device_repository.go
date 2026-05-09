package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateDevice(ctx context.Context, v *domain.Device) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO devices(id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceCode, v.Name, v.Type, boolInt(v.Active), dbTime(v.RegisteredAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetDevice(ctx context.Context, id string) (*domain.Device, error) {
	var v domain.Device
	var active int
	var registered, created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at FROM devices WHERE id = ?`, id).
		Scan(&v.ID, &v.RestaurantID, &v.DeviceCode, &v.Name, &v.Type, &active, &registered, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.RegisteredAt = parseTime(registered)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) ListDevices(ctx context.Context) ([]domain.Device, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at FROM devices ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Device
	for rows.Next() {
		var v domain.Device
		var active int
		var registered, created, updated string
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.DeviceCode, &v.Name, &v.Type, &active, &registered, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.RegisteredAt = parseTime(registered)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertEdgeNodeIdentity(ctx context.Context, v *domain.EdgeNodeIdentity) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO edge_node_identity(id,node_device_id,restaurant_id,status,pairing_code_hash,paired_at,created_at,updated_at)
VALUES (?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET node_device_id = excluded.node_device_id, restaurant_id = excluded.restaurant_id, status = excluded.status, pairing_code_hash = excluded.pairing_code_hash, paired_at = excluded.paired_at, updated_at = excluded.updated_at`,
		v.ID, v.NodeDeviceID, v.RestaurantID, string(v.Status), v.PairingCodeHash, dbTime(v.PairedAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetEdgeNodeIdentity(ctx context.Context) (*domain.EdgeNodeIdentity, error) {
	var v domain.EdgeNodeIdentity
	var status, paired, created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,node_device_id,restaurant_id,status,pairing_code_hash,paired_at,created_at,updated_at FROM edge_node_identity WHERE id = 'local'`).
		Scan(&v.ID, &v.NodeDeviceID, &v.RestaurantID, &status, &v.PairingCodeHash, &paired, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.EdgeNodeStatus(status)
	v.PairedAt = parseTime(paired)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) UpsertEdgeProvisioningState(ctx context.Context, v *domain.EdgeProvisioningState) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO edge_provisioning_state(id,node_device_id,cloud_url,license_url,restaurant_id,status,credentials_type,credentials_token,last_error,created_at,updated_at)
VALUES ('local',?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  node_device_id = excluded.node_device_id,
  cloud_url = excluded.cloud_url,
  license_url = excluded.license_url,
  restaurant_id = excluded.restaurant_id,
  status = excluded.status,
  credentials_type = excluded.credentials_type,
  credentials_token = excluded.credentials_token,
  last_error = excluded.last_error,
  updated_at = excluded.updated_at`,
		v.NodeDeviceID, nullableStringValue(v.CloudURL), nullableStringValue(v.LicenseURL), nullableStringValue(v.RestaurantID), string(v.Status), nullableStringValue(v.CredentialsType), nullableStringValue(v.CredentialsToken), nullableStringValue(v.LastError), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetEdgeProvisioningState(ctx context.Context) (*domain.EdgeProvisioningState, error) {
	var v domain.EdgeProvisioningState
	var cloudURL, licenseURL, restaurantID, credentialsType, credentialsToken, lastError sql.NullString
	var status, created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,node_device_id,cloud_url,license_url,restaurant_id,status,credentials_type,credentials_token,last_error,created_at,updated_at FROM edge_provisioning_state WHERE id = 'local'`).
		Scan(&v.ID, &v.NodeDeviceID, &cloudURL, &licenseURL, &restaurantID, &status, &credentialsType, &credentialsToken, &lastError, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.CloudURL = stringFromNull(cloudURL)
	v.LicenseURL = stringFromNull(licenseURL)
	v.RestaurantID = stringFromNull(restaurantID)
	v.Status = domain.ProvisioningStatus(status)
	v.CredentialsType = stringFromNull(credentialsType)
	v.CredentialsToken = stringFromNull(credentialsToken)
	v.LastError = stringFromNull(lastError)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) CreateClientDevice(ctx context.Context, v *domain.ClientDevice) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO client_devices(id,restaurant_id,node_device_id,client_device_id,status,first_seen_at,last_seen_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.NodeDeviceID, v.ClientDeviceID, string(v.Status), dbTime(v.FirstSeenAt), dbTime(v.LastSeenAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetClientDevice(ctx context.Context, nodeDeviceID, clientDeviceID string) (*domain.ClientDevice, error) {
	var v domain.ClientDevice
	var status, firstSeen, lastSeen, created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,node_device_id,client_device_id,status,first_seen_at,last_seen_at,created_at,updated_at FROM client_devices WHERE node_device_id = ? AND client_device_id = ?`, nodeDeviceID, clientDeviceID).
		Scan(&v.ID, &v.RestaurantID, &v.NodeDeviceID, &v.ClientDeviceID, &status, &firstSeen, &lastSeen, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.ClientDeviceStatus(status)
	v.FirstSeenAt = parseTime(firstSeen)
	v.LastSeenAt = parseTime(lastSeen)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) TouchClientDevice(ctx context.Context, nodeDeviceID, clientDeviceID, seenAt string) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE client_devices SET last_seen_at = ?, updated_at = ? WHERE node_device_id = ? AND client_device_id = ? AND status = 'active'`, seenAt, seenAt, nodeDeviceID, clientDeviceID)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func nullableStringValue(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func stringFromNull(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}
