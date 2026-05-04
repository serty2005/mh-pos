package employee

import (
	"context"
	"encoding/json"
	"fmt"
	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
	"strings"
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

type CreateRoleCommand struct {
	shared.CommandMeta
	Name            string `json:"name"`
	PermissionsJSON string `json:"permissions_json"`
}

type CreateEmployeeCommand struct {
	shared.CommandMeta
	RestaurantID string `json:"restaurant_id"`
	RoleID       string `json:"role_id"`
	Name         string `json:"name"`
	PINHash      string `json:"pin_hash"`
}

type ArchiveEmployeeCommand struct {
	shared.CommandMeta
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
}

func (s *Service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.repo.ListRoles(ctx)
}

func (s *Service) ListEmployees(ctx context.Context) ([]domain.Employee, error) {
	return s.repo.ListEmployees(ctx)
}

func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCommand) (*domain.Role, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	permissions := strings.TrimSpace(cmd.PermissionsJSON)
	if permissions == "" {
		permissions = "{}"
	}
	if strings.TrimSpace(cmd.Name) == "" || !json.Valid([]byte(permissions)) {
		return nil, fmt.Errorf("%w: name and valid permissions_json are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Role{ID: s.ids.NewID(), Name: cmd.Name, PermissionsJSON: permissions, Active: true, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateRole(ctx, v); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, "", "Role", v.ID, "RoleCreated", v)
	})
}

func (s *Service) CreateEmployee(ctx context.Context, cmd CreateEmployeeCommand) (*domain.Employee, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.RoleID) == "" || strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.PINHash) == "" {
		return nil, fmt.Errorf("%w: restaurant_id, role_id, name and pin_hash are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Employee{ID: s.ids.NewID(), RestaurantID: cmd.RestaurantID, RoleID: cmd.RoleID, Name: cmd.Name, PINHash: cmd.PINHash, Active: true, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateEmployee(ctx, v); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, v.RestaurantID, "Employee", v.ID, "EmployeeCreated", v)
	})
}

func (s *Service) ArchiveEmployee(ctx context.Context, cmd ArchiveEmployeeCommand) error {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ID) == "" {
		return fmt.Errorf("%w: employee id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.ArchiveEmployee(ctx, cmd.ID, shared.DBTime(now)); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, cmd.RestaurantID, "Employee", cmd.ID, "EmployeeArchived", cmd)
	})
}
