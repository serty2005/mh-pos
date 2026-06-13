package licensehttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud-backend/internal/provisioning/app"
	"cloud-backend/internal/provisioning/domain"
)

func TestRegisterPairingCodePostsPayload(t *testing.T) {
	expiresAt := time.Date(2026, 6, 13, 12, 30, 0, 0, time.UTC)
	payload := app.LicensePairingPayload{
		PairingCode:  "123456",
		CloudURL:     "https://cloud.example.test",
		RestaurantID: "restaurant-1",
		NodeDeviceID: "edge-node-1",
		Credentials:  domain.Credentials{Type: "node_token", Token: "token-1"},
		ExpiresAt:    expiresAt,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/pairing-codes" {
			t.Fatalf("expected pairing-codes path, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected JSON content type, got %q", got)
		}

		var got app.LicensePairingPayload
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if got.PairingCode != payload.PairingCode ||
			got.CloudURL != payload.CloudURL ||
			got.RestaurantID != payload.RestaurantID ||
			got.NodeDeviceID != payload.NodeDeviceID ||
			got.Credentials != payload.Credentials ||
			!got.ExpiresAt.Equal(payload.ExpiresAt) {
			t.Fatalf("unexpected payload: %+v", got)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(" " + server.URL + "/// ")
	if err := client.RegisterPairingCode(context.Background(), payload); err != nil {
		t.Fatalf("RegisterPairingCode returned error: %v", err)
	}
}

func TestRegisterPairingCodeReturnsUnavailable(t *testing.T) {
	payload := app.LicensePairingPayload{PairingCode: "123456"}

	t.Run("nil client", func(t *testing.T) {
		var client *Client
		if err := client.RegisterPairingCode(context.Background(), payload); !errors.Is(err, domain.ErrLicenseServerUnavailable) {
			t.Fatalf("expected license server unavailable, got %v", err)
		}
	})

	t.Run("empty base URL", func(t *testing.T) {
		client := NewClient("   ")
		if err := client.RegisterPairingCode(context.Background(), payload); !errors.Is(err, domain.ErrLicenseServerUnavailable) {
			t.Fatalf("expected license server unavailable, got %v", err)
		}
	})

	t.Run("non-success status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		if err := client.RegisterPairingCode(context.Background(), payload); !errors.Is(err, domain.ErrLicenseServerUnavailable) {
			t.Fatalf("expected license server unavailable, got %v", err)
		}
	})
}
