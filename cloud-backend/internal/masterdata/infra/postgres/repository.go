package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
)

// Repository реализует Cloud master-data persistence port поверх PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository создает PostgreSQL repository для master-data authority.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateRestaurant(ctx context.Context, v domain.Restaurant) (domain.Restaurant, error) {
	out, err := scanRestaurant(r.pool.QueryRow(ctx, `
INSERT INTO cloud_restaurants(id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,status,cloud_version,archived_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.Name, v.Timezone, v.Currency, v.BusinessDayMode, v.BusinessDayBoundaryLocalTime, v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateRestaurant(ctx context.Context, v domain.Restaurant) (domain.Restaurant, error) {
	out, err := scanRestaurant(r.pool.QueryRow(ctx, `
UPDATE cloud_restaurants
SET name=$2,timezone=$3,currency=$4,business_day_mode=$5,business_day_boundary_local_time=$6,status=$7,cloud_version=$8,archived_at=$9,updated_at=$10
WHERE id=$1
RETURNING id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.Name, v.Timezone, v.Currency, v.BusinessDayMode, v.BusinessDayBoundaryLocalTime, v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetRestaurant(ctx context.Context, id string) (domain.Restaurant, error) {
	v, err := scanRestaurant(r.pool.QueryRow(ctx, `SELECT id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,status,cloud_version,archived_at,created_at,updated_at FROM cloud_restaurants WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListRestaurants(ctx context.Context) ([]domain.Restaurant, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,status,cloud_version,archived_at,created_at,updated_at FROM cloud_restaurants ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Restaurant
	for rows.Next() {
		v, err := scanRestaurant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateRole(ctx context.Context, v domain.Role) (domain.Role, error) {
	out, err := scanRole(r.pool.QueryRow(ctx, `
INSERT INTO cloud_roles(id,restaurant_id,name,permissions_json,active,cloud_version,archived_at,created_at,updated_at)
VALUES ($1,$2,$3,$4::jsonb,$5,$6,$7,$8,$9)
RETURNING id,restaurant_id,name,permissions_json::text,active,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.RestaurantID, v.Name, v.PermissionsJSON, v.Active, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateRole(ctx context.Context, v domain.Role) (domain.Role, error) {
	out, err := scanRole(r.pool.QueryRow(ctx, `
UPDATE cloud_roles
SET name=$2,permissions_json=$3::jsonb,active=$4,cloud_version=$5,archived_at=$6,updated_at=$7
WHERE id=$1
RETURNING id,restaurant_id,name,permissions_json::text,active,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.Name, v.PermissionsJSON, v.Active, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetRole(ctx context.Context, id string) (domain.Role, error) {
	v, err := scanRole(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,name,permissions_json::text,active,cloud_version,archived_at,created_at,updated_at FROM cloud_roles WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListRoles(ctx context.Context, restaurantID string) ([]domain.Role, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,name,permissions_json::text,active,cloud_version,archived_at,created_at,updated_at FROM cloud_roles WHERE restaurant_id = $1 ORDER BY id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Role
	for rows.Next() {
		v, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateEmployee(ctx context.Context, v domain.Employee) (domain.Employee, error) {
	out, err := scanEmployee(r.pool.QueryRow(ctx, `
INSERT INTO cloud_employees(id,restaurant_id,role_id,name,status,pin_hash,pin_credential_version,permission_snapshot_json,cloud_version,suspended_at,archived_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,$12,$13)
RETURNING id,restaurant_id,role_id,name,status,pin_hash,pin_credential_version,permission_snapshot_json::text,cloud_version,suspended_at,archived_at,created_at,updated_at`,
		v.ID, v.RestaurantID, v.RoleID, v.Name, v.Status, v.PINHash, v.PINCredentialVersion, v.PermissionSnapshotJSON, v.CloudVersion, v.SuspendedAt, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateEmployee(ctx context.Context, v domain.Employee) (domain.Employee, error) {
	out, err := scanEmployee(r.pool.QueryRow(ctx, `
UPDATE cloud_employees
SET role_id=$2,name=$3,status=$4,pin_hash=$5,pin_credential_version=$6,permission_snapshot_json=$7::jsonb,cloud_version=$8,suspended_at=$9,archived_at=$10,updated_at=$11
WHERE id=$1
RETURNING id,restaurant_id,role_id,name,status,pin_hash,pin_credential_version,permission_snapshot_json::text,cloud_version,suspended_at,archived_at,created_at,updated_at`,
		v.ID, v.RoleID, v.Name, v.Status, v.PINHash, v.PINCredentialVersion, v.PermissionSnapshotJSON, v.CloudVersion, statusTime(v.Status, domain.EmployeeSuspended, v.SuspendedAt, v.UpdatedAt), v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetEmployee(ctx context.Context, id string) (domain.Employee, error) {
	v, err := scanEmployee(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,role_id,name,status,pin_hash,pin_credential_version,permission_snapshot_json::text,cloud_version,suspended_at,archived_at,created_at,updated_at FROM cloud_employees WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListEmployees(ctx context.Context, restaurantID string) ([]domain.Employee, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,role_id,name,status,pin_hash,pin_credential_version,permission_snapshot_json::text,cloud_version,suspended_at,archived_at,created_at,updated_at FROM cloud_employees WHERE restaurant_id = $1 ORDER BY id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Employee
	for rows.Next() {
		v, err := scanEmployee(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCatalogItem(ctx context.Context, v domain.CatalogItem) (domain.CatalogItem, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.CatalogItem{}, err
	}
	defer tx.Rollback(ctx)
	out, err := scanCatalogItem(tx.QueryRow(ctx, `
INSERT INTO cloud_catalog_items(id,restaurant_id,kind,name,sku,base_unit,status,cloud_version,archived_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING id,restaurant_id,kind,name,sku,base_unit,status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.RestaurantID, v.Kind, v.Name, v.SKU, v.BaseUnit, v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	if err != nil {
		return domain.CatalogItem{}, normalizeErr(err)
	}
	if err := upsertKindFoundation(ctx, tx, out); err != nil {
		return domain.CatalogItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.CatalogItem{}, err
	}
	return out, nil
}

func (r *Repository) UpdateCatalogItem(ctx context.Context, v domain.CatalogItem) (domain.CatalogItem, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.CatalogItem{}, err
	}
	defer tx.Rollback(ctx)
	out, err := scanCatalogItem(tx.QueryRow(ctx, `
UPDATE cloud_catalog_items
SET kind=$2,name=$3,sku=$4,base_unit=$5,status=$6,cloud_version=$7,archived_at=$8,updated_at=$9
WHERE id=$1
RETURNING id,restaurant_id,kind,name,sku,base_unit,status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.Kind, v.Name, v.SKU, v.BaseUnit, v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	if err != nil {
		return domain.CatalogItem{}, normalizeErr(err)
	}
	if err := upsertKindFoundation(ctx, tx, out); err != nil {
		return domain.CatalogItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.CatalogItem{}, err
	}
	return out, nil
}

func (r *Repository) GetCatalogItem(ctx context.Context, id string) (domain.CatalogItem, error) {
	v, err := scanCatalogItem(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,kind,name,sku,base_unit,status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_items WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListCatalogItems(ctx context.Context, restaurantID string) ([]domain.CatalogItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,kind,name,sku,base_unit,status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_items WHERE restaurant_id = $1 ORDER BY id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CatalogItem
	for rows.Next() {
		v, err := scanCatalogItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCategory(ctx context.Context, v domain.Category) (domain.Category, error) {
	out, err := scanCategory(r.pool.QueryRow(ctx, `
INSERT INTO cloud_categories(id,restaurant_id,name,status,sort_order,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id,restaurant_id,name,status,sort_order,created_at,updated_at`,
		v.ID, v.RestaurantID, v.Name, v.Status, v.SortOrder, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) ListCategories(ctx context.Context, restaurantID string) ([]domain.Category, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,name,status,sort_order,created_at,updated_at FROM cloud_categories WHERE restaurant_id = $1 ORDER BY sort_order,id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Category
	for rows.Next() {
		v, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateHall(ctx context.Context, v domain.Hall) (domain.Hall, error) {
	out, err := scanHall(r.pool.QueryRow(ctx, `
INSERT INTO cloud_halls(id,restaurant_id,name,status,cloud_version,archived_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING id,restaurant_id,name,status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.RestaurantID, v.Name, v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateHall(ctx context.Context, v domain.Hall) (domain.Hall, error) {
	out, err := scanHall(r.pool.QueryRow(ctx, `
UPDATE cloud_halls
SET name=$2,status=$3,cloud_version=$4,archived_at=$5,updated_at=$6
WHERE id=$1
RETURNING id,restaurant_id,name,status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.Name, v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetHall(ctx context.Context, id string) (domain.Hall, error) {
	v, err := scanHall(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,name,status,cloud_version,archived_at,created_at,updated_at FROM cloud_halls WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListHalls(ctx context.Context, restaurantID string) ([]domain.Hall, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,name,status,cloud_version,archived_at,created_at,updated_at FROM cloud_halls WHERE restaurant_id = $1 ORDER BY id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Hall
	for rows.Next() {
		v, err := scanHall(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateTable(ctx context.Context, v domain.Table) (domain.Table, error) {
	out, err := scanTable(r.pool.QueryRow(ctx, `
INSERT INTO cloud_tables(id,restaurant_id,hall_id,name,seats,status,cloud_version,archived_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id,restaurant_id,hall_id,name,seats,status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.RestaurantID, v.HallID, v.Name, v.Seats, v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateTable(ctx context.Context, v domain.Table) (domain.Table, error) {
	out, err := scanTable(r.pool.QueryRow(ctx, `
UPDATE cloud_tables
SET hall_id=$2,name=$3,seats=$4,status=$5,cloud_version=$6,archived_at=$7,updated_at=$8
WHERE id=$1
RETURNING id,restaurant_id,hall_id,name,seats,status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.HallID, v.Name, v.Seats, v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetTable(ctx context.Context, id string) (domain.Table, error) {
	v, err := scanTable(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,hall_id,name,seats,status,cloud_version,archived_at,created_at,updated_at FROM cloud_tables WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListTables(ctx context.Context, restaurantID string) ([]domain.Table, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,hall_id,name,seats,status,cloud_version,archived_at,created_at,updated_at FROM cloud_tables WHERE restaurant_id = $1 ORDER BY hall_id,id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Table
	for rows.Next() {
		v, err := scanTable(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateMenuItem(ctx context.Context, v domain.MenuItem) (domain.MenuItem, error) {
	out, err := scanMenuItem(r.pool.QueryRow(ctx, `
INSERT INTO cloud_menu_items(id,restaurant_id,catalog_item_id,category_id,name,price,currency,status,availability_json,station_routing_key,cloud_version,archived_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13,$14)
RETURNING id,restaurant_id,catalog_item_id,COALESCE(category_id,''),name,price,currency,status,availability_json::text,COALESCE(station_routing_key,''),cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.RestaurantID, v.CatalogItemID, nullableText(v.CategoryID), v.Name, v.Price, v.Currency, v.Status, v.AvailabilityJSON, nullableText(v.StationRoutingKey), v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateMenuItem(ctx context.Context, v domain.MenuItem) (domain.MenuItem, error) {
	out, err := scanMenuItem(r.pool.QueryRow(ctx, `
UPDATE cloud_menu_items
SET catalog_item_id=$2,category_id=$3,name=$4,price=$5,currency=$6,status=$7,availability_json=$8::jsonb,station_routing_key=$9,cloud_version=$10,archived_at=$11,updated_at=$12
WHERE id=$1
RETURNING id,restaurant_id,catalog_item_id,COALESCE(category_id,''),name,price,currency,status,availability_json::text,COALESCE(station_routing_key,''),cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.CatalogItemID, nullableText(v.CategoryID), v.Name, v.Price, v.Currency, v.Status, v.AvailabilityJSON, nullableText(v.StationRoutingKey), v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetMenuItem(ctx context.Context, id string) (domain.MenuItem, error) {
	v, err := scanMenuItem(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,catalog_item_id,COALESCE(category_id,''),name,price,currency,status,availability_json::text,COALESCE(station_routing_key,''),cloud_version,archived_at,created_at,updated_at FROM cloud_menu_items WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListMenuItems(ctx context.Context, restaurantID string) ([]domain.MenuItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,catalog_item_id,COALESCE(category_id,''),name,price,currency,status,availability_json::text,COALESCE(station_routing_key,''),cloud_version,archived_at,created_at,updated_at FROM cloud_menu_items WHERE restaurant_id = $1 ORDER BY id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MenuItem
	for rows.Next() {
		v, err := scanMenuItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) NextPublicationVersion(ctx context.Context, restaurantID string) (int64, error) {
	var version int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(version),0) + 1 FROM cloud_master_data_publications WHERE restaurant_id = $1`, strings.TrimSpace(restaurantID)).Scan(&version)
	return version, err
}

func (r *Repository) SavePublication(ctx context.Context, pub domain.Publication, packages []app.StreamPackage) (domain.Publication, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Publication{}, err
	}
	defer tx.Rollback(ctx)
	stored, err := scanPublication(tx.QueryRow(ctx, `
INSERT INTO cloud_master_data_publications(id,restaurant_id,version,status,cloud_version,published_at,published_by,package_json,package_sha256,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11)
RETURNING id,restaurant_id,version,status,cloud_version,published_at,published_by,package_json,package_sha256,created_at,updated_at`,
		pub.ID, pub.RestaurantID, pub.Version, pub.Status, pub.CloudVersion, pub.PublishedAt, pub.PublishedBy, string(pub.PackageJSON), pub.PackageSHA256, pub.CreatedAt, pub.UpdatedAt))
	if err != nil {
		return domain.Publication{}, err
	}
	for _, pkg := range packages {
		if _, err := tx.Exec(ctx, `
INSERT INTO cloud_master_data_packages(stream_name,node_device_id,restaurant_id,sync_mode,full_snapshot_reason,cloud_version,checkpoint_token,cloud_updated_at,payload_json,created_at,updated_at)
VALUES ($1,$2,$3,$4,'',$5,$6,$7,$8::jsonb,$7,$7)
ON CONFLICT (stream_name,node_device_id) DO UPDATE SET
  restaurant_id = EXCLUDED.restaurant_id,
  sync_mode = EXCLUDED.sync_mode,
  full_snapshot_reason = '',
  cloud_version = EXCLUDED.cloud_version,
  checkpoint_token = EXCLUDED.checkpoint_token,
  cloud_updated_at = EXCLUDED.cloud_updated_at,
  payload_json = EXCLUDED.payload_json,
  updated_at = EXCLUDED.updated_at`,
			pkg.StreamName, strings.TrimSpace(pkg.NodeDeviceID), pkg.RestaurantID, pkg.SyncMode, pkg.CloudVersion, nullableText(pkg.CheckpointToken), pkg.CloudUpdatedAt, string(pkg.PayloadJSON)); err != nil {
			return domain.Publication{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Publication{}, err
	}
	return stored, nil
}

func (r *Repository) GetCurrentPublication(ctx context.Context, restaurantID string) (domain.Publication, error) {
	v, err := scanPublication(r.pool.QueryRow(ctx, `
SELECT id,restaurant_id,version,status,cloud_version,published_at,published_by,package_json,package_sha256,created_at,updated_at
FROM cloud_master_data_publications
WHERE restaurant_id = $1 AND status = 'published'
ORDER BY version DESC
LIMIT 1`, strings.TrimSpace(restaurantID)))
	return v, normalizeErr(err)
}

func (r *Repository) GetPublication(ctx context.Context, restaurantID, packageID string) (domain.Publication, error) {
	v, err := scanPublication(r.pool.QueryRow(ctx, `
SELECT id,restaurant_id,version,status,cloud_version,published_at,published_by,package_json,package_sha256,created_at,updated_at
FROM cloud_master_data_publications
WHERE restaurant_id = $1 AND id = $2
LIMIT 1`, strings.TrimSpace(restaurantID), strings.TrimSpace(packageID)))
	return v, normalizeErr(err)
}

func upsertKindFoundation(ctx context.Context, tx pgx.Tx, v domain.CatalogItem) error {
	switch v.Kind {
	case domain.CatalogItemDish:
		_, err := tx.Exec(ctx, `INSERT INTO cloud_dishes(catalog_item_id,restaurant_id,updated_at) VALUES ($1,$2,$3) ON CONFLICT (catalog_item_id) DO UPDATE SET updated_at = EXCLUDED.updated_at`, v.ID, v.RestaurantID, v.UpdatedAt)
		return err
	case domain.CatalogItemGood, domain.CatalogItemIngredient:
		_, err := tx.Exec(ctx, `INSERT INTO cloud_goods(catalog_item_id,restaurant_id,updated_at) VALUES ($1,$2,$3) ON CONFLICT (catalog_item_id) DO UPDATE SET updated_at = EXCLUDED.updated_at`, v.ID, v.RestaurantID, v.UpdatedAt)
		return err
	case domain.CatalogItemSemiFinished:
		_, err := tx.Exec(ctx, `INSERT INTO cloud_semi_finished_products(catalog_item_id,restaurant_id,updated_at) VALUES ($1,$2,$3) ON CONFLICT (catalog_item_id) DO UPDATE SET updated_at = EXCLUDED.updated_at`, v.ID, v.RestaurantID, v.UpdatedAt)
		return err
	default:
		return fmt.Errorf("%w: unsupported catalog item kind %q", domain.ErrInvalid, v.Kind)
	}
}

type scanner interface {
	Scan(...any) error
}

func scanRole(row scanner) (domain.Role, error) {
	var v domain.Role
	err := row.Scan(&v.ID, &v.RestaurantID, &v.Name, &v.PermissionsJSON, &v.Active, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

func scanRestaurant(row scanner) (domain.Restaurant, error) {
	var v domain.Restaurant
	var status string
	err := row.Scan(&v.ID, &v.Name, &v.Timezone, &v.Currency, &v.BusinessDayMode, &v.BusinessDayBoundaryLocalTime, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.RestaurantStatus(status)
	return v, err
}

func scanEmployee(row scanner) (domain.Employee, error) {
	var v domain.Employee
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.RoleID, &v.Name, &status, &v.PINHash, &v.PINCredentialVersion, &v.PermissionSnapshotJSON, &v.CloudVersion, &v.SuspendedAt, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.EmployeeStatus(status)
	v.PINConfigured = strings.TrimSpace(v.PINHash) != ""
	return v, err
}

func scanCatalogItem(row scanner) (domain.CatalogItem, error) {
	var v domain.CatalogItem
	var kind, status string
	err := row.Scan(&v.ID, &v.RestaurantID, &kind, &v.Name, &v.SKU, &v.BaseUnit, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return v, err
	}
	v.Kind = domain.CatalogItemKind(kind)
	v.Status = domain.LifecycleStatus(status)
	if err := domain.ValidateCatalogItemKind(v.Kind); err != nil {
		return v, fmt.Errorf("%w: scanned catalog item %s has unsupported kind", err, v.ID)
	}
	return v, nil
}

func scanCategory(row scanner) (domain.Category, error) {
	var v domain.Category
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.Name, &status, &v.SortOrder, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanHall(row scanner) (domain.Hall, error) {
	var v domain.Hall
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.Name, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanTable(row scanner) (domain.Table, error) {
	var v domain.Table
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.HallID, &v.Name, &v.Seats, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanMenuItem(row scanner) (domain.MenuItem, error) {
	var v domain.MenuItem
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.CatalogItemID, &v.CategoryID, &v.Name, &v.Price, &v.Currency, &status, &v.AvailabilityJSON, &v.StationRoutingKey, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanPublication(row scanner) (domain.Publication, error) {
	var v domain.Publication
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.Version, &status, &v.CloudVersion, &v.PublishedAt, &v.PublishedBy, &v.PackageJSON, &v.PackageSHA256, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	if json.Valid(v.PackageJSON) {
		v.PackageJSON = append(json.RawMessage(nil), v.PackageJSON...)
	}
	return v, err
}

func statusTime(status, target domain.EmployeeStatus, current *time.Time, now time.Time) *time.Time {
	if status != target {
		return current
	}
	if current != nil {
		return current
	}
	return &now
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
