package kitchen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	kitchendomain "pos-backend/internal/pos/domain/kitchen"
	"pos-backend/internal/pos/ports"
)

type Service struct {
	repo    ports.Repository
	tx      txmanager.Manager
	ids     idgen.Generator
	clock   clock.Clock
	options Options
}

// Options задает runtime limits для Edge kitchen proposal validation.
type Options struct {
	RecipeSuggestionMaxPrepTimeDeltaMinutes int
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return NewServiceWithOptions(repo, tx, ids, clock, Options{})
}

func NewServiceWithOptions(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock, options Options) *Service {
	if options.RecipeSuggestionMaxPrepTimeDeltaMinutes <= 0 {
		options.RecipeSuggestionMaxPrepTimeDeltaMinutes = 120
	}
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock, options: options}
}

type ListTicketsCommand struct {
	shared.CommandMeta
	Status  kitchendomain.TicketStatus `json:"status,omitempty"`
	Station string                     `json:"station,omitempty"`
	Limit   int                        `json:"limit,omitempty"`
	Offset  int                        `json:"offset,omitempty"`
}

type ListOrderQueueCommand struct {
	shared.CommandMeta
	Status  kitchendomain.OrderStatus `json:"status,omitempty"`
	Station string                    `json:"station,omitempty"`
	Limit   int                       `json:"limit,omitempty"`
	Offset  int                       `json:"offset,omitempty"`
}

type ChangeTicketStatusCommand struct {
	shared.CommandMeta
	TicketID string `json:"ticket_id"`
	Action   string `json:"action"`
}

type StockReceiptLineCommand struct {
	LineID              string `json:"line_id"`
	CatalogItemID       string `json:"catalog_item_id,omitempty"`
	CatalogSuggestionID string `json:"catalog_suggestion_id,omitempty"`
	NameSnapshot        string `json:"name_snapshot"`
	Quantity            string `json:"quantity"`
	UnitCode            string `json:"unit_code"`
	UnitCostMinor       int64  `json:"unit_cost_minor"`
	LineTotalMinor      int64  `json:"line_total_minor"`
	Currency            string `json:"currency"`
}

type CaptureStockReceiptCommand struct {
	shared.CommandMeta
	ReceiptID              string                    `json:"receipt_id"`
	WarehouseID            string                    `json:"warehouse_id,omitempty"`
	SupplierCounterpartyID string                    `json:"supplier_counterparty_id"`
	SupplierNameSnapshot   string                    `json:"supplier_name_snapshot"`
	DocumentNumber         string                    `json:"document_number"`
	DocumentDate           string                    `json:"document_date"`
	ReceivedAt             time.Time                 `json:"received_at"`
	BusinessDateLocal      string                    `json:"business_date_local"`
	Currency               string                    `json:"currency"`
	Items                  []StockReceiptLineCommand `json:"items"`
}

type InventoryCountLineCommand struct {
	LineID          string `json:"line_id"`
	CatalogItemID   string `json:"catalog_item_id"`
	CountedQuantity string `json:"counted_quantity"`
	UnitCode        string `json:"unit_code"`
}

type CaptureInventoryCountCommand struct {
	shared.CommandMeta
	CountID           string                      `json:"count_id"`
	WarehouseID       string                      `json:"warehouse_id,omitempty"`
	CountedAt         time.Time                   `json:"counted_at"`
	BusinessDateLocal string                      `json:"business_date_local"`
	Items             []InventoryCountLineCommand `json:"items"`
}

type StockWriteOffLineCommand struct {
	LineID        string `json:"line_id"`
	CatalogItemID string `json:"catalog_item_id"`
	Quantity      string `json:"quantity"`
	UnitCode      string `json:"unit_code"`
}

type CaptureStockWriteOffCommand struct {
	shared.CommandMeta
	WriteOffID        string                     `json:"write_off_id"`
	WarehouseID       string                     `json:"warehouse_id,omitempty"`
	WrittenOffAt      time.Time                  `json:"written_off_at"`
	BusinessDateLocal string                     `json:"business_date_local"`
	ReasonCode        string                     `json:"reason_code"`
	Reason            string                     `json:"reason"`
	Items             []StockWriteOffLineCommand `json:"items"`
}

type CompleteProductionCommand struct {
	shared.CommandMeta
	ProductionID              string    `json:"production_id"`
	WarehouseID               string    `json:"warehouse_id,omitempty"`
	SemiFinishedCatalogItemID string    `json:"semi_finished_catalog_item_id"`
	Quantity                  string    `json:"quantity"`
	UnitCode                  string    `json:"unit_code"`
	CompletedAt               time.Time `json:"completed_at"`
	BusinessDateLocal         string    `json:"business_date_local"`
}

type UpdateStopListCommand struct {
	shared.CommandMeta
	StopListID        string   `json:"stop_list_id,omitempty"`
	WarehouseID       string   `json:"warehouse_id,omitempty"`
	CatalogItemID     string   `json:"catalog_item_id"`
	AvailableQuantity *float64 `json:"available_quantity,omitempty"`
	Active            bool     `json:"active"`
	Reason            string   `json:"reason,omitempty"`
}

type ListStopListStateCommand struct {
	shared.CommandMeta
}

type StockCommandResult struct {
	ID          string `json:"id"`
	WarehouseID string `json:"warehouse_id"`
	EventType   string `json:"event_type"`
	Replayed    bool   `json:"replayed"`
}

