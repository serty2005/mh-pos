package app_test

import (
	"context"
	"encoding/json"
	"errors"
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
		RestaurantID:    "restaurant-1",
		Name:            "manager",
		PermissionsJSON: `{"pos.menu.view":true,"pos.payment.cash":true}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{
		RestaurantID: "restaurant-1",
		RoleID:       role.ID,
		Name:         "Anna",
		PIN:          "1111",
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
		RestaurantID:    "restaurant-1",
		Name:            "broken",
		PermissionsJSON: `{"pos.order.create":true,"pos.unknown.permission":true}`,
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected invalid unknown permission, got %v", err)
	}
}

func TestDuplicateActivePINIsRejectedPerRestaurant(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	role, err := service.CreateRole(ctx, app.CreateRoleCommand{RestaurantID: "restaurant-1", Name: "cashier", PermissionsJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantID: "restaurant-1", RoleID: role.ID, Name: "Anna", PIN: "1111"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantID: "restaurant-1", RoleID: role.ID, Name: "Ivan", PIN: "1111"}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected duplicate PIN conflict, got %v", err)
	}
}

func TestCatalogMenuValidationAndPublicationPackageShape(t *testing.T) {
	service, repo := newService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo Bistro", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	role, err := service.CreateRole(ctx, app.CreateRoleCommand{RestaurantID: restaurant.ID, Name: "cashier", PermissionsJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantID: restaurant.ID, RoleID: role.ID, Name: "Oleg", PIN: "3333"})
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

func newService() (*app.Service, *memory.Repository) {
	repo := memory.NewRepository()
	return app.NewService(repo, fixedClock{}, &fixedIDs{}), repo
}
