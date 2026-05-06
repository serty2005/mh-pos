package sqlite

import (
	"context"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateDevice(ctx context.Context, v *domain.Device) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO devices(id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceCode, v.Name, v.Type, boolInt(v.Active), dbTime(v.RegisteredAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetDevice(ctx context.Context, id string) (*domain.Device, error) {
	var v domain.Device
	var active int
	var registered, created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at FROM devices WHERE id = ?`, id).
		Scan(&v.ID, &v.RestaurantID, &v.DeviceCode, &v.Name, &v.Type, &active, &registered, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.RegisteredAt = parseTime(registered)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) ListDevices(ctx context.Context) ([]domain.Device, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,device_code,name,type,active,registered_at,created_at,updated_at FROM devices ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Device
	for rows.Next() {
		var v domain.Device
		var active int
		var registered, created, updated string
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.DeviceCode, &v.Name, &v.Type, &active, &registered, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.RegisteredAt = parseTime(registered)
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}
