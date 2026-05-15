package provisioninghttp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pos-backend/internal/pos/app/provisioning"
	"pos-backend/internal/pos/domain"
)

func TestLicenseResolveMapsPairingCodeInvalidToDomainInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/pairing-codes/resolve" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("X-Error-Code", "PAIRING_CODE_INVALID")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"PAIRING_CODE_INVALID","message_key":"errors.pairing.invalid"}}`))
	}))
	defer server.Close()

	client := NewLicenseClient(time.Second)
	_, err := client.Resolve(context.Background(), server.URL, provisioning.LicenseResolveRequest{
		PairingCode:  "123456",
		NodeDeviceID: "edge-node-1",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("expected domain invalid for license 400, got %v", err)
	}
}
