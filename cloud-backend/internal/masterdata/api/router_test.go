package api_test

import (
	"bytes"
	"context"
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
	"cloud-backend/internal/masterdata/domain"
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
	role := post(t, router, "/api/v1/master-data/roles", `{"name":"cashier","permissions_json":"{}"}`)
	if role.Code != http.StatusCreated {
		t.Fatalf("expected role created, got %d: %s", role.Code, role.Body.String())
	}
	var roleBody struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(role.Body.Bytes(), &roleBody); err != nil {
		t.Fatal(err)
	}
	employee := post(t, router, "/api/v1/master-data/employees", `{"restaurant_ids":["restaurant-1"],"role_id":"`+roleBody.ID+`","name":"Anna","pin":"1111"}`)
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
	role := post(t, router, "/api/v1/master-data/roles", `{"name":"cashier","permissions_json":"{}"}`)
	var roleBody struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(role.Body.Bytes(), &roleBody)
	_ = post(t, router, "/api/v1/master-data/employees", `{"restaurant_ids":["restaurant-1"],"role_id":"`+roleBody.ID+`","name":"Anna","pin":"1111"}`)
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

func TestRestaurantPublicationStateReturnsNullBeforeFirstPublish(t *testing.T) {
	router := newRouter()
	current := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/restaurants/restaurant-1/master-data/publication-state", nil)
	router.ServeHTTP(current, req)
	if current.Code != http.StatusOK {
		t.Fatalf("expected optional publication state empty response, got %d: %s", current.Code, current.Body.String())
	}
	if strings.TrimSpace(current.Body.String()) != "null" {
		t.Fatalf("expected JSON null publication state, got %q", current.Body.String())
	}
}

func TestProductionRestaurantPublishAndSnapshotEndpoints(t *testing.T) {
	router := newRouter()
	restaurant := post(t, router, "/api/v1/restaurants", `{"name":"Demo Bistro","timezone":"Europe/Moscow","currency":"RUB","business_day_mode":"standard","business_day_boundary_local_time":"04:00"}`)
	if restaurant.Code != http.StatusCreated {
		t.Fatalf("expected restaurant created, got %d: %s", restaurant.Code, restaurant.Body.String())
	}
	var restaurantBody struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(restaurant.Body.Bytes(), &restaurantBody)
	role := post(t, router, "/api/v1/roles", `{"name":"cashier","permissions_json":"{}"}`)
	var roleBody struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(role.Body.Bytes(), &roleBody)
	_ = post(t, router, "/api/v1/employees", `{"restaurant_ids":["`+restaurantBody.ID+`"],"role_id":"`+roleBody.ID+`","name":"Anna","pin":"1111"}`)
	catalog := post(t, router, "/api/v1/catalog/items", `{"restaurant_id":"`+restaurantBody.ID+`","type":"dish","name":"Tea","sku":"TEA","base_unit":"portion"}`)
	var catalogBody struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(catalog.Body.Bytes(), &catalogBody)
	_ = post(t, router, "/api/v1/menu/items", `{"restaurant_id":"`+restaurantBody.ID+`","catalog_item_id":"`+catalogBody.ID+`","name":"Tea","price":1000,"currency":"RUB"}`)

	pub := post(t, router, "/api/v1/restaurants/"+restaurantBody.ID+"/master-data/publish", `{"published_by":"operator-1"}`)
	if pub.Code != http.StatusCreated {
		t.Fatalf("expected publish created, got %d: %s", pub.Code, pub.Body.String())
	}
	snapshot := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/restaurants/"+restaurantBody.ID+"/edge-nodes/node-1/master-data/snapshot", nil)
	router.ServeHTTP(snapshot, req)
	if snapshot.Code != http.StatusOK {
		t.Fatalf("expected snapshot, got %d: %s", snapshot.Code, snapshot.Body.String())
	}
	body := snapshot.Body.String()
	for _, required := range []string{`"node_device_id":"node-1"`, `"restaurants"`, `"roles"`, `"employees"`, `"catalog_items"`, `"menu_items"`} {
		if !strings.Contains(body, required) {
			t.Fatalf("expected snapshot to contain %s: %s", required, body)
		}
	}
	var packet struct {
		Restaurants []struct {
			ID     string `json:"id"`
			Active bool   `json:"active"`
		} `json:"restaurants"`
	}
	if err := json.Unmarshal(snapshot.Body.Bytes(), &packet); err != nil {
		t.Fatal(err)
	}
	if len(packet.Restaurants) != 1 || packet.Restaurants[0].ID != restaurantBody.ID || !packet.Restaurants[0].Active {
		t.Fatalf("expected active restaurant in snapshot, got %+v", packet.Restaurants)
	}
	if strings.Contains(body, "1111") {
		t.Fatalf("snapshot leaked raw PIN: %s", body)
	}
}

func TestStopListUpdateReviewRoutesDoNotExposeRawPayload(t *testing.T) {
	router, repo := newRouterWithRepo()
	now := fixedClock{}.Now()
	repo.SeedStopListUpdateReview(domain.StopListUpdateReview{
		ID:               "event-stop-api-1",
		RestaurantID:     "restaurant-1",
		DeviceID:         "edge-1",
		StopListID:       "stop-api-1",
		CatalogItemID:    "dish-1",
		Active:           true,
		ConflictPolicy:   "edge_overlay_requires_manager_review",
		Source:           "edge",
		Reason:           "sold out",
		ProjectionAction: "requires_manager_review",
		Status:           domain.SuggestionStatusPending,
		UpdatedAt:        now,
		OccurredAt:       now.Add(-time.Minute),
		ProjectedAt:      now,
		CreatedAt:        now,
	})

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/manager/stop-list-updates?restaurant_id=restaurant-1&status=pending", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), "event-stop-api-1") || strings.Contains(list.Body.String(), "payload_json") || strings.Contains(list.Body.String(), "raw_payload") {
		t.Fatalf("list must expose safe DTO only, got %s", list.Body.String())
	}
	detail := httptest.NewRecorder()
	router.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/api/v1/manager/stop-list-updates/event-stop-api-1", nil))
	if detail.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d: %s", detail.Code, detail.Body.String())
	}
	rejected := post(t, router, "/api/v1/manager/stop-list-updates/event-stop-api-1/reject", `{"reviewed_by_employee_id":"manager-1","review_comment":"not approved"}`)
	if rejected.Code != http.StatusOK {
		t.Fatalf("expected reject 200, got %d: %s", rejected.Code, rejected.Body.String())
	}
	replayed := post(t, router, "/api/v1/manager/stop-list-updates/event-stop-api-1/reject", `{"reviewed_by_employee_id":"manager-1"}`)
	if replayed.Code != http.StatusOK {
		t.Fatalf("expected idempotent reject 200, got %d: %s", replayed.Code, replayed.Body.String())
	}
	if strings.Contains(replayed.Body.String(), "payload_json") || strings.Contains(replayed.Body.String(), "raw_payload") {
		t.Fatalf("review response leaked raw payload field: %s", replayed.Body.String())
	}
}

