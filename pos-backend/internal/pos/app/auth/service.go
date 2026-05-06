package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

type LogoutCommand struct {
	shared.CommandMeta
	SessionID string `json:"session_id"`
}

func (s *Service) PinLogin(ctx context.Context, cmd PinLoginCommand) (*domain.PinLoginResult, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.ClientDeviceID) == "" {
		return nil, fmt.Errorf("%w: client_device_id is required", domain.ErrInvalid)
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
		identity, err := s.repo.GetEdgeNodeIdentity(ctx)
		if err != nil {
			return err
		}
		if identity.Status != domain.EdgeNodePaired || identity.NodeDeviceID != cmd.NodeDeviceID {
			return fmt.Errorf("%w: edge node is not paired for requested node_device_id", domain.ErrForbidden)
		}
		device, err := s.repo.GetDevice(ctx, identity.NodeDeviceID)
		if err != nil {
			return err
		}
		if !device.Active || device.RestaurantID != identity.RestaurantID {
			return fmt.Errorf("%w: node device is archived or mismatched", domain.ErrForbidden)
		}
		if err := s.ensureClientDevice(ctx, identity.RestaurantID, identity.NodeDeviceID, cmd.ClientDeviceID, now); err != nil {
			return err
		}
		employees, err := s.repo.ListEmployeesByRestaurant(ctx, identity.RestaurantID)
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
			ID:             s.ids.NewID(),
			RestaurantID:   identity.RestaurantID,
			NodeDeviceID:   identity.NodeDeviceID,
			ClientDeviceID: cmd.ClientDeviceID,
			EmployeeID:     employee.ID,
			Status:         domain.AuthSessionActive,
			StartedAt:      now,
			LastSeenAt:     now,
			CreatedAt:      now,
			UpdatedAt:      now,
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
			"node_device_id":    identity.NodeDeviceID,
			"client_device_id":  cmd.ClientDeviceID,
		})
	})
	return result, err
}

func (s *Service) Logout(ctx context.Context, cmd LogoutCommand) (*domain.AuthSession, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(cmd.SessionID)
	if sessionID == "" {
		sessionID = strings.TrimSpace(cmd.CommandMeta.SessionID)
	}
	if sessionID == "" || strings.TrimSpace(cmd.ClientDeviceID) == "" {
		return nil, fmt.Errorf("%w: session_id and client_device_id are required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	now := s.clock.Now()
	var session *domain.AuthSession
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		var err error
		session, err = s.repo.GetAuthSession(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.NodeDeviceID != cmd.NodeDeviceID || session.ClientDeviceID != cmd.ClientDeviceID {
			return fmt.Errorf("%w: session context does not match logout device", domain.ErrForbidden)
		}
		if session.Status == domain.AuthSessionActive {
			if err := s.repo.RevokeAuthSession(ctx, session.ID, shared.DBTime(now)); err != nil {
				return err
			}
			session.Status = domain.AuthSessionRevoked
			session.RevokedAt = &now
			session.UpdatedAt = now
		}
		eventMeta := cmd.CommandMeta
		eventMeta.ActorEmployeeID = session.EmployeeID
		eventMeta.SessionID = session.ID
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, eventMeta, session.RestaurantID, "", "AuthSession", session.ID, "AuthSessionRevoked", map[string]any{
			"session_id":       session.ID,
			"employee_id":      session.EmployeeID,
			"node_device_id":   session.NodeDeviceID,
			"client_device_id": session.ClientDeviceID,
		})
	})
	return session, err
}

func (s *Service) GetSession(ctx context.Context, sessionID, nodeDeviceID, clientDeviceID string) (*domain.PinLoginResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	clientDeviceID = strings.TrimSpace(clientDeviceID)
	if sessionID == "" || nodeDeviceID == "" || clientDeviceID == "" {
		return nil, fmt.Errorf("%w: session_id, node_device_id and client_device_id are required", domain.ErrInvalid)
	}
	session, err := s.repo.GetAuthSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.NodeDeviceID != nodeDeviceID || session.ClientDeviceID != clientDeviceID {
		return nil, fmt.Errorf("%w: session is not for requested device context", domain.ErrForbidden)
	}
	if session.Status == domain.AuthSessionActive {
		now := shared.DBTime(s.clock.Now())
		_ = s.repo.UpdateAuthSessionSeen(ctx, session.ID, now)
		_ = s.repo.TouchClientDevice(ctx, session.NodeDeviceID, session.ClientDeviceID, now)
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

func (s *Service) ensureClientDevice(ctx context.Context, restaurantID, nodeDeviceID, clientDeviceID string, now time.Time) error {
	if _, err := s.repo.GetClientDevice(ctx, nodeDeviceID, clientDeviceID); err == nil {
		return s.repo.TouchClientDevice(ctx, nodeDeviceID, clientDeviceID, shared.DBTime(now))
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	client := &domain.ClientDevice{
		ID:             s.ids.NewID(),
		RestaurantID:   restaurantID,
		NodeDeviceID:   nodeDeviceID,
		ClientDeviceID: clientDeviceID,
		Status:         domain.ClientDeviceActive,
		FirstSeenAt:    now,
		LastSeenAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return s.repo.CreateClientDevice(ctx, client)
}
