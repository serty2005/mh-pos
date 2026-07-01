package ports

import (
	"context"
	"time"

	"pos-backend/internal/pos/domain/check"
)

type CheckRepository interface {
	CreateCheck(context.Context, *check.Check) error
	GetCheck(context.Context, string) (*check.Check, error)
	GetCheckByOrder(context.Context, string) (*check.Check, error)
	MarkCheckPrintConfirmedIfReady(context.Context, string, time.Time) (bool, error)
	UpdateCheckPaidTotal(context.Context, *check.Check) error
	CreatePayment(context.Context, *check.Payment) error
	CreatePaymentAttempt(context.Context, *check.PaymentAttempt) error
	ListPaymentsByPrecheck(context.Context, string) ([]check.Payment, error)
	GetPayment(context.Context, string) (*check.Payment, error)
	UpdatePaymentStatus(context.Context, *check.Payment) error
	NextPaymentAttemptNo(context.Context, string) (int, error)
}