type StopListState struct {
	ID                string     `json:"id"`
	CatalogItemID     string     `json:"catalog_item_id"`
	AvailableQuantity *float64   `json:"available_quantity,omitempty"`
	Source            string     `json:"source"`
	Reason            *string    `json:"reason,omitempty"`
	Active            bool       `json:"active"`
	UpdatedAt         time.Time  `json:"updated_at"`
	SyncState         string     `json:"sync_state"`
	OutboxCommandID   string     `json:"outbox_command_id,omitempty"`
	OutboxStatus      string     `json:"outbox_status,omitempty"`
	OutboxSequenceNo  int64      `json:"outbox_sequence_no,omitempty"`
	OutboxAttempts    int        `json:"outbox_attempts,omitempty"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	NextRetryAt       *time.Time `json:"next_retry_at,omitempty"`
}

type GetRecipeCommand struct {
	shared.CommandMeta
	CatalogItemID string `json:"catalog_item_id"`
}

type RecipeIngredientView struct {
	LineID          string `json:"line_id"`
	CatalogItemID   string `json:"catalog_item_id"`
	CatalogItemName string `json:"catalog_item_name"`
	Quantity        string `json:"quantity"`
	UnitCode        string `json:"unit_code"`
	LossPercent     string `json:"loss_percent"`
	CloudVersion    int64  `json:"cloud_version,omitempty"`
	CloudUpdatedAt  string `json:"cloud_updated_at,omitempty"`
}

type RecipeView struct {
	CatalogItem   domain.CatalogItem       `json:"catalog_item"`
	RecipeVersion domain.RecipeVersion     `json:"recipe_version"`
	Ingredients   []RecipeIngredientView   `json:"ingredients"`
	Proposals     []kitchendomain.Proposal `json:"proposals"`
}

type CreateCatalogSuggestionCommand struct {
	shared.CommandMeta
	SuggestionID       string                 `json:"suggestion_id,omitempty"`
	ProposalGroupID    string                 `json:"proposal_group_id,omitempty"`
	Action             string                 `json:"action"`
	CatalogItemID      string                 `json:"catalog_item_id,omitempty"`
	Kind               domain.CatalogItemType `json:"kind"`
	Name               string                 `json:"name"`
	SKU                string                 `json:"sku,omitempty"`
	BaseUnit           string                 `json:"base_unit"`
	KitchenType        string                 `json:"kitchen_type,omitempty"`
	AccountingCategory string                 `json:"accounting_category,omitempty"`
	Reason             string                 `json:"reason"`
}

type RecipeSuggestionChangeCommand struct {
	LineID            string `json:"line_id,omitempty"`
	Action            string `json:"action"`
	FromCatalogItemID string `json:"from_catalog_item_id,omitempty"`
	ToCatalogItemID   string `json:"to_catalog_item_id,omitempty"`
	Quantity          string `json:"quantity,omitempty"`
	UnitCode          string `json:"unit_code,omitempty"`
	LossPercent       string `json:"loss_percent,omitempty"`
}

type CreateRecipeSuggestionCommand struct {
	shared.CommandMeta
	SuggestionID             string                          `json:"suggestion_id,omitempty"`
	ProposalGroupID          string                          `json:"proposal_group_id,omitempty"`
	RecipeVersionID          string                          `json:"recipe_version_id,omitempty"`
	OwnerCatalogItemID       string                          `json:"owner_catalog_item_id,omitempty"`
	OwnerCatalogSuggestionID string                          `json:"owner_catalog_suggestion_id,omitempty"`
	Action                   string                          `json:"action"`
	PrepTimeDeltaMinutes     int                             `json:"prep_time_delta_minutes,omitempty"`
	Changes                  []RecipeSuggestionChangeCommand `json:"changes,omitempty"`
	Reason                   string                          `json:"reason"`
}

type ListProposalsCommand struct {
	shared.CommandMeta
	Kind   kitchendomain.ProposalKind   `json:"kind,omitempty"`
	Status kitchendomain.ProposalStatus `json:"status,omitempty"`
	Limit  int                          `json:"limit,omitempty"`
	Offset int                          `json:"offset,omitempty"`
}

func (s *Service) ListTickets(ctx context.Context, cmd ListTicketsCommand) ([]kitchendomain.Ticket, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenView))
	if err != nil {
		return nil, err
	}
	if cmd.Status != "" && !validStatus(cmd.Status) {
		return nil, fmt.Errorf("%w: unsupported kitchen status", domain.ErrInvalid)
	}
	return s.repo.ListKitchenTickets(ctx, kitchendomain.TicketListQuery{
		RestaurantID: operator.Employee.RestaurantID,
		Status:       cmd.Status,
		Station:      cmd.Station,
		Limit:        cmd.Limit,
		Offset:       cmd.Offset,
	})
}

func (s *Service) ListOrderQueue(ctx context.Context, cmd ListOrderQueueCommand) (kitchendomain.OrderQueue, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenView))
	if err != nil {
		return kitchendomain.OrderQueue{}, err
	}
	if cmd.Status != "" && !validOrderStatus(cmd.Status) {
		return kitchendomain.OrderQueue{}, fmt.Errorf("%w: unsupported kitchen order status", domain.ErrInvalid)
	}
	limit, offset := normalizeLimitOffset(cmd.Limit, cmd.Offset)
	rows, err := s.repo.ListKitchenOrderQueueTickets(ctx, kitchendomain.OrderQueueQuery{
		RestaurantID: operator.Employee.RestaurantID,
		Station:      strings.TrimSpace(cmd.Station),
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return kitchendomain.OrderQueue{}, err
	}
	orders := buildOrderQueue(rows, s.clock.Now(), cmd.Status)
	if offset > len(orders) {
		orders = nil
	} else {
		orders = orders[offset:]
	}
	if len(orders) > limit {
		orders = orders[:limit]
	}
	return kitchendomain.OrderQueue{Orders: orders, Limit: limit, Offset: offset}, nil
}

func (s *Service) GetRecipe(ctx context.Context, cmd GetRecipeCommand) (RecipeView, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenRecipeView))
	if err != nil {
		return RecipeView{}, err
	}
	catalogItemID := strings.TrimSpace(cmd.CatalogItemID)
	if catalogItemID == "" {
		return RecipeView{}, fmt.Errorf("%w: catalog_item_id is required", domain.ErrInvalid)
	}
	item, err := s.repo.GetCatalogItem(ctx, catalogItemID)
	if err != nil {
		return RecipeView{}, err
	}
	if !item.Active || (item.Type != domain.CatalogItemDish && item.Type != domain.CatalogItemSemiFinished) {
		return RecipeView{}, fmt.Errorf("%w: kitchen recipe not found", domain.ErrNotFound)
	}
	recipe, err := s.repo.GetActiveRecipeVersionByCatalogItem(ctx, item.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return RecipeView{}, fmt.Errorf("%w: kitchen recipe not found", domain.ErrNotFound)
		}
		return RecipeView{}, err
	}
	lines, err := s.repo.ListRecipeLines(ctx, recipe.ID)
	if err != nil {
		return RecipeView{}, err
	}
	ingredients := make([]RecipeIngredientView, 0, len(lines))
	for _, line := range lines {
		ingredient, err := s.repo.GetCatalogItem(ctx, line.CatalogItemID)
		if err != nil {
			return RecipeView{}, err
		}
		ingredients = append(ingredients, RecipeIngredientView{
			LineID:          line.ID,
			CatalogItemID:   line.CatalogItemID,
			CatalogItemName: ingredient.Name,
			Quantity:        fmt.Sprintf("%d", line.Quantity),
			UnitCode:        line.Unit,
			LossPercent:     fmt.Sprintf("%d", line.LossPercent),
			CloudVersion:    line.CloudVersion,
			CloudUpdatedAt:  stringValue(line.CloudUpdatedAt),
		})
	}
	proposals, err := s.repo.ListKitchenProposals(ctx, kitchendomain.ProposalListQuery{
		RestaurantID:       operator.Employee.RestaurantID,
		Kind:               kitchendomain.ProposalKindRecipe,
		OwnerCatalogItemID: item.ID,
		RecipeVersionID:    recipe.ID,
		Limit:              100,
		IncludeTerminal:    true,
	})
	if err != nil {
		return RecipeView{}, err
	}
	return RecipeView{CatalogItem: *item, RecipeVersion: *recipe, Ingredients: ingredients, Proposals: proposals}, nil
}

func (s *Service) CreateCatalogSuggestion(ctx context.Context, cmd CreateCatalogSuggestionCommand) (*kitchendomain.Proposal, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	var out *kitchendomain.Proposal
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenCatalogSuggest))
		if err != nil {
			return err
		}
		if replayed, ok, err := s.replayedProposalCommand(ctx, cmd.CommandID, "CatalogItemChangeSuggested"); err != nil || ok {
			out = replayed
			return err
		}
		if err := validateCatalogSuggestion(cmd); err != nil {
			return err
		}
		if strings.TrimSpace(cmd.CatalogItemID) != "" {
			if _, err := s.repo.GetCatalogItem(ctx, strings.TrimSpace(cmd.CatalogItemID)); err != nil {
				return err
			}
		}
		suggestionID := trimOrNewID(cmd.SuggestionID, s.ids)
		now := s.clock.Now()
		payload := map[string]any{
			"suggestion_id":            suggestionID,
			"proposal_group_id":        strings.TrimSpace(cmd.ProposalGroupID),
			"restaurant_id":            operator.Employee.RestaurantID,
			"action":                   strings.TrimSpace(cmd.Action),
			"catalog_item_id":          strings.TrimSpace(cmd.CatalogItemID),
			"kind":                     string(cmd.Kind),
			"name":                     strings.TrimSpace(cmd.Name),
			"sku":                      strings.TrimSpace(cmd.SKU),
			"base_unit":                strings.TrimSpace(cmd.BaseUnit),
			"kitchen_type":             strings.TrimSpace(cmd.KitchenType),
			"accounting_category":      strings.TrimSpace(cmd.AccountingCategory),
			"reason":                   strings.TrimSpace(cmd.Reason),
			"suggested_by_employee_id": operator.Employee.ID,
			"suggested_at":             now,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		proposal := &kitchendomain.Proposal{
			ID:                  suggestionID,
			RestaurantID:        operator.Employee.RestaurantID,
			ProposalGroupID:     strings.TrimSpace(cmd.ProposalGroupID),
			Kind:                kitchendomain.ProposalKindCatalog,
			Status:              kitchendomain.ProposalPendingSync,
			Action:              strings.TrimSpace(cmd.Action),
			OwnerCatalogItemID:  strings.TrimSpace(cmd.CatalogItemID),
			Payload:             body,
			OutboxCommandID:     strings.TrimSpace(cmd.CommandID),
			OutboxEventType:     "CatalogItemChangeSuggested",
			CreatedByEmployeeID: operator.Employee.ID,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		if err := s.repo.CreateKitchenProposal(ctx, proposal); err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, operator.Employee.RestaurantID, "", "KitchenProposal", proposal.ID, "CatalogItemChangeSuggested", payload); err != nil {
			return err
		}
		out = proposal
		return nil
	})
	return out, err
}

func (s *Service) CreateRecipeSuggestion(ctx context.Context, cmd CreateRecipeSuggestionCommand) (*kitchendomain.Proposal, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	var out *kitchendomain.Proposal
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenRecipeSuggest))
		if err != nil {
			return err
		}
		if replayed, ok, err := s.replayedProposalCommand(ctx, cmd.CommandID, "RecipeChangeSuggested"); err != nil || ok {
			out = replayed
			return err
		}
		recipeVersionID, ownerCatalogItemID, err := s.validateRecipeSuggestion(ctx, cmd)
		if err != nil {
			return err
		}
		suggestionID := trimOrNewID(cmd.SuggestionID, s.ids)
		now := s.clock.Now()
		payload := map[string]any{
			"suggestion_id":               suggestionID,
			"proposal_group_id":           strings.TrimSpace(cmd.ProposalGroupID),
			"restaurant_id":               operator.Employee.RestaurantID,
			"recipe_version_id":           recipeVersionID,
			"owner_catalog_item_id":       ownerCatalogItemID,
			"owner_catalog_suggestion_id": strings.TrimSpace(cmd.OwnerCatalogSuggestionID),
			"action":                      strings.TrimSpace(cmd.Action),
			"prep_time_delta_minutes":     cmd.PrepTimeDeltaMinutes,
			"changes":                     normalizedRecipeSuggestionChanges(cmd.Changes),
			"reason":                      strings.TrimSpace(cmd.Reason),
			"suggested_by_employee_id":    operator.Employee.ID,
			"suggested_at":                now,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		proposal := &kitchendomain.Proposal{
			ID:                       suggestionID,
			RestaurantID:             operator.Employee.RestaurantID,
			ProposalGroupID:          strings.TrimSpace(cmd.ProposalGroupID),
			Kind:                     kitchendomain.ProposalKindRecipe,
			Status:                   kitchendomain.ProposalPendingSync,
			Action:                   strings.TrimSpace(cmd.Action),
			OwnerCatalogItemID:       ownerCatalogItemID,
			OwnerCatalogSuggestionID: strings.TrimSpace(cmd.OwnerCatalogSuggestionID),
			RecipeVersionID:          recipeVersionID,
			Payload:                  body,
			OutboxCommandID:          strings.TrimSpace(cmd.CommandID),
			OutboxEventType:          "RecipeChangeSuggested",
			CreatedByEmployeeID:      operator.Employee.ID,
			CreatedAt:                now,
			UpdatedAt:                now,
		}
		if err := s.repo.CreateKitchenProposal(ctx, proposal); err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, operator.Employee.RestaurantID, "", "KitchenProposal", proposal.ID, "RecipeChangeSuggested", payload); err != nil {
			return err
		}
		out = proposal
		return nil
	})
	return out, err
}

func (s *Service) ListProposals(ctx context.Context, cmd ListProposalsCommand) ([]kitchendomain.Proposal, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenView))
	if err != nil {
		return nil, err
	}
	if cmd.Kind != "" && !validProposalKind(cmd.Kind) {
		return nil, fmt.Errorf("%w: unsupported kitchen proposal kind", domain.ErrInvalid)
	}
	if cmd.Status != "" && !validProposalStatus(cmd.Status) {
		return nil, fmt.Errorf("%w: unsupported kitchen proposal status", domain.ErrInvalid)
	}
	return s.repo.ListKitchenProposals(ctx, kitchendomain.ProposalListQuery{
		RestaurantID:    operator.Employee.RestaurantID,
		Kind:            cmd.Kind,
		Status:          cmd.Status,
		Limit:           cmd.Limit,
		Offset:          cmd.Offset,
		IncludeTerminal: cmd.Status != "",
	})
}

func (s *Service) ChangeTicketStatus(ctx context.Context, cmd ChangeTicketStatusCommand) (*kitchendomain.Ticket, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.TicketID) == "" || strings.TrimSpace(cmd.Action) == "" {
		return nil, fmt.Errorf("%w: ticket_id and action are required", domain.ErrInvalid)
	}
	var ticket *kitchendomain.Ticket
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenStatusChange))
		if err != nil {
			return err
		}
		ticket, err = s.repo.GetKitchenTicket(ctx, cmd.TicketID)
		if err != nil {
			return err
		}
		if ticket.RestaurantID != operator.Employee.RestaurantID {
			return fmt.Errorf("%w: kitchen ticket belongs to another restaurant", domain.ErrForbidden)
		}
		if repeated, err := s.repo.GetKitchenTicketEventByCommandID(ctx, strings.TrimSpace(cmd.CommandID)); err == nil {
			if repeated.TicketID != ticket.ID || !actionMatchesStatus(cmd.Action, repeated.ToStatus) {
				return fmt.Errorf("%w: %s", domain.ErrDuplicateCommand, strings.TrimSpace(cmd.CommandID))
			}
			return nil
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		next, err := nextStatus(ticket.Status, cmd.Action)
		if err != nil {
			return err
		}
		now := s.clock.Now()
		var serveSequence int
		var supersedesServedEventID *string
		if next == kitchendomain.TicketServed {
			servedCount, err := s.repo.CountKitchenServedEvents(ctx, ticket.ID)
			if err != nil {
				return err
			}
			serveSequence = servedCount + 1
			latestServed, err := s.repo.GetLatestKitchenServedEvent(ctx, ticket.ID)
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				return err
			}
			if latestServed != nil {
				id := latestServed.ID
				supersedesServedEventID = &id
			}
		}
		event := &kitchendomain.TicketEvent{
			ID:              s.ids.NewID(),
			TicketID:        ticket.ID,
			OrderLineID:     ticket.OrderLineID,
			FromStatus:      ticket.Status,
			ToStatus:        next,
			CommandID:       strings.TrimSpace(cmd.CommandID),
			ActorEmployeeID: operator.Employee.ID,
			OccurredAt:      now,
			CreatedAt:       now,
		}
		if err := s.repo.CreateKitchenTicketEvent(ctx, event); err != nil {
			return err
		}
		if err := s.repo.UpdateKitchenTicketStatus(ctx, ticket.ID, next, shared.DBTime(now)); err != nil {
			return err
		}
		ticket.Status = next
		ticket.UpdatedAt = now
		statusPayload := struct {
			TicketID    string                     `json:"ticket_id"`
			OrderID     string                     `json:"order_id"`
			OrderLineID string                     `json:"order_line_id"`
			FromStatus  kitchendomain.TicketStatus `json:"from_status"`
			ToStatus    kitchendomain.TicketStatus `json:"to_status"`
			ChangedAt   any                        `json:"changed_at"`
			StationID   string                     `json:"station_id,omitempty"`
		}{
			TicketID:    ticket.ID,
			OrderID:     ticket.OrderID,
			OrderLineID: ticket.OrderLineID,
			FromStatus:  event.FromStatus,
			ToStatus:    event.ToStatus,
			ChangedAt:   now,
			StationID:   ticket.StationRoutingKey,
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, ticket.RestaurantID, ticket.ShiftID, "KitchenTicket", ticket.ID, "KitchenTicketStatusChanged", statusPayload); err != nil {
			return err
		}
		if next == kitchendomain.TicketServed {
			servedPayload := struct {
				ServedEventID           string  `json:"served_event_id"`
				TicketID                string  `json:"ticket_id"`
				ServeSequence           int     `json:"serve_sequence"`
				SupersedesServedEventID *string `json:"supersedes_served_event_id,omitempty"`
				OrderID                 string  `json:"order_id"`
				OrderLineID             string  `json:"order_line_id"`
				CatalogItemID           string  `json:"catalog_item_id"`
				Quantity                string  `json:"quantity"`
				UnitCode                string  `json:"unit_code"`
				ServedAt                any     `json:"served_at"`
				StationID               string  `json:"station_id,omitempty"`
			}{
				ServedEventID:           event.ID,
				TicketID:                ticket.ID,
				ServeSequence:           serveSequence,
				SupersedesServedEventID: supersedesServedEventID,
				OrderID:                 ticket.OrderID,
				OrderLineID:             ticket.OrderLineID,
				CatalogItemID:           ticket.CatalogItemID,
				Quantity:                fmt.Sprintf("%d.000", ticket.Quantity),
				UnitCode:                ticket.UnitCode,
				ServedAt:                now,
				StationID:               ticket.StationRoutingKey,
			}
			if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, ticket.RestaurantID, ticket.ShiftID, "KitchenTicket", ticket.ID, "ItemServed", servedPayload); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *Service) CaptureStockReceipt(ctx context.Context, cmd CaptureStockReceiptCommand) (StockCommandResult, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return StockCommandResult{}, err
	}
	var out StockCommandResult
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenStockReceipt))
		if err != nil {
			return err
		}
		if replayed, ok, err := s.replayedStockCommand(ctx, cmd.CommandID, "StockReceiptCaptured"); err != nil || ok {
			out = replayed
			return err
		}
		warehouseID, err := s.resolveWarehouseID(ctx, operator.Employee.RestaurantID, cmd.WarehouseID)
		if err != nil {
			return err
		}
		payload, receiptID, err := s.stockReceiptPayload(ctx, operator.Employee.RestaurantID, warehouseID, cmd)
		if err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, operator.Employee.RestaurantID, "", "KitchenStockReceipt", receiptID, "StockReceiptCaptured", payload); err != nil {
			return err
		}
		out = StockCommandResult{ID: receiptID, WarehouseID: warehouseID, EventType: "StockReceiptCaptured"}
		return nil
	})
	return out, err
}

func (s *Service) CaptureInventoryCount(ctx context.Context, cmd CaptureInventoryCountCommand) (StockCommandResult, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return StockCommandResult{}, err
	}
	var out StockCommandResult
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenStockInventoryCount))
		if err != nil {
			return err
		}
		if replayed, ok, err := s.replayedStockCommand(ctx, cmd.CommandID, "InventoryCountCaptured"); err != nil || ok {
			out = replayed
			return err
		}
		warehouseID, err := s.resolveWarehouseID(ctx, operator.Employee.RestaurantID, cmd.WarehouseID)
		if err != nil {
			return err
		}
		payload, countID, err := s.inventoryCountPayload(ctx, operator.Employee.RestaurantID, warehouseID, cmd)
		if err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, operator.Employee.RestaurantID, "", "KitchenInventoryCount", countID, "InventoryCountCaptured", payload); err != nil {
			return err
		}
		out = StockCommandResult{ID: countID, WarehouseID: warehouseID, EventType: "InventoryCountCaptured"}
		return nil
	})
	return out, err
}

func (s *Service) CaptureStockWriteOff(ctx context.Context, cmd CaptureStockWriteOffCommand) (StockCommandResult, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return StockCommandResult{}, err
	}
	var out StockCommandResult
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenStockWriteOff))
		if err != nil {
			return err
		}
		if replayed, ok, err := s.replayedStockCommand(ctx, cmd.CommandID, "StockWriteOffCaptured"); err != nil || ok {
			out = replayed
			return err
		}
		warehouseID, err := s.resolveWarehouseID(ctx, operator.Employee.RestaurantID, cmd.WarehouseID)
		if err != nil {
			return err
		}
		payload, writeOffID, err := s.stockWriteOffPayload(ctx, operator.Employee.RestaurantID, warehouseID, cmd)
		if err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, operator.Employee.RestaurantID, "", "KitchenStockWriteOff", writeOffID, "StockWriteOffCaptured", payload); err != nil {
			return err
		}
		out = StockCommandResult{ID: writeOffID, WarehouseID: warehouseID, EventType: "StockWriteOffCaptured"}
		return nil
	})
	return out, err
}

func (s *Service) CompleteProduction(ctx context.Context, cmd CompleteProductionCommand) (StockCommandResult, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return StockCommandResult{}, err
	}
	var out StockCommandResult
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenProductionComplete))
		if err != nil {
			return err
		}
		if replayed, ok, err := s.replayedStockCommand(ctx, cmd.CommandID, "ProductionCompleted"); err != nil || ok {
			out = replayed
			return err
		}
		warehouseID, err := s.resolveWarehouseID(ctx, operator.Employee.RestaurantID, cmd.WarehouseID)
		if err != nil {
			return err
		}
		payload, productionID, err := s.productionPayload(ctx, operator.Employee.RestaurantID, warehouseID, cmd)
		if err != nil {
			return err
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, operator.Employee.RestaurantID, "", "KitchenProduction", productionID, "ProductionCompleted", payload); err != nil {
			return err
		}
		out = StockCommandResult{ID: productionID, WarehouseID: warehouseID, EventType: "ProductionCompleted"}
		return nil
	})
	return out, err
}

func (s *Service) UpdateStopList(ctx context.Context, cmd UpdateStopListCommand) (StockCommandResult, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.ValidateWriteMeta(cmd.CommandMeta); err != nil {
		return StockCommandResult{}, err
	}
	var out StockCommandResult
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenStopListUpdate))
		if err != nil {
			return err
		}
		if replayed, ok, err := s.replayedStockCommand(ctx, cmd.CommandID, "StopListUpdated"); err != nil || ok {
			out = replayed
			return err
		}
		if strings.TrimSpace(cmd.CatalogItemID) == "" {
			return fmt.Errorf("%w: catalog_item_id is required", domain.ErrInvalid)
		}
		if cmd.AvailableQuantity != nil && *cmd.AvailableQuantity < 0 {
			return fmt.Errorf("%w: available_quantity must be non-negative", domain.ErrInvalid)
		}
		if _, err := s.repo.GetCatalogItem(ctx, strings.TrimSpace(cmd.CatalogItemID)); err != nil {
			return err
		}
		warehouseID, err := s.resolveWarehouseID(ctx, operator.Employee.RestaurantID, cmd.WarehouseID)
		if err != nil {
			return err
		}
		now := s.clock.Now()
		stopListID := trimOrNewID(cmd.StopListID, s.ids)
		reason := strings.TrimSpace(cmd.Reason)
		entry := &domain.StopListEntry{
			ID:                stopListID,
			RestaurantID:      operator.Employee.RestaurantID,
			CatalogItemID:     strings.TrimSpace(cmd.CatalogItemID),
			AvailableQuantity: cmd.AvailableQuantity,
			Source:            "edge_overlay_requires_manager_review",
			Reason:            stringPtr(reason),
			Active:            cmd.Active,
			UpdatedAt:         now,
		}
		if err := s.repo.UpsertLocalStopListEntry(ctx, entry); err != nil {
			return err
		}
		payload := map[string]any{
			"stop_list_id":           stopListID,
			"restaurant_id":          operator.Employee.RestaurantID,
			"warehouse_id":           warehouseID,
			"catalog_item_id":        strings.TrimSpace(cmd.CatalogItemID),
			"active":                 cmd.Active,
			"conflict_policy":        "edge_overlay_requires_manager_review",
			"source":                 "edge",
			"reason":                 reason,
			"updated_at":             now,
			"updated_by_employee_id": operator.Employee.ID,
		}
		if cmd.AvailableQuantity != nil {
			payload["available_quantity"] = fmt.Sprintf("%.3f", *cmd.AvailableQuantity)
		}
		if err := shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, operator.Employee.RestaurantID, "", "StopList", stopListID, "StopListUpdated", payload); err != nil {
			return err
		}
		out = StockCommandResult{ID: stopListID, WarehouseID: warehouseID, EventType: "StopListUpdated"}
		return nil
	})
	return out, err
}

func (s *Service) ListStopListState(ctx context.Context, cmd ListStopListStateCommand) ([]StopListState, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	operator, err := shared.EnsureOperatorSession(ctx, s.repo, cmd.CommandMeta, string(shared.PermissionKitchenStopListView))
	if err != nil {
		return nil, err
	}
	entries, err := s.repo.ListStopListEntries(ctx, operator.Employee.RestaurantID)
	if err != nil {
		return nil, err
	}
	outboxRows, err := s.repo.ListOutboxByCommandType(ctx, "StopListUpdated", 500)
	if err != nil {
		return nil, err
	}
	latestByStopListID := map[string]domain.OutboxMessage{}
	for _, msg := range outboxRows {
		if msg.AggregateType != "StopList" || msg.CommandType != "StopListUpdated" {
			continue
		}
		if current, ok := latestByStopListID[msg.AggregateID]; !ok || msg.SequenceNo > current.SequenceNo {
			latestByStopListID[msg.AggregateID] = msg
		}
	}
	out := make([]StopListState, 0, len(entries))
	for _, entry := range entries {
		state := StopListState{
			ID:                entry.ID,
			CatalogItemID:     entry.CatalogItemID,
			AvailableQuantity: entry.AvailableQuantity,
			Source:            entry.Source,
			Reason:            entry.Reason,
			Active:            entry.Active,
			UpdatedAt:         entry.UpdatedAt,
			SyncState:         "cloud_authority",
		}
		if strings.HasPrefix(entry.Source, "edge_") {
			state.SyncState = "unknown"
		}
		if msg, ok := latestByStopListID[entry.ID]; ok {
			state.OutboxCommandID = msg.CommandID
			state.OutboxStatus = string(msg.Status)
			state.OutboxSequenceNo = msg.SequenceNo
			state.OutboxAttempts = msg.Attempts
			state.SentAt = msg.SentAt
			state.NextRetryAt = msg.NextRetryAt
			state.SyncState = stopListSyncState(msg.Status)
		}
		out = append(out, state)
	}
	return out, nil
}

func (s *Service) replayedStockCommand(ctx context.Context, commandID, eventType string) (StockCommandResult, bool, error) {
	commandID = strings.TrimSpace(commandID)
	if commandID == "" {
		return StockCommandResult{}, false, fmt.Errorf("%w: command_id is required", domain.ErrInvalid)
	}
	msg, err := s.repo.GetOutboxByCommandID(ctx, commandID)
	if errors.Is(err, domain.ErrNotFound) {
		return StockCommandResult{}, false, nil
	}
	if err != nil {
		return StockCommandResult{}, false, err
	}
	if msg.CommandType != eventType {
		return StockCommandResult{}, false, fmt.Errorf("%w: %s", domain.ErrDuplicateCommand, commandID)
	}
	return replayedStockCommandResult(msg), true, nil
}

func stopListSyncState(status domain.OutboxStatus) string {
	switch status {
	case domain.OutboxSent:
		return "acknowledged"
	case domain.OutboxFailed, domain.OutboxSuspended:
		return "problem"
	case domain.OutboxPending, domain.OutboxProcessing:
		return "pending"
	default:
		return "unknown"
	}
}

func replayedStockCommandResult(msg *domain.OutboxMessage) StockCommandResult {
	out := StockCommandResult{
		ID:        strings.TrimSpace(msg.AggregateID),
		EventType: strings.TrimSpace(msg.CommandType),
		Replayed:  true,
	}
	var envelope struct {
		Payload struct {
			Data map[string]any `json:"data"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(msg.PayloadJSON), &envelope); err == nil {
		if warehouseID, ok := envelope.Payload.Data["warehouse_id"].(string); ok {
			out.WarehouseID = strings.TrimSpace(warehouseID)
		}
	}
	return out
}

