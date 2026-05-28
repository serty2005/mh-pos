package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	masterapp "cloud-backend/internal/masterdata/app"
	masterdomain "cloud-backend/internal/masterdata/domain"
	"cloud-backend/internal/platform/clock"
	"cloud-backend/internal/provisioning/domain"
)

type Repository interface {
	RegisterUnassigned(context.Context, domain.UnassignedEdgeNode) (domain.UnassignedEdgeNode, error)
	ListUnassigned(context.Context) ([]domain.UnassignedEdgeNode, error)
	UpsertEdgeNode(context.Context, domain.EdgeNode) (domain.EdgeNode, error)
	GetEdgeNode(context.Context, string) (domain.EdgeNode, error)
	MarkUnassignedAssigned(context.Context, string, string, time.Time) error
	CreatePairingCode(context.Context, domain.PairingCode) (domain.PairingCode, error)
}

type IDGenerator interface {
	NewID() string
}

type LicenseClient interface {
	RegisterPairingCode(context.Context, LicensePairingPayload) error
}

type LicensePairingPayload struct {
	PairingCode  string             `json:"pairing_code"`
	CloudURL     string             `json:"cloud_url"`
	RestaurantID string             `json:"restaurant_id"`
	NodeDeviceID string             `json:"node_device_id"`
	Credentials  domain.Credentials `json:"credentials"`
	ExpiresAt    time.Time          `json:"expires_at"`
}

type Service struct {
	repo     Repository
	master   *masterapp.Service
	clock    clock.Clock
	ids      IDGenerator
	cloudURL string
	license  LicenseClient
}

func NewService(repo Repository, master *masterapp.Service, clock clock.Clock, ids IDGenerator, cloudURL string, license LicenseClient) *Service {
	return &Service{repo: repo, master: master, clock: clock, ids: ids, cloudURL: strings.TrimRight(strings.TrimSpace(cloudURL), "/"), license: license}
}

type RegisterDeviceCommand struct {
	NodeDeviceID string `json:"node_device_id"`
	DisplayName  string `json:"display_name"`
	AppVersion   string `json:"app_version"`
}

type RegisterDeviceResult struct {
	NodeDeviceID string `json:"node_device_id"`
	Status       string `json:"status"`
	RestaurantID string `json:"restaurant_id,omitempty"`
}

type AssignDeviceResult struct {
	NodeDeviceID string `json:"node_device_id"`
	RestaurantID string `json:"restaurant_id"`
	Status       string `json:"status"`
	SnapshotURL  string `json:"snapshot_url"`
}

type AssignmentStatus struct {
	NodeDeviceID string              `json:"node_device_id"`
	Status       string              `json:"status"`
	RestaurantID string              `json:"restaurant_id,omitempty"`
	CloudURL     string              `json:"cloud_url,omitempty"`
	SnapshotURL  string              `json:"snapshot_url,omitempty"`
	Credentials  *domain.Credentials `json:"credentials,omitempty"`
}

type GeneratePairingCodeCommand struct {
	NodeDeviceID     string `json:"node_device_id"`
	DisplayName      string `json:"display_name"`
	ExpiresInMinutes int    `json:"expires_in_minutes"`
}

