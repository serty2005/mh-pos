package check

import (
	"encoding/json"
	"time"
)

type CheckStatus string

const (
	CheckOpen     CheckStatus = "open"
	CheckPaid     CheckStatus = "paid"
	CheckRefunded CheckStatus = "refunded"
	CheckVoided   CheckStatus = "voided"
)

type Check struct {
	ID                string          `json:"id"`
	OrderID           string          `json:"order_id"`
	Status            CheckStatus     `json:"status"`
	Subtotal          int64           `json:"subtotal"`
	DiscountTotal     int64           `json:"discount_total"`
	TaxTotal          int64           `json:"tax_total"`
	Total             int64           `json:"total"`
	PaidTotal         int64           `json:"paid_total"`
	BusinessDateLocal string          `json:"business_date_local"`
	ClosedAt          time.Time       `json:"closed_at"`
	Snapshot          json.RawMessage `json:"snapshot,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}
