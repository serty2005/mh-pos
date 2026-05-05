package shared

import "time"

const SyncEnvelopeVersion = "1"

type LocalEvent struct {
	ID              string    `json:"id"`
	EventID         string    `json:"event_id"`
	EnvelopeVersion string    `json:"envelope_version"`
	EventType       string    `json:"event_type"`
	AggregateType   string    `json:"aggregate_type"`
	AggregateID     string    `json:"aggregate_id"`
	RestaurantID    *string   `json:"restaurant_id,omitempty"`
	DeviceID        string    `json:"device_id"`
	ShiftID         *string   `json:"shift_id,omitempty"`
	PayloadJSON     string    `json:"payload_json"`
	OccurredAt      time.Time `json:"occurred_at"`
	CreatedAt       time.Time `json:"created_at"`
}

type SyncEnvelope struct {
	Version       string    `json:"version"`
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	AggregateType string    `json:"aggregate_type"`
	AggregateID   string    `json:"aggregate_id"`
	RestaurantID  *string   `json:"restaurant_id,omitempty"`
	DeviceID      string    `json:"device_id"`
	ShiftID       *string   `json:"shift_id,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
	Payload       any       `json:"payload"`
}
