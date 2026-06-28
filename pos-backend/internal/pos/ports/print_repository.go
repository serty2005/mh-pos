package ports

import (
	"context"
	"time"

	"pos-backend/internal/pos/domain/receipt"
)

// PrintRepository хранит локальную Edge очередь печати и claim/update операции worker-а.
type PrintRepository interface {
	EnqueuePrintJob(context.Context, *receipt.PrintJob) error
	GetPrintJob(context.Context, string) (*receipt.PrintJob, error)
	ListPrintJobs(context.Context, receipt.PrintJobListQuery) ([]receipt.PrintJob, error)
	ClaimDuePrintJob(context.Context, string, time.Time) (*receipt.PrintJob, error)
	MarkPrintJobSucceeded(context.Context, string, int, time.Time) error
	MarkPrintJobFailedAttempt(context.Context, string, int, receipt.PrintJobStatus, *time.Time, string, time.Time) error
	ResetPrintJobForRetry(context.Context, string, time.Time) (*receipt.PrintJob, error)
}
