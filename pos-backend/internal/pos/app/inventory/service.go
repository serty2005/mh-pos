package inventory

import (
	"context"
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

// Service задает отдельную application boundary для ручных складских документов.
type Service struct {
	repo  ports.Repository
	tx    txmanager.Manager
	ids   idgen.Generator
	clock clock.Clock
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock}
}

// CreateManualStockDocumentCommand создает posted stock document и immutable moves без побочных эффектов из продаж.
type CreateManualStockDocumentCommand struct {
	shared.CommandMeta
	RestaurantID         string                   `json:"restaurant_id"`
	DocumentType         domain.StockDocumentType `json:"document_type"`
	SourceType           string                   `json:"source_type,omitempty"`
	SourceID             string                   `json:"source_id,omitempty"`
	CheckID              string                   `json:"check_id,omitempty"`
	PrecheckID           string                   `json:"precheck_id,omitempty"`
	FinancialOperationID string                   `json:"financial_operation_id,omitempty"`
	BusinessDateLocal    string                   `json:"business_date_local,omitempty"`
	ShiftID              string                   `json:"shift_id,omitempty"`
	CashSessionID        string                   `json:"cash_session_id,omitempty"`
	OccurredAt           time.Time                `json:"occurred_at,omitempty"`
	ApplyToBalance       bool                     `json:"apply_to_balance"`
	Moves                []CreateStockMoveCommand `json:"moves"`
}

// CreateStockMoveCommand описывает одну append-only строку движения внутри stock document.
type CreateStockMoveCommand struct {
	CatalogItemID string               `json:"catalog_item_id"`
	OrderLineID   string               `json:"order_line_id,omitempty"`
	LocationID    string               `json:"location_id,omitempty"`
	MovementType  domain.StockMoveType `json:"movement_type"`
	Quantity      int64                `json:"quantity"`
	Unit          string               `json:"unit"`
	UnitCost      *int64               `json:"unit_cost,omitempty"`
	TotalCost     *int64               `json:"total_cost,omitempty"`
}

// CreateManualStockDocument записывает stock document, moves и, если явно запрошено, balance delta в одной транзакции.
func (s *Service) CreateManualStockDocument(ctx context.Context, cmd CreateManualStockDocumentCommand) (*domain.StockDocument, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.CommandID) == "" {
		cmd.CommandID = s.ids.NewID()
	}
	if err := validateDocumentCommand(cmd); err != nil {
		return nil, err
	}
	now := s.clock.Now()
	occurredAt := cmd.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = now
	}
	var document *domain.StockDocument
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if err := s.validateMoveItems(ctx, cmd.Moves); err != nil {
			return err
		}
		document = &domain.StockDocument{
			ID:                   s.ids.NewID(),
			RestaurantID:         strings.TrimSpace(cmd.RestaurantID),
			DeviceID:             cmd.DeviceID,
			Type:                 cmd.DocumentType,
			SourceType:           optionalString(cmd.SourceType),
			SourceID:             optionalString(cmd.SourceID),
			CheckID:              optionalString(cmd.CheckID),
			PrecheckID:           optionalString(cmd.PrecheckID),
			FinancialOperationID: optionalString(cmd.FinancialOperationID),
			BusinessDateLocal:    strings.TrimSpace(cmd.BusinessDateLocal),
			ShiftID:              optionalString(cmd.ShiftID),
			CashSessionID:        optionalString(cmd.CashSessionID),
			CreatedByEmployeeID:  optionalString(cmd.ActorEmployeeID),
			Status:               domain.StockDocumentPosted,
			OccurredAt:           occurredAt,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if err := s.repo.CreateStockDocument(ctx, document); err != nil {
			return err
		}
		for _, input := range cmd.Moves {
			move := stockMoveFromCommand(s.ids.NewID(), document.ID, occurredAt, now, input)
			if err := s.repo.CreateStockMove(ctx, move); err != nil {
				return err
			}
			if cmd.ApplyToBalance {
				if err := s.applyBalanceDelta(ctx, move, now); err != nil {
					return err
				}
			}
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, document.RestaurantID, strings.TrimSpace(cmd.ShiftID), "StockDocument", document.ID, "StockDocumentPosted", document)
	})
	return document, err
}

