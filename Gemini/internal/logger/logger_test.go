package logger

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"Debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
		{"  debug  ", slog.LevelDebug},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	// Should not panic
	SetLevel(slog.LevelDebug)
	SetLevel(slog.LevelInfo)
	SetLevel(slog.LevelWarn)
	SetLevel(slog.LevelError)
}

func TestDebug(t *testing.T) {
	// Should not panic
	SetLevel(slog.LevelDebug)
	Debug("test debug", "key", "value")
}

func TestInfo(t *testing.T) {
	// Should not panic
	Info("test info", "key", "value")
}

func TestWarn(t *testing.T) {
	// Should not panic
	Warn("test warn", "key", "value")
}

func TestError(t *testing.T) {
	// Should not panic
	Error("test error", "key", "value")
}

func TestPrintf(t *testing.T) {
	// Should not panic
	Printf("test %s %d", "formatted", 42)
}
