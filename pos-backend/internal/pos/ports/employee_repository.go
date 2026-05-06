package ports

import (
	"context"

	"pos-backend/internal/pos/domain/employee"
)

type EmployeeRepository interface {
	CreateRole(context.Context, *employee.Role) error
	GetRole(context.Context, string) (*employee.Role, error)
	ListRoles(context.Context) ([]employee.Role, error)
	CreateEmployee(context.Context, *employee.Employee) error
	GetEmployee(context.Context, string) (*employee.Employee, error)
	ListEmployees(context.Context) ([]employee.Employee, error)
	ArchiveEmployee(context.Context, string, string) error
	CreateManagerOverrideAudit(context.Context, *employee.ManagerOverrideAudit) error
}
