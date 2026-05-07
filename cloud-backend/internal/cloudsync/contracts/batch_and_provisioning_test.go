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

func TestValidateMasterDataPackageRejectsInvalidStream(t *testing.T) {
	err := contracts.ValidateMasterDataPackage(contracts.MasterDataPackage{
		StreamName:   "unknown",
		SyncMode:     contracts.SyncModeFullSnapshot,
		CloudVersion: 1,
		PayloadJSON:  json.RawMessage(`{"x":1}`),
	})
	if err == nil {
		t.Fatal("expected invalid stream validation error")
	}
}
