package kitchen

import "time"

// TicketStatus описывает backend-authoritative lifecycle кухонной строки.
type TicketStatus string

const (
	TicketNew        TicketStatus = "new"
	TicketAccepted   TicketStatus = "accepted"
	TicketInProgress TicketStatus = "in_progress"
	TicketHold       TicketStatus = "hold"
	TicketReady      TicketStatus = "ready"
	TicketServed     TicketStatus = "served"
	TicketRecall     TicketStatus = "recall"
	TicketCancelled  TicketStatus = "cancelled"
)

// Ticket является KDS read/action model, привязанной к одной active order line.
type Ticket struct {
	ID                string       `json:"id"`
	RestaurantID      string       `json:"restaurant_id"`
	DeviceID          string       `json:"device_id"`
	ShiftID           string       `json:"shift_id"`
	OrderID           string       `json:"order_id"`
	OrderLineID       string       `json:"order_line_id"`
	TableName         string       `json:"table_name"`
	MenuItemID        string       `json:"menu_item_id"`
	CatalogItemID     string       `json:"catalog_item_id"`
	Name              string       `json:"name"`
	Quantity          int64        `json:"quantity"`
	UnitCode          string       `json:"unit_code"`
	StationRoutingKey string       `json:"station_routing_key,omitempty"`
	Course            *string      `json:"course,omitempty"`
	Comment           *string      `json:"comment,omitempty"`
	Status            TicketStatus `json:"status"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

// TicketListQuery задает bounded read параметры KDS списка.
type TicketListQuery struct {
	RestaurantID string
	Status       TicketStatus
	Limit        int
	Offset       int
}

// TicketEvent фиксирует локальный audit trail смены KDS статуса.
type TicketEvent struct {
	ID             string       `json:"id"`
	TicketID       string       `json:"ticket_id"`
	OrderLineID    string       `json:"order_line_id"`
	FromStatus     TicketStatus `json:"from_status"`
	ToStatus       TicketStatus `json:"to_status"`
	CommandID      string       `json:"command_id"`
	ActorEmployeeID string     `json:"actor_employee_id,omitempty"`
	OccurredAt     time.Time    `json:"occurred_at"`
	CreatedAt      time.Time    `json:"created_at"`
}

