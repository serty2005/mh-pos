package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"cloud-backend/internal/masterdata/app"
	"cloud-backend/internal/masterdata/domain"
	"cloud-backend/internal/masterdata/infra/memory"
)

// steppingClock выдает монотонно растущее время, чтобы updated_at и stream checkpoint
// различались между последовательными мутациями шаблонов.
type steppingClock struct{ t time.Time }

func (c *steppingClock) Now() time.Time {
	c.t = c.t.Add(time.Second)
	return c.t
}

func newReceiptTemplateService() (*app.Service, *memory.Repository) {
	repo := memory.NewRepository()
	clock := &steppingClock{t: time.Date(2026, 6, 27, 9, 0, 0, 0, time.UTC)}
	return app.NewService(repo, clock, &fixedIDs{}), repo
}

func ptrBool(v bool) *bool { return &v }

func TestReceiptTemplateCRUDLifecycle(t *testing.T) {
	service, _ := newReceiptTemplateService()
	ctx := context.Background()

	created, err := service.CreateReceiptTemplate(ctx, app.CreateReceiptTemplateCommand{
		RestaurantID: "restaurant-1",
		DocumentType: "precheck",
		Name:         "Пречек",
		Content:      "{header}\n{lines}\n{total}",
		CPL:          48,
		IsDefault:    true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Version != 1 || !created.IsActive || created.PrinterClass != "generic" || created.Level != 1 {
		t.Fatalf("unexpected created defaults: %+v", created)
	}

	got, err := service.GetReceiptTemplate(ctx, created.ID)
	if err != nil || got.ID != created.ID {
		t.Fatalf("get: %v %+v", err, got)
	}

	updated, err := service.UpdateReceiptTemplate(ctx, created.ID, app.UpdateReceiptTemplateCommand{
		Content: "{header}\n{lines}\n{qr}\n{total}",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version increment to 2, got %d", updated.Version)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("expected updated_at to advance: %v !> %v", updated.UpdatedAt, created.UpdatedAt)
	}

	active := true
	list, err := service.ListReceiptTemplates(ctx, app.ReceiptTemplateFilter{RestaurantID: "restaurant-1", IsActive: &active})
	if err != nil || len(list) != 1 {
		t.Fatalf("list active: %v len=%d", err, len(list))
	}

	deactivated, err := service.DeactivateReceiptTemplate(ctx, created.ID)
	if err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if deactivated.IsActive || deactivated.IsDefault {
		t.Fatalf("soft-delete must clear is_active and is_default: %+v", deactivated)
	}
	if deactivated.Version != 3 {
		t.Fatalf("expected version 3 after soft-delete, got %d", deactivated.Version)
	}
	activeList, err := service.ListReceiptTemplates(ctx, app.ReceiptTemplateFilter{IsActive: ptrBool(true)})
	if err != nil || len(activeList) != 0 {
		t.Fatalf("expected no active templates after soft-delete: %v len=%d", err, len(activeList))
	}
}

func TestReceiptTemplateValidationRejectsBadFields(t *testing.T) {
	service, _ := newReceiptTemplateService()
	ctx := context.Background()
	cases := []app.CreateReceiptTemplateCommand{
		{DocumentType: "fiscal_check", Name: "x", Content: "y", CPL: 48}, // fiscal запрещён
		{DocumentType: "precheck", Name: "", Content: "y", CPL: 48},      // пустое имя
		{DocumentType: "precheck", Name: "x", Content: "", CPL: 48},      // пустой content
		{DocumentType: "precheck", Name: "x", Content: "y", CPL: 33},     // недопустимый cpl
	}
	for i, cmd := range cases {
		if _, err := service.CreateReceiptTemplate(ctx, cmd); !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("case %d expected ErrInvalid, got %v", i, err)
		}
	}
}

func TestReceiptTemplateDefaultUniquenessConflict(t *testing.T) {
	service, _ := newReceiptTemplateService()
	ctx := context.Background()
	if _, err := service.CreateReceiptTemplate(ctx, app.CreateReceiptTemplateCommand{
		RestaurantID: "restaurant-1", DocumentType: "precheck", Name: "A", Content: "a", CPL: 48, IsDefault: true,
	}); err != nil {
		t.Fatalf("first default: %v", err)
	}
	_, err := service.CreateReceiptTemplate(ctx, app.CreateReceiptTemplateCommand{
		RestaurantID: "restaurant-1", DocumentType: "precheck", Name: "B", Content: "b", CPL: 48, IsDefault: true,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for second active default, got %v", err)
	}
}

func TestReceiptTemplatesStreamPackageAssemblyAndCheckpoint(t *testing.T) {
	service, repo := newReceiptTemplateService()
	ctx := context.Background()
	restaurant, err := service.CreateRestaurant(ctx, app.CreateRestaurantCommand{Name: "Demo", Timezone: "Europe/Moscow", Currency: "RUB"})
	if err != nil {
		t.Fatal(err)
	}
	// tenant-level defaults (restaurant_id IS NULL) для precheck и ticket
	if _, err := service.CreateReceiptTemplate(ctx, app.CreateReceiptTemplateCommand{DocumentType: "precheck", Name: "tenant precheck", Content: "TENANT-PRECHECK", CPL: 32, IsDefault: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateReceiptTemplate(ctx, app.CreateReceiptTemplateCommand{DocumentType: "ticket", Name: "tenant ticket", Content: "TENANT-TICKET", CPL: 48, IsDefault: true}); err != nil {
		t.Fatal(err)
	}
	// restaurant-specific precheck перекрывает tenant precheck
	restPrecheck, err := service.CreateReceiptTemplate(ctx, app.CreateReceiptTemplateCommand{RestaurantID: restaurant.ID, DocumentType: "precheck", Name: "rest precheck", Content: "REST-PRECHECK", CPL: 48, IsDefault: true})
	if err != nil {
		t.Fatal(err)
	}

	pub, err := service.Publish(ctx, app.PublishCommand{RestaurantID: restaurant.ID, PublishedBy: "operator-1", NodeDeviceID: "node-1"})
	if err != nil {
		t.Fatal(err)
	}
	if pub.Counts["receipt_templates"] != 2 {
		t.Fatalf("expected 2 effective templates (rest precheck + tenant ticket), got %d", pub.Counts["receipt_templates"])
	}
	pkg, ok := repo.Package("receipt_templates", "node-1")
	if !ok {
		t.Fatal("expected receipt_templates stream package")
	}
	payload := string(pkg.PayloadJSON)
	if !strings.Contains(payload, "REST-PRECHECK") {
		t.Fatalf("expected restaurant precheck in package, payload=%s", payload)
	}
	if strings.Contains(payload, "TENANT-PRECHECK") {
		t.Fatalf("tenant precheck must be overridden by restaurant-specific, payload=%s", payload)
	}
	if !strings.Contains(payload, "TENANT-TICKET") {
		t.Fatalf("expected tenant ticket default in package, payload=%s", payload)
	}
	if !strings.HasPrefix(pkg.CheckpointToken, "receipt-templates:") {
		t.Fatalf("expected stream-specific checkpoint token, got %q", pkg.CheckpointToken)
	}
	checkpoint1 := pkg.CheckpointToken
	version1 := pkg.CloudVersion

	// Изменение шаблона должно сдвинуть checkpoint (MAX updated_at) и cloud_version.
	if _, err := service.UpdateReceiptTemplate(ctx, restPrecheck.ID, app.UpdateReceiptTemplateCommand{Content: "REST-PRECHECK-V2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Publish(ctx, app.PublishCommand{RestaurantID: restaurant.ID, PublishedBy: "operator-1", NodeDeviceID: "node-1"}); err != nil {
		t.Fatal(err)
	}
	pkg2, ok := repo.Package("receipt_templates", "node-1")
	if !ok {
		t.Fatal("expected receipt_templates stream package after update")
	}
	if pkg2.CheckpointToken == checkpoint1 {
		t.Fatalf("expected checkpoint token to change after template update, still %q", checkpoint1)
	}
	if pkg2.CloudVersion <= version1 {
		t.Fatalf("expected cloud_version to advance after update: %d <= %d", pkg2.CloudVersion, version1)
	}
}
