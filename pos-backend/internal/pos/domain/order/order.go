package order

import (
	"time"

	"pos-backend/internal/pos/domain/check"
)

type OrderStatus string

const (
	OrderOpen      OrderStatus = "open"
	OrderLocked    OrderStatus = "locked"
	OrderClosed    OrderStatus = "closed"
	OrderCancelled OrderStatus = "cancelled"
)

type OrderSummary struct {
	ID        string       `json:"id"`
	TableName string       `json:"table_name"`
	OpenedAt  time.Time    `json:"opened_at"`
	ClosedAt  *time.Time   `json:"closed_at,omitempty"`
	Total     int64        `json:"total"`
	Status    OrderStatus  `json:"status"`
	Check     *check.Check `json:"check,omitempty"`
}

// ClosedOrderListQuery задает bounded read model для просмотра закрытых заказов без полной выгрузки истории.
type ClosedOrderListQuery struct {
	RestaurantID          string
	BusinessDateLocal     string
	FromBusinessDateLocal string
	ToBusinessDateLocal   string
	ShiftID               string
	DeviceID              string
	CheckID               string
	Limit                 int
	Offset                int
}

type Order struct {
	ID            string       `json:"id"`
	EdgeOrderID   string       `json:"edge_order_id"`
	RestaurantID  string       `json:"restaurant_id"`
	DeviceID      string       `json:"device_id"`
	ShiftID       string       `json:"shift_id"`
	Status        OrderStatus  `json:"status"`
	TableID       string       `json:"table_id"`
	TableName     string       `json:"table_name"`
	GuestCount    int          `json:"guest_count"`
	Subtotal      int64        `json:"subtotal"`
	DiscountTotal int64        `json:"discount_total"`
	TaxTotal      int64        `json:"tax_total"`
	Total         int64        `json:"total"`
	OpenedAt      time.Time    `json:"opened_at"`
	ClosedAt      *time.Time   `json:"closed_at,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	Lines         []OrderLine  `json:"lines,omitempty"`
	Check         *check.Check `json:"check,omitempty"`
}
