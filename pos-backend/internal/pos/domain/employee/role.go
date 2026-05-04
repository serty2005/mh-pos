package employee

import "time"

type Role struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	PermissionsJSON string    `json:"permissions_json"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
