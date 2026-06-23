package mastersync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/domain/kitchen"
	"pos-backend/internal/pos/ports"
)

const (
	fullSnapshotReasonTerminalRestaurantChanged = "terminal_restaurant_changed"
	fullSnapshotReasonNodeRoleChanged           = "node_role_changed"
)

type Service struct {
	repo                     ports.Repository
	tx                       txmanager.Manager
	ids                      idgen.Generator
	clock                    clock.Clock
	backupBeforeFullSnapshot BackupFunc
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return NewServiceWithOptions(repo, tx, ids, clock, Options{})
}

// Options задает внешние runtime hooks для Cloud -> Edge master-data ingest.
type Options struct {
	BackupBeforeFullSnapshot BackupFunc
}

// BackupRequest содержит безопасные metadata для backup-before-data-load без payload dump.
type BackupRequest struct {
	NodeDeviceID       string
	RestaurantID       string
	Streams            []domain.MasterDataStream
	CloudVersion       int64
	FullSnapshotReason string
	AppliedAt          string
	CloudUpdatedAt     string
}

// BackupFunc выполняет recoverable backup перед применением master-data full_snapshot.
type BackupFunc func(context.Context, BackupRequest) error

// NewServiceWithOptions создает master sync service с явно заданными runtime hooks.
func NewServiceWithOptions(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock, options Options) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock, backupBeforeFullSnapshot: options.BackupBeforeFullSnapshot}
}

type ApplyMasterDataCommand struct {
	shared.CommandMeta
	RestaurantID           string                         `json:"restaurant_id,omitempty"`
	StreamName             domain.MasterDataStream        `json:"stream,omitempty"`
	SyncMode               domain.SyncMode                `json:"sync_mode,omitempty"`
	FullSnapshotReason     string                         `json:"full_snapshot_reason,omitempty"`
	CheckpointToken        string                         `json:"checkpoint_token,omitempty"`
	CloudVersion           int64                          `json:"cloud_version,omitempty"`
	CloudUpdatedAt         string                         `json:"cloud_updated_at,omitempty"`
	Restaurants            []domain.Restaurant            `json:"restaurants,omitempty"`
	Devices                []domain.Device                `json:"devices,omitempty"`
	Roles                  []domain.Role                  `json:"roles,omitempty"`
	Employees              []domain.Employee              `json:"employees,omitempty"`
	Halls                  []domain.Hall                  `json:"halls,omitempty"`
	Tables                 []domain.Table                 `json:"tables,omitempty"`
	CatalogItems           []domain.CatalogItem           `json:"catalog_items,omitempty"`
	Folders                []domain.CatalogFolder         `json:"folders,omitempty"`
	FolderParameters       []domain.FolderParameter       `json:"folder_parameters,omitempty"`
	Tags                   []domain.CatalogTag            `json:"tags,omitempty"`
	ItemTags               []domain.CatalogItemTag        `json:"item_tags,omitempty"`
	ModifierGroups         []domain.ModifierGroup         `json:"modifier_groups,omitempty"`
	ModifierOptions        []domain.ModifierOption        `json:"modifier_options,omitempty"`
	ModifierBindings       []domain.ModifierGroupBinding  `json:"modifier_bindings,omitempty"`
	MenuItemModifierGroups []domain.MenuItemModifierGroup `json:"menu_item_modifier_groups,omitempty"`
	MenuItems              []domain.MenuItem              `json:"menu_items,omitempty"`
	TaxProfiles            []domain.TaxProfile            `json:"tax_profiles,omitempty"`
	TaxRules               []domain.TaxRule               `json:"tax_rules,omitempty"`
	ServiceChargeRules     []domain.ServiceChargeRule     `json:"service_charge_rules,omitempty"`
	PricingPolicies        []domain.PricingPolicy         `json:"pricing_policies,omitempty"`
	RecipeVersions         []domain.RecipeVersion         `json:"recipe_versions,omitempty"`
	RecipeLines            []domain.RecipeLine            `json:"recipe_lines,omitempty"`
	StopListEntries        []domain.StopListEntry         `json:"stop_lists,omitempty"`
	Warehouses             []domain.WarehouseReference    `json:"warehouses,omitempty"`
	CatalogSuggestions     []ProposalFeedback             `json:"catalog_suggestions,omitempty"`
	RecipeSuggestions      []ProposalFeedback             `json:"recipe_suggestions,omitempty"`
}

