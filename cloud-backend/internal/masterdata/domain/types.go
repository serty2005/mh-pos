package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalid  = errors.New("invalid master data")
	ErrNotFound = errors.New("master data not found")
)

// EmployeeStatus задает Cloud-owned lifecycle сотрудника для sync-safe POS projection.
type EmployeeStatus string

const (
	EmployeeActive    EmployeeStatus = "active"
	EmployeeSuspended EmployeeStatus = "suspended"
	EmployeeArchived  EmployeeStatus = "archived"
)

// LifecycleStatus задает draft/published/archive состояние публикуемых справочников.
type LifecycleStatus string

const (
	StatusDraft     LifecycleStatus = "draft"
	StatusPublished LifecycleStatus = "published"
	StatusArchived  LifecycleStatus = "archived"
)

// CatalogItemKind разделяет базовую номенклатуру по будущим доменным веткам учета.
type CatalogItemKind string

const (
	CatalogItemDish         CatalogItemKind = "dish"
	CatalogItemGood         CatalogItemKind = "good"
	CatalogItemRawMaterial  CatalogItemKind = "raw_material"
	CatalogItemSemiFinished CatalogItemKind = "semi_finished"
)

// Role описывает Cloud-authored роль и snapshot прав для доставки на POS Edge.
type Role struct {
	ID              string    `json:"id"`
	RestaurantID    string    `json:"restaurant_id"`
	Name            string    `json:"name"`
	PermissionsJSON string    `json:"permissions_json"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Employee описывает Cloud-authored сотрудника без раскрытия PIN credential в API JSON.
type Employee struct {
	ID                     string         `json:"id"`
	RestaurantID           string         `json:"restaurant_id"`
	RoleID                 string         `json:"role_id"`
	Name                   string         `json:"name"`
	Status                 EmployeeStatus `json:"status"`
	PINHash                string         `json:"-"`
	PINCredentialVersion   int64          `json:"pin_credential_version"`
	PermissionSnapshotJSON string         `json:"permission_snapshot_json"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	SuspendedAt            *time.Time     `json:"suspended_at,omitempty"`
	ArchivedAt             *time.Time     `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, может ли сотрудник участвовать в POS login после sync.
func (e Employee) ActiveForPOS() bool {
	return e.Status == EmployeeActive
}

// Category описывает Cloud-authored категорию меню.
type Category struct {
	ID           string          `json:"id"`
	RestaurantID string          `json:"restaurant_id"`
	Name         string          `json:"name"`
	Status       LifecycleStatus `json:"status"`
	SortOrder    int64           `json:"sort_order"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CatalogItem описывает общую Cloud-owned номенклатуру без сведения всех видов в одну финальную модель.
type CatalogItem struct {
	ID           string          `json:"id"`
	RestaurantID string          `json:"restaurant_id"`
	Kind         CatalogItemKind `json:"kind"`
	Name         string          `json:"name"`
	SKU          string          `json:"sku"`
	BaseUnit     string          `json:"base_unit"`
	Status       LifecycleStatus `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ActiveForPOS сообщает, должен ли catalog item быть активным в Edge read model.
func (c CatalogItem) ActiveForPOS() bool {
	return c.Status == StatusPublished
}

// EdgeType возвращает текущий POS Edge-compatible тип для существующего ingest contract.
func (c CatalogItem) EdgeType() string {
	switch c.Kind {
	case CatalogItemDish:
		return "dish"
	case CatalogItemGood:
		return "good"
	case CatalogItemRawMaterial, CatalogItemSemiFinished:
		return "ingredient"
	default:
		return string(c.Kind)
	}
}

// MenuItem описывает продаваемую позицию меню с lifecycle и основой routing/availability.
type MenuItem struct {
	ID                string          `json:"id"`
	RestaurantID      string          `json:"restaurant_id"`
	CatalogItemID     string          `json:"catalog_item_id"`
	CategoryID        string          `json:"category_id,omitempty"`
	Name              string          `json:"name"`
	Price             int64           `json:"price"`
	Currency          string          `json:"currency"`
	Status            LifecycleStatus `json:"status"`
	AvailabilityJSON  string          `json:"availability_json"`
	StationRoutingKey string          `json:"station_routing_key,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// ActiveForPOS сообщает, должна ли позиция меню быть доступна в POS runtime.
func (m MenuItem) ActiveForPOS() bool {
	return m.Status == StatusPublished
}

// Publication фиксирует опубликованный versioned snapshot Cloud-authored master data.
type Publication struct {
	ID            string          `json:"id"`
	RestaurantID  string          `json:"restaurant_id"`
	Version       int64           `json:"version"`
	Status        LifecycleStatus `json:"status"`
	CloudVersion  int64           `json:"cloud_version"`
	PublishedAt   time.Time       `json:"published_at"`
	PublishedBy   string          `json:"published_by"`
	PackageJSON   json.RawMessage `json:"package_json"`
	PackageSHA256 string          `json:"package_sha256"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// PublishedState связывает публикацию с пакетом для внутренних проверок и будущих query API.
type PublishedState struct {
	Publication Publication      `json:"publication"`
	Package     MasterDataPacket `json:"package"`
}

// MasterDataPacket описывает deterministic Cloud -> Edge package payload.
type MasterDataPacket struct {
	NodeDeviceID    string               `json:"node_device_id,omitempty"`
	RestaurantID    string               `json:"restaurant_id"`
	SyncMode        string               `json:"sync_mode"`
	CheckpointToken string               `json:"checkpoint_token,omitempty"`
	CloudVersion    int64                `json:"cloud_version"`
	CloudUpdatedAt  time.Time            `json:"cloud_updated_at"`
	Roles           []EdgeRole           `json:"roles,omitempty"`
	Employees       []EdgeEmployee       `json:"employees,omitempty"`
	CatalogItems    []EdgeCatalogItem    `json:"catalog_items,omitempty"`
	MenuItems       []EdgeMenuItem       `json:"menu_items,omitempty"`
	Categories      []EdgeCategory       `json:"categories,omitempty"`
	ModifierGroups  []EdgeModifierGroup  `json:"modifier_groups,omitempty"`
	ModifierOptions []EdgeModifierOption `json:"modifier_options,omitempty"`
}

// EdgeRole является projection роли в существующий POS Edge staff stream.
type EdgeRole struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	PermissionsJSON string    `json:"permissions_json"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// EdgeEmployee является projection сотрудника в существующий POS Edge staff stream.
type EdgeEmployee struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	RoleID       string    `json:"role_id"`
	Name         string    `json:"name"`
	PINHash      string    `json:"pin_hash"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EdgeCatalogItem является projection catalog item в существующий POS Edge catalog stream.
type EdgeCatalogItem struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	BaseUnit  string    `json:"base_unit"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EdgeMenuItem является projection menu item в существующий POS Edge menu stream.
type EdgeMenuItem struct {
	ID            string    `json:"id"`
	CatalogItemID string    `json:"catalog_item_id"`
	Name          string    `json:"name"`
	Price         int64     `json:"price"`
	Currency      string    `json:"currency"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EdgeCategory является foundation projection категории для будущего Edge menu layout.
type EdgeCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int64  `json:"sort_order"`
	Active    bool   `json:"active"`
}

// EdgeModifierGroup является foundation projection modifier group для будущего Edge menu layout.
type EdgeModifierGroup struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Required bool   `json:"required"`
	MinCount int64  `json:"min_count"`
	MaxCount int64  `json:"max_count"`
	Active   bool   `json:"active"`
}

// EdgeModifierOption является foundation projection modifier option для будущего Edge menu layout.
type EdgeModifierOption struct {
	ID              string `json:"id"`
	ModifierGroupID string `json:"modifier_group_id"`
	Name            string `json:"name"`
	PriceDelta      int64  `json:"price_delta"`
	Active          bool   `json:"active"`
}

// ValidateEmployeeStatus проверяет допустимое lifecycle состояние сотрудника.
func ValidateEmployeeStatus(v EmployeeStatus) error {
	switch v {
	case EmployeeActive, EmployeeSuspended, EmployeeArchived:
		return nil
	default:
		return fmt.Errorf("%w: unsupported employee status %q", ErrInvalid, v)
	}
}

// ValidateLifecycleStatus проверяет допустимое lifecycle состояние справочника.
func ValidateLifecycleStatus(v LifecycleStatus) error {
	switch v {
	case StatusDraft, StatusPublished, StatusArchived:
		return nil
	default:
		return fmt.Errorf("%w: unsupported lifecycle status %q", ErrInvalid, v)
	}
}

// ValidateCatalogItemKind проверяет допустимый Cloud catalog item kind.
func ValidateCatalogItemKind(v CatalogItemKind) error {
	switch v {
	case CatalogItemDish, CatalogItemGood, CatalogItemRawMaterial, CatalogItemSemiFinished:
		return nil
	default:
		return fmt.Errorf("%w: unsupported catalog item kind %q", ErrInvalid, v)
	}
}

// ValidatePermissionsJSON проверяет, что snapshot прав является валидным JSON.
func ValidatePermissionsJSON(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%w: permissions_json is required", ErrInvalid)
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fmt.Errorf("%w: permissions_json must be valid JSON", ErrInvalid)
	}
	return nil
}
