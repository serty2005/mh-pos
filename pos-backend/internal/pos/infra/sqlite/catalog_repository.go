package sqlite

import (
	"context"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateCatalogItem(ctx context.Context, v *domain.CatalogItem) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO catalog_items(id,type,name,sku,base_unit,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, string(v.Type), v.Name, v.SKU, v.BaseUnit, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListCatalogItems(ctx context.Context) ([]domain.CatalogItem, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,type,name,sku,base_unit,active,created_at,updated_at FROM catalog_items ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CatalogItem
	for rows.Next() {
		var v domain.CatalogItem
		var typ string
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &typ, &v.Name, &v.SKU, &v.BaseUnit, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Type = domain.CatalogItemType(typ)
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
	var active int
	var created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,type,name,sku,base_unit,active,created_at,updated_at FROM catalog_items WHERE id = ?`, id).
		Scan(&v.ID, &typ, &v.Name, &v.SKU, &v.BaseUnit, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Type = domain.CatalogItemType(typ)
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) CatalogItemInUse(ctx context.Context, id string) (bool, error) {
	var n int
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT COUNT(1) FROM menu_items WHERE catalog_item_id = ?`, id).Scan(&n)
	return n > 0, normalizeErr(err)
}
