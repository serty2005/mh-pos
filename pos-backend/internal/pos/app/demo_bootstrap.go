package app

import (
	"context"
	"strings"

	appshared "pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
)

const (
	demoRestaurantName = "Demo Bistro"
	demoNodeDeviceID   = "demo-edge-node-1"
	demoCashierPIN     = "1111"
	demoManagerPIN     = "2222"
	demoCurrency       = "RUB"
)

type DemoBootstrapResult struct {
	RestaurantID      string   `json:"restaurant_id"`
	NodeDeviceID      string   `json:"node_device_id"`
	PairingCode       string   `json:"pairing_code"`
	CashierPIN        string   `json:"cashier_pin"`
	ManagerPIN        string   `json:"manager_pin"`
	CashierEmployeeID string   `json:"cashier_employee_id"`
	ManagerEmployeeID string   `json:"manager_employee_id"`
	HallID            string   `json:"hall_id"`
	TableIDs          []string `json:"table_ids"`
	MenuItemIDs       []string `json:"menu_item_ids"`
}

func (s *Service) BootstrapDemo(ctx context.Context) (*DemoBootstrapResult, error) {
	restaurant, err := s.ensureDemoRestaurant(ctx)
	if err != nil {
		return nil, err
	}
	pairingCode := "MHPOS:" + restaurant.ID + ":" + demoNodeDeviceID
	if _, err := s.PairEdgeNode(ctx, PairEdgeNodeCommand{PairingCode: pairingCode}); err != nil {
		return nil, err
	}
	cashierRole, err := s.ensureDemoRole(ctx, string(appshared.RoleCashier), appshared.RolePermissionsJSON(appshared.RoleCashier), appshared.PermissionEmployeeShiftOpen, appshared.PermissionEmployeeShiftRecent, appshared.PermissionFloorView, appshared.PermissionMenuView, appshared.PermissionOrderCreate, appshared.PermissionOrderView, appshared.PermissionPrecheckIssue, appshared.PermissionPrecheckView, appshared.PermissionPaymentCash, appshared.PermissionCheckView)
	if err != nil {
		return nil, err
	}
	managerRole, err := s.ensureDemoRole(ctx, string(appshared.RoleManager), appshared.RolePermissionsJSON(appshared.RoleManager), appshared.PermissionFloorView, appshared.PermissionMenuView, appshared.PermissionPrecheckCancelRequest, appshared.PermissionPrecheckCancel, appshared.PermissionSyncView, appshared.PermissionSyncRetryFailed)
	if err != nil {
		return nil, err
	}
	cashier, err := s.ensureDemoEmployee(ctx, restaurant.ID, cashierRole.ID, "Demo Cashier", demoCashierPIN, []byte("demo-cashier-pin-salt-v1"))
	if err != nil {
		return nil, err
	}
	manager, err := s.ensureDemoEmployee(ctx, restaurant.ID, managerRole.ID, "Demo Manager", demoManagerPIN, []byte("demo-manager-pin-salt-v1"))
	if err != nil {
		return nil, err
	}
	hall, err := s.ensureDemoHall(ctx, restaurant.ID)
	if err != nil {
		return nil, err
	}
	tables, err := s.ensureDemoTables(ctx, restaurant.ID, hall.ID)
	if err != nil {
		return nil, err
	}
	menuItems, err := s.ensureDemoMenu(ctx)
	if err != nil {
		return nil, err
	}
	result := &DemoBootstrapResult{
		RestaurantID:      restaurant.ID,
		NodeDeviceID:      demoNodeDeviceID,
		PairingCode:       pairingCode,
		CashierPIN:        demoCashierPIN,
		ManagerPIN:        demoManagerPIN,
		CashierEmployeeID: cashier.ID,
		ManagerEmployeeID: manager.ID,
		HallID:            hall.ID,
		TableIDs:          idsOfTables(tables),
		MenuItemIDs:       idsOfMenuItems(menuItems),
	}
	return result, nil
}

func (s *Service) ensureDemoRestaurant(ctx context.Context) (*domain.Restaurant, error) {
	items, err := s.ListRestaurants(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Active && strings.EqualFold(items[i].Name, demoRestaurantName) {
			return &items[i], nil
		}
	}
	return s.CreateRestaurant(ctx, CreateRestaurantCommand{
		CommandMeta: seedMeta("demo-create-restaurant"),
		Name:        demoRestaurantName,
		Timezone:    "Europe/Moscow",
		Currency:    demoCurrency,
	})
}

func (s *Service) ensureDemoRole(ctx context.Context, name, permissions string, requiredPermissions ...appshared.PermissionID) (*domain.Role, error) {
	items, err := s.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Active && strings.EqualFold(items[i].Name, name) && roleHasAllPermissions(items[i], requiredPermissions...) {
			return &items[i], nil
		}
	}
	return s.CreateRole(ctx, CreateRoleCommand{
		CommandMeta:     seedMeta("demo-create-role-" + strings.ToLower(name)),
		Name:            name,
		PermissionsJSON: permissions,
	})
}

