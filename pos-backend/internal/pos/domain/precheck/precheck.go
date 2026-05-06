package precheck

import (
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/pos/domain/shared"
)

type PrecheckStatus string

const (
	PrecheckIssued     PrecheckStatus = "issued"
	PrecheckClosed     PrecheckStatus = "closed"
	PrecheckCancelled  PrecheckStatus = "cancelled"
	PrecheckSuperseded PrecheckStatus = "superseded"
)

type Precheck struct {
	ID                    string         `json:"id"`
	OrderID               string         `json:"order_id"`
	Status                PrecheckStatus `json:"status"`
	Version               int            `json:"version"`
	SupersedesPrecheckID  *string        `json:"supersedes_precheck_id,omitempty"`
	Subtotal              int64          `json:"subtotal"`
	DiscountTotal         int64          `json:"discount_total"`
	TaxTotal              int64          `json:"tax_total"`
	Total                 int64          `json:"total"`
	PaidTotal             int64          `json:"paid_total"`
	CreatedAt             time.Time      `json:"created_at"`
	IssuedAt              time.Time      `json:"issued_at"`
	ClosedAt              *time.Time     `json:"closed_at,omitempty"`
	CancelledByEmployeeID *string        `json:"cancelled_by_employee_id,omitempty"`
	CancellationReason    *string        `json:"cancellation_reason,omitempty"`
}

func NewIssued(id, orderID string, subtotal, discountTotal, taxTotal int64, now time.Time) (*Precheck, error) {
	return NewIssuedVersion(id, orderID, 1, nil, subtotal, discountTotal, taxTotal, now)
}

func NewIssuedVersion(id, orderID string, version int, supersedesPrecheckID *string, subtotal, discountTotal, taxTotal int64, now time.Time) (*Precheck, error) {
	id = strings.TrimSpace(id)
	orderID = strings.TrimSpace(orderID)
	if id == "" || orderID == "" {
		return nil, fmt.Errorf("%w: precheck id and order_id are required", shared.ErrInvalid)
	}
	if version <= 0 {
		return nil, fmt.Errorf("%w: precheck version must be positive", shared.ErrInvalid)
	}
	if subtotal < 0 || discountTotal < 0 || taxTotal < 0 {
		return nil, fmt.Errorf("%w: precheck totals must be non-negative", shared.ErrInvalid)
	}
	total := subtotal - discountTotal + taxTotal
	if total < 0 {
		return nil, fmt.Errorf("%w: precheck total cannot be negative", shared.ErrInvalid)
	}
	var supersedes *string
	if supersedesPrecheckID != nil {
		trimmed := strings.TrimSpace(*supersedesPrecheckID)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: supersedes_precheck_id cannot be empty", shared.ErrInvalid)
		}
		supersedes = &trimmed
	}
	return &Precheck{
		ID:                   id,
		OrderID:              orderID,
		Status:               PrecheckIssued,
		Version:              version,
		SupersedesPrecheckID: supersedes,
		Subtotal:             subtotal,
		DiscountTotal:        discountTotal,
		TaxTotal:             taxTotal,
		Total:                total,
		CreatedAt:            now,
		IssuedAt:             now,
	}, nil
}

func (p Precheck) IsActive() bool {
	return p.Status == PrecheckIssued
}

func (p *Precheck) Cancel(now time.Time, cancelledByEmployeeID, reason string) error {
	if p.Status != PrecheckIssued {
		return fmt.Errorf("%w: only issued precheck can be cancelled", shared.ErrConflict)
	}
	if p.PaidTotal > 0 {
		return fmt.Errorf("%w: cannot cancel paid precheck", shared.ErrConflict)
	}
	p.Status = PrecheckCancelled
	p.ClosedAt = &now
	p.CancelledByEmployeeID = optionalTrimmed(cancelledByEmployeeID)
	p.CancellationReason = optionalTrimmed(reason)
	return nil
}

func (p *Precheck) Supersede(now time.Time) error {
	if p.Status != PrecheckIssued {
		return fmt.Errorf("%w: only issued precheck can be superseded", shared.ErrConflict)
	}
	if p.PaidTotal > 0 {
		return fmt.Errorf("%w: cannot supersede paid precheck", shared.ErrConflict)
	}
	p.Status = PrecheckSuperseded
	p.ClosedAt = &now
	return nil
}

func (p Precheck) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.OrderID) == "" {
		return fmt.Errorf("%w: precheck id and order_id are required", shared.ErrInvalid)
	}
	if p.Status != PrecheckIssued && p.Status != PrecheckClosed && p.Status != PrecheckCancelled && p.Status != PrecheckSuperseded {
		return fmt.Errorf("%w: unsupported precheck status", shared.ErrInvalid)
	}
	if p.Version <= 0 {
		return fmt.Errorf("%w: precheck version must be positive", shared.ErrInvalid)
	}
	if p.Subtotal < 0 || p.DiscountTotal < 0 || p.TaxTotal < 0 || p.Total < 0 || p.PaidTotal < 0 {
		return fmt.Errorf("%w: precheck totals must be non-negative", shared.ErrInvalid)
	}
	if p.Total != p.Subtotal-p.DiscountTotal+p.TaxTotal {
		return fmt.Errorf("%w: precheck total snapshot is inconsistent", shared.ErrInvalid)
	}
	if p.PaidTotal > p.Total {
		return fmt.Errorf("%w: precheck paid_total cannot exceed total", shared.ErrInvalid)
	}
	if p.ClosedAt != nil && p.Status == PrecheckIssued {
		return fmt.Errorf("%w: issued precheck cannot have closed_at", shared.ErrInvalid)
	}
	if p.ClosedAt == nil && (p.Status == PrecheckClosed || p.Status == PrecheckCancelled || p.Status == PrecheckSuperseded) {
		return fmt.Errorf("%w: terminal precheck must have closed_at", shared.ErrInvalid)
	}
	return nil
}

func optionalTrimmed(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
