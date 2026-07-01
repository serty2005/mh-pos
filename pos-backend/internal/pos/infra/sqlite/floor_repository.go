package sqlite

import (
	"context"
	"database/sql"

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
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO tables(id,restaurant_id,hall_id,section_id,name,seats,is_default,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, nullableStringValue(v.HallID), v.SectionID, v.Name, v.Seats, boolInt(v.IsDefault), boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetTable(ctx context.Context, id string) (*domain.Table, error) {
	var v domain.Table
	var active, isDefault int
	var created, updated string
	var hallID sql.NullString
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,hall_id,section_id,name,seats,is_default,active,created_at,updated_at FROM tables WHERE id = ?`, id).
		Scan(&v.ID, &v.RestaurantID, &hallID, &v.SectionID, &v.Name, &v.Seats, &isDefault, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	if hallID.Valid {
		v.HallID = hallID.String
	}
	v.IsDefault = isDefault == 1
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) ListTables(ctx context.Context, restaurantID, hallID string) ([]domain.Table, error) {
	query := `SELECT id,restaurant_id,hall_id,section_id,name,seats,is_default,active,created_at,updated_at FROM tables WHERE restaurant_id = ?`
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
		var active, isDefault int
		var created, updated string
		var hallID sql.NullString
		if err := rows.Scan(&v.ID, &v.RestaurantID, &hallID, &v.SectionID, &v.Name, &v.Seats, &isDefault, &active, &created, &updated); err != nil {
			return nil, err
		}
		if hallID.Valid {
			v.HallID = hallID.String
		}
		v.IsDefault = isDefault == 1
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

func (r *Repository) GetRestaurantSection(ctx context.Context, id string) (*domain.RestaurantSection, error) {
	return scanRestaurantSection(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,name,mode,hall_id,kitchen_routing_key,warehouse_id,is_default,is_active,cloud_version,synced_at,created_at,updated_at FROM restaurant_sections WHERE id = ?`, id))
}

func (r *Repository) ListRestaurantSections(ctx context.Context, restaurantID string) ([]domain.RestaurantSection, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,name,mode,hall_id,kitchen_routing_key,warehouse_id,is_default,is_active,cloud_version,synced_at,created_at,updated_at FROM restaurant_sections WHERE restaurant_id = ? ORDER BY mode,name,id`, restaurantID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	out := []domain.RestaurantSection{}
	for rows.Next() {
		v, err := scanRestaurantSection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func scanRestaurantSection(row scanner) (*domain.RestaurantSection, error) {
	var v domain.RestaurantSection
	var mode string
	var hallID, kitchenRoutingKey, warehouseID, synced, created, updated sql.NullString
	var isDefault, isActive int
	if err := row.Scan(&v.ID, &v.RestaurantID, &v.Name, &mode, &hallID, &kitchenRoutingKey, &warehouseID, &isDefault, &isActive, &v.CloudVersion, &synced, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.Mode = domain.RestaurantSectionMode(mode)
	if hallID.Valid {
		v.HallID = hallID.String
	}
	if kitchenRoutingKey.Valid {
		v.KitchenRoutingKey = kitchenRoutingKey.String
	}
	if warehouseID.Valid {
		v.WarehouseID = warehouseID.String
	}
	v.IsDefault = isDefault == 1
	v.IsActive = isActive == 1
	if synced.Valid {
		t := parseTime(synced.String)
		v.SyncedAt = &t
	}
	if created.Valid {
		v.CreatedAt = parseTime(created.String)
	}
	if updated.Valid {
		v.UpdatedAt = parseTime(updated.String)
	}
	return &v, nil
}

func (r *Repository) GetSalesPoint(ctx context.Context, id string) (*domain.SalesPoint, error) {
	return scanSalesPoint(r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,name,analytics_tag,default_table_id,is_active,cloud_version,synced_at,created_at,updated_at FROM sales_points WHERE id = ?`, id))
}

func (r *Repository) ListSalesPoints(ctx context.Context, restaurantID string) ([]domain.SalesPoint, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,name,analytics_tag,default_table_id,is_active,cloud_version,synced_at,created_at,updated_at FROM sales_points WHERE restaurant_id = ? ORDER BY name,id`, restaurantID)
	if err != nil {
		return nil, normalizeErr(err)
	}
	defer rows.Close()
	out := []domain.SalesPoint{}
	for rows.Next() {
		v, err := scanSalesPoint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, normalizeErr(rows.Err())
}

func scanSalesPoint(row scanner) (*domain.SalesPoint, error) {
	var v domain.SalesPoint
	var active int
	var synced, created, updated sql.NullString
	if err := row.Scan(&v.ID, &v.RestaurantID, &v.Name, &v.AnalyticsTag, &v.DefaultTableID, &active, &v.CloudVersion, &synced, &created, &updated); err != nil {
		return nil, normalizeErr(err)
	}
	v.IsActive = active == 1
	if synced.Valid {
		t := parseTime(synced.String)
		v.SyncedAt = &t
	}
	if created.Valid {
		v.CreatedAt = parseTime(created.String)
	}
	if updated.Valid {
		v.UpdatedAt = parseTime(updated.String)
	}
	return &v, nil
}
