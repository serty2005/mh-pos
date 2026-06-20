package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
	menudomain "pos-backend/internal/pos/domain/menu"
)

func (r *Repository) CreateMenuItem(ctx context.Context, v *domain.MenuItem) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO menu_items(id,catalog_item_id,category_id,tag_id,name,price,currency,tax_profile_id,runtime_status,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.CatalogItemID, nullableStringValue(v.CategoryID), nullableStringValue(v.TagID), v.Name, v.Price, v.Currency, nullableString(v.TaxProfileID), menuRuntimeStatus(v.RuntimeStatus), boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListMenuItems(ctx context.Context) ([]domain.MenuItem, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT m.id,m.catalog_item_id,COALESCE(m.category_id,''),COALESCE(m.tag_id,''),COALESCE(c.type,''),m.name,m.price,m.currency,m.tax_profile_id,COALESCE(m.runtime_status,'available'),m.active,m.created_at,m.updated_at,
COALESCE(sl.active,0),sl.available_quantity
FROM menu_items m
LEFT JOIN catalog_items c ON c.id = m.catalog_item_id
LEFT JOIN stop_lists sl ON sl.catalog_item_id = m.catalog_item_id AND sl.active = 1 AND sl.cloud_deleted_at IS NULL
ORDER BY m.created_at`)
	if err != nil {
		return nil, err
	}
	var out []domain.MenuItem
	for rows.Next() {
		var v domain.MenuItem
		var active int
		var stopListActive int
		var created, updated string
		var taxProfileID sql.NullString
		var stopListQuantity sql.NullFloat64
		if err := rows.Scan(&v.ID, &v.CatalogItemID, &v.CategoryID, &v.TagID, &v.ItemType, &v.Name, &v.Price, &v.Currency, &taxProfileID, &v.RuntimeStatus, &active, &created, &updated, &stopListActive, &stopListQuantity); err != nil {
			return nil, err
		}
		v.TaxProfileID = stringPtr(taxProfileID)
		v.Active = active == 1
		applyMenuStopListOverlay(&v, stopListActive, stopListQuantity)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := r.hydrateMenuItemModifiers(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *Repository) GetMenuItem(ctx context.Context, id string) (*domain.MenuItem, error) {
	var v domain.MenuItem
	var active int
	var stopListActive int
	var created, updated string
	var taxProfileID sql.NullString
	var stopListQuantity sql.NullFloat64
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT m.id,m.catalog_item_id,COALESCE(m.category_id,''),COALESCE(m.tag_id,''),COALESCE(c.type,''),m.name,m.price,m.currency,m.tax_profile_id,COALESCE(m.runtime_status,'available'),m.active,m.created_at,m.updated_at,
COALESCE(sl.active,0),sl.available_quantity
FROM menu_items m
LEFT JOIN catalog_items c ON c.id = m.catalog_item_id
LEFT JOIN stop_lists sl ON sl.catalog_item_id = m.catalog_item_id AND sl.active = 1 AND sl.cloud_deleted_at IS NULL
WHERE m.id = ?`, id).
		Scan(&v.ID, &v.CatalogItemID, &v.CategoryID, &v.TagID, &v.ItemType, &v.Name, &v.Price, &v.Currency, &taxProfileID, &v.RuntimeStatus, &active, &created, &updated, &stopListActive, &stopListQuantity)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.TaxProfileID = stringPtr(taxProfileID)
	applyMenuStopListOverlay(&v, stopListActive, stopListQuantity)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	if err := r.hydrateMenuItemModifiers(ctx, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func applyMenuStopListOverlay(item *domain.MenuItem, active int, available sql.NullFloat64) {
	item.StopListActive = active == 1
	if !item.StopListActive {
		return
	}
	if available.Valid {
		value := available.Float64
		item.StopListAvailableQuantity = &value
		item.StopListBlocked = value <= 0
		return
	}
	item.StopListBlocked = true
}

func menuRuntimeStatus(value string) string {
	if value == "unavailable" || value == "hidden" {
		return value
	}
	return "available"
}

func (r *Repository) hydrateMenuItemModifiers(ctx context.Context, item *domain.MenuItem) error {
	groups, err := r.ListModifierGroupsForMenuItem(ctx, item.ID)
	if err != nil {
		return err
	}
	if len(groups) == 0 {
		return nil
	}
	groupIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	optionsByGroup, err := r.ListModifierOptionsByGroupIDs(ctx, groupIDs)
	if err != nil {
		return err
	}
	item.ModifierGroups = make([]menudomain.MenuItemModifierGroup, 0, len(groups))
	for _, group := range groups {
		view := menudomain.MenuItemModifierGroup{
			ID:           group.ID,
			RestaurantID: group.RestaurantID,
			Name:         group.Name,
			Required:     group.Required,
			MinCount:     group.MinCount,
			MaxCount:     group.MaxCount,
			Active:       group.Active,
		}
		for _, option := range optionsByGroup[group.ID] {
			view.Options = append(view.Options, menudomain.MenuItemModifierOption{
				ID:              option.ID,
				RestaurantID:    option.RestaurantID,
				ModifierGroupID: option.ModifierGroupID,
				Name:            option.Name,
				PriceMinor:      option.PriceMinor,
				Active:          option.Active,
			})
		}
		item.ModifierGroups = append(item.ModifierGroups, view)
	}
	return nil
}
