package app_test

import (
	"bytes"
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
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantID: "restaurant-1", RoleID: role.ID, Name: "Ivan", PIN: "1111"}); !errors.Is(err, domain.ErrPINAlreadyExists) {
		t.Fatalf("expected duplicate PIN conflict, got %v", err)
	}
}

func TestSuspendedEmployeePINStillBlocksReuse(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	role, err := service.CreateRole(ctx, app.CreateRoleCommand{RestaurantID: "restaurant-1", Name: "cashier", PermissionsJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantID: "restaurant-1", RoleID: role.ID, Name: "Anna", PIN: "1111"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SuspendEmployee(ctx, employee.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantID: "restaurant-1", RoleID: role.ID, Name: "Ivan", PIN: "1111"}); !errors.Is(err, domain.ErrPINAlreadyExists) {
		t.Fatalf("expected suspended employee PIN to stay reserved, got %v", err)
	}
}

func TestArchivedEmployeePINCanBeReused(t *testing.T) {
	service, _ := newService()
	ctx := context.Background()
	role, err := service.CreateRole(ctx, app.CreateRoleCommand{RestaurantID: "restaurant-1", Name: "cashier", PermissionsJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	employee, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantID: "restaurant-1", RoleID: role.ID, Name: "Anna", PIN: "1111"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ArchiveEmployee(ctx, employee.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEmployee(ctx, app.CreateEmployeeCommand{RestaurantID: "restaurant-1", RoleID: role.ID, Name: "Ivan", PIN: "1111"}); err != nil {
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
	modifierGroup, err := service.CreateModifierGroup(ctx, app.CreateModifierGroupCommand{RestaurantID: restaurant.ID, Name: "Milk", Required: true, MinCount: 1, MaxCount: 2})
	if err != nil {
		t.Fatal(err)
	}
	modifierOption, err := service.CreateModifierOption(ctx, app.CreateModifierOptionCommand{RestaurantID: restaurant.ID, ModifierGroupID: modifierGroup.ID, Name: "Oat milk", PriceMinor: 300})
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
	assertPOSEdgeModifierIngestPackage(t, fullBody, restaurant.ID, menu.ID, modifierGroup.ID, modifierOption.ID, modifierBinding.ID)
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
	CatalogItems           []json.RawMessage              `json:"catalog_items,omitempty"`
	Folders                []json.RawMessage              `json:"folders,omitempty"`
	FolderParameters       []json.RawMessage              `json:"folder_parameters,omitempty"`
	Tags                   []json.RawMessage              `json:"tags,omitempty"`
	ItemTags               []json.RawMessage              `json:"item_tags,omitempty"`
	ModifierGroups         []posEdgeModifierGroup         `json:"modifier_groups,omitempty"`
	ModifierOptions        []posEdgeModifierOption        `json:"modifier_options,omitempty"`
	ModifierBindings       []posEdgeModifierGroupBinding  `json:"modifier_bindings,omitempty"`
	MenuItemModifierGroups []posEdgeMenuItemModifierGroup `json:"menu_item_modifier_groups,omitempty"`
	MenuItems              []posEdgeMenuItem              `json:"menu_items,omitempty"`
	TaxProfiles            []json.RawMessage              `json:"tax_profiles,omitempty"`
	TaxRules               []json.RawMessage              `json:"tax_rules,omitempty"`
	ServiceChargeRules     []json.RawMessage              `json:"service_charge_rules,omitempty"`
	PricingPolicies        []json.RawMessage              `json:"pricing_policies,omitempty"`
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
	ID              string `json:"id"`
	RestaurantID    string `json:"restaurant_id"`
	ModifierGroupID string `json:"modifier_group_id"`
	Name            string `json:"name"`
	PriceMinor      int64  `json:"price_minor"`
	Active          bool   `json:"active"`
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
	Name          string `json:"name"`
	Price         int64  `json:"price"`
	Currency      string `json:"currency"`
	Active        bool   `json:"active"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
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
	table, err := service.CreateTable(ctx, app.CreateTableCommand{RestaurantID: restaurant.ID, HallID: hall.ID, Name: "A1", Seats: 2})
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
