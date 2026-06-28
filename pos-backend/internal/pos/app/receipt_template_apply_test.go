package app_test

import (
	"encoding/json"
	"testing"

	"pos-backend/internal/pos/domain"
)

func receiptTemplatesPayload(t *testing.T, templates ...map[string]any) json.RawMessage {
	t.Helper()
	body, err := json.Marshal(map[string]any{"receipt_templates": templates})
	if err != nil {
		t.Fatalf("marshal receipt_templates payload: %v", err)
	}
	return body
}

func countReceiptTemplates(t *testing.T, f *fixture) int {
	t.Helper()
	var n int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM receipt_templates`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func countReceiptTemplatesWhere(t *testing.T, f *fixture, where string) int {
	t.Helper()
	var n int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(1) FROM receipt_templates WHERE `+where).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestApplyReceiptTemplatesStreamReplacesRowsAtomically(t *testing.T) {
	f := newFixture(t)
	first := receiptTemplatesPayload(t,
		map[string]any{"id": "tmpl-precheck", "document_type": "precheck", "name": "Пречек", "content": "PRECHECK-V1", "level": 1, "cpl": 48, "printer_class": "generic", "is_default": true, "version": 1},
		map[string]any{"id": "tmpl-ticket", "document_type": "ticket", "name": "Билет", "content": "TICKET-V1", "level": 1, "cpl": 32, "printer_class": "generic", "is_default": true, "version": 1},
	)
	if err := f.service.ApplySyncExchangeCloudPackages(f.ctx, []domain.CloudPackage{{
		StreamName:      string(domain.MasterDataStreamReceiptTemplates),
		NodeDeviceID:    f.device.ID,
		RestaurantID:    f.restaurant.ID,
		SyncMode:        string(domain.SyncModeIncremental),
		CloudVersion:    5,
		CheckpointToken: "receipt-templates:5",
		PayloadJSON:     first,
	}}); err != nil {
		t.Fatal(err)
	}
	if got := countReceiptTemplates(t, f); got != 2 {
		t.Fatalf("expected 2 templates after first apply, got %d", got)
	}

	// Второй пакет содержит меньше строк: атомарная замена должна удалить отсутствующий ticket.
	second := receiptTemplatesPayload(t,
		map[string]any{"id": "tmpl-precheck", "document_type": "precheck", "name": "Пречек", "content": "PRECHECK-V2", "level": 1, "cpl": 48, "printer_class": "generic", "is_default": true, "version": 2},
	)
	if err := f.service.ApplySyncExchangeCloudPackages(f.ctx, []domain.CloudPackage{{
		StreamName:      string(domain.MasterDataStreamReceiptTemplates),
		NodeDeviceID:    f.device.ID,
		RestaurantID:    f.restaurant.ID,
		SyncMode:        string(domain.SyncModeIncremental),
		CloudVersion:    6,
		CheckpointToken: "receipt-templates:6",
		PayloadJSON:     second,
	}}); err != nil {
		t.Fatal(err)
	}
	if got := countReceiptTemplates(t, f); got != 1 {
		t.Fatalf("expected atomic replace to leave 1 template, got %d", got)
	}
	var content string
	var version int
	if err := f.db.QueryRowContext(f.ctx, `SELECT content,version FROM receipt_templates WHERE id = 'tmpl-precheck'`).Scan(&content, &version); err != nil {
		t.Fatal(err)
	}
	if content != "PRECHECK-V2" || version != 2 {
		t.Fatalf("expected replaced precheck content/version, got %q v%d", content, version)
	}
	if countReceiptTemplatesWhere(t, f, "id = 'tmpl-ticket'") != 0 {
		t.Fatal("expected removed ticket template to be gone after atomic replace")
	}
}

