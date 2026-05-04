package device

import "time"

type Device struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	DeviceCode   string    `json:"device_code"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Active       bool      `json:"active"`
	RegisteredAt time.Time `json:"registered_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
