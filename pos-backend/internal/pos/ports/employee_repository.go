package ports

import (
	"context"

	"pos-backend/internal/pos/domain/employee"
)

type EmployeeRepository interface {
	CreateRole(context.Context, *employee.Role) error
	ListRoles(context.Context) ([]employee.Role, error)
	CreateEmployee(context.Context, *employee.Employee) error
	ListEmployees(context.Context) ([]employee.Employee, error)
	ArchiveEmployee(context.Context, string, string) error
}