type GeneratePairingCodeResult struct {
	PairingCode  string    `json:"pairing_code"`
	RestaurantID string    `json:"restaurant_id"`
	NodeDeviceID string    `json:"node_device_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (s *Service) RegisterDevice(ctx context.Context, cmd RegisterDeviceCommand) (RegisterDeviceResult, error) {
	nodeID := strings.TrimSpace(cmd.NodeDeviceID)
	name := strings.TrimSpace(cmd.DisplayName)
	if nodeID == "" || name == "" {
		return RegisterDeviceResult{}, fmt.Errorf("%w: node_device_id and display_name are required", domain.ErrInvalid)
	}
	if node, err := s.repo.GetEdgeNode(ctx, nodeID); err == nil {
		if node.Status == domain.EdgeNodeRevoked {
			return RegisterDeviceResult{}, fmt.Errorf("%w: device is revoked", domain.ErrConflict)
		}
		return RegisterDeviceResult{NodeDeviceID: node.NodeDeviceID, Status: string(node.Status), RestaurantID: node.RestaurantID}, nil
	}
	now := s.clock.Now().UTC()
	registered, err := s.repo.RegisterUnassigned(ctx, domain.UnassignedEdgeNode{
		ID:              s.ids.NewID(),
		NodeDeviceID:    nodeID,
		ClaimedCloudURL: s.cloudURL,
		DisplayName:     name,
		AppVersion:      strings.TrimSpace(cmd.AppVersion),
		Status:          domain.UnassignedPending,
		FirstSeenAt:     now,
		LastSeenAt:      now,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		return RegisterDeviceResult{}, err
	}
	return RegisterDeviceResult{NodeDeviceID: registered.NodeDeviceID, Status: string(registered.Status)}, nil
}

func (s *Service) ListUnassigned(ctx context.Context) ([]domain.UnassignedEdgeNode, error) {
	return s.repo.ListUnassigned(ctx)
}

func (s *Service) AssignDevice(ctx context.Context, restaurantID, nodeDeviceID string) (AssignDeviceResult, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	if restaurantID == "" || nodeDeviceID == "" {
		return AssignDeviceResult{}, fmt.Errorf("%w: restaurant_id and node_device_id are required", domain.ErrInvalid)
	}
	if err := s.ensureRestaurantReady(ctx, restaurantID); err != nil {
		return AssignDeviceResult{}, err
	}
	now := s.clock.Now().UTC()
	node, err := s.repo.UpsertEdgeNode(ctx, domain.EdgeNode{
		ID:           s.ids.NewID(),
		RestaurantID: restaurantID,
		NodeDeviceID: nodeDeviceID,
		DisplayName:  "POS Edge Node",
		Status:       domain.EdgeNodeAssigned,
		LastSeenAt:   &now,
		AssignedAt:   &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return AssignDeviceResult{}, err
	}
	_ = s.repo.MarkUnassignedAssigned(ctx, nodeDeviceID, restaurantID, now)
	return AssignDeviceResult{NodeDeviceID: node.NodeDeviceID, RestaurantID: restaurantID, Status: string(node.Status), SnapshotURL: snapshotURL(restaurantID, nodeDeviceID)}, nil
}

func (s *Service) AssignmentStatus(ctx context.Context, nodeDeviceID string) (AssignmentStatus, error) {
	nodeDeviceID = strings.TrimSpace(nodeDeviceID)
	if nodeDeviceID == "" {
		return AssignmentStatus{}, fmt.Errorf("%w: node_device_id is required", domain.ErrInvalid)
	}
	node, err := s.repo.GetEdgeNode(ctx, nodeDeviceID)
	if err != nil {
		return AssignmentStatus{NodeDeviceID: nodeDeviceID, Status: "pending"}, nil
	}
	if node.Status != domain.EdgeNodeAssigned {
		return AssignmentStatus{NodeDeviceID: nodeDeviceID, Status: string(node.Status)}, nil
	}
	status := AssignmentStatus{
		NodeDeviceID: node.NodeDeviceID,
		Status:       string(node.Status),
		RestaurantID: node.RestaurantID,
		CloudURL:     s.cloudURL,
		SnapshotURL:  snapshotURL(node.RestaurantID, node.NodeDeviceID),
	}
	if strings.TrimSpace(node.CredentialsHash) != "" {
		return status, nil
	}
	token := newSecret("node")
	now := s.clock.Now().UTC()
	node.CredentialsHash = secretHash(token)
	node.LastSeenAt = &now
	node.UpdatedAt = now
	if _, err := s.repo.UpsertEdgeNode(ctx, node); err != nil {
		return AssignmentStatus{}, err
	}
	status.Credentials = &domain.Credentials{Type: "node_token", Token: token}
	return status, nil
}

func (s *Service) GeneratePairingCode(ctx context.Context, restaurantID string, cmd GeneratePairingCodeCommand) (GeneratePairingCodeResult, error) {
	restaurantID = strings.TrimSpace(restaurantID)
	if err := s.ensureRestaurantReady(ctx, restaurantID); err != nil {
		return GeneratePairingCodeResult{}, err
	}
	if s.license == nil {
		return GeneratePairingCodeResult{}, domain.ErrLicenseServerUnavailable
	}
	nodeID := strings.TrimSpace(cmd.NodeDeviceID)
	if nodeID == "" {
		nodeID = s.ids.NewID()
	}
	displayName := strings.TrimSpace(cmd.DisplayName)
	if displayName == "" {
		displayName = "POS Terminal"
	}
	minutes := cmd.ExpiresInMinutes
	if minutes <= 0 {
		minutes = 30
	}
	if minutes > 24*60 {
		minutes = 24 * 60
	}
	now := s.clock.Now().UTC()
	expiresAt := now.Add(time.Duration(minutes) * time.Minute)
	token := newSecret("node")
	code := pairingCode()
	if _, err := s.repo.UpsertEdgeNode(ctx, domain.EdgeNode{
		ID:              s.ids.NewID(),
		RestaurantID:    restaurantID,
		NodeDeviceID:    nodeID,
		DisplayName:     displayName,
		Status:          domain.EdgeNodeAssigned,
		CredentialsHash: secretHash(token),
		AssignedAt:      &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		return GeneratePairingCodeResult{}, err
	}
	if _, err := s.repo.CreatePairingCode(ctx, domain.PairingCode{
		ID:              s.ids.NewID(),
		PairingCodeHash: secretHash(code),
		RestaurantID:    restaurantID,
		NodeDeviceID:    nodeID,
		CloudURL:        s.cloudURL,
		Status:          domain.PairingCodeActive,
		ExpiresAt:       expiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		return GeneratePairingCodeResult{}, err
	}
	err := s.license.RegisterPairingCode(ctx, LicensePairingPayload{
		PairingCode:  code,
		CloudURL:     s.cloudURL,
		RestaurantID: restaurantID,
		NodeDeviceID: nodeID,
		Credentials:  domain.Credentials{Type: "node_token", Token: token},
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		return GeneratePairingCodeResult{}, domain.ErrLicenseServerUnavailable
	}
	return GeneratePairingCodeResult{PairingCode: code, RestaurantID: restaurantID, NodeDeviceID: nodeID, ExpiresAt: expiresAt}, nil
}

func (s *Service) ensureRestaurantReady(ctx context.Context, restaurantID string) error {
	restaurant, err := s.master.GetRestaurant(ctx, restaurantID)
	if err != nil {
		if errorsIsMasterNotFound(err) {
			return domain.ErrNotFound
		}
		return err
	}
	if restaurant.Status != masterdomain.RestaurantActive {
		return fmt.Errorf("%w: restaurant is archived", domain.ErrInvalid)
	}
	if _, err := s.master.GetCurrentPublishedPackage(ctx, restaurantID, ""); err != nil {
		if errorsIsMasterNotFound(err) {
			if _, err := s.master.Publish(ctx, masterapp.PublishCommand{RestaurantID: restaurantID, PublishedBy: "provisioning"}); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

func snapshotURL(restaurantID, nodeDeviceID string) string {
	return "/api/v1/restaurants/" + restaurantID + "/edge-nodes/" + nodeDeviceID + "/master-data/snapshot"
}

func secretHash(v string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(v)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func newSecret(prefix string) string {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(b[:])
}

func pairingCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "ABC23456"
	}
	var out [8]byte
	for i := range out {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out[:])
}

func errorsIsMasterNotFound(err error) bool {
	return strings.Contains(strings.ToLower(fmt.Sprint(err)), "not found")
}
