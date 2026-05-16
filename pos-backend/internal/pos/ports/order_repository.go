package ports

import (
	"context"

	"pos-backend/internal/pos/domain/order"
)

type OrderRepository interface {
	CreateOrder(context.Context, *order.Order) error
	GetOrder(context.Context, string) (*order.Order, error)
	GetActiveOrderByDeviceAndTable(context.Context, string, string) (*order.Order, error)
	ListActiveOrdersByRestaurantAndHall(context.Context, string, string) ([]order.Order, error)
	ListClosedOrders(context.Context, int) ([]order.OrderSummary, error)
	UpdateOrderOpen(context.Context, *order.Order) error
	UpdateOrderLocked(context.Context, *order.Order) error
	UpdateOrderClosed(context.Context, *order.Order) error
	CreateOrderLine(context.Context, *order.OrderLine) error
	CreateOrderLineModifier(context.Context, *order.LineModifier) error
	GetOrderLine(context.Context, string) (*order.OrderLine, error)
	UpdateOrderLine(context.Context, *order.OrderLine) error
	UpdateOrderLineDetails(context.Context, *order.OrderLine) error
	ListOrderLines(context.Context, string) ([]order.OrderLine, error)
}
