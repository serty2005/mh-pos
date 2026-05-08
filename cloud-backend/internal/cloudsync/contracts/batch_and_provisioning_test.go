package contracts_test

import (
	"encoding/json"
	"testing"

	"cloud-backend/internal/cloudsync/contracts"
)

func TestValidateMasterDataPackageAllowsIncrementalAndPayload(t *testing.T) {
	err := contracts.ValidateMasterDataPackage(contracts.MasterDataPackage{
		StreamName:   contracts.MasterDataStreamCatalog,
		SyncMode:     contracts.SyncModeIncremental,
		CloudVersion: 1,
		PayloadJSON:  json.RawMessage(`{"catalog_items":[{"id":"c-1"}]}`),
	})
	if err != nil {
		t.Fatalf("expected valid package, got %v", err)
	}
}

func TestNormalizeSyncModeDefaultsToIncremental(t *testing.T) {
	if got := contracts.NormalizeSyncMode(""); got != contracts.SyncModeIncremental {
		t.Fatalf("expected empty sync_mode to default to incremental, got %q", got)
	}
}

func TestValidateMasterDataPackageRejectsInvalidStream(t *testing.T) {
	err := contracts.ValidateMasterDataPackage(contracts.MasterDataPackage{
		StreamName:         "unknown",
		SyncMode:           contracts.SyncModeFullSnapshot,
		FullSnapshotReason: contracts.FullSnapshotReasonTerminalRestaurantChanged,
		CloudVersion:       1,
		PayloadJSON:        json.RawMessage(`{"x":1}`),
	})
	if err == nil {
		t.Fatal("expected invalid stream validation error")
	}
}

func TestValidateMasterDataPackageCurrenciesStream(t *testing.T) {
	err := contracts.ValidateMasterDataPackage(contracts.MasterDataPackage{
		StreamName:         contracts.MasterDataStreamCurrencies,
		SyncMode:           contracts.SyncModeFullSnapshot,
		FullSnapshotReason: contracts.FullSnapshotReasonNodeRoleChanged,
		CloudVersion:       1,
		PayloadJSON: json.RawMessage(`{
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
		}`),
	})
	if err != nil {
		t.Fatalf("expected valid currencies stream package, got %v", err)
	}
}

func TestValidateMasterDataPackageRequiresReasonForFullSnapshot(t *testing.T) {
	err := contracts.ValidateMasterDataPackage(contracts.MasterDataPackage{
		StreamName:   contracts.MasterDataStreamCatalog,
		SyncMode:     contracts.SyncModeFullSnapshot,
		CloudVersion: 1,
		PayloadJSON:  json.RawMessage(`{"catalog_items":[{"id":"c-1"}]}`),
	})
	if err == nil {
		t.Fatal("expected full_snapshot without reason to be rejected")
	}
}

func TestValidateMasterDataPackageRejectsReasonForIncremental(t *testing.T) {
	err := contracts.ValidateMasterDataPackage(contracts.MasterDataPackage{
		StreamName:         contracts.MasterDataStreamCatalog,
		SyncMode:           contracts.SyncModeIncremental,
		FullSnapshotReason: contracts.FullSnapshotReasonTerminalRestaurantChanged,
		CloudVersion:       1,
		PayloadJSON:        json.RawMessage(`{"catalog_items":[{"id":"c-1"}]}`),
	})
	if err == nil {
		t.Fatal("expected incremental with full_snapshot_reason to be rejected")
	}
}
