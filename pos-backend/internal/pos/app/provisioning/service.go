package provisioning

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	appdevice "pos-backend/internal/pos/app/device"
	appmastersync "pos-backend/internal/pos/app/mastersync"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
)

type CloudClient interface {
	RegisterDevice(context.Context, string, CloudRegisterRequest) (CloudRegisterResponse, error)
	AssignmentStatus(context.Context, string, string) (CloudAssignmentStatus, error)
	DownloadSnapshot(context.Context, string) (appmastersync.ApplyMasterDataCommand, error)
}

type LicenseClient interface {
	Resolve(context.Context, string, LicenseResolveRequest) (LicenseResolveResponse, error)
}

type CloudRegisterRequest struct {
	NodeDeviceID string `json:"node_device_id"`
	DisplayName  string `json:"display_name"`
	AppVersion   string `json:"app_version"`
}

type CloudRegisterResponse struct {
	NodeDeviceID string `json:"node_device_id"`
	Status       string `json:"status"`
	RestaurantID string `json:"restaurant_id,omitempty"`
}

type Credentials struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type CloudAssignmentStatus struct {
	NodeDeviceID string       `json:"node_device_id"`
	Status       string       `json:"status"`
	RestaurantID string       `json:"restaurant_id,omitempty"`
	CloudURL     string       `json:"cloud_url,omitempty"`
	SnapshotURL  string       `json:"snapshot_url,omitempty"`
	Credentials  *Credentials `json:"credentials,omitempty"`
}

type LicenseResolveRequest struct {
	PairingCode  string `json:"pairing_code"`
	NodeDeviceID string `json:"node_device_id"`
}

type LicenseResolveResponse struct {
	CloudURL     string      `json:"cloud_url"`
	RestaurantID string      `json:"restaurant_id"`
	NodeDeviceID string      `json:"node_device_id"`
	Credentials  Credentials `json:"credentials"`
}

type Service struct {
	repo       ports.Repository
	tx         txmanager.Manager
	ids        idgen.Generator
	clock      clock.Clock
	cloud      CloudClient
	license    LicenseClient
	cloudURL   string
	licenseURL string
	apply      func(context.Context, appmastersync.ApplyMasterDataCommand) (*appmastersync.ApplyMasterDataResult, error)
	pair       func(context.Context, appdevice.PairEdgeNodeCommand) (*domain.EdgeNodeIdentity, error)
}

type Options struct {
	CloudURL   string
	LicenseURL string
	Cloud      CloudClient
	License    LicenseClient
	Apply      func(context.Context, appmastersync.ApplyMasterDataCommand) (*appmastersync.ApplyMasterDataResult, error)
	Pair       func(context.Context, appdevice.PairEdgeNodeCommand) (*domain.EdgeNodeIdentity, error)
}

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock, options Options) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock, cloud: options.Cloud, license: options.License, cloudURL: strings.TrimRight(strings.TrimSpace(options.CloudURL), "/"), licenseURL: strings.TrimRight(strings.TrimSpace(options.LicenseURL), "/"), apply: options.Apply, pair: options.Pair}
}

type RegisterCloudCommand struct {
	CloudURL    string `json:"cloud_url"`
	DisplayName string `json:"display_name"`
	AppVersion  string `json:"app_version"`
}

type PairViaLicenseCommand struct {
	PairingCode string `json:"pairing_code"`
}

func (s *Service) GetStatus(ctx context.Context) (domain.ProvisioningStatusView, error) {
	state, err := s.ensureState(ctx)
	if err != nil {
		return domain.ProvisioningStatusView{}, err
	}
	paired := false
	if identity, err := s.repo.GetEdgeNodeIdentity(ctx); err == nil && identity.Status == domain.EdgeNodePaired {
		paired = true
		state.Status = domain.ProvisioningPaired
		state.RestaurantID = identity.RestaurantID
		_ = s.repo.UpsertEdgeProvisioningState(ctx, state)
	}
	return view(state, paired), nil
}

func (s *Service) RegisterCloud(ctx context.Context, cmd RegisterCloudCommand) (domain.ProvisioningStatusView, error) {
	state, err := s.ensureState(ctx)
	if err != nil {
		return domain.ProvisioningStatusView{}, err
	}
	cloudURL := strings.TrimRight(strings.TrimSpace(cmd.CloudURL), "/")
	if cloudURL == "" {
		cloudURL = s.cloudURL
	}
	if cloudURL == "" || s.cloud == nil {
		return domain.ProvisioningStatusView{}, fmt.Errorf("%w: cloud_url is required", domain.ErrInvalid)
	}
	displayName := strings.TrimSpace(cmd.DisplayName)
	if displayName == "" {
		displayName = "POS Terminal"
	}
	if _, err := s.cloud.RegisterDevice(ctx, cloudURL, CloudRegisterRequest{NodeDeviceID: state.NodeDeviceID, DisplayName: displayName, AppVersion: strings.TrimSpace(cmd.AppVersion)}); err != nil {
		state.Status = domain.ProvisioningError
		state.LastError = safeError(err)
		state.UpdatedAt = s.clock.Now()
		_ = s.repo.UpsertEdgeProvisioningState(ctx, state)
		return domain.ProvisioningStatusView{}, err
	}
	state.CloudURL = cloudURL
	state.Status = domain.ProvisioningUnpairedRegistered
	state.LastError = ""
	state.UpdatedAt = s.clock.Now()
	if err := s.repo.UpsertEdgeProvisioningState(ctx, state); err != nil {
		return domain.ProvisioningStatusView{}, err
	}
	return s.PollAssignment(ctx)
}

