package restaurant

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

type CreateRestaurantCommand struct {
	shared.CommandMeta
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
	Currency string `json:"currency"`
}

func (s *Service) ListRestaurants(ctx context.Context) ([]domain.Restaurant, error) {
	return s.repo.ListRestaurants(ctx)
}

func (s *Service) CreateRestaurant(ctx context.Context, cmd CreateRestaurantCommand) (*domain.Restaurant, error) {
	if err := shared.EnsureMasterDataWriteAllowed(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.Timezone) == "" || strings.TrimSpace(cmd.Currency) == "" {
		return nil, fmt.Errorf("%w: name, timezone and currency are required", domain.ErrInvalid)
	}
	currency, err := shared.ValidateCurrencyCode(cmd.Currency)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	now := s.clock.Now()
	v := &domain.Restaurant{
		ID:        s.ids.NewID(),
		Name:      strings.TrimSpace(cmd.Name),
		Timezone:  strings.TrimSpace(cmd.Timezone),
		Currency:  currency,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateRestaurant(ctx, v); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, v.ID, "", "Restaurant", v.ID, "RestaurantCreated", v)
	})
}
