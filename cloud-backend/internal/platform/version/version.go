package version

import (
	"os"
	"strings"
)

const (
	// DefaultProductVersion задает единую версию продукта для модулей монорепозитория.
	// POS-84: повышено до 0.1.15 для применения managed baseline со stream printers.
	DefaultProductVersion = "0.1.15"
)

// Resolve возвращает версию модуля из env или canonical default.
func Resolve(envVar string) string {
	raw := strings.TrimSpace(os.Getenv(envVar))
	if raw == "" {
		return DefaultProductVersion
	}
	return raw
}
