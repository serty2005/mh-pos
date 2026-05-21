package shared

import "errors"

var (
	ErrInvalid          = errors.New("invalid domain operation")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("domain invariant violation")
	ErrForbidden        = errors.New("forbidden")
	ErrTooManyRequests  = errors.New("too many requests")
	ErrDuplicate        = errors.New("duplicate resource")
	ErrDuplicateCommand = errors.New("duplicate command")
	ErrSaleUnavailable  = errors.New("sale unavailable")
)
