package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud-backend/internal/cloudsync/api"
	"cloud-backend/internal/cloudsync/app"
	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/cloudsync/infra/memory"
	"mh-pos-platform/licensegate"
)

type fixedClock struct{}

type deniedModuleGate struct {
	moduleID string
}

func (g deniedModuleGate) Require(_ context.Context, moduleID string) error {
	if moduleID == g.moduleID {
		return licensegate.ErrDenied
	}
	return nil
}

func (deniedModuleGate) Current(context.Context) (licensegate.Snapshot, error) {
	return licensegate.Snapshot{}, errors.New("not used")
}

func (fixedClock) Now() time.Time {
	return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
}

func TestPostDuplicateEnvelopeDoesNotCreateDuplicateReceipt(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	raw := []byte(`{
	  "version":"1",
	  "event_id":"018f0000-0000-7000-8000-000000000001",
	  "command_id":"command-1",
	  "event_type":"OrderCreated",
	  "aggregate_type":"Order",
	  "aggregate_id":"order-1",
	  "restaurant_id":"restaurant-1",
	  "device_id":"device-1",
	  "shift_id":"shift-1",
	  "occurred_at":"2026-05-05T09:00:00Z",
	  "payload":{
	    "origin":"edge_device",
	    "data":{
	      "id":"order-1",
	      "edge_order_id":"edge-order-1",
	      "restaurant_id":"restaurant-1",
	      "device_id":"device-1",
	      "shift_id":"shift-1",
	      "status":"open",
	      "table_name":"A1",
	      "guest_count":2,
	      "opened_at":"2026-05-05T09:00:00Z",
	      "created_at":"2026-05-05T09:00:00Z",
	      "updated_at":"2026-05-05T09:00:00Z"
	    }
	  }
	}`)

	first := postEnvelope(t, router, raw)
	second := postEnvelope(t, router, raw)
	if first != second {
		t.Fatalf("expected stable ack on replay\nfirst=%+v\nsecond=%+v", first, second)
	}
	if repo.Count() != 1 {
		t.Fatalf("expected one business receipt, got %d", repo.Count())
	}
	if got := string(repo.RawPayload(first.CloudReceiptID)); got != string(bytes.TrimSpace(raw)) {
		t.Fatalf("raw payload was not preserved\nwant=%s\ngot=%s", bytes.TrimSpace(raw), got)
	}
}

