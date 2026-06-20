package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalid          = errors.New("invalid master data")
	ErrNotFound         = errors.New("master data not found")
	ErrConflict         = errors.New("master data conflict")
	ErrPINAlreadyExists = errors.New("pin already exists")
)

// RestaurantStatus задает Cloud-owned lifecycle ресторана.
type RestaurantStatus string

const (
	RestaurantActive   RestaurantStatus = "active"
	RestaurantArchived RestaurantStatus = "archived"
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
	CatalogItemSemiFinished CatalogItemKind = "semi_finished"
	CatalogItemService      CatalogItemKind = "service"
)

// Restaurant описывает Cloud-owned ресторан и настройки учетного дня.
type Restaurant struct {
	ID                           string           `json:"id"`
	Name                         string           `json:"name"`
	Timezone                     string           `json:"timezone"`
	Currency                     string           `json:"currency"`
	BusinessDayMode              string           `json:"business_day_mode"`
	BusinessDayBoundaryLocalTime string           `json:"business_day_boundary_local_time"`
	Status                       RestaurantStatus `json:"status"`
	CloudVersion                 int64            `json:"cloud_version"`
	CreatedAt                    time.Time        `json:"created_at"`
	UpdatedAt                    time.Time        `json:"updated_at"`
	ArchivedAt                   *time.Time       `json:"archived_at,omitempty"`
}

