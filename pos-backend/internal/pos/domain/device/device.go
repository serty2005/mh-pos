package device

import "time"

type Device struct {
	ID           string    `json:"id"`
	RestaurantID string    `json:"restaurant_id"`
	DeviceCode   string    `json:"device_code"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Active       bool      `json:"active"`
	RegisteredAt time.Time `json:"registered_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type EdgeNodeStatus string

const (
	EdgeNodePaired EdgeNodeStatus = "paired"
)

type EdgeNodeIdentity struct {
	ID              string         `json:"id"`
	NodeDeviceID    string         `json:"node_device_id"`
	RestaurantID    string         `json:"restaurant_id"`
	Status          EdgeNodeStatus `json:"status"`
	PairingCodeHash string         `json:"pairing_code_hash,omitempty"`
	PairedAt        time.Time      `json:"paired_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type ClientDeviceStatus string

const (
	ClientDeviceActive ClientDeviceStatus = "active"
)

type ClientDevice struct {
	ID             string             `json:"id"`
	RestaurantID   string             `json:"restaurant_id"`
	NodeDeviceID   string             `json:"node_device_id"`
	ClientDeviceID string             `json:"client_device_id"`
	Status         ClientDeviceStatus `json:"status"`
	FirstSeenAt    time.Time          `json:"first_seen_at"`
	LastSeenAt     time.Time          `json:"last_seen_at"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type PairingStatus struct {
	Paired       bool              `json:"paired"`
	Identity     *EdgeNodeIdentity `json:"identity,omitempty"`
	NodeDeviceID string            `json:"node_device_id,omitempty"`
	RestaurantID string            `json:"restaurant_id,omitempty"`
}

type ProvisioningStatus string

const (
	ProvisioningNotConfigured               ProvisioningStatus = "not_configured"
	ProvisioningUnpairedRegistered          ProvisioningStatus = "pending_admin_approval"
	ProvisioningAssignedDownloadingSnapshot ProvisioningStatus = "assigned_downloading_snapshot"
	ProvisioningPaired                      ProvisioningStatus = "paired"
	ProvisioningError                       ProvisioningStatus = "error"
)

type EdgeProvisioningState struct {
	ID               string             `json:"id"`
	NodeDeviceID     string             `json:"node_device_id"`
	CloudURL         string             `json:"cloud_url,omitempty"`
	LicenseURL       string             `json:"license_url,omitempty"`
	RestaurantID     string             `json:"restaurant_id,omitempty"`
	Status           ProvisioningStatus `json:"status"`
	CredentialsType  string             `json:"credentials_type,omitempty"`
	CredentialsToken string             `json:"-"`
	LastError        string             `json:"last_error,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

type ProvisioningStatusView struct {
	NodeDeviceID string             `json:"node_device_id"`
	CloudURL     string             `json:"cloud_url,omitempty"`
	LicenseURL   string             `json:"license_url,omitempty"`
	RestaurantID string             `json:"restaurant_id,omitempty"`
	Status       ProvisioningStatus `json:"status"`
	Paired       bool               `json:"paired"`
	LastError    string             `json:"last_error,omitempty"`
}
