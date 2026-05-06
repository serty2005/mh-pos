package sqlite

import (
	"context"

	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateHall(ctx context.Context, v *domain.Hall) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO halls(id,restaurant_id,name,active,created_at,updated_at) VALUES (?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.Name, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetHall(ctx context.Context, id string) (*domain.Hall, error) {
	var v domain.Hall
	var active int
	var created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,name,active,created_at,updated_at FROM halls WHERE id = ?`, id).
		Scan(&v.ID, &v.RestaurantID, &v.Name, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) ListHalls(ctx context.Context, restaurantID string) ([]domain.Hall, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,name,active,created_at,updated_at FROM halls WHERE restaurant_id = ? ORDER BY name`, restaurantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Hall
	for rows.Next() {
		var v domain.Hall
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.Name, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) ArchiveHall(ctx context.Context, id, updatedAt string) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE halls SET active = 0, updated_at = ? WHERE id = ?`, updatedAt, id)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateTable(ctx context.Context, v *domain.Table) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO tables(id,restaurant_id,hall_id,name,seats,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.HallID, v.Name, v.Seats, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetTable(ctx context.Context, id string) (*domain.Table, error) {
	var v domain.Table
	var active int
	var created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,hall_id,name,seats,active,created_at,updated_at FROM tables WHERE id = ?`, id).
		Scan(&v.ID, &v.RestaurantID, &v.HallID, &v.Name, &v.Seats, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) ListTables(ctx context.Context, restaurantID, hallID string) ([]domain.Table, error) {
	query := `SELECT id,restaurant_id,hall_id,name,seats,active,created_at,updated_at FROM tables WHERE restaurant_id = ?`
	args := []any{restaurantID}
	if hallID != "" {
		query += ` AND hall_id = ?`
		args = append(args, hallID)
	}
	query += ` ORDER BY hall_id, name`
	rows, err := r.queryer(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Table
	for rows.Next() {
		var v domain.Table
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.HallID, &v.Name, &v.Seats, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) ArchiveTable(ctx context.Context, id, updatedAt string) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE tables SET active = 0, updated_at = ? WHERE id = ?`, updatedAt, id)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