// Role описывает Cloud-authored роль и snapshot прав для доставки на POS Edge.
type Role struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	PermissionsJSON string     `json:"permissions_json"`
	Active          bool       `json:"active"`
	CloudVersion    int64      `json:"cloud_version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
}

// ManagesOrganization сообщает, что роль охватывает все рестораны tenant без memberships.
func (r Role) ManagesOrganization() bool {
	var value map[string]any
	if json.Unmarshal([]byte(r.PermissionsJSON), &value) != nil {
		return false
	}
	for _, permission := range permissionsFromJSON(value) {
		if permission == "organization.manage" {
			return true
		}
	}
	return false
}

// Employee описывает Cloud-authored сотрудника без раскрытия PIN credential в API JSON.
type Employee struct {
	ID                     string         `json:"id"`
	RoleID                 string         `json:"role_id"`
	RestaurantIDs          []string       `json:"restaurant_ids"`
	AllRestaurants         bool           `json:"all_restaurants"`
	Name                   string         `json:"name"`
	Status                 EmployeeStatus `json:"status"`
	PINHash                string         `json:"-"`
	PINConfigured          bool           `json:"pin_configured"`
	PINCredentialVersion   int64          `json:"pin_credential_version"`
	PermissionSnapshotJSON string         `json:"permission_snapshot_json"`
	CloudVersion           int64          `json:"cloud_version"`
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

// Hall описывает Cloud-owned зал ресторана для доставки floor read model на POS Edge.
type Hall struct {
	ID           string          `json:"id"`
	RestaurantID string          `json:"restaurant_id"`
	Name         string          `json:"name"`
	Status       LifecycleStatus `json:"status"`
	CloudVersion int64           `json:"cloud_version"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ArchivedAt   *time.Time      `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должен ли зал быть доступен в POS runtime.
func (h Hall) ActiveForPOS() bool {
	return h.Status == StatusPublished
}

// Table описывает Cloud-owned стол ресторана для доставки floor read model на POS Edge.
type Table struct {
	ID           string          `json:"id"`
	RestaurantID string          `json:"restaurant_id"`
	HallID       string          `json:"hall_id"`
	Name         string          `json:"name"`
	Seats        int64           `json:"seats"`
	Status       LifecycleStatus `json:"status"`
	CloudVersion int64           `json:"cloud_version"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ArchivedAt   *time.Time      `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должен ли стол быть доступен в POS runtime.
func (t Table) ActiveForPOS() bool {
	return t.Status == StatusPublished
}

// CatalogItem описывает tenant-owned номенклатуру без restaurant ownership.
type CatalogItem struct {
	ID                 string          `json:"id"`
	RestaurantID       string          `json:"restaurant_id,omitempty"`
	Kind               CatalogItemKind `json:"kind"`
	FolderID           string          `json:"folder_id,omitempty"`
	Name               string          `json:"name"`
	SKU                string          `json:"sku"`
	BaseUnit           string          `json:"base_unit"`
	KitchenType        string          `json:"kitchen_type,omitempty"`
	AccountingCategory string          `json:"accounting_category,omitempty"`
	Status             LifecycleStatus `json:"status"`
	CloudVersion       int64           `json:"cloud_version"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	ArchivedAt         *time.Time      `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должен ли catalog item быть активным в Edge read model.
func (c CatalogItem) ActiveForPOS() bool {
	return c.Status == StatusPublished
}

// EdgeType возвращает текущий POS Edge-compatible тип для существующего ingest contract.
func (c CatalogItem) EdgeType() string {
	return string(c.Kind)
}

// CatalogFolder описывает tenant-owned папку номенклатуры, отдельную от категорий меню.
type CatalogFolder struct {
	ID           string          `json:"id"`
	RestaurantID string          `json:"restaurant_id,omitempty"`
	ParentID     string          `json:"parent_id,omitempty"`
	Name         string          `json:"name"`
	SortOrder    int64           `json:"sort_order"`
	Status       LifecycleStatus `json:"status"`
	CloudVersion int64           `json:"cloud_version"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ArchivedAt   *time.Time      `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должна ли папка номенклатуры быть опубликована на Edge.
func (f CatalogFolder) ActiveForPOS() bool {
	return f.Status == StatusPublished
}

// FolderParameter задает наследуемый параметр папки номенклатуры в расширяемом формате.
type FolderParameter struct {
	ID           string          `json:"id"`
	RestaurantID string          `json:"restaurant_id"`
	FolderID     string          `json:"folder_id"`
	Key          string          `json:"parameter_key"`
	ValueType    string          `json:"value_type"`
	ValueJSON    string          `json:"value_json"`
	Status       LifecycleStatus `json:"status"`
	CloudVersion int64           `json:"cloud_version"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ArchivedAt   *time.Time      `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должен ли параметр папки участвовать в публикации.
func (p FolderParameter) ActiveForPOS() bool {
	return p.Status == StatusPublished
}

// CatalogTag описывает tenant-owned аналитическую метку каталога.
type CatalogTag struct {
	ID           string          `json:"id"`
	RestaurantID string          `json:"restaurant_id,omitempty"`
	Name         string          `json:"name"`
	Code         string          `json:"code"`
	Status       LifecycleStatus `json:"status"`
	CloudVersion int64           `json:"cloud_version"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ArchivedAt   *time.Time      `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должна ли метка каталога быть опубликована на Edge.
func (t CatalogTag) ActiveForPOS() bool {
	return t.Status == StatusPublished
}

// CatalogItemTag связывает позицию каталога с аналитической меткой.
type CatalogItemTag struct {
	RestaurantID    string    `json:"restaurant_id,omitempty"`
	CatalogItemID   string    `json:"catalog_item_id"`
	TagID           string    `json:"tag_id"`
	CloudVersion    int64     `json:"cloud_version"`
	CreatedAt       time.Time `json:"created_at"`
	LastSyncedAtUTC time.Time `json:"-"`
}

// ModifierTargetType задает тип цели, к которой привязана группа модификаторов.
type ModifierTargetType string

const (
	ModifierTargetMenuItem    ModifierTargetType = "menu_item"
	ModifierTargetCatalogItem ModifierTargetType = "catalog_item"
	ModifierTargetFolder      ModifierTargetType = "folder"
	ModifierTargetTag         ModifierTargetType = "tag"
)

// ModifierGroup описывает Cloud-owned группу модификаторов.
type ModifierGroup struct {
	ID           string          `json:"id"`
	RestaurantID string          `json:"restaurant_id"`
	Name         string          `json:"name"`
	Status       LifecycleStatus `json:"status"`
	Required     bool            `json:"required"`
	MinCount     int64           `json:"min_count"`
	MaxCount     int64           `json:"max_count"`
	CloudVersion int64           `json:"cloud_version"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ArchivedAt   *time.Time      `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должна ли группа модификаторов быть опубликована на Edge.
func (g ModifierGroup) ActiveForPOS() bool {
	return g.Status == StatusPublished
}

// ModifierOption описывает вариант модификатора с канонической ценой, а не price_delta.
type ModifierOption struct {
	ID                  string          `json:"id"`
	RestaurantID        string          `json:"restaurant_id"`
	ModifierGroupID     string          `json:"modifier_group_id"`
	LinkedCatalogItemID string          `json:"linked_catalog_item_id,omitempty"`
	Name                string          `json:"name"`
	PriceMinor          int64           `json:"price_minor"`
	Status              LifecycleStatus `json:"status"`
	CloudVersion        int64           `json:"cloud_version"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	ArchivedAt          *time.Time      `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должен ли вариант модификатора быть опубликован на Edge.
func (o ModifierOption) ActiveForPOS() bool {
	return o.Status == StatusPublished
}

// ModifierGroupBinding задает явную привязку группы модификаторов к menu item, catalog item, folder или tag.
type ModifierGroupBinding struct {
	ID              string             `json:"id"`
	RestaurantID    string             `json:"restaurant_id"`
	ModifierGroupID string             `json:"modifier_group_id"`
	TargetType      ModifierTargetType `json:"target_type"`
	TargetID        string             `json:"target_id"`
	SortOrder       int64              `json:"sort_order"`
	Status          LifecycleStatus    `json:"status"`
	CloudVersion    int64              `json:"cloud_version"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	ArchivedAt      *time.Time         `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должна ли привязка модификаторов быть опубликована на Edge.
func (b ModifierGroupBinding) ActiveForPOS() bool {
	return b.Status == StatusPublished
}

// PricingPolicyKind различает Cloud-owned правила скидок и надбавок.
type PricingPolicyKind string

const (
	PricingPolicyDiscount  PricingPolicyKind = "discount"
	PricingPolicySurcharge PricingPolicyKind = "surcharge"
)

// PricingPolicy описывает Cloud-authored правило скидки или надбавки для Edge calculator.
type PricingPolicy struct {
	ID                 string            `json:"id"`
	RestaurantID       string            `json:"restaurant_id"`
	Name               string            `json:"name"`
	Kind               PricingPolicyKind `json:"kind"`
	Scope              string            `json:"scope"`
	AmountKind         string            `json:"amount_kind"`
	AmountMinor        int64             `json:"amount_minor,omitempty"`
	ValueBasisPoints   int64             `json:"value_basis_points,omitempty"`
	ApplicationIndex   int               `json:"application_index"`
	Manual             bool              `json:"manual"`
	RequiresPermission string            `json:"requires_permission,omitempty"`
	Status             LifecycleStatus   `json:"status"`
	CloudVersion       int64             `json:"cloud_version"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	ArchivedAt         *time.Time        `json:"archived_at,omitempty"`
}

