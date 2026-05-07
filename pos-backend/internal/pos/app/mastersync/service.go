package mastersync

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
)

type Service struct {
	repo  ports.Repository
	tx    txmanager.Manager
	ids   idgen.Generator
	clock clock.Clock
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock}
}

type ApplyMasterDataCommand struct {
	shared.CommandMeta
	RestaurantID    string                  `json:"restaurant_id,omitempty"`
	StreamName      domain.MasterDataStream `json:"stream,omitempty"`
	SyncMode        domain.SyncMode         `json:"sync_mode,omitempty"`
	CheckpointToken string                  `json:"checkpoint_token,omitempty"`
	CloudVersion    int64                   `json:"cloud_version,omitempty"`
	CloudUpdatedAt  string                  `json:"cloud_updated_at,omitempty"`
	Restaurants     []domain.Restaurant     `json:"restaurants,omitempty"`
	Devices         []domain.Device         `json:"devices,omitempty"`
	Roles           []domain.Role           `json:"roles,omitempty"`
	Employees       []domain.Employee       `json:"employees,omitempty"`
	Halls           []domain.Hall           `json:"halls,omitempty"`
	Tables          []domain.Table          `json:"tables,omitempty"`
	CatalogItems    []domain.CatalogItem    `json:"catalog_items,omitempty"`
	MenuItems       []domain.MenuItem       `json:"menu_items,omitempty"`
}

type ApplyMasterDataResult struct {
	NodeDeviceID   string                       `json:"node_device_id"`
	AppliedAt      string                       `json:"applied_at"`
	AppliedStreams []domain.MasterDataStream    `json:"applied_streams"`
	Counts         map[string]int               `json:"counts"`
	SyncStates     []domain.MasterDataSyncState `json:"sync_states"`
}

