package sqlite

import (
	"context"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateRestaurant(ctx context.Context, v *domain.Restaurant) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO restaurants(id,name,timezone,currency,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		v.ID, v.Name, v.Timezone, v.Currency, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRestaurants(ctx context.Context) ([]domain.Restaurant, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,name,timezone,currency,active,created_at,updated_at FROM restaurants ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Restaurant
	for rows.Next() {
		var v domain.Restaurant
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.Name, &v.Timezone, &v.Currency, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}
