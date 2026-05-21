package idgen

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

// Generator задает общий контракт генерации UUID v7 для Cloud runtime.
type Generator interface {
	NewID() string
}

// UUIDGenerator генерирует UUID v7 без внешней зависимости.
type UUIDGenerator struct{}

// NewID возвращает UUID v7 или завершает выполнение, если crypto/rand недоступен.
func (UUIDGenerator) NewID() string {
	id, err := NewV7()
	if err != nil {
		panic(fmt.Errorf("generate uuidv7: %w", err))
	}
	return id
}

// NewV7 генерирует UUID v7 по timestamp milliseconds и случайному хвосту.
func NewV7() (string, error) {
	var b [16]byte
	ms := uint64(time.Now().UTC().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	if _, err := rand.Read(b[6:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
	), nil
}
