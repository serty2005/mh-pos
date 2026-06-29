package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"license-server/internal/license/app"
)

type memoryRepo struct {
	items         map[string]app.PairingCode
	entitlements  map[string]app.EntitlementSnapshot
	servers       map[string]app.ConnectedServer
	adminUsers    map[string]app.AdminUser
	adminSessions map[string]app.AdminSession
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		items:         map[string]app.PairingCode{},
		entitlements:  map[string]app.EntitlementSnapshot{},
		servers:       map[string]app.ConnectedServer{},
		adminUsers:    map[string]app.AdminUser{},
		adminSessions: map[string]app.AdminSession{},
	}
}

func (r *memoryRepo) SaveEntitlements(_ context.Context, item app.EntitlementSnapshot) error {
	r.entitlements[item.TenantID+":"+item.ServerID] = item
	r.servers[item.TenantID+":"+item.ServerID] = app.ConnectedServer{TenantID: item.TenantID, ServerID: item.ServerID, FirstSeenAt: item.UpdatedAt, LastSeenAt: item.UpdatedAt, Snapshot: &item}
	return nil
}

func (r *memoryRepo) GetEntitlements(_ context.Context, tenantID, serverID string) (app.EntitlementSnapshot, error) {
	item, ok := r.entitlements[tenantID+":"+serverID]
	if !ok {
		return app.EntitlementSnapshot{}, app.ErrEntitlementNotFound
	}
	return item, nil
}

func (r *memoryRepo) ListEntitlements(_ context.Context) ([]app.EntitlementSnapshot, error) {
	items := make([]app.EntitlementSnapshot, 0, len(r.entitlements))
	for _, item := range r.entitlements {
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b app.EntitlementSnapshot) int {
		if a.TenantID == b.TenantID {
			return cmpString(a.ServerID, b.ServerID)
		}
		return cmpString(a.TenantID, b.TenantID)
	})
	return items, nil
}

func (r *memoryRepo) SaveConnectedServer(_ context.Context, item app.ConnectedServer) error {
	key := item.TenantID + ":" + item.ServerID
	if current, ok := r.servers[key]; ok {
		item.FirstSeenAt = current.FirstSeenAt
		item.Snapshot = current.Snapshot
	}
	r.servers[key] = item
	return nil
}

func (r *memoryRepo) ListConnectedServers(_ context.Context) ([]app.ConnectedServer, error) {
	items := make([]app.ConnectedServer, 0, len(r.servers))
	for _, item := range r.servers {
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b app.ConnectedServer) int {
		if a.TenantID == b.TenantID {
			return cmpString(a.ServerID, b.ServerID)
		}
		return cmpString(a.TenantID, b.TenantID)
	})
	return items, nil
}

func (r *memoryRepo) SaveAdminUser(_ context.Context, item app.AdminUser) error {
	r.adminUsers[item.Username] = item
	return nil
}

func (r *memoryRepo) GetAdminUser(_ context.Context, username string) (app.AdminUser, error) {
	item, ok := r.adminUsers[username]
	if !ok {
		return app.AdminUser{}, app.ErrAdminAuth
	}
	return item, nil
}

func (r *memoryRepo) SaveAdminSession(_ context.Context, item app.AdminSession) error {
	r.adminSessions[item.TokenHash] = item
	return nil
}

func (r *memoryRepo) GetAdminSession(_ context.Context, tokenHash string) (app.AdminSession, error) {
	item, ok := r.adminSessions[tokenHash]
	if !ok {
		return app.AdminSession{}, app.ErrAdminAuth
	}
	return item, nil
}

func (r *memoryRepo) DeleteAdminSession(_ context.Context, tokenHash string) error {
	delete(r.adminSessions, tokenHash)
	return nil
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
	list, err := service.ListEntitlements(t.Context())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !slices.ContainsFunc(list, func(item app.EntitlementSnapshot) bool {
		return item.TenantID == "tenant-1" && item.ServerID == "edge-1" && item.Version == 3
	}) {
		t.Fatalf("expected saved entitlement in list, got %+v", list)
	}
}

func TestBootstrapSuperAdminLoginSessionAndServerList(t *testing.T) {
	repo := newMemoryRepo()
	service := app.NewService(repo)
	if err := service.BootstrapSuperAdmin(t.Context(), "admin", "change-me-123"); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	if _, err := service.LoginAdmin(t.Context(), "admin", "wrong-password"); !errors.Is(err, app.ErrAdminAuth) {
		t.Fatalf("expected invalid login to be rejected, got %v", err)
	}
	login, err := service.LoginAdmin(t.Context(), "admin", "change-me-123")
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}
	if username, ok := service.ValidateAdminSession(t.Context(), login.Token); !ok || username != "admin" {
		t.Fatalf("expected valid admin session, username=%q ok=%v", username, ok)
	}
	if err := service.LogoutAdmin(t.Context(), login.Token); err != nil {
		t.Fatalf("logout admin: %v", err)
	}
	if _, ok := service.ValidateAdminSession(t.Context(), login.Token); ok {
		t.Fatal("expected logout to invalidate session")
	}
	if err := service.RecordServerSeen(t.Context(), "tenant-1", "cloud-1"); err != nil {
		t.Fatalf("record connected server: %v", err)
	}
	servers, err := service.ListConnectedServers(t.Context())
	if err != nil || len(servers) != 1 || servers[0].TenantID != "tenant-1" || servers[0].ServerID != "cloud-1" {
		t.Fatalf("unexpected connected servers: %+v err=%v", servers, err)
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

func cmpString(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
