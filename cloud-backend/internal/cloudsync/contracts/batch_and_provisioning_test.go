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

func TestValidateMasterDataPackagePricingPolicyStream(t *testing.T) {
	err := contracts.ValidateMasterDataPackage(contracts.MasterDataPackage{
		StreamName:   contracts.MasterDataStreamPricing,
		SyncMode:     contracts.SyncModeIncremental,
		CloudVersion: 2,
		PayloadJSON: json.RawMessage(`{
			"tax_profiles":[{"id":"tax-vat-10","name":"VAT 10","tax_exempt":false,"active":true}],
			"tax_rules":[{"id":"tax-rule-10","tax_profile_id":"tax-vat-10","name":"VAT 10","kind":"percentage","mode":"exclusive","rate_basis_points":1000,"active":true}],
			"service_charge_rules":[{"id":"svc-10","restaurant_id":"restaurant-1","name":"Service 10","kind":"service_charge","amount_kind":"percentage","value_basis_points":1000,"active":true}]
		}`),
	})
	if err != nil {
		t.Fatalf("expected valid pricing_policy stream package, got %v", err)
	}
}

func TestValidateMasterDataPackageRejectsUnknownPayloadShape(t *testing.T) {
	err := contracts.ValidateMasterDataPackage(contracts.MasterDataPackage{
		StreamName:   contracts.MasterDataStreamCatalog,
		SyncMode:     contracts.SyncModeIncremental,
		CloudVersion: 3,
		PayloadJSON:  json.RawMessage(`{"catalog_items":[{"id":"c-1"}],"unexpected":true}`),
	})
	if err == nil {
		t.Fatal("expected unknown catalog payload field to be rejected")
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