func (s *Service) replayedProposalCommand(ctx context.Context, commandID, eventType string) (*kitchendomain.Proposal, bool, error) {
	commandID = strings.TrimSpace(commandID)
	if commandID == "" {
		return nil, false, fmt.Errorf("%w: command_id is required", domain.ErrInvalid)
	}
	proposal, err := s.repo.GetKitchenProposalByCommandID(ctx, commandID)
	if err == nil {
		if proposal.OutboxEventType != eventType {
			return nil, false, fmt.Errorf("%w: %s", domain.ErrDuplicateCommand, commandID)
		}
		proposal.Replayed = true
		return proposal, true, nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, false, err
	}
	if err := shared.EnsureCommandNotProcessed(ctx, s.repo, commandID); err != nil {
		return nil, false, err
	}
	return nil, false, nil
}

func (s *Service) resolveWarehouseID(ctx context.Context, restaurantID, requestedWarehouseID string) (string, error) {
	requestedWarehouseID = strings.TrimSpace(requestedWarehouseID)
	if requestedWarehouseID != "" {
		warehouse, err := s.repo.GetWarehouseReference(ctx, restaurantID, requestedWarehouseID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return "", fmt.Errorf("%w: kitchen warehouse required", domain.ErrInvalid)
			}
			return "", err
		}
		return warehouse.ID, nil
	}
	warehouse, err := s.repo.GetDefaultWarehouseReference(ctx, restaurantID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", fmt.Errorf("%w: kitchen warehouse required", domain.ErrInvalid)
		}
		return "", err
	}
	return warehouse.ID, nil
}

