package device

import (
	"context"
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

type RegisterDeviceCommand struct {
	shared.CommandMeta
	RestaurantID string `json:"restaurant_id"`
	DeviceCode   string `json:"device_code"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

func (s *Service) ListDevices(ctx context.Context) ([]domain.Device, error) {
	return s.repo.ListDevices(ctx)
}

func (s *Service) RegisterDevice(ctx context.Context, cmd RegisterDeviceCommand) (*domain.Device, error) {
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.DeviceCode) == "" || strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.Type) == "" {
		return nil, fmt.Errorf("%w: restaurant_id, device_code, name and type are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Device{ID: s.ids.NewID(), RestaurantID: cmd.RestaurantID, DeviceCode: cmd.DeviceCode, Name: cmd.Name, Type: cmd.Type, Active: true, RegisteredAt: now, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateDevice(ctx, v); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, v.RestaurantID, "", "Device", v.ID, "DeviceRegistered", v)
	})
}
