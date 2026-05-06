package employee

import "time"

type AuthSessionStatus string

const (
	AuthSessionActive  AuthSessionStatus = "active"
	AuthSessionRevoked AuthSessionStatus = "revoked"
)

type AuthSession struct {
	ID           string            `json:"id"`
	RestaurantID string            `json:"restaurant_id"`
	DeviceID     string            `json:"device_id"`
	EmployeeID   string            `json:"employee_id"`
	Status       AuthSessionStatus `json:"status"`
	StartedAt    time.Time         `json:"started_at"`
	LastSeenAt   time.Time         `json:"last_seen_at"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type ActorContext struct {
	EmployeeID   string   `json:"employee_id"`
	RestaurantID string   `json:"restaurant_id"`
	RoleID       string   `json:"role_id"`
	Name         string   `json:"name"`
	Permissions  []string `json:"permissions"`
}

type PinLoginResult struct {
	Session     AuthSession  `json:"session"`
	Actor       ActorContext `json:"actor"`
	Permissions []string     `json:"permissions"`
}
