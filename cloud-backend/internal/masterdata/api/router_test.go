package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"cloud-backend/internal/masterdata/api"
	"cloud-backend/internal/masterdata/app"
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
	return "api-id-" + strconv.Itoa(f.next)
}

func TestEmployeeEndpointsDoNotExposePINMaterial(t *testing.T) {
	router := newRouter()
	role := post(t, router, "/api/v1/master-data/roles", `{"restaurant_id":"restaurant-1","name":"cashier","permissions_json":"{}"}`)
	if role.Code != http.StatusCreated {
		t.Fatalf("expected role created, got %d: %s", role.Code, role.Body.String())
	}
	var roleBody struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(role.Body.Bytes(), &roleBody); err != nil {
		t.Fatal(err)
	}
	employee := post(t, router, "/api/v1/master-data/employees", `{"restaurant_id":"restaurant-1","role_id":"`+roleBody.ID+`","name":"Anna","pin":"1111"}`)
	if employee.Code != http.StatusCreated {
		t.Fatalf("expected employee created, got %d: %s", employee.Code, employee.Body.String())
	}
	body := employee.Body.String()
	if strings.Contains(body, "pin_hash") || strings.Contains(body, "1111") || strings.Contains(body, "pbkdf2") {
		t.Fatalf("employee response leaked PIN material: %s", body)
	}
	rotated := post(t, router, "/api/v1/master-data/employees/api-id-2/pin", `{"pin":"2222"}`)
	if rotated.Code != http.StatusOK {
		t.Fatalf("expected pin rotation, got %d: %s", rotated.Code, rotated.Body.String())
	}
	if strings.Contains(rotated.Body.String(), "pin_hash") || strings.Contains(rotated.Body.String(), "2222") || strings.Contains(rotated.Body.String(), "pbkdf2") {
		t.Fatalf("pin rotation response leaked PIN material: %s", rotated.Body.String())
	}
}

func TestPublicationEndpointsReturnSummary(t *testing.T) {
	router := newRouter()
	role := post(t, router, "/api/v1/master-data/roles", `{"restaurant_id":"restaurant-1","name":"cashier","permissions_json":"{}"}`)
	var roleBody struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(role.Body.Bytes(), &roleBody)
	_ = post(t, router, "/api/v1/master-data/employees", `{"restaurant_id":"restaurant-1","role_id":"`+roleBody.ID+`","name":"Anna","pin":"1111"}`)
	catalog := post(t, router, "/api/v1/master-data/catalog/items", `{"restaurant_id":"restaurant-1","kind":"dish","name":"Tea","sku":"TEA","base_unit":"portion"}`)
	var catalogBody struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(catalog.Body.Bytes(), &catalogBody)
	published := `"published"`
	patch(t, router, "/api/v1/master-data/catalog/items/"+catalogBody.ID, `{"status":`+published+`}`)
	menu := post(t, router, "/api/v1/master-data/menu/items", `{"restaurant_id":"restaurant-1","catalog_item_id":"`+catalogBody.ID+`","name":"Tea","price":1000,"currency":"RUB"}`)
	var menuBody struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(menu.Body.Bytes(), &menuBody)
	patch(t, router, "/api/v1/master-data/menu/items/"+menuBody.ID, `{"status":`+published+`}`)

	pub := post(t, router, "/api/v1/master-data/publications", `{"restaurant_id":"restaurant-1","published_by":"operator-1"}`)
	if pub.Code != http.StatusCreated {
		t.Fatalf("expected publication created, got %d: %s", pub.Code, pub.Body.String())
	}
	if strings.Contains(pub.Body.String(), "pin_hash") || strings.Contains(pub.Body.String(), "pbkdf2") || strings.Contains(pub.Body.String(), "package_json") {
		t.Fatalf("publication response should be summary without package/PIN material: %s", pub.Body.String())
	}
	current := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/master-data/published?restaurant_id=restaurant-1", nil)
	router.ServeHTTP(current, req)
	if current.Code != http.StatusOK {
		t.Fatalf("expected current published state, got %d: %s", current.Code, current.Body.String())
	}
	if !strings.Contains(current.Body.String(), `"version":1`) {
		t.Fatalf("unexpected current publication: %s", current.Body.String())
	}
}

func newRouter() http.Handler {
	r := chi.NewRouter()
	service := app.NewService(memory.NewRepository(), fixedClock{}, &fixedIDs{})
	r.Route("/api/v1", func(r chi.Router) {
		api.RegisterRoutes(r, service)
	})
	return r
}

func post(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func patch(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
