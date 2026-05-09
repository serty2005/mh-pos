package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalid                  = errors.New("invalid provisioning request")
	ErrNotFound                 = errors.New("provisioning resource not found")
	ErrConflict                 = errors.New("provisioning conflict")
	ErrDeviceNotAssigned        = errors.New("device not assigned")
	ErrPairingCodeInvalid       = errors.New("pairing code invalid")
	ErrPairingCodeExpired       = errors.New("pairing code expired")
	ErrLicenseServerUnavailable = errors.New("license server unavailable")
	ErrSnapshotNotPublished     = errors.New("snapshot not published")
)

type EdgeNodeStatus string

const (
	EdgeNodeUnassigned EdgeNodeStatus = "unassigned"
	EdgeNodeAssigned   EdgeNodeStatus = "assigned"
	EdgeNodeRevoked    EdgeNodeStatus = "revoked"
)

type UnassignedNodeStatus string

const (
	UnassignedPending  UnassignedNodeStatus = "pending"
	UnassignedAssigned UnassignedNodeStatus = "assigned"
	UnassignedRejected UnassignedNodeStatus = "rejected"
	UnassignedExpired  UnassignedNodeStatus = "expired"
)

type PairingCodeStatus string

const (
	PairingCodeActive   PairingCodeStatus = "active"
	PairingCodeConsumed PairingCodeStatus = "consumed"
	PairingCodeExpired  PairingCodeStatus = "expired"
	PairingCodeRevoked  PairingCodeStatus = "revoked"
)

type EdgeNode struct {
	ID              string         `json:"id"`
	RestaurantID    string         `json:"restaurant_id,omitempty"`
	NodeDeviceID    string         `json:"node_device_id"`
	DisplayName     string         `json:"display_name"`
	Status          EdgeNodeStatus `json:"status"`
	CredentialsHash string         `json:"-"`
	LastSeenAt      *time.Time     `json:"last_seen_at,omitempty"`
	AssignedAt      *time.Time     `json:"assigned_at,omitempty"`
	RevokedAt       *time.Time     `json:"revoked_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type UnassignedEdgeNode struct {
	ID                   string               `json:"id"`
	NodeDeviceID         string               `json:"node_device_id"`
	ClaimedCloudURL      string               `json:"claimed_cloud_url"`
	DisplayName          string               `json:"display_name"`
	AppVersion           string               `json:"app_version"`
	Status               UnassignedNodeStatus `json:"status"`
	FirstSeenAt          time.Time            `json:"first_seen_at"`
	LastSeenAt           time.Time            `json:"last_seen_at"`
	AssignedRestaurantID string               `json:"assigned_restaurant_id,omitempty"`
	AssignedAt           *time.Time           `json:"assigned_at,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
}

type PairingCode struct {
	ID              string            `json:"id"`
	PairingCodeHash string            `json:"-"`
	RestaurantID    string            `json:"restaurant_id"`
	NodeDeviceID    string            `json:"node_device_id"`
	CloudURL        string            `json:"cloud_url"`
	Status          PairingCodeStatus `json:"status"`
	ExpiresAt       time.Time         `json:"expires_at"`
	ConsumedAt      *time.Time        `json:"consumed_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type Credentials struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}
