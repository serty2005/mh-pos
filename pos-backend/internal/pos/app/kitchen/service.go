package kitchen

import (
	"context"
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