func (s *Service) stockReceiptPayload(ctx context.Context, restaurantID, warehouseID string, cmd CaptureStockReceiptCommand) (any, string, error) {
	receiptID := trimOrNewID(cmd.ReceiptID, s.ids)
	if strings.TrimSpace(cmd.SupplierCounterpartyID) == "" && strings.TrimSpace(cmd.SupplierNameSnapshot) == "" {
		return nil, "", fmt.Errorf("%w: receipt supplier is required", domain.ErrInvalid)
	}
	if !validBusinessDate(cmd.DocumentDate) || !validBusinessDate(cmd.BusinessDateLocal) || cmd.ReceivedAt.IsZero() {
		return nil, "", fmt.Errorf("%w: receipt document_date, business_date_local and received_at are required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.Currency) == "" {
		return nil, "", fmt.Errorf("%w: receipt currency is required", domain.ErrInvalid)
	}
	if len(cmd.Items) == 0 {
		return nil, "", fmt.Errorf("%w: receipt items are required", domain.ErrInvalid)
	}
	items := make([]any, 0, len(cmd.Items))
	for _, line := range cmd.Items {
		lineID := trimOrNewID(line.LineID, s.ids)
		catalogItemID := strings.TrimSpace(line.CatalogItemID)
		catalogSuggestionID := strings.TrimSpace(line.CatalogSuggestionID)
		if catalogItemID == "" {
			return nil, "", fmt.Errorf("%w: kitchen receipt line item required", domain.ErrInvalid)
		}
		if err := s.ensureStockCatalogItem(ctx, catalogItemID); err != nil {
			return nil, "", err
		}
		if !positiveDecimal(line.Quantity) || strings.TrimSpace(line.UnitCode) == "" {
			return nil, "", fmt.Errorf("%w: receipt line quantity and unit_code are required", domain.ErrInvalid)
		}
		if line.UnitCostMinor < 0 || line.LineTotalMinor <= 0 {
			return nil, "", fmt.Errorf("%w: kitchen receipt line total required", domain.ErrInvalid)
		}
		lineCurrency := strings.TrimSpace(line.Currency)
		if lineCurrency == "" {
			lineCurrency = strings.TrimSpace(cmd.Currency)
		}
		items = append(items, map[string]any{
			"line_id":               lineID,
			"catalog_item_id":       catalogItemID,
			"catalog_suggestion_id": catalogSuggestionID,
			"name_snapshot":         strings.TrimSpace(line.NameSnapshot),
			"quantity":              strings.TrimSpace(line.Quantity),
			"unit_code":             strings.TrimSpace(line.UnitCode),
			"unit_cost_minor":       line.UnitCostMinor,
			"line_total_minor":      line.LineTotalMinor,
			"currency":              lineCurrency,
		})
	}
	return map[string]any{
		"receipt_id":               receiptID,
		"restaurant_id":            restaurantID,
		"warehouse_id":             warehouseID,
		"supplier_id":              strings.TrimSpace(cmd.SupplierCounterpartyID),
		"supplier_counterparty_id": strings.TrimSpace(cmd.SupplierCounterpartyID),
		"supplier_name_snapshot":   strings.TrimSpace(cmd.SupplierNameSnapshot),
		"document_number":          strings.TrimSpace(cmd.DocumentNumber),
		"document_date":            strings.TrimSpace(cmd.DocumentDate),
		"received_at":              cmd.ReceivedAt,
		"business_date_local":      strings.TrimSpace(cmd.BusinessDateLocal),
		"currency":                 strings.TrimSpace(cmd.Currency),
		"items":                    items,
	}, receiptID, nil
}

func (s *Service) inventoryCountPayload(ctx context.Context, restaurantID, warehouseID string, cmd CaptureInventoryCountCommand) (any, string, error) {
	countID := trimOrNewID(cmd.CountID, s.ids)
	if !validBusinessDate(cmd.BusinessDateLocal) || cmd.CountedAt.IsZero() {
		return nil, "", fmt.Errorf("%w: inventory count business_date_local and counted_at are required", domain.ErrInvalid)
	}
	if len(cmd.Items) == 0 {
		return nil, "", fmt.Errorf("%w: kitchen inventory count empty", domain.ErrInvalid)
	}
	items := make([]any, 0, len(cmd.Items))
	for _, line := range cmd.Items {
		if strings.TrimSpace(line.CatalogItemID) == "" || !positiveDecimal(line.CountedQuantity) || strings.TrimSpace(line.UnitCode) == "" {
			return nil, "", fmt.Errorf("%w: inventory count line catalog_item_id, counted_quantity and unit_code are required", domain.ErrInvalid)
		}
		if err := s.ensureStockCatalogItem(ctx, line.CatalogItemID); err != nil {
			return nil, "", err
		}
		items = append(items, map[string]any{
			"line_id":          trimOrNewID(line.LineID, s.ids),
			"catalog_item_id":  strings.TrimSpace(line.CatalogItemID),
			"counted_quantity": strings.TrimSpace(line.CountedQuantity),
			"unit_code":        strings.TrimSpace(line.UnitCode),
		})
	}
	return map[string]any{
		"count_id":            countID,
		"restaurant_id":       restaurantID,
		"warehouse_id":        warehouseID,
		"counted_at":          cmd.CountedAt,
		"business_date_local": strings.TrimSpace(cmd.BusinessDateLocal),
		"items":               items,
	}, countID, nil
}

func (s *Service) stockWriteOffPayload(ctx context.Context, restaurantID, warehouseID string, cmd CaptureStockWriteOffCommand) (any, string, error) {
	writeOffID := trimOrNewID(cmd.WriteOffID, s.ids)
	if !validBusinessDate(cmd.BusinessDateLocal) || cmd.WrittenOffAt.IsZero() {
		return nil, "", fmt.Errorf("%w: write-off business_date_local and written_off_at are required", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.Reason) == "" && strings.TrimSpace(cmd.ReasonCode) == "" {
		return nil, "", fmt.Errorf("%w: kitchen write-off reason required", domain.ErrInvalid)
	}
	if len(cmd.Items) == 0 {
		return nil, "", fmt.Errorf("%w: write-off items are required", domain.ErrInvalid)
	}
	items := make([]any, 0, len(cmd.Items))
	for _, line := range cmd.Items {
		if strings.TrimSpace(line.CatalogItemID) == "" || !positiveDecimal(line.Quantity) || strings.TrimSpace(line.UnitCode) == "" {
			return nil, "", fmt.Errorf("%w: write-off line catalog_item_id, quantity and unit_code are required", domain.ErrInvalid)
		}
		if err := s.ensureStockCatalogItem(ctx, line.CatalogItemID); err != nil {
			return nil, "", err
		}
		items = append(items, map[string]any{
			"line_id":         trimOrNewID(line.LineID, s.ids),
			"catalog_item_id": strings.TrimSpace(line.CatalogItemID),
			"quantity":        strings.TrimSpace(line.Quantity),
			"unit_code":       strings.TrimSpace(line.UnitCode),
		})
	}
	return map[string]any{
		"write_off_id":        writeOffID,
		"restaurant_id":       restaurantID,
		"warehouse_id":        warehouseID,
		"written_off_at":      cmd.WrittenOffAt,
		"business_date_local": strings.TrimSpace(cmd.BusinessDateLocal),
		"reason_code":         strings.TrimSpace(cmd.ReasonCode),
		"reason":              strings.TrimSpace(cmd.Reason),
		"items":               items,
	}, writeOffID, nil
}

func (s *Service) productionPayload(ctx context.Context, restaurantID, warehouseID string, cmd CompleteProductionCommand) (any, string, error) {
	productionID := trimOrNewID(cmd.ProductionID, s.ids)
	if strings.TrimSpace(cmd.SemiFinishedCatalogItemID) == "" || !positiveDecimal(cmd.Quantity) || strings.TrimSpace(cmd.UnitCode) == "" {
		return nil, "", fmt.Errorf("%w: production semi_finished_catalog_item_id, quantity and unit_code are required", domain.ErrInvalid)
	}
	if !validBusinessDate(cmd.BusinessDateLocal) || cmd.CompletedAt.IsZero() {
		return nil, "", fmt.Errorf("%w: production business_date_local and completed_at are required", domain.ErrInvalid)
	}
	item, err := s.repo.GetCatalogItem(ctx, strings.TrimSpace(cmd.SemiFinishedCatalogItemID))
	if err != nil {
		return nil, "", err
	}
	if item.Type != domain.CatalogItemSemiFinished || !item.Active {
		return nil, "", fmt.Errorf("%w: production item must be active semi_finished catalog item", domain.ErrInvalid)
	}
	if _, err := s.repo.GetActiveRecipeVersionByCatalogItem(ctx, item.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, "", fmt.Errorf("%w: kitchen production recipe required", domain.ErrInvalid)
		}
		return nil, "", err
	}
	return map[string]any{
		"production_id":                 productionID,
		"restaurant_id":                 restaurantID,
		"warehouse_id":                  warehouseID,
		"semi_finished_catalog_item_id": item.ID,
		"quantity":                      strings.TrimSpace(cmd.Quantity),
		"unit_code":                     strings.TrimSpace(cmd.UnitCode),
		"completed_at":                  cmd.CompletedAt,
		"business_date_local":           strings.TrimSpace(cmd.BusinessDateLocal),
	}, productionID, nil
}

func (s *Service) ensureStockCatalogItem(ctx context.Context, catalogItemID string) error {
	item, err := s.repo.GetCatalogItem(ctx, strings.TrimSpace(catalogItemID))
	if err != nil {
		return err
	}
	if !item.Active || item.Type == domain.CatalogItemService {
		return fmt.Errorf("%w: stock event catalog item must be active stock-capable item", domain.ErrInvalid)
	}
	return nil
}

func validateCatalogSuggestion(cmd CreateCatalogSuggestionCommand) error {
	action := strings.TrimSpace(cmd.Action)
	if !validCatalogSuggestionAction(action) {
		return fmt.Errorf("%w: unsupported catalog suggestion action", domain.ErrInvalid)
	}
	if !validCatalogSuggestionKind(cmd.Kind) {
		return fmt.Errorf("%w: unsupported catalog suggestion kind", domain.ErrInvalid)
	}
	if action == "create" && (strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.BaseUnit) == "") {
		return fmt.Errorf("%w: catalog suggestion name and base_unit are required", domain.ErrInvalid)
	}
	if action != "create" && strings.TrimSpace(cmd.CatalogItemID) == "" {
		return fmt.Errorf("%w: catalog_item_id is required for catalog suggestion update/archive", domain.ErrInvalid)
	}
	if strings.TrimSpace(cmd.Reason) == "" {
		return fmt.Errorf("%w: catalog suggestion reason is required", domain.ErrInvalid)
	}
	return nil
}

