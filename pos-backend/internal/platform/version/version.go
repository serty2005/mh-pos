package version

import (
	"os"
	"strings"
)

const (
	// DefaultProductVersion задает единую версию продукта для модулей монорепозитория.
	// POS-84: повышено до 0.1.9 для применения managed baseline с receipt_printers.
	DefaultProductVersion = "0.1.9"
)

// Resolve возвращает версию модуля из env или canonical default.
func Resolve(envVar string) string {
	raw := strings.TrimSpace(os.Getenv(envVar))
	if raw == "" {
		return DefaultProductVersion
	}
	return raw
}
