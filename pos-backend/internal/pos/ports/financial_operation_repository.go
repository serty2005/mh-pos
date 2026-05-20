package ports

import (
	"context"

	"pos-backend/internal/pos/domain/financial"
)

type FinancialOperationRepository interface {
	CreateFinancialOperation(context.Context, *financial.Operation) error
	CreateFinancialOperationItem(context.Context, *financial.OperationItem) error
	ListFinancialOperations(context.Context, financial.OperationListQuery) ([]financial.Operation, error)
	ListFinancialOperationsByCheck(context.Context, string) ([]financial.Operation, error)
	SumFinancialOperationAmountByCheck(context.Context, string, financial.OperationType) (int64, error)
	SumFinancialOperationAmountByPayment(context.Context, string, financial.OperationType) (int64, error)
	SumFinancialOperationAmountByOrderLine(context.Context, string, financial.OperationType) (int64, error)
	SumFinancialOperationQuantityByOrderLine(context.Context, string, financial.OperationType) (int64, error)
}
