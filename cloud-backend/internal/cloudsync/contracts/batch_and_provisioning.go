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

// BatchReceiveRequest contains raw SyncEnvelope items for item-level ACK processing.
type BatchReceiveRequest struct {
	Items []json.RawMessage `json:"items"`
}

// BatchEventAckItem describes per-item receive result.
type BatchEventAckItem struct {
	Index     int             `json:"index"`
	Status    BatchItemStatus `json:"status"`
	ErrorCode string          `json:"error_code,omitempty"`
	Error     string          `json:"error,omitempty"`
	Ack       *EventAck       `json:"ack,omitempty"`
}

// BatchEventAck aggregates item-level ACK decisions for one batch request.
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
)

const (
	SyncModeFullSnapshot = "full_snapshot"
	SyncModeIncremental  = "incremental"
)

// MasterDataPackage stores Cloud-authored provisioning payload for Cloud -> Edge import.
type MasterDataPackage struct {
	StreamName      string          `json:"stream_name"`
	NodeDeviceID    string          `json:"node_device_id,omitempty"`
	RestaurantID    string          `json:"restaurant_id,omitempty"`
	SyncMode        string          `json:"sync_mode"`
	CloudVersion    int64           `json:"cloud_version"`
	CheckpointToken string          `json:"checkpoint_token,omitempty"`
	CloudUpdatedAt  *time.Time      `json:"cloud_updated_at,omitempty"`
	PayloadJSON     json.RawMessage `json:"payload_json"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func ValidateMasterDataStream(streamName string) error {
	switch strings.TrimSpace(streamName) {
	case MasterDataStreamRestaurants, MasterDataStreamDevices, MasterDataStreamStaff, MasterDataStreamFloor, MasterDataStreamCatalog, MasterDataStreamMenu:
		return nil
	default:
		return fmt.Errorf("%w: unsupported stream_name %q", ErrInvalidEnvelope, streamName)
	}
}

func NormalizeSyncMode(syncMode string) string {
	mode := strings.TrimSpace(strings.ToLower(syncMode))
	switch mode {
	case "", SyncModeFullSnapshot:
		return SyncModeFullSnapshot
	case SyncModeIncremental:
		return SyncModeIncremental
	default:
		return ""
	}
}

func ValidateMasterDataPackage(v MasterDataPackage) error {
	if err := ValidateMasterDataStream(v.StreamName); err != nil {
		return err
	}
	if NormalizeSyncMode(v.SyncMode) == "" {
		return fmt.Errorf("%w: unsupported sync_mode %q", ErrInvalidEnvelope, v.SyncMode)
	}
	if v.CloudVersion <= 0 {
		return fmt.Errorf("%w: cloud_version must be positive", ErrInvalidEnvelope)
	}
	if len(v.PayloadJSON) == 0 || string(v.PayloadJSON) == "null" || string(v.PayloadJSON) == "{}" {
		return fmt.Errorf("%w: payload_json is required", ErrInvalidEnvelope)
	}
	return nil
}
