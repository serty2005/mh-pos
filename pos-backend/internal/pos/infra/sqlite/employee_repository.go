package sqlite

import (
	"context"
	"pos-backend/internal/pos/domain"
)

func (r *Repository) CreateRole(ctx context.Context, v *domain.Role) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO roles(id,name,permissions_json,active,created_at,updated_at) VALUES (?,?,?,?,?,?)`,
		v.ID, v.Name, v.PermissionsJSON, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,name,permissions_json,active,created_at,updated_at FROM roles ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Role
	for rows.Next() {
		var v domain.Role
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.Name, &v.PermissionsJSON, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) CreateEmployee(ctx context.Context, v *domain.Employee) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO employees(id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.RoleID, v.Name, v.PINHash, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListEmployees(ctx context.Context) ([]domain.Employee, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, `SELECT id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at FROM employees ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Employee
	for rows.Next() {
		var v domain.Employee
		var active int
		var created, updated string
		if err := rows.Scan(&v.ID, &v.RestaurantID, &v.RoleID, &v.Name, &v.PINHash, &active, &created, &updated); err != nil {
			return nil, err
		}
		v.Active = active == 1
		v.CreatedAt = parseTime(created)
		v.UpdatedAt = parseTime(updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) ArchiveEmployee(ctx context.Context, id, updatedAt string) error {
	res, err := r.execer(ctx).ExecContext(ctx, `UPDATE employees SET active = 0, updated_at = ? WHERE id = ?`, updatedAt, id)
	if err != nil {
		return normalizeErr(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
