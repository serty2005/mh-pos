package kitchen

import (
	"context"
	"fmt"
	"strings"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	kitchendomain "pos-backend/internal/pos/domain/kitchen"
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

type ListTicketsCommand struct {
	shared.CommandMeta
	Status kitchendomain.TicketStatus `json:"status,omitempty"`
	Limit  int                        `json:"limit,omitempty"`
	Offset int                        `json:"offset,omitempty"`
}

type ChangeTicketStatusCommand struct {
	shared.CommandMeta
	TicketID string `json:"ticket_id"`
	Action   string `json:"action"`
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
		Limit:        cmd.Limit,
		Offset:       cmd.Offset,
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
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
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
		next, err := nextStatus(ticket.Status, cmd.Action)
		if err != nil {
			return err
		}
		now := s.clock.Now()
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
			TicketID    string                       `json:"ticket_id"`
			OrderID     string                       `json:"order_id"`
			OrderLineID string                       `json:"order_line_id"`
			FromStatus  kitchendomain.TicketStatus   `json:"from_status"`
			ToStatus    kitchendomain.TicketStatus   `json:"to_status"`
			ChangedAt   any                          `json:"changed_at"`
			StationID   string                       `json:"station_id,omitempty"`
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
				ServedEventID string `json:"served_event_id"`
				OrderID       string `json:"order_id"`
				OrderLineID   string `json:"order_line_id"`
				CatalogItemID string `json:"catalog_item_id"`
				Quantity      string `json:"quantity"`
				UnitCode      string `json:"unit_code"`
				ServedAt      any    `json:"served_at"`
				StationID     string `json:"station_id,omitempty"`
			}{
				ServedEventID: event.ID,
				OrderID:       ticket.OrderID,
				OrderLineID:   ticket.OrderLineID,
				CatalogItemID: ticket.CatalogItemID,
				Quantity:      fmt.Sprintf("%d.000", ticket.Quantity),
				UnitCode:      ticket.UnitCode,
				ServedAt:      now,
				StationID:     ticket.StationRoutingKey,
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

func validStatus(status kitchendomain.TicketStatus) bool {
	switch status {
	case kitchendomain.TicketNew, kitchendomain.TicketAccepted, kitchendomain.TicketInProgress, kitchendomain.TicketHold, kitchendomain.TicketReady, kitchendomain.TicketServed, kitchendomain.TicketRecall, kitchendomain.TicketCancelled:
		return true
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

