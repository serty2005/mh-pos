package sqlite

import (
	"context"
	"database/sql"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateRecipeVersion(ctx context.Context, v *domain.RecipeVersion) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.DishCatalogItemID, v.Version, v.Name, string(v.Status), v.YieldQuantity, v.YieldUnit, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRecipeVersions(ctx context.Context) ([]domain.RecipeVersion, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at FROM recipe_versions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecipeVersion
	for rows.Next() {
		var v domain.RecipeVersion
		var status, created, updated string
		var active int
		var cloudUpdatedAt, cloudDeletedAt, lastSyncedAt sql.NullString
		if err := rows.Scan(&v.ID, &v.DishCatalogItemID, &v.Version, &v.Name, &status, &v.YieldQuantity, &v.YieldUnit, &active, &created, &updated, &v.CloudVersion, &cloudUpdatedAt, &cloudDeletedAt, &lastSyncedAt); err != nil {
			return nil, err
		}
		v.Status = domain.RecipeVersionStatus(status)
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		v.CloudUpdatedAt = nullStringPtr(cloudUpdatedAt)
		v.CloudDeletedAt = nullStringPtr(cloudDeletedAt)
		if lastSyncedAt.Valid {
			v.LastSyncedAt = lastSyncedAt.String
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) GetActiveRecipeVersionByCatalogItem(ctx context.Context, catalogItemID string) (*domain.RecipeVersion, error) {
	var v domain.RecipeVersion
	var status, created, updated string
	var active int
	var cloudUpdatedAt, cloudDeletedAt, lastSyncedAt sql.NullString
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at
FROM recipe_versions
WHERE dish_catalog_item_id = ? AND active = 1 AND status = 'active' AND cloud_deleted_at IS NULL
ORDER BY version DESC
LIMIT 1`, catalogItemID).Scan(&v.ID, &v.DishCatalogItemID, &v.Version, &v.Name, &status, &v.YieldQuantity, &v.YieldUnit, &active, &created, &updated, &v.CloudVersion, &cloudUpdatedAt, &cloudDeletedAt, &lastSyncedAt)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.RecipeVersionStatus(status)
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	v.CloudUpdatedAt = nullStringPtr(cloudUpdatedAt)
	v.CloudDeletedAt = nullStringPtr(cloudDeletedAt)
	if lastSyncedAt.Valid {
		v.LastSyncedAt = lastSyncedAt.String
	}
	return &v, nil
}

func (r *Repository) CreateRecipeLine(ctx context.Context, v *domain.RecipeLine) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.RecipeVersionID, v.CatalogItemID, v.Quantity, v.Unit, v.LossPercent, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRecipeLines(ctx context.Context, recipeVersionID string) ([]domain.RecipeLine, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at FROM recipe_lines WHERE recipe_version_id = ? AND cloud_deleted_at IS NULL ORDER BY created_at`, recipeVersionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecipeLine
	for rows.Next() {
		var v domain.RecipeLine
		var created, updated string
		var cloudUpdatedAt, cloudDeletedAt, lastSyncedAt sql.NullString
		if err := rows.Scan(&v.ID, &v.RecipeVersionID, &v.CatalogItemID, &v.Quantity, &v.Unit, &v.LossPercent, &created, &updated, &v.CloudVersion, &cloudUpdatedAt, &cloudDeletedAt, &lastSyncedAt); err != nil {
			return nil, err
		}
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		v.CloudUpdatedAt = nullStringPtr(cloudUpdatedAt)
		v.CloudDeletedAt = nullStringPtr(cloudDeletedAt)
		if lastSyncedAt.Valid {
			v.LastSyncedAt = lastSyncedAt.String
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertMasterRecipeVersion(ctx context.Context, v *domain.RecipeVersion, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  dish_catalog_item_id = excluded.dish_catalog_item_id,
  version = excluded.version,
  name = excluded.name,
  status = excluded.status,
  yield_quantity = excluded.yield_quantity,
  yield_unit = excluded.yield_unit,
  active = excluded.active,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.DishCatalogItemID, v.Version, v.Name, string(v.Status), v.YieldQuantity, v.YieldUnit, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterRecipeLine(ctx context.Context, v *domain.RecipeLine, meta domain.MasterRecordSyncMeta) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  recipe_version_id = excluded.recipe_version_id,
  catalog_item_id = excluded.catalog_item_id,
  quantity = excluded.quantity,
  unit = excluded.unit,
  loss_percent = excluded.loss_percent,
  updated_at = excluded.updated_at,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at`,
		v.ID, v.RecipeVersionID, v.CatalogItemID, v.Quantity, v.Unit, v.LossPercent, dbTime(v.CreatedAt), dbTime(v.UpdatedAt), meta.CloudVersion, nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt)
	return normalizeErr(err)
}

func (r *Repository) UpsertMasterStopListEntry(ctx context.Context, v *domain.StopListEntry, meta domain.MasterRecordSyncMeta) error {
	cloudVersion := v.CloudVersion
	if cloudVersion == nil {
		value := meta.CloudVersion
		cloudVersion = &value
	}
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO stop_lists(id,restaurant_id,catalog_item_id,available_quantity,source,reason,active,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at,updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(restaurant_id,catalog_item_id) DO UPDATE SET
  id = excluded.id,
  available_quantity = excluded.available_quantity,
  source = excluded.source,
  reason = excluded.reason,
  active = excluded.active,
  cloud_version = excluded.cloud_version,
  cloud_updated_at = excluded.cloud_updated_at,
  cloud_deleted_at = excluded.cloud_deleted_at,
  last_synced_at = excluded.last_synced_at,
  updated_at = excluded.updated_at`,
		v.ID, v.RestaurantID, v.CatalogItemID, nullableFloat64(v.AvailableQuantity), v.Source, nullableString(v.Reason), boolInt(v.Active), nullableInt64(cloudVersion), nullableString(meta.CloudUpdatedAt), nullableString(meta.CloudDeletedAt), meta.LastSyncedAt, dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetBlockingStopListEntry(ctx context.Context, restaurantID, catalogItemID string) (*domain.StopListEntry, error) {
	var v domain.StopListEntry
	var available sql.NullFloat64
	var reason, cloudUpdatedAt, cloudDeletedAt, lastSyncedAt sql.NullString
	var cloudVersion sql.NullInt64
	var active int
	var updatedAt string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,catalog_item_id,available_quantity,source,reason,active,cloud_version,cloud_updated_at,cloud_deleted_at,last_synced_at,updated_at
FROM stop_lists
WHERE restaurant_id = ? AND catalog_item_id = ? AND active = 1 AND cloud_deleted_at IS NULL AND (available_quantity IS NULL OR available_quantity <= 0)
LIMIT 1`, restaurantID, catalogItemID).Scan(&v.ID, &v.RestaurantID, &v.CatalogItemID, &available, &v.Source, &reason, &active, &cloudVersion, &cloudUpdatedAt, &cloudDeletedAt, &lastSyncedAt, &updatedAt)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.AvailableQuantity = nullFloat64Ptr(available)
	v.Reason = nullStringPtr(reason)
	v.Active = active == 1
	v.CloudVersion = nullInt64Ptr(cloudVersion)
	v.CloudUpdatedAt = nullStringPtr(cloudUpdatedAt)
	v.CloudDeletedAt = nullStringPtr(cloudDeletedAt)
	if lastSyncedAt.Valid {
		v.LastSyncedAt = lastSyncedAt.String
	}
	v.UpdatedAt = parseTime(updatedAt)
	return &v, nil
}

func nullableFloat64(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullFloat64Ptr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func nullInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}
