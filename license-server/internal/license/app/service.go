package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalid  = fmt.Errorf("pairing code invalid")
	ErrExpired  = fmt.Errorf("pairing code expired")
	ErrConsumed = fmt.Errorf("pairing code invalid: consumed")
)

type Credentials struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type PairingCode struct {
	PairingCodeHash string
	CloudURL        string
	RestaurantID    string
	NodeDeviceID    string
	Credentials     Credentials
	ExpiresAt       time.Time
	ConsumedAt      *time.Time
	CreatedAt       time.Time
}

type Repository interface {
	Save(context.Context, PairingCode) error
	GetByHash(context.Context, string) (PairingCode, error)
	MarkConsumed(context.Context, string, time.Time) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type RegisterPairingCodeCommand struct {
	PairingCode  string      `json:"pairing_code"`
	CloudURL     string      `json:"cloud_url"`
	RestaurantID string      `json:"restaurant_id"`
	NodeDeviceID string      `json:"node_device_id"`
	Credentials  Credentials `json:"credentials"`
	ExpiresAt    time.Time   `json:"expires_at"`
}

type RegisterResult struct {
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ResolveCommand struct {
	PairingCode  string `json:"pairing_code"`
	NodeDeviceID string `json:"node_device_id"`
}

type ResolveResult struct {
	CloudURL     string      `json:"cloud_url"`
	RestaurantID string      `json:"restaurant_id"`
	NodeDeviceID string      `json:"node_device_id"`
	Credentials  Credentials `json:"credentials"`
}

func (s *Service) Register(ctx context.Context, cmd RegisterPairingCodeCommand) (RegisterResult, error) {
	code := strings.TrimSpace(cmd.PairingCode)
	if code == "" || strings.TrimSpace(cmd.CloudURL) == "" || strings.TrimSpace(cmd.RestaurantID) == "" || strings.TrimSpace(cmd.NodeDeviceID) == "" || strings.TrimSpace(cmd.Credentials.Token) == "" {
		return RegisterResult{}, ErrInvalid
	}
	expiresAt := cmd.ExpiresAt.UTC()
	if expiresAt.IsZero() || !expiresAt.After(time.Now().UTC()) {
		return RegisterResult{}, ErrExpired
	}
	err := s.repo.Save(ctx, PairingCode{
		PairingCodeHash: Hash(code),
		CloudURL:        strings.TrimSpace(cmd.CloudURL),
		RestaurantID:    strings.TrimSpace(cmd.RestaurantID),
		NodeDeviceID:    strings.TrimSpace(cmd.NodeDeviceID),
		Credentials:     cmd.Credentials,
		ExpiresAt:       expiresAt,
		CreatedAt:       time.Now().UTC(),
	})
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{Status: "registered", ExpiresAt: expiresAt}, nil
}

func (s *Service) Resolve(ctx context.Context, cmd ResolveCommand) (ResolveResult, error) {
	code := strings.TrimSpace(cmd.PairingCode)
	if code == "" {
		return ResolveResult{}, ErrInvalid
	}
	item, err := s.repo.GetByHash(ctx, Hash(code))
	if err != nil {
		return ResolveResult{}, ErrInvalid
	}
	if item.ConsumedAt != nil {
		return ResolveResult{}, ErrConsumed
	}
	if !item.ExpiresAt.After(time.Now().UTC()) {
		return ResolveResult{}, ErrExpired
	}
	if requested := strings.TrimSpace(cmd.NodeDeviceID); requested != "" && requested != item.NodeDeviceID {
		return ResolveResult{}, ErrInvalid
	}
	now := time.Now().UTC()
	if err := s.repo.MarkConsumed(ctx, item.PairingCodeHash, now); err != nil {
		return ResolveResult{}, err
	}
	return ResolveResult{CloudURL: item.CloudURL, RestaurantID: item.RestaurantID, NodeDeviceID: item.NodeDeviceID, Credentials: item.Credentials}, nil
}

func Hash(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return "sha256:" + hex.EncodeToString(sum[:])
}
