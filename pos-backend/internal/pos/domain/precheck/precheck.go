package precheck

import (
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/pos/domain/shared"
)

type PrecheckStatus string

const (
	PrecheckIssued    PrecheckStatus = "issued"
	PrecheckClosed    PrecheckStatus = "closed"
	PrecheckCancelled PrecheckStatus = "cancelled"
)

type Precheck struct {
	ID            string         `json:"id"`
	OrderID       string         `json:"order_id"`
	Status        PrecheckStatus `json:"status"`
	Subtotal      int64          `json:"subtotal"`
	DiscountTotal int64          `json:"discount_total"`
	TaxTotal      int64          `json:"tax_total"`
	Total         int64          `json:"total"`
	CreatedAt     time.Time      `json:"created_at"`
	IssuedAt      time.Time      `json:"issued_at"`
	ClosedAt      *time.Time     `json:"closed_at,omitempty"`
}

func NewIssued(id, orderID string, subtotal, discountTotal, taxTotal int64, now time.Time) (*Precheck, error) {
	id = strings.TrimSpace(id)
	orderID = strings.TrimSpace(orderID)
	if id == "" || orderID == "" {
		return nil, fmt.Errorf("%w: precheck id and order_id are required", shared.ErrInvalid)
	}
	if subtotal < 0 || discountTotal < 0 || taxTotal < 0 {
		return nil, fmt.Errorf("%w: precheck totals must be non-negative", shared.ErrInvalid)
	}
	total := subtotal - discountTotal + taxTotal
	if total < 0 {
		return nil, fmt.Errorf("%w: precheck total cannot be negative", shared.ErrInvalid)
	}
	return &Precheck{
		ID:            id,
		OrderID:       orderID,
		Status:        PrecheckIssued,
		Subtotal:      subtotal,
		DiscountTotal: discountTotal,
		TaxTotal:      taxTotal,
		Total:         total,
		CreatedAt:     now,
		IssuedAt:      now,
	}, nil
}

func (p Precheck) IsActive() bool {
	return p.Status == PrecheckIssued
}

func (p Precheck) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.OrderID) == "" {
		return fmt.Errorf("%w: precheck id and order_id are required", shared.ErrInvalid)
	}
	if p.Status != PrecheckIssued && p.Status != PrecheckClosed && p.Status != PrecheckCancelled {
		return fmt.Errorf("%w: unsupported precheck status", shared.ErrInvalid)
	}
	if p.Subtotal < 0 || p.DiscountTotal < 0 || p.TaxTotal < 0 || p.Total < 0 {
		return fmt.Errorf("%w: precheck totals must be non-negative", shared.ErrInvalid)
	}
	if p.Total != p.Subtotal-p.DiscountTotal+p.TaxTotal {
		return fmt.Errorf("%w: precheck total snapshot is inconsistent", shared.ErrInvalid)
	}
	if p.ClosedAt != nil && p.Status == PrecheckIssued {
		return fmt.Errorf("%w: issued precheck cannot have closed_at", shared.ErrInvalid)
	}
	return nil
}
