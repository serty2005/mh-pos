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
INSERT INTO cloud_catalog_items(id,restaurant_id,kind,folder_id,name,sku,base_unit,kitchen_type,accounting_category,status,cloud_version,archived_at,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
RETURNING id,restaurant_id,kind,COALESCE(folder_id,''),name,sku,base_unit,COALESCE(kitchen_type,''),COALESCE(accounting_category,''),status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.RestaurantID, v.Kind, nullableText(v.FolderID), v.Name, v.SKU, v.BaseUnit, nullableText(v.KitchenType), nullableText(v.AccountingCategory), v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
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
SET kind=$2,folder_id=$3,name=$4,sku=$5,base_unit=$6,kitchen_type=$7,accounting_category=$8,status=$9,cloud_version=$10,archived_at=$11,updated_at=$12
WHERE id=$1
RETURNING id,restaurant_id,kind,COALESCE(folder_id,''),name,sku,base_unit,COALESCE(kitchen_type,''),COALESCE(accounting_category,''),status,cloud_version,archived_at,created_at,updated_at`,
		v.ID, v.Kind, nullableText(v.FolderID), v.Name, v.SKU, v.BaseUnit, nullableText(v.KitchenType), nullableText(v.AccountingCategory), v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
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
	v, err := scanCatalogItem(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,kind,COALESCE(folder_id,''),name,sku,base_unit,COALESCE(kitchen_type,''),COALESCE(accounting_category,''),status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_items WHERE id = $1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListCatalogItems(ctx context.Context, restaurantID string) ([]domain.CatalogItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,kind,COALESCE(folder_id,''),name,sku,base_unit,COALESCE(kitchen_type,''),COALESCE(accounting_category,''),status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_items WHERE restaurant_id = $1 ORDER BY id`, strings.TrimSpace(restaurantID))
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

