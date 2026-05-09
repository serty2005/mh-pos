package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Source объединяет значения из окружения и внешнего JSON-файла.
type Source struct {
	values map[string]string
	path   string
}

// Load читает optional JSON-файл конфигурации. Если pathEnv задан, файл обязателен.
func Load(pathEnv, defaultPath string) (Source, error) {
	path := strings.TrimSpace(os.Getenv(pathEnv))
	required := path != ""
	if path == "" {
		path = defaultPath
	}
	if strings.TrimSpace(path) == "" {
		return Source{values: map[string]string{}}, nil
	}

	values, err := loadJSON(path)
	if err != nil {
		if required || !os.IsNotExist(err) {
			return Source{}, fmt.Errorf("load config file %s: %w", path, err)
		}
		return Source{values: map[string]string{}}, nil
	}
	return Source{values: values, path: path}, nil
}

// Path возвращает путь к примененному файлу конфигурации.
func (s Source) Path() string {
	return s.path
}

// Get возвращает файловое значение, затем env, затем fallback.
func (s Source) Get(key, fallback string) string {
	if v, ok := s.values[key]; ok {
		return v
	}
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Bool возвращает bool из файлового/env значения или fallback при некорректном вводе.
func (s Source) Bool(key string, fallback bool) bool {
	raw, ok := s.lookup(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// Int возвращает положительное int-значение или fallback.
func (s Source) Int(key string, fallback int) int {
	raw, ok := s.lookup(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func (s Source) lookup(key string) (string, bool) {
	if v, ok := s.values[key]; ok {
		return v, true
	}
	v := os.Getenv(key)
	return v, v != ""
}

func loadJSON(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	values := make(map[string]string, len(payload))
	for key, value := range payload {
		switch v := value.(type) {
		case string:
			values[key] = v
		case bool:
			values[key] = strconv.FormatBool(v)
		case float64:
			values[key] = strconv.FormatFloat(v, 'f', -1, 64)
		case nil:
			values[key] = ""
		default:
			return nil, fmt.Errorf("unsupported config value for %s", key)
		}
	}
	return values, nil
}
