package order

import "time"

type OrderLineStatus string

const (
	OrderLineActive    OrderLineStatus = "active"
	OrderLineCancelled OrderLineStatus = "cancelled"
	OrderLineVoided    OrderLineStatus = "voided"
)

type OrderLine struct {
	ID            string          `json:"id"`
	OrderID       string          `json:"order_id"`
	MenuItemID    string          `json:"menu_item_id"`
	CatalogItemID string          `json:"catalog_item_id"`
	Name          string          `json:"name"`
	Quantity      int64           `json:"quantity"`
	UnitPrice     int64           `json:"unit_price"`
	TotalPrice    int64           `json:"total_price"`
	Status        OrderLineStatus `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