func (s *Service) validateRecipeSuggestion(ctx context.Context, cmd CreateRecipeSuggestionCommand) (string, string, error) {
	action := strings.TrimSpace(cmd.Action)
	if !validRecipeSuggestionAction(action) {
		return "", "", fmt.Errorf("%w: unsupported recipe suggestion action", domain.ErrInvalid)
	}
	delta := cmd.PrepTimeDeltaMinutes
	if delta < 0 {
		delta = -delta
	}
	if delta > s.options.RecipeSuggestionMaxPrepTimeDeltaMinutes {
		return "", "", fmt.Errorf("%w: kitchen recipe suggestion limit exceeded", domain.ErrInvalid)
	}
	ownerCatalogItemID := strings.TrimSpace(cmd.OwnerCatalogItemID)
	recipeVersionID := strings.TrimSpace(cmd.RecipeVersionID)
	if action == "create_recipe" {
		if ownerCatalogItemID == "" && strings.TrimSpace(cmd.OwnerCatalogSuggestionID) == "" {
			return "", "", fmt.Errorf("%w: owner_catalog_item_id or owner_catalog_suggestion_id is required", domain.ErrInvalid)
		}
		if ownerCatalogItemID != "" {
			item, err := s.repo.GetCatalogItem(ctx, ownerCatalogItemID)
			if err != nil {
				return "", "", err
			}
			if !item.Active || (item.Type != domain.CatalogItemDish && item.Type != domain.CatalogItemSemiFinished) {
				return "", "", fmt.Errorf("%w: recipe owner must be active dish or semi_finished item", domain.ErrInvalid)
			}
		}
	} else {
		if recipeVersionID == "" && ownerCatalogItemID == "" {
			return "", "", fmt.Errorf("%w: recipe_version_id or owner_catalog_item_id is required", domain.ErrInvalid)
		}
	}
	if recipeVersionID != "" {
		recipe, err := s.repo.GetRecipeVersion(ctx, recipeVersionID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return "", "", fmt.Errorf("%w: kitchen recipe not found", domain.ErrNotFound)
			}
			return "", "", err
		}
		if ownerCatalogItemID != "" && recipe.DishCatalogItemID != ownerCatalogItemID {
			return "", "", fmt.Errorf("%w: recipe_version_id does not match owner_catalog_item_id", domain.ErrInvalid)
		}
		ownerCatalogItemID = recipe.DishCatalogItemID
	}
	if recipeVersionID == "" && ownerCatalogItemID != "" && action != "create_recipe" {
		recipe, err := s.repo.GetActiveRecipeVersionByCatalogItem(ctx, ownerCatalogItemID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return "", "", fmt.Errorf("%w: kitchen recipe not found", domain.ErrNotFound)
			}
			return "", "", err
		}
		recipeVersionID = recipe.ID
	}
	if action != "change_prep_time" && len(cmd.Changes) == 0 {
		return "", "", fmt.Errorf("%w: recipe suggestion changes are required", domain.ErrInvalid)
	}
	for _, change := range cmd.Changes {
		if !validRecipeSuggestionAction(strings.TrimSpace(change.Action)) {
			return "", "", fmt.Errorf("%w: unsupported recipe suggestion change action", domain.ErrInvalid)
		}
	}
	if strings.TrimSpace(cmd.Reason) == "" {
		return "", "", fmt.Errorf("%w: recipe suggestion reason is required", domain.ErrInvalid)
	}
	return recipeVersionID, ownerCatalogItemID, nil
}

