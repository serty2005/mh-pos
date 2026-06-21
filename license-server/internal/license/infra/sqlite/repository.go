package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"license-server/internal/license/app"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS pairing_codes (
  pairing_code_hash TEXT PRIMARY KEY,
  pairing_id TEXT NOT NULL DEFAULT '',
  instance_id TEXT NOT NULL DEFAULT '',
  cloud_url TEXT NOT NULL,
  restaurant_id TEXT NOT NULL,
  node_device_id TEXT NOT NULL DEFAULT '',
  credentials_json TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(credentials_json)),
  expires_at TEXT NOT NULL,
  consumed_at TEXT,
  created_at TEXT NOT NULL
)`)
	if err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS entitlement_snapshots (
  tenant_id TEXT NOT NULL,
  server_id TEXT NOT NULL,
  version INTEGER NOT NULL CHECK (version > 0),
  status TEXT NOT NULL CHECK (status IN ('active','revoked')),
  entitlements_json TEXT NOT NULL CHECK (json_valid(entitlements_json)),
  issued_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (tenant_id, server_id)
)`); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS db_runtime_versions (
  module_name TEXT PRIMARY KEY,
  current_version TEXT NOT NULL,
  updated_at TEXT NOT NULL
); INSERT INTO db_runtime_versions(module_name,current_version,updated_at) VALUES ('license-server','0.1.0',?)
ON CONFLICT(module_name) DO UPDATE SET current_version=excluded.current_version,updated_at=excluded.updated_at`, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	for _, stmt := range []string{
		`ALTER TABLE pairing_codes ADD COLUMN pairing_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pairing_codes ADD COLUMN instance_id TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	return r.verifySchema(ctx)
}

func (r *Repository) verifySchema(ctx context.Context) error {
	for table, required := range map[string][]string{
		"pairing_codes":         {"pairing_code_hash", "pairing_id", "instance_id", "expires_at"},
		"entitlement_snapshots": {"tenant_id", "server_id", "version", "status", "entitlements_json", "expires_at"},
		"db_runtime_versions":   {"module_name", "current_version", "updated_at"},
	} {
		rows, err := r.db.QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
		if err != nil {
			return err
		}
		found := map[string]bool{}
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				rows.Close()
				return err
			}
			found[name] = true
		}
		rows.Close()
		for _, column := range required {
			if !found[column] {
				return fmt.Errorf("license schema verification failed: %s.%s missing", table, column)
			}
		}
	}
	return nil
}

func (r *Repository) SaveEntitlements(ctx context.Context, v app.EntitlementSnapshot) error {
	entitlements, err := json.Marshal(v.Entitlements)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO entitlement_snapshots(tenant_id,server_id,version,status,entitlements_json,issued_at,expires_at,updated_at)
VALUES (?,?,?,?,?,?,?,?)
ON CONFLICT(tenant_id,server_id) DO UPDATE SET version=excluded.version,status=excluded.status,entitlements_json=excluded.entitlements_json,issued_at=excluded.issued_at,expires_at=excluded.expires_at,updated_at=excluded.updated_at`,
		v.TenantID, v.ServerID, v.Version, v.Status, string(entitlements), v.IssuedAt.Format(time.RFC3339Nano), v.ExpiresAt.Format(time.RFC3339Nano), v.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (r *Repository) GetEntitlements(ctx context.Context, tenantID, serverID string) (app.EntitlementSnapshot, error) {
	var v app.EntitlementSnapshot
	var raw, issued, expires, updated string
	err := r.db.QueryRowContext(ctx, `SELECT tenant_id,server_id,version,status,entitlements_json,issued_at,expires_at,updated_at FROM entitlement_snapshots WHERE tenant_id=? AND server_id=?`, tenantID, serverID).
		Scan(&v.TenantID, &v.ServerID, &v.Version, &v.Status, &raw, &issued, &expires, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return app.EntitlementSnapshot{}, app.ErrEntitlementNotFound
	}
	if err != nil {
		return app.EntitlementSnapshot{}, err
	}
	if err := json.Unmarshal([]byte(raw), &v.Entitlements); err != nil {
		return app.EntitlementSnapshot{}, err
	}
	v.IssuedAt, _ = time.Parse(time.RFC3339Nano, issued)
	v.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return v, nil
}

func (r *Repository) ListEntitlements(ctx context.Context) ([]app.EntitlementSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id,server_id,version,status,entitlements_json,issued_at,expires_at,updated_at FROM entitlement_snapshots ORDER BY tenant_id, server_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []app.EntitlementSnapshot{}
	for rows.Next() {
		var v app.EntitlementSnapshot
		var raw, issued, expires, updated string
		if err := rows.Scan(&v.TenantID, &v.ServerID, &v.Version, &v.Status, &raw, &issued, &expires, &updated); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &v.Entitlements); err != nil {
			return nil, err
		}
		v.IssuedAt, _ = time.Parse(time.RFC3339Nano, issued)
		v.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
		v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		items = append(items, v)
	}
	return items, rows.Err()
}

func (r *Repository) Save(ctx context.Context, v app.PairingCode) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO pairing_codes(pairing_code_hash,pairing_id,instance_id,cloud_url,restaurant_id,node_device_id,credentials_json,expires_at,created_at)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT(pairing_code_hash) DO UPDATE SET
  pairing_id = excluded.pairing_id,
  instance_id = excluded.instance_id,
  cloud_url = excluded.cloud_url,
  restaurant_id = excluded.restaurant_id,
  node_device_id = excluded.node_device_id,
  credentials_json = excluded.credentials_json,
  expires_at = excluded.expires_at,
  consumed_at = NULL,
  created_at = excluded.created_at`,
		v.PairingCodeHash, v.PairingID, v.InstanceID, v.CloudURL, v.RestaurantID, v.NodeDeviceID, `{}`, v.ExpiresAt.Format(time.RFC3339Nano), v.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (r *Repository) GetByHash(ctx context.Context, hash string) (app.PairingCode, error) {
	var v app.PairingCode
	var credentials, expiresAt, createdAt string
	var consumedAt sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT pairing_code_hash,pairing_id,instance_id,cloud_url,restaurant_id,node_device_id,credentials_json,expires_at,consumed_at,created_at FROM pairing_codes WHERE pairing_code_hash = ?`, hash).
		Scan(&v.PairingCodeHash, &v.PairingID, &v.InstanceID, &v.CloudURL, &v.RestaurantID, &v.NodeDeviceID, &credentials, &expiresAt, &consumedAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return app.PairingCode{}, app.ErrInvalid
	}
	if err != nil {
		return app.PairingCode{}, err
	}
	_ = credentials
	v.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expiresAt)
	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	if consumedAt.Valid {
		parsed, _ := time.Parse(time.RFC3339Nano, consumedAt.String)
		v.ConsumedAt = &parsed
	}
	return v, nil
}

func (r *Repository) MarkConsumed(ctx context.Context, hash string, consumedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE pairing_codes SET consumed_at = ? WHERE pairing_code_hash = ?`, consumedAt.Format(time.RFC3339Nano), hash)
	return err
}
