package shift

import "time"

type ShiftStatus string

const (
	ShiftOpen   ShiftStatus = "open"
	ShiftClosed ShiftStatus = "closed"
)

type Shift struct {
	ID                 string      `json:"id"`
	RestaurantID       string      `json:"restaurant_id"`
	DeviceID           string      `json:"device_id"`
	OpenedByEmployeeID string      `json:"opened_by_employee_id"`
	ClosedByEmployeeID *string     `json:"closed_by_employee_id,omitempty"`
	Status             ShiftStatus `json:"status"`
	BusinessDateLocal  string      `json:"business_date_local"`
	OpenedAt           time.Time   `json:"opened_at"`
	ClosedAt           *time.Time  `json:"closed_at,omitempty"`
	OpeningCashAmount  int64       `json:"-"`
	ClosingCashAmount  *int64      `json:"-"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}