func (s *Service) PollAssignment(ctx context.Context) (domain.ProvisioningStatusView, error) {
	state, err := s.ensureState(ctx)
	if err != nil {
		return domain.ProvisioningStatusView{}, err
	}
	if state.CloudURL == "" || s.cloud == nil {
		return view(state, false), nil
	}
	assignment, err := s.cloud.AssignmentStatus(ctx, state.CloudURL, state.NodeDeviceID)
	if err != nil {
		return view(state, false), nil
	}
	if assignment.Status != "assigned" {
		state.Status = domain.ProvisioningUnpairedRegistered
		state.UpdatedAt = s.clock.Now()
		_ = s.repo.UpsertEdgeProvisioningState(ctx, state)
		return view(state, false), nil
	}
	if err := s.finishAssigned(ctx, state, assignment.CloudURL, assignment.RestaurantID, assignment.SnapshotURL, assignment.Credentials); err != nil {
		return domain.ProvisioningStatusView{}, err
	}
	state, _ = s.repo.GetEdgeProvisioningState(ctx)
	return view(state, true), nil
}

func (s *Service) PairViaLicense(ctx context.Context, cmd PairViaLicenseCommand) (domain.ProvisioningStatusView, error) {
	state, err := s.ensureState(ctx)
	if err != nil {
		return domain.ProvisioningStatusView{}, err
	}
	if s.license == nil || s.licenseURL == "" {
		return domain.ProvisioningStatusView{}, fmt.Errorf("%w: license server url is required", domain.ErrInvalid)
	}
	resp, err := s.license.Resolve(ctx, s.licenseURL, LicenseResolveRequest{PairingCode: strings.TrimSpace(cmd.PairingCode), NodeDeviceID: state.NodeDeviceID})
	if err != nil {
		return domain.ProvisioningStatusView{}, err
	}
	if resp.NodeDeviceID != "" && resp.NodeDeviceID != state.NodeDeviceID {
		return domain.ProvisioningStatusView{}, fmt.Errorf("%w: license node_device_id conflicts with local identity", domain.ErrConflict)
	}
	if err := s.finishAssigned(ctx, state, resp.CloudURL, resp.RestaurantID, "/api/v1/restaurants/"+resp.RestaurantID+"/edge-nodes/"+state.NodeDeviceID+"/master-data/snapshot", &resp.Credentials); err != nil {
		return domain.ProvisioningStatusView{}, err
	}
	state, _ = s.repo.GetEdgeProvisioningState(ctx)
	return view(state, true), nil
}

func (s *Service) finishAssigned(ctx context.Context, state *domain.EdgeProvisioningState, cloudURL, restaurantID, snapshotURL string, credentials *Credentials) error {
	now := s.clock.Now()
	state.Status = domain.ProvisioningAssignedDownloadingSnapshot
	state.CloudURL = strings.TrimRight(strings.TrimSpace(cloudURL), "/")
	state.RestaurantID = strings.TrimSpace(restaurantID)
	if credentials != nil {
		state.CredentialsType = credentials.Type
		state.CredentialsToken = credentials.Token
	}
	state.UpdatedAt = now
	if err := s.repo.UpsertEdgeProvisioningState(ctx, state); err != nil {
		return err
	}
	snapshot, err := s.cloud.DownloadSnapshot(ctx, state.CloudURL+snapshotURL)
	if err != nil {
		state.Status = domain.ProvisioningError
		state.LastError = safeError(err)
		state.UpdatedAt = s.clock.Now()
		_ = s.repo.UpsertEdgeProvisioningState(ctx, state)
		return err
	}
	snapshot.CommandMeta = shared.CommandMeta{NodeDeviceID: state.NodeDeviceID, DeviceID: state.NodeDeviceID, Origin: domain.OriginCloudSync}
	if _, err := s.apply(ctx, snapshot); err != nil {
		return err
	}
	if _, err := s.pair(ctx, appdevice.PairEdgeNodeCommand{PairingCode: "MHPOS:" + state.RestaurantID + ":" + state.NodeDeviceID}); err != nil {
		return err
	}
	state.Status = domain.ProvisioningPaired
	state.LastError = ""
	state.UpdatedAt = s.clock.Now()
	return s.repo.UpsertEdgeProvisioningState(ctx, state)
}

func (s *Service) ensureState(ctx context.Context) (*domain.EdgeProvisioningState, error) {
	state, err := s.repo.GetEdgeProvisioningState(ctx)
	if err == nil {
		return state, nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	now := s.clock.Now()
	state = &domain.EdgeProvisioningState{ID: "local", NodeDeviceID: s.ids.NewID(), CloudURL: s.cloudURL, LicenseURL: s.licenseURL, Status: domain.ProvisioningNotConfigured, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.UpsertEdgeProvisioningState(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

func view(state *domain.EdgeProvisioningState, paired bool) domain.ProvisioningStatusView {
	return domain.ProvisioningStatusView{NodeDeviceID: state.NodeDeviceID, CloudURL: state.CloudURL, LicenseURL: state.LicenseURL, RestaurantID: state.RestaurantID, Status: state.Status, Paired: paired, LastError: state.LastError}
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	if len(text) > 240 {
		return text[:240]
	}
	return text
}