func validateDocumentCommand(cmd CreateManualStockDocumentCommand) error {
	if strings.TrimSpace(cmd.RestaurantID) == "" || len(cmd.Moves) == 0 {
		return fmt.Errorf("%w: restaurant_id and moves are required", domain.ErrInvalid)
	}
	switch cmd.DocumentType {
	case domain.StockDocumentAdjustment, domain.StockDocumentTransfer, domain.StockDocumentWriteOff, domain.StockDocumentProduction, domain.StockDocumentPurchaseReceipt:
	default:
		return fmt.Errorf("%w: unsupported stock document type", domain.ErrInvalid)
	}
	for _, move := range cmd.Moves {
		if strings.TrimSpace(move.CatalogItemID) == "" || strings.TrimSpace(move.Unit) == "" {
			return fmt.Errorf("%w: move catalog_item_id and unit are required", domain.ErrInvalid)
		}
		switch move.MovementType {
		case domain.StockMoveIn, domain.StockMoveOut:
			if move.Quantity <= 0 {
				return fmt.Errorf("%w: in/out stock move quantity must be positive", domain.ErrInvalid)
			}
		case domain.StockMoveAdjustment:
			if move.Quantity == 0 {
				return fmt.Errorf("%w: adjustment stock move quantity must be non-zero", domain.ErrInvalid)
			}
		default:
			return fmt.Errorf("%w: unsupported stock move type", domain.ErrInvalid)
		}
		if move.UnitCost != nil && *move.UnitCost < 0 {
			return fmt.Errorf("%w: unit_cost must be non-negative", domain.ErrInvalid)
		}
		if move.TotalCost != nil && *move.TotalCost < 0 {
			return fmt.Errorf("%w: total_cost must be non-negative", domain.ErrInvalid)
		}
	}
	return nil
}

func (s *Service) validateMoveItems(ctx context.Context, moves []CreateStockMoveCommand) error {
	for _, move := range moves {
		item, err := s.repo.GetCatalogItem(ctx, strings.TrimSpace(move.CatalogItemID))
		if err != nil {
			return err
		}
		if item.Type == domain.CatalogItemService {
			return fmt.Errorf("%w: service catalog items do not have stock moves", domain.ErrInvalid)
		}
	}
	return nil
}

func stockMoveFromCommand(id, documentID string, occurredAt, createdAt time.Time, input CreateStockMoveCommand) *domain.StockMove {
	return &domain.StockMove{
		ID:              id,
		StockDocumentID: documentID,
		CatalogItemID:   strings.TrimSpace(input.CatalogItemID),
		OrderLineID:     optionalString(input.OrderLineID),
		LocationID:      optionalString(input.LocationID),
		Type:            input.MovementType,
		Quantity:        input.Quantity,
		Unit:            strings.TrimSpace(input.Unit),
		UnitCost:        input.UnitCost,
		TotalCost:       input.TotalCost,
		OccurredAt:      occurredAt,
		CreatedAt:       createdAt,
	}
}

func (s *Service) applyBalanceDelta(ctx context.Context, move *domain.StockMove, now time.Time) error {
	balance, err := s.findBalance(ctx, move.CatalogItemID, move.LocationID)
	if err != nil {
		return err
	}
	delta := move.Quantity
	if move.Type == domain.StockMoveOut {
		delta = -move.Quantity
	}
	if balance == nil {
		balance = &domain.StockBalance{
			ID:            s.ids.NewID(),
			CatalogItemID: move.CatalogItemID,
			LocationID:    move.LocationID,
			Unit:          move.Unit,
		}
	} else if !strings.EqualFold(strings.TrimSpace(balance.Unit), strings.TrimSpace(move.Unit)) {
		return fmt.Errorf("%w: stock balance unit mismatch", domain.ErrConflict)
	}
	balance.Quantity += delta
	balance.UpdatedAt = now
	return s.repo.UpsertStockBalance(ctx, balance)
}

func (s *Service) findBalance(ctx context.Context, catalogItemID string, locationID *string) (*domain.StockBalance, error) {
	balances, err := s.repo.ListStockBalances(ctx)
	if err != nil {
		return nil, err
	}
	for i := range balances {
		if balances[i].CatalogItemID == catalogItemID && sameOptionalString(balances[i].LocationID, locationID) {
			return &balances[i], nil
		}
	}
	return nil, nil
}

func optionalString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func sameOptionalString(a, b *string) bool {
	if a == nil || strings.TrimSpace(*a) == "" {
		return b == nil || strings.TrimSpace(*b) == ""
	}
	return b != nil && strings.TrimSpace(*a) == strings.TrimSpace(*b)
}
