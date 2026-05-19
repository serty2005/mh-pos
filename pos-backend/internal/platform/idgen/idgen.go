package idgen

import (
	"fmt"

	"github.com/google/uuid"
)

// Generator задает общий контракт генерации идентификаторов runtime-модулей.
type Generator interface {
	NewID() string
}

// UUIDGenerator генерирует UUID v7 для новых runtime-сущностей и sync events.
type UUIDGenerator struct{}

// NewID возвращает UUID v7 или останавливает выполнение, чтобы не создать событие с неподдерживаемым ID.
func (UUIDGenerator) NewID() string {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Errorf("generate uuidv7: %w", err))
	}
	return id.String()
}
