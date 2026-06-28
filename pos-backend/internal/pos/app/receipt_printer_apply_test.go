package app_test

import (
	"encoding/json"
	"testing"

	"pos-backend/internal/pos/domain"
)

func receiptPrintersPayload(t *testing.T, printers ...map[string]any) json.RawMessage {
	t.Helper()
	body, err := json.Marshal(map[string]any{"printers": printers})
	if err != nil {
		t.Fatalf("marshal printers payload: %v", err)
	}
	return body
}

func countReceiptPrinters(t *testing.T, f *fixture) int {
	t.Helper()
	return countRows(t, f, "receipt_printers")
}

func TestApplyPrintersStreamReplacesRowsForRestaurantAtomically(t *testing.T) {
	f := newFixture(t)
	first := receiptPrintersPayload(t,
		map[string]any{"id": "printer-front", "restaurant_id": f.restaurant.ID, "name": "Front", "type": "tcp", "address": "10.25.1.201", "port": 9100, "document_types": []string{"precheck", "ticket"}, "codepage": "cp437", "paper_cut_type": "partial", "cpl": 48, "version": 5},
		map[string]any{"id": "printer-ticket", "restaurant_id": f.restaurant.ID, "name": "Ticket", "type": "usb", "document_types": []string{"ticket"}, "codepage": "", "paper_cut_type": "partial", "cpl": 48, "version": 5},
	)
	if err := f.service.ApplySyncExchangeCloudPackages(f.ctx, []domain.CloudPackage{{
		StreamName:      string(domain.MasterDataStreamPrinters),
		NodeDeviceID:    f.device.ID,
		RestaurantID:    f.restaurant.ID,
		SyncMode:        string(domain.SyncModeIncremental),
		CloudVersion:    5,
		CheckpointToken: "printers:restaurant:5:2",
		PayloadJSON:     first,
	}}); err != nil {
		t.Fatal(err)
	}
	if got := countReceiptPrinters(t, f); got != 2 {
		t.Fatalf("expected 2 printers after first apply, got %d", got)
	}

	second := receiptPrintersPayload(t,
		map[string]any{"id": "printer-front", "restaurant_id": f.restaurant.ID, "name": "Front v2", "type": "tcp", "address": "10.25.1.202", "port": 9100, "document_types": []string{"precheck"}, "codepage": "cp866", "paper_cut_type": "full", "cpl": 56, "version": 6},
	)
	if err := f.service.ApplySyncExchangeCloudPackages(f.ctx, []domain.CloudPackage{{
		StreamName:      string(domain.MasterDataStreamPrinters),
		NodeDeviceID:    f.device.ID,
		RestaurantID:    f.restaurant.ID,
		SyncMode:        string(domain.SyncModeIncremental),
		CloudVersion:    6,
		CheckpointToken: "printers:restaurant:6:1",
		PayloadJSON:     second,
	}}); err != nil {
		t.Fatal(err)
	}
	if got := countReceiptPrinters(t, f); got != 1 {
		t.Fatalf("expected atomic replace to leave 1 printer, got %d", got)
	}
	var address, codepage, paperCutType string
	var cpl, cloudVersion int
	if err := f.db.QueryRowContext(f.ctx, `SELECT address,codepage,paper_cut_type,cpl,cloud_version FROM receipt_printers WHERE id = 'printer-front'`).Scan(&address, &codepage, &paperCutType, &cpl, &cloudVersion); err != nil {
		t.Fatal(err)
	}
	if address != "10.25.1.202" || codepage != "cp866" || paperCutType != "full" || cpl != 56 || cloudVersion != 6 {
		t.Fatalf("expected replaced printer fields, got address=%q codepage=%q cut=%q cpl=%d version=%d", address, codepage, paperCutType, cpl, cloudVersion)
	}
}

func TestApplyPrintersStreamStoresVersionCheckpoint(t *testing.T) {
	f := newFixture(t)
	payload := receiptPrintersPayload(t,
		map[string]any{"id": "printer-front", "restaurant_id": f.restaurant.ID, "name": "Front", "type": "tcp", "address": "10.25.1.201", "port": 9100, "document_types": []string{"precheck"}, "paper_cut_type": "partial", "cpl": 48, "version": 5},
	)
	if err := f.service.ApplySyncExchangeCloudPackages(f.ctx, []domain.CloudPackage{{
		StreamName:      string(domain.MasterDataStreamPrinters),
		NodeDeviceID:    f.device.ID,
		RestaurantID:    f.restaurant.ID,
		SyncMode:        string(domain.SyncModeIncremental),
		CloudVersion:    5,
		CheckpointToken: "printers:restaurant:5:1",
		PayloadJSON:     payload,
	}}); err != nil {
		t.Fatal(err)
	}

	var status, checkpoint string
	var lastVersion int64
	if err := f.db.QueryRowContext(f.ctx, `SELECT status,checkpoint_token,last_cloud_version FROM cloud_master_sync_state WHERE node_device_id = ? AND stream_name = 'printers'`, f.device.ID).Scan(&status, &checkpoint, &lastVersion); err != nil {
		t.Fatal(err)
	}
	if status != "applied" || checkpoint != "printers:restaurant:5:1" || lastVersion != 5 {
		t.Fatalf("unexpected printers sync state: status=%q checkpoint=%q version=%d", status, checkpoint, lastVersion)
	}
}

func TestSyncExchangeAdvertisesPrintersStream(t *testing.T) {
	f := newFixture(t)
	now := fixedClock{}.Now()
	if err := f.repo.UpsertEdgeProvisioningState(f.ctx, &domain.EdgeProvisioningState{
		ID:               "local-printers",
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
		if stream.StreamName == string(domain.MasterDataStreamPrinters) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Edge to request printers stream during exchange, got %+v", state.Streams)
	}
}
