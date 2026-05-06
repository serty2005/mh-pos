package shared

import (
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/pos/domain"
)

type CommandMeta struct {
	CommandID       string               `json:"command_id,omitempty"`
	NodeDeviceID    string               `json:"node_device_id,omitempty"`
	ClientDeviceID  string               `json:"client_device_id,omitempty"`
	DeviceID        string               `json:"device_id,omitempty"`
	ActorEmployeeID string               `json:"actor_employee_id,omitempty"`
	SessionID       string               `json:"session_id,omitempty"`
	Origin          domain.CommandOrigin `json:"origin,omitempty"`
}

const (
	OriginEdgeDevice = domain.OriginEdgeDevice
	OriginCloudSync  = domain.OriginCloudSync
	OriginSystemSeed = domain.OriginSystemSeed
)

func ValidateWriteMeta(meta CommandMeta) error {
	if EffectiveNodeDeviceID(meta) == "" {
		return fmt.Errorf("%w: node_device_id is required", domain.ErrInvalid)
	}
	switch meta.Origin {
	case "", domain.OriginEdgeDevice, domain.OriginCloudSync, domain.OriginSystemSeed:
		return nil
	default:
		return fmt.Errorf("%w: valid origin is required", domain.ErrInvalid)
	}
}

func EffectiveNodeDeviceID(meta CommandMeta) string {
	if id := strings.TrimSpace(meta.NodeDeviceID); id != "" {
		return id
	}
	return strings.TrimSpace(meta.DeviceID)
}

func NormalizeDeviceMeta(meta *CommandMeta) {
	if strings.TrimSpace(meta.NodeDeviceID) == "" {
		meta.NodeDeviceID = strings.TrimSpace(meta.DeviceID)
	}
	meta.NodeDeviceID = strings.TrimSpace(meta.NodeDeviceID)
	meta.ClientDeviceID = strings.TrimSpace(meta.ClientDeviceID)
	meta.DeviceID = meta.NodeDeviceID
}

func NormalizeOrigin(origin domain.CommandOrigin) domain.CommandOrigin {
	if origin == "" {
		return domain.OriginEdgeDevice
	}
	return origin
}

func OptionalID(id string) *string {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	return &id
}

func DBTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