func TestManagerReviewAssignmentRoutes(t *testing.T) {
	router, repo := newRouterWithRepo()
	now := fixedClock{}.Now()
	repo.SeedStopListUpdateReview(domain.StopListUpdateReview{
		ID:               "event-stop-api-assign-1",
		RestaurantID:     "restaurant-1",
		DeviceID:         "edge-1",
		StopListID:       "stop-api-assign-1",
		CatalogItemID:    "dish-1",
		Active:           true,
		ConflictPolicy:   "edge_overlay_requires_manager_review",
		Source:           "edge",
		Reason:           "sold out",
		ProjectionAction: "requires_manager_review",
		Status:           domain.SuggestionStatusPending,
		UpdatedAt:        now,
		OccurredAt:       now.Add(-time.Minute),
		ProjectedAt:      now,
		CreatedAt:        now,
	})

	assigned := post(t, router, "/api/v1/manager/stop-list-updates/event-stop-api-assign-1/assign", `{"command_id":"018f0000-0000-7000-8000-000000000801","assigned_to_employee_id":"manager-2","assigned_by_employee_id":"manager-1","reason":"take ownership"}`)
	if assigned.Code != http.StatusOK {
		t.Fatalf("expected assign 200, got %d: %s", assigned.Code, assigned.Body.String())
	}
	if !strings.Contains(assigned.Body.String(), `"assigned_to_employee_id":"manager-2"`) || strings.Contains(assigned.Body.String(), "payload_json") || strings.Contains(assigned.Body.String(), "raw_payload") {
		t.Fatalf("assignment response must be safe, got %s", assigned.Body.String())
	}
	replayedAssign := post(t, router, "/api/v1/manager/stop-list-updates/event-stop-api-assign-1/assign", `{"command_id":"018f0000-0000-7000-8000-000000000801","assigned_to_employee_id":"manager-2","assigned_by_employee_id":"manager-1","reason":"take ownership"}`)
	if replayedAssign.Code != http.StatusOK || len(repo.ReviewAssignmentAuditEvents()) != 1 {
		t.Fatalf("expected idempotent assign replay, code=%d body=%s audit=%+v", replayedAssign.Code, replayedAssign.Body.String(), repo.ReviewAssignmentAuditEvents())
	}
	unassigned := post(t, router, "/api/v1/manager/stop-list-updates/event-stop-api-assign-1/unassign", `{"command_id":"018f0000-0000-7000-8000-000000000802","unassigned_by_employee_id":"manager-1","reason":"rebalance queue"}`)
	if unassigned.Code != http.StatusOK {
		t.Fatalf("expected unassign 200, got %d: %s", unassigned.Code, unassigned.Body.String())
	}
	if strings.Contains(unassigned.Body.String(), "assigned_to_employee_id") || strings.Contains(unassigned.Body.String(), "raw_payload") {
		t.Fatalf("unassignment response must be safe and cleared, got %s", unassigned.Body.String())
	}
	audit := httptest.NewRecorder()
	router.ServeHTTP(audit, httptest.NewRequest(http.MethodGet, "/api/v1/manager/stop-list-updates/event-stop-api-assign-1/audit?limit=1000&offset=-1", nil))
	if audit.Code != http.StatusOK {
		t.Fatalf("expected audit 200, got %d: %s", audit.Code, audit.Body.String())
	}
	var auditEvents []app.ReviewAssignmentAuditEventResponse
	if err := json.Unmarshal(audit.Body.Bytes(), &auditEvents); err != nil {
		t.Fatal(err)
	}
	if len(auditEvents) != 2 {
		t.Fatalf("expected assigned/unassigned audit events, got %s", audit.Body.String())
	}
	actions := map[string]bool{}
	for _, event := range auditEvents {
		actions[event.Action] = true
		if event.ReviewType != "stop_list_update" || event.ReviewID != "event-stop-api-assign-1" || event.CommandID == "" {
			t.Fatalf("unexpected audit event: %+v", event)
		}
	}
	if !actions["assigned"] || !actions["unassigned"] {
		t.Fatalf("expected assigned/unassigned audit actions, got %+v", auditEvents)
	}
	for _, forbidden := range []string{"payload_json", "raw_payload", "sync_payload", "envelope", "request_dump", "token", "pin", "sql"} {
		if strings.Contains(strings.ToLower(audit.Body.String()), forbidden) {
			t.Fatalf("audit response must not expose %s: %s", forbidden, audit.Body.String())
		}
	}
	invalidCommand := post(t, router, "/api/v1/manager/stop-list-updates/event-stop-api-assign-1/assign", `{"command_id":"not-uuid-v7","assigned_to_employee_id":"manager-2","assigned_by_employee_id":"manager-1","reason":"take ownership"}`)
	if invalidCommand.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid UUIDv7 command id to return 400, got %d: %s", invalidCommand.Code, invalidCommand.Body.String())
	}
}

