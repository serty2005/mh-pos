package ports

import (
	"context"

	"pos-backend/internal/pos/domain/check"
)

type CheckRepository interface {
	CreateCheck(context.Context, *check.Check) error
	GetCheck(context.Context, string) (*check.Check, error)
	GetCheckByOrder(context.Context, string) (*check.Check, error)
	UpdateCheckPaidTotal(context.Context, *check.Check) error
	CreatePayment(context.Context, *check.Payment) error
	CreatePaymentAttempt(context.Context, *check.PaymentAttempt) error
}
