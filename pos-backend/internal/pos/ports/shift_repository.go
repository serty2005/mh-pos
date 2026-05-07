package ports

import (
	"context"

	"pos-backend/internal/pos/domain/shift"
)

type ShiftRepository interface {
	CreateShift(context.Context, *shift.Shift) error
	UpdateShiftClosed(context.Context, *shift.Shift) error
	GetShift(context.Context, string) (*shift.Shift, error)
	GetOpenShiftByDevice(context.Context, string) (*shift.Shift, error)
	GetOpenShiftByEmployee(context.Context, string, string) (*shift.Shift, error)
	ListRecentShiftsByEmployee(context.Context, string, string, int) ([]shift.Shift, error)
	HasOpenOrdersForShift(context.Context, string) (bool, error)
}
