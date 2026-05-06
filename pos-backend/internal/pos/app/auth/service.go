package auth

import (
	"context"
	"fmt"
	"strings"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
)

type Service struct {
	repo  ports.Repository
	tx    txmanager.Manager
	ids   idgen.Generator
	clock clock.Clock
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock}
}

type PinLoginCommand struct {
	shared.CommandMeta
	PIN string `json:"pin"`
}

func (s *Service) PinLogin(ctx context.Context, cmd PinLoginCommand) (*domain.PinLoginResult, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.PIN) == "" {
		return nil, fmt.Errorf("%w: pin is required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var result *domain.PinLoginResult
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		device, err := s.repo.GetDevice(ctx, cmd.DeviceID)
		if err != nil {
			return err
		}
		if !device.Active {
			return fmt.Errorf("%w: device is archived", domain.ErrForbidden)
		}
		employees, err := s.repo.ListEmployeesByRestaurant(ctx, device.RestaurantID)
		if err != nil {
			return err
		}
		var employee *domain.Employee
		for i := range employees {
			item := employees[i]
			if !item.Active {
				continue
			}
			if err := shared.VerifyPIN(item.PINHash, cmd.PIN); err == nil {
				employee = &item
				break
			}
		}
		if employee == nil {
			return fmt.Errorf("%w: pin is invalid", domain.ErrForbidden)
		}
		role, err := s.repo.GetRole(ctx, employee.RoleID)
		if err != nil {
			return err
		}
		if !role.Active {
			return fmt.Errorf("%w: employee role is archived", domain.ErrForbidden)
		}
		session := &domain.AuthSession{
			ID:           s.ids.NewID(),
			RestaurantID: device.RestaurantID,
			DeviceID:     device.ID,
			EmployeeID:   employee.ID,
			Status:       domain.AuthSessionActive,
			StartedAt:    now,
			LastSeenAt:   now,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.repo.CreateAuthSession(ctx, session); err != nil {
			return err
		}
		permissions := shared.PermissionsFromJSON(role.PermissionsJSON)
		actor := domain.ActorContext{
			EmployeeID:   employee.ID,
			RestaurantID: employee.RestaurantID,
			RoleID:       employee.RoleID,
			Name:         employee.Name,
			Permissions:  permissions,
		}
		result = &domain.PinLoginResult{Session: *session, Actor: actor, Permissions: permissions}
		eventMeta := cmd.CommandMeta
		eventMeta.ActorEmployeeID = employee.ID
		eventMeta.SessionID = session.ID
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, eventMeta, device.RestaurantID, "", "AuthSession", session.ID, "AuthSessionStarted", map[string]any{
			"session_id":        session.ID,
			"employee_id":       employee.ID,
			"role_id":           employee.RoleID,
			"actor_employee_id": employee.ID,
		})
	})
	return result, err
}

func (s *Service) GetSession(ctx context.Context, sessionID, deviceID string) (*domain.PinLoginResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	deviceID = strings.TrimSpace(deviceID)
	if sessionID == "" || deviceID == "" {
		return nil, fmt.Errorf("%w: session_id and device_id are required", domain.ErrInvalid)
	}
	session, err := s.repo.GetAuthSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.DeviceID != deviceID || session.Status != domain.AuthSessionActive {
		return nil, fmt.Errorf("%w: session is not active for device", domain.ErrForbidden)
	}
	employee, err := s.repo.GetEmployee(ctx, session.EmployeeID)
	if err != nil {
		return nil, err
	}
	role, err := s.repo.GetRole(ctx, employee.RoleID)
	if err != nil {
		return nil, err
	}
	permissions := shared.PermissionsFromJSON(role.PermissionsJSON)
	actor := domain.ActorContext{
		EmployeeID:   employee.ID,
		RestaurantID: employee.RestaurantID,
		RoleID:       employee.RoleID,
		Name:         employee.Name,
		Permissions:  permissions,
	}
	return &domain.PinLoginResult{Session: *session, Actor: actor, Permissions: permissions}, nil
}
