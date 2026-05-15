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
	ListRoles(context.Context, string) ([]domain.Role, error)
	CreateEmployee(context.Context, domain.Employee) (domain.Employee, error)
	UpdateEmployee(context.Context, domain.Employee) (domain.Employee, error)
	GetEmployee(context.Context, string) (domain.Employee, error)
	ListEmployees(context.Context, string) ([]domain.Employee, error)
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
	NextPublicationVersion(context.Context, string) (int64, error)
	SavePublication(context.Context, domain.Publication, []StreamPackage) (domain.Publication, error)
	GetCurrentPublication(context.Context, string) (domain.Publication, error)
	GetPublication(context.Context, string, string) (domain.Publication, error)
}

// IDGenerator задает источник идентификаторов для use cases и тестов.
type IDGenerator interface {
	NewID() string
}

// RandomIDGenerator генерирует UUID-like identifiers без инфраструктурной зависимости.
type RandomIDGenerator struct{}

// NewID возвращает новый случайный identifier.
func (RandomIDGenerator) NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("id-%d", time.Now().UTC().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
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
		ids = RandomIDGenerator{}
	}
	return &Service{repo: repo, clock: clock, ids: ids}
}

// CreateRoleCommand описывает создание роли с JSON snapshot прав.
type CreateRoleCommand struct {
	RestaurantID    string `json:"restaurant_id"`
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
	RestaurantID string `json:"restaurant_id"`
	RoleID       string `json:"role_id"`
	Name         string `json:"name"`
	PIN          string `json:"pin"`
}

// UpdateEmployeeCommand описывает безопасное изменение карточки сотрудника.
type UpdateEmployeeCommand struct {
	RoleID string                 `json:"role_id,omitempty"`
	Name   string                 `json:"name,omitempty"`
	Status *domain.EmployeeStatus `json:"status,omitempty"`
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
	RestaurantID       string                 `json:"restaurant_id"`
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
	RestaurantID string `json:"restaurant_id"`
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
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	Code         string `json:"code"`
}

type UpdateCatalogTagCommand struct {
	Name   string                  `json:"name,omitempty"`
	Code   string                  `json:"code,omitempty"`
	Status *domain.LifecycleStatus `json:"status,omitempty"`
}

type AssignCatalogItemTagCommand struct {
	RestaurantID  string `json:"restaurant_id"`
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
	RestaurantID     string `json:"restaurant_id"`
	ModifierGroupID  string `json:"modifier_group_id"`
	Name             string `json:"name"`
	PriceMinor       int64  `json:"price_minor"`
	LegacyPriceDelta *int64 `json:"price_delta,omitempty"`
}

type UpdateModifierOptionCommand struct {
	Name       string                  `json:"name,omitempty"`
	PriceMinor *int64                  `json:"price_minor,omitempty"`
	Status     *domain.LifecycleStatus `json:"status,omitempty"`
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
	Name              string `json:"name"`
	Price             int64  `json:"price"`
	Currency          string `json:"currency"`
	AvailabilityJSON  string `json:"availability_json"`
	StationRoutingKey string `json:"station_routing_key"`
}

// UpdateMenuItemCommand описывает изменение menu item и его publication lifecycle.
type UpdateMenuItemCommand struct {
	CatalogItemID     string                  `json:"catalog_item_id,omitempty"`
	CategoryID        string                  `json:"category_id,omitempty"`
	Name              string                  `json:"name,omitempty"`
	Price             *int64                  `json:"price,omitempty"`
	Currency          string                  `json:"currency,omitempty"`
	Status            *domain.LifecycleStatus `json:"status,omitempty"`
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
	return s.repo.CreateRestaurant(ctx, restaurant)
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
	return s.repo.UpdateRestaurant(ctx, restaurant)
}

