package catalog

import "time"

type CatalogItemType string

const (
	CatalogItemDish         CatalogItemType = "dish"
	CatalogItemGood         CatalogItemType = "good"
	CatalogItemSemiFinished CatalogItemType = "semi_finished"
	CatalogItemService      CatalogItemType = "service"
)

type CatalogItem struct {
	ID                 string          `json:"id"`
	Type               CatalogItemType `json:"type"`
	FolderID           *string         `json:"folder_id,omitempty"`
	Name               string          `json:"name"`
	SKU                string          `json:"sku"`
	BaseUnit           string          `json:"base_unit"`
	KitchenType        string          `json:"kitchen_type,omitempty"`
	AccountingCategory string          `json:"accounting_category,omitempty"`
	Active             bool            `json:"active"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type CatalogFolder struct {
	ID           string  `json:"id"`
	RestaurantID string  `json:"restaurant_id"`
	ParentID     *string `json:"parent_id,omitempty"`
	Name         string  `json:"name"`
	SortOrder    int     `json:"sort_order"`
	Active       bool    `json:"active"`
}

type FolderParameter struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
	FolderID     string `json:"folder_id"`
	ParameterKey string `json:"parameter_key"`
	ValueType    string `json:"value_type"`
	ValueJSON    string `json:"value_json"`
}

type CatalogTag struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
	Active       bool   `json:"active"`
}

type CatalogItemTag struct {
	CatalogItemID string `json:"catalog_item_id"`
	TagID         string `json:"tag_id"`
	RestaurantID  string `json:"restaurant_id"`
}

type ModifierTargetType string

const (
	ModifierTargetMenuItem    ModifierTargetType = "menu_item"
	ModifierTargetCatalogItem ModifierTargetType = "catalog_item"
	ModifierTargetFolder      ModifierTargetType = "folder"
	ModifierTargetTag         ModifierTargetType = "tag"
)

type ModifierGroup struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Required     bool   `json:"required"`
	MinCount     int    `json:"min_count"`
	MaxCount     int    `json:"max_count"`
	Active       bool   `json:"active"`
}

type ModifierOption struct {
	ID              string `json:"id"`
	RestaurantID    string `json:"restaurant_id"`
	ModifierGroupID string `json:"modifier_group_id"`
	Name            string `json:"name"`
	PriceMinor      int64  `json:"price_minor"`
	Active          bool   `json:"active"`
}

type ModifierGroupBinding struct {
	ID              string             `json:"id"`
	RestaurantID    string             `json:"restaurant_id"`
	ModifierGroupID string             `json:"modifier_group_id"`
	TargetType      ModifierTargetType `json:"target_type"`
	TargetID        string             `json:"target_id"`
	SortOrder       int                `json:"sort_order"`
	Active          bool               `json:"active"`
}

type MenuItemModifierGroup struct {
	MenuItemID      string `json:"menu_item_id"`
	ModifierGroupID string `json:"modifier_group_id"`
	SortOrder       int    `json:"sort_order"`
}
