package floor

import "time"

type Hall struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	Name         string    `json:"name"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Table struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	HallID       string    `json:"hall_id,omitempty"`
	SectionID    string    `json:"section_id"`
	Name         string    `json:"name"`
	Seats        int       `json:"seats"`
	IsDefault    bool      `json:"is_default"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
