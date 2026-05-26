package ports

import (
	"context"

	"pos-backend/internal/pos/domain/kitchen"
)

type KitchenRepository interface {
	CreateKitchenTicket(context.Context, *kitchen.Ticket) error
	GetKitchenTicket(context.Context, string) (*kitchen.Ticket, error)
	ListKitchenTickets(context.Context, kitchen.TicketListQuery) ([]kitchen.Ticket, error)
	UpdateKitchenTicketStatus(context.Context, string, kitchen.TicketStatus, string) error
	CreateKitchenTicketEvent(context.Context, *kitchen.TicketEvent) error
}

