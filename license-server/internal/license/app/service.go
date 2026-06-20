package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalid             = fmt.Errorf("pairing code invalid")
	ErrExpired             = fmt.Errorf("pairing code expired")
	ErrConsumed            = fmt.Errorf("pairing code invalid: consumed")
	ErrEntitlementNotFound = fmt.Errorf("entitlement snapshot not found")
	ErrEntitlementVersion  = fmt.Errorf("entitlement snapshot version conflict")
)

// SafeErrorReason возвращает стабильную внутреннюю причину отказа без pairing code и токенов.
func SafeErrorReason(err error) string {
	var safe *safeError
	if errors.As(err, &safe) {
		return safe.reason
	}
	switch {
	case errors.Is(err, ErrExpired):
		return "expired"
	case errors.Is(err, ErrConsumed):
		return "consumed"
	case errors.Is(err, ErrInvalid):
		return "invalid"
	default:
		return "internal_error"
	}
}

type safeError struct {
	err    error
	reason string
}

func (e *safeError) Error() string {
	return e.err.Error() + ": " + e.reason
}

func (e *safeError) Unwrap() error {
	return e.err
}

func invalid(reason string) error {
	return &safeError{err: ErrInvalid, reason: reason}
}

func expired(reason string) error {
	return &safeError{err: ErrExpired, reason: reason}
}

func consumed(reason string) error {
	return &safeError{err: ErrConsumed, reason: reason}
}

type PairingCode struct {
	PairingCodeHash string
	PairingID       string
	InstanceID      string
	CloudURL        string
	RestaurantID    string
	NodeDeviceID    string
	ExpiresAt       time.Time
	ConsumedAt      *time.Time
	CreatedAt       time.Time
}

type Repository interface {
	Save(context.Context, PairingCode) error
	GetByHash(context.Context, string) (PairingCode, error)
	MarkConsumed(context.Context, string, time.Time) error
	SaveEntitlements(context.Context, EntitlementSnapshot) error
	GetEntitlements(context.Context, string, string) (EntitlementSnapshot, error)
}

// EntitlementSnapshot хранит authoritative module grants для tenant/server.
type EntitlementSnapshot struct {
	TenantID     string          `json:"tenant_id"`
	ServerID     string          `json:"server_id"`
	Version      int64           `json:"version"`
	Status       string          `json:"status"`
	Entitlements map[string]bool `json:"entitlements"`
	IssuedAt     time.Time       `json:"issued_at"`
	ExpiresAt    time.Time       `json:"expires_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type PutEntitlementsCommand struct {
	Version      int64           `json:"version"`
	Status       string          `json:"status"`
	Entitlements map[string]bool `json:"entitlements"`
	IssuedAt     time.Time       `json:"issued_at"`
	ExpiresAt    time.Time       `json:"expires_at"`
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type RegisterPairingCodeCommand struct {
	PairingCode  string    `json:"pairing_code"`
	PairingID    string    `json:"pairing_id"`
	InstanceID   string    `json:"instance_id"`
	CloudURL     string    `json:"cloud_url"`
	RestaurantID string    `json:"restaurant_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type RegisterResult struct {
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ResolveCommand struct {
	PairingCode string `json:"pairing_code"`
}

type ResolveResult struct {
	PairingID    string    `json:"pairing_id"`
	CloudURL     string    `json:"cloud_url"`
	RestaurantID string    `json:"restaurant_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (s *Service) Register(ctx context.Context, cmd RegisterPairingCodeCommand) (RegisterResult, error) {
	code := strings.TrimSpace(cmd.PairingCode)
	if code == "" || strings.TrimSpace(cmd.PairingID) == "" || strings.TrimSpace(cmd.InstanceID) == "" || strings.TrimSpace(cmd.CloudURL) == "" || strings.TrimSpace(cmd.RestaurantID) == "" {
		return RegisterResult{}, invalid("registration_required_fields_missing")
	}
	expiresAt := cmd.ExpiresAt.UTC()
	if expiresAt.IsZero() || !expiresAt.After(time.Now().UTC()) {
		return RegisterResult{}, expired("registration_expires_at_not_future")
	}
	err := s.repo.Save(ctx, PairingCode{
		PairingCodeHash: Hash(code),
		PairingID:       strings.TrimSpace(cmd.PairingID),
		InstanceID:      strings.TrimSpace(cmd.InstanceID),
		CloudURL:        strings.TrimSpace(cmd.CloudURL),
		RestaurantID:    strings.TrimSpace(cmd.RestaurantID),
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
		return ResolveResult{}, invalid("pairing_code_required")
	}
	item, err := s.repo.GetByHash(ctx, Hash(code))
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			return ResolveResult{}, invalid("pairing_code_not_found")
		}
		return ResolveResult{}, err
	}
	if item.ConsumedAt != nil {
		return ResolveResult{}, consumed("pairing_code_consumed")
	}
	if !item.ExpiresAt.After(time.Now().UTC()) {
		return ResolveResult{}, expired("pairing_code_expired")
	}
	return ResolveResult{PairingID: item.PairingID, CloudURL: item.CloudURL, RestaurantID: item.RestaurantID, ExpiresAt: item.ExpiresAt}, nil
}

// PutEntitlements сохраняет только более новую provider-issued версию snapshot.
func (s *Service) PutEntitlements(ctx context.Context, tenantID, serverID string, cmd PutEntitlementsCommand) (EntitlementSnapshot, error) {
	tenantID, serverID = strings.TrimSpace(tenantID), strings.TrimSpace(serverID)
	status := strings.TrimSpace(cmd.Status)
	if tenantID == "" || serverID == "" || cmd.Version <= 0 || (status != "active" && status != "revoked") || cmd.IssuedAt.IsZero() || cmd.ExpiresAt.IsZero() || len(cmd.Entitlements) == 0 {
		return EntitlementSnapshot{}, invalid("entitlement_snapshot_invalid")
	}
	if status == "active" && !cmd.ExpiresAt.After(cmd.IssuedAt) {
		return EntitlementSnapshot{}, invalid("entitlement_expiry_invalid")
	}
	for id := range cmd.Entitlements {
		if !validEntitlementID(id) {
			return EntitlementSnapshot{}, invalid("entitlement_id_invalid")
		}
	}
	if current, err := s.repo.GetEntitlements(ctx, tenantID, serverID); err == nil && cmd.Version <= current.Version {
		return EntitlementSnapshot{}, ErrEntitlementVersion
	} else if err != nil && !errors.Is(err, ErrEntitlementNotFound) {
		return EntitlementSnapshot{}, err
	}
	snapshot := EntitlementSnapshot{TenantID: tenantID, ServerID: serverID, Version: cmd.Version, Status: status, Entitlements: cmd.Entitlements, IssuedAt: cmd.IssuedAt.UTC(), ExpiresAt: cmd.ExpiresAt.UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.repo.SaveEntitlements(ctx, snapshot); err != nil {
		return EntitlementSnapshot{}, err
	}
	return snapshot, nil
}

func (s *Service) GetEntitlements(ctx context.Context, tenantID, serverID string) (EntitlementSnapshot, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(serverID) == "" {
		return EntitlementSnapshot{}, invalid("entitlement_scope_required")
	}
	return s.repo.GetEntitlements(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(serverID))
}

func validEntitlementID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 100 || id[0] == '-' || id[len(id)-1] == '-' {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '.' {
			return false
		}
	}
	return true
}

func Hash(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return "sha256:" + hex.EncodeToString(sum[:])
}
