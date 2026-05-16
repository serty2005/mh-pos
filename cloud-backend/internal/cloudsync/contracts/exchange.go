package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	SyncExchangeProtocolVersion = "sync_exchange.v1"
	SyncExchangeMaxEdgeEvents   = 100
	SyncExchangeMaxEnvelopeSize = 2 << 20
)

const (
	SyncExchangeStatusAccepted = "accepted"
	SyncExchangeStatusPartial  = "partial"
)

const (
	SyncExchangeStreamChanged  = "changed"
	SyncExchangeStreamUpToDate = "up_to_date"
	SyncExchangeStreamNotFound = "not_found"
)

var (
	ErrSyncUnauthorized       = errors.New("sync exchange unauthorized")
	ErrSyncForbidden          = errors.New("sync exchange forbidden")
	ErrSyncRevisionAhead      = errors.New("sync exchange revision ahead")
	ErrSyncCheckpointConflict = errors.New("sync exchange checkpoint conflict")
)

// SyncExchangeRequest описывает единый Cloud-Edge exchange cycle v1.
type SyncExchangeRequest struct {
	ProtocolVersion string                      `json:"protocol_version"`
	NodeDeviceID    string                      `json:"node_device_id"`
	RestaurantID    string                      `json:"restaurant_id"`
	EdgeEvents      []SyncExchangeEdgeEvent     `json:"edge_events"`
	Streams         []SyncExchangeStreamRequest `json:"streams"`
}

// SyncExchangeEdgeEvent связывает local Edge outbox row с raw SyncEnvelope.
type SyncExchangeEdgeEvent struct {
	ClientItemID string          `json:"client_item_id"`
	Payload      json.RawMessage `json:"payload"`
}

// SyncExchangeStreamRequest передает известный Edge checkpoint по Cloud-owned stream.
type SyncExchangeStreamRequest struct {
	StreamName       string `json:"stream_name"`
	LastCloudVersion int64  `json:"last_cloud_version"`
	CheckpointToken  string `json:"checkpoint_token,omitempty"`
}

// SyncExchangeResponse содержит item-level Edge ACK и Cloud->Edge deltas.
type SyncExchangeResponse struct {
	ProtocolVersion string                     `json:"protocol_version"`
	Status          string                     `json:"status"`
	EdgeAcks        []SyncExchangeEdgeAck      `json:"edge_acks"`
	CloudPackages   []SyncExchangeCloudPackage `json:"cloud_packages"`
	StreamResults   []SyncExchangeStreamResult `json:"stream_results"`
}

// SyncExchangeEdgeAck описывает безопасный per-item result для Edge outbox row.
type SyncExchangeEdgeAck struct {
	ClientItemID string            `json:"client_item_id"`
	Status       BatchItemStatus   `json:"status"`
	ErrorCode    string            `json:"error_code,omitempty"`
	MessageKey   string            `json:"message_key,omitempty"`
	Details      map[string]string `json:"details,omitempty"`
	Ack          *EventAck         `json:"ack,omitempty"`
}

// SyncExchangeCloudPackage переносит Cloud-authored stream package на Edge.
type SyncExchangeCloudPackage struct {
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

// SyncExchangeStreamResult сообщает Edge, что произошло по запрошенному stream.
type SyncExchangeStreamResult struct {
	StreamName      string `json:"stream_name"`
	Status          string `json:"status"`
	CloudVersion    int64  `json:"cloud_version,omitempty"`
	CheckpointToken string `json:"checkpoint_token,omitempty"`
	ErrorCode       string `json:"error_code,omitempty"`
	MessageKey      string `json:"message_key,omitempty"`
}

func ValidateSyncExchangeRequest(v SyncExchangeRequest) error {
	if strings.TrimSpace(v.ProtocolVersion) != SyncExchangeProtocolVersion {
		return fmt.Errorf("%w: protocol_version must be %s", ErrInvalidEnvelope, SyncExchangeProtocolVersion)
	}
	if strings.TrimSpace(v.NodeDeviceID) == "" || strings.TrimSpace(v.RestaurantID) == "" {
		return fmt.Errorf("%w: node_device_id and restaurant_id are required", ErrInvalidEnvelope)
	}
	if len(v.EdgeEvents) > SyncExchangeMaxEdgeEvents {
		return fmt.Errorf("%w: edge_events length must be <= %d", ErrInvalidEnvelope, SyncExchangeMaxEdgeEvents)
	}
	for _, item := range v.EdgeEvents {
		if strings.TrimSpace(item.ClientItemID) == "" {
			return fmt.Errorf("%w: edge_events.client_item_id is required", ErrInvalidEnvelope)
		}
		raw := bytes.TrimSpace(item.Payload)
		if len(raw) == 0 || string(raw) == "null" {
			return fmt.Errorf("%w: edge_events.payload is required", ErrInvalidEnvelope)
		}
		if len(raw) > SyncExchangeMaxEnvelopeSize {
			return fmt.Errorf("%w: edge_events.payload exceeds max envelope size", ErrInvalidEnvelope)
		}
	}
	seenStreams := map[string]struct{}{}
	for _, stream := range v.Streams {
		name := strings.TrimSpace(stream.StreamName)
		if err := ValidateExchangeStream(name); err != nil {
			return err
		}
		if stream.LastCloudVersion < 0 {
			return fmt.Errorf("%w: last_cloud_version must be non-negative", ErrInvalidEnvelope)
		}
		if _, ok := seenStreams[name]; ok {
			return fmt.Errorf("%w: duplicate stream_name %q", ErrInvalidEnvelope, name)
		}
		seenStreams[name] = struct{}{}
	}
	return nil
}

func ValidateExchangeStream(streamName string) error {
	switch strings.TrimSpace(streamName) {
	case MasterDataStreamRestaurants, MasterDataStreamDevices, MasterDataStreamStaff, MasterDataStreamFloor, MasterDataStreamCatalog, MasterDataStreamMenu, MasterDataStreamPricing:
		return nil
	default:
		return fmt.Errorf("%w: unsupported exchange stream_name %q", ErrInvalidEnvelope, streamName)
	}
}

func DecodeSyncExchangeRequest(raw []byte) (SyncExchangeRequest, error) {
	var req SyncExchangeRequest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return SyncExchangeRequest{}, fmt.Errorf("%w: %v", ErrInvalidEnvelope, err)
	}
	if err := ValidateSyncExchangeRequest(req); err != nil {
		return SyncExchangeRequest{}, err
	}
	return req, nil
}