func TestApplyReceiptTemplatesStreamStoresVersionCheckpoint(t *testing.T) {
	f := newFixture(t)
	payload := receiptTemplatesPayload(t,
		map[string]any{"id": "tmpl-precheck", "document_type": "precheck", "name": "Пречек", "content": "PRECHECK-V1", "level": 1, "cpl": 48, "printer_class": "generic", "is_default": true, "version": 1},
	)
	apply := func(version int64, checkpoint string, body json.RawMessage) {
		if err := f.service.ApplySyncExchangeCloudPackages(f.ctx, []domain.CloudPackage{{
			StreamName:      string(domain.MasterDataStreamReceiptTemplates),
			NodeDeviceID:    f.device.ID,
			RestaurantID:    f.restaurant.ID,
			SyncMode:        string(domain.SyncModeIncremental),
			CloudVersion:    version,
			CheckpointToken: checkpoint,
			PayloadJSON:     body,
		}}); err != nil {
			t.Fatal(err)
		}
	}
	apply(5, "receipt-templates:restaurant:5:1", payload)

	var status, checkpoint string
	var lastVersion int64
	if err := f.db.QueryRowContext(f.ctx, `SELECT status,checkpoint_token,last_cloud_version FROM cloud_master_sync_state WHERE node_device_id = ? AND stream_name = 'receipt_templates'`, f.device.ID).Scan(&status, &checkpoint, &lastVersion); err != nil {
		t.Fatal(err)
	}
	if status != "applied" || checkpoint != "receipt-templates:restaurant:5:1" || lastVersion != 5 {
		t.Fatalf("unexpected sync state: status=%q checkpoint=%q version=%d", status, checkpoint, lastVersion)
	}

	// Повторная доставка той же версии и checkpoint должна быть no-op (dedup),
	// даже если payload отличается.
	changed := receiptTemplatesPayload(t,
		map[string]any{"id": "tmpl-other", "document_type": "ticket", "name": "Билет", "content": "SHOULD-NOT-APPLY", "level": 1, "cpl": 32, "printer_class": "generic", "is_default": true, "version": 1},
	)
	apply(5, "receipt-templates:restaurant:5:1", changed)
	if countReceiptTemplatesWhere(t, f, "id = 'tmpl-other'") != 0 {
		t.Fatal("expected duplicate version+checkpoint delivery to be skipped")
	}
	if got := countReceiptTemplates(t, f); got != 1 {
		t.Fatalf("expected dedup to keep original single template, got %d", got)
	}
}

func TestSyncExchangeAdvertisesReceiptTemplatesStream(t *testing.T) {
	f := newFixture(t)
	now := fixedClock{}.Now()
	if err := f.repo.UpsertEdgeProvisioningState(f.ctx, &domain.EdgeProvisioningState{
		ID:               "local",
		NodeDeviceID:     f.device.ID,
		RestaurantID:     f.restaurant.ID,
		Status:           domain.ProvisioningPaired,
		CredentialsType:  "node_token",
		CredentialsToken: "node-token",
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatal(err)
	}
	state, err := f.service.GetSyncExchangeState(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, stream := range state.Streams {
		if stream.StreamName == string(domain.MasterDataStreamReceiptTemplates) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Edge to request receipt_templates stream during exchange, got %+v", state.Streams)
	}
}

func TestApplyReceiptTemplatesStreamRejectsInvalidTemplate(t *testing.T) {
	f := newFixture(t)
	// Фискальный document_type запрещён в нефискальных шаблонах печати.
	invalid := receiptTemplatesPayload(t,
		map[string]any{"id": "tmpl-bad", "document_type": "fiscal_check", "name": "X", "content": "Y", "level": 1, "cpl": 48, "printer_class": "generic", "is_default": false, "version": 1},
	)
	if err := f.service.ApplySyncExchangeCloudPackages(f.ctx, []domain.CloudPackage{{
		StreamName:      string(domain.MasterDataStreamReceiptTemplates),
		NodeDeviceID:    f.device.ID,
		RestaurantID:    f.restaurant.ID,
		SyncMode:        string(domain.SyncModeIncremental),
		CloudVersion:    5,
		CheckpointToken: "receipt-templates:5",
		PayloadJSON:     invalid,
	}}); err != nil {
		t.Fatal(err)
	}
	if got := countReceiptTemplates(t, f); got != 0 {
		t.Fatalf("expected invalid package to write no rows, got %d", got)
	}
	var status string
	if err := f.db.QueryRowContext(f.ctx, `SELECT status FROM cloud_master_sync_state WHERE node_device_id = ? AND stream_name = 'receipt_templates'`, f.device.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "failed" {
		t.Fatalf("expected invalid receipt_templates package quarantined, status=%q", status)
	}
}
