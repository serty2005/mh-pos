package menu

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

type CreateMenuItemCommand struct {
	shared.CommandMeta
	CatalogItemID string `json:"catalog_item_id"`
	Name          string `json:"name"`
	Price         int64  `json:"price"`
	Currency      string `json:"currency"`
	TaxProfileID  string `json:"tax_profile_id,omitempty"`
}

func (s *Service) ListMenuItems(ctx context.Context) ([]domain.MenuItem, error) {
	return s.repo.ListMenuItems(ctx)
}

// ListMenuItemsAsOperator возвращает menu items для аутентифицированных операторских сценариев с проверкой RBAC.
func (s *Service) ListMenuItemsAsOperator(ctx context.Context, meta shared.CommandMeta) ([]domain.MenuItem, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionMenuView)); err != nil {
		return nil, err
	}
	return s.ListMenuItems(ctx)
}

func (s *Service) CreateMenuItem(ctx context.Context, cmd CreateMenuItemCommand) (*domain.MenuItem, error) {
	if err := shared.EnsureMasterDataWriteAllowed(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.CatalogItemID) == "" || strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.Currency) == "" || cmd.Price < 0 {
		return nil, fmt.Errorf("%w: catalog_item_id, name, currency and non-negative price are required", domain.ErrInvalid)
	}
	currency, err := shared.ValidateCurrencyCode(cmd.Currency)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	now := s.clock.Now()
	v := &domain.MenuItem{
		ID:            s.ids.NewID(),
		CatalogItemID: strings.TrimSpace(cmd.CatalogItemID),
		Name:          strings.TrimSpace(cmd.Name),
		Price:         cmd.Price,
		Currency:      currency,
		TaxProfileID:  optionalString(cmd.TaxProfileID),
		Active:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		catalogItem, err := s.repo.GetCatalogItem(ctx, cmd.CatalogItemID)
		if err != nil {
			return err
		}
		if !catalogItem.Active {
			return fmt.Errorf("%w: catalog item is archived", domain.ErrConflict)
		}
		if err := s.repo.CreateMenuItem(ctx, v); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, "", "", "MenuItem", v.ID, "MenuItemCreated", v)
	})
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
