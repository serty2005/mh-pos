package sqlite

import (
	"context"
	"database/sql"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateCatalogItem(ctx context.Context, v *domain.CatalogItem) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO catalog_items(id,type,folder_id,name,sku,base_unit,kitchen_type,accounting_category,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, string(v.Type), nullableString(v.FolderID), v.Name, v.SKU, v.BaseUnit, v.KitchenType, v.AccountingCategory, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListCatalogItems(ctx context.Context) ([]domain.CatalogItem, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,type,folder_id,name,sku,base_unit,kitchen_type,accounting_category,active,created_at,updated_at FROM catalog_items ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CatalogItem
	for rows.Next() {
		var v domain.CatalogItem
		var typ string
		var folderID sql.NullString
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &typ, &folderID, &v.Name, &v.SKU, &v.BaseUnit, &v.KitchenType, &v.AccountingCategory, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Type = domain.CatalogItemType(typ)
		v.FolderID = stringPtr(folderID)
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) GetCatalogItem(ctx context.Context, id string) (*domain.CatalogItem, error) {
	var v domain.CatalogItem
	var typ string
	var folderID sql.NullString
	var active int
	var created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,type,folder_id,name,sku,base_unit,kitchen_type,accounting_category,active,created_at,updated_at FROM catalog_items WHERE id = ?`, id).
		Scan(&v.ID, &typ, &folderID, &v.Name, &v.SKU, &v.BaseUnit, &v.KitchenType, &v.AccountingCategory, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Type = domain.CatalogItemType(typ)
	v.FolderID = stringPtr(folderID)
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) ListModifierGroupsForMenuItem(ctx context.Context, menuItemID string) ([]domain.ModifierGroup, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT g.id,g.restaurant_id,g.name,g.required,g.min_count,g.max_count,g.active
FROM menu_item_modifier_groups mg
JOIN modifier_groups g ON g.id = mg.modifier_group_id
WHERE mg.menu_item_id = ? AND g.active = 1
ORDER BY mg.sort_order, g.name`, menuItemID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	var out []domain.ModifierGroup
	for rows.Next() {
		var v domain.ModifierGroup
		var required, active int
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.Name, &required, &v.MinCount, &v.MaxCount, &active); err != nil {
			return nil, err
		}
		v.Required = required == 1
		v.Active = active == 1
		out = append(out, v)
	}
	return out, normalizeErr(rows.Err())
}

func (r *Repository) ListModifierOptionsByGroupIDs(ctx context.Context, groupIDs []string) (map[string][]domain.ModifierOption, error) {
	out := make(map[string][]domain.ModifierOption, len(groupIDs))
	if len(groupIDs) == 0 {
		return out, nil
	}
	for _, groupID := range groupIDs {
		rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,modifier_group_id,COALESCE(linked_catalog_item_id,''),name,price_minor,active FROM modifier_options WHERE modifier_group_id = ? AND active = 1 ORDER BY name`, groupID)
		if err != nil {
			return nil, normalizeErr(err)
		}
		for rows.Next() {
			var v domain.ModifierOption
			var active int
			if err := rows.Scan(&v.ID, &v.RestaurantID, &v.ModifierGroupID, &v.LinkedCatalogItemID, &v.Name, &v.PriceMinor, &active); err != nil {
				_ = rows.Close()
				return nil, err
			}
			v.Active = active == 1
			out[groupID] = append(out[groupID], v)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, normalizeErr(err)
		}
		if err := rows.Close(); err != nil {
			return nil, normalizeErr(err)
		}
	}
	return out, nil
}

func (r *Repository) CatalogItemInUse(ctx context.Context, id string) (bool, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM menu_items WHERE catalog_item_id = ?`, id).Scan(&n)
	return n > 0, normalizeErr(err)
}
