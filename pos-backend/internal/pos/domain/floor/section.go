package floor

import "time"

type RestaurantSectionMode string

const (
	RestaurantSectionHallSection     RestaurantSectionMode = "hall_section"
	RestaurantSectionKitchenWorkshop RestaurantSectionMode = "kitchen_workshop"
)

type RestaurantSection struct {
	ID                string                `json:"id"`
	RestaurantID      string                `json:"restaurant_id"`
	Name              string                `json:"name"`
	Mode              RestaurantSectionMode `json:"mode"`
	HallID            string                `json:"hall_id,omitempty"`
	KitchenRoutingKey string                `json:"kitchen_routing_key,omitempty"`
	WarehouseID       string                `json:"warehouse_id,omitempty"`
	IsDefault         bool                  `json:"is_default"`
	IsActive          bool                  `json:"is_active"`
	CloudVersion      int                   `json:"cloud_version"`
	SyncedAt          *time.Time            `json:"synced_at,omitempty"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

type SalesPoint struct {
	ID             string     `json:"id"`
	RestaurantID   string     `json:"restaurant_id"`
	Name           string     `json:"name"`
	AnalyticsTag   string     `json:"analytics_tag"`
	DefaultTableID string     `json:"default_table_id"`
	IsActive       bool       `json:"is_active"`
	CloudVersion   int        `json:"cloud_version"`
	SyncedAt       *time.Time `json:"synced_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
