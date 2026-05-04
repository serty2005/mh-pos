package domain

import "time"

type Restaurant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Timezone  string    `json:"timezone"`
	Currency  string    `json:"currency"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Device struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	DeviceCode   string    `json:"device_code"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Active       bool      `json:"active"`
	RegisteredAt time.Time `json:"registered_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Role struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	PermissionsJSON string    `json:"permissions_json"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Employee struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	RoleID       string    `json:"role_id"`
	Name         string    `json:"name"`
	PINHash      string    `json:"pin_hash"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CatalogItemType string

const (
	CatalogItemIngredient CatalogItemType = "ingredient"
	CatalogItemDish       CatalogItemType = "dish"
	CatalogItemGood       CatalogItemType = "good"
)

type CatalogItem struct {
	ID        string          `json:"id"`
	Type      CatalogItemType `json:"type"`
	Name      string          `json:"name"`
	SKU       string          `json:"sku"`
	BaseUnit  string          `json:"base_unit"`
	Active    bool            `json:"active"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type MenuItem struct {
	ID            string    `json:"id"`
	CatalogItemID string    `json:"catalog_item_id"`
	Name          string    `json:"name"`
	Price         int64     `json:"price"`
	Currency      string    `json:"currency"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ShiftStatus string

const (
	ShiftOpen   ShiftStatus = "open"
	ShiftClosed ShiftStatus = "closed"
)

type Shift struct {
	ID                 string      `json:"id"`
	RestaurantID       string      `json:"restaurant_id"`
	DeviceID           string      `json:"device_id"`
	OpenedByEmployeeID string      `json:"opened_by_employee_id"`
	ClosedByEmployeeID *string     `json:"closed_by_employee_id,omitempty"`
	Status             ShiftStatus `json:"status"`
	OpenedAt           time.Time   `json:"opened_at"`
	ClosedAt           *time.Time  `json:"closed_at,omitempty"`
	OpeningCashAmount  int64       `json:"opening_cash_amount"`
	ClosingCashAmount  *int64      `json:"closing_cash_amount,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

type OrderStatus string

const (
	OrderOpen      OrderStatus = "open"
	OrderClosed    OrderStatus = "closed"
	OrderCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID           string      `json:"id"`
	EdgeOrderID  string      `json:"edge_order_id"`
	RestaurantID string      `json:"restaurant_id"`
	DeviceID     string      `json:"device_id"`
	ShiftID      string      `json:"shift_id"`
	Status       OrderStatus `json:"status"`
	TableName    string      `json:"table_name"`
	GuestCount   int         `json:"guest_count"`
	OpenedAt     time.Time   `json:"opened_at"`
	ClosedAt     *time.Time  `json:"closed_at,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Lines        []OrderLine `json:"lines,omitempty"`
	Check        *Check      `json:"check,omitempty"`
}

type OrderLineStatus string

const (
	OrderLineActive    OrderLineStatus = "active"
	OrderLineCancelled OrderLineStatus = "cancelled"
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

type CheckStatus string

const (
	CheckOpen     CheckStatus = "open"
	CheckPaid     CheckStatus = "paid"
	CheckRefunded CheckStatus = "refunded"
	CheckVoided   CheckStatus = "voided"
)

type Check struct {
	ID            string      `json:"id"`
	OrderID       string      `json:"order_id"`
	Status        CheckStatus `json:"status"`
	Subtotal      int64       `json:"subtotal"`
	DiscountTotal int64       `json:"discount_total"`
	TaxTotal      int64       `json:"tax_total"`
	Total         int64       `json:"total"`
	PaidTotal     int64       `json:"paid_total"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

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

type OutboxStatus string
type CommandOrigin string

const (
	OutboxPending OutboxStatus = "pending"
	OutboxSent    OutboxStatus = "sent"
	OutboxFailed  OutboxStatus = "failed"

	OriginEdgeDevice CommandOrigin = "edge_device"
	OriginCloudSync  CommandOrigin = "cloud_sync"
	OriginSystemSeed CommandOrigin = "system_seed"
)

type OutboxMessage struct {
	ID            string        `json:"id"`
	CommandID     string        `json:"command_id"`
	Origin        CommandOrigin `json:"origin"`
	RestaurantID  *string       `json:"restaurant_id,omitempty"`
	DeviceID      string        `json:"device_id"`
	AggregateType string        `json:"aggregate_type"`
	AggregateID   string        `json:"aggregate_id"`
	CommandType   string        `json:"command_type"`
	PayloadJSON   string        `json:"payload_json"`
	Status        OutboxStatus  `json:"status"`
	Attempts      int           `json:"attempts"`
	LastError     *string       `json:"last_error,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}
