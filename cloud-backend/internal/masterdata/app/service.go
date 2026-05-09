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
	CreateCategory(context.Context, domain.Category) (domain.Category, error)
	ListCategories(context.Context, string) ([]domain.Category, error)
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
	RestaurantID string                 `json:"restaurant_id"`
	Kind         domain.CatalogItemKind `json:"kind"`
	Type         domain.CatalogItemKind `json:"type,omitempty"`
	Name         string                 `json:"name"`
	SKU          string                 `json:"sku"`
	BaseUnit     string                 `json:"base_unit"`
}

// UpdateCatalogItemCommand описывает изменение Cloud-owned catalog item.
type UpdateCatalogItemCommand struct {
	Kind     *domain.CatalogItemKind `json:"kind,omitempty"`
	Type     *domain.CatalogItemKind `json:"type,omitempty"`
	Name     string                  `json:"name,omitempty"`
	SKU      string                  `json:"sku,omitempty"`
	BaseUnit string                  `json:"base_unit,omitempty"`
	Status   *domain.LifecycleStatus `json:"status,omitempty"`
}

// CreateCategoryCommand описывает создание категории меню.
type CreateCategoryCommand struct {
	RestaurantID string `json:"restaurant_id"`
	Name         string `json:"name"`
	SortOrder    int64  `json:"sort_order"`
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
		ID:           s.ids.NewID(),
		RestaurantID: strings.TrimSpace(cmd.RestaurantID),
		Kind:         cmd.Kind,
		Name:         strings.TrimSpace(cmd.Name),
		SKU:          strings.TrimSpace(cmd.SKU),
		BaseUnit:     strings.TrimSpace(cmd.BaseUnit),
		Status:       domain.StatusPublished,
		CloudVersion: 1,
		CreatedAt:    now,
		UpdatedAt:    now,
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
		"categories":    len(packet.Categories),
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
	categories, err := s.repo.ListCategories(ctx, restaurantID)
	if err != nil {
		return domain.MasterDataPacket{}, nil, nil, err
	}
	catalogItems, err := s.repo.ListCatalogItems(ctx, restaurantID)
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
	sortCategories(categories)

	restaurants := []domain.Restaurant{}
	if restaurant.ID != "" && restaurant.Status == domain.RestaurantActive {
		restaurants = append(restaurants, restaurant)
	}
	packet := domain.MasterDataPacket{
		NodeDeviceID:    nodeDeviceID,
		RestaurantID:    restaurantID,
		SyncMode:        "incremental",
		CheckpointToken: fmt.Sprintf("master-data:%s:%d", restaurantID, version),
		CloudVersion:    version,
		CloudUpdatedAt:  now,
		Restaurants:     edgeRestaurants(restaurants),
		Roles:           edgeRoles(roles),
		Employees:       edgeEmployees(employees),
		CatalogItems:    edgeCatalogItems(catalogItems),
		MenuItems:       edgeMenuItems(menuItems),
		Categories:      edgeCategories(categories),
		ModifierGroups:  []domain.EdgeModifierGroup{},
		ModifierOptions: []domain.EdgeModifierOption{},
	}
	counts := map[string]int{
		"restaurants":   len(packet.Restaurants),
		"roles":         len(packet.Roles),
		"employees":     len(packet.Employees),
		"catalog_items": len(packet.CatalogItems),
		"menu_items":    len(packet.MenuItems),
		"categories":    len(packet.Categories),
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
		NodeDeviceID    string                   `json:"node_device_id,omitempty"`
		RestaurantID    string                   `json:"restaurant_id"`
		SyncMode        string                   `json:"sync_mode"`
		CheckpointToken string                   `json:"checkpoint_token,omitempty"`
		CloudVersion    int64                    `json:"cloud_version"`
		CloudUpdatedAt  time.Time                `json:"cloud_updated_at"`
		CatalogItems    []domain.EdgeCatalogItem `json:"catalog_items"`
		Categories      []domain.EdgeCategory    `json:"categories,omitempty"`
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
	catalog, err := build("catalog", catalogPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, CatalogItems: packet.CatalogItems, Categories: packet.Categories})
	if err != nil {
		return nil, err
	}
	menu, err := build("menu", menuPayload{NodeDeviceID: packet.NodeDeviceID, RestaurantID: packet.RestaurantID, SyncMode: packet.SyncMode, CheckpointToken: packet.CheckpointToken, CloudVersion: packet.CloudVersion, CloudUpdatedAt: packet.CloudUpdatedAt, MenuItems: packet.MenuItems})
	if err != nil {
		return nil, err
	}
	return []StreamPackage{restaurants, staff, catalog, menu}, nil
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
		out = append(out, domain.EdgeCatalogItem{ID: item.ID, Type: item.EdgeType(), Name: item.Name, SKU: item.SKU, BaseUnit: item.BaseUnit, Active: item.ActiveForPOS(), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
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

func edgeCategories(items []domain.Category) []domain.EdgeCategory {
	out := make([]domain.EdgeCategory, 0, len(items))
	for _, item := range items {
		out = append(out, domain.EdgeCategory{ID: item.ID, Name: item.Name, SortOrder: item.SortOrder, Active: item.Status != domain.StatusArchived})
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
		if employee.ID == strings.TrimSpace(exceptEmployeeID) || employee.Status != domain.EmployeeActive {
			continue
		}
		if verifyPIN(employee.PINHash, pin) == nil {
			return fmt.Errorf("%w: duplicate active employee PIN in restaurant", domain.ErrConflict)
		}
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

func sortCategories(items []domain.Category) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].SortOrder == items[j].SortOrder {
			return items[i].ID < items[j].ID
		}
		return items[i].SortOrder < items[j].SortOrder
	})
}

// CloneStreamPackages возвращает копию списка stream packages для тестов и адаптеров.
func CloneStreamPackages(items []StreamPackage) []StreamPackage {
	return slices.Clone(items)
}
