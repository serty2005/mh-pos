package idgen

import "github.com/google/uuid"

type Generator interface {
	NewID() string
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
