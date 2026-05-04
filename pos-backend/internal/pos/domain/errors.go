package domain

import "errors"

var (
	ErrInvalid   = errors.New("invalid domain operation")
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("domain invariant violation")
	ErrDuplicate = errors.New("duplicate resource")
)
