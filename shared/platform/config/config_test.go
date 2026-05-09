package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileConfigOverridesEnv(t *testing.T) {
	t.Setenv("SERVICE_HTTP_ADDR", ":9000")
	path := filepath.Join(t.TempDir(), "service.json")
	if err := os.WriteFile(path, []byte(`{"SERVICE_HTTP_ADDR":":9100","SERVICE_ENABLED":false,"SERVICE_BATCH_SIZE":7}`), 0o600); err != nil {
		t.Fatal(err)
	}

	src, err := Load("", path)
	if err != nil {
		t.Fatal(err)
	}

	if got := src.Get("SERVICE_HTTP_ADDR", ":8080"); got != ":9100" {
		t.Fatalf("expected file value to override env, got %q", got)
	}
	if got := src.Bool("SERVICE_ENABLED", true); got {
		t.Fatal("expected bool file value to override fallback")
	}
	if got := src.Int("SERVICE_BATCH_SIZE", 25); got != 7 {
		t.Fatalf("expected int file value, got %d", got)
	}
}

func TestMissingDefaultConfigIsOptional(t *testing.T) {
	src, err := Load("", filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := src.Get("SERVICE_HTTP_ADDR", ":8080"); got != ":8080" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestConfiguredPathIsRequired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	t.Setenv("SERVICE_CONFIG_PATH", path)
	if _, err := Load("SERVICE_CONFIG_PATH", ""); err == nil {
		t.Fatal("expected missing configured file to fail")
	}
}
