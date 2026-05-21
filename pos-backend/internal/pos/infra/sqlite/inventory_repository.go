package sqlite

import (
	"context"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateRecipeVersion(ctx context.Context, v *domain.RecipeVersion) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_versions(id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.DishCatalogItemID, v.Version, v.Name, string(v.Status), v.YieldQuantity, v.YieldUnit, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRecipeVersions(ctx context.Context) ([]domain.RecipeVersion, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,dish_catalog_item_id,version,name,status,yield_quantity,yield_unit,active,created_at,updated_at FROM recipe_versions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecipeVersion
	for rows.Next() {
		var v domain.RecipeVersion
		var status, created, updated string
		var active int
		if err := rows.Scan(&v.ID, &v.DishCatalogItemID, &v.Version, &v.Name, &status, &v.YieldQuantity, &v.YieldUnit, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Status = domain.RecipeVersionStatus(status)
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateRecipeLine(ctx context.Context, v *domain.RecipeLine) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO recipe_lines(id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.RecipeVersionID, v.CatalogItemID, v.Quantity, v.Unit, v.LossPercent, dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRecipeLines(ctx context.Context, recipeVersionID string) ([]domain.RecipeLine, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,recipe_version_id,catalog_item_id,quantity,unit,loss_percent,created_at,updated_at FROM recipe_lines WHERE recipe_version_id = ? ORDER BY created_at`, recipeVersionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RecipeLine
	for rows.Next() {
		var v domain.RecipeLine
		var created, updated string
		if err := rows.Scan(&v.ID, &v.RecipeVersionID, &v.CatalogItemID, &v.Quantity, &v.Unit, &v.LossPercent, &created, &updated); err != nil {
			return nil, err
		}
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}
