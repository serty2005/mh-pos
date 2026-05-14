package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateMenuItem(ctx context.Context, v *domain.MenuItem) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO menu_items(id,catalog_item_id,name,price,currency,tax_profile_id,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.CatalogItemID, v.Name, v.Price, v.Currency, nullableString(v.TaxProfileID), boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListMenuItems(ctx context.Context) ([]domain.MenuItem, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT m.id,m.catalog_item_id,COALESCE(c.type,''),m.name,m.price,m.currency,m.tax_profile_id,m.active,m.created_at,m.updated_at FROM menu_items m LEFT JOIN catalog_items c ON c.id = m.catalog_item_id ORDER BY m.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MenuItem
	for rows.Next() {
		var v domain.MenuItem
		var active int
		var created, updated string
		var taxProfileID sql.NullString
		if err := rows.Scan(&v.ID, &v.CatalogItemID, &v.ItemType, &v.Name, &v.Price, &v.Currency, &taxProfileID, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.TaxProfileID = stringPtr(taxProfileID)
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) GetMenuItem(ctx context.Context, id string) (*domain.MenuItem, error) {
	var v domain.MenuItem
	var active int
	var created, updated string
	var taxProfileID sql.NullString
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT m.id,m.catalog_item_id,COALESCE(c.type,''),m.name,m.price,m.currency,m.tax_profile_id,m.active,m.created_at,m.updated_at FROM menu_items m LEFT JOIN catalog_items c ON c.id = m.catalog_item_id WHERE m.id = ?`, id).
		Scan(&v.ID, &v.CatalogItemID, &v.ItemType, &v.Name, &v.Price, &v.Currency, &taxProfileID, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.TaxProfileID = stringPtr(taxProfileID)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}
