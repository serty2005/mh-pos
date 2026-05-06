package ports

type Repository interface {
	RestaurantRepository
	DeviceRepository
	EmployeeRepository
	CatalogRepository
	MenuRepository
	ShiftRepository
	OrderRepository
	PrecheckRepository
	CheckRepository
	CashRepository
	InventoryRepository
	LocalEventRepository
	OutboxRepository
}
