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
	InventoryRepository
	OutboxRepository
}
