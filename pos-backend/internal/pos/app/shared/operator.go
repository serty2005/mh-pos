package shared

import (
	"context"
	"fmt"
	"strings"

	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
)

type OperatorContext struct {
	Session     *domain.AuthSession
	Employee    *domain.Employee
	Role        *domain.Role
	Permissions []string
}

func EnsureOperatorSession(ctx context.Context, repo ports.Repository, meta CommandMeta, requiredPermissions ...string) (*OperatorContext, error) {
	NormalizeDeviceMeta(&meta)
	if err := ValidateWriteMeta(meta); err != nil {
		return nil, err
	}
	if meta.Origin == domain.OriginSystemSeed || meta.Origin == domain.OriginCloudSync {
		return nil, nil
	}
	if strings.TrimSpace(meta.ClientDeviceID) == "" || strings.TrimSpace(meta.ActorEmployeeID) == "" || strings.TrimSpace(meta.SessionID) == "" {
		return nil, fmt.Errorf("%w: client_device_id, actor_employee_id and session_id are required for operator flow", domain.ErrInvalid)
	}
	session, err := repo.GetAuthSession(ctx, meta.SessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != domain.AuthSessionActive {
		return nil, fmt.Errorf("%w: session is not active", domain.ErrForbidden)
	}
	if session.NodeDeviceID != meta.NodeDeviceID || session.ClientDeviceID != meta.ClientDeviceID || session.EmployeeID != meta.ActorEmployeeID {
		return nil, fmt.Errorf("%w: session context does not match command actor/device", domain.ErrForbidden)
	}
	employee, err := repo.GetEmployee(ctx, session.EmployeeID)
	if err != nil {
		return nil, err
	}
	if !employee.Active {
		return nil, fmt.Errorf("%w: employee is archived", domain.ErrForbidden)
	}
	role, err := repo.GetRole(ctx, employee.RoleID)
	if err != nil {
		return nil, err
	}
	if !role.Active {
		return nil, fmt.Errorf("%w: employee role is archived", domain.ErrForbidden)
	}
	for _, permission := range requiredPermissions {
		if strings.TrimSpace(permission) != "" && !HasPermission(role.PermissionsJSON, permission) {
			return nil, fmt.Errorf("%w: permission %s is required", domain.ErrForbidden, permission)
		}
	}
	return &OperatorContext{
		Session:     session,
		Employee:    employee,
		Role:        role,
		Permissions: PermissionsFromJSON(role.PermissionsJSON),
	}, nil
}
