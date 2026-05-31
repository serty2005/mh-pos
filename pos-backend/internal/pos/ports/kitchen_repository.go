package ports

import (
	"context"

	"pos-backend/internal/pos/domain/kitchen"
)

type KitchenRepository interface {
	CreateKitchenTicket(context.Context, *kitchen.Ticket) error
	GetKitchenTicket(context.Context, string) (*kitchen.Ticket, error)
	ListKitchenTickets(context.Context, kitchen.TicketListQuery) ([]kitchen.Ticket, error)
	ListKitchenOrderQueueTickets(context.Context, kitchen.OrderQueueQuery) ([]kitchen.OrderTicket, error)
	UpdateKitchenTicketStatus(context.Context, string, kitchen.TicketStatus, string) error
	UpdateKitchenTicketLineDetails(context.Context, string, *string, *string, string) error
	CreateKitchenTicketEvent(context.Context, *kitchen.TicketEvent) error
	GetKitchenTicketEventByCommandID(context.Context, string) (*kitchen.TicketEvent, error)
	GetLatestKitchenServedEvent(context.Context, string) (*kitchen.TicketEvent, error)
	CountKitchenServedEvents(context.Context, string) (int, error)
	CreateKitchenProposal(context.Context, *kitchen.Proposal) error
	GetKitchenProposalByCommandID(context.Context, string) (*kitchen.Proposal, error)
	ListKitchenProposals(context.Context, kitchen.ProposalListQuery) ([]kitchen.Proposal, error)
	ApplyKitchenProposalFeedback(context.Context, kitchen.ProposalKind, string, kitchen.ProposalStatus, int64, string, string) error
}
