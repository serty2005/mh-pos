package ports

import (
	"context"

	"pos-backend/internal/pos/domain/precheck"
)

type PrecheckRepository interface {
	CreatePrecheck(context.Context, *precheck.Precheck) error
	GetPrecheck(context.Context, string) (*precheck.Precheck, error)
	GetActivePrecheckByOrder(context.Context, string) (*precheck.Precheck, error)
	ListPrechecksByOrder(context.Context, string) ([]precheck.Precheck, error)
	UpdatePrecheckLifecycle(context.Context, *precheck.Precheck) error
	UpdatePrecheckPayment(context.Context, *precheck.Precheck) error
}
