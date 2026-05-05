package ports

type Repository interface {
	RestaurantRepository
	DeviceRepository
	EmployeeRepository
	CatalogRepository
	MenuRepository
	ShiftRepository
	OrderRepository
	CheckRepository
	CashRepository
	InventoryRepository
	LocalEventRepository
	OutboxRepository
}
