package ports

import (
	"context"

	"pos-backend/internal/pos/domain/ticket"
)

// TicketRepository хранит выпущенные QR-билетные единицы. Запись выполняется внутри
// транзакции CapturePayment; повторная выдача исключается UNIQUE(order_line_id).
type TicketRepository interface {
	CreateTicketUnit(context.Context, *ticket.TicketUnit) error
	GetTicketUnit(context.Context, string) (*ticket.TicketUnit, error)
	GetTicketUnitByOrderLine(context.Context, string) (*ticket.TicketUnit, error)
	ListTicketUnitsByCheck(context.Context, string) ([]ticket.TicketUnit, error)
	// NextTicketCashShiftSequence возвращает следующий порядковый номер билета внутри кассовой смены.
	NextTicketCashShiftSequence(context.Context, string) (int64, error)
}
