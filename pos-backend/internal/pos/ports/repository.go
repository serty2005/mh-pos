package ports

type Repository interface {
	RestaurantRepository
	DeviceRepository
	FloorRepository
	EmployeeRepository
	CatalogRepository
	MenuRepository
	ShiftRepository
	OrderRepository
	PrecheckRepository
	CheckRepository
	PricingRepository
	CashRepository
	InventoryRepository
	MasterSyncRepository
	LocalEventRepository
	OutboxRepository
}
