package ports

import (
	"context"

	"pos-backend/internal/pos/domain/device"
)

type DeviceRepository interface {
	CreateDevice(context.Context, *device.Device) error
	GetDevice(context.Context, string) (*device.Device, error)
	ListDevices(context.Context) ([]device.Device, error)
	UpsertEdgeNodeIdentity(context.Context, *device.EdgeNodeIdentity) error
	GetEdgeNodeIdentity(context.Context) (*device.EdgeNodeIdentity, error)
	CreateClientDevice(context.Context, *device.ClientDevice) error
	GetClientDevice(context.Context, string, string) (*device.ClientDevice, error)
	TouchClientDevice(context.Context, string, string, string) error
}
