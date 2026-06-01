package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/provisioning/api"
)

type packageService struct {
	packages map[string]contracts.MasterDataPackage
}

func newPackageService() *packageService {
	return &packageService{packages: map[string]contracts.MasterDataPackage{}}
}

func (s *packageService) UpsertMasterDataPackage(_ context.Context, v contracts.MasterDataPackage) (contracts.MasterDataPackage, error) {
	s.packages[v.StreamName+"|"+v.NodeDeviceID] = v
	return v, nil
}

func (s *packageService) GetMasterDataPackage(_ context.Context, streamName, nodeDeviceID string) (contracts.MasterDataPackage, error) {
	v, ok := s.packages[streamName+"|"+nodeDeviceID]
	if !ok {
		return contracts.MasterDataPackage{}, contracts.ErrNotFound
	}
	return v, nil
}

func TestPricingPolicyMasterDataPackageRoutes(t *testing.T) {
	service := newPackageService()
	router := chi.NewRouter()
	router.Route("/api/v1", func(r chi.Router) {
		api.RegisterMasterDataPackageRoutes(r, service)
	})

	body := `{
		"node_device_id":"node-1",
		"restaurant_id":"restaurant-1",
		"sync_mode":"full_snapshot",
		"full_snapshot_reason":"terminal_restaurant_changed",
		"cloud_version":7,
		"payload_json":{
			"tax_profiles":[{"id":"tax-vat-10","name":"VAT 10","tax_exempt":false,"active":true}],
			"tax_rules":[{"id":"tax-rule-10","tax_profile_id":"tax-vat-10","name":"VAT 10","kind":"percentage","mode":"exclusive","rate_basis_points":1000,"active":true}],
			"service_charge_rules":[{"id":"svc-10","restaurant_id":"restaurant-1","name":"Service 10","kind":"service_charge","amount_kind":"percentage","value_basis_points":1000,"active":true}],
			"pricing_policies":[]
		}
	}`
	put := httptest.NewRecorder()
	router.ServeHTTP(put, httptest.NewRequest(http.MethodPut, "/api/v1/provisioning/master-data/pricing_policy", bytes.NewBufferString(body)))
	if put.Code != http.StatusOK {
		t.Fatalf("expected PUT 200, got %d: %s", put.Code, put.Body.String())
	}

	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/provisioning/master-data/pricing_policy?node_device_id=node-1", nil))
	if get.Code != http.StatusOK {
		t.Fatalf("expected GET 200, got %d: %s", get.Code, get.Body.String())
	}
	var got contracts.MasterDataPackage
	if err := json.Unmarshal(get.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.StreamName != contracts.MasterDataStreamPricing || got.NodeDeviceID != "node-1" || got.RestaurantID != "restaurant-1" || got.CloudVersion != 7 {
		t.Fatalf("unexpected package response: %+v", got)
	}
	if !strings.Contains(string(got.PayloadJSON), `"tax_profiles"`) || strings.Contains(get.Body.String(), "pin_hash") {
		t.Fatalf("unexpected package payload response: %s", get.Body.String())
	}
}
