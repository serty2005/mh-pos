package order

import (
	"time"

	"pos-backend/internal/pos/domain/check"
)

type OrderStatus string

const (
	OrderOpen      OrderStatus = "open"
	OrderClosed    OrderStatus = "closed"
	OrderCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID           string       `json:"id"`
	EdgeOrderID  string       `json:"edge_order_id"`
	RestaurantID string       `json:"restaurant_id"`
	DeviceID     string       `json:"device_id"`
	ShiftID      string       `json:"shift_id"`
	Status       OrderStatus  `json:"status"`
	TableName    string       `json:"table_name"`
	GuestCount   int          `json:"guest_count"`
	OpenedAt     time.Time    `json:"opened_at"`
	ClosedAt     *time.Time   `json:"closed_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Lines        []OrderLine  `json:"lines,omitempty"`
	Check        *check.Check `json:"check,omitempty"`
}