func TestCatalogSuggestionAuditRouteReturnsExpectedEvents(t *testing.T) {
	router, repo := newRouterWithRepo()
	now := fixedClock{}.Now()
	events := []domain.ReviewAssignmentAuditEvent{
		{
			EventID:          "api-catalog-audit-001",
			CommandID:        "api-catalog-command-001",
			ReviewType:       "catalog_suggestion",
			ReviewID:         "catalog-api-audit-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "take catalog review",
			OccurredAt:       now,
		},
		{
			EventID:          "api-catalog-audit-002",
			CommandID:        "api-catalog-command-002",
			ReviewType:       "catalog_suggestion",
			ReviewID:         "catalog-api-audit-1",
			Action:           "unassigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "rebalance catalog queue",
			OccurredAt:       now.Add(time.Minute),
		},
		{
			EventID:          "api-catalog-audit-other-type",
			CommandID:        "api-catalog-command-other-type",
			ReviewType:       "recipe_suggestion",
			ReviewID:         "catalog-api-audit-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-9",
			TargetEmployeeID: "manager-8",
			OccurredAt:       now.Add(2 * time.Minute),
		},
	}
	for _, event := range events {
		if err := repo.AppendReviewAssignmentAuditEvent(t.Context(), event); err != nil {
			t.Fatal(err)
		}
	}

	audit := httptest.NewRecorder()
	router.ServeHTTP(audit, httptest.NewRequest(http.MethodGet, "/api/v1/manager/catalog-suggestions/catalog-api-audit-1/audit?limit=1000&offset=-1", nil))
	if audit.Code != http.StatusOK {
		t.Fatalf("expected catalog audit 200, got %d: %s", audit.Code, audit.Body.String())
	}
	var auditEvents []app.ReviewAssignmentAuditEventResponse
	if err := json.Unmarshal(audit.Body.Bytes(), &auditEvents); err != nil {
		t.Fatal(err)
	}
	if len(auditEvents) != 2 || auditEvents[0].EventID != "api-catalog-audit-002" || auditEvents[1].EventID != "api-catalog-audit-001" {
		t.Fatalf("expected catalog audit events newest-first, got %+v", auditEvents)
	}
	for _, event := range auditEvents {
		if event.ReviewType != "catalog_suggestion" || event.ReviewID != "catalog-api-audit-1" || event.CommandID == "" {
			t.Fatalf("unexpected catalog audit event: %+v", event)
		}
	}
	assertNoRawAuditFields(t, audit.Body.String())
}

