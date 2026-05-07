package logging

import (
	"log/slog"
	"testing"
)

func TestParseLevelSupportsAllConfiguredNames(t *testing.T) {
	fallback := slog.LevelInfo
	cases := map[string]slog.Level{
		"TRACE":   LevelTrace,
		"trace":   LevelTrace,
		"DEBUG":   slog.LevelDebug,
		"INFO":    slog.LevelInfo,
		"WARN":    slog.LevelWarn,
		"WARNING": slog.LevelWarn,
		"ERROR":   slog.LevelError,
		"unknown": fallback,
		"":        fallback,
	}
	for input, want := range cases {
		if got := ParseLevel(input, fallback); got != want {
			t.Fatalf("ParseLevel(%q)=%v, want=%v", input, got, want)
		}
	}
}
