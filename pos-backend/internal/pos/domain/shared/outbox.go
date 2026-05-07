package shared

import "time"

type OutboxStatus string
type CommandOrigin string

const (
	OutboxPending    OutboxStatus = "pending"
	OutboxProcessing OutboxStatus = "processing"
	OutboxSent       OutboxStatus = "sent"
	OutboxFailed     OutboxStatus = "failed"
	OutboxSuspended  OutboxStatus = "suspended"

	OriginEdgeDevice CommandOrigin = "edge_device"
	OriginCloudSync  CommandOrigin = "cloud_sync"
	OriginSystemSeed CommandOrigin = "system_seed"
)

type OutboxMessage struct {
	ID              string        `json:"id"`
	CommandID       string        `json:"command_id"`
	SequenceNo      int64         `json:"sequence_no"`
	Origin          CommandOrigin `json:"origin"`
	RestaurantID    *string       `json:"restaurant_id,omitempty"`
	DeviceID        string        `json:"device_id"`
	NodeDeviceID    string        `json:"node_device_id"`
	ClientDeviceID  *string       `json:"client_device_id,omitempty"`
	ActorEmployeeID *string       `json:"actor_employee_id,omitempty"`
	SessionID       *string       `json:"session_id,omitempty"`
	AggregateType   string        `json:"aggregate_type"`
	AggregateID     string        `json:"aggregate_id"`
	CommandType     string        `json:"command_type"`
	SyncDirection   SyncDirection `json:"sync_direction"`
	PayloadJSON     string        `json:"payload_json"`
	Status          OutboxStatus  `json:"status"`
	Attempts        int           `json:"attempts"`
	NextRetryAt     *time.Time    `json:"next_retry_at,omitempty"`
	LockedAt        *time.Time    `json:"locked_at,omitempty"`
	LockedBy        *string       `json:"locked_by,omitempty"`
	SentAt          *time.Time    `json:"sent_at,omitempty"`
	LastError       *string       `json:"last_error,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type SyncStatus struct {
	Total                   int    `json:"total"`
	Pending                 int    `json:"pending"`
	Processing              int    `json:"processing"`
	Sent                    int    `json:"sent"`
	Failed                  int    `json:"failed"`
	Suspended               int    `json:"suspended"`
	OldestPendingSequenceNo *int64 `json:"oldest_pending_sequence_no,omitempty"`
}
