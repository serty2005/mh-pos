package shared

import "time"

type OutboxStatus string
type CommandOrigin string

const (
	OutboxPending OutboxStatus = "pending"
	OutboxSent    OutboxStatus = "sent"
	OutboxFailed  OutboxStatus = "failed"

	OriginEdgeDevice CommandOrigin = "edge_device"
	OriginCloudSync  CommandOrigin = "cloud_sync"
	OriginSystemSeed CommandOrigin = "system_seed"
)

type OutboxMessage struct {
	ID            string        `json:"id"`
	CommandID     string        `json:"command_id"`
	Origin        CommandOrigin `json:"origin"`
	RestaurantID  *string       `json:"restaurant_id,omitempty"`
	DeviceID      string        `json:"device_id"`
	AggregateType string        `json:"aggregate_type"`
	AggregateID   string        `json:"aggregate_id"`
	CommandType   string        `json:"command_type"`
	PayloadJSON   string        `json:"payload_json"`
	Status        OutboxStatus  `json:"status"`
	Attempts      int           `json:"attempts"`
	LastError     *string       `json:"last_error,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}
