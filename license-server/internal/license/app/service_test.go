package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"license-server/internal/license/app"
)

type memoryRepo struct {
	items        map[string]app.PairingCode
	entitlements map[string]app.EntitlementSnapshot
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{items: map[string]app.PairingCode{}, entitlements: map[string]app.EntitlementSnapshot{}}
}

func (r *memoryRepo) SaveEntitlements(_ context.Context, item app.EntitlementSnapshot) error {
	r.entitlements[item.TenantID+":"+item.ServerID] = item
	return nil
}

func (r *memoryRepo) GetEntitlements(_ context.Context, tenantID, serverID string) (app.EntitlementSnapshot, error) {
	item, ok := r.entitlements[tenantID+":"+serverID]
	if !ok {
		return app.EntitlementSnapshot{}, app.ErrEntitlementNotFound
	}
	return item, nil
}

func (r *memoryRepo) Save(_ context.Context, item app.PairingCode) error {
	r.items[item.PairingCodeHash] = item
	return nil
}

func (r *memoryRepo) GetByHash(_ context.Context, hash string) (app.PairingCode, error) {
	item, ok := r.items[hash]
	if !ok {
		return app.PairingCode{}, app.ErrInvalid
	}
	return item, nil
}

func (r *memoryRepo) MarkConsumed(_ context.Context, hash string, consumedAt time.Time) error {
	item, ok := r.items[hash]
	if !ok {
		return errors.New("not found")
	}
	item.ConsumedAt = &consumedAt
	r.items[hash] = item
	return nil
}

func TestRegisterAndResolvePairingCode(t *testing.T) {
	service := app.NewService(newMemoryRepo())
	ctx := context.Background()

	if _, err := service.Register(ctx, app.RegisterPairingCodeCommand{
		PairingCode:  "123456",
		PairingID:    "pairing-1",
		InstanceID:   "cloud-instance-1",
		CloudURL:     "http://localhost:8090",
		RestaurantID: "restaurant-1",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	resolved, err := service.Resolve(ctx, app.ResolveCommand{PairingCode: "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.PairingID != "pairing-1" || resolved.CloudURL != "http://localhost:8090" || resolved.RestaurantID != "restaurant-1" {
		t.Fatalf("unexpected resolve result: %+v", resolved)
	}
	if _, err := service.Resolve(ctx, app.ResolveCommand{PairingCode: "123456"}); err != nil {
		t.Fatalf("expected license resolve to stay available until Cloud consume, got %v", err)
	}
}

func TestResolveRejectsInvalidAndExpiredCodes(t *testing.T) {
	service := app.NewService(newMemoryRepo())
	ctx := context.Background()

	if _, err := service.Resolve(ctx, app.ResolveCommand{PairingCode: "missing"}); !errors.Is(err, app.ErrInvalid) {
		t.Fatalf("expected invalid code, got %v", err)
	}
	if got := app.SafeErrorReason(errFromResolve(t, service, app.ResolveCommand{PairingCode: "missing"})); got != "pairing_code_not_found" {
		t.Fatalf("expected safe not-found reason, got %q", got)
	}
	if _, err := service.Register(ctx, app.RegisterPairingCodeCommand{
		PairingCode:  "654321",
		PairingID:    "pairing-1",
		InstanceID:   "cloud-instance-1",
		CloudURL:     "http://localhost:8090",
		RestaurantID: "restaurant-1",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}); !errors.Is(err, app.ErrExpired) {
		t.Fatalf("expected expired code registration to be rejected, got %v", err)
	}
}

func TestEntitlementsEnableDisableRevokeAndMonotonicVersion(t *testing.T) {
	service := app.NewService(newMemoryRepo())
	now := time.Now().UTC()
	put := func(version int64, status string, enabled bool) (app.EntitlementSnapshot, error) {
		return service.PutEntitlements(t.Context(), "tenant-1", "edge-1", app.PutEntitlementsCommand{Version: version, Status: status, Entitlements: map[string]bool{"kitchen-space": enabled}, IssuedAt: now, ExpiresAt: now.Add(time.Hour)})
	}
	if _, err := put(1, "active", true); err != nil {
		t.Fatal(err)
	}
	if _, err := put(2, "active", false); err != nil {
		t.Fatal(err)
	}
	revoked, err := put(3, "revoked", false)
	if err != nil || revoked.Status != "revoked" {
		t.Fatalf("revoke: %+v %v", revoked, err)
	}
	if _, err := put(3, "active", true); !errors.Is(err, app.ErrEntitlementVersion) {
		t.Fatalf("version: %v", err)
	}
	got, err := service.GetEntitlements(t.Context(), "tenant-1", "edge-1")
	if err != nil || got.Version != 3 || got.Status != "revoked" {
		t.Fatalf("get: %+v %v", got, err)
	}
}

func errFromResolve(t *testing.T, service *app.Service, cmd app.ResolveCommand) error {
	t.Helper()
	_, err := service.Resolve(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected resolve error")
	}
	return err
}
