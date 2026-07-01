package version

import (
	"os"
	"strings"
)

const (
	// DefaultProductVersion задает единую версию продукта для модулей монорепозитория.
	// POS-86: повышено до 0.1.16 для sales_points/restaurant_sections и обязательной section_id у tables.
	DefaultProductVersion = "0.1.16"
)

// Resolve возвращает версию модуля из env или canonical default.
func Resolve(envVar string) string {
	raw := strings.TrimSpace(os.Getenv(envVar))
	if raw == "" {
		return DefaultProductVersion
	}
	return raw
}
