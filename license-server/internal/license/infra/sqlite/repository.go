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
	if _, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS license_servers (
  tenant_id TEXT NOT NULL,
  server_id TEXT NOT NULL,
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  PRIMARY KEY (tenant_id, server_id)
)`); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS admin_users (
  username TEXT PRIMARY KEY,
  password_hash TEXT NOT NULL,
  salt TEXT NOT NULL,
  iterations INTEGER NOT NULL CHECK (iterations > 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
)`); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS admin_sessions (
  token_hash TEXT PRIMARY KEY,
  username TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (username) REFERENCES admin_users(username) ON DELETE CASCADE
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
		"license_servers":       {"tenant_id", "server_id", "first_seen_at", "last_seen_at"},
		"admin_users":           {"username", "password_hash", "salt", "iterations"},
		"admin_sessions":        {"token_hash", "username", "expires_at"},
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO entitlement_snapshots(tenant_id,server_id,version,status,entitlements_json,issued_at,expires_at,updated_at)
VALUES (?,?,?,?,?,?,?,?)
ON CONFLICT(tenant_id,server_id) DO UPDATE SET version=excluded.version,status=excluded.status,entitlements_json=excluded.entitlements_json,issued_at=excluded.issued_at,expires_at=excluded.expires_at,updated_at=excluded.updated_at`,
		v.TenantID, v.ServerID, v.Version, v.Status, string(entitlements), v.IssuedAt.Format(time.RFC3339Nano), v.ExpiresAt.Format(time.RFC3339Nano), v.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	now := v.UpdatedAt.Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, `INSERT INTO license_servers(tenant_id,server_id,first_seen_at,last_seen_at)
VALUES (?,?,?,?)
ON CONFLICT(tenant_id,server_id) DO UPDATE SET last_seen_at=excluded.last_seen_at`,
		v.TenantID, v.ServerID, now, now); err != nil {
		return err
	}
	return tx.Commit()
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

func (r *Repository) SaveConnectedServer(ctx context.Context, v app.ConnectedServer) error {
	now := v.LastSeenAt.Format(time.RFC3339Nano)
	_, err := r.db.ExecContext(ctx, `INSERT INTO license_servers(tenant_id,server_id,first_seen_at,last_seen_at)
VALUES (?,?,?,?)
ON CONFLICT(tenant_id,server_id) DO UPDATE SET last_seen_at=excluded.last_seen_at`,
		v.TenantID, v.ServerID, v.FirstSeenAt.Format(time.RFC3339Nano), now)
	return err
}

func (r *Repository) ListConnectedServers(ctx context.Context) ([]app.ConnectedServer, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.tenant_id,s.server_id,s.first_seen_at,s.last_seen_at,
e.version,e.status,e.entitlements_json,e.issued_at,e.expires_at,e.updated_at
FROM license_servers s
LEFT JOIN entitlement_snapshots e ON e.tenant_id=s.tenant_id AND e.server_id=s.server_id
ORDER BY s.last_seen_at DESC, s.tenant_id, s.server_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []app.ConnectedServer{}
	for rows.Next() {
		var item app.ConnectedServer
		var firstSeen, lastSeen string
		var version sql.NullInt64
		var status, raw, issued, expires, updated sql.NullString
		if err := rows.Scan(&item.TenantID, &item.ServerID, &firstSeen, &lastSeen, &version, &status, &raw, &issued, &expires, &updated); err != nil {
			return nil, err
		}
		item.FirstSeenAt, _ = time.Parse(time.RFC3339Nano, firstSeen)
		item.LastSeenAt, _ = time.Parse(time.RFC3339Nano, lastSeen)
		if version.Valid {
			snapshot := app.EntitlementSnapshot{TenantID: item.TenantID, ServerID: item.ServerID, Version: version.Int64, Status: status.String}
			if err := json.Unmarshal([]byte(raw.String), &snapshot.Entitlements); err != nil {
				return nil, err
			}
			snapshot.IssuedAt, _ = time.Parse(time.RFC3339Nano, issued.String)
			snapshot.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires.String)
			snapshot.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated.String)
			item.Snapshot = &snapshot
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SaveAdminUser(ctx context.Context, v app.AdminUser) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO admin_users(username,password_hash,salt,iterations,created_at,updated_at)
VALUES (?,?,?,?,?,?)
ON CONFLICT(username) DO UPDATE SET password_hash=excluded.password_hash,salt=excluded.salt,iterations=excluded.iterations,updated_at=excluded.updated_at`,
		v.Username, v.PasswordHash, v.Salt, v.Iterations, v.CreatedAt.Format(time.RFC3339Nano), v.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (r *Repository) GetAdminUser(ctx context.Context, username string) (app.AdminUser, error) {
	var v app.AdminUser
	var created, updated string
	err := r.db.QueryRowContext(ctx, `SELECT username,password_hash,salt,iterations,created_at,updated_at FROM admin_users WHERE username=?`, username).
		Scan(&v.Username, &v.PasswordHash, &v.Salt, &v.Iterations, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return app.AdminUser{}, app.ErrAdminAuth
	}
	if err != nil {
		return app.AdminUser{}, err
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return v, nil
}

func (r *Repository) SaveAdminSession(ctx context.Context, v app.AdminSession) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO admin_sessions(token_hash,username,expires_at,created_at) VALUES (?,?,?,?)`,
		v.TokenHash, v.Username, v.ExpiresAt.Format(time.RFC3339Nano), v.CreatedAt.Format(time.RFC3339Nano))
	return err
}

func (r *Repository) GetAdminSession(ctx context.Context, tokenHash string) (app.AdminSession, error) {
	var v app.AdminSession
	var expires, created string
	err := r.db.QueryRowContext(ctx, `SELECT token_hash,username,expires_at,created_at FROM admin_sessions WHERE token_hash=?`, tokenHash).
		Scan(&v.TokenHash, &v.Username, &expires, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return app.AdminSession{}, app.ErrAdminAuth
	}
	if err != nil {
		return app.AdminSession{}, err
	}
	v.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return v, nil
}

func (r *Repository) DeleteAdminSession(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE token_hash=?`, tokenHash)
	return err
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
