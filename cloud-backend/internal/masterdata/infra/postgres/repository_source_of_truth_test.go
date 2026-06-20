package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
	"cloud-backend/internal/masterdata/infra/memory"
)

func TestRestaurantPersistenceRoundTripUpdateListAndNotFound(t *testing.T) {
	ctx := t.Context()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)

	base := testMasterDataTime()
	restaurants := []domain.Restaurant{
		{
			ID:                           "restaurant-source-002",
			Name:                         "Beta",
			Timezone:                     "Asia/Almaty",
			Currency:                     "KZT",
			BusinessDayMode:              "24_7",
			BusinessDayBoundaryLocalTime: "00:00",
			Status:                       domain.RestaurantActive,
			CloudVersion:                 1,
			CreatedAt:                    base.Add(time.Minute),
			UpdatedAt:                    base.Add(time.Minute),
		},
		{
			ID:                           "restaurant-source-001",
			Name:                         "Alpha",
			Timezone:                     "Europe/Moscow",
			Currency:                     "RUB",
			BusinessDayMode:              "standard",
			BusinessDayBoundaryLocalTime: "05:30",
			Status:                       domain.RestaurantActive,
			CloudVersion:                 3,
			CreatedAt:                    base,
			UpdatedAt:                    base,
		},
	}
	for _, restaurant := range restaurants {
		got, err := repo.CreateRestaurant(ctx, restaurant)
		if err != nil {
			t.Fatal(err)
		}
		assertRestaurantEqual(t, got, restaurant)
	}

	got, err := repo.GetRestaurant(ctx, restaurants[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	assertRestaurantEqual(t, got, restaurants[1])

	updated := got
	updated.Name = "Alpha Central"
	updated.Timezone = "Europe/Samara"
	updated.Currency = "EUR"
	updated.BusinessDayMode = "24_7"
	updated.BusinessDayBoundaryLocalTime = "00:00"
	updated.Status = domain.RestaurantArchived
	updated.CloudVersion = 4
	updated.UpdatedAt = base.Add(2 * time.Hour)
	archivedAt := base.Add(time.Hour)
	updated.ArchivedAt = &archivedAt
	got, err = repo.UpdateRestaurant(ctx, updated)
	if err != nil {
		t.Fatal(err)
	}
	assertRestaurantEqual(t, got, updated)
	if !got.CreatedAt.Equal(restaurants[1].CreatedAt) {
		t.Fatalf("created_at changed on restaurant update: got %s want %s", got.CreatedAt, restaurants[1].CreatedAt)
	}

	list, err := repo.ListRestaurants(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, restaurantIDs(list), []string{"restaurant-source-001", "restaurant-source-002"})

	_, err = repo.GetRestaurant(ctx, "missing-restaurant")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected restaurant ErrNotFound, got %v", err)
	}
}

func TestPostgresIntegrationDSNGuardAllowsOnlyLocalTestDatabases(t *testing.T) {
	allowed := []string{
		"postgres://postgres:postgres@localhost:5432/mh_pos_cloud?sslmode=disable",
		"postgres://postgres:postgres@127.0.0.1:5432/mh_pos_cloud_test?sslmode=disable",
		"postgres://postgres:postgres@cloud-postgres:5432/mh_pos_cloud?sslmode=disable",
	}
	for _, dsn := range allowed {
		if err := validatePostgresIntegrationTestDSN(dsn); err != nil {
			t.Fatalf("expected DSN to be allowed, got %v", err)
		}
	}

	rejected := []string{
		"postgres://postgres:postgres@db.example.com:5432/mh_pos_cloud?sslmode=disable",
		"postgres://postgres:postgres@localhost:5432/mh_pos_prod?sslmode=disable",
		"postgres://postgres:postgres@localhost:5432/?sslmode=disable",
	}
	for _, dsn := range rejected {
		if err := validatePostgresIntegrationTestDSN(dsn); err == nil {
			t.Fatal("expected unsafe DSN to be rejected")
		}
	}
}

