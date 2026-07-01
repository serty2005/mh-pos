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
	ID                    string             `json:"id"`
	EdgePaymentID         string             `json:"edge_payment_id"`
	RestaurantID          string             `json:"restaurant_id"`
	DeviceID              string             `json:"device_id"`
	ShiftID               string             `json:"shift_id"`
	PrecheckID            string             `json:"precheck_id"`
	Method                PaymentMethod      `json:"method"`
	Amount                int64              `json:"amount"`
	Currency              string             `json:"currency"`
	Status                PaymentStatus      `json:"status"`
	BusinessDateLocal     string             `json:"business_date_local"`
	ProviderName          *string            `json:"provider_name,omitempty"`
	ProviderTransactionID *string            `json:"provider_transaction_id,omitempty"`
	ProviderReference     *string            `json:"provider_reference,omitempty"`
	FingerprintHash       *string            `json:"fingerprint_hash,omitempty"`
	PrintConfirmation     *PrintConfirmation `json:"print_confirmation,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
}

type PrintConfirmation struct {
	CheckID     string                    `json:"check_id"`
	Confirmed   bool                      `json:"confirmed"`
	ConfirmedAt *time.Time                `json:"confirmed_at,omitempty"`
	Targets     []PrintConfirmationTarget `json:"targets"`
}

type PrintConfirmationTarget struct {
	ID            string     `json:"id"`
	PrintJobID    string     `json:"print_job_id"`
	PrinterID     string     `json:"printer_id"`
	ScopeType     string     `json:"scope_type"`
	ScopeID       *string    `json:"scope_id,omitempty"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	MaxAttempts   int        `json:"max_attempts"`
	IsRequired    bool       `json:"is_required"`
	LastError     *string    `json:"last_error,omitempty"`
	NextAttemptAt *time.Time `json:"next_attempt_at,omitempty"`
	PrintedAt     *time.Time `json:"printed_at,omitempty"`
}

type PaymentAttempt struct {
	ID                    string        `json:"id"`
	PaymentID             string        `json:"payment_id"`
	AttemptNo             int           `json:"attempt_no"`
	Method                PaymentMethod `json:"method"`
	Amount                int64         `json:"amount"`
	Currency              string        `json:"currency"`
	Status                PaymentStatus `json:"status"`
	ProviderName          *string       `json:"provider_name,omitempty"`
	ProviderTransactionID *string       `json:"provider_transaction_id,omitempty"`
	ProviderReference     *string       `json:"provider_reference,omitempty"`
	FingerprintHash       *string       `json:"fingerprint_hash,omitempty"`
	AttemptedAt           time.Time     `json:"attempted_at"`
	CreatedAt             time.Time     `json:"created_at"`
}