func TestPostBatchEdgeEventsReturnsItemLevelAck(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	valid := []byte(`{
	  "version":"1",
	  "event_id":"018f0000-0000-7000-8000-000000000001",
	  "command_id":"command-1",
	  "event_type":"OrderCreated",
	  "aggregate_type":"Order",
	  "aggregate_id":"order-1",
	  "restaurant_id":"restaurant-1",
	  "device_id":"device-1",
	  "shift_id":"shift-1",
	  "occurred_at":"2026-05-05T09:00:00Z",
	  "payload":{"origin":"edge_device","data":{"id":"order-1","edge_order_id":"edge-order-1","restaurant_id":"restaurant-1","device_id":"device-1","shift_id":"shift-1","status":"open","table_name":"A1","guest_count":2,"opened_at":"2026-05-05T09:00:00Z","created_at":"2026-05-05T09:00:00Z","updated_at":"2026-05-05T09:00:00Z"}}
	}`)
	reqBody := []byte(`{"items":[` + string(valid) + `,{"version":"1"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/edge-events/batch", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var ack contracts.BatchEventAck
	if err := json.Unmarshal(rec.Body.Bytes(), &ack); err != nil {
		t.Fatal(err)
	}
	if ack.Status != "partial" || len(ack.Items) != 2 {
		t.Fatalf("unexpected batch ack: %+v", ack)
	}
	if ack.Items[0].Status != contracts.BatchItemAccepted {
		t.Fatalf("expected first item accepted, got %+v", ack.Items[0])
	}
	if ack.Items[1].Status != contracts.BatchItemRejected || ack.Items[1].ErrorCode != "INVALID_ENVELOPE" {
		t.Fatalf("expected second item rejected by validation, got %+v", ack.Items[1])
	}
}

func TestPostExchangeRequiresNodeTokenAndReturnsPackage(t *testing.T) {
	repo := memory.NewRepository()
	if err := repo.AuthorizeNodeForTest("node-1", "restaurant-1", "node-token"); err != nil {
		t.Fatal(err)
	}
	service := app.NewService(repo, fixedClock{})
	if _, err := service.UpsertMasterDataPackage(t.Context(), contracts.MasterDataPackage{
		StreamName:      contracts.MasterDataStreamCatalog,
		NodeDeviceID:    "node-1",
		RestaurantID:    "restaurant-1",
		SyncMode:        contracts.SyncModeIncremental,
		CloudVersion:    8,
		CheckpointToken: "catalog:8",
		PayloadJSON:     json.RawMessage(`{"catalog_items":[{"id":"cat-1","name":"Tea"}]}`),
	}); err != nil {
		t.Fatal(err)
	}
	router := api.NewRouter(service)

	body := []byte(`{
		"protocol_version":"sync_exchange.v1",
		"node_device_id":"node-1",
		"restaurant_id":"restaurant-1",
		"edge_events":[],
		"streams":[{"stream_name":"catalog","last_cloud_version":7,"checkpoint_token":"catalog:7"}]
	}`)
	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/api/v1/sync/exchange", bytes.NewReader(body))
	unauthorizedRec := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedRec, unauthorizedReq)
	if unauthorizedRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without node token, got %d: %s", unauthorizedRec.Code, unauthorizedRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/exchange", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer node-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 exchange, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp contracts.SyncExchangeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.CloudPackages) != 1 || resp.CloudPackages[0].CloudVersion != 8 {
		t.Fatalf("expected catalog package in exchange response, got %+v", resp)
	}
}

func TestExchangeOmitsCloudPackageForDisabledModule(t *testing.T) {
	repo := memory.NewRepository()
	if err := repo.AuthorizeNodeForTest("node-1", "restaurant-1", "node-token"); err != nil {
		t.Fatal(err)
	}
	service := app.NewService(repo, fixedClock{})
	packages := []contracts.MasterDataPackage{
		{
			StreamName:      contracts.MasterDataStreamCatalog,
			NodeDeviceID:    "node-1",
			RestaurantID:    "restaurant-1",
			SyncMode:        contracts.SyncModeIncremental,
			CloudVersion:    2,
			CheckpointToken: "catalog:2",
			PayloadJSON:     json.RawMessage(`{"catalog_items":[]}`),
		},
		{
			StreamName:      contracts.MasterDataStreamRecipes,
			NodeDeviceID:    "node-1",
			RestaurantID:    "restaurant-1",
			SyncMode:        contracts.SyncModeIncremental,
			CloudVersion:    3,
			CheckpointToken: "recipes:3",
			PayloadJSON:     json.RawMessage(`{"recipe_versions":[],"recipe_lines":[]}`),
		},
	}
	for _, pkg := range packages {
		if _, err := service.UpsertMasterDataPackage(t.Context(), pkg); err != nil {
			t.Fatal(err)
		}
	}
	router := api.NewRouterWithProvisioningOLAPAndLicense(service, nil, nil, deniedModuleGate{moduleID: licensegate.KitchenSpace})
	body := []byte(`{
		"protocol_version":"sync_exchange.v1",
		"node_device_id":"node-1",
		"restaurant_id":"restaurant-1",
		"edge_events":[],
		"streams":[
			{"stream_name":"catalog","last_cloud_version":1},
			{"stream_name":"recipes","last_cloud_version":1}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/exchange", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer node-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 exchange, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp contracts.SyncExchangeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.CloudPackages) != 1 || resp.CloudPackages[0].StreamName != contracts.MasterDataStreamCatalog {
		t.Fatalf("expected only unlicensed-independent catalog package, got %+v", resp.CloudPackages)
	}
}

func TestReceiptPreviewReturnsSVG(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	body := `{"template_content":"{a:center}{{.title}}\n{qr:size=4:{{.qr_payload}}}","document_type":"precheck","cpl":48,"print_context":{"title":"ПРЕДЧЕК","qr_payload":"MHT1:019"}}`
	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts/preview", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/svg+xml" {
		t.Fatalf("unexpected content type %q", got)
	}
	if !strings.Contains(rec.Body.String(), `<svg `) || !strings.Contains(rec.Body.String(), `ПРЕДЧЕК`) || !strings.Contains(rec.Body.String(), `data-size="4"`) {
		t.Fatalf("unexpected svg: %s", rec.Body.String())
	}
}

func TestReceiptPreviewBadTemplateReturnsParseError(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	body := `{"template_content":"{qr:}","document_type":"precheck","cpl":48,"print_context":{}}`
	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts/preview", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || rec.Header().Get("X-Error-Code") != "TEMPLATE_PARSE_ERROR" {
		t.Fatalf("expected TEMPLATE_PARSE_ERROR, got %d %s: %s", rec.Code, rec.Header().Get("X-Error-Code"), rec.Body.String())
	}
}

func TestReceiptPreviewBadContextReturnsSchemaError(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	body := `{"template_content":"ok","document_type":"precheck","cpl":48,"print_context":[]}`
	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts/preview", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || rec.Header().Get("X-Error-Code") != "CONTEXT_SCHEMA_ERROR" {
		t.Fatalf("expected CONTEXT_SCHEMA_ERROR, got %d %s: %s", rec.Code, rec.Header().Get("X-Error-Code"), rec.Body.String())
	}
}

