package catalog

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

type CreateCatalogItemCommand struct {
	shared.CommandMeta
	Type               domain.CatalogItemType `json:"type"`
	FolderID           string                 `json:"folder_id,omitempty"`
	Name               string                 `json:"name"`
	SKU                string                 `json:"sku"`
	BaseUnit           string                 `json:"base_unit"`
	KitchenType        string                 `json:"kitchen_type,omitempty"`
	AccountingCategory string                 `json:"accounting_category,omitempty"`
}

func (s *Service) ListCatalogItems(ctx context.Context) ([]domain.CatalogItem, error) {
	return s.repo.ListCatalogItems(ctx)
}

// ListCatalogItemsAsOperator возвращает каталог для аутентифицированного операторского сценария с проверкой RBAC.
func (s *Service) ListCatalogItemsAsOperator(ctx context.Context, meta shared.CommandMeta) ([]domain.CatalogItem, error) {
	if _, err := shared.EnsureOperatorSession(ctx, s.repo, meta, string(shared.PermissionCatalogView)); err != nil {
		return nil, err
	}
	return s.ListCatalogItems(ctx)
}

func (s *Service) CreateCatalogItem(ctx context.Context, cmd CreateCatalogItemCommand) (*domain.CatalogItem, error) {
	if err := shared.EnsureMasterDataWriteAllowed(cmd.CommandMeta); err != nil {
		return nil, err
	}
	switch cmd.Type {
	case domain.CatalogItemDish, domain.CatalogItemGood, domain.CatalogItemSemiFinished, domain.CatalogItemService:
	default:
		return nil, fmt.Errorf("%w: unsupported catalog item type", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.SKU) == "" || strings.TrimSpace(cmd.BaseUnit) == "" {
		return nil, fmt.Errorf("%w: name, sku and base_unit are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	var folderID *string
	if v := strings.TrimSpace(cmd.FolderID); v != "" {
		folderID = &v
	}
	v := &domain.CatalogItem{ID: s.ids.NewID(), Type: cmd.Type, FolderID: folderID, Name: strings.TrimSpace(cmd.Name), SKU: strings.TrimSpace(cmd.SKU), BaseUnit: strings.TrimSpace(cmd.BaseUnit), KitchenType: strings.TrimSpace(cmd.KitchenType), AccountingCategory: strings.TrimSpace(cmd.AccountingCategory), Active: true, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateCatalogItem(ctx, v); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, "", "", "CatalogItem", v.ID, "CatalogItemCreated", v)
	})
}