// ProposalFeedback переносит Cloud review status для локальной kitchen proposal записи.
type ProposalFeedback struct {
	ID                   string `json:"id,omitempty"`
	SuggestionID         string `json:"suggestion_id,omitempty"`
	Status               string `json:"status"`
	ReviewComment        string `json:"review_comment,omitempty"`
	ReviewedByEmployeeID string `json:"reviewed_by_employee_id,omitempty"`
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
		mode = domain.SyncModeIncremental
	}
	if mode != domain.SyncModeFullSnapshot && mode != domain.SyncModeIncremental {
		return nil, fmt.Errorf("%w: sync_mode must be full_snapshot or incremental", domain.ErrInvalid)
	}
	fullSnapshotReason := normalizeFullSnapshotReason(cmd.FullSnapshotReason)
	if mode == domain.SyncModeFullSnapshot && fullSnapshotReason == "" {
		return nil, fmt.Errorf("%w: full_snapshot_reason must be terminal_restaurant_changed or node_role_changed", domain.ErrInvalid)
	}
	if mode == domain.SyncModeIncremental && strings.TrimSpace(cmd.FullSnapshotReason) != "" {
		return nil, fmt.Errorf("%w: full_snapshot_reason is allowed only for full_snapshot", domain.ErrInvalid)
	}
	streams, err := streamsToApply(cmd)
	if err != nil {
		return nil, err
	}
	if mode == domain.SyncModeFullSnapshot && payloadRowCount(cmd, streams) == 0 {
		return nil, fmt.Errorf("%w: full_snapshot requires at least one master row", domain.ErrInvalid)
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
	if err := validatePayload(cmd, streams, now); err != nil {
		return nil, err
	}
	if mode == domain.SyncModeFullSnapshot && s.backupBeforeFullSnapshot != nil {
		req := BackupRequest{
			NodeDeviceID:       shared.EffectiveNodeDeviceID(cmd.CommandMeta),
			RestaurantID:       strings.TrimSpace(cmd.RestaurantID),
			Streams:            append([]domain.MasterDataStream(nil), streams...),
			CloudVersion:       cmd.CloudVersion,
			FullSnapshotReason: fullSnapshotReason,
			AppliedAt:          appliedAt,
			CloudUpdatedAt:     cloudUpdatedAt,
		}
		if err := s.backupBeforeFullSnapshot(ctx, req); err != nil {
			slog.ErrorContext(ctx, "master-data backup перед full_snapshot не создан",
				"operation", "sync.master_data",
				"action", "backup_before_data_load",
				"result", "failed",
				"error_code", "DB_BACKUP_FAILED",
				"node_device_id", req.NodeDeviceID,
				"restaurant_id", req.RestaurantID,
				"sync_mode", mode,
				"full_snapshot_reason", req.FullSnapshotReason,
				"cloud_version", req.CloudVersion,
				"stream_count", len(req.Streams),
				"internal_error", err.Error(),
			)
			return nil, fmt.Errorf("master-data ingest: backup before full_snapshot: %w", err)
		}
	}

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
			// Идемпотентно создаём системный зал и стол для counter sale.
			if err := s.repo.EnsureSystemFloor(ctx, v.ID, s.ids.NewID(), s.ids.NewID(), now); err != nil {
				return fmt.Errorf("ensure system floor for restaurant %s: %w", v.ID, err)
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
		eligibleIDs := make([]string, 0, len(cmd.Employees))
		for _, employee := range cmd.Employees {
			eligibleIDs = append(eligibleIDs, employee.ID)
		}
		if err := s.repo.DeactivateMissingMasterEmployees(ctx, cmd.RestaurantID, eligibleIDs, shared.DBTime(now)); err != nil {
			return err
		}
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
		for i := range cmd.Folders {
			v := normalizeCatalogFolder(cmd.Folders[i])
			if err := validateCatalogFolder(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterCatalogFolder(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.FolderParameters {
			v := normalizeFolderParameter(cmd.FolderParameters[i])
			if err := validateFolderParameter(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterFolderParameter(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.Tags {
			v := normalizeCatalogTag(cmd.Tags[i])
			if err := validateCatalogTag(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterCatalogTag(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.CatalogItems {
			v := normalizeCatalogItem(cmd.CatalogItems[i], now)
			if err := validateCatalogItem(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterCatalogItem(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.ItemTags {
			v := normalizeCatalogItemTag(cmd.ItemTags[i])
			if err := validateCatalogItemTag(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterCatalogItemTag(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.ModifierGroups {
			v := normalizeModifierGroup(cmd.ModifierGroups[i])
			if err := validateModifierGroup(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterModifierGroup(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.ModifierOptions {
			v := normalizeModifierOption(cmd.ModifierOptions[i])
			if err := validateModifierOption(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterModifierOption(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.ModifierBindings {
			v := normalizeModifierGroupBinding(cmd.ModifierBindings[i])
			if err := validateModifierGroupBinding(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterModifierGroupBinding(ctx, &v, meta); err != nil {
				return err
			}
		}
		appliedMenuItemModifierGroups := false
		if shouldApplyMenuItemModifierGroupsInCatalog(*cmd) {
			if err := s.applyMenuItemModifierGroups(ctx, cmd, meta); err != nil {
				return err
			}
			appliedMenuItemModifierGroups = true
		}
		counts[string(stream)] = len(cmd.Folders) + len(cmd.FolderParameters) + len(cmd.Tags) + len(cmd.CatalogItems) + len(cmd.ItemTags) + len(cmd.ModifierGroups) + len(cmd.ModifierOptions) + len(cmd.ModifierBindings)
		if appliedMenuItemModifierGroups {
			counts[string(stream)] += len(cmd.MenuItemModifierGroups)
		}
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
		if shouldApplyMenuItemModifierGroupsInMenu(*cmd) {
			if err := s.applyMenuItemModifierGroups(ctx, cmd, meta); err != nil {
				return err
			}
			counts[string(stream)] += len(cmd.MenuItemModifierGroups)
		}
	case domain.MasterDataStreamPricing:
		for i := range cmd.TaxProfiles {
			v := normalizeTaxProfile(cmd.TaxProfiles[i], now)
			if err := validateTaxProfile(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterTaxProfile(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.TaxRules {
			v := normalizeTaxRule(cmd.TaxRules[i], now)
			if err := validateTaxRule(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterTaxRule(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.ServiceChargeRules {
			v := normalizeServiceChargeRule(cmd.ServiceChargeRules[i], now)
			if err := validateServiceChargeRule(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterServiceChargeRule(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.PricingPolicies {
			v := normalizePricingPolicy(cmd.PricingPolicies[i], now)
			if err := validatePricingPolicy(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterPricingPolicy(ctx, &v, meta); err != nil {
				return err
			}
		}
		counts[string(stream)] = len(cmd.TaxProfiles) + len(cmd.TaxRules) + len(cmd.ServiceChargeRules) + len(cmd.PricingPolicies)
	case domain.MasterDataStreamRecipes:
		for i := range cmd.RecipeVersions {
			v := normalizeRecipeVersion(cmd.RecipeVersions[i], now)
			if err := validateRecipeVersion(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterRecipeVersion(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.RecipeLines {
			v := normalizeRecipeLine(cmd.RecipeLines[i], now)
			if err := validateRecipeLine(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterRecipeLine(ctx, &v, meta); err != nil {
				return err
			}
		}
		counts[string(stream)] = len(cmd.RecipeVersions) + len(cmd.RecipeLines)
	case domain.MasterDataStreamInventory:
		for i := range cmd.Warehouses {
			v := normalizeWarehouseReference(cmd.Warehouses[i], cmd.RestaurantID, now)
			if err := validateWarehouseReference(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterWarehouseReference(ctx, &v, meta); err != nil {
				return err
			}
		}
		for i := range cmd.StopListEntries {
			v := normalizeStopListEntry(cmd.StopListEntries[i], cmd.RestaurantID, now)
			if err := validateStopListEntry(v); err != nil {
				return err
			}
			if err := s.repo.UpsertMasterStopListEntry(ctx, &v, meta); err != nil {
				return err
			}
		}
		counts[string(stream)] = len(cmd.Warehouses) + len(cmd.StopListEntries)
	case domain.MasterDataStreamProposalFeedback:
		if err := s.applyProposalFeedback(ctx, kitchen.ProposalKindCatalog, cmd.CatalogSuggestions, cmd.CloudVersion, meta.CloudUpdatedAt, now); err != nil {
			return err
		}
		if err := s.applyProposalFeedback(ctx, kitchen.ProposalKindRecipe, cmd.RecipeSuggestions, cmd.CloudVersion, meta.CloudUpdatedAt, now); err != nil {
			return err
		}
		counts[string(stream)] = len(cmd.CatalogSuggestions) + len(cmd.RecipeSuggestions)
	default:
		return fmt.Errorf("%w: unsupported master data stream %q", domain.ErrInvalid, stream)
	}
	return nil
}

func validatePayload(cmd ApplyMasterDataCommand, streams []domain.MasterDataStream, now time.Time) error {
	for _, stream := range streams {
		switch stream {
		case domain.MasterDataStreamRestaurants:
			for i := range cmd.Restaurants {
				if err := validateRestaurant(normalizeRestaurant(cmd.Restaurants[i], now)); err != nil {
					return err
				}
			}
		case domain.MasterDataStreamDevices:
			for i := range cmd.Devices {
				if err := validateDevice(normalizeDevice(cmd.Devices[i], now)); err != nil {
					return err
				}
			}
		case domain.MasterDataStreamStaff:
			for i := range cmd.Roles {
				if err := validateRole(normalizeRole(cmd.Roles[i], now)); err != nil {
					return err
				}
			}
			for i := range cmd.Employees {
				if err := validateEmployee(normalizeEmployee(cmd.Employees[i], now)); err != nil {
					return err
				}
			}
		case domain.MasterDataStreamFloor:
			for i := range cmd.Halls {
				if err := validateHall(normalizeHall(cmd.Halls[i], now)); err != nil {
					return err
				}
			}
			for i := range cmd.Tables {
				if err := validateTable(normalizeTable(cmd.Tables[i], now)); err != nil {
					return err
				}
			}
		case domain.MasterDataStreamCatalog:
			for i := range cmd.Folders {
				if err := validateCatalogFolder(normalizeCatalogFolder(cmd.Folders[i])); err != nil {
					return err
				}
			}
			for i := range cmd.FolderParameters {
				if err := validateFolderParameter(normalizeFolderParameter(cmd.FolderParameters[i])); err != nil {
					return err
				}
			}
			for i := range cmd.Tags {
				if err := validateCatalogTag(normalizeCatalogTag(cmd.Tags[i])); err != nil {
					return err
				}
			}
			for i := range cmd.CatalogItems {
				if err := validateCatalogItem(normalizeCatalogItem(cmd.CatalogItems[i], now)); err != nil {
					return err
				}
			}
			for i := range cmd.ItemTags {
				if err := validateCatalogItemTag(normalizeCatalogItemTag(cmd.ItemTags[i])); err != nil {
					return err
				}
			}
			for i := range cmd.ModifierGroups {
				if err := validateModifierGroup(normalizeModifierGroup(cmd.ModifierGroups[i])); err != nil {
					return err
				}
			}
			for i := range cmd.ModifierOptions {
				if err := validateModifierOption(normalizeModifierOption(cmd.ModifierOptions[i])); err != nil {
					return err
				}
			}
			for i := range cmd.ModifierBindings {
				if err := validateModifierGroupBinding(normalizeModifierGroupBinding(cmd.ModifierBindings[i])); err != nil {
					return err
				}
			}
			for i := range cmd.MenuItemModifierGroups {
				if err := validateMenuItemModifierGroup(normalizeMenuItemModifierGroup(cmd.MenuItemModifierGroups[i])); err != nil {
					return err
				}
			}
		case domain.MasterDataStreamMenu:
			for i := range cmd.MenuItems {
				if err := validateMenuItem(normalizeMenuItem(cmd.MenuItems[i], now)); err != nil {
					return err
				}
			}
		case domain.MasterDataStreamPricing:
			for i := range cmd.TaxProfiles {
				if err := validateTaxProfile(normalizeTaxProfile(cmd.TaxProfiles[i], now)); err != nil {
					return err
				}
			}
			for i := range cmd.TaxRules {
				if err := validateTaxRule(normalizeTaxRule(cmd.TaxRules[i], now)); err != nil {
					return err
				}
			}
			for i := range cmd.ServiceChargeRules {
				if err := validateServiceChargeRule(normalizeServiceChargeRule(cmd.ServiceChargeRules[i], now)); err != nil {
					return err
				}
			}
			for i := range cmd.PricingPolicies {
				if err := validatePricingPolicy(normalizePricingPolicy(cmd.PricingPolicies[i], now)); err != nil {
					return err
				}
			}
		case domain.MasterDataStreamRecipes:
			for i := range cmd.RecipeVersions {
				if err := validateRecipeVersion(normalizeRecipeVersion(cmd.RecipeVersions[i], now)); err != nil {
					return err
				}
			}
			for i := range cmd.RecipeLines {
				if err := validateRecipeLine(normalizeRecipeLine(cmd.RecipeLines[i], now)); err != nil {
					return err
				}
			}
		case domain.MasterDataStreamInventory:
			for i := range cmd.Warehouses {
				if err := validateWarehouseReference(normalizeWarehouseReference(cmd.Warehouses[i], cmd.RestaurantID, now)); err != nil {
					return err
				}
			}
			for i := range cmd.StopListEntries {
				if err := validateStopListEntry(normalizeStopListEntry(cmd.StopListEntries[i], cmd.RestaurantID, now)); err != nil {
					return err
				}
			}
		case domain.MasterDataStreamProposalFeedback:
			if err := validateProposalFeedback(cmd.CatalogSuggestions); err != nil {
				return err
			}
			if err := validateProposalFeedback(cmd.RecipeSuggestions); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: unsupported master data stream %q", domain.ErrInvalid, stream)
		}
	}
	return nil
}

func payloadRowCount(cmd ApplyMasterDataCommand, streams []domain.MasterDataStream) int {
	total := 0
	for _, stream := range streams {
		switch stream {
		case domain.MasterDataStreamRestaurants:
			total += len(cmd.Restaurants)
		case domain.MasterDataStreamDevices:
			total += len(cmd.Devices)
		case domain.MasterDataStreamStaff:
			total += len(cmd.Roles) + len(cmd.Employees)
		case domain.MasterDataStreamFloor:
			total += len(cmd.Halls) + len(cmd.Tables)
		case domain.MasterDataStreamCatalog:
			total += len(cmd.Folders) + len(cmd.FolderParameters) + len(cmd.Tags) + len(cmd.CatalogItems) + len(cmd.ItemTags) + len(cmd.ModifierGroups) + len(cmd.ModifierOptions) + len(cmd.ModifierBindings) + len(cmd.MenuItemModifierGroups)
		case domain.MasterDataStreamMenu:
			total += len(cmd.MenuItems)
		case domain.MasterDataStreamPricing:
			total += len(cmd.TaxProfiles) + len(cmd.TaxRules) + len(cmd.ServiceChargeRules) + len(cmd.PricingPolicies)
		case domain.MasterDataStreamRecipes:
			total += len(cmd.RecipeVersions) + len(cmd.RecipeLines)
		case domain.MasterDataStreamInventory:
			total += len(cmd.Warehouses) + len(cmd.StopListEntries)
		case domain.MasterDataStreamProposalFeedback:
			total += len(cmd.CatalogSuggestions) + len(cmd.RecipeSuggestions)
		}
	}
	return total
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

func (s *Service) applyMenuItemModifierGroups(ctx context.Context, cmd *ApplyMasterDataCommand, meta domain.MasterRecordSyncMeta) error {
	for i := range cmd.MenuItemModifierGroups {
		v := normalizeMenuItemModifierGroup(cmd.MenuItemModifierGroups[i])
		if err := validateMenuItemModifierGroup(v); err != nil {
			return err
		}
		if err := s.repo.UpsertMasterMenuItemModifierGroup(ctx, &v, meta); err != nil {
			return err
		}
	}
	return nil
}

func shouldApplyMenuItemModifierGroupsInCatalog(cmd ApplyMasterDataCommand) bool {
	if len(cmd.MenuItemModifierGroups) == 0 {
		return false
	}
	return cmd.StreamName == domain.MasterDataStreamCatalog || (cmd.StreamName == "" && len(cmd.MenuItems) == 0)
}

func shouldApplyMenuItemModifierGroupsInMenu(cmd ApplyMasterDataCommand) bool {
	if len(cmd.MenuItemModifierGroups) == 0 {
		return false
	}
	return cmd.StreamName == domain.MasterDataStreamMenu || (cmd.StreamName == "" && len(cmd.MenuItems) > 0)
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
	if len(cmd.Folders) > 0 || len(cmd.FolderParameters) > 0 || len(cmd.Tags) > 0 || len(cmd.CatalogItems) > 0 || len(cmd.ItemTags) > 0 || len(cmd.ModifierGroups) > 0 || len(cmd.ModifierOptions) > 0 || len(cmd.ModifierBindings) > 0 || len(cmd.MenuItemModifierGroups) > 0 {
		streams = append(streams, domain.MasterDataStreamCatalog)
	}
	if len(cmd.MenuItems) > 0 {
		streams = append(streams, domain.MasterDataStreamMenu)
	}
	if len(cmd.TaxProfiles) > 0 || len(cmd.TaxRules) > 0 || len(cmd.ServiceChargeRules) > 0 || len(cmd.PricingPolicies) > 0 {
		streams = append(streams, domain.MasterDataStreamPricing)
	}
	if len(cmd.RecipeVersions) > 0 || len(cmd.RecipeLines) > 0 {
		streams = append(streams, domain.MasterDataStreamRecipes)
	}
	if len(cmd.Warehouses) > 0 || len(cmd.StopListEntries) > 0 {
		streams = append(streams, domain.MasterDataStreamInventory)
	}
	if len(cmd.CatalogSuggestions) > 0 || len(cmd.RecipeSuggestions) > 0 {
		streams = append(streams, domain.MasterDataStreamProposalFeedback)
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
		domain.MasterDataStreamMenu,
		domain.MasterDataStreamPricing,
		domain.MasterDataStreamRecipes,
		domain.MasterDataStreamInventory,
		domain.MasterDataStreamProposalFeedback:
		return true
	default:
		return false
	}
}

func (s *Service) applyProposalFeedback(ctx context.Context, kind kitchen.ProposalKind, items []ProposalFeedback, cloudVersion int64, cloudUpdatedAt *string, now time.Time) error {
	cloudUpdated := shared.DBTime(now)
	if cloudUpdatedAt != nil && strings.TrimSpace(*cloudUpdatedAt) != "" {
		cloudUpdated = strings.TrimSpace(*cloudUpdatedAt)
	}
	updatedAt := shared.DBTime(now)
	for i := range items {
		status, err := mapProposalFeedbackStatus(items[i].Status)
		if err != nil {
			return err
		}
		suggestionID := strings.TrimSpace(items[i].SuggestionID)
		if suggestionID == "" {
			suggestionID = strings.TrimSpace(items[i].ID)
		}
		if suggestionID == "" {
			return fmt.Errorf("%w: proposal_feedback suggestion_id is required", domain.ErrInvalid)
		}
		if err := s.repo.ApplyKitchenProposalFeedback(ctx, kind, suggestionID, status, cloudVersion, cloudUpdated, updatedAt); err != nil {
			return err
		}
	}
	return nil
}

func validateProposalFeedback(items []ProposalFeedback) error {
	for i := range items {
		if _, err := mapProposalFeedbackStatus(items[i].Status); err != nil {
			return err
		}
		if strings.TrimSpace(items[i].SuggestionID) == "" && strings.TrimSpace(items[i].ID) == "" {
			return fmt.Errorf("%w: proposal_feedback suggestion_id is required", domain.ErrInvalid)
		}
	}
	return nil
}

func mapProposalFeedbackStatus(status string) (kitchen.ProposalStatus, error) {
	switch strings.TrimSpace(status) {
	case "pending":
		return kitchen.ProposalSynced, nil
	case string(kitchen.ProposalApproved):
		return kitchen.ProposalApproved, nil
	case string(kitchen.ProposalRejected):
		return kitchen.ProposalRejected, nil
	case string(kitchen.ProposalChangesRequested):
		return kitchen.ProposalChangesRequested, nil
	default:
		return "", fmt.Errorf("%w: unsupported proposal_feedback status %q", domain.ErrInvalid, status)
	}
}

func normalizeFullSnapshotReason(reason string) string {
	switch strings.TrimSpace(strings.ToLower(reason)) {
	case fullSnapshotReasonTerminalRestaurantChanged:
		return fullSnapshotReasonTerminalRestaurantChanged
	case fullSnapshotReasonNodeRoleChanged:
		return fullSnapshotReasonNodeRoleChanged
	default:
		return ""
	}
}

func normalizeRestaurant(v domain.Restaurant, now time.Time) domain.Restaurant {
	v.ID = strings.TrimSpace(v.ID)
	v.Name = strings.TrimSpace(v.Name)
	v.Timezone = strings.TrimSpace(v.Timezone)
	v.Currency = strings.ToUpper(strings.TrimSpace(v.Currency))
	if v.BusinessDayMode == "" {
		v.BusinessDayMode = shared.DefaultBusinessDayMode
	}
	if strings.TrimSpace(v.BusinessDayBoundaryLocalTime) == "" {
		v.BusinessDayBoundaryLocalTime = shared.DefaultBusinessDayBoundaryLocalTime
	} else {
		v.BusinessDayBoundaryLocalTime = strings.TrimSpace(v.BusinessDayBoundaryLocalTime)
	}
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
	if v.FolderID != nil {
		folderID := strings.TrimSpace(*v.FolderID)
		if folderID == "" {
			v.FolderID = nil
		} else {
			v.FolderID = &folderID
		}
	}
	v.Name = strings.TrimSpace(v.Name)
	v.SKU = strings.TrimSpace(v.SKU)
	v.BaseUnit = strings.TrimSpace(v.BaseUnit)
	v.KitchenType = strings.TrimSpace(v.KitchenType)
	v.AccountingCategory = strings.TrimSpace(v.AccountingCategory)
	v.ValidityMode = strings.TrimSpace(v.ValidityMode)
	// single_unit_per_line автоматически следует за qr_confirmation_enabled при доставке из Cloud
	if v.QRConfirmationEnabled {
		v.SingleUnitPerLine = true
	}
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeCatalogFolder(v domain.CatalogFolder) domain.CatalogFolder {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	if v.ParentID != nil {
		parentID := strings.TrimSpace(*v.ParentID)
		if parentID == "" {
			v.ParentID = nil
		} else {
			v.ParentID = &parentID
		}
	}
	v.Name = strings.TrimSpace(v.Name)
	return v
}

func normalizeFolderParameter(v domain.FolderParameter) domain.FolderParameter {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.FolderID = strings.TrimSpace(v.FolderID)
	v.ParameterKey = strings.TrimSpace(v.ParameterKey)
	v.ValueType = strings.TrimSpace(v.ValueType)
	v.ValueJSON = strings.TrimSpace(v.ValueJSON)
	return v
}

func normalizeCatalogTag(v domain.CatalogTag) domain.CatalogTag {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.Name = strings.TrimSpace(v.Name)
	v.Code = strings.TrimSpace(v.Code)
	return v
}

func normalizeCatalogItemTag(v domain.CatalogItemTag) domain.CatalogItemTag {
	v.CatalogItemID = strings.TrimSpace(v.CatalogItemID)
	v.TagID = strings.TrimSpace(v.TagID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	return v
}

func normalizeModifierGroup(v domain.ModifierGroup) domain.ModifierGroup {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.Name = strings.TrimSpace(v.Name)
	return v
}

func normalizeModifierOption(v domain.ModifierOption) domain.ModifierOption {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.ModifierGroupID = strings.TrimSpace(v.ModifierGroupID)
	v.LinkedCatalogItemID = strings.TrimSpace(v.LinkedCatalogItemID)
	v.Name = strings.TrimSpace(v.Name)
	return v
}

func normalizeModifierGroupBinding(v domain.ModifierGroupBinding) domain.ModifierGroupBinding {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.ModifierGroupID = strings.TrimSpace(v.ModifierGroupID)
	v.TargetType = domain.ModifierTargetType(strings.TrimSpace(string(v.TargetType)))
	v.TargetID = strings.TrimSpace(v.TargetID)
	return v
}

func normalizeMenuItemModifierGroup(v domain.MenuItemModifierGroup) domain.MenuItemModifierGroup {
	v.MenuItemID = strings.TrimSpace(v.MenuItemID)
	v.ModifierGroupID = strings.TrimSpace(v.ModifierGroupID)
	return v
}

func normalizeMenuItem(v domain.MenuItem, now time.Time) domain.MenuItem {
	v.ID = strings.TrimSpace(v.ID)
	v.CatalogItemID = strings.TrimSpace(v.CatalogItemID)
	v.CategoryID = strings.TrimSpace(v.CategoryID)
	v.TagID = strings.TrimSpace(v.TagID)
	v.Name = strings.TrimSpace(v.Name)
	v.Currency = strings.ToUpper(strings.TrimSpace(v.Currency))
	switch strings.TrimSpace(v.RuntimeStatus) {
	case "unavailable", "hidden":
	default:
		v.RuntimeStatus = "available"
	}
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeTaxProfile(v domain.TaxProfile, now time.Time) domain.TaxProfile {
	v.ID = strings.TrimSpace(v.ID)
	v.Name = strings.TrimSpace(v.Name)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeTaxRule(v domain.TaxRule, now time.Time) domain.TaxRule {
	v.ID = strings.TrimSpace(v.ID)
	v.TaxProfileID = strings.TrimSpace(v.TaxProfileID)
	v.Name = strings.TrimSpace(v.Name)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeServiceChargeRule(v domain.ServiceChargeRule, now time.Time) domain.ServiceChargeRule {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.Name = strings.TrimSpace(v.Name)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizePricingPolicy(v domain.PricingPolicy, now time.Time) domain.PricingPolicy {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	v.Name = strings.TrimSpace(v.Name)
	v.RequiresPermission = strings.TrimSpace(v.RequiresPermission)
	if v.Kind == domain.PricingPolicySurcharge {
		v.Scope = domain.DiscountScopeOrder
	}
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeRecipeVersion(v domain.RecipeVersion, now time.Time) domain.RecipeVersion {
	v.ID = strings.TrimSpace(v.ID)
	v.DishCatalogItemID = strings.TrimSpace(v.DishCatalogItemID)
	v.Name = strings.TrimSpace(v.Name)
	v.YieldUnit = strings.TrimSpace(v.YieldUnit)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeRecipeLine(v domain.RecipeLine, now time.Time) domain.RecipeLine {
	v.ID = strings.TrimSpace(v.ID)
	v.RecipeVersionID = strings.TrimSpace(v.RecipeVersionID)
	v.CatalogItemID = strings.TrimSpace(v.CatalogItemID)
	v.Unit = strings.TrimSpace(v.Unit)
	v.CreatedAt = defaultTime(v.CreatedAt, now)
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeStopListEntry(v domain.StopListEntry, fallbackRestaurantID string, now time.Time) domain.StopListEntry {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	if v.RestaurantID == "" {
		v.RestaurantID = strings.TrimSpace(fallbackRestaurantID)
	}
	v.CatalogItemID = strings.TrimSpace(v.CatalogItemID)
	v.Source = strings.TrimSpace(v.Source)
	if v.Source == "" {
		v.Source = "cloud"
	}
	v.UpdatedAt = defaultTime(v.UpdatedAt, now)
	return v
}

func normalizeWarehouseReference(v domain.WarehouseReference, fallbackRestaurantID string, now time.Time) domain.WarehouseReference {
	v.ID = strings.TrimSpace(v.ID)
	v.RestaurantID = strings.TrimSpace(v.RestaurantID)
	if v.RestaurantID == "" {
		v.RestaurantID = strings.TrimSpace(fallbackRestaurantID)
	}
	v.Name = strings.TrimSpace(v.Name)
	v.Kind = strings.TrimSpace(v.Kind)
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
	if _, err := shared.ValidateCurrencyCode(v.Currency); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	if _, _, err := shared.NormalizeBusinessDayConfig(v.BusinessDayMode, v.BusinessDayBoundaryLocalTime); err != nil {
		return err
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
	if err := shared.ValidatePermissionsJSON(v.PermissionsJSON); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalid, err)
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
	case domain.CatalogItemDish, domain.CatalogItemGood, domain.CatalogItemSemiFinished, domain.CatalogItemService:
	default:
		return fmt.Errorf("%w: unsupported catalog item type", domain.ErrInvalid)
	}
	if v.QRConfirmationEnabled {
		switch v.ValidityMode {
		case "cash_session", "business_date", "absolute_date":
		default:
			return fmt.Errorf("%w: validity_mode is required when qr_confirmation_enabled is true", domain.ErrInvalid)
		}
		if v.ValidityMode == "absolute_date" && v.ValidityExpiresAt == nil {
			return fmt.Errorf("%w: validity_expires_at is required for absolute_date validity_mode", domain.ErrInvalid)
		}
	}
	return nil
}

func validateCatalogFolder(v domain.CatalogFolder) error {
	if v.ID == "" || v.Name == "" {
		return fmt.Errorf("%w: catalog folder id and name are required", domain.ErrInvalid)
	}
	return nil
}

func validateFolderParameter(v domain.FolderParameter) error {
	if v.ID == "" || v.RestaurantID == "" || v.FolderID == "" || v.ParameterKey == "" || v.ValueType == "" || !json.Valid([]byte(v.ValueJSON)) {
		return fmt.Errorf("%w: folder parameter identity and valid value_json are required", domain.ErrInvalid)
	}
	return nil
}

func validateCatalogTag(v domain.CatalogTag) error {
	if v.ID == "" || v.Name == "" || v.Code == "" {
		return fmt.Errorf("%w: catalog tag id, name and code are required", domain.ErrInvalid)
	}
	return nil
}

func validateCatalogItemTag(v domain.CatalogItemTag) error {
	if v.CatalogItemID == "" || v.TagID == "" {
		return fmt.Errorf("%w: catalog item tag ids are required", domain.ErrInvalid)
	}
	return nil
}

func validateModifierGroup(v domain.ModifierGroup) error {
	if v.ID == "" || v.RestaurantID == "" || v.Name == "" || v.MinCount < 0 || v.MaxCount < 0 || (v.MaxCount > 0 && v.MaxCount < v.MinCount) {
		return fmt.Errorf("%w: modifier group identity and counts are required", domain.ErrInvalid)
	}
	return nil
}

func validateModifierOption(v domain.ModifierOption) error {
	if v.ID == "" || v.RestaurantID == "" || v.ModifierGroupID == "" || v.Name == "" || v.PriceMinor < 0 {
		return fmt.Errorf("%w: modifier option identity and non-negative price are required", domain.ErrInvalid)
	}
	return nil
}

func validateModifierGroupBinding(v domain.ModifierGroupBinding) error {
	if v.ID == "" || v.RestaurantID == "" || v.ModifierGroupID == "" || v.TargetID == "" {
		return fmt.Errorf("%w: modifier binding identity is required", domain.ErrInvalid)
	}
	switch v.TargetType {
	case domain.ModifierTargetMenuItem, domain.ModifierTargetCatalogItem, domain.ModifierTargetFolder, domain.ModifierTargetTag:
		return nil
	default:
		return fmt.Errorf("%w: unsupported modifier binding target_type", domain.ErrInvalid)
	}
}

func validateMenuItemModifierGroup(v domain.MenuItemModifierGroup) error {
	if v.MenuItemID == "" || v.ModifierGroupID == "" {
		return fmt.Errorf("%w: menu item modifier group ids are required", domain.ErrInvalid)
	}
	return nil
}

func validateMenuItem(v domain.MenuItem) error {
	if v.ID == "" || v.CatalogItemID == "" || v.Name == "" || v.Currency == "" || v.Price < 0 {
		return fmt.Errorf("%w: menu item id, catalog_item_id, name, currency and non-negative price are required", domain.ErrInvalid)
	}
	if _, err := shared.ValidateCurrencyCode(v.Currency); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	return nil
}

func validateTaxProfile(v domain.TaxProfile) error {
	if v.ID == "" || v.Name == "" {
		return fmt.Errorf("%w: tax profile id and name are required", domain.ErrInvalid)
	}
	return nil
}

func validateTaxRule(v domain.TaxRule) error {
	if v.ID == "" || v.TaxProfileID == "" || v.Name == "" {
		return fmt.Errorf("%w: tax rule id, tax_profile_id and name are required", domain.ErrInvalid)
	}
	switch v.Kind {
	case domain.TaxRulePercentage:
		if v.RateBasisPoints <= 0 {
			return fmt.Errorf("%w: percentage tax rule requires positive rate_basis_points", domain.ErrInvalid)
		}
	case domain.TaxRuleFixed:
		if v.AmountMinor <= 0 {
			return fmt.Errorf("%w: fixed tax rule requires positive amount_minor", domain.ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: unsupported tax rule kind", domain.ErrInvalid)
	}
	switch v.Mode {
	case domain.TaxModeInclusive, domain.TaxModeExclusive:
	default:
		return fmt.Errorf("%w: unsupported tax mode", domain.ErrInvalid)
	}
	if v.RateBasisPoints < 0 || v.AmountMinor < 0 {
		return fmt.Errorf("%w: tax rule amounts must be non-negative", domain.ErrInvalid)
	}
	return nil
}

func validateServiceChargeRule(v domain.ServiceChargeRule) error {
	if v.ID == "" || v.RestaurantID == "" || v.Name == "" {
		return fmt.Errorf("%w: service charge rule id, restaurant_id and name are required", domain.ErrInvalid)
	}
	switch v.Kind {
	case domain.SurchargeServiceCharge, domain.SurchargePB1ServiceFee, domain.SurchargeManual:
	default:
		return fmt.Errorf("%w: unsupported service charge kind", domain.ErrInvalid)
	}
	switch v.AmountKind {
	case domain.AmountPercentage:
		if v.ValueBasisPoints <= 0 {
			return fmt.Errorf("%w: percentage service charge rule requires positive value_basis_points", domain.ErrInvalid)
		}
	case domain.AmountFixed:
		if v.AmountMinor <= 0 {
			return fmt.Errorf("%w: fixed service charge rule requires positive amount_minor", domain.ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: unsupported service charge amount_kind", domain.ErrInvalid)
	}
	if v.AmountMinor < 0 || v.ValueBasisPoints < 0 {
		return fmt.Errorf("%w: service charge amounts must be non-negative", domain.ErrInvalid)
	}
	return nil
}

func validatePricingPolicy(v domain.PricingPolicy) error {
	if v.ID == "" || v.RestaurantID == "" || v.Name == "" || v.ApplicationIndex <= 0 {
		return fmt.Errorf("%w: pricing policy identity and application_index are required", domain.ErrInvalid)
	}
	switch v.Kind {
	case domain.PricingPolicyDiscount, domain.PricingPolicySurcharge:
	default:
		return fmt.Errorf("%w: unsupported pricing policy kind", domain.ErrInvalid)
	}
	switch v.Scope {
	case domain.DiscountScopeLine, domain.DiscountScopeOrder:
	default:
		return fmt.Errorf("%w: unsupported pricing policy scope", domain.ErrInvalid)
	}
	if v.Kind == domain.PricingPolicySurcharge && v.Scope != domain.DiscountScopeOrder {
		return fmt.Errorf("%w: surcharge pricing policy must be order-scoped", domain.ErrInvalid)
	}
	switch v.AmountKind {
	case domain.AmountPercentage:
		if v.ValueBasisPoints <= 0 {
			return fmt.Errorf("%w: percentage pricing policy requires positive value_basis_points", domain.ErrInvalid)
		}
	case domain.AmountFixed:
		if v.AmountMinor < 0 {
			return fmt.Errorf("%w: fixed pricing policy requires non-negative amount_minor", domain.ErrInvalid)
		}
	default:
		return fmt.Errorf("%w: unsupported pricing policy amount_kind", domain.ErrInvalid)
	}
	return nil
}

func validateRecipeVersion(v domain.RecipeVersion) error {
	if v.ID == "" || v.DishCatalogItemID == "" || v.Name == "" || v.Version <= 0 || v.YieldQuantity <= 0 || v.YieldUnit == "" {
		return fmt.Errorf("%w: recipe version identity, owner, version, yield and unit are required", domain.ErrInvalid)
	}
	switch v.Status {
	case domain.RecipeVersionDraft, domain.RecipeVersionActive, domain.RecipeVersionArchived:
		return nil
	default:
		return fmt.Errorf("%w: unsupported recipe version status", domain.ErrInvalid)
	}
}

func validateRecipeLine(v domain.RecipeLine) error {
	if v.ID == "" || v.RecipeVersionID == "" || v.CatalogItemID == "" || v.Quantity <= 0 || v.Unit == "" || v.LossPercent < 0 || v.LossPercent > 100 {
		return fmt.Errorf("%w: recipe line identity, component, quantity, unit and loss percent are required", domain.ErrInvalid)
	}
	return nil
}

func validateStopListEntry(v domain.StopListEntry) error {
	if v.ID == "" || v.RestaurantID == "" || v.CatalogItemID == "" || v.Source == "" {
		return fmt.Errorf("%w: stop-list id, restaurant_id, catalog_item_id and source are required", domain.ErrInvalid)
	}
	if v.AvailableQuantity != nil && *v.AvailableQuantity < 0 {
		return fmt.Errorf("%w: stop-list available_quantity must be non-negative", domain.ErrInvalid)
	}
	return nil
}

func validateWarehouseReference(v domain.WarehouseReference) error {
	if v.ID == "" || v.RestaurantID == "" || v.Name == "" || v.Kind == "" {
		return fmt.Errorf("%w: warehouse id, restaurant_id, name and kind are required", domain.ErrInvalid)
	}
	return nil
}