func TestReceiptPreviewDoesNotRequireModuleGate(t *testing.T) {
	repo := memory.NewRepository()
	service := app.NewService(repo, fixedClock{})
	router := api.NewRouterWithProvisioningOLAPAndLicense(service, nil, nil, deniedModuleGate{moduleID: licensegate.CheckerFlow})
	body := `{"template_content":"ok","document_type":"precheck","cpl":48,"print_context":{}}`
	req := httptest.NewRequest(http.MethodGet, "/api/v1/receipts/preview", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected preview to stay available without checker-flow, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProvisioningMasterDataPutAndGet(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	putBody := []byte(`{
	  "node_device_id":"node-1",
	  "restaurant_id":"restaurant-1",
	  "sync_mode":"full_snapshot",
	  "full_snapshot_reason":"terminal_restaurant_changed",
	  "cloud_version":12,
	  "payload_json":{"catalog_items":[{"id":"cat-1","name":"Tea"}]}
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/provisioning/master-data/catalog", bytes.NewReader(putBody))
	putRec := httptest.NewRecorder()
	router.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on upsert, got %d: %s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/provisioning/master-data/catalog?node_device_id=node-1", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get, got %d: %s", getRec.Code, getRec.Body.String())
	}
	var pkg contracts.MasterDataPackage
	if err := json.Unmarshal(getRec.Body.Bytes(), &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.StreamName != contracts.MasterDataStreamCatalog || pkg.CloudVersion != 12 {
		t.Fatalf("unexpected package response: %+v", pkg)
	}
}

func TestPricingPolicyPackagePutGetAndExchange(t *testing.T) {
	repo := memory.NewRepository()
	if err := repo.AuthorizeNodeForTest("node-1", "restaurant-1", "node-token"); err != nil {
		t.Fatal(err)
	}
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	putBody := []byte(`{
	  "node_device_id":"node-1",
	  "restaurant_id":"restaurant-1",
	  "sync_mode":"full_snapshot",
	  "full_snapshot_reason":"terminal_restaurant_changed",
	  "cloud_version":21,
	  "payload_json":{
	    "tax_profiles":[{"id":"tax-vat-10","name":"VAT 10","tax_exempt":false,"active":true}],
	    "tax_rules":[{"id":"tax-rule-vat-10","tax_profile_id":"tax-vat-10","name":"VAT 10 exclusive","kind":"percentage","mode":"exclusive","rate_basis_points":1000,"active":true}],
	    "service_charge_rules":[{"id":"service-charge-10","restaurant_id":"restaurant-1","name":"Service charge 10","kind":"service_charge","amount_kind":"percentage","value_basis_points":1000,"active":true}],
	    "pricing_policies":[{"id":"discount-cloud-5","restaurant_id":"restaurant-1","kind":"discount","name":"Cloud discount 5","scope":"order","amount_kind":"percentage","value_basis_points":500,"application_index":30,"active":true}]
	  }
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/provisioning/master-data/pricing_policy", bytes.NewReader(putBody))
	putRec := httptest.NewRecorder()
	router.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected pricing_policy PUT 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/provisioning/master-data/pricing_policy?node_device_id=node-1", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected pricing_policy GET 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	var pkg contracts.MasterDataPackage
	if err := json.Unmarshal(getRec.Body.Bytes(), &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.StreamName != contracts.MasterDataStreamPricing || pkg.CloudVersion != 21 || !bytes.Contains(pkg.PayloadJSON, []byte(`"pricing_policies"`)) {
		t.Fatalf("unexpected pricing_policy package response: %+v payload=%s", pkg, pkg.PayloadJSON)
	}

	exchangeBody := []byte(`{
	  "protocol_version":"sync_exchange.v1",
	  "node_device_id":"node-1",
	  "restaurant_id":"restaurant-1",
	  "edge_events":[],
	  "streams":[{"stream_name":"pricing_policy","last_cloud_version":20,"checkpoint_token":"pricing_policy:20"}]
	}`)
	exchangeReq := httptest.NewRequest(http.MethodPost, "/api/v1/sync/exchange", bytes.NewReader(exchangeBody))
	exchangeReq.Header.Set("Authorization", "Bearer node-token")
	exchangeRec := httptest.NewRecorder()
	router.ServeHTTP(exchangeRec, exchangeReq)
	if exchangeRec.Code != http.StatusAccepted {
		t.Fatalf("expected exchange 202, got %d: %s", exchangeRec.Code, exchangeRec.Body.String())
	}
	var resp contracts.SyncExchangeResponse
	if err := json.Unmarshal(exchangeRec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.CloudPackages) != 1 || resp.CloudPackages[0].StreamName != contracts.MasterDataStreamPricing || resp.CloudPackages[0].CloudVersion != 21 {
		t.Fatalf("expected pricing_policy package in exchange response, got %+v", resp.CloudPackages)
	}
	if !bytes.Contains(resp.CloudPackages[0].PayloadJSON, []byte(`"service_charge_rules"`)) {
		t.Fatalf("expected pricing_policy payload in exchange response, got %s", resp.CloudPackages[0].PayloadJSON)
	}
}

func TestStopListReadinessRouteDoesNotExposeRawPayload(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	raw := []byte(`{
	  "version":"1",
	  "event_id":"018f0000-0000-7000-8000-0000000000b1",
	  "command_id":"command-stop-list-1",
	  "event_type":"StopListUpdated",
	  "aggregate_type":"StopList",
	  "aggregate_id":"stop-1",
	  "restaurant_id":"restaurant-1",
	  "device_id":"device-1",
	  "node_device_id":"device-1",
	  "occurred_at":"2026-05-05T12:05:00Z",
	  "payload":{"origin":"edge_device","data":{"stop_list_id":"stop-1","restaurant_id":"restaurant-1","catalog_item_id":"item-1","available_quantity":"0.000","active":true,"conflict_policy":"edge_overlay_until_next_publication","source":"edge","reason":"ingredient_unavailable","updated_at":"2026-05-05T12:05:00Z"}}
	}`)
	postEnvelope(t, router, raw)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/readiness/stop-list?restaurant_id=restaurant-1&node_device_id=device-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 readiness, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "raw_payload") || strings.Contains(body, "ingredient_unavailable") {
		t.Fatalf("readiness response must not expose raw payload: %s", body)
	}
	if !strings.Contains(body, "edge_overlay_requires_manager_review") || !strings.Contains(body, "async_inventory_worker") {
		t.Fatalf("readiness response missing contract metadata: %s", body)
	}
}

func TestProvisioningCurrenciesPutAndGet(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	putBody := []byte(`{
	  "sync_mode":"full_snapshot",
	  "full_snapshot_reason":"node_role_changed",
	  "cloud_version":21,
	  "payload_json":{
		"currencies":[
		  {
			"currency_code":643,
			"currency_alpha_code":"RUB",
			"minor_unit":2,
			"currency_iso_name":"Russian Ruble",
			"currency_symbol":"₽",
			"curr_basic_name":"р",
			"curr_add_name":"коп.",
			"show_add":true,
			"show_currency_basic_name":true
		  }
		]
	  }
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/provisioning/master-data/currencies", bytes.NewReader(putBody))
	putRec := httptest.NewRecorder()
	router.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on currency package upsert, got %d: %s", putRec.Code, putRec.Body.String())
	}
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/provisioning/master-data/currencies", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on currencies get, got %d: %s", getRec.Code, getRec.Body.String())
	}
	var pkg contracts.MasterDataPackage
	if err := json.Unmarshal(getRec.Body.Bytes(), &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.StreamName != contracts.MasterDataStreamCurrencies || pkg.CloudVersion != 21 {
		t.Fatalf("unexpected currencies package: %+v", pkg)
	}
}

func TestCloudUICORSPreflight(t *testing.T) {
	router := api.NewRouter(app.NewService(memory.NewRepository(), fixedClock{}))
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/restaurants", nil)
	req.Header.Set("Origin", "http://localhost:5174")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5174" {
		t.Fatalf("expected cloud-ui origin, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected CORS methods header")
	}
}

func postEnvelope(t *testing.T, h http.Handler, raw []byte) contracts.EventAck {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/edge-events", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
	var ack contracts.EventAck
	if err := json.Unmarshal(rec.Body.Bytes(), &ack); err != nil {
		t.Fatal(err)
	}
	return ack
}

func TestListEdgeEventsReturnsSafeIncomingEventLog(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	raw := []byte(`{
	  "version":"1",
	  "event_id":"018f0000-0000-7000-8000-0000000000d1",
	  "command_id":"command-log-1",
	  "event_type":"OrderCreated",
	  "aggregate_type":"Order",
	  "aggregate_id":"order-log-1",
	  "restaurant_id":"restaurant-log-1",
	  "device_id":"device-log-1",
	  "shift_id":"shift-log-1",
	  "occurred_at":"2026-05-05T09:00:00Z",
	  "payload":{"origin":"edge_device","data":{"id":"order-log-1","edge_order_id":"edge-order-log-1","restaurant_id":"restaurant-log-1","device_id":"device-log-1","shift_id":"shift-log-1","status":"open","table_name":"A1","guest_count":2,"opened_at":"2026-05-05T09:00:00Z","created_at":"2026-05-05T09:00:00Z","updated_at":"2026-05-05T09:00:00Z"}}
	}`)
	ack := postEnvelope(t, router, raw)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/edge-events?restaurant_id=restaurant-log-1&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var items []contracts.EdgeEventView
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one event, got %+v", items)
	}
	if items[0].CloudReceiptID != ack.CloudReceiptID || items[0].EventType != string(contracts.EventOrderCreated) || items[0].RestaurantID != "restaurant-log-1" {
		t.Fatalf("unexpected event view: %+v", items[0])
	}
	if strings.Contains(rec.Body.String(), "edge_order_log") || strings.Contains(rec.Body.String(), "edge-order-log-1") {
		t.Fatalf("event log response must not expose raw payload: %s", rec.Body.String())
	}
}

func TestListFinancialOperationsReturnsSafeReportingProjection(t *testing.T) {
	repo := memory.NewRepository()
	router := api.NewRouter(app.NewService(repo, fixedClock{}))
	postEnvelope(t, router, sampleFinancialOperationEnvelope(t, contracts.EventRefundRecorded, "018f0000-0000-7000-8000-0000000000f2", "command-refund-api-1", "financial-operation-api-1", "refund", 2500, "shift-refund-api-1", "2026-05-06"))
	postEnvelope(t, router, sampleFinancialOperationEnvelope(t, contracts.EventCancellationRecorded, "018f0000-0000-7000-8000-0000000000c2", "command-cancel-api-1", "financial-operation-api-2", "cancellation", 1000, "shift-cancel-api-1", "2026-05-07"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reporting/financial-operations?restaurant_id=restaurant-1&business_date_from=2026-05-06&business_date_to=2026-05-06&operation_type=refund&shift_id=shift-refund-api-1&original_shift_id=shift-sale-1&check_id=check-1&limit=10&offset=0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var items []contracts.FinancialOperationProjection
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].OperationID != "financial-operation-api-1" || items[0].OperationType != "refund" || items[0].Amount != 2500 {
		t.Fatalf("unexpected financial operation report: %+v", items)
	}
	body := rec.Body.String()
	for _, forbidden := range []string{"snapshot", "document_type", "PIN", "token"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("financial operations report must not expose %s: %s", forbidden, body)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/reporting/financial-operations?business_date_from=2026-05-08&business_date_to=2026-05-06", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid date range, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListInventoryLedgerReturnsBoundedReadOnlyLedger(t *testing.T) {
	repo := memory.NewRepository()
	occurred := time.Date(2026, 5, 5, 9, 0, 0, 0, time.UTC)
	repo.AddInventoryLedgerForTest(
		contracts.InventoryLedgerEntry{
			ID:                "ledger-1",
			RestaurantID:      "restaurant-1",
			StockDocumentID:   "stock-doc-1",
			SourceEventID:     "event-check-closed",
			SourceEventType:   string(contracts.EventCheckClosed),
			CatalogItemID:     "component-1",
			OrderLineID:       "line-1",
			MovementType:      "OUT",
			Quantity:          "2.000",
			UnitCode:          "PC",
			CostingStatus:     "estimated",
			OccurredAt:        occurred,
			BusinessDateLocal: "2026-05-05",
			CreatedAt:         occurred,
		},
		contracts.InventoryLedgerEntry{
			ID:                "ledger-2",
			RestaurantID:      "restaurant-2",
			StockDocumentID:   "stock-doc-2",
			SourceEventID:     "event-other",
			SourceEventType:   string(contracts.EventItemServed),
			CatalogItemID:     "component-2",
			OrderLineID:       "line-2",
			MovementType:      "OUT",
			Quantity:          "1.000",
			UnitCode:          "PC",
			CostingStatus:     "estimated",
			OccurredAt:        occurred,
			BusinessDateLocal: "2026-05-05",
			CreatedAt:         occurred,
		},
	)
	router := api.NewRouter(app.NewService(repo, fixedClock{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock-ledger?restaurant_id=restaurant-1&source_event_type=CheckClosed&order_line_id=line-1&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var items []contracts.InventoryLedgerEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "ledger-1" || items[0].SourceEventID != "event-check-closed" {
		t.Fatalf("unexpected ledger response: %+v", items)
	}
	if strings.Contains(rec.Body.String(), "payload") {
		t.Fatalf("ledger response must not expose raw payload: %s", rec.Body.String())
	}
}

func TestListInventoryStockBalancesReadsMaterializedStateSafely(t *testing.T) {
	repo := memory.NewRepository()
	occurred := time.Date(2026, 5, 5, 9, 0, 0, 0, time.UTC)
	repo.AddInventoryStockBalancesForTest(
		contracts.InventoryStockBalance{RestaurantID: "restaurant-1", WarehouseID: "warehouse-main", CatalogItemID: "component-1", QuantityOnHand: "-2.000", UnitCode: "PC", CostingStatus: "needs_recalculation", NeedsRecalculation: true, LastMovementAt: occurred.Add(time.Hour), BusinessDateTo: "2026-05-05"},
		contracts.InventoryStockBalance{RestaurantID: "restaurant-2", WarehouseID: "warehouse-main", CatalogItemID: "component-1", QuantityOnHand: "10.000", UnitCode: "PC", CostingStatus: "final", LastMovementAt: occurred, BusinessDateTo: "2026-05-05"},
	)
	router := api.NewRouter(app.NewService(repo, fixedClock{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock-balances?restaurant_id=restaurant-1&warehouse_id=warehouse-main&catalog_item_id=component-1&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var items []contracts.InventoryStockBalance
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one balance, got %+v", items)
	}
	if items[0].QuantityOnHand != "-2.000" || items[0].CostingStatus != "needs_recalculation" || !items[0].NeedsRecalculation {
		t.Fatalf("unexpected materialized balance: %+v", items[0])
	}
	body := rec.Body.String()
	for _, forbidden := range []string{"payload", "raw", "COGS", "margin"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("balance response must not expose %s: %s", forbidden, body)
		}
	}
}

func TestListInventoryStockBalancesSupportsBoundsEmptyStateAndStatusFilter(t *testing.T) {
	repo := memory.NewRepository()
	occurred := time.Date(2026, 5, 5, 9, 0, 0, 0, time.UTC)
	repo.AddInventoryStockBalancesForTest(
		contracts.InventoryStockBalance{RestaurantID: "restaurant-1", WarehouseID: "warehouse-main", CatalogItemID: "item-1", QuantityOnHand: "1.000", UnitCode: "PC", CostingStatus: "estimated", NeedsRecalculation: true, LastMovementAt: occurred, BusinessDateTo: "2026-05-05"},
		contracts.InventoryStockBalance{RestaurantID: "restaurant-1", WarehouseID: "warehouse-main", CatalogItemID: "item-2", QuantityOnHand: "1.000", UnitCode: "PC", CostingStatus: "final", LastMovementAt: occurred, BusinessDateTo: "2026-05-05"},
	)
	router := api.NewRouter(app.NewService(repo, fixedClock{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock-balances?restaurant_id=restaurant-1&costing_status=estimated&limit=1&offset=0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var items []contracts.InventoryStockBalance
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].CatalogItemID != "item-1" || items[0].CostingStatus != "estimated" {
		t.Fatalf("unexpected filtered balance: %+v", items)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock-balances?restaurant_id=restaurant-1&catalog_item_id=missing", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty state, got %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty balance list, got %+v", items)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/inventory/stock-balances", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing restaurant_id, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListInventoryRecalculationJobsReturnsBoundedDiagnosticStatus(t *testing.T) {
	repo := memory.NewRepository()
	created := time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC)
	repo.AddInventoryRecalculationJobsForTest(
		contracts.InventoryRecalculationJob{ID: "job-1", RestaurantID: "restaurant-1", TriggerType: "StockReceiptCaptured", TriggerEventID: "event-1", Status: "queued", BusinessDateFrom: "2026-05-05", BusinessDateTo: "2026-05-06", AffectedCatalogItemCount: 1, AffectedWarehouseCount: 1, TotalSteps: 2, CreatedAt: created, UpdatedAt: created},
		contracts.InventoryRecalculationJob{ID: "job-2", RestaurantID: "restaurant-2", TriggerType: "InventoryCountCaptured", Status: "failed", BusinessDateFrom: "2026-05-05", BusinessDateTo: "2026-05-06", FailureCode: "RECIPE_DEPENDENCY_CYCLE", FailureMessageKey: "inventory.recalculation.recipe_cycle", CreatedAt: created.Add(time.Hour), UpdatedAt: created.Add(time.Hour)},
	)
	router := api.NewRouter(app.NewService(repo, fixedClock{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/recalculation-jobs?restaurant_id=restaurant-1&status=queued&trigger_type=StockReceiptCaptured&limit=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var items []contracts.InventoryRecalculationJob
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "job-1" || items[0].TotalSteps != 2 {
		t.Fatalf("unexpected recalculation jobs: %+v", items)
	}
	body := rec.Body.String()
	for _, forbidden := range []string{"payload", "raw", "SQL"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("recalculation job response must not expose %s: %s", forbidden, body)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/inventory/recalculation-jobs/job-1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 detail, got %d: %s", rec.Code, rec.Body.String())
	}
	var detail contracts.InventoryRecalculationJob
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.ID != "job-1" || detail.TriggerEventID != "event-1" {
		t.Fatalf("unexpected detail: %+v", detail)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/inventory/recalculation-jobs/missing", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown job id, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListInventoryRecalculationJobsRejectsInvalidFilters(t *testing.T) {
	router := api.NewRouter(app.NewService(memory.NewRepository(), fixedClock{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/recalculation-jobs?status=retrying", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d: %s", rec.Code, rec.Body.String())
	}
}

func sampleFinancialOperationEnvelope(t *testing.T, eventType contracts.EventType, eventID, commandID, operationID, operationType string, amount int64, shiftID, businessDate string) []byte {
	t.Helper()
	body := map[string]any{
		"version":           "1",
		"event_id":          eventID,
		"command_id":        commandID,
		"event_type":        string(eventType),
		"aggregate_type":    "FinancialOperation",
		"aggregate_id":      operationID,
		"restaurant_id":     "restaurant-1",
		"device_id":         "device-1",
		"node_device_id":    "device-1",
		"client_device_id":  "client-1",
		"actor_employee_id": "manager-1",
		"session_id":        "session-1",
		"shift_id":          shiftID,
		"occurred_at":       businessDate + "T09:00:00Z",
		"payload": map[string]any{
			"origin": "edge_device",
			"data": map[string]any{
				"id":                     operationID,
				"edge_operation_id":      "edge-" + operationID,
				"restaurant_id":          "restaurant-1",
				"device_id":              "device-1",
				"shift_id":               shiftID,
				"original_shift_id":      "shift-sale-1",
				"check_id":               "check-1",
				"precheck_id":            "precheck-1",
				"operation_type":         operationType,
				"operation_kind":         "full",
				"status":                 "recorded",
				"amount":                 amount,
				"currency":               "RUB",
				"business_date_local":    businessDate,
				"inventory_disposition":  "no_stock_effect",
				"reason":                 "guest return",
				"created_by_employee_id": "manager-1",
				"snapshot":               map[string]any{"document_type": "financial_operation", "check_id": "check-1"},
				"created_at":             businessDate + "T09:00:00Z",
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
