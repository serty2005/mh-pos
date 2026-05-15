package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"license-server/internal/license/app"
)

type memoryRepo struct {
	items map[string]app.PairingCode
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{items: map[string]app.PairingCode{}}
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

func TestRegisterResolveAndConsumePairingCode(t *testing.T) {
	service := app.NewService(newMemoryRepo())
	ctx := context.Background()

	if _, err := service.Register(ctx, app.RegisterPairingCodeCommand{
		PairingCode:  "123456",
		CloudURL:     "http://localhost:8090",
		RestaurantID: "restaurant-1",
		NodeDeviceID: "edge-node-1",
		Credentials:  app.Credentials{Type: "node_token", Token: "token-1"},
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	resolved, err := service.Resolve(ctx, app.ResolveCommand{PairingCode: "123456", NodeDeviceID: "edge-node-1"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.CloudURL != "http://localhost:8090" || resolved.RestaurantID != "restaurant-1" || resolved.Credentials.Token != "token-1" {
		t.Fatalf("unexpected resolve result: %+v", resolved)
	}
	if _, err := service.Resolve(ctx, app.ResolveCommand{PairingCode: "123456", NodeDeviceID: "edge-node-1"}); !errors.Is(err, app.ErrConsumed) {
		t.Fatalf("expected consumed code to be rejected, got %v", err)
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
		CloudURL:     "http://localhost:8090",
		RestaurantID: "restaurant-1",
		NodeDeviceID: "edge-node-1",
		Credentials:  app.Credentials{Type: "node_token", Token: "token-1"},
		ExpiresAt:    time.Now().Add(-time.Minute),
	}); !errors.Is(err, app.ErrExpired) {
		t.Fatalf("expected expired code registration to be rejected, got %v", err)
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
