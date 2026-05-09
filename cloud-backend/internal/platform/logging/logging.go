package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

const LevelTrace = slog.Level(-8)

func ParseLevel(raw string, fallback slog.Level) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "TRACE":
		return LevelTrace
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return fallback
	}
}

func NewJSONLogger(envVar string) *slog.Logger {
	level := ParseLevel(os.Getenv(envVar), slog.LevelInfo)
	return NewJSONLoggerWithWriter(os.Stdout, level).With("log_level_env", envVar)
}

func NewJSONLoggerWithLevel(rawLevel string) *slog.Logger {
	level := ParseLevel(rawLevel, slog.LevelInfo)
	return NewJSONLoggerWithWriter(os.Stdout, level)
}

func NewJSONLoggerWithWriter(out io.Writer, level slog.Leveler) *slog.Logger {
	handler := slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				if lv, ok := a.Value.Any().(slog.Level); ok {
					switch {
					case lv <= LevelTrace:
						return slog.String(a.Key, "TRACE")
					case lv <= slog.LevelDebug:
						return slog.String(a.Key, "DEBUG")
					case lv <= slog.LevelInfo:
						return slog.String(a.Key, "INFO")
					case lv <= slog.LevelWarn:
						return slog.String(a.Key, "WARN")
					default:
						return slog.String(a.Key, "ERROR")
					}
				}
			}
			return a
		},
	})
	return slog.New(handler)
}