func TestRolePermissionsPersistenceAndMemoryParity(t *testing.T) {
	ctx := t.Context()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	pgRepo := NewRepository(pool)
	memRepo := memory.NewRepository()

	base := testMasterDataTime()
	roles := []domain.Role{
		{
			ID:              "role-source-empty",
			Name:            "Empty",
			PermissionsJSON: `{}`,
			Active:          true,
			CloudVersion:    1,
			CreatedAt:       base,
			UpdatedAt:       base,
		},
		{
			ID:              "role-source-many",
			Name:            "Manager",
			PermissionsJSON: `{"permissions":["pos.order.create","pos.pricing.discount.apply","custom.experimental.permission"],"pos.floor.view":true}`,
			Active:          true,
			CloudVersion:    1,
			CreatedAt:       base.Add(time.Minute),
			UpdatedAt:       base.Add(time.Minute),
		},
		{
			ID:              "role-source-other",
			Name:            "Other",
			PermissionsJSON: `{"permissions":["pos.menu.view"]}`,
			Active:          true,
			CloudVersion:    1,
			CreatedAt:       base.Add(2 * time.Minute),
			UpdatedAt:       base.Add(2 * time.Minute),
		},
	}
	for _, repo := range []roleRepository{memRepo, pgRepo} {
		for _, role := range roles {
			if _, err := repo.CreateRole(ctx, role); err != nil {
				t.Fatal(err)
			}
		}
		updated := roles[1]
		updated.PermissionsJSON = `{"permissions":["pos.payment.cash","custom.experimental.permission"],"pos.pricing.discount.apply":true}`
		updated.Active = false
		updated.CloudVersion = 2
		updated.UpdatedAt = base.Add(time.Hour)
		if _, err := repo.UpdateRole(ctx, updated); err != nil {
			t.Fatal(err)
		}
	}

	got, err := pgRepo.GetRole(ctx, roles[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONPermissions(t, got.PermissionsJSON, []string{"pos.payment.cash", "custom.experimental.permission", "pos.pricing.discount.apply"})

	pgList, err := pgRepo.ListRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, roleIDs(pgList), []string{"role-source-empty", "role-source-many", "role-source-other"})
	memList, err := memRepo.ListRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertRolesParity(t, pgList, memList)
}

func TestEmployeePINHashPersistenceAndNoPlaintextLeakage(t *testing.T) {
	ctx := t.Context()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	repo := NewRepository(pool)

	base := testMasterDataTime()
	role := domain.Role{
		ID:              "role-employee-source",
		Name:            "Cashier",
		PermissionsJSON: `{}`,
		Active:          true,
		CloudVersion:    1,
		CreatedAt:       base,
		UpdatedAt:       base,
	}
	if _, err := repo.CreateRole(ctx, role); err != nil {
		t.Fatal(err)
	}
	otherRole := role
	otherRole.ID = "role-employee-source-other"
	otherRole.Name = "Other Cashier"
	if _, err := repo.CreateRole(ctx, otherRole); err != nil {
		t.Fatal(err)
	}

	pinHash := "$argon2id$v=19$m=65536,t=3,p=1$source$safehash"
	employee := domain.Employee{
		ID:                     "employee-source-001",
		RestaurantIDs:          []string{"restaurant-employee-a"},
		RoleID:                 role.ID,
		Name:                   "Nina",
		Status:                 domain.EmployeeActive,
		PINHash:                pinHash,
		PINCredentialVersion:   7,
		PermissionSnapshotJSON: `{"permissions":["pos.order.create"]}`,
		CloudVersion:           1,
		CreatedAt:              base,
		UpdatedAt:              base,
	}
	created, err := repo.CreateEmployee(ctx, employee)
	if err != nil {
		t.Fatal(err)
	}
	assertEmployeeEqual(t, created, employee)
	if !created.PINConfigured {
		t.Fatal("expected returned employee to report configured PIN")
	}

	otherEmployee := employee
	otherEmployee.ID = "employee-source-other"
	otherEmployee.RestaurantIDs = []string{"restaurant-employee-b"}
	otherEmployee.RoleID = otherRole.ID
	otherEmployee.Name = "Other"
	if _, err := repo.CreateEmployee(ctx, otherEmployee); err != nil {
		t.Fatal(err)
	}

	plainMarker := "plaintext-pin-1234-marker"
	assertNoSensitiveEmployeeLeak(t, created, plainMarker)
	var rowJSON string
	if err := pool.QueryRow(ctx, `SELECT row_to_json(cloud_employees)::text FROM cloud_employees WHERE id = $1`, employee.ID).Scan(&rowJSON); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rowJSON, plainMarker) {
		t.Fatal("employee row contains plaintext PIN marker")
	}

	got, err := repo.GetEmployee(ctx, employee.ID)
	if err != nil {
		t.Fatal(err)
	}
	got.Name = "Nina Updated"
	got.Status = domain.EmployeeSuspended
	got.CloudVersion = 2
	got.UpdatedAt = base.Add(time.Hour)
	updated, err := repo.UpdateEmployee(ctx, got)
	if err != nil {
		t.Fatal(err)
	}
	if updated.PINHash != pinHash || updated.PINCredentialVersion != 7 {
		t.Fatal("employee update did not preserve PIN credential fields")
	}
	if updated.SuspendedAt == nil || !updated.SuspendedAt.Equal(got.UpdatedAt) {
		t.Fatalf("expected suspended_at to be set from updated_at, got %+v", updated.SuspendedAt)
	}
	assertNoSensitiveEmployeeLeak(t, updated, plainMarker)

	employees, err := repo.ListEmployees(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, employeeIDs(employees), []string{"employee-source-001", "employee-source-other"})
	archived := updated
	archived.Status = domain.EmployeeArchived
	archived.UpdatedAt = base.Add(2 * time.Hour)
	archivedAt := archived.UpdatedAt
	archived.ArchivedAt = &archivedAt
	if _, err := repo.UpdateEmployee(ctx, archived); err != nil {
		t.Fatal(err)
	}
}

