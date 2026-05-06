package sqlite

import (
	"context"
	"database/sql"
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

func (r *Repository) GetRole(ctx context.Context, id string) (*domain.Role, error) {
	var v domain.Role
	var active int
	var created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,name,permissions_json,active,created_at,updated_at FROM roles WHERE id = ?`, id).Scan(&v.ID, &v.Name, &v.PermissionsJSON, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}

func (r *Repository) CreateEmployee(ctx context.Context, v *domain.Employee) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO employees(id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.RoleID, v.Name, v.PINHash, boolInt(v.Active), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) ListEmployees(ctx context.Context) ([]domain.Employee, error) {
	return r.listEmployees(ctx, `SELECT id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at FROM employees ORDER BY created_at`)
}

func (r *Repository) ListEmployeesByRestaurant(ctx context.Context, restaurantID string) ([]domain.Employee, error) {
	return r.listEmployees(ctx, `SELECT id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at FROM employees WHERE restaurant_id = ? ORDER BY created_at`, restaurantID)
}

func (r *Repository) listEmployees(ctx context.Context, query string, args ...any) ([]domain.Employee, error) {
	rows, err := r.queryer(ctx).QueryContext(ctx, query, args...)
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

func (r *Repository) GetEmployee(ctx context.Context, id string) (*domain.Employee, error) {
	var v domain.Employee
	var active int
	var created, updated string
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,role_id,name,pin_hash,active,created_at,updated_at FROM employees WHERE id = ?`, id).Scan(&v.ID, &v.RestaurantID, &v.RoleID, &v.Name, &v.PINHash, &active, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Active = active == 1
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
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

func (r *Repository) CreateManagerOverrideAudit(ctx context.Context, v *domain.ManagerOverrideAudit) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO manager_override_audit(id,command_id,restaurant_id,device_id,shift_id,order_id,precheck_id,manager_employee_id,actor_employee_id,session_id,action,reason,occurred_at,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.CommandID, v.RestaurantID, v.DeviceID, v.ShiftID, v.OrderID, v.PrecheckID, v.ManagerEmployeeID, nullableString(v.ActorEmployeeID), nullableString(v.SessionID), v.Action, v.Reason, dbTime(v.OccurredAt), dbTime(v.CreatedAt))
	return normalizeErr(err)
}

func (r *Repository) CreateAuthSession(ctx context.Context, v *domain.AuthSession) error {
	_, err := r.execer(ctx).ExecContext(ctx, `INSERT INTO auth_sessions(id,restaurant_id,device_id,employee_id,status,started_at,last_seen_at,expires_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.RestaurantID, v.DeviceID, v.EmployeeID, string(v.Status), dbTime(v.StartedAt), dbTime(v.LastSeenAt), nullableTime(v.ExpiresAt), dbTime(v.CreatedAt), dbTime(v.UpdatedAt))
	return normalizeErr(err)
}

func (r *Repository) GetAuthSession(ctx context.Context, id string) (*domain.AuthSession, error) {
	var v domain.AuthSession
	var status, started, lastSeen, created, updated string
	var expires sql.NullString
	err := r.queryer(ctx).QueryRowContext(ctx, `SELECT id,restaurant_id,device_id,employee_id,status,started_at,last_seen_at,expires_at,created_at,updated_at FROM auth_sessions WHERE id = ?`, id).
		Scan(&v.ID, &v.RestaurantID, &v.DeviceID, &v.EmployeeID, &status, &started, &lastSeen, &expires, &created, &updated)
	if err != nil {
		return nil, normalizeErr(err)
	}
	v.Status = domain.AuthSessionStatus(status)
	v.StartedAt = parseTime(started)
	v.LastSeenAt = parseTime(lastSeen)
	v.ExpiresAt = timePtr(expires)
	v.CreatedAt = parseTime(created)
	v.UpdatedAt = parseTime(updated)
	return &v, nil
}
