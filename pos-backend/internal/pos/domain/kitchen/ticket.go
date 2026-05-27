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

// OrderStatus описывает вычисляемое состояние кухонного заказа поверх ticket statuses.
type OrderStatus string

const (
	OrderQueued          OrderStatus = "queued"
	OrderAccepted        OrderStatus = "accepted"
	OrderInProgress      OrderStatus = "in_progress"
	OrderPartiallyReady  OrderStatus = "partially_ready"
	OrderReady           OrderStatus = "ready"
	OrderPartiallyServed OrderStatus = "partially_served"
	OrderServed          OrderStatus = "served"
	OrderCancelled       OrderStatus = "cancelled"
	OrderMixed           OrderStatus = "mixed"
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

// OrderTicket расширяет kitchen ticket ссылкой на edge order id для grouped KDS read model.
type OrderTicket struct {
	Ticket
	EdgeOrderID string `json:"edge_order_id"`
}

// TicketListQuery задает bounded read параметры KDS списка.
type TicketListQuery struct {
	RestaurantID string
	Status       TicketStatus
	Station      string
	Limit        int
	Offset       int
}

// OrderQueueQuery задает фильтры grouped kitchen order queue.
type OrderQueueQuery struct {
	RestaurantID string
	Station      string
	Limit        int
	Offset       int
}

// OrderQueue является paged grouped read model для KDS очереди заказов.
type OrderQueue struct {
	Orders []Order `json:"orders"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// Order группирует kitchen tickets одного POS order и хранит вычисляемый статус кухни.
type Order struct {
	OrderID             string      `json:"order_id"`
	EdgeOrderID         string      `json:"edge_order_id"`
	TableName           string      `json:"table_name"`
	ShiftID             string      `json:"shift_id"`
	KitchenOrderStatus  OrderStatus `json:"kitchen_order_status"`
	CreatedAt           time.Time   `json:"created_at"`
	LastStatusChangedAt time.Time   `json:"last_status_changed_at"`
	ElapsedSeconds      int64       `json:"elapsed_seconds"`
	Tickets             []Ticket    `json:"tickets"`
}

// TicketEvent фиксирует локальный audit trail смены KDS статуса.
type TicketEvent struct {
	ID              string       `json:"id"`
	TicketID        string       `json:"ticket_id"`
	OrderLineID     string       `json:"order_line_id"`
	FromStatus      TicketStatus `json:"from_status"`
	ToStatus        TicketStatus `json:"to_status"`
	CommandID       string       `json:"command_id"`
	ActorEmployeeID string       `json:"actor_employee_id,omitempty"`
	OccurredAt      time.Time    `json:"occurred_at"`
	CreatedAt       time.Time    `json:"created_at"`
}