// ActiveForPOS сообщает, должна ли pricing policy применяться на Edge.
func (p PricingPolicy) ActiveForPOS() bool {
	return p.Status == StatusPublished
}

// RecipeItem описывает Cloud-owned строку рецепта, из которой публикация строит Edge recipe read model.
type RecipeItem struct {
	ID                       string    `json:"id"`
	RestaurantID             string    `json:"restaurant_id"`
	RecipeOwnerCatalogItemID string    `json:"recipe_owner_catalog_item_id"`
	ComponentCatalogItemID   string    `json:"component_catalog_item_id"`
	Quantity                 int64     `json:"quantity"`
	Unit                     string    `json:"unit"`
	LossPercent              int64     `json:"loss_percent"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// RecipeVersionStatus задает Cloud lifecycle версии техкарты до публикации на Edge.
type RecipeVersionStatus string

const (
	RecipeVersionStatusDraft         RecipeVersionStatus = "draft"
	RecipeVersionStatusReviewPending RecipeVersionStatus = "review_pending"
	RecipeVersionStatusActive        RecipeVersionStatus = "active"
	RecipeVersionStatusArchived      RecipeVersionStatus = "archived"
)

// RecipeVersion описывает Cloud-owned версию техкарты; POS Edge получает только опубликованную read model.
type RecipeVersion struct {
	ID                    string              `json:"id"`
	RestaurantID          string              `json:"restaurant_id"`
	OwnerCatalogItemID    string              `json:"owner_catalog_item_id"`
	Version               int                 `json:"version"`
	Name                  string              `json:"name"`
	Status                RecipeVersionStatus `json:"status"`
	YieldQuantity         int64               `json:"yield_quantity"`
	YieldUnit             string              `json:"yield_unit"`
	CreatedByEmployeeID   string              `json:"created_by_employee_id,omitempty"`
	SubmittedByEmployeeID string              `json:"submitted_by_employee_id,omitempty"`
	ApprovedByEmployeeID  string              `json:"approved_by_employee_id,omitempty"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
	SubmittedAt           *time.Time          `json:"submitted_at,omitempty"`
	ApprovedAt            *time.Time          `json:"approved_at,omitempty"`
}

