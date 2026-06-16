package sqlite

import (
	"context"
	"database/sql"
	"errors"
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
	for _, stmt := range []string{
		`ALTER TABLE pairing_codes ADD COLUMN pairing_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE pairing_codes ADD COLUMN instance_id TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := r.db.ExecContext(ctx, stmt); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	return nil
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
