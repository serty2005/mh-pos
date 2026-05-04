package restaurant

import "time"

type Restaurant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Timezone  string    `json:"timezone"`
	Currency  string    `json:"currency"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
