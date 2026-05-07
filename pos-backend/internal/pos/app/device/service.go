package device

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"pos-backend/internal/platform/clock"
	"pos-backend/internal/platform/idgen"
	txmanager "pos-backend/internal/platform/tx"
	"pos-backend/internal/pos/app/shared"
	"pos-backend/internal/pos/domain"
	"pos-backend/internal/pos/ports"
	"strings"
)

type Service struct {
	repo  ports.Repository
	tx    txmanager.Manager
	ids   idgen.Generator
	clock clock.Clock
}

const pairingVerifierVersion = "pairing.hmac-sha256.v1"

func NewService(repo ports.Repository, tx txmanager.Manager, ids idgen.Generator, clock clock.Clock) *Service {
	return &Service{repo: repo, tx: tx, ids: ids, clock: clock}
}

type RegisterDeviceCommand struct {
	shared.CommandMeta
	RestaurantID string `json:"restaurant_id"`
	DeviceCode   string `json:"device_code"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

type PairEdgeNodeCommand struct {
	PairingCode  string `json:"pairing_code"`
	RestaurantID string `json:"restaurant_id,omitempty"`
	NodeDeviceID string `json:"node_device_id,omitempty"`
}

func (s *Service) ListDevices(ctx context.Context) ([]domain.Device, error) {
	return s.repo.ListDevices(ctx)
}

func (s *Service) GetPairingStatus(ctx context.Context) (domain.PairingStatus, error) {
	identity, err := s.repo.GetEdgeNodeIdentity(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.PairingStatus{Paired: false}, nil
		}
		return domain.PairingStatus{}, err
	}
	return domain.PairingStatus{Paired: identity.Status == domain.EdgeNodePaired, Identity: identity, NodeDeviceID: identity.NodeDeviceID, RestaurantID: identity.RestaurantID}, nil
}

func (s *Service) PairEdgeNode(ctx context.Context, cmd PairEdgeNodeCommand) (*domain.EdgeNodeIdentity, error) {
	restaurantID, nodeDeviceID, err := parsePairingPayload(cmd)
	if err != nil {
		return nil, err
	}
	restaurants, err := s.repo.ListRestaurants(ctx)
	if err != nil {
		return nil, err
	}
	knownRestaurant := false
	for _, item := range restaurants {
		if item.ID == restaurantID {
			knownRestaurant = true
			break
		}
	}
	if !knownRestaurant {
		return nil, fmt.Errorf("%w: pairing_code references unknown restaurant_id", domain.ErrInvalid)
	}
	now := s.clock.Now()
	identity := &domain.EdgeNodeIdentity{
		ID:              "local",
		NodeDeviceID:    nodeDeviceID,
		RestaurantID:    restaurantID,
		Status:          domain.EdgeNodePaired,
		PairingCodeHash: pairingCodeVerifier(cmd.PairingCode, restaurantID, nodeDeviceID),
		PairedAt:        now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return identity, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := s.repo.GetDevice(ctx, nodeDeviceID); err != nil {
			if !errors.Is(err, domain.ErrNotFound) {
				return err
			}
			device := &domain.Device{
				ID:           nodeDeviceID,
				RestaurantID: restaurantID,
				DeviceCode:   "EDGE-NODE",
				Name:         "POS Edge Node",
				Type:         "edge-node",
				Active:       true,
				RegisteredAt: now,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := s.repo.CreateDevice(ctx, device); err != nil {
				return err
			}
		}
		if err := s.repo.UpsertEdgeNodeIdentity(ctx, identity); err != nil {
			return err
		}
		meta := shared.CommandMeta{CommandID: s.ids.NewID(), NodeDeviceID: nodeDeviceID, DeviceID: nodeDeviceID, Origin: domain.OriginEdgeDevice}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, meta, restaurantID, "", "EdgeNode", nodeDeviceID, "EdgeNodePaired", map[string]any{
			"node_device_id": nodeDeviceID,
			"restaurant_id":  restaurantID,
		})
	})
}

func pairingCodeVerifier(code, restaurantID, nodeDeviceID string) string {
	key := []byte(pairingVerifierVersion + ":" + strings.TrimSpace(restaurantID) + ":" + strings.TrimSpace(nodeDeviceID))
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(strings.TrimSpace(code)))
	return pairingVerifierVersion + ":" + hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) RegisterDevice(ctx context.Context, cmd RegisterDeviceCommand) (*domain.Device, error) {
	shared.NormalizeDeviceMeta(&cmd.CommandMeta)
	if err := shared.EnsureMasterDataWriteAllowed(cmd.CommandMeta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.DeviceCode) == "" || strings.TrimSpace(cmd.Name) == "" || strings.TrimSpace(cmd.Type) == "" {
		return nil, fmt.Errorf("%w: restaurant_id, device_code, name and type are required", domain.ErrInvalid)
	}
	now := s.clock.Now()
	v := &domain.Device{ID: s.ids.NewID(), RestaurantID: cmd.RestaurantID, DeviceCode: cmd.DeviceCode, Name: cmd.Name, Type: cmd.Type, Active: true, RegisteredAt: now, CreatedAt: now, UpdatedAt: now}
	return v, s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := shared.EnsureCommandNotProcessed(ctx, s.repo, cmd.CommandID); err != nil {
			return err
		}
		if err := s.repo.CreateDevice(ctx, v); err != nil {
			return err
		}
		return shared.WriteOutbox(ctx, s.repo, s.ids, s.clock, cmd.CommandMeta, v.RestaurantID, "", "Device", v.ID, "DeviceRegistered", v)
	})
}

func parsePairingPayload(cmd PairEdgeNodeCommand) (string, string, error) {
	restaurantID := strings.TrimSpace(cmd.RestaurantID)
	nodeDeviceID := strings.TrimSpace(cmd.NodeDeviceID)
	code := strings.TrimSpace(cmd.PairingCode)
	if code == "" {
		return "", "", fmt.Errorf("%w: pairing_code is required", domain.ErrInvalid)
	}
	parts := strings.Split(code, ":")
	if len(parts) == 3 && strings.EqualFold(parts[0], "MHPOS") {
		if restaurantID == "" {
			restaurantID = strings.TrimSpace(parts[1])
		}
		if nodeDeviceID == "" {
			nodeDeviceID = strings.TrimSpace(parts[2])
		}
	}
	if restaurantID == "" || nodeDeviceID == "" {
		return "", "", fmt.Errorf("%w: MVP pairing code must be MHPOS:<restaurant_id>:<node_device_id>", domain.ErrInvalid)
	}
	if strings.ContainsAny(restaurantID, "<>") || strings.ContainsAny(nodeDeviceID, "<>") {
		return "", "", fmt.Errorf("%w: pairing_code must contain real ids, not placeholders", domain.ErrInvalid)
	}
	return restaurantID, nodeDeviceID, nil
}
