package employee

import "time"

type ManagerOverrideAudit struct {
	ID                string    `json:"id"`
	CommandID         string    `json:"command_id"`
	RestaurantID      string    `json:"restaurant_id"`
	DeviceID          string    `json:"device_id"`
	ShiftID           string    `json:"shift_id"`
	OrderID           string    `json:"order_id"`
	PrecheckID        string    `json:"precheck_id"`
	ManagerEmployeeID string    `json:"manager_employee_id"`
	ActorEmployeeID   *string   `json:"actor_employee_id,omitempty"`
	SessionID         *string   `json:"session_id,omitempty"`
	Action            string    `json:"action"`
	Reason            string    `json:"reason"`
	OccurredAt        time.Time `json:"occurred_at"`
	CreatedAt         time.Time `json:"created_at"`
}
