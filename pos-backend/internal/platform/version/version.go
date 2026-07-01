package version

import (
	"os"
	"strings"
)

const (
	// DefaultProductVersion задает единую версию продукта для модулей монорепозитория.
	// POS-86: повышено до 0.1.11 для Cloud-owned sales points/sections и target-level print routing.
	DefaultProductVersion = "0.1.11"
)

// Resolve возвращает версию модуля из env или canonical default.
func Resolve(envVar string) string {
	raw := strings.TrimSpace(os.Getenv(envVar))
	if raw == "" {
		return DefaultProductVersion
	}
	return raw
}
