package restaurant

import "time"

type BusinessDayMode string

const (
	BusinessDayStandard BusinessDayMode = "standard"
	BusinessDay24x7     BusinessDayMode = "24_7"
)

type Restaurant struct {
	ID                           string          `json:"id"`
	Name                         string          `json:"name"`
	Timezone                     string          `json:"timezone"`
	Currency                     string          `json:"currency"`
	BusinessDayMode              BusinessDayMode `json:"business_day_mode"`
	BusinessDayBoundaryLocalTime string          `json:"business_day_boundary_local_time"`
	Active                       bool            `json:"active"`
	CreatedAt                    time.Time       `json:"created_at"`
	UpdatedAt                    time.Time       `json:"updated_at"`
}
