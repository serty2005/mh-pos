package inventory

import "time"

type RecipeVersionStatus string

const (
	RecipeVersionDraft    RecipeVersionStatus = "draft"
	RecipeVersionActive   RecipeVersionStatus = "active"
	RecipeVersionArchived RecipeVersionStatus = "archived"
)

type RecipeVersion struct {
	ID                string              `json:"id"`
	DishCatalogItemID string              `json:"dish_catalog_item_id"`
	Version           int                 `json:"version"`
	Name              string              `json:"name"`
	Status            RecipeVersionStatus `json:"status"`
	YieldQuantity     int64               `json:"yield_quantity"`
	YieldUnit         string              `json:"yield_unit"`
	Active            bool                `json:"active"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type RecipeLine struct {
	ID              string    `json:"id"`
	RecipeVersionID string    `json:"recipe_version_id"`
	CatalogItemID   string    `json:"catalog_item_id"`
	Quantity        int64     `json:"quantity"`
	Unit            string    `json:"unit"`
	LossPercent     int       `json:"loss_percent"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PurchaseReceiptStatus string

const (
	PurchaseReceiptDraft     PurchaseReceiptStatus = "draft"
	PurchaseReceiptPosted    PurchaseReceiptStatus = "posted"
	PurchaseReceiptCancelled PurchaseReceiptStatus = "cancelled"
)

type PurchaseReceipt struct {
	ID             string                `json:"id"`
	RestaurantID   string                `json:"restaurant_id"`
	DeviceID       string                `json:"device_id"`
	SupplierName   string                `json:"supplier_name"`
	DocumentNumber string                `json:"document_number"`
	Status         PurchaseReceiptStatus `json:"status"`
	ReceivedAt     time.Time             `json:"received_at"`
	TotalAmount    int64                 `json:"total_amount"`
	Currency       string                `json:"currency"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type PurchaseReceiptLine struct {
	ID                string    `json:"id"`
	PurchaseReceiptID string    `json:"purchase_receipt_id"`
	CatalogItemID     string    `json:"catalog_item_id"`
	Quantity          int64     `json:"quantity"`
	Unit              string    `json:"unit"`
	UnitCost          int64     `json:"unit_cost"`
	TotalCost         int64     `json:"total_cost"`
	Currency          string    `json:"currency"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type StockDocumentType string
type StockDocumentStatus string

const (
	StockDocumentPurchaseReceipt StockDocumentType = "purchase_receipt"
	StockDocumentAdjustment      StockDocumentType = "adjustment"
	StockDocumentTransfer        StockDocumentType = "transfer"
	StockDocumentWriteOff        StockDocumentType = "write_off"
	StockDocumentProduction      StockDocumentType = "production"

	StockDocumentDraft     StockDocumentStatus = "draft"
	StockDocumentPosted    StockDocumentStatus = "posted"
	StockDocumentCancelled StockDocumentStatus = "cancelled"
)

type StockDocument struct {
	ID           string              `json:"id"`
	RestaurantID string              `json:"restaurant_id"`
	DeviceID     string              `json:"device_id"`
	Type         StockDocumentType   `json:"document_type"`
	SourceType   *string             `json:"source_type,omitempty"`
	SourceID     *string             `json:"source_id,omitempty"`
	Status       StockDocumentStatus `json:"status"`
	OccurredAt   time.Time           `json:"occurred_at"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

type StockMoveType string

const (
	StockMoveIn         StockMoveType = "in"
	StockMoveOut        StockMoveType = "out"
	StockMoveAdjustment StockMoveType = "adjustment"
)

type StockMove struct {
	ID              string        `json:"id"`
	StockDocumentID string        `json:"stock_document_id"`
	CatalogItemID   string        `json:"catalog_item_id"`
	LocationID      *string       `json:"location_id,omitempty"`
	Type            StockMoveType `json:"movement_type"`
	Quantity        int64         `json:"quantity"`
	Unit            string        `json:"unit"`
	UnitCost        *int64        `json:"unit_cost,omitempty"`
	TotalCost       *int64        `json:"total_cost,omitempty"`
	OccurredAt      time.Time     `json:"occurred_at"`
	CreatedAt       time.Time     `json:"created_at"`
}

type StockBalance struct {
	ID            string    `json:"id"`
	CatalogItemID string    `json:"catalog_item_id"`
	LocationID    *string   `json:"location_id,omitempty"`
	Quantity      int64     `json:"quantity"`
	Unit          string    `json:"unit"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ItemCostType string

const (
	ItemCostLastPurchase  ItemCostType = "last_purchase"
	ItemCostMovingAverage ItemCostType = "moving_average"
)

type ItemCost struct {
	ID            string       `json:"id"`
	CatalogItemID string       `json:"catalog_item_id"`
	Type          ItemCostType `json:"cost_type"`
	Amount        int64        `json:"amount"`
	Currency      string       `json:"currency"`
	SourceType    *string      `json:"source_type,omitempty"`
	SourceID      *string      `json:"source_id,omitempty"`
	EffectiveAt   time.Time    `json:"effective_at"`
	CreatedAt     time.Time    `json:"created_at"`
}
