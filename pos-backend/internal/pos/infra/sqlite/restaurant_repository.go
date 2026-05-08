package sqlite

import (
	"context"
	"database/sql"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateRestaurant(ctx context.Context, v *domain.Restaurant) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO restaurants(id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.Name, v.Timezone, v.Currency, string(v.BusinessDayMode), v.BusinessDayBoundaryLocalTime, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetRestaurant(ctx context.Context, id string) (*domain.Restaurant, error) {
	return scanRestaurant(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,active,created_at,updated_at FROM restaurants WHERE id = ?`, id))
}

func (r *Repository) ListRestaurants(ctx context.Context) ([]domain.Restaurant, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,name,timezone,currency,business_day_mode,business_day_boundary_local_time,active,created_at,updated_at FROM restaurants ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Restaurant
	for rows.Next() {
		v, err := scanRestaurantRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func scanRestaurant(row *sql.Row) (*domain.Restaurant, error) {
	return scanRestaurantRows(row)
}

type restaurantScanner interface {
	Scan(dest ...any) error
}

func scanRestaurantRows(row restaurantScanner) (*domain.Restaurant, error) {
	var v domain.Restaurant
	var mode string
	var active int
	var created, updated string
	if err := row.Scan(&v.ID, &v.Name, &v.Timezone, &v.Currency, &mode, &v.BusinessDayBoundaryLocalTime, &active, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.BusinessDayMode = domain.BusinessDayMode(mode)
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}
