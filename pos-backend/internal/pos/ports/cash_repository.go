package ports

import (
	"context"

	"pos-backend/internal/pos/domain/cash"
)

type CashRepository interface {
	CreateCashSession(context.Context, *cash.CashSession) error
	UpdateCashSessionClosed(context.Context, *cash.CashSession) error
	GetCashSession(context.Context, string) (*cash.CashSession, error)
	GetOpenCashSessionByDevice(context.Context, string) (*cash.CashSession, error)
	CreateCashDrawerEvent(context.Context, *cash.CashDrawerEvent) error
}
