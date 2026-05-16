package shared

import "encoding/json"

const (
	SyncExchangeProtocolVersion = "sync_exchange.v1"
	SyncExchangeStatusAccepted  = "accepted"
	SyncExchangeStatusPartial   = "partial"
)

type SyncExchangeState struct {
	NodeDeviceID string
	RestaurantID string
	AuthToken    string
	Streams      []SyncExchangeStreamRequest
}

type SyncExchangeRequest struct {
	ProtocolVersion string                      `json:"protocol_version"`
	NodeDeviceID    string                      `json:"node_device_id"`
	RestaurantID    string                      `json:"restaurant_id"`
	AuthToken       string                      `json:"-"`
	EdgeEvents      []SyncExchangeEdgeEvent     `json:"edge_events"`
	Streams         []SyncExchangeStreamRequest `json:"streams"`
}

type SyncExchangeEdgeEvent struct {
	ClientItemID string          `json:"client_item_id"`
	Payload      json.RawMessage `json:"payload"`
}

type SyncExchangeStreamRequest struct {
	StreamName       string `json:"stream_name"`
	LastCloudVersion int64  `json:"last_cloud_version"`
	CheckpointToken  string `json:"checkpoint_token,omitempty"`
}

type SyncExchangeResponse struct {
	Status        string
	EdgeAcks      []BatchSendResult
	CloudPackages []CloudPackage
}

type BatchSendResult struct {
	OutboxID string
	Status   string
	Reason   string
}

type CloudPackage struct {
	StreamName         string          `json:"stream_name"`
	NodeDeviceID       string          `json:"node_device_id,omitempty"`
	RestaurantID       string          `json:"restaurant_id,omitempty"`
	SyncMode           string          `json:"sync_mode"`
	FullSnapshotReason string          `json:"full_snapshot_reason,omitempty"`
	CloudVersion       int64           `json:"cloud_version"`
	CheckpointToken    string          `json:"checkpoint_token,omitempty"`
	CloudUpdatedAt     string          `json:"cloud_updated_at,omitempty"`
	PayloadJSON        json.RawMessage `json:"payload_json"`
}
