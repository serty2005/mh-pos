package cash

import "time"

type CashSessionStatus string
type CashDrawerEventType string

const (
	CashSessionOpen   CashSessionStatus = "open"
	CashSessionClosed CashSessionStatus = "closed"

	CashDrawerCashIn    CashDrawerEventType = "cash_in"
	CashDrawerCashOut   CashDrawerEventType = "cash_out"
	CashDrawerNoSale    CashDrawerEventType = "no_sale"
	CashDrawerCashCount CashDrawerEventType = "cash_count"
)

type CashSession struct {
	ID                 string            `json:"id"`
	EdgeCashSessionID  string            `json:"edge_cash_session_id"`
	RestaurantID       string            `json:"restaurant_id"`
	DeviceID           string            `json:"device_id"`
	SalesPointID       string            `json:"sales_point_id"`
	ShiftID            string            `json:"shift_id"`
	OpenedByEmployeeID string            `json:"opened_by_employee_id"`
	ClosedByEmployeeID *string           `json:"closed_by_employee_id,omitempty"`
	Status             CashSessionStatus `json:"status"`
	BusinessDateLocal  string            `json:"business_date_local"`
	OpeningCashAmount  int64             `json:"opening_cash_amount"`
	ClosingCashAmount  *int64            `json:"closing_cash_amount,omitempty"`
	OpenedAt           time.Time         `json:"opened_at"`
	ClosedAt           *time.Time        `json:"closed_at,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type CashDrawerEvent struct {
	ID                    string              `json:"id"`
	EdgeCashDrawerEventID string              `json:"edge_cash_drawer_event_id"`
	CashSessionID         string              `json:"cash_session_id"`
	RestaurantID          string              `json:"restaurant_id"`
	DeviceID              string              `json:"device_id"`
	ShiftID               string              `json:"shift_id"`
	CreatedByEmployeeID   string              `json:"created_by_employee_id"`
	EventType             CashDrawerEventType `json:"event_type"`
	Amount                int64               `json:"amount"`
	Reason                *string             `json:"reason,omitempty"`
	Note                  *string             `json:"note,omitempty"`
	OccurredAt            time.Time           `json:"occurred_at"`
	CreatedAt             time.Time           `json:"created_at"`
}
