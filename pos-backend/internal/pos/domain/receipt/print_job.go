package receipt

import (
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/pos/domain/shared"
)

// PrintJobStatus фиксирует lifecycle локальной Edge очереди печати.
type PrintJobStatus string

const (
	PrintJobPending    PrintJobStatus = "pending"
	PrintJobProcessing PrintJobStatus = "processing"
	PrintJobSucceeded  PrintJobStatus = "succeeded"
	PrintJobFailed     PrintJobStatus = "failed"
)

// PrintJob описывает локальную операционную задачу печати без хранения print payload.
// Snapshot документа остается в prechecks/checks/ticket_units, а job хранит только
// routing/status metadata и последнюю безопасную диагностическую ошибку.
type PrintJob struct {
	ID            string           `json:"id"`
	RestaurantID  string           `json:"restaurant_id"`
	DocumentType  DocumentType     `json:"document_type"`
	ScopeID       *string          `json:"scope_id,omitempty"`
	SourceKind    string           `json:"source_kind"`
	SourceID      string           `json:"source_id"`
	Status        PrintJobStatus   `json:"status"`
	Attempts      int              `json:"attempts"`
	MaxAttempts   int              `json:"max_attempts"`
	PrinterClass  string           `json:"printer_class"`
	LastError     *string          `json:"last_error,omitempty"`
	NextAttemptAt *time.Time       `json:"next_attempt_at,omitempty"`
	LockedBy      *string          `json:"locked_by,omitempty"`
	LockedAt      *time.Time       `json:"locked_at,omitempty"`
	PrintedAt     *time.Time       `json:"printed_at,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Targets       []PrintJobTarget `json:"targets,omitempty"`
}

// PrintRoute описывает Edge-local effective route для одного document_type/scope.
type PrintRoute struct {
	ID           string       `json:"id"`
	RestaurantID string       `json:"restaurant_id"`
	DocumentType DocumentType `json:"document_type"`
	ScopeType    string       `json:"scope_type"`
	ScopeID      *string      `json:"scope_id,omitempty"`
	PrinterID    string       `json:"printer_id"`
	IsRequired   bool         `json:"is_required"`
	SortOrder    int          `json:"sort_order"`
	Origin       string       `json:"origin"`
	IsActive     bool         `json:"is_active"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// PrintJobTarget хранит delivery lifecycle одного job на один физический printer.
type PrintJobTarget struct {
	ID            string         `json:"id"`
	PrintJobID    string         `json:"print_job_id"`
	RestaurantID  string         `json:"restaurant_id"`
	PrinterID     string         `json:"printer_id"`
	ScopeType     string         `json:"scope_type"`
	ScopeID       *string        `json:"scope_id,omitempty"`
	Status        PrintJobStatus `json:"status"`
	Attempts      int            `json:"attempts"`
	MaxAttempts   int            `json:"max_attempts"`
	IsRequired    bool           `json:"is_required"`
	LastError     *string        `json:"last_error,omitempty"`
	NextAttemptAt *time.Time     `json:"next_attempt_at,omitempty"`
	LockedBy      *string        `json:"locked_by,omitempty"`
	LockedAt      *time.Time     `json:"locked_at,omitempty"`
	PrintedAt     *time.Time     `json:"printed_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// PrintJobListQuery задает bounded read model запрос очереди печати.
type PrintJobListQuery struct {
	RestaurantID string
	Status       PrintJobStatus
	DocumentType DocumentType
	Limit        int
}

// ValidateForCreate проверяет инварианты новой print job.
func (j PrintJob) ValidateForCreate() error {
	if strings.TrimSpace(j.ID) == "" || strings.TrimSpace(j.RestaurantID) == "" ||
		strings.TrimSpace(j.SourceKind) == "" || strings.TrimSpace(j.SourceID) == "" {
		return fmt.Errorf("%w: print job id, restaurant_id, source_kind and source_id are required", shared.ErrInvalid)
	}
	switch j.DocumentType {
	case DocumentPrecheck, DocumentCheckNonfiscal, DocumentTicket:
	default:
		return fmt.Errorf("%w: print job document_type must be precheck, check_nonfiscal or ticket", shared.ErrInvalid)
	}
	if j.Status == "" {
		return fmt.Errorf("%w: print job status is required", shared.ErrInvalid)
	}
	if j.MaxAttempts <= 0 {
		return fmt.Errorf("%w: print job max_attempts must be positive", shared.ErrInvalid)
	}
	return nil
}