func TestRecipeSuggestionAuditRouteReturnsExpectedEvents(t *testing.T) {
	router, repo := newRouterWithRepo()
	now := fixedClock{}.Now()
	events := []domain.ReviewAssignmentAuditEvent{
		{
			EventID:          "api-recipe-audit-001",
			CommandID:        "api-recipe-command-001",
			ReviewType:       "recipe_suggestion",
			ReviewID:         "recipe-api-audit-1",
			Action:           "assigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "take recipe review",
			OccurredAt:       now,
		},
		{
			EventID:          "api-recipe-audit-002",
			CommandID:        "api-recipe-command-002",
			ReviewType:       "recipe_suggestion",
			ReviewID:         "recipe-api-audit-1",
			Action:           "unassigned",
			ActorEmployeeID:  "manager-1",
			TargetEmployeeID: "manager-2",
			Reason:           "rebalance recipe queue",
			OccurredAt:       now.Add(time.Minute),
		},
		{
			EventID:          "api-recipe-audit-other-review",
			CommandID:        "api-recipe-command-other-review",
			ReviewType:       "recipe_suggestion",
			ReviewID:         "recipe-api-audit-other",
			Action:           "assigned",
			ActorEmployeeID:  "manager-9",
			TargetEmployeeID: "manager-8",
			OccurredAt:       now.Add(2 * time.Minute),
		},
	}
	for _, event := range events {
		if err := repo.AppendReviewAssignmentAuditEvent(t.Context(), event); err != nil {
			t.Fatal(err)
		}
	}

	audit := httptest.NewRecorder()
	router.ServeHTTP(audit, httptest.NewRequest(http.MethodGet, "/api/v1/manager/recipe-suggestions/recipe-api-audit-1/audit", nil))
	if audit.Code != http.StatusOK {
		t.Fatalf("expected recipe audit 200, got %d: %s", audit.Code, audit.Body.String())
	}
	var auditEvents []app.ReviewAssignmentAuditEventResponse
	if err := json.Unmarshal(audit.Body.Bytes(), &auditEvents); err != nil {
		t.Fatal(err)
	}
	if len(auditEvents) != 2 || auditEvents[0].EventID != "api-recipe-audit-002" || auditEvents[1].EventID != "api-recipe-audit-001" {
		t.Fatalf("expected recipe audit events newest-first, got %+v", auditEvents)
	}
	for _, event := range auditEvents {
		if event.ReviewType != "recipe_suggestion" || event.ReviewID != "recipe-api-audit-1" || event.CommandID == "" {
			t.Fatalf("unexpected recipe audit event: %+v", event)
		}
	}
	assertNoRawAuditFields(t, audit.Body.String())
}

func assertNoRawAuditFields(t *testing.T, body string) {
	t.Helper()
	for _, forbidden := range []string{"payload_json", "raw_payload", "sync_payload", "envelope", "request_dump", "token", "pin", "sql"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("audit response must not expose %s: %s", forbidden, body)
		}
	}
}

func newRouter() http.Handler {
	router, _ := newRouterWithRepo()
	return router
}

func newRouterWithRepo() (http.Handler, *memory.Repository) {
	r := chi.NewRouter()
	repo := memory.NewRepository()
	now := fixedClock{}.Now()
	_, _ = repo.CreateRestaurant(context.Background(), domain.Restaurant{ID: "restaurant-1", Name: "Test", Timezone: "Europe/Moscow", Currency: "RUB", BusinessDayMode: "standard", BusinessDayBoundaryLocalTime: "05:00", Status: domain.RestaurantActive, CloudVersion: 1, CreatedAt: now, UpdatedAt: now})
	service := app.NewService(repo, fixedClock{}, &fixedIDs{})
	r.Route("/api/v1", func(r chi.Router) {
		api.RegisterRoutes(r, service)
	})
	return r, repo
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