func normalizedRecipeSuggestionChanges(changes []RecipeSuggestionChangeCommand) []map[string]any {
	out := make([]map[string]any, 0, len(changes))
	for _, change := range changes {
		out = append(out, map[string]any{
			"line_id":              strings.TrimSpace(change.LineID),
			"action":               strings.TrimSpace(change.Action),
			"from_catalog_item_id": strings.TrimSpace(change.FromCatalogItemID),
			"to_catalog_item_id":   strings.TrimSpace(change.ToCatalogItemID),
			"quantity":             strings.TrimSpace(change.Quantity),
			"unit_code":            strings.TrimSpace(change.UnitCode),
			"loss_percent":         strings.TrimSpace(change.LossPercent),
		})
	}
	return out
}

func trimOrNewID(id string, ids idgen.Generator) string {
	if id = strings.TrimSpace(id); id != "" {
		return id
	}
	return ids.NewID()
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func stringPtr(value string) *string {
	if value = strings.TrimSpace(value); value != "" {
		return &value
	}
	return nil
}

func validBusinessDate(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) != len("2006-01-02") {
		return false
	}
	_, err := time.Parse("2006-01-02", v)
	return err == nil
}

func positiveDecimal(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	seenDigit := false
	seenPositive := false
	seenDot := false
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9':
			seenDigit = true
			if r != '0' {
				seenPositive = true
			}
		case r == '.' && !seenDot:
			seenDot = true
		default:
			return false
		}
	}
	return seenDigit && seenPositive
}

func validCatalogSuggestionAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "create", "update", "archive":
		return true
	default:
		return false
	}
}

func validCatalogSuggestionKind(kind domain.CatalogItemType) bool {
	switch kind {
	case domain.CatalogItemDish, domain.CatalogItemGood, domain.CatalogItemSemiFinished, domain.CatalogItemService:
		return true
	default:
		return false
	}
}

func validRecipeSuggestionAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "create_recipe", "update_recipe", "replace_ingredient", "add_ingredient", "remove_ingredient", "change_quantity", "change_loss_percent", "change_prep_time":
		return true
	default:
		return false
	}
}

func validProposalKind(kind kitchendomain.ProposalKind) bool {
	switch kind {
	case kitchendomain.ProposalKindCatalog, kitchendomain.ProposalKindRecipe:
		return true
	default:
		return false
	}
}

func validProposalStatus(status kitchendomain.ProposalStatus) bool {
	switch status {
	case kitchendomain.ProposalDraft, kitchendomain.ProposalPendingSync, kitchendomain.ProposalSynced, kitchendomain.ProposalApproved, kitchendomain.ProposalRejected, kitchendomain.ProposalChangesRequested, kitchendomain.ProposalFailed:
		return true
	default:
		return false
	}
}

func validStatus(status kitchendomain.TicketStatus) bool {
	switch status {
	case kitchendomain.TicketNew, kitchendomain.TicketAccepted, kitchendomain.TicketInProgress, kitchendomain.TicketHold, kitchendomain.TicketReady, kitchendomain.TicketServed, kitchendomain.TicketRecall, kitchendomain.TicketCancelled:
		return true
	default:
		return false
	}
}

