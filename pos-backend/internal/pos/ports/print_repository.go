package ports

import (
	"context"
	"time"

	"pos-backend/internal/pos/domain/receipt"
)

// PrintRepository хранит локальную Edge очередь печати и claim/update операции worker-а.
type PrintRepository interface {
	EnqueuePrintJob(context.Context, *receipt.PrintJob) error
	EnqueuePrintJobWithTargets(context.Context, *receipt.PrintJob, []receipt.PrintJobTarget) error
	GetPrintJob(context.Context, string) (*receipt.PrintJob, error)
	ListPrintJobs(context.Context, receipt.PrintJobListQuery) ([]receipt.PrintJob, error)
	CreatePrintRoute(context.Context, receipt.PrintRoute, string, string, time.Time) (*receipt.PrintRoute, error)
	UpdatePrintRoute(context.Context, receipt.PrintRoute, string, string, time.Time) (*receipt.PrintRoute, error)
	DeactivatePrintRoute(context.Context, string, string, string, time.Time) (*receipt.PrintRoute, error)
	ListPrintRoutes(context.Context, string) ([]receipt.PrintRoute, error)
	ListActivePrintRoutes(context.Context, string, receipt.DocumentType, string, *string) ([]receipt.PrintRoute, error)
	ListPrintJobTargets(context.Context, string) ([]receipt.PrintJobTarget, error)
	ListPrintJobTargetsForCheck(context.Context, string) ([]receipt.PrintJobTarget, error)
	ClaimDuePrintJob(context.Context, string, time.Time) (*receipt.PrintJob, error)
	ClaimDuePrintJobTarget(context.Context, string, time.Time) (*receipt.PrintJob, *receipt.PrintJobTarget, error)
	MarkPrintJobSucceeded(context.Context, string, int, time.Time) error
	MarkPrintJobFailedAttempt(context.Context, string, int, receipt.PrintJobStatus, *time.Time, string, time.Time) error
	MarkPrintJobTargetSucceeded(context.Context, string, int, time.Time) error
	MarkPrintJobTargetFailedAttempt(context.Context, string, int, receipt.PrintJobStatus, *time.Time, string, time.Time) error
	ResetPrintJobForRetry(context.Context, string, time.Time) (*receipt.PrintJob, error)
	ResetPrintJobForRetryWithTargets(context.Context, string, []receipt.PrintJobTarget, time.Time) (*receipt.PrintJob, error)
	RetryPrintJobTarget(context.Context, string, string, time.Time) (*receipt.PrintJob, error)
}
