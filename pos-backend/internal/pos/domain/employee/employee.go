package employee

import "time"

type Employee struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	RoleID       string    `json:"role_id"`
	Name         string    `json:"name"`
	PINHash      string    `json:"-"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