// RecipeLine описывает строку Cloud-owned версии техкарты.
type RecipeLine struct {
	ID                     string    `json:"id"`
	RecipeVersionID        string    `json:"recipe_version_id"`
	ComponentCatalogItemID string    `json:"component_catalog_item_id"`
	Quantity               int64     `json:"quantity"`
	Unit                   string    `json:"unit"`
	LossPercent            int64     `json:"loss_percent"`
	SortOrder              int       `json:"sort_order"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// StopListEntry описывает Cloud-owned состояние stop-list, публикуемое на Edge для offline sale blocking.
type StopListEntry struct {
	ID                string    `json:"id"`
	RestaurantID      string    `json:"restaurant_id"`
	CatalogItemID     string    `json:"catalog_item_id"`
	AvailableQuantity *float64  `json:"available_quantity,omitempty"`
	Source            string    `json:"source"`
	Reason            string    `json:"reason,omitempty"`
	Active            bool      `json:"active"`
	CloudVersion      *int64    `json:"cloud_version,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// StopListUpdateReview хранит безопасную карточку Cloud review для Edge-origin StopListUpdated без raw payload.
type StopListUpdateReview struct {
	ID                   string           `json:"id"`
	RestaurantID         string           `json:"restaurant_id"`
	DeviceID             string           `json:"device_id"`
	StopListID           string           `json:"stop_list_id"`
	WarehouseID          string           `json:"warehouse_id,omitempty"`
	CatalogItemID        string           `json:"catalog_item_id"`
	AvailableQuantity    *float64         `json:"available_quantity,omitempty"`
	Active               bool             `json:"active"`
	ConflictPolicy       string           `json:"conflict_policy"`
	Source               string           `json:"source"`
	Reason               string           `json:"reason,omitempty"`
	ProjectionAction     string           `json:"projection_action"`
	Status               SuggestionStatus `json:"status"`
	ReviewComment        string           `json:"review_comment,omitempty"`
	ReviewedByEmployeeID string           `json:"reviewed_by_employee_id,omitempty"`
	ReviewedAt           *time.Time       `json:"reviewed_at,omitempty"`
	AssignedToEmployeeID string           `json:"assigned_to_employee_id,omitempty"`
	AssignedByEmployeeID string           `json:"assigned_by_employee_id,omitempty"`
	AssignedAt           *time.Time       `json:"assigned_at,omitempty"`
	AssignmentNote       string           `json:"assignment_note,omitempty"`
	AppliedStopListID    string           `json:"applied_stop_list_id,omitempty"`
	UpdatedAt            time.Time        `json:"updated_at"`
	OccurredAt           time.Time        `json:"occurred_at"`
	ProjectedAt          time.Time        `json:"projected_at"`
	CreatedAt            time.Time        `json:"created_at"`
}

// MenuItem описывает продаваемую позицию меню с lifecycle и основой routing/availability.
type MenuItem struct {
	ID                string          `json:"id"`
	RestaurantID      string          `json:"restaurant_id"`
	CatalogItemID     string          `json:"catalog_item_id"`
	CategoryID        string          `json:"category_id,omitempty"`
	TagID             string          `json:"tag_id,omitempty"`
	TaxProfileID      string          `json:"tax_profile_id,omitempty"`
	Name              string          `json:"name"`
	Price             int64           `json:"price"`
	Currency          string          `json:"currency"`
	Status            LifecycleStatus `json:"status"`
	RuntimeStatus     string          `json:"runtime_status"`
	AvailabilityJSON  string          `json:"availability_json"`
	StationRoutingKey string          `json:"station_routing_key,omitempty"`
	CloudVersion      int64           `json:"cloud_version"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	ArchivedAt        *time.Time      `json:"archived_at,omitempty"`
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

// SuggestionStatus задает жизненный цикл Cloud review для kitchen proposals.
type SuggestionStatus string

const (
	SuggestionStatusPending        SuggestionStatus = "pending"
	SuggestionStatusApproved       SuggestionStatus = "approved"
	SuggestionStatusRejected       SuggestionStatus = "rejected"
	SuggestionStatusChangesRequest SuggestionStatus = "changes_requested"
)

// CatalogSuggestion хранит Cloud review карточку предложения номенклатуры от Edge кухни.
type CatalogSuggestion struct {
	ID                   string           `json:"id"`
	SuggestionID         string           `json:"suggestion_id"`
	RestaurantID         string           `json:"restaurant_id"`
	CatalogItemID        string           `json:"catalog_item_id,omitempty"`
	ProposalGroupID      string           `json:"proposal_group_id,omitempty"`
	Action               string           `json:"action"`
	Reason               string           `json:"reason,omitempty"`
	Status               SuggestionStatus `json:"status"`
	ReviewComment        string           `json:"review_comment,omitempty"`
	ReviewedByEmployeeID string           `json:"reviewed_by_employee_id,omitempty"`
	ReviewedAt           *time.Time       `json:"reviewed_at,omitempty"`
	AssignedToEmployeeID string           `json:"assigned_to_employee_id,omitempty"`
	AssignedByEmployeeID string           `json:"assigned_by_employee_id,omitempty"`
	AssignedAt           *time.Time       `json:"assigned_at,omitempty"`
	AssignmentNote       string           `json:"assignment_note,omitempty"`
	AppliedCatalogItemID string           `json:"applied_catalog_item_id,omitempty"`
	SourceEventID        string           `json:"source_event_id,omitempty"`
	SuggestedAt          time.Time        `json:"suggested_at"`
	CloudReceivedAt      time.Time        `json:"cloud_received_at"`
	PayloadJSON          json.RawMessage  `json:"payload_json"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

// RecipeSuggestion хранит Cloud review карточку предложения техкарты от Edge кухни.
type RecipeSuggestion struct {
	ID                       string           `json:"id"`
	SuggestionID             string           `json:"suggestion_id"`
	RestaurantID             string           `json:"restaurant_id"`
	RecipeVersionID          string           `json:"recipe_version_id,omitempty"`
	OwnerCatalogItemID       string           `json:"owner_catalog_item_id,omitempty"`
	OwnerCatalogSuggestionID string           `json:"owner_catalog_suggestion_id,omitempty"`
	ProposalGroupID          string           `json:"proposal_group_id,omitempty"`
	Action                   string           `json:"action"`
	Reason                   string           `json:"reason,omitempty"`
	PrepTimeDeltaMinutes     int              `json:"prep_time_delta_minutes,omitempty"`
	Status                   SuggestionStatus `json:"status"`
	ReviewComment            string           `json:"review_comment,omitempty"`
	ReviewedByEmployeeID     string           `json:"reviewed_by_employee_id,omitempty"`
	ReviewedAt               *time.Time       `json:"reviewed_at,omitempty"`
	AssignedToEmployeeID     string           `json:"assigned_to_employee_id,omitempty"`
	AssignedByEmployeeID     string           `json:"assigned_by_employee_id,omitempty"`
	AssignedAt               *time.Time       `json:"assigned_at,omitempty"`
	AssignmentNote           string           `json:"assignment_note,omitempty"`
	SourceEventID            string           `json:"source_event_id,omitempty"`
	SuggestedAt              time.Time        `json:"suggested_at"`
	CloudReceivedAt          time.Time        `json:"cloud_received_at"`
	PayloadJSON              json.RawMessage  `json:"payload_json"`
	CreatedAt                time.Time        `json:"created_at"`
	UpdatedAt                time.Time        `json:"updated_at"`
}

// ReviewAssignmentAuditEvent фиксирует append-only audit для назначения Cloud review item.
type ReviewAssignmentAuditEvent struct {
	EventID          string    `json:"event_id"`
	CommandID        string    `json:"command_id"`
	ReviewType       string    `json:"review_type"`
	ReviewID         string    `json:"review_id"`
	Action           string    `json:"action"`
	ActorEmployeeID  string    `json:"actor_employee_id"`
	TargetEmployeeID string    `json:"target_employee_id,omitempty"`
	Reason           string    `json:"reason,omitempty"`
	OccurredAt       time.Time `json:"occurred_at"`
}

// RecipeSuggestionChange хранит строки diff для recipe suggestion.
type RecipeSuggestionChange struct {
	ID                 string          `json:"id"`
	RecipeSuggestionID string          `json:"recipe_suggestion_id"`
	LineID             string          `json:"line_id,omitempty"`
	Action             string          `json:"action"`
	FromCatalogItemID  string          `json:"from_catalog_item_id,omitempty"`
	ToCatalogItemID    string          `json:"to_catalog_item_id,omitempty"`
	Quantity           string          `json:"quantity,omitempty"`
	UnitCode           string          `json:"unit_code,omitempty"`
	LossPercent        string          `json:"loss_percent,omitempty"`
	SortOrder          int             `json:"sort_order"`
	PayloadJSON        json.RawMessage `json:"payload_json,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

// PublishedState связывает публикацию с пакетом для внутренних проверок и будущих query API.
type PublishedState struct {
	Publication Publication      `json:"publication"`
	Package     MasterDataPacket `json:"package"`
}

// MasterDataPacket описывает deterministic Cloud -> Edge package payload.
type MasterDataPacket struct {
	NodeDeviceID           string                      `json:"node_device_id,omitempty"`
	RestaurantID           string                      `json:"restaurant_id"`
	SyncMode               string                      `json:"sync_mode"`
	CheckpointToken        string                      `json:"checkpoint_token,omitempty"`
	CloudVersion           int64                       `json:"cloud_version"`
	CloudUpdatedAt         time.Time                   `json:"cloud_updated_at"`
	Restaurants            []EdgeRestaurant            `json:"restaurants,omitempty"`
	Roles                  []EdgeRole                  `json:"roles,omitempty"`
	Employees              []EdgeEmployee              `json:"employees,omitempty"`
	CatalogItems           []EdgeCatalogItem           `json:"catalog_items,omitempty"`
	Folders                []EdgeCatalogFolder         `json:"folders,omitempty"`
	FolderParameters       []EdgeFolderParameter       `json:"folder_parameters,omitempty"`
	Tags                   []EdgeCatalogTag            `json:"tags,omitempty"`
	ItemTags               []EdgeCatalogItemTag        `json:"item_tags,omitempty"`
	ModifierGroups         []EdgeModifierGroup         `json:"modifier_groups,omitempty"`
	ModifierOptions        []EdgeModifierOption        `json:"modifier_options,omitempty"`
	ModifierBindings       []EdgeModifierGroupBinding  `json:"modifier_bindings,omitempty"`
	MenuItemModifierGroups []EdgeMenuItemModifierGroup `json:"menu_item_modifier_groups,omitempty"`
	MenuItems              []EdgeMenuItem              `json:"menu_items,omitempty"`
	Halls                  []EdgeHall                  `json:"halls,omitempty"`
	Tables                 []EdgeTable                 `json:"tables,omitempty"`
	PricingPolicies        []EdgePricingPolicy         `json:"pricing_policies,omitempty"`
	RecipeVersions         []EdgeRecipeVersion         `json:"recipe_versions,omitempty"`
	RecipeLines            []EdgeRecipeLine            `json:"recipe_lines,omitempty"`
	StopLists              []EdgeStopListEntry         `json:"stop_lists,omitempty"`
	Warehouses             []EdgeWarehouseReference    `json:"warehouses,omitempty"`
}

// EdgeRestaurant является projection ресторана в существующий POS Edge restaurants stream.
type EdgeRestaurant struct {
	ID                           string    `json:"id"`
	Name                         string    `json:"name"`
	Timezone                     string    `json:"timezone"`
	Currency                     string    `json:"currency"`
	BusinessDayMode              string    `json:"business_day_mode"`
	BusinessDayBoundaryLocalTime string    `json:"business_day_boundary_local_time"`
	Active                       bool      `json:"active"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
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
	ID                 string    `json:"id"`
	Type               string    `json:"type"`
	FolderID           string    `json:"folder_id,omitempty"`
	Name               string    `json:"name"`
	SKU                string    `json:"sku"`
	BaseUnit           string    `json:"base_unit"`
	KitchenType        string    `json:"kitchen_type,omitempty"`
	AccountingCategory string    `json:"accounting_category,omitempty"`
	Active             bool      `json:"active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// EdgeCatalogFolder является projection папки номенклатуры в catalog stream.
type EdgeCatalogFolder struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	ParentID     string    `json:"parent_id,omitempty"`
	Name         string    `json:"name"`
	SortOrder    int64     `json:"sort_order"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EdgeFolderParameter является projection наследуемого параметра папки.
type EdgeFolderParameter struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	FolderID     string    `json:"folder_id"`
	Key          string    `json:"parameter_key"`
	ValueType    string    `json:"value_type"`
	ValueJSON    string    `json:"value_json"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EdgeCatalogTag является projection аналитической метки каталога.
type EdgeCatalogTag struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EdgeCatalogItemTag является projection связи catalog item и tag.
type EdgeCatalogItemTag struct {
	CatalogItemID string `json:"catalog_item_id"`
	TagID         string `json:"tag_id"`
	RestaurantID  string `json:"restaurant_id"`
}

// EdgeMenuItem является projection menu item в существующий POS Edge menu stream.
type EdgeMenuItem struct {
	ID            string    `json:"id"`
	CatalogItemID string    `json:"catalog_item_id"`
	CategoryID    string    `json:"category_id,omitempty"`
	TagID         string    `json:"tag_id,omitempty"`
	Name          string    `json:"name"`
	Price         int64     `json:"price"`
	Currency      string    `json:"currency"`
	TaxProfileID  string    `json:"tax_profile_id,omitempty"`
	RuntimeStatus string    `json:"runtime_status,omitempty"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EdgeHall является projection зала для существующего POS Edge floor stream.
type EdgeHall struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	Name         string    `json:"name"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EdgeTable является projection стола для существующего POS Edge floor stream.
type EdgeTable struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	HallID       string    `json:"hall_id"`
	Name         string    `json:"name"`
	Seats        int64     `json:"seats"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EdgeModifierGroup является Cloud -> POS Edge ingest projection группы модификаторов.
type EdgeModifierGroup struct {
	ID           string `json:"id"`
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Required     bool   `json:"required"`
	MinCount     int64  `json:"min_count"`
	MaxCount     int64  `json:"max_count"`
	Active       bool   `json:"active"`
}

// EdgeModifierOption является Cloud -> POS Edge ingest projection варианта модификатора.
type EdgeModifierOption struct {
	ID                  string `json:"id"`
	RestaurantID        string `json:"restaurant_id"`
	ModifierGroupID     string `json:"modifier_group_id"`
	LinkedCatalogItemID string `json:"linked_catalog_item_id,omitempty"`
	Name                string `json:"name"`
	PriceMinor          int64  `json:"price_minor"`
	Active              bool   `json:"active"`
}

// EdgeModifierGroupBinding является Cloud -> POS Edge ingest projection явной привязки группы модификаторов.
type EdgeModifierGroupBinding struct {
	ID              string `json:"id"`
	RestaurantID    string `json:"restaurant_id"`
	ModifierGroupID string `json:"modifier_group_id"`
	TargetType      string `json:"target_type"`
	TargetID        string `json:"target_id"`
	SortOrder       int64  `json:"sort_order"`
	Active          bool   `json:"active"`
}

// EdgeMenuItemModifierGroup является link-only projection модификаторов к menu item для POS Edge strict ingest.
type EdgeMenuItemModifierGroup struct {
	MenuItemID      string `json:"menu_item_id"`
	ModifierGroupID string `json:"modifier_group_id"`
	SortOrder       int64  `json:"sort_order"`
}

// EdgePricingPolicy является projection Cloud-authored discount/surcharge policy.
type EdgePricingPolicy struct {
	ID                 string `json:"id"`
	RestaurantID       string `json:"restaurant_id"`
	Name               string `json:"name"`
	Kind               string `json:"kind"`
	Scope              string `json:"scope"`
	AmountKind         string `json:"amount_kind"`
	AmountMinor        int64  `json:"amount_minor,omitempty"`
	ValueBasisPoints   int64  `json:"value_basis_points,omitempty"`
	ApplicationIndex   int    `json:"application_index"`
	Manual             bool   `json:"manual"`
	RequiresPermission string `json:"requires_permission,omitempty"`
	Active             bool   `json:"active"`
}

// EdgeRecipeVersion является projection Cloud recipe owner в POS Edge recipes stream.
type EdgeRecipeVersion struct {
	ID                string    `json:"id"`
	DishCatalogItemID string    `json:"dish_catalog_item_id"`
	Version           int       `json:"version"`
	Name              string    `json:"name"`
	Status            string    `json:"status"`
	YieldQuantity     int64     `json:"yield_quantity"`
	YieldUnit         string    `json:"yield_unit"`
	Active            bool      `json:"active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// EdgeRecipeLine является projection Cloud recipe component в POS Edge recipes stream.
type EdgeRecipeLine struct {
	ID              string    `json:"id"`
	RecipeVersionID string    `json:"recipe_version_id"`
	CatalogItemID   string    `json:"catalog_item_id"`
	Quantity        int64     `json:"quantity"`
	Unit            string    `json:"unit"`
	LossPercent     int       `json:"loss_percent"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// EdgeStopListEntry является projection Cloud stop-list в POS Edge inventory_reference stream.
type EdgeStopListEntry struct {
	ID                string    `json:"id"`
	RestaurantID      string    `json:"restaurant_id"`
	CatalogItemID     string    `json:"catalog_item_id"`
	AvailableQuantity *float64  `json:"available_quantity,omitempty"`
	Source            string    `json:"source"`
	Reason            string    `json:"reason,omitempty"`
	Active            bool      `json:"active"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// EdgeWarehouseReference является projection Cloud-owned склада для POS Edge inventory_reference stream.
type EdgeWarehouseReference struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"`
	Default      bool      `json:"is_default"`
	Active       bool      `json:"active"`
	UpdatedAt    time.Time `json:"updated_at"`
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
	case CatalogItemDish, CatalogItemGood, CatalogItemSemiFinished, CatalogItemService:
		return nil
	default:
		return fmt.Errorf("%w: unsupported catalog item kind %q", ErrInvalid, v)
	}
}

// ValidateModifierTargetType проверяет допустимую цель привязки модификатора.
func ValidateModifierTargetType(v ModifierTargetType) error {
	switch v {
	case ModifierTargetMenuItem, ModifierTargetCatalogItem, ModifierTargetFolder, ModifierTargetTag:
		return nil
	default:
		return fmt.Errorf("%w: unsupported modifier target_type %q", ErrInvalid, v)
	}
}

// ValidatePricingPolicyKind проверяет допустимый тип Cloud pricing policy.
func ValidatePricingPolicyKind(v PricingPolicyKind) error {
	switch v {
	case PricingPolicyDiscount, PricingPolicySurcharge:
		return nil
	default:
		return fmt.Errorf("%w: unsupported pricing policy kind %q", ErrInvalid, v)
	}
}

// ValidateRestaurantStatus проверяет допустимое lifecycle состояние ресторана.
func ValidateRestaurantStatus(v RestaurantStatus) error {
	switch v {
	case RestaurantActive, RestaurantArchived:
		return nil
	default:
		return fmt.Errorf("%w: unsupported restaurant status %q", ErrInvalid, v)
	}
}

// ValidatePermissionsJSON проверяет, что snapshot прав является валидным JSON.
func ValidatePermissionsJSON(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%w: permissions_json is required", ErrInvalid)
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fmt.Errorf("%w: permissions_json must be valid JSON", ErrInvalid)
	}
	for _, permission := range permissionsFromJSON(value) {
		if _, ok := knownPermissionIDs[permission]; !ok {
			return fmt.Errorf("%w: unknown permission id %q", ErrInvalid, permission)
		}
	}
	return nil
}

func permissionsFromJSON(raw map[string]any) []string {
	seen := map[string]struct{}{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" {
			seen[v] = struct{}{}
		}
	}
	for key, value := range raw {
		if allowed, ok := value.(bool); ok && allowed {
			add(key)
		}
	}
	if values, ok := raw["permissions"].([]any); ok {
		for _, value := range values {
			if text, ok := value.(string); ok {
				add(text)
			}
		}
	}
	out := make([]string, 0, len(seen))
	for permission := range seen {
		out = append(out, permission)
	}
	return out
}

var knownPermissionIDs = map[string]struct{}{
	"organization.manage":               {},
	"pos.employee_shift.open":           {},
	"pos.employee_shift.close":          {},
	"pos.employee_shift.view_current":   {},
	"pos.employee_shift.recent":         {},
	"pos.cash_session.open":             {},
	"pos.cash_session.close":            {},
	"pos.cash_session.view_current":     {},
	"pos.cash_drawer.record_event":      {},
	"pos.catalog.view":                  {},
	"pos.floor.view":                    {},
	"pos.menu.view":                     {},
	"pos.order.create":                  {},
	"pos.order.view":                    {},
	"pos.order.add_line":                {},
	"pos.order.change_quantity":         {},
	"pos.order.void_line":               {},
	"pos.order.close":                   {},
	"pos.pricing.view":                  {},
	"pos.pricing.discount.apply":        {},
	"pos.pricing.surcharge.apply":       {},
	"pos.precheck.issue":                {},
	"pos.precheck.view":                 {},
	"pos.precheck.reprint":              {},
	"pos.precheck.cancel.request":       {},
	"pos.precheck.cancel":               {},
	"pos.payment.cash":                  {},
	"pos.payment.card.manual":           {},
	"pos.payment.other":                 {},
	"pos.payment.refund":                {},
	"pos.check.view":                    {},
	"pos.check.reprint":                 {},
	"pos.kitchen.view":                  {},
	"pos.kitchen.status.change":         {},
	"pos.kitchen.catalog.view":          {},
	"pos.kitchen.recipe.view":           {},
	"pos.kitchen.recipe.suggest":        {},
	"pos.kitchen.catalog.suggest":       {},
	"pos.kitchen.stock.receipt":         {},
	"pos.kitchen.stock.inventory_count": {},
	"pos.kitchen.stock.write_off":       {},
	"pos.kitchen.production.complete":   {},
	"pos.kitchen.stop_list.view":        {},
	"pos.kitchen.stop_list.update":      {},
	"pos.sync.view":                     {},
	"pos.sync.retry_failed":             {},
}
