package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-backend/internal/provisioning/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) RegisterUnassigned(ctx context.Context, v domain.UnassignedEdgeNode) (domain.UnassignedEdgeNode, error) {
	out, err := scanUnassigned(r.pool.QueryRow(ctx, `
INSERT INTO cloud_unassigned_edge_nodes(id,node_device_id,claimed_cloud_url,display_name,app_version,status,first_seen_at,last_seen_at,assigned_restaurant_id,assigned_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT(node_device_id) DO UPDATE SET
  claimed_cloud_url = excluded.claimed_cloud_url,
  display_name = excluded.display_name,
  app_version = excluded.app_version,
  last_seen_at = excluded.last_seen_at,
  updated_at = excluded.updated_at
RETURNING id,node_device_id,claimed_cloud_url,display_name,app_version,status,first_seen_at,last_seen_at,COALESCE(assigned_restaurant_id,''),assigned_at,created_at,updated_at`,
		v.ID, v.NodeDeviceID, v.ClaimedCloudURL, v.DisplayName, v.AppVersion, v.Status, v.FirstSeenAt, v.LastSeenAt, nullableText(v.AssignedRestaurantID), v.AssignedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) ListUnassigned(ctx context.Context) ([]domain.UnassignedEdgeNode, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,node_device_id,claimed_cloud_url,display_name,app_version,status,first_seen_at,last_seen_at,COALESCE(assigned_restaurant_id,''),assigned_at,created_at,updated_at FROM cloud_unassigned_edge_nodes WHERE status = 'pending' ORDER BY last_seen_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.UnassignedEdgeNode
	for rows.Next() {
		v, err := scanUnassigned(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) ListEdgeNodesByRestaurant(ctx context.Context, restaurantID string) ([]domain.EdgeNode, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,COALESCE(restaurant_id,''),node_device_id,display_name,status,COALESCE(credentials_hash,''),last_seen_at,assigned_at,revoked_at,created_at,updated_at FROM cloud_edge_nodes WHERE restaurant_id = $1 ORDER BY updated_at DESC`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.EdgeNode
	for rows.Next() {
		v, err := scanEdgeNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertEdgeNode(ctx context.Context, v domain.EdgeNode) (domain.EdgeNode, error) {
	id := strings.TrimSpace(v.ID)
	if id == "" {
		id = strings.TrimSpace(v.NodeDeviceID)
	}
	out, err := scanEdgeNode(r.pool.QueryRow(ctx, `
INSERT INTO cloud_edge_nodes(id,restaurant_id,node_device_id,display_name,status,credentials_hash,last_seen_at,assigned_at,revoked_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT(node_device_id) DO UPDATE SET
  restaurant_id = excluded.restaurant_id,
  display_name = excluded.display_name,
  status = excluded.status,
  credentials_hash = COALESCE(excluded.credentials_hash, cloud_edge_nodes.credentials_hash),
  last_seen_at = excluded.last_seen_at,
  assigned_at = COALESCE(excluded.assigned_at, cloud_edge_nodes.assigned_at),
  revoked_at = excluded.revoked_at,
  updated_at = excluded.updated_at
RETURNING id,COALESCE(restaurant_id,''),node_device_id,display_name,status,COALESCE(credentials_hash,''),last_seen_at,assigned_at,revoked_at,created_at,updated_at`,
		id, nullableText(v.RestaurantID), v.NodeDeviceID, v.DisplayName, v.Status, nullableText(v.CredentialsHash), v.LastSeenAt, v.AssignedAt, v.RevokedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetEdgeNode(ctx context.Context, nodeDeviceID string) (domain.EdgeNode, error) {
	v, err := scanEdgeNode(r.pool.QueryRow(ctx, `SELECT id,COALESCE(restaurant_id,''),node_device_id,display_name,status,COALESCE(credentials_hash,''),last_seen_at,assigned_at,revoked_at,created_at,updated_at FROM cloud_edge_nodes WHERE node_device_id = $1`, strings.TrimSpace(nodeDeviceID)))
	return v, normalizeErr(err)
}

func (r *Repository) MarkUnassignedAssigned(ctx context.Context, nodeDeviceID, restaurantID string, assignedAt time.Time) error {
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	restaurantID = strings.TrimSpace(restaurantID)
	tag, err := r.pool.Exec(ctx, `
UPDATE cloud_unassigned_edge_nodes
SET status = 'assigned',
    assigned_restaurant_id = $2,
    assigned_at = CASE WHEN status = 'pending' THEN $3 ELSE assigned_at END,
    updated_at = CASE WHEN status = 'pending' THEN $3 ELSE updated_at END
WHERE node_device_id = $1
  AND (status = 'pending' OR (status = 'assigned' AND assigned_restaurant_id = $2))`,
		nodeDeviceID, restaurantID, assignedAt)
	if err != nil {
		return normalizeErr(err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cloud_unassigned_edge_nodes WHERE node_device_id = $1)`, nodeDeviceID).Scan(&exists); err != nil {
		return normalizeErr(err)
	}
	if !exists {
		return domain.ErrNotFound
	}
	return domain.ErrConflict
}

func (r *Repository) CreatePairingCode(ctx context.Context, v domain.PairingCode) (domain.PairingCode, error) {
	out, err := scanPairingCode(r.pool.QueryRow(ctx, `
INSERT INTO cloud_pairing_codes(id,pairing_code_hash,pairing_key,restaurant_id,node_device_id,cloud_url,status,expires_at,consumed_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING id,pairing_code_hash,pairing_key,restaurant_id,COALESCE(node_device_id,''),cloud_url,status,expires_at,consumed_at,created_at,updated_at`,
		v.ID, v.PairingCodeHash, v.PairingKey, v.RestaurantID, nullableText(v.NodeDeviceID), v.CloudURL, v.Status, v.ExpiresAt, v.ConsumedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) RevokeActivePairingCodes(ctx context.Context, restaurantID string, now time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE cloud_pairing_codes SET status = 'revoked', updated_at = $2 WHERE restaurant_id = $1 AND status = 'active'`, strings.TrimSpace(restaurantID), now)
	return normalizeErr(err)
}

func (r *Repository) GetPairingCode(ctx context.Context, pairingID string) (domain.PairingCode, error) {
	v, err := scanPairingCode(r.pool.QueryRow(ctx, `SELECT id,pairing_code_hash,pairing_key,restaurant_id,COALESCE(node_device_id,''),cloud_url,status,expires_at,consumed_at,created_at,updated_at FROM cloud_pairing_codes WHERE id = $1`, strings.TrimSpace(pairingID)))
	return v, normalizeErr(err)
}

func (r *Repository) ConsumePairingCode(ctx context.Context, pairingID, nodeDeviceID string, consumedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE cloud_pairing_codes SET status = 'consumed', node_device_id = $2, consumed_at = $3, updated_at = $3 WHERE id = $1 AND status = 'active'`, strings.TrimSpace(pairingID), strings.TrimSpace(nodeDeviceID), consumedAt)
	if err != nil {
		return normalizeErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}

type scanner interface {
	Scan(...any) error
}

func scanUnassigned(row scanner) (domain.UnassignedEdgeNode, error) {
	var v domain.UnassignedEdgeNode
	var status string
	err := row.Scan(&v.ID, &v.NodeDeviceID, &v.ClaimedCloudURL, &v.DisplayName, &v.AppVersion, &status, &v.FirstSeenAt, &v.LastSeenAt, &v.AssignedRestaurantID, &v.AssignedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.UnassignedNodeStatus(status)
	return v, err
}

func scanEdgeNode(row scanner) (domain.EdgeNode, error) {
	var v domain.EdgeNode
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.NodeDeviceID, &v.DisplayName, &status, &v.CredentialsHash, &v.LastSeenAt, &v.AssignedAt, &v.RevokedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.EdgeNodeStatus(status)
	return v, err
}

func scanPairingCode(row scanner) (domain.PairingCode, error) {
	var v domain.PairingCode
	var status string
	err := row.Scan(&v.ID, &v.PairingCodeHash, &v.PairingKey, &v.RestaurantID, &v.NodeDeviceID, &v.CloudURL, &status, &v.ExpiresAt, &v.ConsumedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.PairingCodeStatus(status)
	return v, err
}

func nullableText(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func normalizeErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrConflict
	}
	return err
}