func (s *Service) ApplyMasterData(ctx context.Context, cmd ApplyMasterDataCommand) (*ApplyMasterDataResult, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if shared.EffectiveNodeDeviceID(cmd.CommandMeta) == "" {
		return nil, fmt.Errorf("%w: node_device_id is required", domain.ErrInvalid)
	}
	if shared.NormalizeOrigin(cmd.Origin) != domain.OriginCloudSync {
		return nil, fmt.Errorf("%w: master-data ingest origin must be cloud_sync", domain.ErrInvalid)
	}
	if cmd.CloudVersion < 0 {
		return nil, fmt.Errorf("%w: cloud_version must be non-negative", domain.ErrInvalid)
	}
	mode := cmd.SyncMode
	if mode == "" {
		mode = domain.SyncModeFullSnapshot
	}
	if mode != domain.SyncModeFullSnapshot && mode != domain.SyncModeIncremental {
		return nil, fmt.Errorf("%w: sync_mode must be full_snapshot or incremental", domain.ErrInvalid)
	}
	streams, err := streamsToApply(cmd)
	if err != nil {
		return nil, err
	}

	now := s.clock.Now()
	appliedAt := shared.DBTime(now)
	cloudUpdatedAt := strings.TrimSpace(cmd.CloudUpdatedAt)
	if cloudUpdatedAt == "" {
		cloudUpdatedAt = appliedAt
	}
	recordMeta := domain.MasterRecordSyncMeta{
		CloudVersion:   cmd.CloudVersion,
		CloudUpdatedAt: &cloudUpdatedAt,
		LastSyncedAt:   appliedAt,
	}
	counts := map[string]int{}
	var states []domain.MasterDataSyncState

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		for _, stream := range streams {
			if err := s.applyStream(ctx, stream, &cmd, recordMeta, now, counts); err != nil {
				return err
			}
			state := s.buildAppliedState(cmd, stream, mode, appliedAt, cloudUpdatedAt)
			if err := s.repo.UpsertMasterDataSyncState(ctx, &state); err != nil {
				return err
			}
			states = append(states, state)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &ApplyMasterDataResult{
		NodeDeviceID:   shared.EffectiveNodeDeviceID(cmd.CommandMeta),
		AppliedAt:      appliedAt,
		AppliedStreams: streams,
		Counts:         counts,
		SyncStates:     states,
	}, nil
}

func (s *Service) applyStream(ctx context.Context, stream domain.MasterDataStream, cmd *ApplyMasterDataCommand, meta domain.MasterRecordSyncMeta, now time.Time, counts map[string]int) error {
	switch stream {
	case domain.MasterDataStreamRestaurants:
		for i := range cmd.Restaurants {
			v := normalizeRestaurant(cmd.Restaurants[i], now)
			if err := validateRestaurant(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterRestaurant(ctx, &v, meta); err != nil {
				return err
			}
		}
		counts[string(stream)] = len(cmd.Restaurants)
	case domain.MasterDataStreamDevices:
		for i := range cmd.Devices {
			v := normalizeDevice(cmd.Devices[i], now)
			if err := validateDevice(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterDevice(ctx, &v, meta); err != nil {
				return err
			}
		}
		counts[string(stream)] = len(cmd.Devices)
	case domain.MasterDataStreamStaff:
		for i := range cmd.Roles {
			v := normalizeRole(cmd.Roles[i], now)
			if err := validateRole(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterRole(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.Employees {
			v := normalizeEmployee(cmd.Employees[i], now)
			if err := validateEmployee(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterEmployee(ctx, &v, meta); err != nil {
				return err
			}
		}
		counts[string(stream)] = len(cmd.Roles) + len(cmd.Employees)
	case domain.MasterDataStreamFloor:
		for i := range cmd.Halls {
			v := normalizeHall(cmd.Halls[i], now)
			if err := validateHall(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterHall(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.Tables {
			v := normalizeTable(cmd.Tables[i], now)
			if err := validateTable(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterTable(ctx, &v, meta); err != nil {
				return err
			}
		}
		counts[string(stream)] = len(cmd.Halls) + len(cmd.Tables)
	case domain.MasterDataStreamCatalog:
		for i := range cmd.CatalogItems {
			v := normalizeCatalogItem(cmd.CatalogItems[i], now)
			if err := validateCatalogItem(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterCatalogItem(ctx, &v, meta); err != nil {
				return err
			}
		}
		counts[string(stream)] = len(cmd.CatalogItems)
	case domain.MasterDataStreamMenu:
		for i := range cmd.MenuItems {
			v := normalizeMenuItem(cmd.MenuItems[i], now)
			if err := validateMenuItem(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterMenuItem(ctx, &v, meta); err != nil {
				return err
			}
		}
		counts[string(stream)] = len(cmd.MenuItems)
	default:
		return fmt.Errorf("%w: unsupported master data stream %q", domain.ErrInvalid, stream)
	}
	return nil
}

func (s *Service) buildAppliedState(cmd ApplyMasterDataCommand, stream domain.MasterDataStream, mode domain.SyncMode, appliedAt, cloudUpdatedAt string) domain.MasterDataSyncState {
	var restaurantID *string
	if v := strings.TrimSpace(cmd.RestaurantID); v != "" {
		restaurantID = &v
	}
	var checkpoint *string
	if v := strings.TrimSpace(cmd.CheckpointToken); v != "" {
		checkpoint = &v
	}
	return domain.MasterDataSyncState{
		ID:                 s.ids.NewID(),
		RestaurantID:       restaurantID,
		NodeDeviceID:       shared.EffectiveNodeDeviceID(cmd.CommandMeta),
		StreamName:         stream,
		Direction:          domain.SyncDirectionCloudToEdge,
		SyncMode:           mode,
		CheckpointToken:    checkpoint,
		LastCloudVersion:   cmd.CloudVersion,
		LastCloudUpdatedAt: &cloudUpdatedAt,
		LastAppliedAt:      &appliedAt,
		Status:             "applied",
		CreatedAt:          appliedAt,
		UpdatedAt:          appliedAt,
	}
}

func streamsToApply(cmd ApplyMasterDataCommand) ([]domain.MasterDataStream, error) {
	if cmd.StreamName != "" {
		if !supportedStream(cmd.StreamName) {
			return nil, fmt.Errorf("%w: unsupported master data stream %q", domain.ErrInvalid, cmd.StreamName)
		}
		return []domain.MasterDataStream{cmd.StreamName}, nil
	}
	streams := make([]domain.MasterDataStream, 0, 6)
	if len(cmd.Restaurants) > 0 {
		streams = append(streams, domain.MasterDataStreamRestaurants)
	}
	if len(cmd.Devices) > 0 {
		streams = append(streams, domain.MasterDataStreamDevices)
	}
	if len(cmd.Roles) > 0 || len(cmd.Employees) > 0 {
		streams = append(streams, domain.MasterDataStreamStaff)
	}
	if len(cmd.Halls) > 0 || len(cmd.Tables) > 0 {
		streams = append(streams, domain.MasterDataStreamFloor)
	}
	if len(cmd.CatalogItems) > 0 {
		streams = append(streams, domain.MasterDataStreamCatalog)
	}
	if len(cmd.MenuItems) > 0 {
		streams = append(streams, domain.MasterDataStreamMenu)
	}
	if len(streams) == 0 {
		return nil, fmt.Errorf("%w: at least one supported master data stream is required", domain.ErrInvalid)
	}
	return streams, nil
}

func supportedStream(stream domain.MasterDataStream) bool {
	switch stream {
	case domain.MasterDataStreamRestaurants,
		domain.MasterDataStreamDevices,
		domain.MasterDataStreamStaff,
		domain.MasterDataStreamFloor,
		domain.MasterDataStreamCatalog,
		domain.MasterDataStreamMenu:
		return true
	default:
		return false
	}
}

func normalizeRestaurant(v domain.Restaurant, now time.Time) domain.Restaurant {
	v.ID = strings.TrimSpace(v.ID)
	v.Name = strings.TrimSpace(v.Name)
	v.Timezone = strings.TrimSpace(v.Timezone)
	v.Currency = strings.ToUpper(strings.TrimSpace(v.Currency))
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeDevice(v domain.Device, now time.Time) domain.Device {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.DeviceCode = strings.TrimSpace(v.DeviceCode)
	v.Name = strings.TrimSpace(v.Name)
	v.Type = strings.TrimSpace(v.Type)
	v.RegisteredAt = defaultTime(v.RegisteredAt, now)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeRole(v domain.Role, now time.Time) domain.Role {
	v.ID = strings.TrimSpace(v.ID)
	v.Name = strings.TrimSpace(v.Name)
	v.PermissionsJSON = strings.TrimSpace(v.PermissionsJSON)
	if v.PermissionsJSON == "" {
		v.PermissionsJSON = "{}"
	}
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeEmployee(v domain.Employee, now time.Time) domain.Employee {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.RoleID = strings.TrimSpace(v.RoleID)
	v.Name = strings.TrimSpace(v.Name)
	v.PINHash = strings.TrimSpace(v.PINHash)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeHall(v domain.Hall, now time.Time) domain.Hall {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.Name = strings.TrimSpace(v.Name)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeTable(v domain.Table, now time.Time) domain.Table {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.HallID = strings.TrimSpace(v.HallID)
	v.Name = strings.TrimSpace(v.Name)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeCatalogItem(v domain.CatalogItem, now time.Time) domain.CatalogItem {
	v.ID = strings.TrimSpace(v.ID)
	v.Name = strings.TrimSpace(v.Name)
	v.SKU = strings.TrimSpace(v.SKU)
	v.BaseUnit = strings.TrimSpace(v.BaseUnit)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeMenuItem(v domain.MenuItem, now time.Time) domain.MenuItem {
	v.ID = strings.TrimSpace(v.ID)
	v.CatalogItemID = strings.TrimSpace(v.CatalogItemID)
	v.Name = strings.TrimSpace(v.Name)
	v.Currency = strings.ToUpper(strings.TrimSpace(v.Currency))
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func defaultTime(v, fallback time.Time) time.Time {
	if v.IsZero() {
		return fallback
	}
	return v
}

func validateRestaurant(v domain.Restaurant) error {
	if v.ID == "" || v.Name == "" || v.Timezone == "" || v.Currency == "" {
		return fmt.Errorf("%w: restaurant id, name, timezone and currency are required", domain.ErrInvalid)
	}
	return nil
}

func validateDevice(v domain.Device) error {
	if v.ID == "" || v.RestaurantID == "" || v.DeviceCode == "" || v.Name == "" || v.Type == "" {
		return fmt.Errorf("%w: device id, restaurant_id, device_code, name and type are required", domain.ErrInvalid)
	}
	return nil
}

func validateRole(v domain.Role) error {
	if v.ID == "" || v.Name == "" || v.PermissionsJSON == "" || !json.Valid([]byte(v.PermissionsJSON)) {
		return fmt.Errorf("%w: role id, name and valid permissions_json are required", domain.ErrInvalid)
	}
	return nil
}

func validateEmployee(v domain.Employee) error {
	if v.ID == "" || v.RestaurantID == "" || v.RoleID == "" || v.Name == "" || v.PINHash == "" {
		return fmt.Errorf("%w: employee id, restaurant_id, role_id, name and pin_hash are required", domain.ErrInvalid)
	}
	return nil
}

func validateHall(v domain.Hall) error {
	if v.ID == "" || v.RestaurantID == "" || v.Name == "" {
		return fmt.Errorf("%w: hall id, restaurant_id and name are required", domain.ErrInvalid)
	}
	return nil
}

func validateTable(v domain.Table) error {
	if v.ID == "" || v.RestaurantID == "" || v.HallID == "" || v.Name == "" || v.Seats < 0 {
		return fmt.Errorf("%w: table id, restaurant_id, hall_id, name and non-negative seats are required", domain.ErrInvalid)
	}
	return nil
}

func validateCatalogItem(v domain.CatalogItem) error {
	if v.ID == "" || v.Name == "" || v.SKU == "" || v.BaseUnit == "" {
		return fmt.Errorf("%w: catalog item id, name, sku and base_unit are required", domain.ErrInvalid)
	}
	switch v.Type {
	case domain.CatalogItemIngredient, domain.CatalogItemDish, domain.CatalogItemGood:
		return nil
	default:
		return fmt.Errorf("%w: unsupported catalog item type", domain.ErrInvalid)
	}
}

func validateMenuItem(v domain.MenuItem) error {
	if v.ID == "" || v.CatalogItemID == "" || v.Name == "" || v.Currency == "" || v.Price < 0 {
		return fmt.Errorf("%w: menu item id, catalog_item_id, name, currency and non-negative price are required", domain.ErrInvalid)
	}
	return nil
}
