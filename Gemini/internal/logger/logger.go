// Package logger provides structured logging with levels.
package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var global *slog.Logger

func init() {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	global = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SetLevel updates the global logger level.
func SetLevel(level slog.Level) {
	global = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}

// Debug logs at debug level.
func Debug(msg string, args ...any) { global.Debug(msg, args...) }

// Info logs at info level.
func Info(msg string, args ...any) { global.Info(msg, args...) }

// Warn logs at warn level.
func Warn(msg string, args ...any) { global.Warn(msg, args...) }

// Error logs at error level.
func Error(msg string, args ...any) { global.Error(msg, args...) }

// Fatalf logs at error level and exits.
func Fatalf(msg string, args ...any) {
	global.Error(msg, args...)
	os.Exit(1)
}

// Printf provides fmt.Printf compatibility for gradual migration.
func Printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	global.Info(msg)
}