func validOrderStatus(status kitchendomain.OrderStatus) bool {
	switch status {
	case kitchendomain.OrderQueued, kitchendomain.OrderAccepted, kitchendomain.OrderInProgress, kitchendomain.OrderPartiallyReady, kitchendomain.OrderReady, kitchendomain.OrderPartiallyServed, kitchendomain.OrderServed, kitchendomain.OrderCancelled, kitchendomain.OrderMixed:
		return true
	default:
		return false
	}
}

func normalizeLimitOffset(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func buildOrderQueue(rows []kitchendomain.OrderTicket, now time.Time, filter kitchendomain.OrderStatus) []kitchendomain.Order {
	type group struct {
		order          kitchendomain.Order
		earliestTicket time.Time
	}
	groups := make(map[string]*group)
	for _, row := range rows {
		ticket := row.Ticket
		g, ok := groups[ticket.OrderID]
		if !ok {
			g = &group{
				order: kitchendomain.Order{
					OrderID:             ticket.OrderID,
					EdgeOrderID:         row.EdgeOrderID,
					TableName:           ticket.TableName,
					ShiftID:             ticket.ShiftID,
					CreatedAt:           ticket.CreatedAt,
					LastStatusChangedAt: ticket.UpdatedAt,
				},
				earliestTicket: ticket.CreatedAt,
			}
			groups[ticket.OrderID] = g
		}
		if ticket.CreatedAt.Before(g.order.CreatedAt) {
			g.order.CreatedAt = ticket.CreatedAt
		}
		if ticket.UpdatedAt.After(g.order.LastStatusChangedAt) {
			g.order.LastStatusChangedAt = ticket.UpdatedAt
		}
		if ticket.Status != kitchendomain.TicketServed && ticket.Status != kitchendomain.TicketCancelled && ticket.CreatedAt.Before(g.earliestTicket) {
			g.earliestTicket = ticket.CreatedAt
		}
		g.order.Tickets = append(g.order.Tickets, ticket)
	}
	orders := make([]kitchendomain.Order, 0, len(groups))
	sortKeys := make(map[string]time.Time, len(groups))
	for orderID, g := range groups {
		g.order.KitchenOrderStatus = computeOrderStatus(g.order.Tickets)
		g.order.ElapsedSeconds = int64(now.Sub(g.order.CreatedAt).Seconds())
		if g.order.ElapsedSeconds < 0 {
			g.order.ElapsedSeconds = 0
		}
		if filter == "" {
			if g.order.KitchenOrderStatus == kitchendomain.OrderServed || g.order.KitchenOrderStatus == kitchendomain.OrderCancelled {
				continue
			}
		} else if g.order.KitchenOrderStatus != filter {
			continue
		}
		sortKeys[orderID] = g.earliestTicket
		orders = append(orders, g.order)
	}
	sort.SliceStable(orders, func(i, j int) bool {
		left := sortKeys[orders[i].OrderID]
		right := sortKeys[orders[j].OrderID]
		if left.Equal(right) {
			return orders[i].OrderID < orders[j].OrderID
		}
		return left.Before(right)
	})
	return orders
}

func computeOrderStatus(tickets []kitchendomain.Ticket) kitchendomain.OrderStatus {
	if len(tickets) == 0 {
		return kitchendomain.OrderCancelled
	}
	counts := map[kitchendomain.TicketStatus]int{}
	active := 0
	for _, ticket := range tickets {
		counts[ticket.Status]++
		if ticket.Status != kitchendomain.TicketCancelled {
			active++
		}
	}
	if active == 0 {
		return kitchendomain.OrderCancelled
	}
	if counts[kitchendomain.TicketServed] == active {
		return kitchendomain.OrderServed
	}
	if counts[kitchendomain.TicketServed] > 0 {
		return kitchendomain.OrderPartiallyServed
	}
	if counts[kitchendomain.TicketReady] == active {
		return kitchendomain.OrderReady
	}
	if counts[kitchendomain.TicketReady] > 0 {
		return kitchendomain.OrderPartiallyReady
	}
	if counts[kitchendomain.TicketHold] > 0 || counts[kitchendomain.TicketRecall] > 0 {
		return kitchendomain.OrderMixed
	}
	if counts[kitchendomain.TicketInProgress] > 0 {
		return kitchendomain.OrderInProgress
	}
	if counts[kitchendomain.TicketAccepted] > 0 {
		return kitchendomain.OrderAccepted
	}
	if counts[kitchendomain.TicketNew] == active {
		return kitchendomain.OrderQueued
	}
	return kitchendomain.OrderMixed
}

func actionMatchesStatus(action string, status kitchendomain.TicketStatus) bool {
	switch strings.TrimSpace(action) {
	case "accept":
		return status == kitchendomain.TicketAccepted
	case "start":
		return status == kitchendomain.TicketInProgress
	case "hold":
		return status == kitchendomain.TicketHold
	case "ready":
		return status == kitchendomain.TicketReady
	case "serve":
		return status == kitchendomain.TicketServed
	case "recall":
		return status == kitchendomain.TicketRecall
	case "cancel":
		return status == kitchendomain.TicketCancelled
	default:
		return false
	}
}

func nextStatus(current kitchendomain.TicketStatus, action string) (kitchendomain.TicketStatus, error) {
	action = strings.TrimSpace(action)
	transitions := map[kitchendomain.TicketStatus]map[string]kitchendomain.TicketStatus{
		kitchendomain.TicketNew: {
			"accept": kitchendomain.TicketAccepted,
			"cancel": kitchendomain.TicketCancelled,
		},
		kitchendomain.TicketAccepted: {
			"start":  kitchendomain.TicketInProgress,
			"hold":   kitchendomain.TicketHold,
			"cancel": kitchendomain.TicketCancelled,
		},
		kitchendomain.TicketInProgress: {
			"hold":   kitchendomain.TicketHold,
			"ready":  kitchendomain.TicketReady,
			"cancel": kitchendomain.TicketCancelled,
		},
		kitchendomain.TicketHold: {
			"start":  kitchendomain.TicketInProgress,
			"cancel": kitchendomain.TicketCancelled,
		},
		kitchendomain.TicketReady: {
			"serve":  kitchendomain.TicketServed,
			"recall": kitchendomain.TicketRecall,
		},
		kitchendomain.TicketServed: {
			"recall": kitchendomain.TicketRecall,
		},
		kitchendomain.TicketRecall: {
			"start":  kitchendomain.TicketInProgress,
			"cancel": kitchendomain.TicketCancelled,
		},
	}
	if next, ok := transitions[current][action]; ok {
		return next, nil
	}
	return "", fmt.Errorf("%w: kitchen ticket transition %s from %s is not allowed", domain.ErrConflict, action, current)
}
