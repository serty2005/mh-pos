package ports

import (
	"context"

	"pos-backend/internal/pos/domain"
)

type Repository interface {
	CreateRestaurant(context.Context, *domain.Restaurant) error
	ListRestaurants(context.Context) ([]domain.Restaurant, error)

	CreateDevice(context.Context, *domain.Device) error
	ListDevices(context.Context) ([]domain.Device, error)

	CreateRole(context.Context, *domain.Role) error
	ListRoles(context.Context) ([]domain.Role, error)

	CreateEmployee(context.Context, *domain.Employee) error
	ListEmployees(context.Context) ([]domain.Employee, error)
	ArchiveEmployee(context.Context, string, string) error

	CreateCatalogItem(context.Context, *domain.CatalogItem) error
	ListCatalogItems(context.Context) ([]domain.CatalogItem, error)
	GetCatalogItem(context.Context, string) (*domain.CatalogItem, error)
	CatalogItemInUse(context.Context, string) (bool, error)

	CreateMenuItem(context.Context, *domain.MenuItem) error
	ListMenuItems(context.Context) ([]domain.MenuItem, error)
	GetMenuItem(context.Context, string) (*domain.MenuItem, error)

	CreateShift(context.Context, *domain.Shift) error
	UpdateShiftClosed(context.Context, *domain.Shift) error
	GetShift(context.Context, string) (*domain.Shift, error)
	GetOpenShiftByDevice(context.Context, string) (*domain.Shift, error)
	HasOpenOrdersForShift(context.Context, string) (bool, error)

	CreateOrder(context.Context, *domain.Order) error
	GetOrder(context.Context, string) (*domain.Order, error)
	UpdateOrderClosed(context.Context, *domain.Order) error
	CreateOrderLine(context.Context, *domain.OrderLine) error
	ListOrderLines(context.Context, string) ([]domain.OrderLine, error)

	CreateCheck(context.Context, *domain.Check) error
	GetCheck(context.Context, string) (*domain.Check, error)
	GetCheckByOrder(context.Context, string) (*domain.Check, error)
	UpdateCheckPaidTotal(context.Context, *domain.Check) error

	CreatePayment(context.Context, *domain.Payment) error

	CreateOutboxMessage(context.Context, *domain.OutboxMessage) error
	GetOutboxByCommandID(context.Context, string) (*domain.OutboxMessage, error)
	ListOutbox(context.Context, int) ([]domain.OutboxMessage, error)
	MarkOutboxSent(context.Context, string, string) error
	MarkOutboxFailed(context.Context, string, string, string) error
	CountOutbox(context.Context) (int, error)
}