func TestPublicationPackageSaveCurrentAtomicityVersionAndMemoryParity(t *testing.T) {
	ctx := t.Context()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	pgRepo := NewRepository(pool)
	memRepo := memory.NewRepository()

	base := testMasterDataTime()
	if version, err := pgRepo.NextPublicationVersion(ctx, "restaurant-publication"); err != nil || version != 1 {
		t.Fatalf("expected initial publication version 1, got version=%d err=%v", version, err)
	}

	pub := testPublication("publication-source-001", "restaurant-publication", 1, base)
	packages := []app.StreamPackage{
		testStreamPackage("catalog", "node-publication", "restaurant-publication", 1, base, `{"stream":"catalog","items":[{"id":"catalog-1"}]}`),
		testStreamPackage("menu", "node-publication", "restaurant-publication", 1, base, `{"stream":"menu","items":[{"id":"menu-1"}]}`),
		testStreamPackage("pricing_policy", "node-publication", "restaurant-publication", 1, base, `{"stream":"pricing_policy","pricing_policies":[{"id":"policy-1","requires_permission":"pos.pricing.discount.apply"}]}`),
		testStreamPackage("floor", "node-publication", "restaurant-publication", 1, base, `{"stream":"floor","halls":[{"id":"hall-1"}]}`),
	}
	for _, repo := range []publicationRepository{memRepo, pgRepo} {
		if _, err := repo.SavePublication(ctx, pub, packages); err != nil {
			t.Fatal(err)
		}
	}

	assertPublicationPackageRows(t, ctx, pool, "restaurant-publication", 4)
	current, err := pgRepo.GetCurrentPublication(ctx, "restaurant-publication")
	if err != nil {
		t.Fatal(err)
	}
	assertPublicationEqual(t, current, pub)
	gotPub, err := pgRepo.GetPublication(ctx, "restaurant-publication", pub.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertPublicationEqual(t, gotPub, pub)
	assertPackagePayloadShape(t, ctx, pool, "pricing_policy", "node-publication", "policy-1")
	if version, err := pgRepo.NextPublicationVersion(ctx, "restaurant-publication"); err != nil || version != 2 {
		t.Fatalf("expected second publication version 2, got version=%d err=%v", version, err)
	}

	pub2 := testPublication("publication-source-002", "restaurant-publication", 2, base.Add(time.Hour))
	pub2.PackageSHA256 = "sha-publication-source-002"
	if _, err := pgRepo.SavePublication(ctx, pub2, []app.StreamPackage{testStreamPackage("catalog", "node-publication", "restaurant-publication", 2, base.Add(time.Hour), `{"stream":"catalog","items":[{"id":"catalog-2"}]}`)}); err != nil {
		t.Fatal(err)
	}
	current, err = pgRepo.GetCurrentPublication(ctx, "restaurant-publication")
	if err != nil {
		t.Fatal(err)
	}
	if current.ID != pub2.ID || current.Version != 2 {
		t.Fatalf("expected latest publication v2, got %+v", current)
	}

	badPub := testPublication("publication-source-bad", "restaurant-publication", 3, base.Add(2*time.Hour))
	_, err = pgRepo.SavePublication(ctx, badPub, []app.StreamPackage{
		testStreamPackage("catalog", "node-publication-bad", "restaurant-publication", 3, base, `{"stream":"catalog"}`),
		testStreamPackage("invalid_stream", "node-publication-bad", "restaurant-publication", 3, base, `{"stream":"invalid"}`),
	})
	if err == nil {
		t.Fatal("expected invalid stream package to fail")
	}
	assertNoPublicationPartialRows(t, ctx, pool, badPub.ID, "node-publication-bad")

	memCurrent, err := memRepo.GetCurrentPublication(ctx, "restaurant-publication")
	if err != nil {
		t.Fatal(err)
	}
	assertPublicationEqual(t, memCurrent, pub)
}

func TestStopListUpsertPersistenceIsolationNotFoundAndMemoryParity(t *testing.T) {
	ctx := t.Context()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	pgRepo := NewRepository(pool)
	memRepo := memory.NewRepository()

	base := testMasterDataTime()
	version1 := int64(1)
	version2 := int64(2)
	qty := 3.5
	entry := domain.StopListEntry{
		ID:                "stop-source-001",
		RestaurantID:      "restaurant-stop-a",
		CatalogItemID:     "catalog-stop-001",
		AvailableQuantity: &qty,
		Source:            "cloud",
		Reason:            "initial",
		Active:            true,
		CloudVersion:      &version1,
		UpdatedAt:         base,
	}
	for _, repo := range []stopListRepository{memRepo, pgRepo} {
		if _, err := repo.UpsertStopListEntry(ctx, entry); err != nil {
			t.Fatal(err)
		}
		updated := entry
		updated.ID = "stop-source-replacement-id"
		updated.AvailableQuantity = nil
		updated.Reason = "updated"
		updated.Active = false
		updated.CloudVersion = &version2
		updated.UpdatedAt = base.Add(time.Hour)
		got, err := repo.UpsertStopListEntry(ctx, updated)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != entry.ID {
			t.Fatalf("expected stop-list upsert to keep stable id %q, got %q", entry.ID, got.ID)
		}
	}
	other := entry
	other.ID = "stop-source-other-restaurant"
	other.RestaurantID = "restaurant-stop-b"
	other.CatalogItemID = "catalog-stop-other"
	if _, err := pgRepo.UpsertStopListEntry(ctx, other); err != nil {
		t.Fatal(err)
	}

	got, err := pgRepo.GetStopListEntry(ctx, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AvailableQuantity != nil || got.Reason != "updated" || got.Active || got.CloudVersion == nil || *got.CloudVersion != version2 {
		t.Fatalf("unexpected stop-list update result: %+v", got)
	}
	assertStopListRowCount(t, ctx, pool, "restaurant-stop-a", "catalog-stop-001", 1)

	pgList, err := pgRepo.ListStopListEntries(ctx, "restaurant-stop-a")
	if err != nil {
		t.Fatal(err)
	}
	memList, err := memRepo.ListStopListEntries(ctx, "restaurant-stop-a")
	if err != nil {
		t.Fatal(err)
	}
	assertStopListsParity(t, pgList, memList)
	assertIDs(t, stopListIDs(pgList), []string{"stop-source-001"})

	_, err = pgRepo.GetStopListEntry(ctx, "missing-stop-list")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected stop-list ErrNotFound, got %v", err)
	}
}

func TestPricingPolicyPersistenceUpdateListPublicationFieldAndMemoryParity(t *testing.T) {
	ctx := t.Context()
	pool, closeFn := openPostgresWithBaseline(t, ctx)
	defer closeFn()
	pgRepo := NewRepository(pool)
	memRepo := memory.NewRepository()

	base := testMasterDataTime()
	policies := []domain.PricingPolicy{
		{
			ID:                 "pricing-source-020",
			RestaurantID:       "restaurant-pricing-a",
			Name:               "Manual Discount",
			Kind:               domain.PricingPolicyDiscount,
			Scope:              "order",
			AmountKind:         "percentage",
			ValueBasisPoints:   500,
			ApplicationIndex:   20,
			Manual:             true,
			RequiresPermission: "pos.pricing.discount.apply",
			Status:             domain.StatusPublished,
			CloudVersion:       1,
			CreatedAt:          base.Add(time.Minute),
			UpdatedAt:          base.Add(time.Minute),
		},
		{
			ID:               "pricing-source-010",
			RestaurantID:     "restaurant-pricing-a",
			Name:             "Service Surcharge",
			Kind:             domain.PricingPolicySurcharge,
			Scope:            "order",
			AmountKind:       "fixed",
			AmountMinor:      1500,
			ApplicationIndex: 10,
			Manual:           false,
			Status:           domain.StatusDraft,
			CloudVersion:     1,
			CreatedAt:        base,
			UpdatedAt:        base,
		},
		{
			ID:                 "pricing-source-other",
			RestaurantID:       "restaurant-pricing-b",
			Name:               "Other",
			Kind:               domain.PricingPolicyDiscount,
			Scope:              "line",
			AmountKind:         "percentage",
			ValueBasisPoints:   1000,
			ApplicationIndex:   5,
			Manual:             true,
			RequiresPermission: "custom.pricing.override",
			Status:             domain.StatusPublished,
			CloudVersion:       1,
			CreatedAt:          base,
			UpdatedAt:          base,
		},
	}
	for _, repo := range []pricingPolicyRepository{memRepo, pgRepo} {
		for _, policy := range policies {
			if _, err := repo.CreatePricingPolicy(ctx, policy); err != nil {
				t.Fatal(err)
			}
		}
		updated := policies[0]
		updated.Name = "Manual Discount Updated"
		updated.Scope = "line"
		updated.AmountKind = "fixed"
		updated.AmountMinor = 250
		updated.ValueBasisPoints = 0
		updated.ApplicationIndex = 30
		updated.Manual = false
		updated.RequiresPermission = "custom.pricing.override"
		updated.Status = domain.StatusArchived
		updated.CloudVersion = 2
		updated.UpdatedAt = base.Add(time.Hour)
		if _, err := repo.UpdatePricingPolicy(ctx, updated); err != nil {
			t.Fatal(err)
		}
	}

	got, err := pgRepo.GetPricingPolicy(ctx, policies[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != domain.PricingPolicyDiscount || got.RestaurantID != policies[0].RestaurantID || !got.CreatedAt.Equal(policies[0].CreatedAt) {
		t.Fatalf("pricing stable fields changed unexpectedly: %+v", got)
	}
	if got.RequiresPermission != "custom.pricing.override" || got.AmountKind != "fixed" || got.AmountMinor != 250 || got.ValueBasisPoints != 0 || got.Status != domain.StatusArchived {
		t.Fatalf("pricing mutable fields did not round-trip: %+v", got)
	}

	pgList, err := pgRepo.ListPricingPolicies(ctx, "restaurant-pricing-a")
	if err != nil {
		t.Fatal(err)
	}
	assertIDs(t, pricingPolicyIDs(pgList), []string{"pricing-source-010", "pricing-source-020"})
	memList, err := memRepo.ListPricingPolicies(ctx, "restaurant-pricing-a")
	if err != nil {
		t.Fatal(err)
	}
	assertPricingPoliciesParity(t, pgList, memList)

	pub := testPublication("publication-pricing-source", "restaurant-pricing-a", 1, base.Add(2*time.Hour))
	pkg := testStreamPackage("pricing_policy", "node-pricing", "restaurant-pricing-a", 1, base.Add(2*time.Hour), `{"pricing_policies":[{"id":"pricing-source-020","requires_permission":"custom.pricing.override"}]}`)
	if _, err := pgRepo.SavePublication(ctx, pub, []app.StreamPackage{pkg}); err != nil {
		t.Fatal(err)
	}
	assertPackagePayloadShape(t, ctx, pool, "pricing_policy", "node-pricing", "custom.pricing.override")
}

type roleRepository interface {
	CreateRole(context.Context, domain.Role) (domain.Role, error)
	UpdateRole(context.Context, domain.Role) (domain.Role, error)
	ListRoles(context.Context) ([]domain.Role, error)
}

type publicationRepository interface {
	SavePublication(context.Context, domain.Publication, []app.StreamPackage) (domain.Publication, error)
}

type stopListRepository interface {
	UpsertStopListEntry(context.Context, domain.StopListEntry) (domain.StopListEntry, error)
	ListStopListEntries(context.Context, string) ([]domain.StopListEntry, error)
}

type pricingPolicyRepository interface {
	CreatePricingPolicy(context.Context, domain.PricingPolicy) (domain.PricingPolicy, error)
	UpdatePricingPolicy(context.Context, domain.PricingPolicy) (domain.PricingPolicy, error)
	ListPricingPolicies(context.Context, string) ([]domain.PricingPolicy, error)
}

func testMasterDataTime() time.Time {
	return time.Date(2026, 6, 18, 9, 0, 0, 0, time.UTC)
}

func testPublication(id, restaurantID string, version int64, now time.Time) domain.Publication {
	return domain.Publication{
		ID:            id,
		RestaurantID:  restaurantID,
		Version:       version,
		Status:        domain.StatusPublished,
		CloudVersion:  version,
		PublishedAt:   now,
		PublishedBy:   "operator-source",
		PackageJSON:   json.RawMessage(`{"restaurant_id":"` + restaurantID + `","version":` + strconv.FormatInt(version, 10) + `}`),
		PackageSHA256: "sha-" + id,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func testStreamPackage(streamName, nodeDeviceID, restaurantID string, version int64, now time.Time, payload string) app.StreamPackage {
	return app.StreamPackage{
		StreamName:      streamName,
		NodeDeviceID:    nodeDeviceID,
		RestaurantID:    restaurantID,
		SyncMode:        "full_snapshot",
		CloudVersion:    version,
		CheckpointToken: "checkpoint-" + streamName,
		CloudUpdatedAt:  now,
		PayloadJSON:     json.RawMessage(payload),
	}
}

func assertRestaurantEqual(t *testing.T, got, want domain.Restaurant) {
	t.Helper()
	if got.ID != want.ID ||
		got.Name != want.Name ||
		got.Status != want.Status ||
		got.Timezone != want.Timezone ||
		got.Currency != want.Currency ||
		got.BusinessDayMode != want.BusinessDayMode ||
		got.BusinessDayBoundaryLocalTime != want.BusinessDayBoundaryLocalTime ||
		got.CloudVersion != want.CloudVersion ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.UpdatedAt.Equal(want.UpdatedAt) ||
		!timePtrEqual(got.ArchivedAt, want.ArchivedAt) {
		t.Fatalf("unexpected restaurant: got %+v want %+v", got, want)
	}
}

func assertEmployeeEqual(t *testing.T, got, want domain.Employee) {
	t.Helper()
	if got.ID != want.ID ||
		got.RoleID != want.RoleID ||
		got.Name != want.Name ||
		got.Status != want.Status ||
		got.PINHash != want.PINHash ||
		got.PINCredentialVersion != want.PINCredentialVersion ||
		!jsonEqual([]byte(got.PermissionSnapshotJSON), []byte(want.PermissionSnapshotJSON)) ||
		got.CloudVersion != want.CloudVersion ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.UpdatedAt.Equal(want.UpdatedAt) ||
		!reflect.DeepEqual(got.RestaurantIDs, want.RestaurantIDs) {
		t.Fatalf("unexpected employee round-trip for id %q", want.ID)
	}
}

func assertNoSensitiveEmployeeLeak(t *testing.T, employee domain.Employee, plainMarker string) {
	t.Helper()
	raw, err := json.Marshal(employee)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), plainMarker) || strings.Contains(string(raw), employee.PINHash) {
		t.Fatal("employee public JSON contains sensitive PIN material")
	}
}

func assertPublicationEqual(t *testing.T, got, want domain.Publication) {
	t.Helper()
	if got.ID != want.ID ||
		got.RestaurantID != want.RestaurantID ||
		got.Version != want.Version ||
		got.Status != want.Status ||
		got.CloudVersion != want.CloudVersion ||
		got.PublishedBy != want.PublishedBy ||
		got.PackageSHA256 != want.PackageSHA256 ||
		!got.PublishedAt.Equal(want.PublishedAt) ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.UpdatedAt.Equal(want.UpdatedAt) ||
		!jsonEqual(got.PackageJSON, want.PackageJSON) {
		t.Fatalf("unexpected publication: got %+v want %+v", got, want)
	}
}

func assertJSONPermissions(t *testing.T, raw string, want []string) {
	t.Helper()
	got := permissionsFromRawJSON(t, raw)
	slices.Sort(got)
	slices.Sort(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected permission ids: got %v want %v", got, want)
	}
}

func permissionsFromRawJSON(t *testing.T, raw string) []string {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatalf("permissions_json is not valid JSON: %v", err)
	}
	seen := map[string]struct{}{}
	for key, item := range value {
		if allowed, ok := item.(bool); ok && allowed {
			seen[key] = struct{}{}
		}
	}
	if permissions, ok := value["permissions"].([]any); ok {
		for _, item := range permissions {
			text, ok := item.(string)
			if ok {
				seen[text] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for permission := range seen {
		out = append(out, permission)
	}
	return out
}

func assertRolesParity(t *testing.T, got, want []domain.Role) {
	t.Helper()
	got = normalizeRoles(got)
	want = normalizeRoles(want)
	if len(got) != len(want) {
		t.Fatalf("role parity length mismatch: got %d want %d", len(got), len(want))
	}
	for i := range got {
		if got[i].ID != want[i].ID ||
			got[i].Name != want[i].Name ||
			got[i].Active != want[i].Active ||
			got[i].CloudVersion != want[i].CloudVersion ||
			!jsonEqual([]byte(got[i].PermissionsJSON), []byte(want[i].PermissionsJSON)) {
			t.Fatalf("role parity mismatch: got %+v want %+v", got[i], want[i])
		}
	}
}

func assertStopListsParity(t *testing.T, got, want []domain.StopListEntry) {
	t.Helper()
	got = normalizeStopLists(got)
	want = normalizeStopLists(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stop-list parity mismatch: got %+v want %+v", got, want)
	}
}

func assertPricingPoliciesParity(t *testing.T, got, want []domain.PricingPolicy) {
	t.Helper()
	got = normalizePricingPolicies(got)
	want = normalizePricingPolicies(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pricing policy parity mismatch: got %+v want %+v", got, want)
	}
}

func normalizeRoles(items []domain.Role) []domain.Role {
	out := slices.Clone(items)
	slices.SortFunc(out, func(a, b domain.Role) int {
		return strings.Compare(a.ID, b.ID)
	})
	for i := range out {
		out[i].CreatedAt = time.Time{}
		out[i].UpdatedAt = time.Time{}
		out[i].ArchivedAt = nil
	}
	return out
}

func normalizeStopLists(items []domain.StopListEntry) []domain.StopListEntry {
	out := slices.Clone(items)
	slices.SortFunc(out, func(a, b domain.StopListEntry) int {
		return strings.Compare(a.ID, b.ID)
	})
	for i := range out {
		out[i].UpdatedAt = time.Time{}
	}
	return out
}

func normalizePricingPolicies(items []domain.PricingPolicy) []domain.PricingPolicy {
	out := slices.Clone(items)
	slices.SortFunc(out, func(a, b domain.PricingPolicy) int {
		return strings.Compare(a.ID, b.ID)
	})
	for i := range out {
		out[i].CreatedAt = time.Time{}
		out[i].UpdatedAt = time.Time{}
		out[i].ArchivedAt = nil
	}
	return out
}

func jsonEqual(a, b []byte) bool {
	var av any
	var bv any
	if err := json.Unmarshal(a, &av); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

func timePtrEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func restaurantIDs(items []domain.Restaurant) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func roleIDs(items []domain.Role) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func employeeIDs(items []domain.Employee) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func stopListIDs(items []domain.StopListEntry) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func pricingPolicyIDs(items []domain.PricingPolicy) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func assertIDs(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected ids: got %v want %v", got, want)
	}
}

func assertPublicationPackageRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, restaurantID string, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM cloud_master_data_packages WHERE restaurant_id = $1`, restaurantID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("expected %d package rows, got %d", want, count)
	}
}

func assertPackagePayloadShape(t *testing.T, ctx context.Context, pool *pgxpool.Pool, streamName, nodeDeviceID, wantMarker string) {
	t.Helper()
	var payload string
	if err := pool.QueryRow(ctx, `SELECT payload_json::text FROM cloud_master_data_packages WHERE stream_name = $1 AND node_device_id = $2`, streamName, nodeDeviceID).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, wantMarker) {
		t.Fatalf("package payload for stream %q does not contain expected marker %q", streamName, wantMarker)
	}
}

func assertNoPublicationPartialRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, publicationID, nodeDeviceID string) {
	t.Helper()
	var publicationCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM cloud_master_data_publications WHERE id = $1`, publicationID).Scan(&publicationCount); err != nil {
		t.Fatal(err)
	}
	var packageCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM cloud_master_data_packages WHERE node_device_id = $1`, nodeDeviceID).Scan(&packageCount); err != nil {
		t.Fatal(err)
	}
	if publicationCount != 0 || packageCount != 0 {
		t.Fatalf("expected no partial publication/package rows, got publications=%d packages=%d", publicationCount, packageCount)
	}
}

func assertStopListRowCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, restaurantID, catalogItemID string, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(1) FROM stop_lists WHERE restaurant_id = $1 AND catalog_item_id = $2`, restaurantID, catalogItemID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("expected %d stop-list rows, got %d", want, count)
	}
}
