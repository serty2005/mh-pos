package contracts_test

import (
	"encoding/json"
	"testing"

	"cloud-backend/internal/cloudsync/contracts"
)

// POS-71: receipt_templates должен быть распознан и как master-data, и как exchange stream,
// иначе Cloud package не дойдёт до Edge через sync exchange.
func TestReceiptTemplatesStreamRecognized(t *testing.T) {
	if err := contracts.ValidateMasterDataStream(contracts.MasterDataStreamReceiptTemplates); err != nil {
		t.Fatalf("ValidateMasterDataStream(receipt_templates): %v", err)
	}
	if err := contracts.ValidateExchangeStream(contracts.MasterDataStreamReceiptTemplates); err != nil {
		t.Fatalf("ValidateExchangeStream(receipt_templates): %v", err)
	}
}

func TestPrintersStreamRecognized(t *testing.T) {
	if err := contracts.ValidateMasterDataStream(contracts.MasterDataStreamPrinters); err != nil {
		t.Fatalf("ValidateMasterDataStream(printers): %v", err)
	}
	if err := contracts.ValidateExchangeStream(contracts.MasterDataStreamPrinters); err != nil {
		t.Fatalf("ValidateExchangeStream(printers): %v", err)
	}
}

func TestReceiptTemplatesPayloadValidation(t *testing.T) {
	valid, _ := json.Marshal(map[string]any{
		"node_device_id":   "node-1",
		"restaurant_id":    "restaurant-1",
		"sync_mode":        "incremental",
		"checkpoint_token": "receipt-templates:restaurant-1:1:1",
		"cloud_version":    int64(7),
		"cloud_updated_at": "2026-06-27T09:00:00Z",
		"receipt_templates": []map[string]any{{
			"id": "tmpl-1", "document_type": "precheck", "name": "Пречек",
			"content": "{header}", "level": 1, "cpl": 48, "printer_class": "generic",
			"is_default": true, "version": 1,
		}},
	})
	if err := contracts.ValidateMasterDataPayload(contracts.MasterDataStreamReceiptTemplates, valid); err != nil {
		t.Fatalf("expected valid receipt_templates payload, got %v", err)
	}

	missing, _ := json.Marshal(map[string]any{
		"node_device_id":   "node-1",
		"restaurant_id":    "restaurant-1",
		"sync_mode":        "incremental",
		"checkpoint_token": "receipt-templates:restaurant-1:0:0",
		"cloud_version":    int64(7),
		"cloud_updated_at": "2026-06-27T09:00:00Z",
	})
	if err := contracts.ValidateMasterDataPayload(contracts.MasterDataStreamReceiptTemplates, missing); err == nil {
		t.Fatal("expected error when receipt_templates array is missing")
	}
}

func TestPrintersPayloadValidation(t *testing.T) {
	valid, _ := json.Marshal(map[string]any{
		"node_device_id":   "node-1",
		"restaurant_id":    "restaurant-1",
		"sync_mode":        "incremental",
		"checkpoint_token": "printers:restaurant-1:1:1",
		"cloud_version":    int64(7),
		"cloud_updated_at": "2026-06-28T09:00:00Z",
		"printers": []map[string]any{{
			"id": "printer-1", "restaurant_id": "restaurant-1", "name": "Receipt printer",
			"type": "tcp", "address": "10.25.1.201", "port": 9100,
			"document_types": []string{"precheck", "ticket"}, "codepage": "cp437",
			"paper_cut_type": "partial", "cpl": 48, "version": 1,
		}},
	})
	if err := contracts.ValidateMasterDataPayload(contracts.MasterDataStreamPrinters, valid); err != nil {
		t.Fatalf("expected valid printers payload, got %v", err)
	}

	missing, _ := json.Marshal(map[string]any{
		"node_device_id":   "node-1",
		"restaurant_id":    "restaurant-1",
		"sync_mode":        "incremental",
		"checkpoint_token": "printers:restaurant-1:0:0",
		"cloud_version":    int64(7),
		"cloud_updated_at": "2026-06-28T09:00:00Z",
	})
	if err := contracts.ValidateMasterDataPayload(contracts.MasterDataStreamPrinters, missing); err == nil {
		t.Fatal("expected error when printers array is missing")
	}
}
