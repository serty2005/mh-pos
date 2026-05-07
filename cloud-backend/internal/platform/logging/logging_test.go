package logging

import (
	"log/slog"
	"testing"
)

func TestParseLevelSupportsConfiguredNames(t *testing.T) {
	fallback := slog.LevelInfo
	if got := ParseLevel("TRACE", fallback); got != LevelTrace {
		t.Fatalf("expected TRACE level, got %v", got)
	}
	if got := ParseLevel("DEBUG", fallback); got != slog.LevelDebug {
		t.Fatalf("expected DEBUG level, got %v", got)
	}
	if got := ParseLevel("WARN", fallback); got != slog.LevelWarn {
		t.Fatalf("expected WARN level, got %v", got)
	}
	if got := ParseLevel("ERROR", fallback); got != slog.LevelError {
		t.Fatalf("expected ERROR level, got %v", got)
	}
}
