package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type BatchItemStatus string

const (
	BatchItemAccepted  BatchItemStatus = "accepted"
	BatchItemRejected  BatchItemStatus = "rejected"
	BatchItemRetryable BatchItemStatus = "retryable"
)

// BatchReceiveRequest содержит raw SyncEnvelope items для item-level ACK processing.
type BatchReceiveRequest struct {
	Items []json.RawMessage `json:"items"`
}

// BatchEventAckItem описывает per-item receive result.
type BatchEventAckItem struct {
	Index     int             `json:"index"`
	Status    BatchItemStatus `json:"status"`
	ErrorCode string          `json:"error_code,omitempty"`
	Error     string          `json:"error,omitempty"`
	Ack       *EventAck       `json:"ack,omitempty"`
}

// BatchEventAck агрегирует item-level ACK decisions для одного batch request.
type BatchEventAck struct {
	Status string              `json:"status"`
	Items  []BatchEventAckItem `json:"items"`
}

const (
	MasterDataStreamRestaurants = "restaurants"
	MasterDataStreamDevices     = "devices"
	MasterDataStreamStaff       = "staff"
	MasterDataStreamFloor       = "floor"
	MasterDataStreamCatalog     = "catalog"
	MasterDataStreamMenu        = "menu"
	MasterDataStreamPricing     = "pricing_policy"
	MasterDataStreamCurrencies  = "currencies"
)

const (
	SyncModeFullSnapshot = "full_snapshot"
	SyncModeIncremental  = "incremental"
)

const (
	FullSnapshotReasonTerminalRestaurantChanged = "terminal_restaurant_changed"
	FullSnapshotReasonNodeRoleChanged           = "node_role_changed"
)

// MasterDataPackage хранит Cloud-authored provisioning payload для Cloud -> Edge import.
type MasterDataPackage struct {
	StreamName         string          `json:"stream_name"`
	NodeDeviceID       string          `json:"node_device_id,omitempty"`
	RestaurantID       string          `json:"restaurant_id,omitempty"`
	SyncMode           string          `json:"sync_mode"`
	FullSnapshotReason string          `json:"full_snapshot_reason,omitempty"`
	CloudVersion       int64           `json:"cloud_version"`
	CheckpointToken    string          `json:"checkpoint_token,omitempty"`
	CloudUpdatedAt     *time.Time      `json:"cloud_updated_at,omitempty"`
	PayloadJSON        json.RawMessage `json:"payload_json"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

func ValidateMasterDataStream(streamName string) error {
	switch strings.TrimSpace(streamName) {
	case MasterDataStreamRestaurants, MasterDataStreamDevices, MasterDataStreamStaff, MasterDataStreamFloor, MasterDataStreamCatalog, MasterDataStreamMenu, MasterDataStreamPricing, MasterDataStreamCurrencies:
		return nil
	default:
		return fmt.Errorf("%w: unsupported stream_name %q", ErrInvalidEnvelope, streamName)
	}
}

func NormalizeSyncMode(syncMode string) string {
	mode := strings.TrimSpace(strings.ToLower(syncMode))
	switch mode {
	case "":
		return SyncModeIncremental
	case SyncModeFullSnapshot:
		return SyncModeFullSnapshot
	case SyncModeIncremental:
		return SyncModeIncremental
	default:
		return ""
	}
}

func NormalizeFullSnapshotReason(reason string) string {
	switch strings.TrimSpace(strings.ToLower(reason)) {
	case FullSnapshotReasonTerminalRestaurantChanged:
		return FullSnapshotReasonTerminalRestaurantChanged
	case FullSnapshotReasonNodeRoleChanged:
		return FullSnapshotReasonNodeRoleChanged
	default:
		return ""
	}
}

func ValidateMasterDataPackage(v MasterDataPackage) error {
	if err := ValidateMasterDataStream(v.StreamName); err != nil {
		return err
	}
	mode := NormalizeSyncMode(v.SyncMode)
	if mode == "" {
		return fmt.Errorf("%w: unsupported sync_mode %q", ErrInvalidEnvelope, v.SyncMode)
	}
	reason := NormalizeFullSnapshotReason(v.FullSnapshotReason)
	if mode == SyncModeFullSnapshot && reason == "" {
		return fmt.Errorf("%w: full_snapshot_reason must be terminal_restaurant_changed or node_role_changed", ErrInvalidEnvelope)
	}
	if mode == SyncModeIncremental && strings.TrimSpace(v.FullSnapshotReason) != "" {
		return fmt.Errorf("%w: full_snapshot_reason is allowed only for full_snapshot", ErrInvalidEnvelope)
	}
	if v.CloudVersion <= 0 {
		return fmt.Errorf("%w: cloud_version must be positive", ErrInvalidEnvelope)
	}
	if len(v.PayloadJSON) == 0 || string(v.PayloadJSON) == "null" || string(v.PayloadJSON) == "{}" {
		return fmt.Errorf("%w: payload_json is required", ErrInvalidEnvelope)
	}
	if err := ValidateMasterDataPayload(v.StreamName, v.PayloadJSON); err != nil {
		return err
	}
	return nil
}