func (r *Repository) CreateCatalogFolder(ctx context.Context, v domain.CatalogFolder) (domain.CatalogFolder, error) {
	out, err := scanCatalogFolder(r.pool.QueryRow(ctx, `INSERT INTO cloud_catalog_folders(id,restaurant_id,parent_id,name,sort_order,status,cloud_version,archived_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,restaurant_id,COALESCE(parent_id,''),name,sort_order,status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.RestaurantID, nullableText(v.ParentID), v.Name, v.SortOrder, v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateCatalogFolder(ctx context.Context, v domain.CatalogFolder) (domain.CatalogFolder, error) {
	out, err := scanCatalogFolder(r.pool.QueryRow(ctx, `UPDATE cloud_catalog_folders SET parent_id=$2,name=$3,sort_order=$4,status=$5,cloud_version=$6,archived_at=$7,updated_at=$8 WHERE id=$1 RETURNING id,restaurant_id,COALESCE(parent_id,''),name,sort_order,status,cloud_version,archived_at,created_at,updated_at`, v.ID, nullableText(v.ParentID), v.Name, v.SortOrder, v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetCatalogFolder(ctx context.Context, id string) (domain.CatalogFolder, error) {
	v, err := scanCatalogFolder(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,COALESCE(parent_id,''),name,sort_order,status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_folders WHERE id=$1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListCatalogFolders(ctx context.Context, restaurantID string) ([]domain.CatalogFolder, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,COALESCE(parent_id,''),name,sort_order,status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_folders WHERE restaurant_id=$1 ORDER BY parent_id NULLS FIRST, sort_order, id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CatalogFolder
	for rows.Next() {
		v, err := scanCatalogFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateFolderParameter(ctx context.Context, v domain.FolderParameter) (domain.FolderParameter, error) {
	out, err := scanFolderParameter(r.pool.QueryRow(ctx, `INSERT INTO cloud_catalog_folder_parameters(id,restaurant_id,folder_id,parameter_key,value_type,value_json,status,cloud_version,archived_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10,$11) RETURNING id,restaurant_id,folder_id,parameter_key,value_type,value_json::text,status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.RestaurantID, v.FolderID, v.Key, v.ValueType, v.ValueJSON, v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateFolderParameter(ctx context.Context, v domain.FolderParameter) (domain.FolderParameter, error) {
	out, err := scanFolderParameter(r.pool.QueryRow(ctx, `UPDATE cloud_catalog_folder_parameters SET value_type=$2,value_json=$3::jsonb,status=$4,cloud_version=$5,archived_at=$6,updated_at=$7 WHERE id=$1 RETURNING id,restaurant_id,folder_id,parameter_key,value_type,value_json::text,status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.ValueType, v.ValueJSON, v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetFolderParameter(ctx context.Context, id string) (domain.FolderParameter, error) {
	v, err := scanFolderParameter(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,folder_id,parameter_key,value_type,value_json::text,status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_folder_parameters WHERE id=$1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListFolderParameters(ctx context.Context, restaurantID string) ([]domain.FolderParameter, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,folder_id,parameter_key,value_type,value_json::text,status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_folder_parameters WHERE restaurant_id=$1 ORDER BY folder_id,parameter_key`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FolderParameter
	for rows.Next() {
		v, err := scanFolderParameter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCatalogTag(ctx context.Context, v domain.CatalogTag) (domain.CatalogTag, error) {
	out, err := scanCatalogTag(r.pool.QueryRow(ctx, `INSERT INTO cloud_catalog_tags(id,restaurant_id,name,code,status,cloud_version,archived_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,restaurant_id,name,code,status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.RestaurantID, v.Name, v.Code, v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateCatalogTag(ctx context.Context, v domain.CatalogTag) (domain.CatalogTag, error) {
	out, err := scanCatalogTag(r.pool.QueryRow(ctx, `UPDATE cloud_catalog_tags SET name=$2,code=$3,status=$4,cloud_version=$5,archived_at=$6,updated_at=$7 WHERE id=$1 RETURNING id,restaurant_id,name,code,status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.Name, v.Code, v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetCatalogTag(ctx context.Context, id string) (domain.CatalogTag, error) {
	v, err := scanCatalogTag(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,name,code,status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_tags WHERE id=$1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListCatalogTags(ctx context.Context, restaurantID string) ([]domain.CatalogTag, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,name,code,status,cloud_version,archived_at,created_at,updated_at FROM cloud_catalog_tags WHERE restaurant_id=$1 ORDER BY code,id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CatalogTag
	for rows.Next() {
		v, err := scanCatalogTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) AssignCatalogItemTag(ctx context.Context, v domain.CatalogItemTag) (domain.CatalogItemTag, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO cloud_catalog_item_tags(restaurant_id,catalog_item_id,tag_id,cloud_version,created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT(catalog_item_id,tag_id) DO UPDATE SET cloud_version=EXCLUDED.cloud_version`, v.RestaurantID, v.CatalogItemID, v.TagID, v.CloudVersion, v.CreatedAt)
	return v, normalizeErr(err)
}

func (r *Repository) ListCatalogItemTags(ctx context.Context, restaurantID string) ([]domain.CatalogItemTag, error) {
	rows, err := r.pool.Query(ctx, `SELECT restaurant_id,catalog_item_id,tag_id,cloud_version,created_at FROM cloud_catalog_item_tags WHERE restaurant_id=$1 ORDER BY catalog_item_id,tag_id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CatalogItemTag
	for rows.Next() {
		var v domain.CatalogItemTag
		if err := rows.Scan(&v.RestaurantID, &v.CatalogItemID, &v.TagID, &v.CloudVersion, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateModifierGroup(ctx context.Context, v domain.ModifierGroup) (domain.ModifierGroup, error) {
	out, err := scanModifierGroup(r.pool.QueryRow(ctx, `INSERT INTO cloud_modifier_groups(id,restaurant_id,name,status,required,min_count,max_count,cloud_version,archived_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id,restaurant_id,name,status,required,min_count,max_count,cloud_version,archived_at,created_at,updated_at`, v.ID, v.RestaurantID, v.Name, v.Status, v.Required, v.MinCount, v.MaxCount, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateModifierGroup(ctx context.Context, v domain.ModifierGroup) (domain.ModifierGroup, error) {
	out, err := scanModifierGroup(r.pool.QueryRow(ctx, `UPDATE cloud_modifier_groups SET name=$2,status=$3,required=$4,min_count=$5,max_count=$6,cloud_version=$7,archived_at=$8,updated_at=$9 WHERE id=$1 RETURNING id,restaurant_id,name,status,required,min_count,max_count,cloud_version,archived_at,created_at,updated_at`, v.ID, v.Name, v.Status, v.Required, v.MinCount, v.MaxCount, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetModifierGroup(ctx context.Context, id string) (domain.ModifierGroup, error) {
	v, err := scanModifierGroup(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,name,status,required,min_count,max_count,cloud_version,archived_at,created_at,updated_at FROM cloud_modifier_groups WHERE id=$1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListModifierGroups(ctx context.Context, restaurantID string) ([]domain.ModifierGroup, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,name,status,required,min_count,max_count,cloud_version,archived_at,created_at,updated_at FROM cloud_modifier_groups WHERE restaurant_id=$1 ORDER BY id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ModifierGroup
	for rows.Next() {
		v, err := scanModifierGroup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateModifierOption(ctx context.Context, v domain.ModifierOption) (domain.ModifierOption, error) {
	out, err := scanModifierOption(r.pool.QueryRow(ctx, `INSERT INTO cloud_modifier_options(id,restaurant_id,modifier_group_id,name,price_minor,status,cloud_version,archived_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,restaurant_id,modifier_group_id,name,price_minor,status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.RestaurantID, v.ModifierGroupID, v.Name, v.PriceMinor, v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateModifierOption(ctx context.Context, v domain.ModifierOption) (domain.ModifierOption, error) {
	out, err := scanModifierOption(r.pool.QueryRow(ctx, `UPDATE cloud_modifier_options SET name=$2,price_minor=$3,status=$4,cloud_version=$5,archived_at=$6,updated_at=$7 WHERE id=$1 RETURNING id,restaurant_id,modifier_group_id,name,price_minor,status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.Name, v.PriceMinor, v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetModifierOption(ctx context.Context, id string) (domain.ModifierOption, error) {
	v, err := scanModifierOption(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,modifier_group_id,name,price_minor,status,cloud_version,archived_at,created_at,updated_at FROM cloud_modifier_options WHERE id=$1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListModifierOptions(ctx context.Context, restaurantID string) ([]domain.ModifierOption, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,modifier_group_id,name,price_minor,status,cloud_version,archived_at,created_at,updated_at FROM cloud_modifier_options WHERE restaurant_id=$1 ORDER BY modifier_group_id,id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ModifierOption
	for rows.Next() {
		v, err := scanModifierOption(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateModifierGroupBinding(ctx context.Context, v domain.ModifierGroupBinding) (domain.ModifierGroupBinding, error) {
	out, err := scanModifierBinding(r.pool.QueryRow(ctx, `INSERT INTO cloud_modifier_group_bindings(id,restaurant_id,modifier_group_id,target_type,target_id,sort_order,status,cloud_version,archived_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id,restaurant_id,modifier_group_id,target_type,target_id,sort_order,status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.RestaurantID, v.ModifierGroupID, v.TargetType, v.TargetID, v.SortOrder, v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdateModifierGroupBinding(ctx context.Context, v domain.ModifierGroupBinding) (domain.ModifierGroupBinding, error) {
	out, err := scanModifierBinding(r.pool.QueryRow(ctx, `UPDATE cloud_modifier_group_bindings SET sort_order=$2,status=$3,cloud_version=$4,archived_at=$5,updated_at=$6 WHERE id=$1 RETURNING id,restaurant_id,modifier_group_id,target_type,target_id,sort_order,status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.SortOrder, v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetModifierGroupBinding(ctx context.Context, id string) (domain.ModifierGroupBinding, error) {
	v, err := scanModifierBinding(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,modifier_group_id,target_type,target_id,sort_order,status,cloud_version,archived_at,created_at,updated_at FROM cloud_modifier_group_bindings WHERE id=$1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListModifierGroupBindings(ctx context.Context, restaurantID string) ([]domain.ModifierGroupBinding, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,modifier_group_id,target_type,target_id,sort_order,status,cloud_version,archived_at,created_at,updated_at FROM cloud_modifier_group_bindings WHERE restaurant_id=$1 ORDER BY target_type,target_id,sort_order,id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ModifierGroupBinding
	for rows.Next() {
		v, err := scanModifierBinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreatePricingPolicy(ctx context.Context, v domain.PricingPolicy) (domain.PricingPolicy, error) {
	out, err := scanPricingPolicy(r.pool.QueryRow(ctx, `INSERT INTO cloud_pricing_policies(id,restaurant_id,name,kind,scope,amount_kind,amount_minor,value_basis_points,application_index,manual,requires_permission,status,cloud_version,archived_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id,restaurant_id,name,kind,scope,amount_kind,amount_minor,value_basis_points,application_index,manual,COALESCE(requires_permission,''),status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.RestaurantID, v.Name, v.Kind, v.Scope, v.AmountKind, v.AmountMinor, v.ValueBasisPoints, v.ApplicationIndex, v.Manual, nullableText(v.RequiresPermission), v.Status, v.CloudVersion, v.ArchivedAt, v.CreatedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) UpdatePricingPolicy(ctx context.Context, v domain.PricingPolicy) (domain.PricingPolicy, error) {
	out, err := scanPricingPolicy(r.pool.QueryRow(ctx, `UPDATE cloud_pricing_policies SET name=$2,scope=$3,amount_kind=$4,amount_minor=$5,value_basis_points=$6,application_index=$7,manual=$8,requires_permission=$9,status=$10,cloud_version=$11,archived_at=$12,updated_at=$13 WHERE id=$1 RETURNING id,restaurant_id,name,kind,scope,amount_kind,amount_minor,value_basis_points,application_index,manual,COALESCE(requires_permission,''),status,cloud_version,archived_at,created_at,updated_at`, v.ID, v.Name, v.Scope, v.AmountKind, v.AmountMinor, v.ValueBasisPoints, v.ApplicationIndex, v.Manual, nullableText(v.RequiresPermission), v.Status, v.CloudVersion, v.ArchivedAt, v.UpdatedAt))
	return out, normalizeErr(err)
}

func (r *Repository) GetPricingPolicy(ctx context.Context, id string) (domain.PricingPolicy, error) {
	v, err := scanPricingPolicy(r.pool.QueryRow(ctx, `SELECT id,restaurant_id,name,kind,scope,amount_kind,amount_minor,value_basis_points,application_index,manual,COALESCE(requires_permission,''),status,cloud_version,archived_at,created_at,updated_at FROM cloud_pricing_policies WHERE id=$1`, strings.TrimSpace(id)))
	return v, normalizeErr(err)
}

func (r *Repository) ListPricingPolicies(ctx context.Context, restaurantID string) ([]domain.PricingPolicy, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,restaurant_id,name,kind,scope,amount_kind,amount_minor,value_basis_points,application_index,manual,COALESCE(requires_permission,''),status,cloud_version,archived_at,created_at,updated_at FROM cloud_pricing_policies WHERE restaurant_id=$1 ORDER BY application_index,id`, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PricingPolicy
	for rows.Next() {
		v, err := scanPricingPolicy(rows)
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
	case domain.CatalogItemGood:
		_, err := tx.Exec(ctx, `INSERT INTO cloud_goods(catalog_item_id,restaurant_id,updated_at) VALUES ($1,$2,$3) ON CONFLICT (catalog_item_id) DO UPDATE SET updated_at = EXCLUDED.updated_at`, v.ID, v.RestaurantID, v.UpdatedAt)
		return err
	case domain.CatalogItemSemiFinished:
		_, err := tx.Exec(ctx, `INSERT INTO cloud_semi_finished_products(catalog_item_id,restaurant_id,updated_at) VALUES ($1,$2,$3) ON CONFLICT (catalog_item_id) DO UPDATE SET updated_at = EXCLUDED.updated_at`, v.ID, v.RestaurantID, v.UpdatedAt)
		return err
	case domain.CatalogItemService:
		_, err := tx.Exec(ctx, `INSERT INTO cloud_services(catalog_item_id,restaurant_id,fixed_unit,updated_at) VALUES ($1,$2,$3,$4) ON CONFLICT (catalog_item_id) DO UPDATE SET fixed_unit = EXCLUDED.fixed_unit, updated_at = EXCLUDED.updated_at`, v.ID, v.RestaurantID, v.BaseUnit, v.UpdatedAt)
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
	err := row.Scan(&v.ID, &v.RestaurantID, &kind, &v.FolderID, &v.Name, &v.SKU, &v.BaseUnit, &v.KitchenType, &v.AccountingCategory, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
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

func scanCatalogFolder(row scanner) (domain.CatalogFolder, error) {
	var v domain.CatalogFolder
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.ParentID, &v.Name, &v.SortOrder, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanFolderParameter(row scanner) (domain.FolderParameter, error) {
	var v domain.FolderParameter
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.FolderID, &v.Key, &v.ValueType, &v.ValueJSON, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanCatalogTag(row scanner) (domain.CatalogTag, error) {
	var v domain.CatalogTag
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.Name, &v.Code, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanModifierGroup(row scanner) (domain.ModifierGroup, error) {
	var v domain.ModifierGroup
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.Name, &status, &v.Required, &v.MinCount, &v.MaxCount, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanModifierOption(row scanner) (domain.ModifierOption, error) {
	var v domain.ModifierOption
	var status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.ModifierGroupID, &v.Name, &v.PriceMinor, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanModifierBinding(row scanner) (domain.ModifierGroupBinding, error) {
	var v domain.ModifierGroupBinding
	var targetType, status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.ModifierGroupID, &targetType, &v.TargetID, &v.SortOrder, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.TargetType = domain.ModifierTargetType(targetType)
	v.Status = domain.LifecycleStatus(status)
	return v, err
}

func scanPricingPolicy(row scanner) (domain.PricingPolicy, error) {
	var v domain.PricingPolicy
	var kind, status string
	err := row.Scan(&v.ID, &v.RestaurantID, &v.Name, &kind, &v.Scope, &v.AmountKind, &v.AmountMinor, &v.ValueBasisPoints, &v.ApplicationIndex, &v.Manual, &v.RequiresPermission, &status, &v.CloudVersion, &v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	v.Kind = domain.PricingPolicyKind(kind)
	v.Status = domain.LifecycleStatus(status)
	return v, err
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