// ArchiveRestaurant выполняет soft-delete ресторана.
func (s *Service) ArchiveRestaurant(ctx context.Context, id string) (domain.Restaurant, error) {
	status := domain.RestaurantArchived
	return s.UpdateRestaurant(ctx, id, UpdateRestaurantCommand{Status: &status})
}

// CreateRole создает Cloud-authored роль.
func (s *Service) CreateRole(ctx context.Context, cmd CreateRoleCommand) (domain.Role, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	name := strings.TrimSpace(cmd.Name)
	permissions := strings.TrimSpace(cmd.PermissionsJSON)
	if permissions == "" {
		permissions = "{}"
	}
	if restaurantID == "" || name == "" {
		return domain.Role{}, fmt.Errorf("%w: restaurant_id and name are required", domain.ErrInvalid)
	}
	if err := domain.ValidatePermissionsJSON(permissions); err != nil {
		return domain.Role{}, err
	}
	now := s.clock.Now().UTC()
	role := domain.Role{
		ID:              s.ids.NewID(),
		RestaurantID:    restaurantID,
		Name:            name,
		PermissionsJSON: canonicalJSON(permissions),
		Active:          true,
		CloudVersion:    1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return s.repo.CreateRole(ctx, role)
}

// ListRoles возвращает роли ресторана.
func (s *Service) ListRoles(ctx context.Context, restaurantID string) ([]domain.Role, error) {
	return s.repo.ListRoles(ctx, strings.TrimSpace(restaurantID))
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
	role.CloudVersion++
	role.UpdatedAt = s.clock.Now().UTC()
	if !role.Active && role.ArchivedAt == nil {
		archivedAt := role.UpdatedAt
		role.ArchivedAt = &archivedAt
	}
	return s.repo.UpdateRole(ctx, role)
}

// ArchiveRole архивирует роль без физического удаления.
func (s *Service) ArchiveRole(ctx context.Context, id string) (domain.Role, error) {
	active := false
	return s.UpdateRole(ctx, id, UpdateRoleCommand{Active: &active})
}

// CreateEmployee создает Cloud-authored сотрудника и хэширует PIN credential.
func (s *Service) CreateEmployee(ctx context.Context, cmd CreateEmployeeCommand) (domain.Employee, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	roleID := strings.TrimSpace(cmd.RoleID)
	name := strings.TrimSpace(cmd.Name)
	if restaurantID == "" || roleID == "" || name == "" || strings.TrimSpace(cmd.PIN) == "" {
		return domain.Employee{}, fmt.Errorf("%w: restaurant_id, role_id, name and pin are required", domain.ErrInvalid)
	}
	role, err := s.repo.GetRole(ctx, roleID)
	if err != nil {
		return domain.Employee{}, err
	}
	if !role.Active || role.RestaurantID != restaurantID {
		return domain.Employee{}, fmt.Errorf("%w: role is archived or belongs to another restaurant", domain.ErrInvalid)
	}
	if err := s.ensurePINUnique(ctx, restaurantID, "", cmd.PIN); err != nil {
		return domain.Employee{}, err
	}
	pinHash, err := hashPIN(cmd.PIN)
	if err != nil {
		return domain.Employee{}, err
	}
	now := s.clock.Now().UTC()
	employee := domain.Employee{
		ID:                     s.ids.NewID(),
		RestaurantID:           restaurantID,
		RoleID:                 roleID,
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
	return s.repo.CreateEmployee(ctx, employee)
}

// ListEmployees возвращает сотрудников ресторана.
func (s *Service) ListEmployees(ctx context.Context, restaurantID string) ([]domain.Employee, error) {
	items, err := s.repo.ListEmployees(ctx, strings.TrimSpace(restaurantID))
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].PINConfigured = strings.TrimSpace(items[i].PINHash) != ""
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
		if !role.Active || role.RestaurantID != employee.RestaurantID {
			return domain.Employee{}, fmt.Errorf("%w: role is archived or belongs to another restaurant", domain.ErrInvalid)
		}
		employee.RoleID = role.ID
		employee.PermissionSnapshotJSON = role.PermissionsJSON
	}
	employee.CloudVersion++
	employee.UpdatedAt = s.clock.Now().UTC()
	if employee.Status == domain.EmployeeSuspended && employee.SuspendedAt == nil {
		suspendedAt := employee.UpdatedAt
		employee.SuspendedAt = &suspendedAt
	}
	if employee.Status == domain.EmployeeArchived && employee.ArchivedAt == nil {
		archivedAt := employee.UpdatedAt
		employee.ArchivedAt = &archivedAt
	}
	return s.repo.UpdateEmployee(ctx, employee)
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
	if err := s.ensurePINUnique(ctx, employee.RestaurantID, employee.ID, cmd.PIN); err != nil {
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
	return s.repo.UpdateEmployee(ctx, employee)
}

// CreateCatalogItem создает draft catalog item в Cloud-owned catalog.
func (s *Service) CreateCatalogItem(ctx context.Context, cmd CreateCatalogItemCommand) (domain.CatalogItem, error) {
	if cmd.Kind == "" && cmd.Type != "" {
		cmd.Kind = cmd.Type
	}
	if err := validateCatalogFields(cmd.RestaurantID, cmd.Kind, cmd.Name, cmd.SKU, cmd.BaseUnit); err != nil {
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
	return s.repo.CreateCatalogItem(ctx, item)
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
	return s.repo.UpdateCatalogItem(ctx, item)
}

// ArchiveCatalogItem архивирует catalog item без физического удаления.
func (s *Service) ArchiveCatalogItem(ctx context.Context, id string) (domain.CatalogItem, error) {
	status := domain.StatusArchived
	return s.UpdateCatalogItem(ctx, id, UpdateCatalogItemCommand{Status: &status})
}

func (s *Service) CreateCatalogFolder(ctx context.Context, cmd CreateCatalogFolderCommand) (domain.CatalogFolder, error) {
	restaurantID, name := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.Name)
	if restaurantID == "" || name == "" {
		return domain.CatalogFolder{}, fmt.Errorf("%w: restaurant_id and name are required", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	folder := domain.CatalogFolder{ID: s.ids.NewID(), RestaurantID: restaurantID, ParentID: strings.TrimSpace(cmd.ParentID), Name: name, SortOrder: cmd.SortOrder, Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	return s.repo.CreateCatalogFolder(ctx, folder)
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
	return s.repo.UpdateCatalogFolder(ctx, folder)
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
	return s.repo.CreateFolderParameter(ctx, parameter)
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
	return s.repo.UpdateFolderParameter(ctx, parameter)
}

func (s *Service) CreateCatalogTag(ctx context.Context, cmd CreateCatalogTagCommand) (domain.CatalogTag, error) {
	restaurantID, name, code := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.Name), strings.TrimSpace(cmd.Code)
	if restaurantID == "" || name == "" || code == "" {
		return domain.CatalogTag{}, fmt.Errorf("%w: restaurant_id, name and code are required", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	tag := domain.CatalogTag{ID: s.ids.NewID(), RestaurantID: restaurantID, Name: name, Code: code, Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	return s.repo.CreateCatalogTag(ctx, tag)
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
	return s.repo.UpdateCatalogTag(ctx, tag)
}

func (s *Service) AssignCatalogItemTag(ctx context.Context, cmd AssignCatalogItemTagCommand) (domain.CatalogItemTag, error) {
	restaurantID, itemID, tagID := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.CatalogItemID), strings.TrimSpace(cmd.TagID)
	if restaurantID == "" || itemID == "" || tagID == "" {
		return domain.CatalogItemTag{}, fmt.Errorf("%w: restaurant_id, catalog_item_id and tag_id are required", domain.ErrInvalid)
	}
	tag := domain.CatalogItemTag{RestaurantID: restaurantID, CatalogItemID: itemID, TagID: tagID, CloudVersion: 1, CreatedAt: s.clock.Now().UTC()}
	return s.repo.AssignCatalogItemTag(ctx, tag)
}

func (s *Service) CreateModifierGroup(ctx context.Context, cmd CreateModifierGroupCommand) (domain.ModifierGroup, error) {
	restaurantID, name := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.Name)
	if restaurantID == "" || name == "" || cmd.MinCount < 0 || cmd.MaxCount < 0 || (cmd.MaxCount > 0 && cmd.MinCount > cmd.MaxCount) {
		return domain.ModifierGroup{}, fmt.Errorf("%w: modifier group requires valid restaurant_id, name and min/max counts", domain.ErrInvalid)
	}
	now := s.clock.Now().UTC()
	group := domain.ModifierGroup{ID: s.ids.NewID(), RestaurantID: restaurantID, Name: name, Required: cmd.Required, MinCount: cmd.MinCount, MaxCount: cmd.MaxCount, Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	return s.repo.CreateModifierGroup(ctx, group)
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
	return s.repo.UpdateModifierGroup(ctx, group)
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
	now := s.clock.Now().UTC()
	option := domain.ModifierOption{ID: s.ids.NewID(), RestaurantID: restaurantID, ModifierGroupID: groupID, Name: name, PriceMinor: price, Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	return s.repo.CreateModifierOption(ctx, option)
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
	return s.repo.UpdateModifierOption(ctx, option)
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
	return s.repo.CreateModifierGroupBinding(ctx, binding)
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
	return s.repo.UpdateModifierGroupBinding(ctx, binding)
}

func (s *Service) CreatePricingPolicy(ctx context.Context, cmd CreatePricingPolicyCommand) (domain.PricingPolicy, error) {
	restaurantID, name := strings.TrimSpace(cmd.RestaurantID), strings.TrimSpace(cmd.Name)
	if restaurantID == "" || name == "" || cmd.ApplicationIndex <= 0 {
		return domain.PricingPolicy{}, fmt.Errorf("%w: pricing policy requires restaurant_id, name and positive application_index", domain.ErrInvalid)
	}
	if err := domain.ValidatePricingPolicyKind(cmd.Kind); err != nil {
		return domain.PricingPolicy{}, err
	}
	if err := validatePolicyAmount(cmd.AmountKind, cmd.AmountMinor, cmd.ValueBasisPoints); err != nil {
		return domain.PricingPolicy{}, err
	}
	now := s.clock.Now().UTC()
	policy := domain.PricingPolicy{ID: s.ids.NewID(), RestaurantID: restaurantID, Name: name, Kind: cmd.Kind, Scope: strings.TrimSpace(cmd.Scope), AmountKind: strings.TrimSpace(cmd.AmountKind), AmountMinor: cmd.AmountMinor, ValueBasisPoints: cmd.ValueBasisPoints, ApplicationIndex: cmd.ApplicationIndex, Manual: cmd.Manual, RequiresPermission: strings.TrimSpace(cmd.RequiresPermission), Status: domain.StatusPublished, CloudVersion: 1, CreatedAt: now, UpdatedAt: now}
	return s.repo.CreatePricingPolicy(ctx, policy)
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
	return s.repo.UpdatePricingPolicy(ctx, policy)
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
	return s.repo.CreateHall(ctx, hall)
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
	return s.repo.UpdateHall(ctx, hall)
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
	return s.repo.CreateTable(ctx, table)
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
	return s.repo.UpdateTable(ctx, table)
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
	catalogItem, err := s.repo.GetCatalogItem(ctx, catalogItemID)
	if err != nil {
		return domain.MenuItem{}, err
	}
	if catalogItem.RestaurantID != restaurantID || catalogItem.Status == domain.StatusArchived {
		return domain.MenuItem{}, fmt.Errorf("%w: catalog item is archived or belongs to another restaurant", domain.ErrInvalid)
	}
	availability, err := normalizeAvailability(cmd.AvailabilityJSON)
	if err != nil {
		return domain.MenuItem{}, err
	}
	now := s.clock.Now().UTC()
	item := domain.MenuItem{
		ID:                s.ids.NewID(),
		RestaurantID:      restaurantID,
		CatalogItemID:     catalogItemID,
		CategoryID:        strings.TrimSpace(cmd.CategoryID),
		Name:              name,
		Price:             cmd.Price,
		Currency:          currency,
		Status:            domain.StatusPublished,
		AvailabilityJSON:  availability,
		StationRoutingKey: strings.TrimSpace(cmd.StationRoutingKey),
		CloudVersion:      1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	return s.repo.CreateMenuItem(ctx, item)
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
		catalogItem, err := s.repo.GetCatalogItem(ctx, strings.TrimSpace(cmd.CatalogItemID))
		if err != nil {
			return domain.MenuItem{}, err
		}
		if catalogItem.RestaurantID != item.RestaurantID || catalogItem.Status == domain.StatusArchived {
			return domain.MenuItem{}, fmt.Errorf("%w: catalog item is archived or belongs to another restaurant", domain.ErrInvalid)
		}
		item.CatalogItemID = catalogItem.ID
	}
	if strings.TrimSpace(cmd.CategoryID) != "" {
		item.CategoryID = strings.TrimSpace(cmd.CategoryID)
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
	if item.Status == domain.StatusPublished {
		catalogItem, err := s.repo.GetCatalogItem(ctx, item.CatalogItemID)
		if err != nil {
			return domain.MenuItem{}, err
		}
		if catalogItem.Status == domain.StatusArchived {
			return domain.MenuItem{}, fmt.Errorf("%w: archived catalog item cannot be used by published menu item", domain.ErrInvalid)
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
	return s.repo.UpdateMenuItem(ctx, item)
}

// ArchiveMenuItem архивирует menu item без физического удаления.
func (s *Service) ArchiveMenuItem(ctx context.Context, id string) (domain.MenuItem, error) {
	status := domain.StatusArchived
	return s.UpdateMenuItem(ctx, id, UpdateMenuItemCommand{Status: &status})
}

// Publish создает versioned deterministic package для Cloud -> Edge sync.
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

// GetCurrentPublishedState возвращает Cloud UI-safe metadata текущей публикации.
func (s *Service) GetCurrentPublishedState(ctx context.Context, restaurantID string) (PublicationSummary, error) {
	pub, err := s.repo.GetCurrentPublication(ctx, strings.TrimSpace(restaurantID))
	if err != nil {
		return PublicationSummary{}, err
	}
	var packet domain.MasterDataPacket
	_ = json.Unmarshal(pub.PackageJSON, &packet)
	counts := map[string]int{
		"restaurants":   len(packet.Restaurants),
		"roles":         len(packet.Roles),
		"employees":     len(packet.Employees),
		"catalog_items": len(packet.CatalogItems),
		"menu_items":    len(packet.MenuItems),
		"halls":         len(packet.Halls),
		"tables":        len(packet.Tables),
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

func (s *Service) buildPacket(ctx context.Context, restaurantID, nodeDeviceID string, version int64, now time.Time) (domain.MasterDataPacket, map[string]int, []StreamPackage, error) {
	restaurant, err := s.repo.GetRestaurant(ctx, restaurantID)
	if err != nil && !errorsIsNotFound(err) {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	roles, err := s.repo.ListRoles(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	employees, err := s.repo.ListEmployees(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
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
	sortRoles(roles)
	sortEmployees(employees)
	sortCatalog(catalogItems)
	sortMenu(menuItems)
	sortHalls(halls)
	sortTables(tables)

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
		Employees:              edgeEmployees(employees),
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
	}
	streams, err := streamPackages(packet)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	return packet, counts, streams, nil
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
		NodeDeviceID           string                             `json:"node_device_id,omitempty"`
		RestaurantID           string                             `json:"restaurant_id"`
		SyncMode               string                             `json:"sync_mode"`
		CheckpointToken        string                             `json:"checkpoint_token,omitempty"`
		CloudVersion           int64                              `json:"cloud_version"`
		CloudUpdatedAt         time.Time                          `json:"cloud_updated_at"`
		CatalogItems           []domain.EdgeCatalogItem           `json:"catalog_items"`
		Folders                []domain.EdgeCatalogFolder         `json:"folders,omitempty"`
		FolderParameters       []domain.EdgeFolderParameter       `json:"folder_parameters,omitempty"`
		Tags                   []domain.EdgeCatalogTag            `json:"tags,omitempty"`
		ItemTags               []domain.EdgeCatalogItemTag        `json:"item_tags,omitempty"`
		ModifierGroups         []domain.EdgeModifierGroup         `json:"modifier_groups,omitempty"`
		ModifierOptions        []domain.EdgeModifierOption        `json:"modifier_options,omitempty"`
		ModifierBindings       []domain.EdgeModifierGroupBinding  `json:"modifier_bindings,omitempty"`
		MenuItemModifierGroups []domain.EdgeMenuItemModifierGroup `json:"menu_item_modifier_groups,omitempty"`
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
		NodeDeviceID    string                `json:"node_device_id,omitempty"`
		RestaurantID    string                `json:"restaurant_id"`
		SyncMode        string                `json:"sync_mode"`
		CheckpointToken string                `json:"checkpoint_token,omitempty"`
		CloudVersion    int64                 `json:"cloud_version"`
		CloudUpdatedAt  time.Time             `json:"cloud_updated_at"`
		MenuItems       []domain.EdgeMenuItem `json:"menu_items"`
	}
	type pricingPayload struct {
		NodeDeviceID    string                     `json:"node_device_id,omitempty"`
		RestaurantID    string                     `json:"restaurant_id"`
		SyncMode        string                     `json:"sync_mode"`
		CheckpointToken string                     `json:"checkpoint_token,omitempty"`
		CloudVersion    int64                      `json:"cloud_version"`
		CloudUpdatedAt  time.Time                  `json:"cloud_updated_at"`
		PricingPolicies []domain.EdgePricingPolicy `json:"pricing_policies,omitempty"`
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
	catalog, err := build("catalog", catalogPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, CatalogItems: packet.CatalogItems, Folders: packet.Folders, FolderParameters: packet.FolderParameters, Tags: packet.Tags, ItemTags: packet.ItemTags, ModifierGroups: packet.ModifierGroups, ModifierOptions: packet.ModifierOptions, ModifierBindings: packet.ModifierBindings, MenuItemModifierGroups: packet.MenuItemModifierGroups})
	if err != nil {
		return nil, err
	}
	floor, err := build("floor", floorPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, Halls: packet.Halls, Tables: packet.Tables})
	if err != nil {
		return nil, err
	}
	menu, err := build("menu", menuPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, MenuItems: packet.MenuItems})
	if err != nil {
		return nil, err
	}
	pricing, err := build("pricing_policy", pricingPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, PricingPolicies: packet.PricingPolicies})
	if err != nil {
		return nil, err
	}
	return []StreamPackage{restaurants, staff, catalog, floor, menu, pricing}, nil
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

func edgeEmployees(items []domain.Employee) []domain.EdgeEmployee {
	out := make([]domain.EdgeEmployee, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeEmployee{ID: item.ID, RestaurantID: item.RestaurantID, RoleID: item.RoleID, Name: item.Name, PINHash: item.PINHash, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
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
		out = append(out, domain.EdgeFolderParameter{ID: item.ID, FolderID: item.FolderID, Key: item.Key, ValueType: item.ValueType, ValueJSON: item.ValueJSON, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeTags(items []domain.CatalogTag) []domain.EdgeCatalogTag {
	out := make([]domain.EdgeCatalogTag, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeCatalogTag{ID: item.ID, Name: item.Name, Code: item.Code, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out
}

func edgeItemTags(items []domain.CatalogItemTag) []domain.EdgeCatalogItemTag {
	out := make([]domain.EdgeCatalogItemTag, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeCatalogItemTag{CatalogItemID: item.CatalogItemID, TagID: item.TagID})
	}
	return out
}

func edgeModifierGroups(items []domain.ModifierGroup) []domain.EdgeModifierGroup {
	out := make([]domain.EdgeModifierGroup, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeModifierGroup{ID: item.ID, Name: item.Name, Required: item.Required, MinCount: item.MinCount, MaxCount: item.MaxCount, Active: item.ActiveForPOS()})
	}
	return out
}

func edgeModifierOptions(items []domain.ModifierOption) []domain.EdgeModifierOption {
	out := make([]domain.EdgeModifierOption, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeModifierOption{ID: item.ID, ModifierGroupID: item.ModifierGroupID, Name: item.Name, PriceMinor: item.PriceMinor, Active: item.ActiveForPOS()})
	}
	return out
}

func edgeModifierBindings(items []domain.ModifierGroupBinding) []domain.EdgeModifierGroupBinding {
	out := make([]domain.EdgeModifierGroupBinding, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeModifierGroupBinding{ID: item.ID, ModifierGroupID: item.ModifierGroupID, TargetType: string(item.TargetType), TargetID: item.TargetID, SortOrder: item.SortOrder, Active: item.ActiveForPOS()})
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
				out = append(out, domain.EdgeMenuItemModifierGroup{MenuItemID: menuItem.ID, ModifierGroupID: binding.ModifierGroupID, SortOrder: binding.SortOrder, Required: group.Required, MinCount: group.MinCount, MaxCount: group.MaxCount, Active: true})
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

func edgeMenuItems(items []domain.MenuItem) []domain.EdgeMenuItem {
	out := make([]domain.EdgeMenuItem, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeMenuItem{ID: item.ID, CatalogItemID: item.CatalogItemID, Name: item.Name, Price: item.Price, Currency: item.Currency, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
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

func validateCatalogFields(restaurantID string, kind domain.CatalogItemKind, name, sku, baseUnit string) error {
	if strings.TrimSpace(restaurantID) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(sku) == "" || strings.TrimSpace(baseUnit) == "" {
		return fmt.Errorf("%w: restaurant_id, name, sku and base_unit are required", domain.ErrInvalid)
	}
	return domain.ValidateCatalogItemKind(kind)
}

func validatePolicyAmount(amountKind string, amountMinor, valueBasisPoints int64) error {
	switch strings.TrimSpace(amountKind) {
	case "fixed":
		if amountMinor <= 0 || valueBasisPoints != 0 {
			return fmt.Errorf("%w: fixed pricing policy requires positive amount_minor only", domain.ErrInvalid)
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

func (s *Service) ensurePINUnique(ctx context.Context, restaurantID, exceptEmployeeID, pin string) error {
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return fmt.Errorf("%w: pin is required", domain.ErrInvalid)
	}
	employees, err := s.repo.ListEmployees(ctx, strings.TrimSpace(restaurantID))
	if err != nil {
		return err
	}
	for _, employee := range employees {
		if employee.ID == strings.TrimSpace(exceptEmployeeID) || employee.Status == domain.EmployeeArchived {
			continue
		}
		if verifyPIN(employee.PINHash, pin) == nil {
			return fmt.Errorf("%w: duplicate non-archived employee PIN in restaurant", domain.ErrPINAlreadyExists)
		}
	}
	return nil
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

// CloneStreamPackages возвращает копию списка stream packages для тестов и адаптеров.
func CloneStreamPackages(items []StreamPackage) []StreamPackage {
	return slices.Clone(items)
}
