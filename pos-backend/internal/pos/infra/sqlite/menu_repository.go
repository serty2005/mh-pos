package sqlite

import (
	"context"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateMenuItem(ctx context.Context, v *domain.MenuItem) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO menu_items(id,catalog_item_id,name,price,currency,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.CatalogItemID, v.Name, v.Price, v.Currency, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListMenuItems(ctx context.Context) ([]domain.MenuItem, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,catalog_item_id,name,price,currency,active,created_at,updated_at FROM menu_items ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MenuItem
	for rows.Next() {
		var v domain.MenuItem
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.CatalogItemID, &v.Name, &v.Price, &v.Currency, &active, &created, &updated); err != nil {
			return nil, err
		}
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
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,catalog_item_id,name,price,currency,active,created_at,updated_at FROM menu_items WHERE id = ?`, id).
		Scan(&v.ID, &v.CatalogItemID, &v.Name, &v.Price, &v.Currency, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}
