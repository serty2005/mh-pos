package licensegate_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mh-pos-platform/licensegate"
)

func TestClientEnableDisableRevokeAndExpiry(t *testing.T) {
	status, enabled, expires := "active", true, time.Now().Add(time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tenant_id":"tenant","server_id":"edge","version":1,"status":"` + status + `","entitlements":{"kitchen-space":` + map[bool]string{true: "true", false: "false"}[enabled] + `},"issued_at":"2026-06-20T00:00:00Z","expires_at":"` + expires.UTC().Format(time.RFC3339Nano) + `"}`))
	}))
	defer server.Close()
	client := licensegate.NewClient(server.URL, "tenant", "edge", 0)

	if err := client.Require(t.Context(), licensegate.KitchenSpace); err != nil {
		t.Fatal(err)
	}
	enabled = false
	if err := client.Require(t.Context(), licensegate.KitchenSpace); !errors.Is(err, licensegate.ErrDenied) {
		t.Fatalf("disable: %v", err)
	}
	enabled, status = true, "revoked"
	if err := client.Require(t.Context(), licensegate.KitchenSpace); !errors.Is(err, licensegate.ErrDenied) {
		t.Fatalf("revoke: %v", err)
	}
	status, expires = "active", time.Now().Add(-time.Second)
	if err := client.Require(t.Context(), licensegate.KitchenSpace); !errors.Is(err, licensegate.ErrDenied) {
		t.Fatalf("expiry: %v", err)
	}
}

func TestClientUsesBoundedGraceOnlyWhenAuthorityUnavailable(t *testing.T) {
	expires := time.Now().Add(50 * time.Millisecond)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tenant_id":"tenant","server_id":"edge","version":1,"status":"active","entitlements":{"table-mode":true},"issued_at":"2026-06-20T00:00:00Z","expires_at":"` + expires.UTC().Format(time.RFC3339Nano) + `"}`))
	}))
	client := licensegate.NewClient(server.URL, "tenant", "edge", time.Second)
	if _, err := client.Current(context.Background()); err != nil {
		t.Fatal(err)
	}
	server.Close()
	time.Sleep(75 * time.Millisecond)
	if _, err := client.Current(context.Background()); err != nil {
		t.Fatalf("grace: %v", err)
	}
}