func (s *Service) ensureDemoEmployee(ctx context.Context, restaurantID, roleID, name, pin string, salt []byte) (*domain.Employee, error) {
	items, err := s.ListEmployees(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Active && items[i].RestaurantID == restaurantID && items[i].RoleID == roleID && appshared.VerifyPIN(items[i].PINHash, pin) == nil {
			return &items[i], nil
		}
	}
	hash, err := appshared.HashPIN(pin, salt)
	if err != nil {
		return nil, err
	}
	return s.CreateEmployee(ctx, CreateEmployeeCommand{
		CommandMeta:  seedMeta("demo-create-employee-" + strings.ToLower(strings.ReplaceAll(name, " ", "-"))),
		RestaurantID: restaurantID,
		RoleID:       roleID,
		Name:         name,
		PINHash:      hash,
	})
}

func (s *Service) ensureDemoHall(ctx context.Context, restaurantID string) (*domain.Hall, error) {
	items, err := s.ListHalls(ctx, restaurantID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Active && strings.EqualFold(items[i].Name, "Main Hall") {
			return &items[i], nil
		}
	}
	return s.CreateHall(ctx, CreateHallCommand{
		CommandMeta:  seedMeta("demo-create-hall-main"),
		RestaurantID: restaurantID,
		Name:         "Main Hall",
	})
}

func (s *Service) ensureDemoTables(ctx context.Context, restaurantID, hallID string) ([]domain.Table, error) {
	items, err := s.ListTables(ctx, restaurantID, hallID)
	if err != nil {
		return nil, err
	}
	required := []struct {
		name  string
		seats int
	}{
		{name: "A1", seats: 2},
		{name: "A2", seats: 4},
	}
	for _, want := range required {
		found := false
		for _, item := range items {
			if item.Active && strings.EqualFold(item.Name, want.name) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		table, err := s.CreateTable(ctx, CreateTableCommand{
			CommandMeta:  seedMeta("demo-create-table-" + strings.ToLower(want.name)),
			RestaurantID: restaurantID,
			HallID:       hallID,
			Name:         want.name,
			Seats:        want.seats,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, *table)
	}
	return items, nil
}

func (s *Service) ensureDemoMenu(ctx context.Context) ([]domain.MenuItem, error) {
	required := []struct {
		name  string
		sku   string
		price int64
	}{
		{name: "Coffee", sku: "DEMO-COFFEE", price: 25000},
		{name: "Pasta", sku: "DEMO-PASTA", price: 75000},
		{name: "Salad", sku: "DEMO-SALAD", price: 52000},
	}
	var out []domain.MenuItem
	for _, want := range required {
		catalogItem, err := s.ensureDemoCatalogItem(ctx, want.name, want.sku)
		if err != nil {
			return nil, err
		}
		menuItem, err := s.ensureDemoMenuItem(ctx, catalogItem.ID, want.name, want.price)
		if err != nil {
			return nil, err
		}
		out = append(out, *menuItem)
	}
	return out, nil
}

func (s *Service) ensureDemoCatalogItem(ctx context.Context, name, sku string) (*domain.CatalogItem, error) {
	items, err := s.ListCatalogItems(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Active && strings.EqualFold(items[i].SKU, sku) {
			return &items[i], nil
		}
	}
	return s.CreateCatalogItem(ctx, CreateCatalogItemCommand{
		CommandMeta: seedMeta("demo-create-catalog-" + strings.ToLower(sku)),
		Type:        domain.CatalogItemDish,
		Name:        name,
		SKU:         sku,
		BaseUnit:    "portion",
	})
}

func (s *Service) ensureDemoMenuItem(ctx context.Context, catalogItemID, name string, price int64) (*domain.MenuItem, error) {
	items, err := s.ListMenuItems(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Active && items[i].CatalogItemID == catalogItemID {
			return &items[i], nil
		}
	}
	return s.CreateMenuItem(ctx, CreateMenuItemCommand{
		CommandMeta:   seedMeta("demo-create-menu-" + strings.ToLower(strings.ReplaceAll(name, " ", "-"))),
		CatalogItemID: catalogItemID,
		Name:          name,
		Price:         price,
		Currency:      demoCurrency,
	})
}

func seedMeta(commandID string) CommandMeta {
	return CommandMeta{
		CommandID:    commandID,
		NodeDeviceID: demoNodeDeviceID,
		DeviceID:     demoNodeDeviceID,
		Origin:       OriginSystemSeed,
	}
}

func idsOfTables(items []domain.Table) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.Active {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func idsOfMenuItems(items []domain.MenuItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.Active {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func roleHasAllPermissions(role domain.Role, required ...appshared.PermissionID) bool {
	for _, permission := range required {
		if !appshared.HasPermission(role.PermissionsJSON, string(permission)) {
			return false
		}
	}
	return true
}
