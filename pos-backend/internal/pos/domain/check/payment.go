package check

import "time"

type PaymentStatus string
type PaymentMethod string

const (
	PaymentCaptured PaymentStatus = "captured"
	PaymentRefunded PaymentStatus = "refunded"
	PaymentFailed   PaymentStatus = "failed"

	PaymentCash  PaymentMethod = "cash"
	PaymentCard  PaymentMethod = "card"
	PaymentOther PaymentMethod = "other"
)

type Payment struct {
	ID        string        `json:"id"`
	CheckID   string        `json:"check_id"`
	Method    PaymentMethod `json:"method"`
	Amount    int64         `json:"amount"`
	Currency  string        `json:"currency"`
	Status    PaymentStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
