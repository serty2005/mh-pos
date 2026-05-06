package ports

import (
	"context"

	"pos-backend/internal/pos/domain/device"
)

type DeviceRepository interface {
	CreateDevice(context.Context, *device.Device) error
	GetDevice(context.Context, string) (*device.Device, error)
	ListDevices(context.Context) ([]device.Device, error)
}
