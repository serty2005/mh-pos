package shared

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"crypto/pbkdf2"

	"pos-backend/internal/pos/domain"
)

const (
	pinHashPrefix     = "pin.pbkdf2.sha256"
	pinHashVersion    = "v1"
	pinHashIterations = 120000
	pinHashKeyLength  = 32
)

func HashPIN(pin string, salt []byte) (string, error) {
	pin = strings.TrimSpace(pin)
	if pin == "" || len(salt) == 0 {
		return "", fmt.Errorf("%w: pin and salt are required", domain.ErrInvalid)
	}
	key, err := pbkdf2.Key(sha256.New, pin, salt, pinHashIterations, pinHashKeyLength)
	if err != nil {
		return "", err
	}
	return strings.Join([]string{
		pinHashPrefix,
		pinHashVersion,
		strconv.Itoa(pinHashIterations),
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	}, ":"), nil
}

func VerifyPIN(storedHash, pin string) error {
	storedHash = strings.TrimSpace(storedHash)
	pin = strings.TrimSpace(pin)
	if storedHash == "" || pin == "" {
		return fmt.Errorf("%w: manager override pin is invalid", domain.ErrForbidden)
	}
	parts := strings.Split(storedHash, ":")
	if len(parts) != 5 || parts[0] != pinHashPrefix || parts[1] != pinHashVersion {
		return fmt.Errorf("%w: manager override pin hash is unsupported", domain.ErrForbidden)
	}
	iterations, err := strconv.Atoi(parts[2])
	if err != nil || iterations <= 0 {
		return fmt.Errorf("%w: manager override pin hash is invalid", domain.ErrForbidden)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(salt) == 0 {
		return fmt.Errorf("%w: manager override pin hash is invalid", domain.ErrForbidden)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(want) == 0 {
		return fmt.Errorf("%w: manager override pin hash is invalid", domain.ErrForbidden)
	}
	got, err := pbkdf2.Key(sha256.New, pin, salt, iterations, len(want))
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return fmt.Errorf("%w: manager override pin is invalid", domain.ErrForbidden)
	}
	return nil
}
