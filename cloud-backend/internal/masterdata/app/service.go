package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"crypto/pbkdf2"

	"cloud-backend/internal/cloudsync/contracts"
	"cloud-backend/internal/masterdata/domain"
	"cloud-backend/internal/platform/clock"
	"cloud-backend/internal/platform/idgen"
)

const (
	pinHashPrefix     = "pin.pbkdf2.sha256"
	pinHashVersion    = "v1"
	pinHashIterations = 120000
	pinHashKeyLength  = 32
)

// Repository задает persistence port для Cloud master-data use cases.
type Repository interface {
	CreateRestaurant(context.Context, domain.Restaurant) (domain.Restaurant, error)
	UpdateRestaurant(context.Context, domain.Restaurant) (domain.Restaurant, error)
	GetRestaurant(context.Context, string) (domain.Restaurant, error)
	ListRestaurants(context.Context) ([]domain.Restaurant, error)
	CreateRole(context.Context, domain.Role) (domain.Role, error)
	UpdateRole(context.Context, domain.Role) (domain.Role, error)
	GetRole(context.Context, string) (domain.Role, error)
	ListRoles(context.Context) ([]domain.Role, error)
	CreateEmployee(context.Context, domain.Employee) (domain.Employee, error)
	UpdateEmployee(context.Context, domain.Employee) (domain.Employee, error)
	GetEmployee(context.Context, string) (domain.Employee, error)
	ListEmployees(context.Context) ([]domain.Employee, error)
	CreateCatalogItem(context.Context, domain.CatalogItem) (domain.CatalogItem, error)
	UpdateCatalogItem(context.Context, domain.CatalogItem) (domain.CatalogItem, error)
	GetCatalogItem(context.Context, string) (domain.CatalogItem, error)
	ListCatalogItems(context.Context, string) ([]domain.CatalogItem, error)
	CreateCatalogFolder(context.Context, domain.CatalogFolder) (domain.CatalogFolder, error)
	UpdateCatalogFolder(context.Context, domain.CatalogFolder) (domain.CatalogFolder, error)
	GetCatalogFolder(context.Context, string) (domain.CatalogFolder, error)
	ListCatalogFolders(context.Context, string) ([]domain.CatalogFolder, error)
	CreateFolderParameter(context.Context, domain.FolderParameter) (domain.FolderParameter, error)
	UpdateFolderParameter(context.Context, domain.FolderParameter) (domain.FolderParameter, error)
	GetFolderParameter(context.Context, string) (domain.FolderParameter, error)
	ListFolderParameters(context.Context, string) ([]domain.FolderParameter, error)
	CreateCatalogTag(context.Context, domain.CatalogTag) (domain.CatalogTag, error)
	UpdateCatalogTag(context.Context, domain.CatalogTag) (domain.CatalogTag, error)
	GetCatalogTag(context.Context, string) (domain.CatalogTag, error)
	ListCatalogTags(context.Context, string) ([]domain.CatalogTag, error)
	AssignCatalogItemTag(context.Context, domain.CatalogItemTag) (domain.CatalogItemTag, error)
	ListCatalogItemTags(context.Context, string) ([]domain.CatalogItemTag, error)
	CreateModifierGroup(context.Context, domain.ModifierGroup) (domain.ModifierGroup, error)
	UpdateModifierGroup(context.Context, domain.ModifierGroup) (domain.ModifierGroup, error)
	GetModifierGroup(context.Context, string) (domain.ModifierGroup, error)
	ListModifierGroups(context.Context, string) ([]domain.ModifierGroup, error)
	CreateModifierOption(context.Context, domain.ModifierOption) (domain.ModifierOption, error)
	UpdateModifierOption(context.Context, domain.ModifierOption) (domain.ModifierOption, error)
	GetModifierOption(context.Context, string) (domain.ModifierOption, error)
	ListModifierOptions(context.Context, string) ([]domain.ModifierOption, error)
	CreateModifierGroupBinding(context.Context, domain.ModifierGroupBinding) (domain.ModifierGroupBinding, error)
	UpdateModifierGroupBinding(context.Context, domain.ModifierGroupBinding) (domain.ModifierGroupBinding, error)
	GetModifierGroupBinding(context.Context, string) (domain.ModifierGroupBinding, error)
	ListModifierGroupBindings(context.Context, string) ([]domain.ModifierGroupBinding, error)
	CreatePricingPolicy(context.Context, domain.PricingPolicy) (domain.PricingPolicy, error)
	UpdatePricingPolicy(context.Context, domain.PricingPolicy) (domain.PricingPolicy, error)
	GetPricingPolicy(context.Context, string) (domain.PricingPolicy, error)
	ListPricingPolicies(context.Context, string) ([]domain.PricingPolicy, error)
	CreateRecipeItem(context.Context, domain.RecipeItem) (domain.RecipeItem, error)
	UpdateRecipeItem(context.Context, domain.RecipeItem) (domain.RecipeItem, error)
	GetRecipeItem(context.Context, string) (domain.RecipeItem, error)
	ListRecipeItems(context.Context, string) ([]domain.RecipeItem, error)
	CreateRecipeVersion(context.Context, domain.RecipeVersion, []domain.RecipeLine) (domain.RecipeVersion, error)
	UpdateRecipeVersion(context.Context, domain.RecipeVersion) (domain.RecipeVersion, error)
	GetRecipeVersion(context.Context, string) (domain.RecipeVersion, error)
	ListRecipeVersions(context.Context, string, string, string, int, int) ([]domain.RecipeVersion, error)
	ListRecipeLines(context.Context, string) ([]domain.RecipeLine, error)
	SubmitRecipeSuggestion(context.Context, domain.RecipeSuggestion, []domain.RecipeSuggestionChange) (domain.RecipeSuggestion, error)
	ActivateRecipeVersion(context.Context, string, string, time.Time) (domain.RecipeVersion, error)
	UpsertStopListEntry(context.Context, domain.StopListEntry) (domain.StopListEntry, error)
	GetStopListEntry(context.Context, string) (domain.StopListEntry, error)
	ListStopListEntries(context.Context, string) ([]domain.StopListEntry, error)
	CreateCategory(context.Context, domain.Category) (domain.Category, error)
	ListCategories(context.Context, string) ([]domain.Category, error)
	CreateHall(context.Context, domain.Hall) (domain.Hall, error)
	UpdateHall(context.Context, domain.Hall) (domain.Hall, error)
	GetHall(context.Context, string) (domain.Hall, error)
	ListHalls(context.Context, string) ([]domain.Hall, error)
	CreateTable(context.Context, domain.Table) (domain.Table, error)
	UpdateTable(context.Context, domain.Table) (domain.Table, error)
	GetTable(context.Context, string) (domain.Table, error)
	ListTables(context.Context, string) ([]domain.Table, error)
	CreateMenuItem(context.Context, domain.MenuItem) (domain.MenuItem, error)
	UpdateMenuItem(context.Context, domain.MenuItem) (domain.MenuItem, error)
	GetMenuItem(context.Context, string) (domain.MenuItem, error)
	ListMenuItems(context.Context, string) ([]domain.MenuItem, error)
	ListAssignedNodeDeviceIDs(context.Context, string) ([]string, error)
	ListAssignedRestaurantIDs(context.Context) ([]string, error)
	NextPublicationVersion(context.Context, string) (int64, error)
	SavePublication(context.Context, domain.Publication, []StreamPackage) (domain.Publication, error)
	GetCurrentPublication(context.Context, string) (domain.Publication, error)
	GetPublication(context.Context, string, string) (domain.Publication, error)
	ListCatalogSuggestions(context.Context, string, string, int, int) ([]domain.CatalogSuggestion, error)
	GetCatalogSuggestion(context.Context, string) (domain.CatalogSuggestion, error)
	UpdateCatalogSuggestion(context.Context, domain.CatalogSuggestion) (domain.CatalogSuggestion, error)
	ListRecipeSuggestions(context.Context, string, string, int, int) ([]domain.RecipeSuggestion, error)
	GetRecipeSuggestion(context.Context, string) (domain.RecipeSuggestion, error)
	UpdateRecipeSuggestion(context.Context, domain.RecipeSuggestion) (domain.RecipeSuggestion, error)
	ListRecipeSuggestionChanges(context.Context, string) ([]domain.RecipeSuggestionChange, error)
	ListStopListUpdateReviews(context.Context, string, string, int, int) ([]domain.StopListUpdateReview, error)
	GetStopListUpdateReview(context.Context, string) (domain.StopListUpdateReview, error)
	UpdateStopListUpdateReview(context.Context, domain.StopListUpdateReview) (domain.StopListUpdateReview, error)
	GetReviewAssignmentAuditEvent(context.Context, string) (domain.ReviewAssignmentAuditEvent, error)
	AppendReviewAssignmentAuditEvent(context.Context, domain.ReviewAssignmentAuditEvent) error
	ListReviewAssignmentAuditEvents(context.Context, string, string, int, int) ([]domain.ReviewAssignmentAuditEvent, error)
}

// IDGenerator задает источник идентификаторов для use cases и тестов.
type IDGenerator interface {
	NewID() string
}

// Service реализует use cases Cloud-authored master data и publication workflow.
type Service struct {
	repo  Repository
	clock clock.Clock
	ids   IDGenerator
}

// NewService создает application service Cloud master-data authority.
func NewService(repo Repository, clock clock.Clock, ids IDGenerator) *Service {
	if ids == nil {
		ids = idgen.UUIDGenerator{}
	}
	return &Service{repo: repo, clock: clock, ids: ids}
}

func afterRestaurantCommit[T any](s *Service, ctx context.Context, restaurantID string, v T, err error) (T, error) {
	if err != nil {
		return v, err
	}
	if strings.TrimSpace(restaurantID) == "" {
		return v, nil
	}
	if _, refreshErr := s.RefreshDeliveryPackages(ctx, restaurantID); refreshErr != nil {
		return v, refreshErr
	}
	return v, nil
}

func afterTenantCommit[T any](s *Service, ctx context.Context, v T, err error) (T, error) {
	if err != nil {
		return v, err
	}
	restaurantIDs, err := s.repo.ListAssignedRestaurantIDs(ctx)
	if err != nil {
		return v, err
	}
	for _, restaurantID := range restaurantIDs {
		if _, refreshErr := s.RefreshDeliveryPackages(ctx, restaurantID); refreshErr != nil {
			return v, refreshErr
		}
	}
	return v, nil
}

// CreateRoleCommand описывает создание роли с JSON snapshot прав.
type CreateRoleCommand struct {
	Name            string `json:"name"`
	PermissionsJSON string `json:"permissions_json"`
}

// CreateRestaurantCommand описывает production onboarding ресторана в Cloud.
type CreateRestaurantCommand struct {
	Name                         string `json:"name"`
	Timezone                     string `json:"timezone"`
	Currency                     string `json:"currency"`
	BusinessDayMode              string `json:"business_day_mode"`
	BusinessDayBoundaryLocalTime string `json:"business_day_boundary_local_time"`
}

// UpdateRestaurantCommand описывает изменение Cloud-owned настроек ресторана.
type UpdateRestaurantCommand struct {
	Name                         string                   `json:"name,omitempty"`
	Timezone                     string                   `json:"timezone,omitempty"`
	Currency                     string                   `json:"currency,omitempty"`
	BusinessDayMode              string                   `json:"business_day_mode,omitempty"`
	BusinessDayBoundaryLocalTime string                   `json:"business_day_boundary_local_time,omitempty"`
	Status                       *domain.RestaurantStatus `json:"status,omitempty"`
}

// UpdateRoleCommand описывает изменение роли и permission snapshot.
type UpdateRoleCommand struct {
	Name            string `json:"name,omitempty"`
	PermissionsJSON string `json:"permissions_json,omitempty"`
	Active          *bool  `json:"active,omitempty"`
}

// CreateEmployeeCommand описывает создание сотрудника с plaintext PIN только на входе use case.
type CreateEmployeeCommand struct {
	RoleID        string   `json:"role_id"`
	RestaurantIDs []string `json:"restaurant_ids"`
	Name          string   `json:"name"`
	PIN           string   `json:"pin"`
}

// UpdateEmployeeCommand описывает безопасное изменение карточки сотрудника.
type UpdateEmployeeCommand struct {
	RoleID        string                 `json:"role_id,omitempty"`
	RestaurantIDs *[]string              `json:"restaurant_ids,omitempty"`
	Name          string                 `json:"name,omitempty"`
	Status        *domain.EmployeeStatus `json:"status,omitempty"`
}

// AssignRoleCommand описывает смену роли сотрудника с пересчетом permission snapshot.
type AssignRoleCommand struct {
	RoleID string `json:"role_id"`
}

// RotatePINCommand описывает ротацию PIN credential без возврата PIN material.
type RotatePINCommand struct {
	PIN string `json:"pin"`
}

// CreateCatalogItemCommand описывает создание Cloud-owned catalog item.
type CreateCatalogItemCommand struct {
	RestaurantID       string                 `json:"restaurant_id,omitempty"`
	Kind               domain.CatalogItemKind `json:"kind"`
	Type               domain.CatalogItemKind `json:"type,omitempty"`
	FolderID           string                 `json:"folder_id,omitempty"`
	Name               string                 `json:"name"`
	SKU                string                 `json:"sku"`
	BaseUnit           string                 `json:"base_unit"`
	KitchenType        string                 `json:"kitchen_type,omitempty"`
	AccountingCategory string                 `json:"accounting_category,omitempty"`
}

// UpdateCatalogItemCommand описывает изменение Cloud-owned catalog item.
type UpdateCatalogItemCommand struct {
	Kind               *domain.CatalogItemKind `json:"kind,omitempty"`
	Type               *domain.CatalogItemKind `json:"type,omitempty"`
	FolderID           *string                 `json:"folder_id,omitempty"`
	Name               string                  `json:"name,omitempty"`
	SKU                string                  `json:"sku,omitempty"`
	BaseUnit           string                  `json:"base_unit,omitempty"`
	KitchenType        *string                 `json:"kitchen_type,omitempty"`
	AccountingCategory *string                 `json:"accounting_category,omitempty"`
	Status             *domain.LifecycleStatus `json:"status,omitempty"`
}

type CreateCatalogFolderCommand struct {
	RestaurantID string `json:"restaurant_id,omitempty"`
	ParentID     string `json:"parent_id,omitempty"`
	Name         string `json:"name"`
	SortOrder    int64  `json:"sort_order"`
}

type UpdateCatalogFolderCommand struct {
	ParentID  *string                 `json:"parent_id,omitempty"`
	Name      string                  `json:"name,omitempty"`
	SortOrder *int64                  `json:"sort_order,omitempty"`
	Status    *domain.LifecycleStatus `json:"status,omitempty"`
}

type CreateFolderParameterCommand struct {
	RestaurantID string `json:"restaurant_id"`
	FolderID     string `json:"folder_id"`
	Key          string `json:"parameter_key"`
	ValueType    string `json:"value_type"`
	ValueJSON    string `json:"value_json"`
}

type UpdateFolderParameterCommand struct {
	ValueType string                  `json:"value_type,omitempty"`
	ValueJSON string                  `json:"value_json,omitempty"`
	Status    *domain.LifecycleStatus `json:"status,omitempty"`
}

type CreateCatalogTagCommand struct {
	RestaurantID string `json:"restaurant_id,omitempty"`
	Name         string `json:"name"`
	Code         string `json:"code"`
}

type UpdateCatalogTagCommand struct {
	Name   string                  `json:"name,omitempty"`
	Code   string                  `json:"code,omitempty"`
	Status *domain.LifecycleStatus `json:"status,omitempty"`
}

type AssignCatalogItemTagCommand struct {
	RestaurantID  string `json:"restaurant_id,omitempty"`
	CatalogItemID string `json:"catalog_item_id"`
	TagID         string `json:"tag_id"`
}

type CreateModifierGroupCommand struct {
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Required     bool   `json:"required"`
	MinCount     int64  `json:"min_count"`
	MaxCount     int64  `json:"max_count"`
}

type UpdateModifierGroupCommand struct {
	Name     string                  `json:"name,omitempty"`
	Required *bool                   `json:"required,omitempty"`
	MinCount *int64                  `json:"min_count,omitempty"`
	MaxCount *int64                  `json:"max_count,omitempty"`
	Status   *domain.LifecycleStatus `json:"status,omitempty"`
}

type CreateModifierOptionCommand struct {
	RestaurantID        string `json:"restaurant_id"`
	ModifierGroupID     string `json:"modifier_group_id"`
	LinkedCatalogItemID string `json:"linked_catalog_item_id,omitempty"`
	Name                string `json:"name"`
	PriceMinor          int64  `json:"price_minor"`
	LegacyPriceDelta    *int64 `json:"price_delta,omitempty"`
}

type UpdateModifierOptionCommand struct {
	Name                string                  `json:"name,omitempty"`
	LinkedCatalogItemID *string                 `json:"linked_catalog_item_id,omitempty"`
	PriceMinor          *int64                  `json:"price_minor,omitempty"`
	Status              *domain.LifecycleStatus `json:"status,omitempty"`
}

type CreateModifierGroupBindingCommand struct {
	RestaurantID    string                    `json:"restaurant_id"`
	ModifierGroupID string                    `json:"modifier_group_id"`
	TargetType      domain.ModifierTargetType `json:"target_type"`
	TargetID        string                    `json:"target_id"`
	SortOrder       int64                     `json:"sort_order"`
}

type UpdateModifierGroupBindingCommand struct {
	SortOrder *int64                  `json:"sort_order,omitempty"`
	Status    *domain.LifecycleStatus `json:"status,omitempty"`
}

type CreatePricingPolicyCommand struct {
	RestaurantID       string                   `json:"restaurant_id"`
	Name               string                   `json:"name"`
	Kind               domain.PricingPolicyKind `json:"kind"`
	Scope              string                   `json:"scope"`
	AmountKind         string                   `json:"amount_kind"`
	AmountMinor        int64                    `json:"amount_minor,omitempty"`
	ValueBasisPoints   int64                    `json:"value_basis_points,omitempty"`
	ApplicationIndex   int                      `json:"application_index"`
	Manual             bool                     `json:"manual"`
	RequiresPermission string                   `json:"requires_permission,omitempty"`
}

type UpdatePricingPolicyCommand struct {
	Name               string                  `json:"name,omitempty"`
	Scope              string                  `json:"scope,omitempty"`
	AmountKind         string                  `json:"amount_kind,omitempty"`
	AmountMinor        *int64                  `json:"amount_minor,omitempty"`
	ValueBasisPoints   *int64                  `json:"value_basis_points,omitempty"`
	ApplicationIndex   *int                    `json:"application_index,omitempty"`
	Manual             *bool                   `json:"manual,omitempty"`
	RequiresPermission *string                 `json:"requires_permission,omitempty"`
	Status             *domain.LifecycleStatus `json:"status,omitempty"`
}

// RecipeVersionLineCommand описывает строку draft версии техкарты в Cloud authoring API.
type RecipeVersionLineCommand struct {
	ComponentCatalogItemID string `json:"component_catalog_item_id"`
	Quantity               int64  `json:"quantity"`
	Unit                   string `json:"unit"`
	LossPercent            int64  `json:"loss_percent"`
}

// CreateRecipeVersionDraftCommand создает Cloud draft версии техкарты без мутаций POS Edge.
type CreateRecipeVersionDraftCommand struct {
	RestaurantID        string                     `json:"restaurant_id"`
	OwnerCatalogItemID  string                     `json:"owner_catalog_item_id"`
	Name                string                     `json:"name"`
	YieldQuantity       int64                      `json:"yield_quantity"`
	YieldUnit           string                     `json:"yield_unit"`
	Lines               []RecipeVersionLineCommand `json:"lines"`
	CreatedByEmployeeID string                     `json:"created_by_employee_id,omitempty"`
	SubmitForReview     bool                       `json:"submit_for_review,omitempty"`
	Reason              string                     `json:"reason,omitempty"`
}

// SubmitRecipeVersionCommand отправляет draft техкарты в существующий review/apply flow.
type SubmitRecipeVersionCommand struct {
	SubmittedByEmployeeID string `json:"submitted_by_employee_id"`
	Reason                string `json:"reason,omitempty"`
}

// RecipeVersionView возвращает версию техкарты вместе со строками.
type RecipeVersionView struct {
	Version domain.RecipeVersion `json:"version"`
	Lines   []domain.RecipeLine  `json:"lines"`
}

// CreateRecipeItemCommand описывает минимальную Cloud-owned строку рецепта для публикации read-only Edge recipe.
type CreateRecipeItemCommand struct {
	RestaurantID             string `json:"restaurant_id"`
	RecipeOwnerCatalogItemID string `json:"recipe_owner_catalog_item_id"`
	ComponentCatalogItemID   string `json:"component_catalog_item_id"`
	Quantity                 int64  `json:"quantity"`
	Unit                     string `json:"unit"`
	LossPercent              int64  `json:"loss_percent"`
}

