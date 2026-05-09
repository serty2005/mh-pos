package employee

import (
	"encoding/json"
	"time"
)

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

// UnmarshalJSON принимает device/system snapshot с pin_hash, не раскрывая hash при обычной сериализации Employee.
func (e *Employee) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID           string    `json:"id"`
		RestaurantID string    `json:"restaurant_id"`
		RoleID       string    `json:"role_id"`
		Name         string    `json:"name"`
		PINHash      string    `json:"pin_hash"`
		Active       bool      `json:"active"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*e = Employee{
		ID:           raw.ID,
		RestaurantID: raw.RestaurantID,
		RoleID:       raw.RoleID,
		Name:         raw.Name,
		PINHash:      raw.PINHash,
		Active:       raw.Active,
		CreatedAt:    raw.CreatedAt,
		UpdatedAt:    raw.UpdatedAt,
	}
	return nil
}
