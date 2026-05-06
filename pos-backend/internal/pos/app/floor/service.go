package floor

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

type CreateHallCommand struct {
	shared.CommandMeta
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
}

type ArchiveHallCommand struct {
	shared.CommandMeta
	ID string `json:"id"`
}

type CreateTableCommand struct {
	shared.CommandMeta
	RestaurantID string `json:"restaurant_id"`
	HallID       string `json:"hall_id"`
	Name         string `json:"name"`
	Seats        int    `json:"seats"`
}

type ArchiveTableCommand struct {
	shared.CommandMeta
	ID string `json:"id"`
}

func (s *Service) CreateHall(ctx context.Context, cmd CreateHallCommand) (*domain.Hall, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.Name) == "" {
		return nil, fmt.Errorf("%w: restaurant_id and name are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	hall := &domain.Hall{ID: s.ids.NewID(), RestaurantID: strings.TrimSpace(cmd.RestaurantID), Name: strings.TrimSpace(cmd.Name), Active: true, CreatedAt: now, UpdatedAt: now}
	return hall, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta); err != nil {
			return err
		}
		if err := s.repo.CreateHall(ctx, hall); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, hall.RestaurantID, "", "Hall", hall.ID, "HallCreated", hall)
	})
}

func (s *Service) ListHalls(ctx context.Context, restaurantID string) ([]domain.Hall, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return nil, fmt.Errorf("%w: restaurant_id is required", domain.ErrInvalid)
	}
	return s.repo.ListHalls(ctx, restaurantID)
}

func (s *Service) ArchiveHall(ctx context.Context, cmd ArchiveHallCommand) error {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ID) == "" {
		return fmt.Errorf("%w: hall id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta); err != nil {
			return err
		}
		hall, err := s.repo.GetHall(ctx, cmd.ID)
		if err != nil {
			return err
		}
		if err := s.repo.ArchiveHall(ctx, hall.ID, shared.DBTime(now)); err != nil {
			return err
		}
		hall.Active = false
		hall.UpdatedAt = now
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, hall.RestaurantID, "", "Hall", hall.ID, "HallArchived", hall)
	})
}

func (s *Service) CreateTable(ctx context.Context, cmd CreateTableCommand) (*domain.Table, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.HallID) == "" || strings.TrimSpace(cmd.Name) == "" || cmd.Seats < 0 {
		return nil, fmt.Errorf("%w: restaurant_id, hall_id, name and non-negative seats are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	table := &domain.Table{ID: s.ids.NewID(), RestaurantID: strings.TrimSpace(cmd.RestaurantID), HallID: strings.TrimSpace(cmd.HallID), Name: strings.TrimSpace(cmd.Name), Seats: cmd.Seats, Active: true, CreatedAt: now, UpdatedAt: now}
	return table, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta); err != nil {
			return err
		}
		hall, err := s.repo.GetHall(ctx, table.HallID)
		if err != nil {
			return err
		}
		if !hall.Active || hall.RestaurantID != table.RestaurantID {
			return fmt.Errorf("%w: table hall is not active for restaurant", domain.ErrConflict)
		}
		if err := s.repo.CreateTable(ctx, table); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, table.RestaurantID, "", "Table", table.ID, "TableCreated", table)
	})
}

func (s *Service) ListTables(ctx context.Context, restaurantID, hallID string) ([]domain.Table, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return nil, fmt.Errorf("%w: restaurant_id is required", domain.ErrInvalid)
	}
	return s.repo.ListTables(ctx, restaurantID, strings.TrimSpace(hallID))
}

func (s *Service) ArchiveTable(ctx context.Context, cmd ArchiveTableCommand) error {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ID) == "" {
		return fmt.Errorf("%w: table id is required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if _, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta); err != nil {
			return err
		}
		table, err := s.repo.GetTable(ctx, cmd.ID)
		if err != nil {
			return err
		}
		if err := s.repo.ArchiveTable(ctx, table.ID, shared.DBTime(now)); err != nil {
			return err
		}
		table.Active = false
		table.UpdatedAt = now
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, table.RestaurantID, "", "Table", table.ID, "TableArchived", table)
	})
}
