package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
	"cloud-backend/internal/masterdata/infra/memory"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
}

type fixedIDs struct {
	next int
}

func (f *fixedIDs) NewID() string {
	f.next++
	return "id-" + strconv.Itoa(f.next)
}

func TestRestaurantCRUDAndValidation(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{
		Name:                         "Demo Bistro",
		Timezone:                     "Asia/Jakarta",
		Currency:                     "IDR",
		BusinessDayMode:              "standard",
		BusinessDayBoundaryLocalTime: "05:30",
	})
	if err != nil {
		t.Fatal(err)
	}
	if restaurant.Status != domain.RestaurantActive || restaurant.CloudVersion != 1 {
		t.Fatalf("unexpected restaurant: %+v", restaurant)
	}
	if _, err := service.UpdateRestaurant(ctx, restaurant.ID, app.UpdateRestaurantCommand{Currency: "ZZZ"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid currency, got %v", err)
	}
	archived, err := service.ArchiveRestaurant(ctx, restaurant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if archived.Status != domain.RestaurantArchived || archived.ArchivedAt == nil || archived.CloudVersion != 2 {
		t.Fatalf("expected soft archive with version bump, got %+v", archived)
	}
}

func TestEmployeeLifecyclePINAndPermissionSnapshot(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	role, err := service.CreateRole(ctx, app.CreateRoleCommand{
		Name:            "manager",
		PermissionsJSON: `{"pos.menu.view":true,"pos.payment.cash":true}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{
		RestaurantIDs: []string{"restaurant-1"}, RoleID: role.ID,
		Name: "Anna",
		PIN:  "1111",
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(employee)
	if strings.Contains(string(body), "pin_hash") || strings.Contains(string(body), "1111") || strings.Contains(string(body), employee.PINHash) {
		t.Fatalf("employee API model must not expose PIN or hash: %s", body)
	}
	if employee.Status != domain.EmployeeActive || employee.PermissionSnapshotJSON == "" || employee.PINCredentialVersion != 1 {
		t.Fatalf("unexpected employee foundation: %+v", employee)
	}
	suspended, err := service.SuspendEmployee(ctx, employee.ID)
	if err != nil {
		t.Fatal(err)
	}
	if suspended.Status != domain.EmployeeSuspended || suspended.ActiveForPOS() {
		t.Fatalf("expected suspended employee to be inactive for POS: %+v", suspended)
	}
	rotated, err := service.RotateEmployeePIN(ctx, employee.ID, app.RotatePINCommand{PIN: "2222"})
	if err != nil {
		t.Fatal(err)
	}
	if rotated.PINCredentialVersion != 2 || rotated.PINHash == employee.PINHash {
		t.Fatalf("expected PIN rotation version/hash change: %+v", rotated)
	}
	archived, err := service.ArchiveEmployee(ctx, employee.ID)
	if err != nil {
		t.Fatal(err)
	}
	if archived.Status != domain.EmployeeArchived || archived.ActiveForPOS() {
		t.Fatalf("expected archived employee to be inactive for POS: %+v", archived)
	}
}

func TestRoleRejectsUnknownPermissionID(t *testing.T) {
	service, _ := newService()
	_, err := service.CreateRole(context.Background(), app.CreateRoleCommand{
		Name:            "broken",
		PermissionsJSON: `{"pos.order.create":true,"pos.unknown.permission":true}`,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid unknown permission, got %v", err)
	}
}

func TestRoleAcceptsKitchenPermissionIDs(t *testing.T) {
	service, _ := newService()
	role, err := service.CreateRole(context.Background(), app.CreateRoleCommand{
		Name:            "kitchen",
		PermissionsJSON: `{"pos.kitchen.view":true,"pos.kitchen.status.change":true,"pos.print.status":true,"permissions":["pos.kitchen.stock.receipt","pos.kitchen.production.complete","pos.print.retry","pos.print_routing.view","pos.print_routing.manage","pos.order.cancel_unconfirmed"]}`,
	})
	if err != nil {
		t.Fatalf("expected kitchen/print permissions to be accepted, got %v", err)
	}
	if !strings.Contains(role.PermissionsJSON, "pos.kitchen.stock.receipt") || !strings.Contains(role.PermissionsJSON, "pos.print_routing.manage") || !strings.Contains(role.PermissionsJSON, "pos.order.cancel_unconfirmed") {
		t.Fatalf("expected kitchen/print permissions to be persisted, got %s", role.PermissionsJSON)
	}
}

func TestTenantStaffMembershipsFilterRestaurantPublication(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	second, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Second", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	managerRole, err := service.CreateRole(ctx, app.CreateRoleCommand{Name: "organization manager", PermissionsJSON: `{"organization.manage":true}`})
	if err != nil {
		t.Fatal(err)
	}
	staffRole, err := service.CreateRole(ctx, app.CreateRoleCommand{Name: "cashier", PermissionsJSON: `{"pos.order.create":true}`})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RoleID: managerRole.ID, Name: "Manager", PIN: "1111"})
	if err != nil || !manager.AllRestaurants || len(manager.RestaurantIDs) != 0 {
		t.Fatalf("unexpected organization manager: %+v err=%v", manager, err)
	}
	employee, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RoleID: staffRole.ID, RestaurantIDs: []string{"restaurant-1"}, Name: "Cashier", PIN: "2222"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RoleID: staffRole.ID, Name: "No scope", PIN: "3333"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected last-membership validation, got %v", err)
	}
	if _, err := service.Publish(ctx, app.PublishCommand{RestaurantID: "restaurant-1", PublishedBy: "test"}); err != nil {
		t.Fatal(err)
	}
	first, err := service.GetCurrentPublishedPackage(ctx, "restaurant-1", "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Employees) != 2 {
		t.Fatalf("expected manager and member, got %+v", first.Employees)
	}
	if _, err := service.Publish(ctx, app.PublishCommand{RestaurantID: second.ID, PublishedBy: "test"}); err != nil {
		t.Fatal(err)
	}
	other, err := service.GetCurrentPublishedPackage(ctx, second.ID, "node-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(other.Employees) != 1 || other.Employees[0].ID != manager.ID {
		t.Fatalf("expected only organization manager, got %+v", other.Employees)
	}
	empty := []string{}
	if _, err := service.UpdateEmployee(ctx, employee.ID, app.UpdateEmployeeCommand{RestaurantIDs: &empty}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected revoke of last membership to fail, got %v", err)
	}
	secondOnly := []string{second.ID}
	if _, err := service.UpdateEmployee(ctx, employee.ID, app.UpdateEmployeeCommand{RestaurantIDs: &secondOnly}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Publish(ctx, app.PublishCommand{RestaurantID: "restaurant-1", PublishedBy: "test"}); err != nil {
		t.Fatal(err)
	}
	revoked, err := service.GetCurrentPublishedPackage(ctx, "restaurant-1", "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(revoked.Employees) != 1 || revoked.Employees[0].ID != manager.ID {
		t.Fatalf("revoked employee remained eligible: %+v", revoked.Employees)
	}
}

func TestDuplicateActivePINIsRejectedPerRestaurant(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	role, err := service.CreateRole(ctx, app.CreateRoleCommand{Name: "cashier", PermissionsJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantIDs: []string{"restaurant-1"}, RoleID: role.ID, Name: "Anna", PIN: "1111"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantIDs: []string{"restaurant-1"}, RoleID: role.ID, Name: "Ivan", PIN: "1111"}); !errors.Is(err, domain.ErrPINAlreadyExists) {
		t.Fatalf("expected duplicate PIN conflict, got %v", err)
	}
}

func TestSuspendedEmployeePINStillBlocksReuse(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	role, err := service.CreateRole(ctx, app.CreateRoleCommand{Name: "cashier", PermissionsJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantIDs: []string{"restaurant-1"}, RoleID: role.ID, Name: "Anna", PIN: "1111"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SuspendEmployee(ctx, employee.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantIDs: []string{"restaurant-1"}, RoleID: role.ID, Name: "Ivan", PIN: "1111"}); !errors.Is(err, domain.ErrPINAlreadyExists) {
		t.Fatalf("expected suspended employee PIN to stay reserved, got %v", err)
	}
}

func TestArchivedEmployeePINCanBeReused(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	role, err := service.CreateRole(ctx, app.CreateRoleCommand{Name: "cashier", PermissionsJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantIDs: []string{"restaurant-1"}, RoleID: role.ID, Name: "Anna", PIN: "1111"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ArchiveEmployee(ctx, employee.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantIDs: []string{"restaurant-1"}, RoleID: role.ID, Name: "Ivan", PIN: "1111"}); err != nil {
		t.Fatalf("expected archived employee PIN to be reusable, got %v", err)
	}
}

func TestCatalogMenuValidationAndPublicationPackageShape(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo Bistro", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	role, err := service.CreateRole(ctx, app.CreateRoleCommand{Name: "cashier", PermissionsJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantIDs: []string{restaurant.ID}, RoleID: role.ID, Name: "Oleg", PIN: "3333"})
	if err != nil {
		t.Fatal(err)
	}
	category, err := service.CreateCategory(ctx, app.CreateCategoryCommand{RestaurantID: restaurant.ID, Name: "Bar", SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemSemiFinished, Name: "Syrup", SKU: "SYRUP", BaseUnit: "ml"})
	if err != nil {
		t.Fatal(err)
	}
	folder, err := service.CreateCatalogFolder(ctx, app.CreateCatalogFolderCommand{RestaurantID: restaurant.ID, Name: "Bar folder", SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	folderParameter, err := service.CreateFolderParameter(ctx, app.CreateFolderParameterCommand{RestaurantID: restaurant.ID, FolderID: folder.ID, Key: "station", ValueType: "string", ValueJSON: `"bar"`})
	if err != nil {
		t.Fatal(err)
	}
	tag, err := service.CreateCatalogTag(ctx, app.CreateCatalogTagCommand{RestaurantID: restaurant.ID, Name: "Coffee", Code: "COFFEE"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AssignCatalogItemTag(ctx, app.AssignCatalogItemTagCommand{RestaurantID: restaurant.ID, CatalogItemID: catalog.ID, TagID: tag.ID}); err != nil {
		t.Fatal(err)
	}
	published := domain.StatusPublished
	if _, err := service.UpdateCatalogItem(ctx, catalog.ID, app.UpdateCatalogItemCommand{Status: &published}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateMenuItem(ctx, app.CreateMenuItemCommand{RestaurantID: restaurant.ID, CatalogItemID: catalog.ID, CategoryID: category.ID, Name: "Tea", Price: 2500, Currency: "rub", AvailabilityJSON: `not json`}); err == nil {
		t.Fatal("expected invalid availability_json to be rejected")
	}
	menu, err := service.CreateMenuItem(ctx, app.CreateMenuItemCommand{RestaurantID: restaurant.ID, CatalogItemID: catalog.ID, CategoryID: category.ID, Name: "Tea", Price: 2500, Currency: "rub", AvailabilityJSON: `{"days":["mon"]}`})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateMenuItem(ctx, menu.ID, app.UpdateMenuItemCommand{Status: &published}); err != nil {
		t.Fatal(err)
	}
	modifierGroup, err := service.CreateModifierGroup(ctx, app.CreateModifierGroupCommand{RestaurantID: restaurant.ID, Name: "Milk", Required: true, MinCount: 1, MaxCount: 2})
	if err != nil {
		t.Fatal(err)
	}
	modifierOption, err := service.CreateModifierOption(ctx, app.CreateModifierOptionCommand{RestaurantID: restaurant.ID, ModifierGroupID: modifierGroup.ID, LinkedCatalogItemID: catalog.ID, Name: "Oat milk", PriceMinor: 300})
	if err != nil {
		t.Fatal(err)
	}
	modifierBinding, err := service.CreateModifierGroupBinding(ctx, app.CreateModifierGroupBindingCommand{RestaurantID: restaurant.ID, ModifierGroupID: modifierGroup.ID, TargetType: domain.ModifierTargetMenuItem, TargetID: menu.ID, SortOrder: 7})
	if err != nil {
		t.Fatal(err)
	}

	pub, err := service.Publish(ctx, app.PublishCommand{RestaurantID: restaurant.ID, PublishedBy: "operator-1", NodeDeviceID: "node-1"})
	if err != nil {
		t.Fatal(err)
	}
	if pub.Version != 1 || pub.CloudVersion != 1 || pub.Counts["employees"] != 1 || pub.Counts["menu_items"] != 1 {
		t.Fatalf("unexpected publication summary: %+v", pub)
	}
	if pub.Counts["restaurants"] != 1 {
		t.Fatalf("expected restaurant stream to be included, got %+v", pub.Counts)
	}
	restaurantPackage, ok := repo.Package("restaurants", "node-1")
	if !ok || !strings.Contains(string(restaurantPackage.PayloadJSON), restaurant.ID) {
		t.Fatalf("expected restaurants stream package, ok=%v payload=%s", ok, restaurantPackage.PayloadJSON)
	}
	if !strings.Contains(string(restaurantPackage.PayloadJSON), `"active":true`) {
		t.Fatalf("expected active restaurant projection, payload=%s", restaurantPackage.PayloadJSON)
	}
	staffPackage, ok := repo.Package("staff", "node-1")
	if !ok {
		t.Fatal("expected staff stream package to be generated")
	}
	var staff struct {
		Employees []domain.EdgeEmployee `json:"employees"`
	}
	if err := json.Unmarshal(staffPackage.PayloadJSON, &staff); err != nil {
		t.Fatal(err)
	}
	if len(staff.Employees) != 1 || staff.Employees[0].ID != employee.ID || staff.Employees[0].PINHash == "" {
		t.Fatalf("unexpected staff package: %+v", staff)
	}
	menuPackage, ok := repo.Package("menu", "node-1")
	if !ok || !strings.Contains(string(menuPackage.PayloadJSON), `"active":true`) {
		t.Fatalf("expected active menu package, ok=%v payload=%s", ok, menuPackage.PayloadJSON)
	}
	catalogPackage, ok := repo.Package("catalog", "node-1")
	if !ok {
		t.Fatal("expected catalog stream package to be generated")
	}
	if strings.Contains(string(catalogPackage.PayloadJSON), `"categories"`) {
		t.Fatalf("catalog stream must not publish unsupported categories payload: %s", catalogPackage.PayloadJSON)
	}
	assertPOSEdgeCatalogReferencePackage(t, catalogPackage.PayloadJSON, restaurant.ID, folderParameter.ID, folder.ID, tag.ID, catalog.ID)
	fullPackage, err := service.GetCurrentPublishedPackage(ctx, restaurant.ID, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	fullBody, err := json.Marshal(fullPackage)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(fullBody), `"categories"`) {
		t.Fatalf("full package must not publish unsupported categories payload: %s", fullBody)
	}
	assertPOSEdgeCatalogReferencePackage(t, fullBody, restaurant.ID, folderParameter.ID, folder.ID, tag.ID, catalog.ID)
	assertPOSEdgeModifierIngestPackage(t, fullBody, restaurant.ID, menu.ID, modifierGroup.ID, modifierOption.ID, modifierBinding.ID)
}

func TestTenantCatalogItemFeedsIndependentRestaurantMenus(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	first, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Expo A", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Expo B", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{Kind: domain.CatalogItemService, Name: "Entrance ticket", SKU: "TICKET-GA", BaseUnit: "service"})
	if err != nil {
		t.Fatal(err)
	}
	firstCategory, err := service.CreateCategory(ctx, app.CreateCategoryCommand{RestaurantID: first.ID, Name: "Tickets A", SortOrder: 1})
	if err != nil {
		t.Fatal(err)
	}
	secondCategory, err := service.CreateCategory(ctx, app.CreateCategoryCommand{RestaurantID: second.ID, Name: "Tickets B", SortOrder: 1})
	if err != nil {
		t.Fatal(err)
	}
	firstMenu, err := service.CreateMenuItem(ctx, app.CreateMenuItemCommand{RestaurantID: first.ID, CatalogItemID: catalog.ID, CategoryID: firstCategory.ID, TagID: "tag-vip", TaxProfileID: "tax-a", Name: "Adult ticket", Price: 100000, Currency: "RUB", RuntimeStatus: "available"})
	if err != nil {
		t.Fatal(err)
	}
	secondMenu, err := service.CreateMenuItem(ctx, app.CreateMenuItemCommand{RestaurantID: second.ID, CatalogItemID: catalog.ID, CategoryID: secondCategory.ID, TagID: "tag-regular", TaxProfileID: "tax-b", Name: "Guest pass", Price: 75000, Currency: "RUB", RuntimeStatus: "available"})
	if err != nil {
		t.Fatal(err)
	}
	hidden := "hidden"
	updatedFirst, err := service.UpdateMenuItem(ctx, firstMenu.ID, app.UpdateMenuItemCommand{Name: "Adult ticket online", Price: ptrInt64(120000), RuntimeStatus: &hidden})
	if err != nil {
		t.Fatal(err)
	}
	if updatedFirst.Name != "Adult ticket online" || updatedFirst.Price != 120000 || updatedFirst.RuntimeStatus != "hidden" {
		t.Fatalf("first menu override was not updated: %+v", updatedFirst)
	}
	secondAfter, err := service.GetMenuItem(ctx, secondMenu.ID)
	if err != nil {
		t.Fatal(err)
	}
	if secondAfter.Name != "Guest pass" || secondAfter.Price != 75000 || secondAfter.CategoryID != secondCategory.ID || secondAfter.CatalogItemID != catalog.ID {
		t.Fatalf("second restaurant menu was changed by first override: %+v", secondAfter)
	}
	if _, err := service.Publish(ctx, app.PublishCommand{RestaurantID: first.ID, PublishedBy: "operator-1", NodeDeviceID: "node-a"}); err != nil {
		t.Fatal(err)
	}
	firstPackage, err := service.GetCurrentPublishedPackage(ctx, first.ID, "node-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPackage.MenuItems) != 1 || firstPackage.MenuItems[0].ID != firstMenu.ID || firstPackage.MenuItems[0].CatalogItemID != catalog.ID || firstPackage.MenuItems[0].CategoryID != firstCategory.ID || firstPackage.MenuItems[0].Name != "Adult ticket online" || firstPackage.MenuItems[0].Price != 120000 || firstPackage.MenuItems[0].Active {
		t.Fatalf("first Edge package must contain only first restaurant-effective menu, got %+v", firstPackage.MenuItems)
	}
	if _, err := service.Publish(ctx, app.PublishCommand{RestaurantID: second.ID, PublishedBy: "operator-1", NodeDeviceID: "node-b"}); err != nil {
		t.Fatal(err)
	}
	secondPackage, err := service.GetCurrentPublishedPackage(ctx, second.ID, "node-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPackage.MenuItems) != 1 || secondPackage.MenuItems[0].ID != secondMenu.ID || secondPackage.MenuItems[0].CatalogItemID != catalog.ID || secondPackage.MenuItems[0].CategoryID != secondCategory.ID || secondPackage.MenuItems[0].Name != "Guest pass" || secondPackage.MenuItems[0].Price != 75000 || !secondPackage.MenuItems[0].Active {
		t.Fatalf("second Edge package must contain only second restaurant-effective menu, got %+v", secondPackage.MenuItems)
	}
}

func TestMenuCategoryLifecycle(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Expo", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListCategories(ctx, ""); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected restaurant_id validation, got %v", err)
	}
	category, err := service.CreateCategory(ctx, app.CreateCategoryCommand{RestaurantID: restaurant.ID, Name: "Tickets", SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	listed, err := service.ListCategories(ctx, restaurant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != category.ID {
		t.Fatalf("expected created category in list, got %+v", listed)
	}
	nextOrder := int64(20)
	published := domain.StatusPublished
	updated, err := service.UpdateCategory(ctx, category.ID, app.UpdateCategoryCommand{Name: "Main tickets", SortOrder: &nextOrder, Status: &published})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Main tickets" || updated.SortOrder != 20 || updated.Status != domain.StatusPublished {
		t.Fatalf("unexpected updated category: %+v", updated)
	}
	archived, err := service.ArchiveCategory(ctx, category.ID)
	if err != nil {
		t.Fatal(err)
	}
	if archived.Status != domain.StatusArchived {
		t.Fatalf("expected archived category, got %+v", archived)
	}
}

func TestModifierOptionLinkedCatalogItemValidation(t *testing.T) {
	tests := []struct {
		name    string
		link    func(mainItem domain.CatalogItem, otherItem domain.CatalogItem) string
		wantErr bool
	}{
		{name: "nullable link is accepted", link: func(domain.CatalogItem, domain.CatalogItem) string { return "" }},
		{name: "same restaurant item is accepted", link: func(mainItem domain.CatalogItem, _ domain.CatalogItem) string { return mainItem.ID }},
		{name: "unknown item is rejected", link: func(domain.CatalogItem, domain.CatalogItem) string { return "missing-item" }, wantErr: true},
		{name: "tenant catalog item used by another restaurant is accepted", link: func(_ domain.CatalogItem, otherItem domain.CatalogItem) string { return otherItem.ID }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := newService()
			ctx := context.Background()
			restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo", Timezone: "Europe/Moscow", Currency: "RUB"})
			if err != nil {
				t.Fatal(err)
			}
			otherRestaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Other", Timezone: "Europe/Moscow", Currency: "RUB"})
			if err != nil {
				t.Fatal(err)
			}
			group, err := service.CreateModifierGroup(ctx, app.CreateModifierGroupCommand{RestaurantID: restaurant.ID, Name: "Sauce", MaxCount: 1})
			if err != nil {
				t.Fatal(err)
			}
			item, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemGood, Name: "Sauce pack", SKU: "SAUCE", BaseUnit: "pc"})
			if err != nil {
				t.Fatal(err)
			}
			otherItem, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: otherRestaurant.ID, Kind: domain.CatalogItemGood, Name: "Other sauce", SKU: "OTHER", BaseUnit: "pc"})
			if err != nil {
				t.Fatal(err)
			}
			link := tt.link(item, otherItem)
			got, err := service.CreateModifierOption(ctx, app.CreateModifierOptionCommand{RestaurantID: restaurant.ID, ModifierGroupID: group.ID, LinkedCatalogItemID: link, Name: "Extra sauce", PriceMinor: 0})
			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalid) && !errors.Is(err, domain.ErrNotFound) {
					t.Fatalf("expected validation error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected modifier option to be accepted, got %v", err)
			}
			if strings.TrimSpace(link) != got.LinkedCatalogItemID {
				t.Fatalf("unexpected linked catalog item id: %+v", got)
			}
		})
	}
}

func TestCatalogItemKindServiceAndSemiFinishedRoundTripAndPublication(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo Bistro", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	item, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{
		RestaurantID:       restaurant.ID,
		Kind:               domain.CatalogItemService,
		Name:               "Delivery",
		SKU:                "DELIVERY",
		BaseUnit:           "service",
		AccountingCategory: "services",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Kind != domain.CatalogItemService || item.AccountingCategory != "services" {
		t.Fatalf("expected canonical service kind after create, got %+v", item)
	}
	got, err := service.GetCatalogItem(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := service.ListCatalogItems(ctx, restaurant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != domain.CatalogItemService || len(listed) != 1 || listed[0].Kind != domain.CatalogItemService {
		t.Fatalf("expected service round-trip through get/list, got=%+v listed=%+v", got, listed)
	}
	semiFinished := domain.CatalogItemSemiFinished
	kitchenType := "cold"
	updated, err := service.UpdateCatalogItem(ctx, item.ID, app.UpdateCatalogItemCommand{Kind: &semiFinished, KitchenType: &kitchenType})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Kind != domain.CatalogItemSemiFinished || updated.KitchenType != "cold" {
		t.Fatalf("expected update to keep canonical semi_finished kind and kitchen type, got %+v", updated)
	}
	if _, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemKind("raw_material"), Name: "Flour", SKU: "FLOUR", BaseUnit: "kg"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected raw_material to be rejected, got %v", err)
	}
	if _, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemKind("ingredient"), Name: "Potato", SKU: "POTATO", BaseUnit: "kg"}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected legacy ingredient to be rejected, got %v", err)
	}
	if _, err := service.Publish(ctx, app.PublishCommand{RestaurantID: restaurant.ID, PublishedBy: "operator-1", NodeDeviceID: "node-1"}); err != nil {
		t.Fatal(err)
	}
	catalogPackage, ok := repo.Package("catalog", "node-1")
	if !ok {
		t.Fatal("expected catalog stream package to be generated")
	}
	payload := string(catalogPackage.PayloadJSON)
	if !strings.Contains(payload, `"type":"semi_finished"`) || !strings.Contains(payload, `"kitchen_type":"cold"`) || strings.Contains(payload, "raw_material") || strings.Contains(payload, `"type":"ingredient"`) {
		t.Fatalf("expected published Edge catalog type to use canonical v2 enum, payload=%s", catalogPackage.PayloadJSON)
	}
}

func TestPublicationIncludesRecipesAndStopListPackages(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo Bistro", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	dish, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemDish, Name: "Soup", SKU: "SOUP", BaseUnit: "portion"})
	if err != nil {
		t.Fatal(err)
	}
	component, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemGood, Name: "Potato", SKU: "POTATO", BaseUnit: "g"})
	if err != nil {
		t.Fatal(err)
	}
	published := domain.StatusPublished
	if _, err := service.UpdateCatalogItem(ctx, dish.ID, app.UpdateCatalogItemCommand{Status: &published}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateCatalogItem(ctx, component.ID, app.UpdateCatalogItemCommand{Status: &published}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRecipeItem(ctx, app.CreateRecipeItemCommand{RestaurantID: restaurant.ID, RecipeOwnerCatalogItemID: dish.ID, ComponentCatalogItemID: component.ID, Quantity: 150, Unit: "g"}); err != nil {
		t.Fatal(err)
	}
	zero := 0.0
	if _, err := service.UpsertStopListEntry(ctx, app.UpsertStopListEntryCommand{RestaurantID: restaurant.ID, CatalogItemID: component.ID, AvailableQuantity: &zero, Reason: "ingredient unavailable"}); err != nil {
		t.Fatal(err)
	}
	pub, err := service.Publish(ctx, app.PublishCommand{RestaurantID: restaurant.ID, PublishedBy: "operator-1", NodeDeviceID: "node-1"})
	if err != nil {
		t.Fatal(err)
	}
	if pub.Counts["recipe_versions"] != 1 || pub.Counts["recipe_lines"] != 1 || pub.Counts["stop_lists"] != 1 {
		t.Fatalf("expected recipes and stop-list counts, got %+v", pub.Counts)
	}
	recipesPackage, ok := repo.Package("recipes", "node-1")
	if !ok {
		t.Fatal("expected recipes stream package")
	}
	if !strings.Contains(string(recipesPackage.PayloadJSON), `"recipe_versions"`) || !strings.Contains(string(recipesPackage.PayloadJSON), component.ID) {
		t.Fatalf("expected recipes package to include component, payload=%s", recipesPackage.PayloadJSON)
	}
	inventoryPackage, ok := repo.Package("inventory_reference", "node-1")
	if !ok {
		t.Fatal("expected inventory_reference stream package")
	}
	if !strings.Contains(string(inventoryPackage.PayloadJSON), `"stop_lists"`) || !strings.Contains(string(inventoryPackage.PayloadJSON), `"available_quantity":0`) {
		t.Fatalf("expected inventory_reference package to include blocking stop-list, payload=%s", inventoryPackage.PayloadJSON)
	}
	assertPOSEdgeRecipesAndInventoryReferencePackage(t, recipesPackage.PayloadJSON, inventoryPackage.PayloadJSON, restaurant.ID, dish.ID, component.ID)
}

func TestRecipeVersionDraftSubmitApprovePublishesActiveVersion(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Recipe Lab", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	repo.AssignEdgeNodeForTest(restaurant.ID, "node-recipe")
	dish, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemDish, Name: "Soup", SKU: "SOUP-DRAFT", BaseUnit: "portion"})
	if err != nil {
		t.Fatal(err)
	}
	component, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemGood, Name: "Potato", SKU: "POTATO-DRAFT", BaseUnit: "g"})
	if err != nil {
		t.Fatal(err)
	}
	published := domain.StatusPublished
	if _, err := service.UpdateCatalogItem(ctx, dish.ID, app.UpdateCatalogItemCommand{Status: &published}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateCatalogItem(ctx, component.ID, app.UpdateCatalogItemCommand{Status: &published}); err != nil {
		t.Fatal(err)
	}

	draft, err := service.CreateRecipeVersionDraft(ctx, app.CreateRecipeVersionDraftCommand{
		RestaurantID:        restaurant.ID,
		OwnerCatalogItemID:  dish.ID,
		Name:                "Soup pilot v1",
		YieldQuantity:       1,
		YieldUnit:           "portion",
		CreatedByEmployeeID: "manager-1",
		SubmitForReview:     true,
		Reason:              "pilot recipe",
		Lines: []app.RecipeVersionLineCommand{{
			ComponentCatalogItemID: component.ID,
			Quantity:               120,
			Unit:                   "g",
			LossPercent:            3,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Version.Status != domain.RecipeVersionStatusReviewPending || len(draft.Lines) != 1 {
		t.Fatalf("expected submitted draft with one line, got %+v", draft)
	}
	suggestions, err := service.ListRecipeSuggestions(ctx, restaurant.ID, "pending", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 || suggestions[0].RecipeVersionID != draft.Version.ID {
		t.Fatalf("expected pending recipe version suggestion, got %+v", suggestions)
	}
	approved, err := service.ApproveRecipeSuggestion(ctx, suggestions[0].ID, app.SuggestionReviewCommand{ReviewedByEmployeeID: "manager-1", PublishedBy: "cloud-ui"})
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != domain.SuggestionStatusApproved {
		t.Fatalf("expected approved suggestion, got %+v", approved)
	}
	versions, err := service.ListRecipeVersions(ctx, restaurant.ID, dish.ID, "active", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0].Version.ID != draft.Version.ID {
		t.Fatalf("expected approved draft to become active version, got %+v", versions)
	}
	pub, err := repo.GetCurrentPublication(ctx, restaurant.ID)
	if err != nil {
		t.Fatal(err)
	}
	var packet posEdgeMasterDataCommand
	if err := json.Unmarshal(pub.PackageJSON, &packet); err != nil {
		t.Fatal(err)
	}
	if len(packet.RecipeVersions) != 1 || len(packet.RecipeLines) != 1 || packet.RecipeVersions[0].ID != draft.Version.ID {
		t.Fatalf("expected publication to use versioned recipe authority, versions=%+v lines=%+v", packet.RecipeVersions, packet.RecipeLines)
	}
}

func TestRecipeVersionDraftRejectsInvalidLine(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Recipe Lab", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	dish, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemDish, Name: "Soup", SKU: "SOUP-INVALID", BaseUnit: "portion"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateRecipeVersionDraft(ctx, app.CreateRecipeVersionDraftCommand{
		RestaurantID:       restaurant.ID,
		OwnerCatalogItemID: dish.ID,
		Lines: []app.RecipeVersionLineCommand{{
			ComponentCatalogItemID: dish.ID,
			Quantity:               0,
			Unit:                   "",
		}},
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid recipe line, got %v", err)
	}
}

func TestCatalogActiveSKUCanBeReusedAfterArchive(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	item, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: "restaurant-1", Type: domain.CatalogItemGood, Name: "Tea", SKU: "TEA", BaseUnit: "pcs"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: "restaurant-1", Type: domain.CatalogItemGood, Name: "Tea 2", SKU: "TEA", BaseUnit: "pcs"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected active SKU conflict, got %v", err)
	}
	if _, err := service.ArchiveCatalogItem(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: "restaurant-1", Type: domain.CatalogItemGood, Name: "Tea 2", SKU: "TEA", BaseUnit: "pcs"}); err != nil {
		t.Fatalf("expected SKU reuse after archive, got %v", err)
	}
}

func TestPublicationVersioningIsMonotonic(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	if _, err := service.Publish(ctx, app.PublishCommand{RestaurantID: "restaurant-empty", PublishedBy: "operator-1"}); err != nil {
		t.Fatal(err)
	}
	second, err := service.Publish(ctx, app.PublishCommand{RestaurantID: "restaurant-empty", PublishedBy: "operator-1"})
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 2 || second.CloudVersion != 2 {
		t.Fatalf("expected second publication version=2, got %+v", second)
	}
}

func TestStopListUpdateReviewApprovePublishesCloudAuthorityWithoutRawPayload(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	quantity := 0.0
	repo.AssignEdgeNodeForTest("restaurant-1", "edge-1")
	repo.SeedStopListUpdateReview(domain.StopListUpdateReview{
		ID:                "event-stop-1",
		RestaurantID:      "restaurant-1",
		DeviceID:          "edge-1",
		StopListID:        "edge-stop-1",
		CatalogItemID:     "dish-1",
		AvailableQuantity: &quantity,
		Active:            true,
		ConflictPolicy:    "edge_overlay_requires_manager_review",
		Source:            "edge",
		Reason:            "sold out",
		ProjectionAction:  "requires_manager_review",
		Status:            domain.SuggestionStatusPending,
		UpdatedAt:         now,
		OccurredAt:        now.Add(-time.Minute),
		ProjectedAt:       now,
		CreatedAt:         now,
	})

	items, err := service.ListStopListUpdateReviews(ctx, "restaurant-1", "pending", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(items)
	if len(items) != 1 || strings.Contains(string(body), "raw_payload") || strings.Contains(string(body), "payload_json") {
		t.Fatalf("expected one safe stop-list review row without raw payload, got %s", body)
	}
	approved, err := service.ApproveStopListUpdateReview(ctx, "event-stop-1", app.SuggestionReviewCommand{ReviewedByEmployeeID: "manager-1", PublishedBy: "cloud-ui"})
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != domain.SuggestionStatusApproved || approved.AppliedStopListID != "edge-stop-1" {
		t.Fatalf("unexpected approved row: %+v", approved)
	}
	if _, err := service.ApproveStopListUpdateReview(ctx, "event-stop-1", app.SuggestionReviewCommand{ReviewedByEmployeeID: "manager-1"}); err != nil {
		t.Fatalf("approve must be idempotent, got %v", err)
	}
	stopLists, err := repo.ListStopListEntries(ctx, "restaurant-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(stopLists) != 1 || stopLists[0].CatalogItemID != "dish-1" || stopLists[0].Source != "edge_review" || !stopLists[0].Active {
		t.Fatalf("expected approved Edge update to become Cloud authority stop-list, got %+v", stopLists)
	}
	pub, err := repo.GetCurrentPublication(ctx, "restaurant-1")
	if err != nil {
		t.Fatal(err)
	}
	if pub.Version != 1 {
		t.Fatalf("expected approval to publish new package, got %+v", pub)
	}
}

func TestCatalogSuggestionApproveAcceptsCurrentCreateAction(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Suggestion Lab", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	repo.AssignEdgeNodeForTest(restaurant.ID, "node-suggestion")
	payload := json.RawMessage(`{"data":{"action":"create","kind":"good","name":"Smoke Herb","sku":"SMOKE-HERB","base_unit":"g","kitchen_type":"hot","accounting_category":"good"}}`)
	repo.SeedCatalogSuggestion(domain.CatalogSuggestion{
		ID:              "catalog-suggestion-1",
		SuggestionID:    "edge-catalog-suggestion-1",
		RestaurantID:    restaurant.ID,
		ProposalGroupID: "proposal-group-1",
		Action:          "create",
		Reason:          "smoke catalog proposal",
		Status:          domain.SuggestionStatusPending,
		SourceEventID:   "edge-event-1",
		SuggestedAt:     now.Add(-time.Minute),
		CloudReceivedAt: now,
		PayloadJSON:     payload,
		CreatedAt:       now,
		UpdatedAt:       now,
	})

	approved, err := service.ApproveCatalogSuggestion(ctx, "catalog-suggestion-1", app.SuggestionReviewCommand{ReviewedByEmployeeID: "manager-1", PublishedBy: "cloud-ui"})
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != domain.SuggestionStatusApproved || approved.AppliedCatalogItemID == "" {
		t.Fatalf("unexpected approved suggestion: %+v", approved)
	}
	items, err := repo.ListCatalogItems(ctx, restaurant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Kind != domain.CatalogItemGood || items[0].Name != "Smoke Herb" || items[0].KitchenType != "hot" {
		t.Fatalf("expected current create action to create suggested catalog item, got %+v", items)
	}
	if _, err := repo.GetCurrentPublication(ctx, restaurant.ID); err != nil {
		t.Fatalf("approval must publish updated package, got %v", err)
	}
}

func TestStopListUpdateReviewRejectRequestChangesAndInvalidTransition(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	repo.SeedStopListUpdateReview(domain.StopListUpdateReview{
		ID:               "event-stop-2",
		RestaurantID:     "restaurant-1",
		DeviceID:         "edge-1",
		StopListID:       "edge-stop-2",
		CatalogItemID:    "dish-2",
		Active:           true,
		ConflictPolicy:   "edge_overlay_requires_manager_review",
		Source:           "edge",
		ProjectionAction: "requires_manager_review",
		Status:           domain.SuggestionStatusPending,
		UpdatedAt:        now,
		OccurredAt:       now,
		ProjectedAt:      now,
		CreatedAt:        now,
	})

	rejected, err := service.RejectStopListUpdateReview(ctx, "event-stop-2", app.SuggestionReviewCommand{ReviewedByEmployeeID: "manager-1", ReviewComment: "not approved"})
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != domain.SuggestionStatusRejected {
		t.Fatalf("expected rejected status, got %+v", rejected)
	}
	if _, err := service.RejectStopListUpdateReview(ctx, "event-stop-2", app.SuggestionReviewCommand{ReviewedByEmployeeID: "manager-1"}); err != nil {
		t.Fatalf("reject must be idempotent, got %v", err)
	}
	if _, err := service.ApproveStopListUpdateReview(ctx, "event-stop-2", app.SuggestionReviewCommand{ReviewedByEmployeeID: "manager-1"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected invalid transition conflict, got %v", err)
	}
	if stopLists, err := repo.ListStopListEntries(ctx, "restaurant-1"); err != nil || len(stopLists) != 0 {
		t.Fatalf("reject must not mutate Cloud authority, rows=%+v err=%v", stopLists, err)
	}

	repo.SeedStopListUpdateReview(domain.StopListUpdateReview{
		ID:               "event-stop-3",
		RestaurantID:     "restaurant-1",
		DeviceID:         "edge-1",
		StopListID:       "edge-stop-3",
		CatalogItemID:    "dish-3",
		Active:           false,
		ConflictPolicy:   "edge_overlay_requires_manager_review",
		Source:           "edge",
		ProjectionAction: "requires_manager_review",
		Status:           domain.SuggestionStatusPending,
		UpdatedAt:        now,
		OccurredAt:       now,
		ProjectedAt:      now.Add(time.Second),
		CreatedAt:        now,
	})
	changes, err := service.RequestChangesStopListUpdateReview(ctx, "event-stop-3", app.SuggestionReviewCommand{ReviewedByEmployeeID: "manager-1", ReviewComment: "clarify reason"})
	if err != nil {
		t.Fatal(err)
	}
	if changes.Status != domain.SuggestionStatusChangesRequest || changes.ReviewComment != "clarify reason" {
		t.Fatalf("unexpected request changes row: %+v", changes)
	}
}

func TestReviewAssignmentCommandsAreIdempotentAndAudited(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	repo.SeedStopListUpdateReview(domain.StopListUpdateReview{
		ID:               "event-stop-assign-1",
		RestaurantID:     "restaurant-1",
		DeviceID:         "edge-1",
		StopListID:       "edge-stop-assign-1",
		CatalogItemID:    "dish-assign-1",
		Active:           true,
		ConflictPolicy:   "edge_overlay_requires_manager_review",
		Source:           "edge",
		ProjectionAction: "requires_manager_review",
		Status:           domain.SuggestionStatusPending,
		UpdatedAt:        now,
		OccurredAt:       now,
		ProjectedAt:      now,
		CreatedAt:        now,
	})

	assignCommandID := "018f0000-0000-7000-8000-000000000701"
	assigned, err := service.AssignReviewItem(ctx, "stop_list_update", "event-stop-assign-1", app.ReviewAssignCommand{
		CommandID:            assignCommandID,
		AssignedToEmployeeID: "manager-2",
		AssignedByEmployeeID: "manager-1",
		Reason:               "take ownership",
	})
	if err != nil {
		t.Fatal(err)
	}
	if assigned.AssignedToEmployeeID != "manager-2" || assigned.AssignedByEmployeeID != "manager-1" || assigned.AssignmentNote != "take ownership" {
		t.Fatalf("unexpected assignment response: %+v", assigned)
	}
	replayedAssign, err := service.AssignReviewItem(ctx, "stop_list_update", "event-stop-assign-1", app.ReviewAssignCommand{
		CommandID:            assignCommandID,
		AssignedToEmployeeID: "manager-2",
		AssignedByEmployeeID: "manager-1",
		Reason:               "take ownership",
	})
	if err != nil {
		t.Fatalf("assign replay must be idempotent, got %v", err)
	}
	if replayedAssign.AssignedToEmployeeID != "manager-2" || len(repo.ReviewAssignmentAuditEvents()) != 1 {
		t.Fatalf("expected replay without duplicate audit, response=%+v audit=%+v", replayedAssign, repo.ReviewAssignmentAuditEvents())
	}

	unassignCommandID := "018f0000-0000-7000-8000-000000000702"
	unassigned, err := service.UnassignReviewItem(ctx, "stop_list_update", "event-stop-assign-1", app.ReviewUnassignCommand{
		CommandID:              unassignCommandID,
		UnassignedByEmployeeID: "manager-1",
		Reason:                 "rebalance queue",
	})
	if err != nil {
		t.Fatal(err)
	}
	if unassigned.AssignedToEmployeeID != "" || unassigned.AssignedByEmployeeID != "" || unassigned.AssignedAt != nil || unassigned.AssignmentNote != "" {
		t.Fatalf("unexpected unassignment response: %+v", unassigned)
	}
	replayedUnassign, err := service.UnassignReviewItem(ctx, "stop_list_update", "event-stop-assign-1", app.ReviewUnassignCommand{
		CommandID:              unassignCommandID,
		UnassignedByEmployeeID: "manager-1",
		Reason:                 "rebalance queue",
	})
	if err != nil {
		t.Fatalf("unassign replay must be idempotent, got %v", err)
	}
	if replayedUnassign.AssignedToEmployeeID != "" || len(repo.ReviewAssignmentAuditEvents()) != 2 {
		t.Fatalf("expected replay without duplicate audit, response=%+v audit=%+v", replayedUnassign, repo.ReviewAssignmentAuditEvents())
	}
	body, _ := json.Marshal(unassigned)
	if strings.Contains(string(body), "payload_json") || strings.Contains(string(body), "raw_payload") {
		t.Fatalf("assignment response must not expose raw payload fields: %s", body)
	}
	events := repo.ReviewAssignmentAuditEvents()
	if events[0].Action != "assigned" || events[1].Action != "unassigned" {
		t.Fatalf("expected assigned/unassigned audit trail, got %+v", events)
	}
	if !isTestUUIDv7(events[0].EventID) || !isTestUUIDv7(events[1].EventID) {
		t.Fatalf("audit event_id must be UUIDv7, got %+v", events)
	}
	if events[0].ReviewID != "event-stop-assign-1" || events[0].ActorEmployeeID != "manager-1" || events[0].TargetEmployeeID != "manager-2" || !events[0].OccurredAt.Equal(now) {
		t.Fatalf("unexpected assign audit event: %+v", events[0])
	}
	if events[1].ActorEmployeeID != "manager-1" || events[1].TargetEmployeeID != "manager-2" || events[1].Reason != "rebalance queue" || !events[1].OccurredAt.Equal(now) {
		t.Fatalf("unexpected unassign audit event: %+v", events[1])
	}
}

func TestListStopListUpdateReviewAuditReturnsBoundedSafeEvents(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	repo.SeedStopListUpdateReview(domain.StopListUpdateReview{
		ID:               "event-stop-audit-1",
		RestaurantID:     "restaurant-1",
		DeviceID:         "edge-1",
		StopListID:       "edge-stop-audit-1",
		CatalogItemID:    "dish-audit-1",
		Active:           true,
		ConflictPolicy:   "edge_overlay_requires_manager_review",
		Source:           "edge",
		ProjectionAction: "requires_manager_review",
		Status:           domain.SuggestionStatusPending,
		UpdatedAt:        now,
		OccurredAt:       now,
		ProjectedAt:      now,
		CreatedAt:        now,
	})

	if _, err := service.AssignReviewItem(ctx, "stop_list_update", "event-stop-audit-1", app.ReviewAssignCommand{
		CommandID:            "018f0000-0000-7000-8000-000000000721",
		AssignedToEmployeeID: "manager-2",
		AssignedByEmployeeID: "manager-1",
		Reason:               "take ownership",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UnassignReviewItem(ctx, "stop_list_update", "event-stop-audit-1", app.ReviewUnassignCommand{
		CommandID:              "018f0000-0000-7000-8000-000000000722",
		UnassignedByEmployeeID: "manager-1",
		Reason:                 "rebalance queue",
	}); err != nil {
		t.Fatal(err)
	}

	events, err := service.ListStopListUpdateReviewAudit(ctx, "event-stop-audit-1", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected two audit events, got %+v", events)
	}
	actions := map[string]bool{}
	for _, event := range events {
		actions[event.Action] = true
		if event.ReviewType != "stop_list_update" || event.ReviewID != "event-stop-audit-1" || event.CommandID == "" {
			t.Fatalf("unexpected audit event: %+v", event)
		}
	}
	if !actions["assigned"] || !actions["unassigned"] {
		t.Fatalf("expected assigned and unassigned audit actions, got %+v", events)
	}
	body, _ := json.Marshal(events)
	for _, forbidden := range []string{"payload_json", "raw_payload", "sync_payload", "envelope"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("audit response must not expose %s: %s", forbidden, body)
		}
	}
}

func TestListStopListUpdateReviewAuditCapsLimitToHundred(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	for i := 0; i < 101; i++ {
		event := domain.ReviewAssignmentAuditEvent{
			EventID:          fmt.Sprintf("audit-event-%03d", i),
			CommandID:        fmt.Sprintf("audit-command-%03d", i),
			ReviewType:       "stop_list_update",
			ReviewID:         "event-stop-audit-limit",
			Action:           "assigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "bounded list",
			OccurredAt:       now.Add(time.Duration(i) * time.Second),
		}
		if err := repo.AppendReviewAssignmentAuditEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.AppendReviewAssignmentAuditEvent(ctx, domain.ReviewAssignmentAuditEvent{
		EventID:          "audit-event-other",
		CommandID:        "audit-command-other",
		ReviewType:       "stop_list_update",
		ReviewID:         "event-stop-audit-other",
		Action:           "assigned",
		ActorEmployeeID:  "manager-1",
		TargetEmployeeID: "manager-2",
		OccurredAt:       now.Add(2 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	events, err := service.ListStopListUpdateReviewAudit(ctx, "event-stop-audit-limit", 1000, -10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 100 {
		t.Fatalf("expected max 100 audit events, got %d", len(events))
	}
	if events[0].EventID != "audit-event-100" || events[99].EventID != "audit-event-001" {
		t.Fatalf("expected newest-first bounded events, got first=%+v last=%+v", events[0], events[99])
	}

	missing, err := service.ListStopListUpdateReviewAudit(ctx, "missing-review", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Fatalf("unknown review id should return a safe empty list, got %+v", missing)
	}
}

func TestListCatalogSuggestionReviewAuditReturnsBoundedSafeEvents(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	events := []domain.ReviewAssignmentAuditEvent{
		{
			EventID:          "catalog-audit-001",
			CommandID:        "catalog-command-001",
			ReviewType:       "catalog_suggestion",
			ReviewID:         "catalog-audit-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "take catalog review",
			OccurredAt:       now,
		},
		{
			EventID:          "catalog-audit-002",
			CommandID:        "catalog-command-002",
			ReviewType:       "catalog_suggestion",
			ReviewID:         "catalog-audit-1",
			Action:           "unassigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "rebalance catalog queue",
			OccurredAt:       now.Add(time.Minute),
		},
		{
			EventID:          "catalog-audit-other-type",
			CommandID:        "catalog-command-other-type",
			ReviewType:       "recipe_suggestion",
			ReviewID:         "catalog-audit-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-9",
			TargetEmployeeID: "manager-8",
			OccurredAt:       now.Add(2 * time.Minute),
		},
	}
	for _, event := range events {
		if err := repo.AppendReviewAssignmentAuditEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}

	got, err := service.ListCatalogSuggestionReviewAudit(ctx, "catalog-audit-1", 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].EventID != "catalog-audit-002" || got[0].ReviewType != "catalog_suggestion" || got[0].CommandID == "" {
		t.Fatalf("expected bounded newest catalog audit event, got %+v", got)
	}
	body, _ := json.Marshal(got)
	for _, forbidden := range []string{"payload_json", "raw_payload", "sync_payload", "envelope", "request_dump", "token", "pin", "sql"} {
		if strings.Contains(strings.ToLower(string(body)), forbidden) {
			t.Fatalf("catalog audit response must not expose %s: %s", forbidden, body)
		}
	}
}

func TestListRecipeSuggestionReviewAuditReturnsBoundedSafeEvents(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	events := []domain.ReviewAssignmentAuditEvent{
		{
			EventID:          "recipe-audit-001",
			CommandID:        "recipe-command-001",
			ReviewType:       "recipe_suggestion",
			ReviewID:         "recipe-audit-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "take recipe review",
			OccurredAt:       now,
		},
		{
			EventID:          "recipe-audit-002",
			CommandID:        "recipe-command-002",
			ReviewType:       "recipe_suggestion",
			ReviewID:         "recipe-audit-1",
			Action:           "unassigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "rebalance recipe queue",
			OccurredAt:       now.Add(time.Minute),
		},
		{
			EventID:          "recipe-audit-other-review",
			CommandID:        "recipe-command-other-review",
			ReviewType:       "recipe_suggestion",
			ReviewID:         "recipe-audit-other",
			Action:           "assigned",
			ActorEmployeeID:  "manager-9",
			TargetEmployeeID: "manager-8",
			OccurredAt:       now.Add(2 * time.Minute),
		},
	}
	for _, event := range events {
		if err := repo.AppendReviewAssignmentAuditEvent(ctx, event); err != nil {
			t.Fatal(err)
		}
	}

	got, err := service.ListRecipeSuggestionReviewAudit(ctx, "recipe-audit-1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].EventID != "recipe-audit-002" || got[1].EventID != "recipe-audit-001" {
		t.Fatalf("expected newest-first recipe audit events, got %+v", got)
	}
	for _, event := range got {
		if event.ReviewType != "recipe_suggestion" || event.ReviewID != "recipe-audit-1" || event.CommandID == "" {
			t.Fatalf("unexpected recipe audit event: %+v", event)
		}
	}
	body, _ := json.Marshal(got)
	for _, forbidden := range []string{"payload_json", "raw_payload", "sync_payload", "envelope", "request_dump", "token", "pin", "sql"} {
		if strings.Contains(strings.ToLower(string(body)), forbidden) {
			t.Fatalf("recipe audit response must not expose %s: %s", forbidden, body)
		}
	}
}

func TestListCatalogSuggestionReviewAuditCapsLimitToHundred(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	for i := 0; i < 101; i++ {
		if err := repo.AppendReviewAssignmentAuditEvent(ctx, domain.ReviewAssignmentAuditEvent{
			EventID:          fmt.Sprintf("catalog-limit-event-%03d", i),
			CommandID:        fmt.Sprintf("catalog-limit-command-%03d", i),
			ReviewType:       "catalog_suggestion",
			ReviewID:         "catalog-audit-limit",
			Action:           "assigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "bounded catalog list",
			OccurredAt:       now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := service.ListCatalogSuggestionReviewAudit(ctx, "catalog-audit-limit", 1000, -10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 100 {
		t.Fatalf("expected max 100 catalog audit events, got %d", len(got))
	}
	if got[0].EventID != "catalog-limit-event-100" || got[99].EventID != "catalog-limit-event-001" {
		t.Fatalf("expected newest-first capped catalog audit events, got first=%+v last=%+v", got[0], got[99])
	}
}

func TestReviewAssignmentRejectsTerminalAndUnknownReviewType(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	for _, tc := range []struct {
		id     string
		status domain.SuggestionStatus
	}{
		{id: "event-stop-approved", status: domain.SuggestionStatusApproved},
		{id: "event-stop-rejected", status: domain.SuggestionStatusRejected},
	} {
		repo.SeedStopListUpdateReview(domain.StopListUpdateReview{
			ID:               tc.id,
			RestaurantID:     "restaurant-1",
			DeviceID:         "edge-1",
			StopListID:       tc.id + "-stop",
			CatalogItemID:    tc.id + "-dish",
			Active:           true,
			ConflictPolicy:   "edge_overlay_requires_manager_review",
			Source:           "edge",
			ProjectionAction: "requires_manager_review",
			Status:           tc.status,
			UpdatedAt:        now,
			OccurredAt:       now,
			ProjectedAt:      now,
			CreatedAt:        now,
		})
		if _, err := service.AssignReviewItem(ctx, "stop_list_update", tc.id, app.ReviewAssignCommand{
			CommandID:            "018f0000-0000-7000-8000-000000000711",
			AssignedToEmployeeID: "manager-2",
			AssignedByEmployeeID: "manager-1",
			Reason:               "take ownership",
		}); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected terminal %s assign conflict, got %v", tc.status, err)
		}
		if _, err := service.UnassignReviewItem(ctx, "stop_list_update", tc.id, app.ReviewUnassignCommand{
			CommandID:              "018f0000-0000-7000-8000-000000000712",
			UnassignedByEmployeeID: "manager-1",
			Reason:                 "rebalance queue",
		}); !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected terminal %s unassign conflict, got %v", tc.status, err)
		}
	}
	if _, err := service.AssignReviewItem(ctx, "unknown", "event-stop-approved", app.ReviewAssignCommand{
		CommandID:            "018f0000-0000-7000-8000-000000000713",
		AssignedToEmployeeID: "manager-2",
		AssignedByEmployeeID: "manager-1",
		Reason:               "take ownership",
	}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected unknown review_type validation error, got %v", err)
	}
}

func TestReviewAssignmentRejectsCatalogAndRecipeSuggestions(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	repo.SeedCatalogSuggestion(domain.CatalogSuggestion{
		ID:              "catalog-assignment-1",
		SuggestionID:    "edge-catalog-assignment-1",
		RestaurantID:    "restaurant-1",
		Action:          "create",
		Status:          domain.SuggestionStatusPending,
		SuggestedAt:     now,
		CloudReceivedAt: now,
		PayloadJSON:     json.RawMessage(`{"data":{"name":"Tea"}}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if _, err := repo.SubmitRecipeSuggestion(ctx, domain.RecipeSuggestion{
		ID:              "recipe-assignment-1",
		SuggestionID:    "edge-recipe-assignment-1",
		RestaurantID:    "restaurant-1",
		Action:          "update",
		Status:          domain.SuggestionStatusPending,
		SuggestedAt:     now,
		CloudReceivedAt: now,
		PayloadJSON:     json.RawMessage(`{"data":{"action":"update"}}`),
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		reviewType string
		id         string
		commandID  string
	}{
		{reviewType: "catalog_suggestion", id: "catalog-assignment-1", commandID: "018f0000-0000-7000-8000-000000000721"},
		{reviewType: "recipe_suggestion", id: "recipe-assignment-1", commandID: "018f0000-0000-7000-8000-000000000722"},
	} {
		_, err := service.AssignReviewItem(ctx, tc.reviewType, tc.id, app.ReviewAssignCommand{
			CommandID:            tc.commandID,
			AssignedToEmployeeID: "manager-2",
			AssignedByEmployeeID: "manager-1",
			Reason:               "take ownership",
		})
		if !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("expected %s assignment to stay unsupported, got %v", tc.reviewType, err)
		}
	}
}

func TestStopListReviewDecisionPreservesAssignment(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	now := fixedClock{}.Now()
	repo.SeedStopListUpdateReview(domain.StopListUpdateReview{
		ID:                   "event-stop-preserve-assignment",
		RestaurantID:         "restaurant-1",
		DeviceID:             "edge-1",
		StopListID:           "edge-stop-preserve-assignment",
		CatalogItemID:        "dish-preserve-assignment",
		Active:               true,
		ConflictPolicy:       "edge_overlay_requires_manager_review",
		Source:               "edge",
		ProjectionAction:     "requires_manager_review",
		Status:               domain.SuggestionStatusPending,
		AssignedToEmployeeID: "manager-2",
		AssignedByEmployeeID: "manager-1",
		AssignedAt:           &now,
		AssignmentNote:       "take ownership",
		UpdatedAt:            now,
		OccurredAt:           now,
		ProjectedAt:          now,
		CreatedAt:            now,
	})
	reviewed, err := service.RequestChangesStopListUpdateReview(ctx, "event-stop-preserve-assignment", app.SuggestionReviewCommand{
		ReviewedByEmployeeID: "manager-2",
		ReviewComment:        "need details",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.AssignedToEmployeeID != "manager-2" || reviewed.AssignedByEmployeeID != "manager-1" || reviewed.AssignedAt == nil || reviewed.AssignmentNote != "take ownership" {
		t.Fatalf("stop-list review decision must preserve assignment fields, got %+v", reviewed)
	}
}

func isTestUUIDv7(v string) bool {
	if len(v) != 36 || v[14] != '7' {
		return false
	}
	variant := v[19]
	return variant == '8' || variant == '9' || variant == 'a' || variant == 'A' || variant == 'b' || variant == 'B'
}

func newService() (*app.Service, *memory.Repository) {
	repo := memory.NewRepository()
	now := fixedClock{}.Now()
	_, _ = repo.CreateRestaurant(context.Background(), domain.Restaurant{ID: "restaurant-1", Name: "Test", Timezone: "Europe/Moscow", Currency: "RUB", BusinessDayMode: "standard", BusinessDayBoundaryLocalTime: "05:00", Status: domain.RestaurantActive, CloudVersion: 1, CreatedAt: now, UpdatedAt: now})
	return app.NewService(repo, fixedClock{}, &fixedIDs{}), repo
}

type posEdgeMasterDataCommand struct {
	NodeDeviceID           string                         `json:"node_device_id,omitempty"`
	RestaurantID           string                         `json:"restaurant_id,omitempty"`
	SyncMode               string                         `json:"sync_mode,omitempty"`
	CheckpointToken        string                         `json:"checkpoint_token,omitempty"`
	CloudVersion           int64                          `json:"cloud_version,omitempty"`
	CloudUpdatedAt         string                         `json:"cloud_updated_at,omitempty"`
	Restaurants            []json.RawMessage              `json:"restaurants,omitempty"`
	Roles                  []json.RawMessage              `json:"roles,omitempty"`
	Employees              []json.RawMessage              `json:"employees,omitempty"`
	Halls                  []json.RawMessage              `json:"halls,omitempty"`
	Tables                 []json.RawMessage              `json:"tables,omitempty"`
	RestaurantSections     []json.RawMessage              `json:"restaurant_sections,omitempty"`
	SalesPoints            []json.RawMessage              `json:"sales_points,omitempty"`
	CatalogItems           []json.RawMessage              `json:"catalog_items,omitempty"`
	Folders                []json.RawMessage              `json:"folders,omitempty"`
	FolderParameters       []posEdgeFolderParameter       `json:"folder_parameters,omitempty"`
	Tags                   []posEdgeCatalogTag            `json:"tags,omitempty"`
	ItemTags               []posEdgeCatalogItemTag        `json:"item_tags,omitempty"`
	ModifierGroups         []posEdgeModifierGroup         `json:"modifier_groups,omitempty"`
	ModifierOptions        []posEdgeModifierOption        `json:"modifier_options,omitempty"`
	ModifierBindings       []posEdgeModifierGroupBinding  `json:"modifier_bindings,omitempty"`
	MenuItemModifierGroups []posEdgeMenuItemModifierGroup `json:"menu_item_modifier_groups,omitempty"`
	MenuItems              []posEdgeMenuItem              `json:"menu_items,omitempty"`
	TaxProfiles            []json.RawMessage              `json:"tax_profiles,omitempty"`
	TaxRules               []json.RawMessage              `json:"tax_rules,omitempty"`
	ServiceChargeRules     []json.RawMessage              `json:"service_charge_rules,omitempty"`
	PricingPolicies        []json.RawMessage              `json:"pricing_policies,omitempty"`
	RecipeVersions         []posEdgeRecipeVersion         `json:"recipe_versions,omitempty"`
	RecipeLines            []posEdgeRecipeLine            `json:"recipe_lines,omitempty"`
	StopLists              []posEdgeStopListEntry         `json:"stop_lists,omitempty"`
	Warehouses             []posEdgeWarehouseReference    `json:"warehouses,omitempty"`
}

type posEdgeModifierGroup struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Required     bool   `json:"required"`
	MinCount     int    `json:"min_count"`
	MaxCount     int    `json:"max_count"`
	Active       bool   `json:"active"`
}

type posEdgeModifierOption struct {
	ID                  string `json:"id"`
	RestaurantID        string `json:"restaurant_id"`
	ModifierGroupID     string `json:"modifier_group_id"`
	LinkedCatalogItemID string `json:"linked_catalog_item_id,omitempty"`
	Name                string `json:"name"`
	PriceMinor          int64  `json:"price_minor"`
	Active              bool   `json:"active"`
}

type posEdgeModifierGroupBinding struct {
	ID              string `json:"id"`
	RestaurantID    string `json:"restaurant_id"`
	ModifierGroupID string `json:"modifier_group_id"`
	TargetType      string `json:"target_type"`
	TargetID        string `json:"target_id"`
	SortOrder       int    `json:"sort_order"`
	Active          bool   `json:"active"`
}

type posEdgeMenuItemModifierGroup struct {
	MenuItemID      string `json:"menu_item_id"`
	ModifierGroupID string `json:"modifier_group_id"`
	SortOrder       int    `json:"sort_order"`
}

type posEdgeMenuItem struct {
	ID            string `json:"id"`
	CatalogItemID string `json:"catalog_item_id"`
	CategoryID    string `json:"category_id,omitempty"`
	TagID         string `json:"tag_id,omitempty"`
	Name          string `json:"name"`
	Price         int64  `json:"price"`
	Currency      string `json:"currency"`
	TaxProfileID  string `json:"tax_profile_id,omitempty"`
	RuntimeStatus string `json:"runtime_status,omitempty"`
	Active        bool   `json:"active"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type posEdgeRecipeVersion struct {
	ID                string `json:"id"`
	DishCatalogItemID string `json:"dish_catalog_item_id"`
	Version           int    `json:"version"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	YieldQuantity     int64  `json:"yield_quantity"`
	YieldUnit         string `json:"yield_unit"`
	Active            bool   `json:"active"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type posEdgeRecipeLine struct {
	ID              string `json:"id"`
	RecipeVersionID string `json:"recipe_version_id"`
	CatalogItemID   string `json:"catalog_item_id"`
	Quantity        int64  `json:"quantity"`
	Unit            string `json:"unit"`
	LossPercent     int    `json:"loss_percent"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type posEdgeStopListEntry struct {
	ID                string   `json:"id"`
	RestaurantID      string   `json:"restaurant_id"`
	CatalogItemID     string   `json:"catalog_item_id"`
	AvailableQuantity *float64 `json:"available_quantity,omitempty"`
	Source            string   `json:"source"`
	Reason            string   `json:"reason,omitempty"`
	Active            bool     `json:"active"`
	UpdatedAt         string   `json:"updated_at"`
}

type posEdgeWarehouseReference struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Default      bool   `json:"is_default"`
	Active       bool   `json:"active"`
	UpdatedAt    string `json:"updated_at"`
}

type posEdgeFolderParameter struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
	FolderID     string `json:"folder_id"`
	ParameterKey string `json:"parameter_key"`
	ValueType    string `json:"value_type"`
	ValueJSON    string `json:"value_json"`
	Active       bool   `json:"active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type posEdgeCatalogTag struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Active       bool   `json:"active"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type posEdgeCatalogItemTag struct {
	CatalogItemID string `json:"catalog_item_id"`
	TagID         string `json:"tag_id"`
	RestaurantID  string `json:"restaurant_id"`
}

func assertPOSEdgeCatalogReferencePackage(t *testing.T, body []byte, restaurantID, parameterID, folderID, tagID, catalogItemID string) {
	t.Helper()

	var cmd posEdgeMasterDataCommand
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cmd); err != nil {
		t.Fatalf("published package must match POS Edge strict catalog ingest shape: %v\npayload=%s", err, body)
	}
	if len(cmd.FolderParameters) != 1 {
		t.Fatalf("expected one folder parameter, got %+v", cmd.FolderParameters)
	}
	parameter := cmd.FolderParameters[0]
	if parameter.ID != parameterID || parameter.RestaurantID != restaurantID || parameter.FolderID != folderID || parameter.ParameterKey != "station" || parameter.ValueJSON != `"bar"` {
		t.Fatalf("folder parameter lost POS Edge identity fields: %+v", parameter)
	}
	if len(cmd.Tags) != 1 || cmd.Tags[0].ID != tagID || cmd.Tags[0].RestaurantID != restaurantID {
		t.Fatalf("catalog tag lost POS Edge identity fields: %+v", cmd.Tags)
	}
	if len(cmd.ItemTags) != 1 || cmd.ItemTags[0].CatalogItemID != catalogItemID || cmd.ItemTags[0].TagID != tagID || cmd.ItemTags[0].RestaurantID != restaurantID {
		t.Fatalf("catalog item tag lost POS Edge identity fields: %+v", cmd.ItemTags)
	}
}

func assertPOSEdgeModifierIngestPackage(t *testing.T, body []byte, restaurantID, menuItemID, groupID, optionID, bindingID string) {
	t.Helper()

	var cmd posEdgeMasterDataCommand
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cmd); err != nil {
		t.Fatalf("published package must match POS Edge strict master-data ingest shape: %v\npayload=%s", err, body)
	}
	if len(cmd.ModifierGroups) != 1 {
		t.Fatalf("expected one top-level modifier group, got %+v", cmd.ModifierGroups)
	}
	group := cmd.ModifierGroups[0]
	if group.ID != groupID || group.RestaurantID != restaurantID || !group.Required || group.MinCount != 1 || group.MaxCount != 2 || !group.Active {
		t.Fatalf("top-level modifier group lost POS Edge semantics: %+v", group)
	}
	if len(cmd.ModifierOptions) != 1 || cmd.ModifierOptions[0].ID != optionID || cmd.ModifierOptions[0].RestaurantID != restaurantID || cmd.ModifierOptions[0].ModifierGroupID != groupID {
		t.Fatalf("unexpected modifier options projection: %+v", cmd.ModifierOptions)
	}
	if cmd.ModifierOptions[0].LinkedCatalogItemID == "" {
		t.Fatalf("expected linked catalog item id in modifier option projection: %+v", cmd.ModifierOptions[0])
	}
	if len(cmd.ModifierBindings) != 1 || cmd.ModifierBindings[0].ID != bindingID || cmd.ModifierBindings[0].RestaurantID != restaurantID || cmd.ModifierBindings[0].ModifierGroupID != groupID {
		t.Fatalf("unexpected modifier binding projection: %+v", cmd.ModifierBindings)
	}
	if len(cmd.MenuItemModifierGroups) != 1 {
		t.Fatalf("expected one menu item modifier link, got %+v", cmd.MenuItemModifierGroups)
	}
	link := cmd.MenuItemModifierGroups[0]
	if link.MenuItemID != menuItemID || link.ModifierGroupID != groupID || link.SortOrder != 7 {
		t.Fatalf("unexpected menu item modifier link projection: %+v", link)
	}
	if len(cmd.MenuItems) != 1 || cmd.MenuItems[0].ID != menuItemID {
		t.Fatalf("unexpected menu item projection: %+v", cmd.MenuItems)
	}
}

func assertPOSEdgeRecipesAndInventoryReferencePackage(t *testing.T, recipesBody, inventoryBody []byte, restaurantID, dishCatalogItemID, componentCatalogItemID string) {
	t.Helper()

	for _, body := range [][]byte{recipesBody, inventoryBody} {
		if strings.Contains(string(body), "stock_documents") || strings.Contains(string(body), "stock_moves") || strings.Contains(string(body), "stock_balances") {
			t.Fatalf("publication must not include Edge-side stock document payload: %s", body)
		}
	}

	var recipes posEdgeMasterDataCommand
	recipesDecoder := json.NewDecoder(bytes.NewReader(recipesBody))
	recipesDecoder.DisallowUnknownFields()
	if err := recipesDecoder.Decode(&recipes); err != nil {
		t.Fatalf("recipes package must match POS Edge strict ingest shape: %v\npayload=%s", err, recipesBody)
	}
	if recipes.RestaurantID != restaurantID || len(recipes.RecipeVersions) != 1 || len(recipes.RecipeLines) != 1 {
		t.Fatalf("unexpected recipes package rows: %+v", recipes)
	}
	version := recipes.RecipeVersions[0]
	if version.DishCatalogItemID != dishCatalogItemID || version.Status != "active" || !version.Active || version.YieldQuantity != 1 || version.YieldUnit != "portion" {
		t.Fatalf("unexpected recipe version projection: %+v", version)
	}
	line := recipes.RecipeLines[0]
	if line.RecipeVersionID != version.ID || line.CatalogItemID != componentCatalogItemID || line.Quantity != 150 || line.Unit != "g" {
		t.Fatalf("unexpected recipe line projection: %+v", line)
	}

	var inventory posEdgeMasterDataCommand
	inventoryDecoder := json.NewDecoder(bytes.NewReader(inventoryBody))
	inventoryDecoder.DisallowUnknownFields()
	if err := inventoryDecoder.Decode(&inventory); err != nil {
		t.Fatalf("inventory_reference package must match POS Edge strict ingest shape: %v\npayload=%s", err, inventoryBody)
	}
	if inventory.RestaurantID != restaurantID || len(inventory.StopLists) != 1 || len(inventory.Warehouses) != 1 {
		t.Fatalf("unexpected inventory_reference package rows: %+v", inventory)
	}
	stop := inventory.StopLists[0]
	if stop.RestaurantID != restaurantID || stop.CatalogItemID != componentCatalogItemID || stop.AvailableQuantity == nil || *stop.AvailableQuantity != 0 || stop.Source != "cloud" || !stop.Active {
		t.Fatalf("unexpected stop-list projection: %+v", stop)
	}
	warehouse := inventory.Warehouses[0]
	if warehouse.ID != "warehouse-main" || warehouse.RestaurantID != restaurantID || warehouse.Kind != "kitchen" || !warehouse.Default || !warehouse.Active {
		t.Fatalf("unexpected warehouse projection: %+v", warehouse)
	}
}

func TestExistingReferenceUpdatesKeepLifecycleStatusesExact(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo Bistro", Timezone: "Europe/Moscow", Currency: "RUB", BusinessDayMode: "standard", BusinessDayBoundaryLocalTime: "04:00"})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemDish, Name: "Tea", SKU: "TEA", BaseUnit: "portion"})
	if err != nil {
		t.Fatal(err)
	}
	archived := domain.StatusArchived
	catalog, err = service.UpdateCatalogItem(ctx, catalog.ID, app.UpdateCatalogItemCommand{Status: &archived})
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Status != domain.StatusArchived || catalog.ArchivedAt == nil || catalog.CloudVersion != 2 {
		t.Fatalf("expected archived catalog item with version bump, got %+v", catalog)
	}
	published := domain.StatusPublished
	catalog, err = service.UpdateCatalogItem(ctx, catalog.ID, app.UpdateCatalogItemCommand{Name: "Black tea", Status: &published})
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Status != domain.StatusPublished || catalog.ArchivedAt != nil || catalog.Name != "Black tea" || catalog.CloudVersion != 3 {
		t.Fatalf("expected exact published status after edit, got %+v", catalog)
	}

	hall, err := service.CreateHall(ctx, app.CreateHallCommand{RestaurantID: restaurant.ID, Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	sections, err := service.ListRestaurantSections(ctx, restaurant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) == 0 {
		t.Fatal("expected bootstrap default section")
	}
	table, err := service.CreateTable(ctx, app.CreateTableCommand{RestaurantID: restaurant.ID, HallID: hall.ID, SectionID: sections[0].ID, Name: "A1", Seats: 2})
	if err != nil {
		t.Fatal(err)
	}
	draft := domain.StatusDraft
	table, err = service.UpdateTable(ctx, table.ID, app.UpdateTableCommand{Name: "A2", Seats: ptrInt64(4), Status: &draft})
	if err != nil {
		t.Fatal(err)
	}
	if table.Status != domain.StatusDraft || table.Name != "A2" || table.Seats != 4 || table.ArchivedAt != nil {
		t.Fatalf("expected table edit to keep draft status exactly, got %+v", table)
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}

func TestPricingPolicyValidationAndPublicationPayload(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreatePricingPolicy(ctx, app.CreatePricingPolicyCommand{RestaurantID: restaurant.ID, Name: "Bad surcharge", Kind: domain.PricingPolicySurcharge, Scope: "line", AmountKind: "percentage", ValueBasisPoints: 500, ApplicationIndex: 10}); err != nil {
		t.Fatalf("surcharge scope is normalized to order for pilot authoring, got %v", err)
	}
	if _, err := service.CreatePricingPolicy(ctx, app.CreatePricingPolicyCommand{RestaurantID: restaurant.ID, Name: "Bad percent", Kind: domain.PricingPolicyDiscount, Scope: "order", AmountKind: "percentage", ValueBasisPoints: 0, ApplicationIndex: 20}); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid percentage policy, got %v", err)
	}
	policy, err := service.CreatePricingPolicy(ctx, app.CreatePricingPolicyCommand{RestaurantID: restaurant.ID, Name: "Manager 5%", Kind: domain.PricingPolicyDiscount, Scope: "order", AmountKind: "percentage", ValueBasisPoints: 500, ApplicationIndex: 30, Manual: true, RequiresPermission: "pos.pricing.discount.apply"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Publish(ctx, app.PublishCommand{RestaurantID: restaurant.ID, PublishedBy: "manager"}); err != nil {
		t.Fatal(err)
	}
	pub, err := repo.GetCurrentPublication(ctx, restaurant.ID)
	if err != nil {
		t.Fatal(err)
	}
	var packet posEdgeMasterDataCommand
	if err := json.Unmarshal(pub.PackageJSON, &packet); err != nil {
		t.Fatal(err)
	}
	if len(packet.PricingPolicies) != 2 {
		t.Fatalf("expected pricing policies in full package, got %d", len(packet.PricingPolicies))
	}
	var published struct {
		ID                 string `json:"id"`
		ApplicationIndex   int    `json:"application_index"`
		Manual             bool   `json:"manual"`
		RequiresPermission string `json:"requires_permission"`
		Active             bool   `json:"active"`
	}
	if err := json.Unmarshal(packet.PricingPolicies[1], &published); err != nil {
		t.Fatal(err)
	}
	if published.ID != policy.ID || published.ApplicationIndex != 30 || !published.Manual || published.RequiresPermission != "pos.pricing.discount.apply" || !published.Active {
		t.Fatalf("unexpected pricing policy payload: %+v", published)
	}
}

func TestAutomaticDeliveryCreatesLatestRowsForAssignedEdgesAndReplayIsIdempotent(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	item, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemGood, Name: "Tea", SKU: "TEA", BaseUnit: "pcs"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetCurrentPublishedState(ctx, restaurant.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("CRUD before assignment must not create publication, got %v", err)
	}

	repo.AssignEdgeNodeForTest(restaurant.ID, "edge-a")
	repo.AssignEdgeNodeForTest(restaurant.ID, "edge-b")
	first, err := service.RefreshDeliveryPackages(ctx, restaurant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 || len(first.Deliveries) != 2 {
		t.Fatalf("expected first full batch for two Edge nodes, got %+v", first)
	}
	for _, nodeID := range []string{"edge-a", "edge-b"} {
		if _, ok := repo.Package("catalog", nodeID); !ok {
			t.Fatalf("expected latest catalog row for %s", nodeID)
		}
	}
	if _, err := service.RefreshDeliveryPackagesForNode(ctx, restaurant.ID, "edge-a"); err != nil {
		t.Fatal(err)
	}
	replayed, err := service.GetCurrentPublishedState(ctx, restaurant.ID)
	if err != nil || replayed.Version != first.Version {
		t.Fatalf("unchanged replay must keep publication version, got %+v err=%v", replayed, err)
	}

	name := "Black tea"
	if _, err := service.UpdateCatalogItem(ctx, item.ID, app.UpdateCatalogItemCommand{Name: name}); err != nil {
		t.Fatal(err)
	}
	updated, err := service.GetCurrentPublishedState(ctx, restaurant.ID)
	if err != nil || updated.Version != 2 || len(updated.Deliveries) != 2 {
		t.Fatalf("effective commit must update both Edge nodes once, got %+v err=%v", updated, err)
	}
}

func TestAutomaticDeliveryFailureKeepsAuthorityCommitAndRetries(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	repo.AssignEdgeNodeForTest(restaurant.ID, "edge-a")
	if _, err := service.RefreshDeliveryPackagesForNode(ctx, restaurant.ID, "edge-a"); err != nil {
		t.Fatal(err)
	}
	repo.FailDeliveryAssemblyForTest(errors.New("storage unavailable"))
	item, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{RestaurantID: restaurant.ID, Kind: domain.CatalogItemGood, Name: "Tea", SKU: "TEA", BaseUnit: "pcs"})
	if err != nil || item.ID == "" {
		t.Fatalf("authority commit must succeed when delivery assembly fails, item=%+v err=%v", item, err)
	}
	state, err := repo.GetDeliveryState(ctx, "edge-a")
	if err != nil || state.Status != "error" || state.LastErrorCode != "DELIVERY_ASSEMBLY_FAILED" || state.ConsecutiveFailures != 1 {
		t.Fatalf("expected observable retryable delivery error, got %+v err=%v", state, err)
	}
	repo.FailDeliveryAssemblyForTest(nil)
	if err := service.RetryDeliveryForNode(ctx, restaurant.ID, "edge-a"); err != nil {
		t.Fatal(err)
	}
	state, err = repo.GetDeliveryState(ctx, "edge-a")
	if err != nil || state.Status != "pending" || state.CloudVersion != 2 || state.LastErrorCode != "" {
		t.Fatalf("expected successful retry with a new latest package, got %+v err=%v", state, err)
	}
}

func TestQRConfirmationEnabledCatalogItemValidation(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "QR Bistro", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}

	// QR включён без validity_mode — должен вернуть ErrInvalid
	_, err = service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{
		RestaurantID:          restaurant.ID,
		Kind:                  domain.CatalogItemService,
		Name:                  "Entry Ticket",
		SKU:                   "TICKET-1",
		BaseUnit:              "service",
		QRConfirmationEnabled: true,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid when qr_confirmation_enabled without validity_mode, got %v", err)
	}

	// QR включён с absolute_date без expires_at — должен вернуть ErrInvalid
	_, err = service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{
		RestaurantID:          restaurant.ID,
		Kind:                  domain.CatalogItemService,
		Name:                  "Entry Ticket",
		SKU:                   "TICKET-2",
		BaseUnit:              "service",
		QRConfirmationEnabled: true,
		ValidityMode:          domain.TicketValidityAbsoluteDate,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected ErrInvalid when absolute_date without validity_expires_at, got %v", err)
	}

	// QR включён с cash_session — успех, single_unit_per_line должен быть auto-set
	item, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{
		RestaurantID:          restaurant.ID,
		Kind:                  domain.CatalogItemService,
		Name:                  "Entry Ticket",
		SKU:                   "TICKET-3",
		BaseUnit:              "service",
		QRConfirmationEnabled: true,
		ValidityMode:          domain.TicketValidityCashSession,
	})
	if err != nil {
		t.Fatalf("expected success for valid QR config, got %v", err)
	}
	if !item.QRConfirmationEnabled {
		t.Fatal("expected qr_confirmation_enabled=true on created item")
	}
	if !item.SingleUnitPerLine {
		t.Fatal("expected single_unit_per_line=true auto-derived from qr_confirmation_enabled")
	}
	if item.ValidityMode != domain.TicketValidityCashSession {
		t.Fatalf("expected validity_mode=cash_session, got %v", item.ValidityMode)
	}

	// Update: отключаем QR — single_unit_per_line должен стать false
	disabled := false
	updated, err := service.UpdateCatalogItem(ctx, item.ID, app.UpdateCatalogItemCommand{
		QRConfirmationEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("expected success disabling QR, got %v", err)
	}
	if updated.QRConfirmationEnabled || updated.SingleUnitPerLine {
		t.Fatal("expected both qr_confirmation_enabled and single_unit_per_line=false after disabling QR")
	}

	// absolute_date с expires_at — успех
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	itemAbs, err := service.CreateCatalogItem(ctx, app.CreateCatalogItemCommand{
		RestaurantID:          restaurant.ID,
		Kind:                  domain.CatalogItemService,
		Name:                  "Season Pass",
		SKU:                   "SEASON-1",
		BaseUnit:              "service",
		QRConfirmationEnabled: true,
		ValidityMode:          domain.TicketValidityAbsoluteDate,
		ValidityExpiresAt:     &expiresAt,
	})
	if err != nil {
		t.Fatalf("expected success for absolute_date with expires_at, got %v", err)
	}
	if itemAbs.ValidityExpiresAt == nil {
		t.Fatal("expected validity_expires_at to be set")
	}
}
