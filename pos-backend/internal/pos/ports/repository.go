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
	FinancialOperationRepository
	PricingRepository
	CashRepository
	InventoryRepository
	MasterSyncRepository
	StorageLifecycleRepository
	LocalEventRepository
	OutboxRepository
}