// UpdateRecipeItemCommand описывает изменение количества/единицы компонента рецепта без Edge-side authoring.
type UpdateRecipeItemCommand struct {
	Quantity    *int64 `json:"quantity,omitempty"`
	Unit        string `json:"unit,omitempty"`
	LossPercent *int64 `json:"loss_percent,omitempty"`
}

// UpsertStopListEntryCommand описывает Cloud-owned stop-list состояние для публикации sale blocking.
type UpsertStopListEntryCommand struct {
	RestaurantID      string   `json:"restaurant_id"`
	CatalogItemID     string   `json:"catalog_item_id"`
	AvailableQuantity *float64 `json:"available_quantity,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	Active            *bool    `json:"active,omitempty"`
}

// CreateCategoryCommand описывает создание категории меню.
type CreateCategoryCommand struct {
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	SortOrder    int64  `json:"sort_order"`
}

// CreateHallCommand описывает создание Cloud-owned зала.
type CreateHallCommand struct {
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
}

// UpdateHallCommand описывает изменение Cloud-owned зала.
type UpdateHallCommand struct {
	Name   string                  `json:"name,omitempty"`
	Status *domain.LifecycleStatus `json:"status,omitempty"`
}

// CreateTableCommand описывает создание Cloud-owned стола.
type CreateTableCommand struct {
	RestaurantID string `json:"restaurant_id"`
	HallID       string `json:"hall_id"`
	Name         string `json:"name"`
	Seats        int64  `json:"seats"`
}

// UpdateTableCommand описывает изменение Cloud-owned стола.
type UpdateTableCommand struct {
	HallID string                  `json:"hall_id,omitempty"`
	Name   string                  `json:"name,omitempty"`
	Seats  *int64                  `json:"seats,omitempty"`
	Status *domain.LifecycleStatus `json:"status,omitempty"`
}

// CreateMenuItemCommand описывает создание draft menu item.
type CreateMenuItemCommand struct {
	RestaurantID      string `json:"restaurant_id"`
	CatalogItemID     string `json:"catalog_item_id"`
	CategoryID        string `json:"category_id"`
	TagID             string `json:"tag_id,omitempty"`
	TaxProfileID      string `json:"tax_profile_id,omitempty"`
	Name              string `json:"name"`
	Price             int64  `json:"price"`
	Currency          string `json:"currency"`
	RuntimeStatus     string `json:"runtime_status,omitempty"`
	AvailabilityJSON  string `json:"availability_json"`
	StationRoutingKey string `json:"station_routing_key"`
}

// UpdateMenuItemCommand описывает изменение menu item и его publication lifecycle.
type UpdateMenuItemCommand struct {
	CatalogItemID     string                  `json:"catalog_item_id,omitempty"`
	CategoryID        string                  `json:"category_id,omitempty"`
	TagID             *string                 `json:"tag_id,omitempty"`
	TaxProfileID      *string                 `json:"tax_profile_id,omitempty"`
	Name              string                  `json:"name,omitempty"`
	Price             *int64                  `json:"price,omitempty"`
	Currency          string                  `json:"currency,omitempty"`
	Status            *domain.LifecycleStatus `json:"status,omitempty"`
	RuntimeStatus     *string                 `json:"runtime_status,omitempty"`
	AvailabilityJSON  string                  `json:"availability_json,omitempty"`
	StationRoutingKey string                  `json:"station_routing_key,omitempty"`
}

// PublishCommand описывает явную публикацию справочников для Cloud -> Edge delivery.
type PublishCommand struct {
	RestaurantID string `json:"restaurant_id"`
	PublishedBy  string `json:"published_by"`
	NodeDeviceID string `json:"node_device_id,omitempty"`
}

// PublicationSummary возвращает Cloud UI-safe metadata публикации без package payload и PIN material.
type PublicationSummary struct {
	ID            string         `json:"id"`
	RestaurantID  string         `json:"restaurant_id"`
	Version       int64          `json:"version"`
	Status        string         `json:"status"`
	CloudVersion  int64          `json:"cloud_version"`
	PublishedAt   time.Time      `json:"published_at"`
	PublishedBy   string         `json:"published_by"`
	PackageSHA256 string         `json:"package_sha256"`
	Counts        map[string]int `json:"counts"`
}

type SuggestionReviewCommand struct {
	ReviewedByEmployeeID string `json:"reviewed_by_employee_id"`
	ReviewComment        string `json:"review_comment,omitempty"`
	PublishedBy          string `json:"published_by,omitempty"`
}

type ReviewAssignCommand struct {
	CommandID            string `json:"command_id"`
	AssignedToEmployeeID string `json:"assigned_to_employee_id"`
	AssignedByEmployeeID string `json:"assigned_by_employee_id"`
	Reason               string `json:"reason,omitempty"`
}

type ReviewUnassignCommand struct {
	CommandID              string `json:"command_id"`
	UnassignedByEmployeeID string `json:"unassigned_by_employee_id"`
	Reason                 string `json:"reason,omitempty"`
}

type ReviewAssignmentResponse struct {
	ReviewType           string     `json:"review_type"`
	ID                   string     `json:"id"`
	Status               string     `json:"status"`
	AssignedToEmployeeID string     `json:"assigned_to_employee_id,omitempty"`
	AssignedByEmployeeID string     `json:"assigned_by_employee_id,omitempty"`
	AssignedAt           *time.Time `json:"assigned_at,omitempty"`
	AssignmentNote       string     `json:"assignment_note,omitempty"`
}

// ReviewAssignmentAuditEventResponse описывает безопасную выдачу audit trail без raw payload.
type ReviewAssignmentAuditEventResponse struct {
	EventID          string    `json:"event_id"`
	ReviewID         string    `json:"review_id"`
	ReviewType       string    `json:"review_type"`
	Action           string    `json:"action"`
	ActorEmployeeID  string    `json:"actor_employee_id"`
	TargetEmployeeID string    `json:"target_employee_id,omitempty"`
	OccurredAt       time.Time `json:"occurred_at"`
	Reason           string    `json:"reason,omitempty"`
	CommandID        string    `json:"command_id,omitempty"`
}

// StreamPackage описывает stream-specific package, сохраняемый для Edge import.
type StreamPackage struct {
	StreamName      string          `json:"stream_name"`
	NodeDeviceID    string          `json:"node_device_id,omitempty"`
	RestaurantID    string          `json:"restaurant_id"`
	SyncMode        string          `json:"sync_mode"`
	CloudVersion    int64           `json:"cloud_version"`
	CheckpointToken string          `json:"checkpoint_token"`
	CloudUpdatedAt  time.Time       `json:"cloud_updated_at"`
	PayloadJSON     json.RawMessage `json:"payload_json"`
}

// CreateRestaurant создает Cloud-owned ресторан с production-настройками учетного дня.
func (s *Service) CreateRestaurant(ctx context.Context, cmd CreateRestaurantCommand) (domain.Restaurant, error) {
	name := strings.TrimSpace(cmd.Name)
	timezone := strings.TrimSpace(cmd.Timezone)
	currency := strings.ToUpper(strings.TrimSpace(cmd.Currency))
	mode, boundary, err := normalizeBusinessDayConfig(cmd.BusinessDayMode, cmd.BusinessDayBoundaryLocalTime)
	if err != nil {
		return domain.Restaurant{}, err
	}
	if name == "" || timezone == "" || currency == "" {
		return domain.Restaurant{}, fmt.Errorf("%w: name, timezone and currency are required", domain.ErrInvalid)
	}
	if !isActiveCurrencyCode(currency) {
		return domain.Restaurant{}, fmt.Errorf("%w: currency must be active ISO 4217 code", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	restaurant := domain.Restaurant{
		ID:                           s.ids.NewID(),
		Name:                         name,
		Timezone:                     timezone,
		Currency:                     currency,
		BusinessDayMode:              mode,
		BusinessDayBoundaryLocalTime: boundary,
		Status:                       domain.RestaurantActive,
		CloudVersion:                 1,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}
	stored, err := s.repo.CreateRestaurant(ctx, restaurant)
	return afterRestaurantCommit(s, ctx, restaurant.ID, stored, err)
}

// ListRestaurants возвращает рестораны для будущего Cloud UI/backoffice.
func (s *Service) ListRestaurants(ctx context.Context) ([]domain.Restaurant, error) {
	return s.repo.ListRestaurants(ctx)
}

// GetRestaurant возвращает один Cloud-owned ресторан.
func (s *Service) GetRestaurant(ctx context.Context, id string) (domain.Restaurant, error) {
	return s.repo.GetRestaurant(ctx, strings.TrimSpace(id))
}

// UpdateRestaurant изменяет настройки ресторана и увеличивает cloud_version.
func (s *Service) UpdateRestaurant(ctx context.Context, id string, cmd UpdateRestaurantCommand) (domain.Restaurant, error) {
	restaurant, err := s.repo.GetRestaurant(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Restaurant{}, err
	}
	if strings.TrimSpace(cmd.Name) != "" {
		restaurant.Name = strings.TrimSpace(cmd.Name)
	}
	if strings.TrimSpace(cmd.Timezone) != "" {
		restaurant.Timezone = strings.TrimSpace(cmd.Timezone)
	}
	if strings.TrimSpace(cmd.Currency) != "" {
		currency := strings.ToUpper(strings.TrimSpace(cmd.Currency))
		if !isActiveCurrencyCode(currency) {
			return domain.Restaurant{}, fmt.Errorf("%w: currency must be active ISO 4217 code", domain.ErrInvalid)
		}
		restaurant.Currency = currency
	}
	if strings.TrimSpace(cmd.BusinessDayMode) != "" || strings.TrimSpace(cmd.BusinessDayBoundaryLocalTime) != "" {
		mode, boundary, err := normalizeBusinessDayConfig(firstNonEmpty(cmd.BusinessDayMode, restaurant.BusinessDayMode), firstNonEmpty(cmd.BusinessDayBoundaryLocalTime, restaurant.BusinessDayBoundaryLocalTime))
		if err != nil {
			return domain.Restaurant{}, err
		}
		restaurant.BusinessDayMode = mode
		restaurant.BusinessDayBoundaryLocalTime = boundary
	}
	if cmd.Status != nil {
		if err := domain.ValidateRestaurantStatus(*cmd.Status); err != nil {
			return domain.Restaurant{}, err
		}
		restaurant.Status = *cmd.Status
	}
	restaurant.CloudVersion++
	restaurant.UpdatedAt = s.clock.Now().UTC()
	if restaurant.Status == domain.RestaurantArchived && restaurant.ArchivedAt == nil {
		archivedAt := restaurant.UpdatedAt
		restaurant.ArchivedAt = &archivedAt
	}
	if restaurant.Status != domain.RestaurantArchived {
		restaurant.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateRestaurant(ctx, restaurant)
	return afterRestaurantCommit(s, ctx, restaurant.ID, stored, err)
}

// ArchiveRestaurant выполняет soft-delete ресторана.
func (s *Service) ArchiveRestaurant(ctx context.Context, id string) (domain.Restaurant, error) {
	status := domain.RestaurantArchived
	return s.UpdateRestaurant(ctx, id, UpdateRestaurantCommand{Status: &status})
}

// CreateRole создает Cloud-authored роль.
func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCommand) (domain.Role, error) {
	name := strings.TrimSpace(cmd.Name)
	permissions := strings.TrimSpace(cmd.PermissionsJSON)
	if permissions == "" {
		permissions = "{}"
	}
	if name == "" {
		return domain.Role{}, fmt.Errorf("%w: name is required", domain.ErrInvalid)
	}
	if err := domain.ValidatePermissionsJSON(permissions); err != nil {
		return domain.Role{}, err
	}
	now := s.clock.Now().UTC()
	role := domain.Role{
		ID:              s.ids.NewID(),
		Name:            name,
		PermissionsJSON: canonicalJSON(permissions),
		Active:          true,
		CloudVersion:    1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	stored, err := s.repo.CreateRole(ctx, role)
	return afterTenantCommit(s, ctx, stored, err)
}

// ListRoles возвращает tenant-level роли.
func (s *Service) ListRoles(ctx context.Context) ([]domain.Role, error) {
	return s.repo.ListRoles(ctx)
}

// GetRole возвращает одну роль.
func (s *Service) GetRole(ctx context.Context, id string) (domain.Role, error) {
	return s.repo.GetRole(ctx, strings.TrimSpace(id))
}

// UpdateRole изменяет роль и увеличивает cloud_version.
func (s *Service) UpdateRole(ctx context.Context, id string, cmd UpdateRoleCommand) (domain.Role, error) {
	role, err := s.repo.GetRole(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Role{}, err
	}
	if strings.TrimSpace(cmd.Name) != "" {
		role.Name = strings.TrimSpace(cmd.Name)
	}
	if strings.TrimSpace(cmd.PermissionsJSON) != "" {
		if err := domain.ValidatePermissionsJSON(cmd.PermissionsJSON); err != nil {
			return domain.Role{}, err
		}
		role.PermissionsJSON = canonicalJSON(cmd.PermissionsJSON)
	}
	if cmd.Active != nil {
		role.Active = *cmd.Active
	}
	if !role.ManagesOrganization() {
		employees, err := s.repo.ListEmployees(ctx)
		if err != nil {
			return domain.Role{}, err
		}
		for _, employee := range employees {
			if employee.RoleID == role.ID && employee.Status != domain.EmployeeArchived && len(employee.RestaurantIDs) == 0 {
				return domain.Role{}, fmt.Errorf("%w: removing organization.manage requires employee memberships", domain.ErrInvalid)
			}
		}
	}
	role.CloudVersion++
	role.UpdatedAt = s.clock.Now().UTC()
	if !role.Active && role.ArchivedAt == nil {
		archivedAt := role.UpdatedAt
		role.ArchivedAt = &archivedAt
	}
	stored, err := s.repo.UpdateRole(ctx, role)
	return afterTenantCommit(s, ctx, stored, err)
}

// ArchiveRole архивирует роль без физического удаления.
func (s *Service) ArchiveRole(ctx context.Context, id string) (domain.Role, error) {
	active := false
	return s.UpdateRole(ctx, id, UpdateRoleCommand{Active: &active})
}

// CreateEmployee создает Cloud-authored сотрудника и хэширует PIN credential.
func (s *Service) CreateEmployee(ctx context.Context, cmd CreateEmployeeCommand) (domain.Employee, error) {
	roleID := strings.TrimSpace(cmd.RoleID)
	name := strings.TrimSpace(cmd.Name)
	if roleID == "" || name == "" || strings.TrimSpace(cmd.PIN) == "" {
		return domain.Employee{}, fmt.Errorf("%w: role_id, name and pin are required", domain.ErrInvalid)
	}
	role, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return domain.Employee{}, err
	}
	if !role.Active {
		return domain.Employee{}, fmt.Errorf("%w: role is archived", domain.ErrInvalid)
	}
	restaurantIDs, allRestaurants, err := s.normalizeEmployeeScope(ctx, role, cmd.RestaurantIDs)
	if err != nil {
		return domain.Employee{}, err
	}
	if err := s.ensurePINUnique(ctx, "", cmd.PIN); err != nil {
		return domain.Employee{}, err
	}
	pinHash, err := hashPIN(cmd.PIN)
	if err != nil {
		return domain.Employee{}, err
	}
	now := s.clock.Now().UTC()
	employee := domain.Employee{
		ID:                     s.ids.NewID(),
		RoleID:                 roleID,
		RestaurantIDs:          restaurantIDs,
		AllRestaurants:         allRestaurants,
		Name:                   name,
		Status:                 domain.EmployeeActive,
		PINHash:                pinHash,
		PINConfigured:          true,
		PINCredentialVersion:   1,
		PermissionSnapshotJSON: role.PermissionsJSON,
		CloudVersion:           1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	stored, err := s.repo.CreateEmployee(ctx, employee)
	return afterTenantCommit(s, ctx, stored, err)
}

// ListEmployees возвращает tenant-level сотрудников и их memberships.
func (s *Service) ListEmployees(ctx context.Context) ([]domain.Employee, error) {
	items, err := s.repo.ListEmployees(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].PINConfigured = strings.TrimSpace(items[i].PINHash) != ""
		role, roleErr := s.repo.GetRole(ctx, items[i].RoleID)
		if roleErr != nil {
			return nil, roleErr
		}
		items[i].AllRestaurants = role.ManagesOrganization()
	}
	return items, nil
}

// GetEmployee возвращает одного сотрудника без раскрытия PIN material в JSON.
func (s *Service) GetEmployee(ctx context.Context, id string) (domain.Employee, error) {
	employee, err := s.repo.GetEmployee(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Employee{}, err
	}
	employee.PINConfigured = strings.TrimSpace(employee.PINHash) != ""
	role, err := s.repo.GetRole(ctx, employee.RoleID)
	if err != nil {
		return domain.Employee{}, err
	}
	employee.AllRestaurants = role.ManagesOrganization()
	return employee, nil
}

// UpdateEmployee обновляет карточку сотрудника и permission snapshot при смене роли.
func (s *Service) UpdateEmployee(ctx context.Context, id string, cmd UpdateEmployeeCommand) (domain.Employee, error) {
	employee, err := s.repo.GetEmployee(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Employee{}, err
	}
	if strings.TrimSpace(cmd.Name) != "" {
		employee.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.Status != nil {
		if err := domain.ValidateEmployeeStatus(*cmd.Status); err != nil {
			return domain.Employee{}, err
		}
		employee.Status = *cmd.Status
	}
	if strings.TrimSpace(cmd.RoleID) != "" {
		role, err := s.repo.GetRole(ctx, strings.TrimSpace(cmd.RoleID))
		if err != nil {
			return domain.Employee{}, err
		}
		if !role.Active {
			return domain.Employee{}, fmt.Errorf("%w: role is archived", domain.ErrInvalid)
		}
		employee.RoleID = role.ID
		employee.PermissionSnapshotJSON = role.PermissionsJSON
	}
	role, err := s.repo.GetRole(ctx, employee.RoleID)
	if err != nil {
		return domain.Employee{}, err
	}
	requested := employee.RestaurantIDs
	if cmd.RestaurantIDs != nil {
		requested = *cmd.RestaurantIDs
	}
	employee.RestaurantIDs, employee.AllRestaurants, err = s.normalizeEmployeeScope(ctx, role, requested)
	if err != nil {
		return domain.Employee{}, err
	}
	employee.CloudVersion++
	employee.UpdatedAt = s.clock.Now().UTC()
	if employee.Status == domain.EmployeeSuspended && employee.SuspendedAt == nil {
		suspendedAt := employee.UpdatedAt
		employee.SuspendedAt = &suspendedAt
	}
	if employee.Status != domain.EmployeeSuspended {
		employee.SuspendedAt = nil
	}
	if employee.Status == domain.EmployeeArchived && employee.ArchivedAt == nil {
		archivedAt := employee.UpdatedAt
		employee.ArchivedAt = &archivedAt
	}
	if employee.Status != domain.EmployeeArchived {
		employee.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateEmployee(ctx, employee)
	return afterTenantCommit(s, ctx, stored, err)
}

// SuspendEmployee переводит сотрудника в состояние, недоступное для POS login.
func (s *Service) SuspendEmployee(ctx context.Context, id string) (domain.Employee, error) {
	status := domain.EmployeeSuspended
	return s.UpdateEmployee(ctx, id, UpdateEmployeeCommand{Status: &status})
}

// ArchiveEmployee архивирует сотрудника без удаления истории.
func (s *Service) ArchiveEmployee(ctx context.Context, id string) (domain.Employee, error) {
	status := domain.EmployeeArchived
	return s.UpdateEmployee(ctx, id, UpdateEmployeeCommand{Status: &status})
}

// ActivateEmployee возвращает сотрудника в active lifecycle.
func (s *Service) ActivateEmployee(ctx context.Context, id string) (domain.Employee, error) {
	status := domain.EmployeeActive
	return s.UpdateEmployee(ctx, id, UpdateEmployeeCommand{Status: &status})
}

// AssignEmployeeRole назначает сотруднику роль и обновляет permission snapshot.
func (s *Service) AssignEmployeeRole(ctx context.Context, id string, cmd AssignRoleCommand) (domain.Employee, error) {
	return s.UpdateEmployee(ctx, id, UpdateEmployeeCommand{RoleID: cmd.RoleID})
}

// RotateEmployeePIN ротирует PIN credential и увеличивает credential version.
func (s *Service) RotateEmployeePIN(ctx context.Context, id string, cmd RotatePINCommand) (domain.Employee, error) {
	employee, err := s.repo.GetEmployee(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Employee{}, err
	}
	if err := s.ensurePINUnique(ctx, employee.ID, cmd.PIN); err != nil {
		return domain.Employee{}, err
	}
	pinHash, err := hashPIN(cmd.PIN)
	if err != nil {
		return domain.Employee{}, err
	}
	employee.PINHash = pinHash
	employee.PINConfigured = true
	employee.PINCredentialVersion++
	employee.CloudVersion++
	employee.UpdatedAt = s.clock.Now().UTC()
	stored, err := s.repo.UpdateEmployee(ctx, employee)
	return afterTenantCommit(s, ctx, stored, err)
}

// CreateCatalogItem создает draft catalog item в Cloud-owned catalog.
func (s *Service) CreateCatalogItem(ctx context.Context, cmd CreateCatalogItemCommand) (domain.CatalogItem, error) {
	if cmd.Kind == "" && cmd.Type != "" {
		cmd.Kind = cmd.Type
	}
	if err := validateCatalogFields(cmd.Kind, cmd.Name, cmd.SKU, cmd.BaseUnit); err != nil {
		return domain.CatalogItem{}, err
	}
	now := s.clock.Now().UTC()
	item := domain.CatalogItem{
		ID:                 s.ids.NewID(),
		RestaurantID:       strings.TrimSpace(cmd.RestaurantID),
		Kind:               cmd.Kind,
		FolderID:           strings.TrimSpace(cmd.FolderID),
		Name:               strings.TrimSpace(cmd.Name),
		SKU:                strings.TrimSpace(cmd.SKU),
		BaseUnit:           strings.TrimSpace(cmd.BaseUnit),
		KitchenType:        strings.TrimSpace(cmd.KitchenType),
		AccountingCategory: strings.TrimSpace(cmd.AccountingCategory),
		Status:             domain.StatusPublished,
		CloudVersion:       1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	stored, err := s.repo.CreateCatalogItem(ctx, item)
	return afterRestaurantCommit(s, ctx, item.RestaurantID, stored, err)
}

// ListCatalogItems возвращает catalog items ресторана.
func (s *Service) ListCatalogItems(ctx context.Context, restaurantID string) ([]domain.CatalogItem, error) {
	return s.repo.ListCatalogItems(ctx, strings.TrimSpace(restaurantID))
}

// GetCatalogItem возвращает один catalog item.
func (s *Service) GetCatalogItem(ctx context.Context, id string) (domain.CatalogItem, error) {
	return s.repo.GetCatalogItem(ctx, strings.TrimSpace(id))
}

// UpdateCatalogItem обновляет catalog item и его lifecycle.
func (s *Service) UpdateCatalogItem(ctx context.Context, id string, cmd UpdateCatalogItemCommand) (domain.CatalogItem, error) {
	item, err := s.repo.GetCatalogItem(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.CatalogItem{}, err
	}
	if cmd.Kind != nil {
		if err := domain.ValidateCatalogItemKind(*cmd.Kind); err != nil {
			return domain.CatalogItem{}, err
		}
		item.Kind = *cmd.Kind
	}
	if cmd.Type != nil {
		if err := domain.ValidateCatalogItemKind(*cmd.Type); err != nil {
			return domain.CatalogItem{}, err
		}
		item.Kind = *cmd.Type
	}
	if strings.TrimSpace(cmd.Name) != "" {
		item.Name = strings.TrimSpace(cmd.Name)
	}
	if strings.TrimSpace(cmd.SKU) != "" {
		item.SKU = strings.TrimSpace(cmd.SKU)
	}
	if strings.TrimSpace(cmd.BaseUnit) != "" {
		item.BaseUnit = strings.TrimSpace(cmd.BaseUnit)
	}
	if cmd.FolderID != nil {
		item.FolderID = strings.TrimSpace(*cmd.FolderID)
	}
	if cmd.KitchenType != nil {
		item.KitchenType = strings.TrimSpace(*cmd.KitchenType)
	}
	if cmd.AccountingCategory != nil {
		item.AccountingCategory = strings.TrimSpace(*cmd.AccountingCategory)
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.CatalogItem{}, err
		}
		item.Status = *cmd.Status
	}
	item.CloudVersion++
	item.UpdatedAt = s.clock.Now().UTC()
	if item.Status == domain.StatusArchived && item.ArchivedAt == nil {
		archivedAt := item.UpdatedAt
		item.ArchivedAt = &archivedAt
	}
	if item.Status != domain.StatusArchived {
		item.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateCatalogItem(ctx, item)
	return afterRestaurantCommit(s, ctx, item.RestaurantID, stored, err)
}

// ArchiveCatalogItem архивирует catalog item без физического удаления.
func (s *Service) ArchiveCatalogItem(ctx context.Context, id string) (domain.CatalogItem, error) {
	status := domain.StatusArchived
	return s.UpdateCatalogItem(ctx, id, UpdateCatalogItemCommand{Status: &status})
}

func (s *Service) CreateCatalogFolder(ctx context.Context, cmd CreateCatalogFolderCommand) (domain.CatalogFolder, error) {
	restaurantID, name := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.Name)
	if name == "" {
		return domain.CatalogFolder{}, fmt.Errorf("%w: name is required", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	folder := domain.CatalogFolder{ID: s.ids.NewID(), RestaurantID: restaurantID, ParentID: strings.TrimSpace(cmd.ParentID), Name: name, SortOrder: cmd.SortOrder, Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	stored, err := s.repo.CreateCatalogFolder(ctx, folder)
	return afterRestaurantCommit(s, ctx, folder.RestaurantID, stored, err)
}

func (s *Service) ListCatalogFolders(ctx context.Context, restaurantID string) ([]domain.CatalogFolder, error) {
	return s.repo.ListCatalogFolders(ctx, strings.TrimSpace(restaurantID))
}

func (s *Service) UpdateCatalogFolder(ctx context.Context, id string, cmd UpdateCatalogFolderCommand) (domain.CatalogFolder, error) {
	folder, err := s.repo.GetCatalogFolder(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.CatalogFolder{}, err
	}
	if cmd.ParentID != nil {
		folder.ParentID = strings.TrimSpace(*cmd.ParentID)
	}
	if strings.TrimSpace(cmd.Name) != "" {
		folder.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.SortOrder != nil {
		folder.SortOrder = *cmd.SortOrder
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.CatalogFolder{}, err
		}
		folder.Status = *cmd.Status
	}
	folder.CloudVersion++
	folder.UpdatedAt = s.clock.Now().UTC()
	if folder.Status == domain.StatusArchived && folder.ArchivedAt == nil {
		archivedAt := folder.UpdatedAt
		folder.ArchivedAt = &archivedAt
	}
	if folder.Status != domain.StatusArchived {
		folder.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateCatalogFolder(ctx, folder)
	return afterRestaurantCommit(s, ctx, folder.RestaurantID, stored, err)
}

func (s *Service) ArchiveCatalogFolder(ctx context.Context, id string) (domain.CatalogFolder, error) {
	status := domain.StatusArchived
	return s.UpdateCatalogFolder(ctx, id, UpdateCatalogFolderCommand{Status: &status})
}

func (s *Service) CreateFolderParameter(ctx context.Context, cmd CreateFolderParameterCommand) (domain.FolderParameter, error) {
	restaurantID, folderID, key := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.FolderID), strings.TrimSpace(cmd.Key)
	valueType, valueJSON := strings.TrimSpace(cmd.ValueType), strings.TrimSpace(cmd.ValueJSON)
	if restaurantID == "" || folderID == "" || key == "" || valueType == "" || valueJSON == "" || !json.Valid([]byte(valueJSON)) {
		return domain.FolderParameter{}, fmt.Errorf("%w: folder parameter requires restaurant_id, folder_id, parameter_key, value_type and valid value_json", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	parameter := domain.FolderParameter{ID: s.ids.NewID(), RestaurantID: restaurantID, FolderID: folderID, Key: key, ValueType: valueType, ValueJSON: canonicalJSON(valueJSON), Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	stored, err := s.repo.CreateFolderParameter(ctx, parameter)
	return afterRestaurantCommit(s, ctx, parameter.RestaurantID, stored, err)
}

func (s *Service) ListFolderParameters(ctx context.Context, restaurantID string) ([]domain.FolderParameter, error) {
	return s.repo.ListFolderParameters(ctx, strings.TrimSpace(restaurantID))
}

func (s *Service) UpdateFolderParameter(ctx context.Context, id string, cmd UpdateFolderParameterCommand) (domain.FolderParameter, error) {
	parameter, err := s.repo.GetFolderParameter(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.FolderParameter{}, err
	}
	if strings.TrimSpace(cmd.ValueType) != "" {
		parameter.ValueType = strings.TrimSpace(cmd.ValueType)
	}
	if strings.TrimSpace(cmd.ValueJSON) != "" {
		if !json.Valid([]byte(cmd.ValueJSON)) {
			return domain.FolderParameter{}, fmt.Errorf("%w: value_json must be valid JSON", domain.ErrInvalid)
		}
		parameter.ValueJSON = canonicalJSON(cmd.ValueJSON)
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.FolderParameter{}, err
		}
		parameter.Status = *cmd.Status
	}
	parameter.CloudVersion++
	parameter.UpdatedAt = s.clock.Now().UTC()
	if parameter.Status == domain.StatusArchived && parameter.ArchivedAt == nil {
		archivedAt := parameter.UpdatedAt
		parameter.ArchivedAt = &archivedAt
	}
	if parameter.Status != domain.StatusArchived {
		parameter.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateFolderParameter(ctx, parameter)
	return afterRestaurantCommit(s, ctx, parameter.RestaurantID, stored, err)
}

func (s *Service) CreateCatalogTag(ctx context.Context, cmd CreateCatalogTagCommand) (domain.CatalogTag, error) {
	restaurantID, name, code := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.Name), strings.TrimSpace(cmd.Code)
	if name == "" || code == "" {
		return domain.CatalogTag{}, fmt.Errorf("%w: name and code are required", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	tag := domain.CatalogTag{ID: s.ids.NewID(), RestaurantID: restaurantID, Name: name, Code: code, Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	stored, err := s.repo.CreateCatalogTag(ctx, tag)
	return afterRestaurantCommit(s, ctx, tag.RestaurantID, stored, err)
}

func (s *Service) ListCatalogTags(ctx context.Context, restaurantID string) ([]domain.CatalogTag, error) {
	return s.repo.ListCatalogTags(ctx, strings.TrimSpace(restaurantID))
}

func (s *Service) UpdateCatalogTag(ctx context.Context, id string, cmd UpdateCatalogTagCommand) (domain.CatalogTag, error) {
	tag, err := s.repo.GetCatalogTag(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.CatalogTag{}, err
	}
	if strings.TrimSpace(cmd.Name) != "" {
		tag.Name = strings.TrimSpace(cmd.Name)
	}
	if strings.TrimSpace(cmd.Code) != "" {
		tag.Code = strings.TrimSpace(cmd.Code)
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.CatalogTag{}, err
		}
		tag.Status = *cmd.Status
	}
	tag.CloudVersion++
	tag.UpdatedAt = s.clock.Now().UTC()
	if tag.Status == domain.StatusArchived && tag.ArchivedAt == nil {
		archivedAt := tag.UpdatedAt
		tag.ArchivedAt = &archivedAt
	}
	if tag.Status != domain.StatusArchived {
		tag.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateCatalogTag(ctx, tag)
	return afterRestaurantCommit(s, ctx, tag.RestaurantID, stored, err)
}

func (s *Service) AssignCatalogItemTag(ctx context.Context, cmd AssignCatalogItemTagCommand) (domain.CatalogItemTag, error) {
	restaurantID, itemID, tagID := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.CatalogItemID), strings.TrimSpace(cmd.TagID)
	if itemID == "" || tagID == "" {
		return domain.CatalogItemTag{}, fmt.Errorf("%w: catalog_item_id and tag_id are required", domain.ErrInvalid)
	}
	if _, err := s.ensureCatalogItemAvailable(ctx, itemID); err != nil {
		return domain.CatalogItemTag{}, err
	}
	if _, err := s.repo.GetCatalogTag(ctx, tagID); err != nil {
		return domain.CatalogItemTag{}, err
	}
	tag := domain.CatalogItemTag{RestaurantID: restaurantID, CatalogItemID: itemID, TagID: tagID, CloudVersion: 1, CreatedAt: s.clock.Now().UTC()}
	stored, err := s.repo.AssignCatalogItemTag(ctx, tag)
	return afterRestaurantCommit(s, ctx, tag.RestaurantID, stored, err)
}

func (s *Service) CreateModifierGroup(ctx context.Context, cmd CreateModifierGroupCommand) (domain.ModifierGroup, error) {
	restaurantID, name := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.Name)
	if restaurantID == "" || name == "" || cmd.MinCount < 0 || cmd.MaxCount < 0 || (cmd.MaxCount > 0 && cmd.MinCount > cmd.MaxCount) {
		return domain.ModifierGroup{}, fmt.Errorf("%w: modifier group requires valid restaurant_id, name and min/max counts", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	group := domain.ModifierGroup{ID: s.ids.NewID(), RestaurantID: restaurantID, Name: name, Required: cmd.Required, MinCount: cmd.MinCount, MaxCount: cmd.MaxCount, Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	stored, err := s.repo.CreateModifierGroup(ctx, group)
	return afterRestaurantCommit(s, ctx, group.RestaurantID, stored, err)
}

func (s *Service) ListModifierGroups(ctx context.Context, restaurantID string) ([]domain.ModifierGroup, error) {
	return s.repo.ListModifierGroups(ctx, strings.TrimSpace(restaurantID))
}

func (s *Service) UpdateModifierGroup(ctx context.Context, id string, cmd UpdateModifierGroupCommand) (domain.ModifierGroup, error) {
	group, err := s.repo.GetModifierGroup(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.ModifierGroup{}, err
	}
	if strings.TrimSpace(cmd.Name) != "" {
		group.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.Required != nil {
		group.Required = *cmd.Required
	}
	if cmd.MinCount != nil {
		group.MinCount = *cmd.MinCount
	}
	if cmd.MaxCount != nil {
		group.MaxCount = *cmd.MaxCount
	}
	if group.MinCount < 0 || group.MaxCount < 0 || (group.MaxCount > 0 && group.MinCount > group.MaxCount) {
		return domain.ModifierGroup{}, fmt.Errorf("%w: modifier group min/max counts are invalid", domain.ErrInvalid)
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.ModifierGroup{}, err
		}
		group.Status = *cmd.Status
	}
	group.CloudVersion++
	group.UpdatedAt = s.clock.Now().UTC()
	if group.Status == domain.StatusArchived && group.ArchivedAt == nil {
		archivedAt := group.UpdatedAt
		group.ArchivedAt = &archivedAt
	}
	if group.Status != domain.StatusArchived {
		group.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateModifierGroup(ctx, group)
	return afterRestaurantCommit(s, ctx, group.RestaurantID, stored, err)
}

func (s *Service) CreateModifierOption(ctx context.Context, cmd CreateModifierOptionCommand) (domain.ModifierOption, error) {
	restaurantID, groupID, name := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.ModifierGroupID), strings.TrimSpace(cmd.Name)
	price := cmd.PriceMinor
	if cmd.LegacyPriceDelta != nil && price == 0 {
		price = *cmd.LegacyPriceDelta
	}
	if restaurantID == "" || groupID == "" || name == "" || price < 0 {
		return domain.ModifierOption{}, fmt.Errorf("%w: modifier option requires restaurant_id, modifier_group_id, name and non-negative price_minor", domain.ErrInvalid)
	}
	linkedCatalogItemID, err := s.validateLinkedModifierCatalogItem(ctx, restaurantID, cmd.LinkedCatalogItemID)
	if err != nil {
		return domain.ModifierOption{}, err
	}
	now := s.clock.Now().UTC()
	option := domain.ModifierOption{ID: s.ids.NewID(), RestaurantID: restaurantID, ModifierGroupID: groupID, LinkedCatalogItemID: linkedCatalogItemID, Name: name, PriceMinor: price, Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	stored, err := s.repo.CreateModifierOption(ctx, option)
	return afterRestaurantCommit(s, ctx, option.RestaurantID, stored, err)
}

func (s *Service) ListModifierOptions(ctx context.Context, restaurantID string) ([]domain.ModifierOption, error) {
	return s.repo.ListModifierOptions(ctx, strings.TrimSpace(restaurantID))
}

func (s *Service) UpdateModifierOption(ctx context.Context, id string, cmd UpdateModifierOptionCommand) (domain.ModifierOption, error) {
	option, err := s.repo.GetModifierOption(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.ModifierOption{}, err
	}
	if strings.TrimSpace(cmd.Name) != "" {
		option.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.LinkedCatalogItemID != nil {
		linkedCatalogItemID, err := s.validateLinkedModifierCatalogItem(ctx, option.RestaurantID, *cmd.LinkedCatalogItemID)
		if err != nil {
			return domain.ModifierOption{}, err
		}
		option.LinkedCatalogItemID = linkedCatalogItemID
	}
	if cmd.PriceMinor != nil {
		if *cmd.PriceMinor < 0 {
			return domain.ModifierOption{}, fmt.Errorf("%w: price_minor must be non-negative", domain.ErrInvalid)
		}
		option.PriceMinor = *cmd.PriceMinor
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.ModifierOption{}, err
		}
		option.Status = *cmd.Status
	}
	option.CloudVersion++
	option.UpdatedAt = s.clock.Now().UTC()
	if option.Status == domain.StatusArchived && option.ArchivedAt == nil {
		archivedAt := option.UpdatedAt
		option.ArchivedAt = &archivedAt
	}
	if option.Status != domain.StatusArchived {
		option.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateModifierOption(ctx, option)
	return afterRestaurantCommit(s, ctx, option.RestaurantID, stored, err)
}

func (s *Service) CreateModifierGroupBinding(ctx context.Context, cmd CreateModifierGroupBindingCommand) (domain.ModifierGroupBinding, error) {
	restaurantID, groupID, targetID := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.ModifierGroupID), strings.TrimSpace(cmd.TargetID)
	if restaurantID == "" || groupID == "" || targetID == "" {
		return domain.ModifierGroupBinding{}, fmt.Errorf("%w: modifier binding requires restaurant_id, modifier_group_id and target_id", domain.ErrInvalid)
	}
	if err := domain.ValidateModifierTargetType(cmd.TargetType); err != nil {
		return domain.ModifierGroupBinding{}, err
	}
	now := s.clock.Now().UTC()
	binding := domain.ModifierGroupBinding{ID: s.ids.NewID(), RestaurantID: restaurantID, ModifierGroupID: groupID, TargetType: cmd.TargetType, TargetID: targetID, SortOrder: cmd.SortOrder, Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	stored, err := s.repo.CreateModifierGroupBinding(ctx, binding)
	return afterRestaurantCommit(s, ctx, binding.RestaurantID, stored, err)
}

func (s *Service) ListModifierGroupBindings(ctx context.Context, restaurantID string) ([]domain.ModifierGroupBinding, error) {
	return s.repo.ListModifierGroupBindings(ctx, strings.TrimSpace(restaurantID))
}

func (s *Service) UpdateModifierGroupBinding(ctx context.Context, id string, cmd UpdateModifierGroupBindingCommand) (domain.ModifierGroupBinding, error) {
	binding, err := s.repo.GetModifierGroupBinding(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.ModifierGroupBinding{}, err
	}
	if cmd.SortOrder != nil {
		binding.SortOrder = *cmd.SortOrder
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.ModifierGroupBinding{}, err
		}
		binding.Status = *cmd.Status
	}
	binding.CloudVersion++
	binding.UpdatedAt = s.clock.Now().UTC()
	if binding.Status == domain.StatusArchived && binding.ArchivedAt == nil {
		archivedAt := binding.UpdatedAt
		binding.ArchivedAt = &archivedAt
	}
	if binding.Status != domain.StatusArchived {
		binding.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateModifierGroupBinding(ctx, binding)
	return afterRestaurantCommit(s, ctx, binding.RestaurantID, stored, err)
}

func (s *Service) CreatePricingPolicy(ctx context.Context, cmd CreatePricingPolicyCommand) (domain.PricingPolicy, error) {
	restaurantID, name := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.Name)
	if restaurantID == "" || name == "" || cmd.ApplicationIndex <= 0 {
		return domain.PricingPolicy{}, fmt.Errorf("%w: pricing policy requires restaurant_id, name and positive application_index", domain.ErrInvalid)
	}
	if err := domain.ValidatePricingPolicyKind(cmd.Kind); err != nil {
		return domain.PricingPolicy{}, err
	}
	scope := strings.TrimSpace(cmd.Scope)
	if cmd.Kind == domain.PricingPolicySurcharge {
		scope = "order"
	}
	if err := validatePolicyScope(cmd.Kind, scope); err != nil {
		return domain.PricingPolicy{}, err
	}
	if err := validatePolicyAmount(cmd.AmountKind, cmd.AmountMinor, cmd.ValueBasisPoints); err != nil {
		return domain.PricingPolicy{}, err
	}
	now := s.clock.Now().UTC()
	policy := domain.PricingPolicy{ID: s.ids.NewID(), RestaurantID: restaurantID, Name: name, Kind: cmd.Kind, Scope: scope, AmountKind: strings.TrimSpace(cmd.AmountKind), AmountMinor: cmd.AmountMinor, ValueBasisPoints: cmd.ValueBasisPoints, ApplicationIndex: cmd.ApplicationIndex, Manual: cmd.Manual, RequiresPermission: strings.TrimSpace(cmd.RequiresPermission), Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	stored, err := s.repo.CreatePricingPolicy(ctx, policy)
	return afterRestaurantCommit(s, ctx, policy.RestaurantID, stored, err)
}

func (s *Service) ListPricingPolicies(ctx context.Context, restaurantID string) ([]domain.PricingPolicy, error) {
	return s.repo.ListPricingPolicies(ctx, strings.TrimSpace(restaurantID))
}

func (s *Service) UpdatePricingPolicy(ctx context.Context, id string, cmd UpdatePricingPolicyCommand) (domain.PricingPolicy, error) {
	policy, err := s.repo.GetPricingPolicy(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.PricingPolicy{}, err
	}
	if strings.TrimSpace(cmd.Name) != "" {
		policy.Name = strings.TrimSpace(cmd.Name)
	}
	if strings.TrimSpace(cmd.Scope) != "" {
		policy.Scope = strings.TrimSpace(cmd.Scope)
	}
	if policy.Kind == domain.PricingPolicySurcharge {
		policy.Scope = "order"
	}
	if strings.TrimSpace(cmd.AmountKind) != "" {
		policy.AmountKind = strings.TrimSpace(cmd.AmountKind)
	}
	if cmd.AmountMinor != nil {
		policy.AmountMinor = *cmd.AmountMinor
	}
	if cmd.ValueBasisPoints != nil {
		policy.ValueBasisPoints = *cmd.ValueBasisPoints
	}
	if cmd.ApplicationIndex != nil {
		policy.ApplicationIndex = *cmd.ApplicationIndex
	}
	if cmd.Manual != nil {
		policy.Manual = *cmd.Manual
	}
	if cmd.RequiresPermission != nil {
		policy.RequiresPermission = strings.TrimSpace(*cmd.RequiresPermission)
	}
	if err := validatePolicyScope(policy.Kind, policy.Scope); err != nil {
		return domain.PricingPolicy{}, err
	}
	if err := validatePolicyAmount(policy.AmountKind, policy.AmountMinor, policy.ValueBasisPoints); err != nil {
		return domain.PricingPolicy{}, err
	}
	if policy.ApplicationIndex <= 0 {
		return domain.PricingPolicy{}, fmt.Errorf("%w: application_index must be positive", domain.ErrInvalid)
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.PricingPolicy{}, err
		}
		policy.Status = *cmd.Status
	}
	policy.CloudVersion++
	policy.UpdatedAt = s.clock.Now().UTC()
	if policy.Status == domain.StatusArchived && policy.ArchivedAt == nil {
		archivedAt := policy.UpdatedAt
		policy.ArchivedAt = &archivedAt
	}
	if policy.Status != domain.StatusArchived {
		policy.ArchivedAt = nil
	}
	stored, err := s.repo.UpdatePricingPolicy(ctx, policy)
	return afterRestaurantCommit(s, ctx, policy.RestaurantID, stored, err)
}

// CreateRecipeItem добавляет Cloud-owned recipe component для будущей публикации на Edge.
func (s *Service) CreateRecipeItem(ctx context.Context, cmd CreateRecipeItemCommand) (domain.RecipeItem, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	ownerID := strings.TrimSpace(cmd.RecipeOwnerCatalogItemID)
	componentID := strings.TrimSpace(cmd.ComponentCatalogItemID)
	unit := strings.TrimSpace(cmd.Unit)
	if restaurantID == "" || ownerID == "" || componentID == "" || cmd.Quantity <= 0 || unit == "" || cmd.LossPercent < 0 || cmd.LossPercent > 100 {
		return domain.RecipeItem{}, fmt.Errorf("%w: recipe item requires restaurant_id, owner, component, positive quantity, unit and loss_percent 0..100", domain.ErrInvalid)
	}
	if err := s.ensureRestaurantActive(ctx, restaurantID); err != nil {
		return domain.RecipeItem{}, err
	}
	if err := s.ensureCatalogItemKind(ctx, restaurantID, ownerID, domain.CatalogItemDish, domain.CatalogItemSemiFinished); err != nil {
		return domain.RecipeItem{}, err
	}
	if err := s.ensureCatalogItemKind(ctx, restaurantID, componentID, domain.CatalogItemGood, domain.CatalogItemSemiFinished); err != nil {
		return domain.RecipeItem{}, err
	}
	now := s.clock.Now().UTC()
	item := domain.RecipeItem{
		ID:                       s.ids.NewID(),
		RestaurantID:             restaurantID,
		RecipeOwnerCatalogItemID: ownerID,
		ComponentCatalogItemID:   componentID,
		Quantity:                 cmd.Quantity,
		Unit:                     unit,
		LossPercent:              cmd.LossPercent,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	stored, err := s.repo.CreateRecipeItem(ctx, item)
	return afterRestaurantCommit(s, ctx, item.RestaurantID, stored, err)
}

// ListRecipeItems возвращает recipe reference rows ресторана.
func (s *Service) ListRecipeItems(ctx context.Context, restaurantID string) ([]domain.RecipeItem, error) {
	return s.repo.ListRecipeItems(ctx, strings.TrimSpace(restaurantID))
}

// CreateRecipeVersionDraft создает новую draft version и опционально отправляет ее в manager review.
func (s *Service) CreateRecipeVersionDraft(ctx context.Context, cmd CreateRecipeVersionDraftCommand) (RecipeVersionView, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	ownerID := strings.TrimSpace(cmd.OwnerCatalogItemID)
	name := strings.TrimSpace(cmd.Name)
	yieldUnit := strings.TrimSpace(cmd.YieldUnit)
	if name == "" {
		name = ownerID + " recipe"
	}
	if cmd.YieldQuantity <= 0 {
		cmd.YieldQuantity = 1
	}
	if yieldUnit == "" {
		yieldUnit = "portion"
	}
	lines, err := s.validateRecipeVersionLines(ctx, restaurantID, ownerID, cmd.Lines)
	if err != nil {
		return RecipeVersionView{}, err
	}
	if err := s.ensureRestaurantActive(ctx, restaurantID); err != nil {
		return RecipeVersionView{}, err
	}
	if err := s.ensureCatalogItemKind(ctx, restaurantID, ownerID, domain.CatalogItemDish, domain.CatalogItemSemiFinished); err != nil {
		return RecipeVersionView{}, err
	}
	existing, err := s.repo.ListRecipeVersions(ctx, restaurantID, ownerID, "", 200, 0)
	if err != nil {
		return RecipeVersionView{}, err
	}
	versionNo := 1
	for _, item := range existing {
		if item.Version >= versionNo {
			versionNo = item.Version + 1
		}
	}
	now := s.clock.Now().UTC()
	version := domain.RecipeVersion{
		ID:                  s.ids.NewID(),
		RestaurantID:        restaurantID,
		OwnerCatalogItemID:  ownerID,
		Version:             versionNo,
		Name:                name,
		Status:              domain.RecipeVersionStatusDraft,
		YieldQuantity:       cmd.YieldQuantity,
		YieldUnit:           yieldUnit,
		CreatedByEmployeeID: strings.TrimSpace(cmd.CreatedByEmployeeID),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	for i := range lines {
		lines[i].ID = s.ids.NewID()
		lines[i].RecipeVersionID = version.ID
		lines[i].SortOrder = i + 1
		lines[i].CreatedAt = now
		lines[i].UpdatedAt = now
	}
	stored, err := s.repo.CreateRecipeVersion(ctx, version, lines)
	if err != nil {
		return RecipeVersionView{}, err
	}
	if cmd.SubmitForReview {
		if _, err := s.SubmitRecipeVersion(ctx, stored.ID, SubmitRecipeVersionCommand{SubmittedByEmployeeID: strings.TrimSpace(cmd.CreatedByEmployeeID), Reason: cmd.Reason}); err != nil {
			return RecipeVersionView{}, err
		}
		stored, err = s.repo.GetRecipeVersion(ctx, stored.ID)
		if err != nil {
			return RecipeVersionView{}, err
		}
	}
	storedLines, err := s.repo.ListRecipeLines(ctx, stored.ID)
	if err != nil {
		return RecipeVersionView{}, err
	}
	return RecipeVersionView{Version: stored, Lines: storedLines}, nil
}

// ListRecipeVersions возвращает bounded список Cloud recipe versions.
func (s *Service) ListRecipeVersions(ctx context.Context, restaurantID, ownerCatalogItemID, status string, limit, offset int) ([]RecipeVersionView, error) {
	versions, err := s.repo.ListRecipeVersions(ctx, strings.TrimSpace(restaurantID), strings.TrimSpace(ownerCatalogItemID), strings.TrimSpace(status), limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]RecipeVersionView, 0, len(versions))
	for _, version := range versions {
		lines, err := s.repo.ListRecipeLines(ctx, version.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, RecipeVersionView{Version: version, Lines: lines})
	}
	return out, nil
}

// SubmitRecipeVersion отправляет draft в существующую RecipeChangeSuggested review queue.
func (s *Service) SubmitRecipeVersion(ctx context.Context, id string, cmd SubmitRecipeVersionCommand) (domain.RecipeSuggestion, error) {
	version, err := s.repo.GetRecipeVersion(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.RecipeSuggestion{}, err
	}
	if version.Status == domain.RecipeVersionStatusActive {
		return domain.RecipeSuggestion{}, fmt.Errorf("%w: active recipe version is already published", domain.ErrConflict)
	}
	if version.Status == domain.RecipeVersionStatusArchived {
		return domain.RecipeSuggestion{}, fmt.Errorf("%w: archived recipe version cannot be submitted", domain.ErrConflict)
	}
	lines, err := s.repo.ListRecipeLines(ctx, version.ID)
	if err != nil {
		return domain.RecipeSuggestion{}, err
	}
	if len(lines) == 0 {
		return domain.RecipeSuggestion{}, fmt.Errorf("%w: recipe draft requires at least one line", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	suggestionID := "recipe-version-" + version.ID
	payload, err := json.Marshal(map[string]any{
		"data": map[string]any{
			"suggestion_id":            suggestionID,
			"restaurant_id":            version.RestaurantID,
			"recipe_version_id":        version.ID,
			"owner_catalog_item_id":    version.OwnerCatalogItemID,
			"action":                   "publish_recipe_version",
			"reason":                   strings.TrimSpace(cmd.Reason),
			"suggested_by_employee_id": strings.TrimSpace(cmd.SubmittedByEmployeeID),
			"suggested_at":             now,
			"changes":                  recipeVersionChangesPayload(lines),
		},
	})
	if err != nil {
		return domain.RecipeSuggestion{}, err
	}
	changes := make([]domain.RecipeSuggestionChange, 0, len(lines))
	for i, line := range lines {
		changes = append(changes, domain.RecipeSuggestionChange{
			ID:                 fmt.Sprintf("%s-change-%d", suggestionID, i+1),
			RecipeSuggestionID: "recipe-suggestion-" + suggestionID,
			LineID:             line.ID,
			Action:             "add_ingredient",
			ToCatalogItemID:    line.ComponentCatalogItemID,
			Quantity:           strconv.FormatInt(line.Quantity, 10),
			UnitCode:           line.Unit,
			LossPercent:        strconv.FormatInt(line.LossPercent, 10),
			SortOrder:          line.SortOrder,
			CreatedAt:          now,
		})
	}
	suggestion := domain.RecipeSuggestion{
		ID:                 "recipe-suggestion-" + suggestionID,
		SuggestionID:       suggestionID,
		RestaurantID:       version.RestaurantID,
		RecipeVersionID:    version.ID,
		OwnerCatalogItemID: version.OwnerCatalogItemID,
		Action:             "publish_recipe_version",
		Reason:             strings.TrimSpace(cmd.Reason),
		Status:             domain.SuggestionStatusPending,
		SuggestedAt:        now,
		CloudReceivedAt:    now,
		PayloadJSON:        payload,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	stored, err := s.repo.SubmitRecipeSuggestion(ctx, suggestion, changes)
	if err != nil {
		return domain.RecipeSuggestion{}, err
	}
	version.Status = domain.RecipeVersionStatusReviewPending
	version.SubmittedByEmployeeID = strings.TrimSpace(cmd.SubmittedByEmployeeID)
	version.SubmittedAt = &now
	version.UpdatedAt = now
	if _, updateErr := s.repo.UpdateRecipeVersion(ctx, version); updateErr != nil {
		return domain.RecipeSuggestion{}, updateErr
	}
	return stored, nil
}

// UpdateRecipeItem изменяет recipe component без создания Edge-side stock documents.
func (s *Service) UpdateRecipeItem(ctx context.Context, id string, cmd UpdateRecipeItemCommand) (domain.RecipeItem, error) {
	item, err := s.repo.GetRecipeItem(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.RecipeItem{}, err
	}
	if cmd.Quantity != nil {
		if *cmd.Quantity <= 0 {
			return domain.RecipeItem{}, fmt.Errorf("%w: recipe quantity must be positive", domain.ErrInvalid)
		}
		item.Quantity = *cmd.Quantity
	}
	if strings.TrimSpace(cmd.Unit) != "" {
		item.Unit = strings.TrimSpace(cmd.Unit)
	}
	if cmd.LossPercent != nil {
		if *cmd.LossPercent < 0 || *cmd.LossPercent > 100 {
			return domain.RecipeItem{}, fmt.Errorf("%w: loss_percent must be 0..100", domain.ErrInvalid)
		}
		item.LossPercent = *cmd.LossPercent
	}
	item.UpdatedAt = s.clock.Now().UTC()
	stored, err := s.repo.UpdateRecipeItem(ctx, item)
	return afterRestaurantCommit(s, ctx, item.RestaurantID, stored, err)
}

// UpsertStopListEntry создает или обновляет Cloud-owned stop-list row по restaurant/catalog item.
func (s *Service) UpsertStopListEntry(ctx context.Context, cmd UpsertStopListEntryCommand) (domain.StopListEntry, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	catalogItemID := strings.TrimSpace(cmd.CatalogItemID)
	if restaurantID == "" || catalogItemID == "" {
		return domain.StopListEntry{}, fmt.Errorf("%w: restaurant_id and catalog_item_id are required", domain.ErrInvalid)
	}
	if cmd.AvailableQuantity != nil && *cmd.AvailableQuantity < 0 {
		return domain.StopListEntry{}, fmt.Errorf("%w: available_quantity must be non-negative or null", domain.ErrInvalid)
	}
	if err := s.ensureRestaurantActive(ctx, restaurantID); err != nil {
		return domain.StopListEntry{}, err
	}
	if _, err := s.ensureCatalogItemInRestaurant(ctx, restaurantID, catalogItemID); err != nil {
		return domain.StopListEntry{}, err
	}
	active := true
	if cmd.Active != nil {
		active = *cmd.Active
	}
	now := s.clock.Now().UTC()
	version := int64(1)
	entry := domain.StopListEntry{
		ID:                s.ids.NewID(),
		RestaurantID:      restaurantID,
		CatalogItemID:     catalogItemID,
		AvailableQuantity: cmd.AvailableQuantity,
		Source:            "cloud",
		Reason:            strings.TrimSpace(cmd.Reason),
		Active:            active,
		CloudVersion:      &version,
		UpdatedAt:         now,
	}
	stored, err := s.repo.UpsertStopListEntry(ctx, entry)
	return afterRestaurantCommit(s, ctx, entry.RestaurantID, stored, err)
}

// ListStopListEntries возвращает Cloud-owned stop-list rows ресторана.
func (s *Service) ListStopListEntries(ctx context.Context, restaurantID string) ([]domain.StopListEntry, error) {
	return s.repo.ListStopListEntries(ctx, strings.TrimSpace(restaurantID))
}

// UpdateStopListEntry изменяет Cloud-owned stop-list row по identifier.
func (s *Service) UpdateStopListEntry(ctx context.Context, id string, cmd UpsertStopListEntryCommand) (domain.StopListEntry, error) {
	entry, err := s.repo.GetStopListEntry(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.StopListEntry{}, err
	}
	if cmd.AvailableQuantity != nil && *cmd.AvailableQuantity < 0 {
		return domain.StopListEntry{}, fmt.Errorf("%w: available_quantity must be non-negative or null", domain.ErrInvalid)
	}
	if cmd.AvailableQuantity != nil {
		entry.AvailableQuantity = cmd.AvailableQuantity
	}
	if strings.TrimSpace(cmd.Reason) != "" {
		entry.Reason = strings.TrimSpace(cmd.Reason)
	}
	if cmd.Active != nil {
		entry.Active = *cmd.Active
	}
	version := int64(1)
	if entry.CloudVersion != nil {
		version = *entry.CloudVersion + 1
	}
	entry.CloudVersion = &version
	entry.UpdatedAt = s.clock.Now().UTC()
	stored, err := s.repo.UpsertStopListEntry(ctx, entry)
	return afterRestaurantCommit(s, ctx, entry.RestaurantID, stored, err)
}

// DeactivateStopListEntry переводит stop-list row в inactive, чтобы Edge получил явное снятие блокировки.
func (s *Service) DeactivateStopListEntry(ctx context.Context, id string) (domain.StopListEntry, error) {
	active := false
	return s.UpdateStopListEntry(ctx, id, UpsertStopListEntryCommand{Active: &active})
}

// CreateCategory создает draft категорию меню.
func (s *Service) CreateCategory(ctx context.Context, cmd CreateCategoryCommand) (domain.Category, error) {
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.Name) == "" {
		return domain.Category{}, fmt.Errorf("%w: restaurant_id and name are required", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	category := domain.Category{
		ID:           s.ids.NewID(),
		RestaurantID: strings.TrimSpace(cmd.RestaurantID),
		Name:         strings.TrimSpace(cmd.Name),
		Status:       domain.StatusDraft,
		SortOrder:    cmd.SortOrder,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return s.repo.CreateCategory(ctx, category)
}

// CreateHall создает published зал для Zero-to-Cashier floor stream.
func (s *Service) CreateHall(ctx context.Context, cmd CreateHallCommand) (domain.Hall, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	name := strings.TrimSpace(cmd.Name)
	if restaurantID == "" || name == "" {
		return domain.Hall{}, fmt.Errorf("%w: restaurant_id and name are required", domain.ErrInvalid)
	}
	if err := s.ensureActiveRestaurant(ctx, restaurantID); err != nil {
		return domain.Hall{}, err
	}
	now := s.clock.Now().UTC()
	hall := domain.Hall{
		ID:           s.ids.NewID(),
		RestaurantID: restaurantID,
		Name:         name,
		Status:       domain.StatusPublished,
		CloudVersion: 1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	stored, err := s.repo.CreateHall(ctx, hall)
	return afterRestaurantCommit(s, ctx, hall.RestaurantID, stored, err)
}

// ListHalls возвращает Cloud-owned залы ресторана.
func (s *Service) ListHalls(ctx context.Context, restaurantID string) ([]domain.Hall, error) {
	return s.repo.ListHalls(ctx, strings.TrimSpace(restaurantID))
}

// UpdateHall изменяет зал и его lifecycle.
func (s *Service) UpdateHall(ctx context.Context, id string, cmd UpdateHallCommand) (domain.Hall, error) {
	hall, err := s.repo.GetHall(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Hall{}, err
	}
	if strings.TrimSpace(cmd.Name) != "" {
		hall.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.Hall{}, err
		}
		hall.Status = *cmd.Status
	}
	hall.CloudVersion++
	hall.UpdatedAt = s.clock.Now().UTC()
	if hall.Status == domain.StatusArchived && hall.ArchivedAt == nil {
		archivedAt := hall.UpdatedAt
		hall.ArchivedAt = &archivedAt
	}
	if hall.Status != domain.StatusArchived {
		hall.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateHall(ctx, hall)
	return afterRestaurantCommit(s, ctx, hall.RestaurantID, stored, err)
}

// ArchiveHall архивирует зал без физического удаления.
func (s *Service) ArchiveHall(ctx context.Context, id string) (domain.Hall, error) {
	status := domain.StatusArchived
	return s.UpdateHall(ctx, id, UpdateHallCommand{Status: &status})
}

// CreateTable создает published стол для Zero-to-Cashier floor stream.
func (s *Service) CreateTable(ctx context.Context, cmd CreateTableCommand) (domain.Table, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	hallID := strings.TrimSpace(cmd.HallID)
	name := strings.TrimSpace(cmd.Name)
	if restaurantID == "" || hallID == "" || name == "" || cmd.Seats < 0 {
		return domain.Table{}, fmt.Errorf("%w: restaurant_id, hall_id, name and non-negative seats are required", domain.ErrInvalid)
	}
	if err := s.ensureActiveRestaurant(ctx, restaurantID); err != nil {
		return domain.Table{}, err
	}
	hall, err := s.repo.GetHall(ctx, hallID)
	if err != nil {
		return domain.Table{}, err
	}
	if hall.RestaurantID != restaurantID || hall.Status == domain.StatusArchived {
		return domain.Table{}, fmt.Errorf("%w: hall is archived or belongs to another restaurant", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	table := domain.Table{
		ID:           s.ids.NewID(),
		RestaurantID: restaurantID,
		HallID:       hallID,
		Name:         name,
		Seats:        cmd.Seats,
		Status:       domain.StatusPublished,
		CloudVersion: 1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	stored, err := s.repo.CreateTable(ctx, table)
	return afterRestaurantCommit(s, ctx, table.RestaurantID, stored, err)
}

// ListTables возвращает Cloud-owned столы ресторана.
func (s *Service) ListTables(ctx context.Context, restaurantID string) ([]domain.Table, error) {
	return s.repo.ListTables(ctx, strings.TrimSpace(restaurantID))
}

// UpdateTable изменяет стол и его lifecycle.
func (s *Service) UpdateTable(ctx context.Context, id string, cmd UpdateTableCommand) (domain.Table, error) {
	table, err := s.repo.GetTable(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Table{}, err
	}
	if strings.TrimSpace(cmd.HallID) != "" {
		hall, err := s.repo.GetHall(ctx, strings.TrimSpace(cmd.HallID))
		if err != nil {
			return domain.Table{}, err
		}
		if hall.RestaurantID != table.RestaurantID || hall.Status == domain.StatusArchived {
			return domain.Table{}, fmt.Errorf("%w: hall is archived or belongs to another restaurant", domain.ErrInvalid)
		}
		table.HallID = hall.ID
	}
	if strings.TrimSpace(cmd.Name) != "" {
		table.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.Seats != nil {
		if *cmd.Seats < 0 {
			return domain.Table{}, fmt.Errorf("%w: seats must be non-negative", domain.ErrInvalid)
		}
		table.Seats = *cmd.Seats
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.Table{}, err
		}
		table.Status = *cmd.Status
	}
	table.CloudVersion++
	table.UpdatedAt = s.clock.Now().UTC()
	if table.Status == domain.StatusArchived && table.ArchivedAt == nil {
		archivedAt := table.UpdatedAt
		table.ArchivedAt = &archivedAt
	}
	if table.Status != domain.StatusArchived {
		table.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateTable(ctx, table)
	return afterRestaurantCommit(s, ctx, table.RestaurantID, stored, err)
}

// ArchiveTable архивирует стол без физического удаления.
func (s *Service) ArchiveTable(ctx context.Context, id string) (domain.Table, error) {
	status := domain.StatusArchived
	return s.UpdateTable(ctx, id, UpdateTableCommand{Status: &status})
}

// CreateMenuItem создает draft menu item поверх catalog item.
func (s *Service) CreateMenuItem(ctx context.Context, cmd CreateMenuItemCommand) (domain.MenuItem, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	catalogItemID := strings.TrimSpace(cmd.CatalogItemID)
	name := strings.TrimSpace(cmd.Name)
	currency := strings.ToUpper(strings.TrimSpace(cmd.Currency))
	if restaurantID == "" || catalogItemID == "" || name == "" || currency == "" || cmd.Price < 0 {
		return domain.MenuItem{}, fmt.Errorf("%w: restaurant_id, catalog_item_id, name, currency and non-negative price are required", domain.ErrInvalid)
	}
	if !isCurrencyCode(currency) {
		return domain.MenuItem{}, fmt.Errorf("%w: currency must be ISO-like alpha-3 code", domain.ErrInvalid)
	}
	restaurant, err := s.repo.GetRestaurant(ctx, restaurantID)
	if err != nil {
		return domain.MenuItem{}, err
	}
	if restaurant.Status != domain.RestaurantActive {
		return domain.MenuItem{}, fmt.Errorf("%w: restaurant is archived", domain.ErrInvalid)
	}
	if currency != restaurant.Currency {
		return domain.MenuItem{}, fmt.Errorf("%w: menu item currency must match restaurant currency", domain.ErrInvalid)
	}
	if _, err := s.ensureCatalogItemAvailable(ctx, catalogItemID); err != nil {
		return domain.MenuItem{}, err
	}
	availability, err := normalizeAvailability(cmd.AvailabilityJSON)
	if err != nil {
		return domain.MenuItem{}, err
	}
	runtimeStatus, err := normalizeMenuRuntimeStatus(cmd.RuntimeStatus)
	if err != nil {
		return domain.MenuItem{}, err
	}
	now := s.clock.Now().UTC()
	item := domain.MenuItem{
		ID:                s.ids.NewID(),
		RestaurantID:      restaurantID,
		CatalogItemID:     catalogItemID,
		CategoryID:        strings.TrimSpace(cmd.CategoryID),
		TagID:             strings.TrimSpace(cmd.TagID),
		TaxProfileID:      strings.TrimSpace(cmd.TaxProfileID),
		Name:              name,
		Price:             cmd.Price,
		Currency:          currency,
		Status:            domain.StatusPublished,
		RuntimeStatus:     runtimeStatus,
		AvailabilityJSON:  availability,
		StationRoutingKey: strings.TrimSpace(cmd.StationRoutingKey),
		CloudVersion:      1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	stored, err := s.repo.CreateMenuItem(ctx, item)
	return afterRestaurantCommit(s, ctx, item.RestaurantID, stored, err)
}

// ListMenuItems возвращает menu items ресторана.
func (s *Service) ListMenuItems(ctx context.Context, restaurantID string) ([]domain.MenuItem, error) {
	return s.repo.ListMenuItems(ctx, strings.TrimSpace(restaurantID))
}

// GetMenuItem возвращает один menu item.
func (s *Service) GetMenuItem(ctx context.Context, id string) (domain.MenuItem, error) {
	return s.repo.GetMenuItem(ctx, strings.TrimSpace(id))
}

// UpdateMenuItem обновляет menu item и его lifecycle.
func (s *Service) UpdateMenuItem(ctx context.Context, id string, cmd UpdateMenuItemCommand) (domain.MenuItem, error) {
	item, err := s.repo.GetMenuItem(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.MenuItem{}, err
	}
	if strings.TrimSpace(cmd.CatalogItemID) != "" {
		catalogItem, err := s.ensureCatalogItemAvailable(ctx, strings.TrimSpace(cmd.CatalogItemID))
		if err != nil {
			return domain.MenuItem{}, err
		}
		item.CatalogItemID = catalogItem.ID
	}
	if strings.TrimSpace(cmd.CategoryID) != "" {
		item.CategoryID = strings.TrimSpace(cmd.CategoryID)
	}
	if cmd.TagID != nil {
		item.TagID = strings.TrimSpace(*cmd.TagID)
	}
	if cmd.TaxProfileID != nil {
		item.TaxProfileID = strings.TrimSpace(*cmd.TaxProfileID)
	}
	if strings.TrimSpace(cmd.Name) != "" {
		item.Name = strings.TrimSpace(cmd.Name)
	}
	if cmd.Price != nil {
		if *cmd.Price < 0 {
			return domain.MenuItem{}, fmt.Errorf("%w: price must be non-negative", domain.ErrInvalid)
		}
		item.Price = *cmd.Price
	}
	if strings.TrimSpace(cmd.Currency) != "" {
		currency := strings.ToUpper(strings.TrimSpace(cmd.Currency))
		if !isCurrencyCode(currency) {
			return domain.MenuItem{}, fmt.Errorf("%w: currency must be ISO-like alpha-3 code", domain.ErrInvalid)
		}
		restaurant, err := s.repo.GetRestaurant(ctx, item.RestaurantID)
		if err != nil {
			return domain.MenuItem{}, err
		}
		if restaurant.Status != domain.RestaurantActive {
			return domain.MenuItem{}, fmt.Errorf("%w: restaurant is archived", domain.ErrInvalid)
		}
		if currency != restaurant.Currency {
			return domain.MenuItem{}, fmt.Errorf("%w: menu item currency must match restaurant currency", domain.ErrInvalid)
		}
		item.Currency = currency
	}
	if cmd.Status != nil {
		if err := domain.ValidateLifecycleStatus(*cmd.Status); err != nil {
			return domain.MenuItem{}, err
		}
		item.Status = *cmd.Status
	}
	if cmd.RuntimeStatus != nil {
		runtimeStatus, err := normalizeMenuRuntimeStatus(*cmd.RuntimeStatus)
		if err != nil {
			return domain.MenuItem{}, err
		}
		item.RuntimeStatus = runtimeStatus
	}
	if item.Status == domain.StatusPublished {
		if _, err := s.ensureCatalogItemAvailable(ctx, item.CatalogItemID); err != nil {
			return domain.MenuItem{}, err
		}
	}
	if strings.TrimSpace(cmd.AvailabilityJSON) != "" {
		availability, err := normalizeAvailability(cmd.AvailabilityJSON)
		if err != nil {
			return domain.MenuItem{}, err
		}
		item.AvailabilityJSON = availability
	}
	if strings.TrimSpace(cmd.StationRoutingKey) != "" {
		item.StationRoutingKey = strings.TrimSpace(cmd.StationRoutingKey)
	}
	item.CloudVersion++
	item.UpdatedAt = s.clock.Now().UTC()
	if item.Status == domain.StatusArchived && item.ArchivedAt == nil {
		archivedAt := item.UpdatedAt
		item.ArchivedAt = &archivedAt
	}
	if item.Status != domain.StatusArchived {
		item.ArchivedAt = nil
	}
	stored, err := s.repo.UpdateMenuItem(ctx, item)
	return afterRestaurantCommit(s, ctx, item.RestaurantID, stored, err)
}

// ArchiveMenuItem архивирует menu item без физического удаления.
func (s *Service) ArchiveMenuItem(ctx context.Context, id string) (domain.MenuItem, error) {
	status := domain.StatusArchived
	return s.UpdateMenuItem(ctx, id, UpdateMenuItemCommand{Status: &status})
}

// Publish создает versioned deterministic package для Cloud -> Edge sync.
//
// Deprecated: production delivery обновляется автоматическими refresh-методами после
// assignment и Cloud commits. Метод оставлен для внутренних тестов и legacy route.
func (s *Service) Publish(ctx context.Context, cmd PublishCommand) (PublicationSummary, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	publishedBy := strings.TrimSpace(cmd.PublishedBy)
	if restaurantID == "" || publishedBy == "" {
		return PublicationSummary{}, fmt.Errorf("%w: restaurant_id and published_by are required", domain.ErrInvalid)
	}
	version, err := s.repo.NextPublicationVersion(ctx, restaurantID)
	if err != nil {
		return PublicationSummary{}, err
	}
	now := s.clock.Now().UTC()
	packet, counts, streamPackages, err := s.buildPacket(ctx, restaurantID, strings.TrimSpace(cmd.NodeDeviceID), version, now)
	if err != nil {
		return PublicationSummary{}, err
	}
	body, err := json.Marshal(packet)
	if err != nil {
		return PublicationSummary{}, err
	}
	sum := sha256.Sum256(body)
	pub := domain.Publication{
		ID:            s.ids.NewID(),
		RestaurantID:  restaurantID,
		Version:       version,
		Status:        domain.StatusPublished,
		CloudVersion:  version,
		PublishedAt:   now,
		PublishedBy:   publishedBy,
		PackageJSON:   body,
		PackageSHA256: hex.EncodeToString(sum[:]),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	stored, err := s.repo.SavePublication(ctx, pub, streamPackages)
	if err != nil {
		return PublicationSummary{}, err
	}
	return summarizePublication(stored, counts), nil
}

// RefreshDeliveryPackages обновляет latest Cloud -> Edge packages для всех назначенных Edge выбранного ресторана.
func (s *Service) RefreshDeliveryPackages(ctx context.Context, restaurantID string) (PublicationSummary, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	if restaurantID == "" {
		return PublicationSummary{}, fmt.Errorf("%w: restaurant_id is required", domain.ErrInvalid)
	}
	nodeDeviceIDs, err := s.repo.ListAssignedNodeDeviceIDs(ctx, restaurantID)
	if err != nil {
		return PublicationSummary{}, err
	}
	if len(nodeDeviceIDs) == 0 {
		return PublicationSummary{}, nil
	}
	return s.refreshDeliveryPackages(ctx, restaurantID, nodeDeviceIDs, "cloud-auto")
}

// RefreshDeliveryPackagesForNode собирает current full batch при assignment/first connection конкретного Edge.
func (s *Service) RefreshDeliveryPackagesForNode(ctx context.Context, restaurantID, nodeDeviceID string) (PublicationSummary, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	if restaurantID == "" || nodeDeviceID == "" {
		return PublicationSummary{}, fmt.Errorf("%w: restaurant_id and node_device_id are required", domain.ErrInvalid)
	}
	return s.refreshDeliveryPackages(ctx, restaurantID, []string{nodeDeviceID}, "edge-assignment")
}

func (s *Service) refreshDeliveryPackages(ctx context.Context, restaurantID string, nodeDeviceIDs []string, publishedBy string) (PublicationSummary, error) {
	version, err := s.repo.NextPublicationVersion(ctx, restaurantID)
	if err != nil {
		return PublicationSummary{}, err
	}
	now := s.clock.Now().UTC()
	var packet domain.MasterDataPacket
	var counts map[string]int
	streamPackages := make([]StreamPackage, 0, len(nodeDeviceIDs)*9)
	for index, nodeDeviceID := range nodeDeviceIDs {
		nextPacket, nextCounts, streams, err := s.buildPacket(ctx, restaurantID, strings.TrimSpace(nodeDeviceID), version, now)
		if err != nil {
			return PublicationSummary{}, err
		}
		if index == 0 {
			packet = nextPacket
			counts = nextCounts
		}
		streamPackages = append(streamPackages, streams...)
	}
	body, err := json.Marshal(packet)
	if err != nil {
		return PublicationSummary{}, err
	}
	sum := sha256.Sum256(body)
	pub := domain.Publication{
		ID:            s.ids.NewID(),
		RestaurantID:  restaurantID,
		Version:       version,
		Status:        domain.StatusPublished,
		CloudVersion:  version,
		PublishedAt:   now,
		PublishedBy:   strings.TrimSpace(publishedBy),
		PackageJSON:   body,
		PackageSHA256: hex.EncodeToString(sum[:]),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	stored, err := s.repo.SavePublication(ctx, pub, streamPackages)
	if err != nil {
		return PublicationSummary{}, err
	}
	return summarizePublication(stored, counts), nil
}

// GetCurrentPublishedState возвращает Cloud UI-safe metadata текущей публикации.
func (s *Service) GetCurrentPublishedState(ctx context.Context, restaurantID string) (PublicationSummary, error) {
	pub, err := s.repo.GetCurrentPublication(ctx, strings.TrimSpace(restaurantID))
	if err != nil {
		return PublicationSummary{}, err
	}
	var packet domain.MasterDataPacket
	_ = json.Unmarshal(pub.PackageJSON, &packet)
	counts := map[string]int{
		"restaurants":     len(packet.Restaurants),
		"roles":           len(packet.Roles),
		"employees":       len(packet.Employees),
		"catalog_items":   len(packet.CatalogItems),
		"menu_items":      len(packet.MenuItems),
		"halls":           len(packet.Halls),
		"tables":          len(packet.Tables),
		"recipe_versions": len(packet.RecipeVersions),
		"recipe_lines":    len(packet.RecipeLines),
		"stop_lists":      len(packet.StopLists),
	}
	return summarizePublication(pub, counts), nil
}

// GetCurrentPublishedPackage возвращает последний full multi-stream payload для прямой доставки на POS Edge.
func (s *Service) GetCurrentPublishedPackage(ctx context.Context, restaurantID, nodeDeviceID string) (domain.MasterDataPacket, error) {
	pub, err := s.repo.GetCurrentPublication(ctx, strings.TrimSpace(restaurantID))
	if err != nil {
		return domain.MasterDataPacket{}, err
	}
	return packageFromPublication(pub, nodeDeviceID)
}

// GetPublishedPackage возвращает конкретный package по publication id.
func (s *Service) GetPublishedPackage(ctx context.Context, restaurantID, packageID, nodeDeviceID string) (domain.MasterDataPacket, error) {
	pub, err := s.repo.GetPublication(ctx, strings.TrimSpace(restaurantID), strings.TrimSpace(packageID))
	if err != nil {
		return domain.MasterDataPacket{}, err
	}
	return packageFromPublication(pub, nodeDeviceID)
}

func (s *Service) ListCatalogSuggestions(ctx context.Context, restaurantID, status string, limit, offset int) ([]domain.CatalogSuggestion, error) {
	return s.repo.ListCatalogSuggestions(ctx, strings.TrimSpace(restaurantID), strings.TrimSpace(status), limit, offset)
}

func (s *Service) ListRecipeSuggestions(ctx context.Context, restaurantID, status string, limit, offset int) ([]domain.RecipeSuggestion, error) {
	return s.repo.ListRecipeSuggestions(ctx, strings.TrimSpace(restaurantID), strings.TrimSpace(status), limit, offset)
}

func (s *Service) ListStopListUpdateReviews(ctx context.Context, restaurantID, status string, limit, offset int) ([]domain.StopListUpdateReview, error) {
	return s.repo.ListStopListUpdateReviews(ctx, strings.TrimSpace(restaurantID), strings.TrimSpace(status), limit, offset)
}

func (s *Service) GetStopListUpdateReview(ctx context.Context, id string) (domain.StopListUpdateReview, error) {
	return s.repo.GetStopListUpdateReview(ctx, strings.TrimSpace(id))
}

// ListStopListUpdateReviewAudit возвращает bounded assignment audit для stop-list review item без raw payload.
func (s *Service) ListStopListUpdateReviewAudit(ctx context.Context, id string, limit, offset int) ([]ReviewAssignmentAuditEventResponse, error) {
	return s.listReviewAssignmentAudit(ctx, "stop_list_update", id, limit, offset)
}

// ListCatalogSuggestionReviewAudit возвращает bounded assignment audit для catalog suggestion review item без raw payload.
func (s *Service) ListCatalogSuggestionReviewAudit(ctx context.Context, id string, limit, offset int) ([]ReviewAssignmentAuditEventResponse, error) {
	return s.listReviewAssignmentAudit(ctx, "catalog_suggestion", id, limit, offset)
}

// ListRecipeSuggestionReviewAudit возвращает bounded assignment audit для recipe suggestion review item без raw payload.
func (s *Service) ListRecipeSuggestionReviewAudit(ctx context.Context, id string, limit, offset int) ([]ReviewAssignmentAuditEventResponse, error) {
	return s.listReviewAssignmentAudit(ctx, "recipe_suggestion", id, limit, offset)
}

func (s *Service) listReviewAssignmentAudit(ctx context.Context, reviewType, id string, limit, offset int) ([]ReviewAssignmentAuditEventResponse, error) {
	events, err := s.repo.ListReviewAssignmentAuditEvents(ctx, reviewType, strings.TrimSpace(id), normalizeAuditLimit(limit), normalizeAuditOffset(offset))
	if err != nil {
		return nil, err
	}
	out := make([]ReviewAssignmentAuditEventResponse, 0, len(events))
	for _, event := range events {
		out = append(out, reviewAssignmentAuditEventResponse(event))
	}
	return out, nil
}

func (s *Service) AssignReviewItem(ctx context.Context, reviewType, id string, cmd ReviewAssignCommand) (ReviewAssignmentResponse, error) {
	reviewType = strings.TrimSpace(reviewType)
	id = strings.TrimSpace(id)
	cmd.CommandID = strings.TrimSpace(cmd.CommandID)
	cmd.AssignedToEmployeeID = strings.TrimSpace(cmd.AssignedToEmployeeID)
	cmd.AssignedByEmployeeID = strings.TrimSpace(cmd.AssignedByEmployeeID)
	cmd.Reason = strings.TrimSpace(cmd.Reason)
	if replay, ok, err := s.replayReviewAssignment(ctx, reviewType, id, cmd.CommandID, "assigned"); err != nil || ok {
		return replay, err
	}
	if err := validateReviewAssignCommand(cmd); err != nil {
		return ReviewAssignmentResponse{}, err
	}
	if reviewType != "stop_list_update" {
		return ReviewAssignmentResponse{}, fmt.Errorf("%w: only stop_list_update assignment is supported", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	v, err := s.repo.GetStopListUpdateReview(ctx, id)
	if err != nil {
		return ReviewAssignmentResponse{}, err
	}
	if isTerminalSuggestionStatus(v.Status) {
		return ReviewAssignmentResponse{}, fmt.Errorf("%w: terminal review item cannot be assigned", domain.ErrConflict)
	}
	v.AssignedToEmployeeID = cmd.AssignedToEmployeeID
	v.AssignedByEmployeeID = cmd.AssignedByEmployeeID
	v.AssignedAt = &now
	v.AssignmentNote = cmd.Reason
	v.UpdatedAt = now
	stored, err := s.repo.UpdateStopListUpdateReview(ctx, v)
	if err != nil {
		return ReviewAssignmentResponse{}, err
	}
	if err := s.appendReviewAssignmentAudit(ctx, cmd.CommandID, reviewType, id, "assigned", cmd.AssignedToEmployeeID, cmd.AssignedByEmployeeID, cmd.Reason, now); err != nil {
		return ReviewAssignmentResponse{}, err
	}
	return stopListAssignmentResponse(reviewType, stored), nil
}

func (s *Service) UnassignReviewItem(ctx context.Context, reviewType, id string, cmd ReviewUnassignCommand) (ReviewAssignmentResponse, error) {
	reviewType = strings.TrimSpace(reviewType)
	id = strings.TrimSpace(id)
	cmd.CommandID = strings.TrimSpace(cmd.CommandID)
	cmd.UnassignedByEmployeeID = strings.TrimSpace(cmd.UnassignedByEmployeeID)
	cmd.Reason = strings.TrimSpace(cmd.Reason)
	if replay, ok, err := s.replayReviewAssignment(ctx, reviewType, id, cmd.CommandID, "unassigned"); err != nil || ok {
		return replay, err
	}
	if err := validateReviewUnassignCommand(cmd); err != nil {
		return ReviewAssignmentResponse{}, err
	}
	if reviewType != "stop_list_update" {
		return ReviewAssignmentResponse{}, fmt.Errorf("%w: only stop_list_update assignment is supported", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	v, err := s.repo.GetStopListUpdateReview(ctx, id)
	if err != nil {
		return ReviewAssignmentResponse{}, err
	}
	if isTerminalSuggestionStatus(v.Status) {
		return ReviewAssignmentResponse{}, fmt.Errorf("%w: terminal review item cannot be unassigned", domain.ErrConflict)
	}
	previous := v.AssignedToEmployeeID
	v.AssignedToEmployeeID = ""
	v.AssignedByEmployeeID = ""
	v.AssignedAt = nil
	v.AssignmentNote = ""
	v.UpdatedAt = now
	stored, err := s.repo.UpdateStopListUpdateReview(ctx, v)
	if err != nil {
		return ReviewAssignmentResponse{}, err
	}
	if err := s.appendReviewAssignmentAudit(ctx, cmd.CommandID, reviewType, id, "unassigned", previous, cmd.UnassignedByEmployeeID, cmd.Reason, now); err != nil {
		return ReviewAssignmentResponse{}, err
	}
	return stopListAssignmentResponse(reviewType, stored), nil
}

func (s *Service) ApproveCatalogSuggestion(ctx context.Context, id string, cmd SuggestionReviewCommand) (domain.CatalogSuggestion, error) {
	v, err := s.repo.GetCatalogSuggestion(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.CatalogSuggestion{}, err
	}
	now := s.clock.Now().UTC()
	if err := s.applyCatalogSuggestion(ctx, &v, now); err != nil {
		return domain.CatalogSuggestion{}, err
	}
	v.Status = domain.SuggestionStatusApproved
	v.ReviewComment = strings.TrimSpace(cmd.ReviewComment)
	v.ReviewedByEmployeeID = strings.TrimSpace(cmd.ReviewedByEmployeeID)
	v.ReviewedAt = &now
	v.UpdatedAt = now
	stored, err := s.repo.UpdateCatalogSuggestion(ctx, v)
	if err != nil {
		return domain.CatalogSuggestion{}, err
	}
	if _, err := s.RefreshDeliveryPackages(ctx, v.RestaurantID); err != nil {
		return domain.CatalogSuggestion{}, err
	}
	return stored, nil
}

func (s *Service) RejectCatalogSuggestion(ctx context.Context, id string, cmd SuggestionReviewCommand) (domain.CatalogSuggestion, error) {
	return s.reviewCatalogSuggestion(ctx, id, cmd, domain.SuggestionStatusRejected)
}

func (s *Service) RequestChangesCatalogSuggestion(ctx context.Context, id string, cmd SuggestionReviewCommand) (domain.CatalogSuggestion, error) {
	return s.reviewCatalogSuggestion(ctx, id, cmd, domain.SuggestionStatusChangesRequest)
}

func (s *Service) ApproveRecipeSuggestion(ctx context.Context, id string, cmd SuggestionReviewCommand) (domain.RecipeSuggestion, error) {
	v, err := s.repo.GetRecipeSuggestion(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.RecipeSuggestion{}, err
	}
	now := s.clock.Now().UTC()
	v.Status = domain.SuggestionStatusApproved
	v.ReviewComment = strings.TrimSpace(cmd.ReviewComment)
	v.ReviewedByEmployeeID = strings.TrimSpace(cmd.ReviewedByEmployeeID)
	v.ReviewedAt = &now
	v.UpdatedAt = now
	if err := s.applyRecipeSuggestion(ctx, &v, now); err != nil {
		return domain.RecipeSuggestion{}, err
	}
	stored, err := s.repo.UpdateRecipeSuggestion(ctx, v)
	if err != nil {
		return domain.RecipeSuggestion{}, err
	}
	if _, err := s.RefreshDeliveryPackages(ctx, v.RestaurantID); err != nil {
		return domain.RecipeSuggestion{}, err
	}
	return stored, nil
}

func (s *Service) RejectRecipeSuggestion(ctx context.Context, id string, cmd SuggestionReviewCommand) (domain.RecipeSuggestion, error) {
	return s.reviewRecipeSuggestion(ctx, id, cmd, domain.SuggestionStatusRejected)
}

func (s *Service) RequestChangesRecipeSuggestion(ctx context.Context, id string, cmd SuggestionReviewCommand) (domain.RecipeSuggestion, error) {
	return s.reviewRecipeSuggestion(ctx, id, cmd, domain.SuggestionStatusChangesRequest)
}

func (s *Service) ApproveStopListUpdateReview(ctx context.Context, id string, cmd SuggestionReviewCommand) (domain.StopListUpdateReview, error) {
	v, err := s.repo.GetStopListUpdateReview(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.StopListUpdateReview{}, err
	}
	if v.Status == domain.SuggestionStatusApproved {
		return v, nil
	}
	if v.Status != domain.SuggestionStatusPending {
		return domain.StopListUpdateReview{}, fmt.Errorf("%w: stop-list update is already reviewed", domain.ErrConflict)
	}
	now := s.clock.Now().UTC()
	stopListID := strings.TrimSpace(v.StopListID)
	if stopListID == "" {
		stopListID = s.ids.NewID()
	}
	entry, err := s.repo.UpsertStopListEntry(ctx, domain.StopListEntry{
		ID:                stopListID,
		RestaurantID:      v.RestaurantID,
		CatalogItemID:     v.CatalogItemID,
		AvailableQuantity: v.AvailableQuantity,
		Source:            "edge_review",
		Reason:            v.Reason,
		Active:            v.Active,
		UpdatedAt:         now,
	})
	if err != nil {
		return domain.StopListUpdateReview{}, err
	}
	v.Status = domain.SuggestionStatusApproved
	v.ReviewComment = strings.TrimSpace(cmd.ReviewComment)
	v.ReviewedByEmployeeID = strings.TrimSpace(cmd.ReviewedByEmployeeID)
	v.ReviewedAt = &now
	v.AppliedStopListID = entry.ID
	stored, err := s.repo.UpdateStopListUpdateReview(ctx, v)
	if err != nil {
		return domain.StopListUpdateReview{}, err
	}
	if _, err := s.RefreshDeliveryPackages(ctx, v.RestaurantID); err != nil {
		return domain.StopListUpdateReview{}, err
	}
	return stored, nil
}

func (s *Service) RejectStopListUpdateReview(ctx context.Context, id string, cmd SuggestionReviewCommand) (domain.StopListUpdateReview, error) {
	return s.reviewStopListUpdate(ctx, id, cmd, domain.SuggestionStatusRejected)
}

func (s *Service) RequestChangesStopListUpdateReview(ctx context.Context, id string, cmd SuggestionReviewCommand) (domain.StopListUpdateReview, error) {
	return s.reviewStopListUpdate(ctx, id, cmd, domain.SuggestionStatusChangesRequest)
}

func (s *Service) buildPacket(ctx context.Context, restaurantID, nodeDeviceID string, version int64, now time.Time) (domain.MasterDataPacket, map[string]int, []StreamPackage, error) {
	restaurant, err := s.repo.GetRestaurant(ctx, restaurantID)
	if err != nil && !errorsIsNotFound(err) {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	employees, err := s.repo.ListEmployees(ctx)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	roleByID := make(map[string]domain.Role, len(roles))
	for _, role := range roles {
		roleByID[role.ID] = role
	}
	eligible := employees[:0]
	eligibleRoleIDs := map[string]struct{}{}
	for _, employee := range employees {
		role, ok := roleByID[employee.RoleID]
		if !ok || !role.Active || !employee.ActiveForPOS() || (!role.ManagesOrganization() && !slices.Contains(employee.RestaurantIDs, restaurantID)) {
			continue
		}
		eligible = append(eligible, employee)
		eligibleRoleIDs[role.ID] = struct{}{}
	}
	employees = eligible
	eligibleRoles := roles[:0]
	for _, role := range roles {
		if _, ok := eligibleRoleIDs[role.ID]; ok {
			eligibleRoles = append(eligibleRoles, role)
		}
	}
	roles = eligibleRoles
	halls, err := s.repo.ListHalls(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	tables, err := s.repo.ListTables(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	catalogItems, err := s.repo.ListCatalogItems(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	folders, err := s.repo.ListCatalogFolders(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	folderParameters, err := s.repo.ListFolderParameters(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	tags, err := s.repo.ListCatalogTags(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	itemTags, err := s.repo.ListCatalogItemTags(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	modifierGroups, err := s.repo.ListModifierGroups(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	modifierOptions, err := s.repo.ListModifierOptions(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	modifierBindings, err := s.repo.ListModifierGroupBindings(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	pricingPolicies, err := s.repo.ListPricingPolicies(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	menuItems, err := s.repo.ListMenuItems(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	recipeItems, err := s.repo.ListRecipeItems(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	recipeAuthorityVersions, err := s.repo.ListRecipeVersions(ctx, restaurantID, "", "", 500, 0)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	recipeAuthorityLines := map[string][]domain.RecipeLine{}
	for _, version := range recipeAuthorityVersions {
		lines, err := s.repo.ListRecipeLines(ctx, version.ID)
		if err != nil {
			return domain.MasterDataPacket{}, nil, nil, err
		}
		recipeAuthorityLines[version.ID] = lines
	}
	stopLists, err := s.repo.ListStopListEntries(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	itemTags = filterItemTagsForCatalog(itemTags, catalogItems)
	sortRoles(roles)
	sortEmployees(employees)
	sortCatalog(catalogItems)
	sortMenu(menuItems)
	sortHalls(halls)
	sortTables(tables)
	sortRecipeItems(recipeItems)
	sortStopLists(stopLists)
	recipeVersions, recipeLines := edgeRecipeVersions(recipeAuthorityVersions, recipeAuthorityLines)
	versionedOwners := map[string]struct{}{}
	for _, version := range recipeVersions {
		versionedOwners[version.DishCatalogItemID] = struct{}{}
	}
	legacyRecipeItems := make([]domain.RecipeItem, 0, len(recipeItems))
	for _, item := range recipeItems {
		if _, ok := versionedOwners[item.RecipeOwnerCatalogItemID]; !ok {
			legacyRecipeItems = append(legacyRecipeItems, item)
		}
	}
	legacyVersions, legacyLines := edgeRecipes(legacyRecipeItems, catalogItems)
	if len(legacyVersions) > 0 {
		recipeVersions = append(recipeVersions, legacyVersions...)
		recipeLines = append(recipeLines, legacyLines...)
	}
	sort.SliceStable(recipeVersions, func(i, j int) bool { return recipeVersions[i].ID < recipeVersions[j].ID })
	sort.SliceStable(recipeLines, func(i, j int) bool { return recipeLines[i].ID < recipeLines[j].ID })

	restaurants := []domain.Restaurant{}
	if restaurant.ID != "" && restaurant.Status == domain.RestaurantActive {
		restaurants = append(restaurants, restaurant)
	}
	packet := domain.MasterDataPacket{
		NodeDeviceID:           nodeDeviceID,
		RestaurantID:           restaurantID,
		SyncMode:               "incremental",
		CheckpointToken:        fmt.Sprintf("master-data:%s:%d", restaurantID, version),
		CloudVersion:           version,
		CloudUpdatedAt:         now,
		Restaurants:            edgeRestaurants(restaurants),
		Roles:                  edgeRoles(roles),
		Employees:              edgeEmployees(employees, restaurantID),
		CatalogItems:           edgeCatalogItems(catalogItems),
		Folders:                edgeFolders(folders),
		FolderParameters:       edgeFolderParameters(folderParameters),
		Tags:                   edgeTags(tags),
		ItemTags:               edgeItemTags(itemTags),
		ModifierGroups:         edgeModifierGroups(modifierGroups),
		ModifierOptions:        edgeModifierOptions(modifierOptions),
		ModifierBindings:       edgeModifierBindings(modifierBindings),
		MenuItemModifierGroups: edgeMenuItemModifierGroups(menuItems, catalogItems, itemTags, modifierGroups, modifierBindings),
		MenuItems:              edgeMenuItems(menuItems),
		Halls:                  edgeHalls(halls),
		Tables:                 edgeTables(tables),
		PricingPolicies:        edgePricingPolicies(pricingPolicies),
		RecipeVersions:         recipeVersions,
		RecipeLines:            recipeLines,
		StopLists:              edgeStopLists(stopLists),
		Warehouses:             edgeDefaultWarehouses(restaurantID, now),
	}
	counts := map[string]int{
		"restaurants":               len(packet.Restaurants),
		"roles":                     len(packet.Roles),
		"employees":                 len(packet.Employees),
		"catalog_items":             len(packet.CatalogItems),
		"folders":                   len(packet.Folders),
		"folder_parameters":         len(packet.FolderParameters),
		"tags":                      len(packet.Tags),
		"item_tags":                 len(packet.ItemTags),
		"modifier_groups":           len(packet.ModifierGroups),
		"modifier_options":          len(packet.ModifierOptions),
		"modifier_bindings":         len(packet.ModifierBindings),
		"menu_item_modifier_groups": len(packet.MenuItemModifierGroups),
		"pricing_policies":          len(packet.PricingPolicies),
		"menu_items":                len(packet.MenuItems),
		"halls":                     len(packet.Halls),
		"tables":                    len(packet.Tables),
		"recipe_versions":           len(packet.RecipeVersions),
		"recipe_lines":              len(packet.RecipeLines),
		"stop_lists":                len(packet.StopLists),
		"warehouses":                len(packet.Warehouses),
	}
	streams, err := streamPackages(packet)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	feedback, err := s.proposalFeedbackStream(ctx, restaurantID, packet.NodeDeviceID, packet.SyncMode, packet.CheckpointToken, packet.CloudVersion, packet.CloudUpdatedAt)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	streams = append(streams, feedback)
	return packet, counts, streams, nil
}

func (s *Service) proposalFeedbackStream(ctx context.Context, restaurantID, nodeDeviceID, syncMode, checkpoint string, cloudVersion int64, updatedAt time.Time) (StreamPackage, error) {
	catalog, err := s.repo.ListCatalogSuggestions(ctx, restaurantID, "", 200, 0)
	if err != nil && !errorsIsNotFound(err) {
		return StreamPackage{}, err
	}
	recipes, err := s.repo.ListRecipeSuggestions(ctx, restaurantID, "", 200, 0)
	if err != nil && !errorsIsNotFound(err) {
		return StreamPackage{}, err
	}
	body, err := json.Marshal(map[string]any{
		"node_device_id":      nodeDeviceID,
		"restaurant_id":       restaurantID,
		"sync_mode":           syncMode,
		"checkpoint_token":    checkpoint,
		"cloud_version":       cloudVersion,
		"cloud_updated_at":    updatedAt,
		"catalog_suggestions": catalog,
		"recipe_suggestions":  recipes,
	})
	if err != nil {
		return StreamPackage{}, err
	}
	return StreamPackage{
		StreamName:      "proposal_feedback",
		NodeDeviceID:    nodeDeviceID,
		RestaurantID:    restaurantID,
		SyncMode:        syncMode,
		CloudVersion:    cloudVersion,
		CheckpointToken: checkpoint,
		CloudUpdatedAt:  updatedAt,
		PayloadJSON:     body,
	}, nil
}

func streamPackages(packet domain.MasterDataPacket) ([]StreamPackage, error) {
	type restaurantsPayload struct {
		NodeDeviceID    string                  `json:"node_device_id,omitempty"`
		RestaurantID    string                  `json:"restaurant_id"`
		SyncMode        string                  `json:"sync_mode"`
		CheckpointToken string                  `json:"checkpoint_token,omitempty"`
		CloudVersion    int64                   `json:"cloud_version"`
		CloudUpdatedAt  time.Time               `json:"cloud_updated_at"`
		Restaurants     []domain.EdgeRestaurant `json:"restaurants"`
	}
	type staffPayload struct {
		NodeDeviceID    string                `json:"node_device_id,omitempty"`
		RestaurantID    string                `json:"restaurant_id"`
		SyncMode        string                `json:"sync_mode"`
		CheckpointToken string                `json:"checkpoint_token,omitempty"`
		CloudVersion    int64                 `json:"cloud_version"`
		CloudUpdatedAt  time.Time             `json:"cloud_updated_at"`
		Roles           []domain.EdgeRole     `json:"roles"`
		Employees       []domain.EdgeEmployee `json:"employees"`
	}
	type catalogPayload struct {
		NodeDeviceID     string                            `json:"node_device_id,omitempty"`
		RestaurantID     string                            `json:"restaurant_id"`
		SyncMode         string                            `json:"sync_mode"`
		CheckpointToken  string                            `json:"checkpoint_token,omitempty"`
		CloudVersion     int64                             `json:"cloud_version"`
		CloudUpdatedAt   time.Time                         `json:"cloud_updated_at"`
		CatalogItems     []domain.EdgeCatalogItem          `json:"catalog_items"`
		Folders          []domain.EdgeCatalogFolder        `json:"folders,omitempty"`
		FolderParameters []domain.EdgeFolderParameter      `json:"folder_parameters,omitempty"`
		Tags             []domain.EdgeCatalogTag           `json:"tags,omitempty"`
		ItemTags         []domain.EdgeCatalogItemTag       `json:"item_tags,omitempty"`
		ModifierGroups   []domain.EdgeModifierGroup        `json:"modifier_groups,omitempty"`
		ModifierOptions  []domain.EdgeModifierOption       `json:"modifier_options,omitempty"`
		ModifierBindings []domain.EdgeModifierGroupBinding `json:"modifier_bindings,omitempty"`
	}
	type floorPayload struct {
		NodeDeviceID    string             `json:"node_device_id,omitempty"`
		RestaurantID    string             `json:"restaurant_id"`
		SyncMode        string             `json:"sync_mode"`
		CheckpointToken string             `json:"checkpoint_token,omitempty"`
		CloudVersion    int64              `json:"cloud_version"`
		CloudUpdatedAt  time.Time          `json:"cloud_updated_at"`
		Halls           []domain.EdgeHall  `json:"halls"`
		Tables          []domain.EdgeTable `json:"tables"`
	}
	type menuPayload struct {
		NodeDeviceID           string                             `json:"node_device_id,omitempty"`
		RestaurantID           string                             `json:"restaurant_id"`
		SyncMode               string                             `json:"sync_mode"`
		CheckpointToken        string                             `json:"checkpoint_token,omitempty"`
		CloudVersion           int64                              `json:"cloud_version"`
		CloudUpdatedAt         time.Time                          `json:"cloud_updated_at"`
		MenuItems              []domain.EdgeMenuItem              `json:"menu_items"`
		MenuItemModifierGroups []domain.EdgeMenuItemModifierGroup `json:"menu_item_modifier_groups,omitempty"`
	}
	type pricingPayload struct {
		NodeDeviceID       string                     `json:"node_device_id,omitempty"`
		RestaurantID       string                     `json:"restaurant_id"`
		SyncMode           string                     `json:"sync_mode"`
		CheckpointToken    string                     `json:"checkpoint_token,omitempty"`
		CloudVersion       int64                      `json:"cloud_version"`
		CloudUpdatedAt     time.Time                  `json:"cloud_updated_at"`
		TaxProfiles        []json.RawMessage          `json:"tax_profiles"`
		TaxRules           []json.RawMessage          `json:"tax_rules"`
		ServiceChargeRules []json.RawMessage          `json:"service_charge_rules"`
		PricingPolicies    []domain.EdgePricingPolicy `json:"pricing_policies"`
	}
	type recipesPayload struct {
		NodeDeviceID    string                     `json:"node_device_id,omitempty"`
		RestaurantID    string                     `json:"restaurant_id"`
		SyncMode        string                     `json:"sync_mode"`
		CheckpointToken string                     `json:"checkpoint_token,omitempty"`
		CloudVersion    int64                      `json:"cloud_version"`
		CloudUpdatedAt  time.Time                  `json:"cloud_updated_at"`
		RecipeVersions  []domain.EdgeRecipeVersion `json:"recipe_versions"`
		RecipeLines     []domain.EdgeRecipeLine    `json:"recipe_lines"`
	}
	type inventoryPayload struct {
		NodeDeviceID    string                          `json:"node_device_id,omitempty"`
		RestaurantID    string                          `json:"restaurant_id"`
		SyncMode        string                          `json:"sync_mode"`
		CheckpointToken string                          `json:"checkpoint_token,omitempty"`
		CloudVersion    int64                           `json:"cloud_version"`
		CloudUpdatedAt  time.Time                       `json:"cloud_updated_at"`
		StopLists       []domain.EdgeStopListEntry      `json:"stop_lists"`
		Warehouses      []domain.EdgeWarehouseReference `json:"warehouses"`
	}
	build := func(stream string, payload any) (StreamPackage, error) {
		body, err := json.Marshal(payload)
		if err != nil {
			return StreamPackage{}, err
		}
		return StreamPackage{
			StreamName:      stream,
			NodeDeviceID:    packet.NodeDeviceID,
			RestaurantID:    packet.RestaurantID,
			SyncMode:        packet.SyncMode,
			CloudVersion:    packet.CloudVersion,
			CheckpointToken: packet.CheckpointToken,
			CloudUpdatedAt:  packet.CloudUpdatedAt,
			PayloadJSON:     body,
		}, nil
	}
	restaurants, err := build("restaurants", restaurantsPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, Restaurants: packet.Restaurants})
	if err != nil {
		return nil, err
	}
	staff, err := build("staff", staffPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, Roles: packet.Roles, Employees: packet.Employees})
	if err != nil {
		return nil, err
	}
	catalog, err := build("catalog", catalogPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, CatalogItems: packet.CatalogItems, Folders: packet.Folders, FolderParameters: packet.FolderParameters, Tags: packet.Tags, ItemTags: packet.ItemTags, ModifierGroups: packet.ModifierGroups, ModifierOptions: packet.ModifierOptions, ModifierBindings: packet.ModifierBindings})
	if err != nil {
		return nil, err
	}
	floor, err := build("floor", floorPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, Halls: packet.Halls, Tables: packet.Tables})
	if err != nil {
		return nil, err
	}
	menu, err := build("menu", menuPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, MenuItems: packet.MenuItems, MenuItemModifierGroups: packet.MenuItemModifierGroups})
	if err != nil {
		return nil, err
	}
	pricing, err := build("pricing_policy", pricingPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, TaxProfiles: []json.RawMessage{}, TaxRules: []json.RawMessage{}, ServiceChargeRules: []json.RawMessage{}, PricingPolicies: packet.PricingPolicies})
	if err != nil {
		return nil, err
	}
	recipes, err := build("recipes", recipesPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, RecipeVersions: packet.RecipeVersions, RecipeLines: packet.RecipeLines})
	if err != nil {
		return nil, err
	}
	inventory, err := build("inventory_reference", inventoryPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, StopLists: packet.StopLists, Warehouses: packet.Warehouses})
	if err != nil {
		return nil, err
	}
	return []StreamPackage{restaurants, staff, catalog, floor, menu, pricing, recipes, inventory}, nil
}

func edgeRestaurants(items []domain.Restaurant) []domain.EdgeRestaurant {
	out := make([]domain.EdgeRestaurant, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeRestaurant{
			ID:                           item.ID,
			Name:                         item.Name,
			Timezone:                     item.Timezone,
			Currency:                     item.Currency,
			BusinessDayMode:              item.BusinessDayMode,
			BusinessDayBoundaryLocalTime: item.BusinessDayBoundaryLocalTime,
			Active:                       item.Status == domain.RestaurantActive,
			CreatedAt:                    item.CreatedAt,
			UpdatedAt:                    item.UpdatedAt,
		})
	}
	return out
}

func edgeRoles(items []domain.Role) []domain.EdgeRole {
	out := make([]domain.EdgeRole, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeRole{ID: item.ID, Name: item.Name, PermissionsJSON: item.PermissionsJSON, Active: item.Active, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeEmployees(items []domain.Employee, restaurantID string) []domain.EdgeEmployee {
	out := make([]domain.EdgeEmployee, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeEmployee{ID: item.ID, RestaurantID: restaurantID, RoleID: item.RoleID, Name: item.Name, PINHash: item.PINHash, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeCatalogItems(items []domain.CatalogItem) []domain.EdgeCatalogItem {
	out := make([]domain.EdgeCatalogItem, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeCatalogItem{ID: item.ID, Type: item.EdgeType(), FolderID: item.FolderID, Name: item.Name, SKU: item.SKU, BaseUnit: item.BaseUnit, KitchenType: item.KitchenType, AccountingCategory: item.AccountingCategory, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeFolders(items []domain.CatalogFolder) []domain.EdgeCatalogFolder {
	out := make([]domain.EdgeCatalogFolder, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeCatalogFolder{ID: item.ID, RestaurantID: item.RestaurantID, ParentID: item.ParentID, Name: item.Name, SortOrder: item.SortOrder, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeFolderParameters(items []domain.FolderParameter) []domain.EdgeFolderParameter {
	out := make([]domain.EdgeFolderParameter, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeFolderParameter{ID: item.ID, RestaurantID: item.RestaurantID, FolderID: item.FolderID, Key: item.Key, ValueType: item.ValueType, ValueJSON: item.ValueJSON, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeTags(items []domain.CatalogTag) []domain.EdgeCatalogTag {
	out := make([]domain.EdgeCatalogTag, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeCatalogTag{ID: item.ID, RestaurantID: item.RestaurantID, Name: item.Name, Code: item.Code, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeItemTags(items []domain.CatalogItemTag) []domain.EdgeCatalogItemTag {
	out := make([]domain.EdgeCatalogItemTag, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeCatalogItemTag{CatalogItemID: item.CatalogItemID, TagID: item.TagID, RestaurantID: item.RestaurantID})
	}
	return out
}

func edgeModifierGroups(items []domain.ModifierGroup) []domain.EdgeModifierGroup {
	out := make([]domain.EdgeModifierGroup, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeModifierGroup{ID: item.ID, RestaurantID: item.RestaurantID, Name: item.Name, Required: item.Required, MinCount: item.MinCount, MaxCount: item.MaxCount, Active: item.ActiveForPOS()})
	}
	return out
}

func edgeModifierOptions(items []domain.ModifierOption) []domain.EdgeModifierOption {
	out := make([]domain.EdgeModifierOption, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeModifierOption{ID: item.ID, RestaurantID: item.RestaurantID, ModifierGroupID: item.ModifierGroupID, LinkedCatalogItemID: item.LinkedCatalogItemID, Name: item.Name, PriceMinor: item.PriceMinor, Active: item.ActiveForPOS()})
	}
	return out
}

func edgeModifierBindings(items []domain.ModifierGroupBinding) []domain.EdgeModifierGroupBinding {
	out := make([]domain.EdgeModifierGroupBinding, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeModifierGroupBinding{ID: item.ID, RestaurantID: item.RestaurantID, ModifierGroupID: item.ModifierGroupID, TargetType: string(item.TargetType), TargetID: item.TargetID, SortOrder: item.SortOrder, Active: item.ActiveForPOS()})
	}
	return out
}

func edgeMenuItemModifierGroups(menuItems []domain.MenuItem, catalogItems []domain.CatalogItem, itemTags []domain.CatalogItemTag, groups []domain.ModifierGroup, bindings []domain.ModifierGroupBinding) []domain.EdgeMenuItemModifierGroup {
	catalogByID := map[string]domain.CatalogItem{}
	for _, item := range catalogItems {
		catalogByID[item.ID] = item
	}
	groupByID := map[string]domain.ModifierGroup{}
	for _, group := range groups {
		groupByID[group.ID] = group
	}
	tagsByItem := map[string]map[string]struct{}{}
	for _, link := range itemTags {
		if tagsByItem[link.CatalogItemID] == nil {
			tagsByItem[link.CatalogItemID] = map[string]struct{}{}
		}
		tagsByItem[link.CatalogItemID][link.TagID] = struct{}{}
	}
	seen := map[string]struct{}{}
	var out []domain.EdgeMenuItemModifierGroup
	for _, menuItem := range menuItems {
		catalog := catalogByID[menuItem.CatalogItemID]
		for _, binding := range bindings {
			if !binding.ActiveForPOS() {
				continue
			}
			group, ok := groupByID[binding.ModifierGroupID]
			if !ok || !group.ActiveForPOS() {
				continue
			}
			matches := false
			switch binding.TargetType {
			case domain.ModifierTargetMenuItem:
				matches = binding.TargetID == menuItem.ID
			case domain.ModifierTargetCatalogItem:
				matches = binding.TargetID == menuItem.CatalogItemID
			case domain.ModifierTargetFolder:
				matches = binding.TargetID != "" && binding.TargetID == catalog.FolderID
			case domain.ModifierTargetTag:
				_, matches = tagsByItem[menuItem.CatalogItemID][binding.TargetID]
			}
			key := menuItem.ID + "|" + binding.ModifierGroupID
			if matches {
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				out = append(out, domain.EdgeMenuItemModifierGroup{MenuItemID: menuItem.ID, ModifierGroupID: binding.ModifierGroupID, SortOrder: binding.SortOrder})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MenuItemID == out[j].MenuItemID {
			if out[i].SortOrder == out[j].SortOrder {
				return out[i].ModifierGroupID < out[j].ModifierGroupID
			}
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].MenuItemID < out[j].MenuItemID
	})
	return out
}

func edgePricingPolicies(items []domain.PricingPolicy) []domain.EdgePricingPolicy {
	out := make([]domain.EdgePricingPolicy, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgePricingPolicy{ID: item.ID, RestaurantID: item.RestaurantID, Name: item.Name, Kind: string(item.Kind), Scope: item.Scope, AmountKind: item.AmountKind, AmountMinor: item.AmountMinor, ValueBasisPoints: item.ValueBasisPoints, ApplicationIndex: item.ApplicationIndex, Manual: item.Manual, RequiresPermission: item.RequiresPermission, Active: item.ActiveForPOS()})
	}
	return out
}

func filterItemTagsForCatalog(itemTags []domain.CatalogItemTag, catalogItems []domain.CatalogItem) []domain.CatalogItemTag {
	allowed := map[string]struct{}{}
	for _, item := range catalogItems {
		allowed[item.ID] = struct{}{}
	}
	out := itemTags[:0]
	for _, link := range itemTags {
		if _, ok := allowed[link.CatalogItemID]; ok {
			out = append(out, link)
		}
	}
	return out
}

func edgeRecipes(items []domain.RecipeItem, catalogItems []domain.CatalogItem) ([]domain.EdgeRecipeVersion, []domain.EdgeRecipeLine) {
	catalogByID := map[string]domain.CatalogItem{}
	for _, item := range catalogItems {
		catalogByID[item.ID] = item
	}
	versionIDByOwner := map[string]string{}
	versions := make([]domain.EdgeRecipeVersion, 0)
	lines := make([]domain.EdgeRecipeLine, 0, len(items))
	for _, item := range items {
		owner := catalogByID[item.RecipeOwnerCatalogItemID]
		versionID := "recipe-" + item.RecipeOwnerCatalogItemID + "-v1"
		if _, ok := versionIDByOwner[item.RecipeOwnerCatalogItemID]; !ok {
			versionIDByOwner[item.RecipeOwnerCatalogItemID] = versionID
			name := owner.Name
			if strings.TrimSpace(name) == "" {
				name = item.RecipeOwnerCatalogItemID
			}
			yieldUnit := owner.BaseUnit
			if strings.TrimSpace(yieldUnit) == "" {
				yieldUnit = "portion"
			}
			active := owner.ActiveForPOS()
			status := "archived"
			if active {
				status = "active"
			}
			versions = append(versions, domain.EdgeRecipeVersion{
				ID:                versionID,
				DishCatalogItemID: item.RecipeOwnerCatalogItemID,
				Version:           1,
				Name:              name + " recipe",
				Status:            status,
				YieldQuantity:     1,
				YieldUnit:         yieldUnit,
				Active:            active,
				CreatedAt:         item.CreatedAt,
				UpdatedAt:         item.UpdatedAt,
			})
		}
		lines = append(lines, domain.EdgeRecipeLine{
			ID:              item.ID,
			RecipeVersionID: versionID,
			CatalogItemID:   item.ComponentCatalogItemID,
			Quantity:        item.Quantity,
			Unit:            item.Unit,
			LossPercent:     int(item.LossPercent),
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	sort.SliceStable(versions, func(i, j int) bool { return versions[i].ID < versions[j].ID })
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].ID < lines[j].ID })
	return versions, lines
}

func edgeRecipeVersions(items []domain.RecipeVersion, linesByVersion map[string][]domain.RecipeLine) ([]domain.EdgeRecipeVersion, []domain.EdgeRecipeLine) {
	versions := make([]domain.EdgeRecipeVersion, 0, len(items))
	lines := make([]domain.EdgeRecipeLine, 0)
	for _, item := range items {
		if item.Status != domain.RecipeVersionStatusActive && item.Status != domain.RecipeVersionStatusArchived {
			continue
		}
		active := item.Status == domain.RecipeVersionStatusActive
		status := "archived"
		if active {
			status = "active"
		}
		versions = append(versions, domain.EdgeRecipeVersion{
			ID:                item.ID,
			DishCatalogItemID: item.OwnerCatalogItemID,
			Version:           item.Version,
			Name:              item.Name,
			Status:            status,
			YieldQuantity:     item.YieldQuantity,
			YieldUnit:         item.YieldUnit,
			Active:            active,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		})
		for _, line := range linesByVersion[item.ID] {
			lines = append(lines, domain.EdgeRecipeLine{
				ID:              line.ID,
				RecipeVersionID: item.ID,
				CatalogItemID:   line.ComponentCatalogItemID,
				Quantity:        line.Quantity,
				Unit:            line.Unit,
				LossPercent:     int(line.LossPercent),
				CreatedAt:       line.CreatedAt,
				UpdatedAt:       line.UpdatedAt,
			})
		}
	}
	sort.SliceStable(versions, func(i, j int) bool { return versions[i].ID < versions[j].ID })
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].ID < lines[j].ID })
	return versions, lines
}

func edgeStopLists(items []domain.StopListEntry) []domain.EdgeStopListEntry {
	out := make([]domain.EdgeStopListEntry, 0, len(items))
	for _, item := range items {
		source := strings.TrimSpace(item.Source)
		if source == "" {
			source = "cloud"
		}
		out = append(out, domain.EdgeStopListEntry{ID: item.ID, RestaurantID: item.RestaurantID, CatalogItemID: item.CatalogItemID, AvailableQuantity: item.AvailableQuantity, Source: source, Reason: item.Reason, Active: item.Active, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeDefaultWarehouses(restaurantID string, now time.Time) []domain.EdgeWarehouseReference {
	return []domain.EdgeWarehouseReference{{
		ID:           "warehouse-main",
		RestaurantID: restaurantID,
		Name:         "Main Kitchen Warehouse",
		Kind:         "kitchen",
		Default:      true,
		Active:       true,
		UpdatedAt:    now,
	}}
}

func edgeMenuItems(items []domain.MenuItem) []domain.EdgeMenuItem {
	out := make([]domain.EdgeMenuItem, 0, len(items))
	for _, item := range items {
		active := item.ActiveForPOS() && item.RuntimeStatus == "available"
		out = append(out, domain.EdgeMenuItem{
			ID:            item.ID,
			CatalogItemID: item.CatalogItemID,
			CategoryID:    item.CategoryID,
			TagID:         item.TagID,
			Name:          item.Name,
			Price:         item.Price,
			Currency:      item.Currency,
			TaxProfileID:  item.TaxProfileID,
			RuntimeStatus: item.RuntimeStatus,
			Active:        active,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}
	return out
}

func summarizePublication(pub domain.Publication, counts map[string]int) PublicationSummary {
	return PublicationSummary{
		ID:            pub.ID,
		RestaurantID:  pub.RestaurantID,
		Version:       pub.Version,
		Status:        string(pub.Status),
		CloudVersion:  pub.CloudVersion,
		PublishedAt:   pub.PublishedAt,
		PublishedBy:   pub.PublishedBy,
		PackageSHA256: pub.PackageSHA256,
		Counts:        counts,
	}
}

func validateCatalogFields(kind domain.CatalogItemKind, name, sku, baseUnit string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(sku) == "" || strings.TrimSpace(baseUnit) == "" {
		return fmt.Errorf("%w: name, sku and base_unit are required", domain.ErrInvalid)
	}
	return domain.ValidateCatalogItemKind(kind)
}

func (s *Service) ensureRestaurantActive(ctx context.Context, restaurantID string) error {
	restaurant, err := s.repo.GetRestaurant(ctx, strings.TrimSpace(restaurantID))
	if err != nil {
		return err
	}
	if restaurant.Status != domain.RestaurantActive {
		return fmt.Errorf("%w: restaurant must be active", domain.ErrInvalid)
	}
	return nil
}

func (s *Service) ensureCatalogItemInRestaurant(ctx context.Context, restaurantID, catalogItemID string) (domain.CatalogItem, error) {
	if strings.TrimSpace(restaurantID) == "" {
		return domain.CatalogItem{}, fmt.Errorf("%w: restaurant_id is required", domain.ErrInvalid)
	}
	return s.ensureCatalogItemAvailable(ctx, catalogItemID)
}

func (s *Service) ensureCatalogItemAvailable(ctx context.Context, catalogItemID string) (domain.CatalogItem, error) {
	item, err := s.repo.GetCatalogItem(ctx, strings.TrimSpace(catalogItemID))
	if err != nil {
		return domain.CatalogItem{}, err
	}
	if item.Status == domain.StatusArchived {
		return domain.CatalogItem{}, fmt.Errorf("%w: catalog item is archived", domain.ErrInvalid)
	}
	return item, nil
}

func (s *Service) ensureCatalogItemKind(ctx context.Context, restaurantID, catalogItemID string, allowed ...domain.CatalogItemKind) error {
	item, err := s.ensureCatalogItemInRestaurant(ctx, restaurantID, catalogItemID)
	if err != nil {
		return err
	}
	for _, kind := range allowed {
		if item.Kind == kind {
			return nil
		}
	}
	return fmt.Errorf("%w: catalog item kind is not valid for recipe role", domain.ErrInvalid)
}

func validatePolicyScope(kind domain.PricingPolicyKind, scope string) error {
	switch strings.TrimSpace(scope) {
	case "line", "order":
	default:
		return fmt.Errorf("%w: unsupported pricing policy scope", domain.ErrInvalid)
	}
	if kind == domain.PricingPolicySurcharge && strings.TrimSpace(scope) != "order" {
		return fmt.Errorf("%w: surcharge pricing policy must be order-scoped", domain.ErrInvalid)
	}
	return nil
}

func validatePolicyAmount(amountKind string, amountMinor, valueBasisPoints int64) error {
	switch strings.TrimSpace(amountKind) {
	case "fixed":
		if amountMinor < 0 || valueBasisPoints != 0 {
			return fmt.Errorf("%w: fixed pricing policy requires non-negative amount_minor only", domain.ErrInvalid)
		}
	case "percentage":
		if valueBasisPoints <= 0 || amountMinor != 0 {
			return fmt.Errorf("%w: percentage pricing policy requires positive value_basis_points only", domain.ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: amount_kind must be fixed or percentage", domain.ErrInvalid)
	}
	return nil
}

func isCurrencyCode(v string) bool {
	return isActiveCurrencyCode(v)
}

func isActiveCurrencyCode(v string) bool {
	v = strings.ToUpper(strings.TrimSpace(v))
	if len(v) != 3 {
		return false
	}
	for _, profile := range contracts.CanonicalActiveCurrencyProfiles() {
		if profile.CurrencyAlphaCode == v {
			return true
		}
	}
	return false
}

func normalizeBusinessDayConfig(mode, boundary string) (string, string, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "standard"
	}
	if mode != "standard" && mode != "24_7" {
		return "", "", fmt.Errorf("%w: business_day_mode must be standard or 24_7", domain.ErrInvalid)
	}
	boundary = strings.TrimSpace(boundary)
	if boundary == "" {
		boundary = "04:00"
	}
	if len(boundary) != 5 || boundary[2] != ':' {
		return "", "", fmt.Errorf("%w: business_day_boundary_local_time must be HH:MM", domain.ErrInvalid)
	}
	hour, err := strconv.Atoi(boundary[:2])
	if err != nil {
		return "", "", fmt.Errorf("%w: business_day_boundary_local_time must be HH:MM", domain.ErrInvalid)
	}
	minute, err := strconv.Atoi(boundary[3:])
	if err != nil {
		return "", "", fmt.Errorf("%w: business_day_boundary_local_time must be HH:MM", domain.ErrInvalid)
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return "", "", fmt.Errorf("%w: business_day_boundary_local_time must be valid HH:MM", domain.ErrInvalid)
	}
	return mode, boundary, nil
}

func normalizeAvailability(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}", nil
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", fmt.Errorf("%w: availability_json must be valid JSON", domain.ErrInvalid)
	}
	body, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func normalizeMenuRuntimeStatus(raw string) (string, error) {
	status := strings.TrimSpace(raw)
	if status == "" {
		return "available", nil
	}
	switch status {
	case "available", "unavailable", "hidden":
		return status, nil
	default:
		return "", fmt.Errorf("%w: runtime_status must be available, unavailable or hidden", domain.ErrInvalid)
	}
}

func canonicalJSON(raw string) string {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	body, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return string(body)
}

func hashPIN(pin string) (string, error) {
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return "", fmt.Errorf("%w: pin is required", domain.ErrInvalid)
	}
	var salt [16]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return "", err
	}
	key, err := pbkdf2.Key(sha256.New, pin, salt[:], pinHashIterations, pinHashKeyLength)
	if err != nil {
		return "", err
	}
	return strings.Join([]string{
		pinHashPrefix,
		pinHashVersion,
		strconv.Itoa(pinHashIterations),
		base64.RawStdEncoding.EncodeToString(salt[:]),
		base64.RawStdEncoding.EncodeToString(key),
	}, ":"), nil
}

func (s *Service) ensurePINUnique(ctx context.Context, exceptEmployeeID, pin string) error {
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return fmt.Errorf("%w: pin is required", domain.ErrInvalid)
	}
	employees, err := s.repo.ListEmployees(ctx)
	if err != nil {
		return err
	}
	for _, employee := range employees {
		if employee.ID == strings.TrimSpace(exceptEmployeeID) || employee.Status == domain.EmployeeArchived {
			continue
		}
		if verifyPIN(employee.PINHash, pin) == nil {
			return fmt.Errorf("%w: duplicate non-archived employee PIN in tenant", domain.ErrPINAlreadyExists)
		}
	}
	return nil
}

func (s *Service) normalizeEmployeeScope(ctx context.Context, role domain.Role, restaurantIDs []string) ([]string, bool, error) {
	if role.ManagesOrganization() {
		return []string{}, true, nil
	}
	seen := map[string]struct{}{}
	for _, id := range restaurantIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := s.ensureActiveRestaurant(ctx, id); err != nil {
			return nil, false, err
		}
		seen[id] = struct{}{}
	}
	if len(seen) == 0 {
		return nil, false, fmt.Errorf("%w: employee requires at least one restaurant membership", domain.ErrInvalid)
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	slices.Sort(out)
	return out, false, nil
}

func edgeHalls(items []domain.Hall) []domain.EdgeHall {
	out := make([]domain.EdgeHall, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeHall{ID: item.ID, RestaurantID: item.RestaurantID, Name: item.Name, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeTables(items []domain.Table) []domain.EdgeTable {
	out := make([]domain.EdgeTable, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeTable{ID: item.ID, RestaurantID: item.RestaurantID, HallID: item.HallID, Name: item.Name, Seats: item.Seats, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func (s *Service) ensureActiveRestaurant(ctx context.Context, restaurantID string) error {
	restaurant, err := s.repo.GetRestaurant(ctx, strings.TrimSpace(restaurantID))
	if err != nil {
		return err
	}
	if restaurant.Status != domain.RestaurantActive {
		return fmt.Errorf("%w: restaurant is archived", domain.ErrInvalid)
	}
	return nil
}

func (s *Service) validateLinkedModifierCatalogItem(ctx context.Context, restaurantID, catalogItemID string) (string, error) {
	catalogItemID = strings.TrimSpace(catalogItemID)
	if catalogItemID == "" {
		return "", nil
	}
	if strings.TrimSpace(restaurantID) == "" {
		return "", fmt.Errorf("%w: restaurant_id is required", domain.ErrInvalid)
	}
	if _, err := s.ensureCatalogItemAvailable(ctx, catalogItemID); err != nil {
		return "", err
	}
	return catalogItemID, nil
}

func verifyPIN(encoded, pin string) error {
	parts := strings.Split(strings.TrimSpace(encoded), ":")
	if len(parts) != 5 || parts[0] != pinHashPrefix || parts[1] != pinHashVersion {
		return fmt.Errorf("%w: unsupported pin hash", domain.ErrInvalid)
	}
	iterations, err := strconv.Atoi(parts[2])
	if err != nil || iterations <= 0 {
		return fmt.Errorf("%w: invalid pin hash iterations", domain.ErrInvalid)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return err
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return err
	}
	got, err := pbkdf2.Key(sha256.New, strings.TrimSpace(pin), salt, iterations, len(want))
	if err != nil {
		return err
	}
	if subtleCompare(got, want) {
		return nil
	}
	return domain.ErrInvalid
}

func subtleCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

func packageFromPublication(pub domain.Publication, nodeDeviceID string) (domain.MasterDataPacket, error) {
	var packet domain.MasterDataPacket
	if err := json.Unmarshal(pub.PackageJSON, &packet); err != nil {
		return domain.MasterDataPacket{}, fmt.Errorf("%w: invalid publication package", domain.ErrInvalid)
	}
	packet.NodeDeviceID = strings.TrimSpace(nodeDeviceID)
	return packet, nil
}

func errorsIsNotFound(err error) bool {
	return errors.Is(err, domain.ErrNotFound)
}

func validateReviewAssignCommand(cmd ReviewAssignCommand) error {
	if !isUUIDv7(cmd.CommandID) {
		return fmt.Errorf("%w: command_id must be uuidv7", domain.ErrInvalid)
	}
	if cmd.AssignedToEmployeeID == "" {
		return fmt.Errorf("%w: assigned_to_employee_id is required", domain.ErrInvalid)
	}
	if cmd.AssignedByEmployeeID == "" {
		return fmt.Errorf("%w: assigned_by_employee_id is required", domain.ErrInvalid)
	}
	if cmd.Reason == "" {
		return fmt.Errorf("%w: reason is required", domain.ErrInvalid)
	}
	if len(cmd.Reason) > 500 {
		return fmt.Errorf("%w: reason must be 500 characters or less", domain.ErrInvalid)
	}
	return nil
}

func validateReviewUnassignCommand(cmd ReviewUnassignCommand) error {
	if !isUUIDv7(cmd.CommandID) {
		return fmt.Errorf("%w: command_id must be uuidv7", domain.ErrInvalid)
	}
	if cmd.UnassignedByEmployeeID == "" {
		return fmt.Errorf("%w: unassigned_by_employee_id is required", domain.ErrInvalid)
	}
	if cmd.Reason == "" {
		return fmt.Errorf("%w: reason is required", domain.ErrInvalid)
	}
	if len(cmd.Reason) > 500 {
		return fmt.Errorf("%w: reason must be 500 characters or less", domain.ErrInvalid)
	}
	return nil
}

func (s *Service) replayReviewAssignment(ctx context.Context, reviewType, id, commandID, action string) (ReviewAssignmentResponse, bool, error) {
	if !isUUIDv7(commandID) {
		return ReviewAssignmentResponse{}, false, nil
	}
	event, err := s.repo.GetReviewAssignmentAuditEvent(ctx, commandID)
	if errors.Is(err, domain.ErrNotFound) {
		return ReviewAssignmentResponse{}, false, nil
	}
	if err != nil {
		return ReviewAssignmentResponse{}, false, err
	}
	if event.ReviewType != reviewType || event.ReviewID != id || event.Action != action {
		return ReviewAssignmentResponse{}, true, fmt.Errorf("%w: command_id belongs to another review assignment command", domain.ErrConflict)
	}
	response, err := s.reviewAssignmentResponse(ctx, reviewType, id)
	return response, true, err
}

func (s *Service) reviewAssignmentResponse(ctx context.Context, reviewType, id string) (ReviewAssignmentResponse, error) {
	switch reviewType {
	case "stop_list_update":
		v, err := s.repo.GetStopListUpdateReview(ctx, id)
		return stopListAssignmentResponse(reviewType, v), err
	default:
		return ReviewAssignmentResponse{}, fmt.Errorf("%w: unknown review_type", domain.ErrInvalid)
	}
}

func (s *Service) appendReviewAssignmentAudit(ctx context.Context, commandID, reviewType, reviewID, action, assignedToEmployeeID, actorEmployeeID, reason string, now time.Time) error {
	eventID, err := idgen.NewV7()
	if err != nil {
		return err
	}
	return s.repo.AppendReviewAssignmentAuditEvent(ctx, domain.ReviewAssignmentAuditEvent{
		EventID:          eventID,
		CommandID:        commandID,
		ReviewType:       reviewType,
		ReviewID:         reviewID,
		Action:           action,
		ActorEmployeeID:  actorEmployeeID,
		TargetEmployeeID: assignedToEmployeeID,
		Reason:           reason,
		OccurredAt:       now,
	})
}

func isTerminalSuggestionStatus(status domain.SuggestionStatus) bool {
	return status == domain.SuggestionStatusApproved || status == domain.SuggestionStatusRejected
}

func catalogAssignmentResponse(reviewType string, v domain.CatalogSuggestion) ReviewAssignmentResponse {
	return ReviewAssignmentResponse{
		ReviewType:           reviewType,
		ID:                   v.ID,
		Status:               string(v.Status),
		AssignedToEmployeeID: v.AssignedToEmployeeID,
		AssignedByEmployeeID: v.AssignedByEmployeeID,
		AssignedAt:           v.AssignedAt,
		AssignmentNote:       v.AssignmentNote,
	}
}

func recipeAssignmentResponse(reviewType string, v domain.RecipeSuggestion) ReviewAssignmentResponse {
	return ReviewAssignmentResponse{
		ReviewType:           reviewType,
		ID:                   v.ID,
		Status:               string(v.Status),
		AssignedToEmployeeID: v.AssignedToEmployeeID,
		AssignedByEmployeeID: v.AssignedByEmployeeID,
		AssignedAt:           v.AssignedAt,
		AssignmentNote:       v.AssignmentNote,
	}
}

func stopListAssignmentResponse(reviewType string, v domain.StopListUpdateReview) ReviewAssignmentResponse {
	return ReviewAssignmentResponse{
		ReviewType:           reviewType,
		ID:                   v.ID,
		Status:               string(v.Status),
		AssignedToEmployeeID: v.AssignedToEmployeeID,
		AssignedByEmployeeID: v.AssignedByEmployeeID,
		AssignedAt:           v.AssignedAt,
		AssignmentNote:       v.AssignmentNote,
	}
}

func reviewAssignmentAuditEventResponse(v domain.ReviewAssignmentAuditEvent) ReviewAssignmentAuditEventResponse {
	return ReviewAssignmentAuditEventResponse{
		EventID:          v.EventID,
		ReviewID:         v.ReviewID,
		ReviewType:       v.ReviewType,
		Action:           v.Action,
		ActorEmployeeID:  v.ActorEmployeeID,
		TargetEmployeeID: v.TargetEmployeeID,
		OccurredAt:       v.OccurredAt,
		Reason:           v.Reason,
		CommandID:        v.CommandID,
	}
}

func normalizeAuditLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func normalizeAuditOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func isUUIDv7(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, r := range v {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
				return false
			}
		}
	}
	if v[14] != '7' {
		return false
	}
	variant := v[19]
	return variant == '8' || variant == '9' || variant == 'a' || variant == 'A' || variant == 'b' || variant == 'B'
}

func (s *Service) reviewCatalogSuggestion(ctx context.Context, id string, cmd SuggestionReviewCommand, status domain.SuggestionStatus) (domain.CatalogSuggestion, error) {
	v, err := s.repo.GetCatalogSuggestion(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.CatalogSuggestion{}, err
	}
	now := s.clock.Now().UTC()
	v.Status = status
	v.ReviewComment = strings.TrimSpace(cmd.ReviewComment)
	v.ReviewedByEmployeeID = strings.TrimSpace(cmd.ReviewedByEmployeeID)
	v.ReviewedAt = &now
	v.UpdatedAt = now
	return s.repo.UpdateCatalogSuggestion(ctx, v)
}

func (s *Service) reviewRecipeSuggestion(ctx context.Context, id string, cmd SuggestionReviewCommand, status domain.SuggestionStatus) (domain.RecipeSuggestion, error) {
	v, err := s.repo.GetRecipeSuggestion(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.RecipeSuggestion{}, err
	}
	now := s.clock.Now().UTC()
	v.Status = status
	v.ReviewComment = strings.TrimSpace(cmd.ReviewComment)
	v.ReviewedByEmployeeID = strings.TrimSpace(cmd.ReviewedByEmployeeID)
	v.ReviewedAt = &now
	v.UpdatedAt = now
	return s.repo.UpdateRecipeSuggestion(ctx, v)
}

func (s *Service) reviewStopListUpdate(ctx context.Context, id string, cmd SuggestionReviewCommand, status domain.SuggestionStatus) (domain.StopListUpdateReview, error) {
	v, err := s.repo.GetStopListUpdateReview(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.StopListUpdateReview{}, err
	}
	if v.Status == status {
		return v, nil
	}
	if v.Status != domain.SuggestionStatusPending {
		return domain.StopListUpdateReview{}, fmt.Errorf("%w: stop-list update is already reviewed", domain.ErrConflict)
	}
	now := s.clock.Now().UTC()
	v.Status = status
	v.ReviewComment = strings.TrimSpace(cmd.ReviewComment)
	v.ReviewedByEmployeeID = strings.TrimSpace(cmd.ReviewedByEmployeeID)
	v.ReviewedAt = &now
	return s.repo.UpdateStopListUpdateReview(ctx, v)
}

func (s *Service) applyCatalogSuggestion(ctx context.Context, v *domain.CatalogSuggestion, now time.Time) error {
	var payload map[string]any
	_ = json.Unmarshal(v.PayloadJSON, &payload)
	data, _ := payload["data"].(map[string]any)
	action := strings.TrimSpace(v.Action)
	if action == "" {
		action = strings.TrimSpace(stringValue(data, "action"))
	}
	switch action {
	case "create", "create_item", "create_dish":
		item := domain.CatalogItem{
			ID:                 firstNonEmpty(stringValue(data, "catalog_item_id"), s.ids.NewID()),
			RestaurantID:       v.RestaurantID,
			Kind:               domain.CatalogItemDish,
			Name:               firstNonEmpty(stringValue(data, "name"), "Suggested item"),
			SKU:                firstNonEmpty(stringValue(data, "sku"), "SUGGESTED-"+v.SuggestionID),
			BaseUnit:           firstNonEmpty(stringValue(data, "base_unit"), "portion"),
			KitchenType:        strings.TrimSpace(stringValue(data, "kitchen_type")),
			AccountingCategory: strings.TrimSpace(stringValue(data, "accounting_category")),
			Status:             domain.StatusPublished,
			CloudVersion:       1,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if kind := strings.TrimSpace(stringValue(data, "kind")); kind != "" {
			item.Kind = domain.CatalogItemKind(kind)
		}
		created, err := s.repo.CreateCatalogItem(ctx, item)
		if err != nil {
			return err
		}
		v.AppliedCatalogItemID = created.ID
	default:
		targetID := firstNonEmpty(v.CatalogItemID, stringValue(data, "catalog_item_id"))
		if strings.TrimSpace(targetID) == "" {
			return fmt.Errorf("%w: catalog_item_id is required for update", domain.ErrInvalid)
		}
		item, err := s.repo.GetCatalogItem(ctx, targetID)
		if err != nil {
			return err
		}
		item.Name = firstNonEmpty(stringValue(data, "name"), item.Name)
		item.SKU = firstNonEmpty(stringValue(data, "sku"), item.SKU)
		item.BaseUnit = firstNonEmpty(stringValue(data, "base_unit"), item.BaseUnit)
		item.UpdatedAt = now
		item.CloudVersion++
		if _, err := s.repo.UpdateCatalogItem(ctx, item); err != nil {
			return err
		}
		v.AppliedCatalogItemID = item.ID
	}
	return nil
}

func (s *Service) applyRecipeSuggestion(ctx context.Context, v *domain.RecipeSuggestion, now time.Time) error {
	var payload map[string]any
	_ = json.Unmarshal(v.PayloadJSON, &payload)
	data, _ := payload["data"].(map[string]any)
	if strings.TrimSpace(v.Action) == "publish_recipe_version" || strings.TrimSpace(stringValue(data, "action")) == "publish_recipe_version" {
		versionID := strings.TrimSpace(firstNonEmpty(v.RecipeVersionID, stringValue(data, "recipe_version_id")))
		if versionID == "" {
			return fmt.Errorf("%w: recipe_version_id is required", domain.ErrInvalid)
		}
		approved, err := s.repo.ActivateRecipeVersion(ctx, versionID, strings.TrimSpace(v.ReviewedByEmployeeID), now)
		if err != nil {
			return err
		}
		v.OwnerCatalogItemID = approved.OwnerCatalogItemID
		return nil
	}
	ownerID := strings.TrimSpace(firstNonEmpty(v.OwnerCatalogItemID, stringValue(data, "owner_catalog_item_id")))
	if ownerID == "" {
		return fmt.Errorf("%w: owner_catalog_item_id is required", domain.ErrInvalid)
	}
	changes, _ := s.repo.ListRecipeSuggestionChanges(ctx, v.ID)
	for _, change := range changes {
		if strings.TrimSpace(change.ToCatalogItemID) == "" {
			continue
		}
		qty := int64(1)
		if parsed, err := strconv.ParseInt(strings.TrimSpace(change.Quantity), 10, 64); err == nil && parsed > 0 {
			qty = parsed
		}
		loss := int64(0)
		if parsed, err := strconv.ParseInt(strings.TrimSpace(change.LossPercent), 10, 64); err == nil && parsed >= 0 {
			loss = parsed
		}
		_, err := s.repo.CreateRecipeItem(ctx, domain.RecipeItem{
			ID:                       s.ids.NewID(),
			RestaurantID:             v.RestaurantID,
			RecipeOwnerCatalogItemID: ownerID,
			ComponentCatalogItemID:   change.ToCatalogItemID,
			Quantity:                 qty,
			Unit:                     firstNonEmpty(change.UnitCode, "unit"),
			LossPercent:              loss,
			CreatedAt:                now,
			UpdatedAt:                now,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) validateRecipeVersionLines(ctx context.Context, restaurantID, ownerID string, lines []RecipeVersionLineCommand) ([]domain.RecipeLine, error) {
	if strings.TrimSpace(restaurantID) == "" || strings.TrimSpace(ownerID) == "" {
		return nil, fmt.Errorf("%w: restaurant_id and owner_catalog_item_id are required", domain.ErrInvalid)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("%w: recipe version requires at least one line", domain.ErrInvalid)
	}
	seen := map[string]struct{}{}
	out := make([]domain.RecipeLine, 0, len(lines))
	for _, line := range lines {
		componentID := strings.TrimSpace(line.ComponentCatalogItemID)
		unit := strings.TrimSpace(line.Unit)
		if componentID == "" || line.Quantity <= 0 || unit == "" || line.LossPercent < 0 || line.LossPercent > 100 {
			return nil, fmt.Errorf("%w: recipe line requires component, positive quantity, unit and loss_percent 0..100", domain.ErrInvalid)
		}
		if componentID == strings.TrimSpace(ownerID) {
			return nil, fmt.Errorf("%w: recipe component cannot equal owner item", domain.ErrInvalid)
		}
		if _, ok := seen[componentID]; ok {
			return nil, fmt.Errorf("%w: duplicate recipe component", domain.ErrInvalid)
		}
		seen[componentID] = struct{}{}
		if err := s.ensureCatalogItemKind(ctx, restaurantID, componentID, domain.CatalogItemGood, domain.CatalogItemSemiFinished); err != nil {
			return nil, err
		}
		out = append(out, domain.RecipeLine{ComponentCatalogItemID: componentID, Quantity: line.Quantity, Unit: unit, LossPercent: line.LossPercent})
	}
	return out, nil
}

func recipeVersionChangesPayload(lines []domain.RecipeLine) []map[string]any {
	out := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		out = append(out, map[string]any{
			"line_id":            line.ID,
			"action":             "add_ingredient",
			"to_catalog_item_id": line.ComponentCatalogItemID,
			"quantity":           strconv.FormatInt(line.Quantity, 10),
			"unit_code":          line.Unit,
			"loss_percent":       strconv.FormatInt(line.LossPercent, 10),
		})
	}
	return out
}

func stringValue(v map[string]any, key string) string {
	raw, _ := v[key]
	s, _ := raw.(string)
	return strings.TrimSpace(s)
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func sortRoles(items []domain.Role) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}

func sortEmployees(items []domain.Employee) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}

func sortCatalog(items []domain.CatalogItem) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}

func sortMenu(items []domain.MenuItem) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}

func sortHalls(items []domain.Hall) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}

func sortTables(items []domain.Table) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].HallID == items[j].HallID {
			return items[i].ID < items[j].ID
		}
		return items[i].HallID < items[j].HallID
	})
}

func sortRecipeItems(items []domain.RecipeItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RecipeOwnerCatalogItemID == items[j].RecipeOwnerCatalogItemID {
			return items[i].ID < items[j].ID
		}
		return items[i].RecipeOwnerCatalogItemID < items[j].RecipeOwnerCatalogItemID
	})
}

func sortStopLists(items []domain.StopListEntry) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CatalogItemID == items[j].CatalogItemID {
			return items[i].ID < items[j].ID
		}
		return items[i].CatalogItemID < items[j].CatalogItemID
	})
}

// CloneStreamPackages возвращает копию списка stream packages для тестов и адаптеров.
func CloneStreamPackages(items []StreamPackage) []StreamPackage {
	return slices.Clone(items)
}
